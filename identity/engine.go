package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sports-Data-Africa/contracts/canonical"
)

// Engine is the single authoritative fixture identity resolution engine for the platform.
// It coordinates L1 local process memory, L2 Redis distributed caching, and MySQL persistence.
//
// Identity Layer Separation Invariant:
//   - Engine resolves and manages FIXTURE identities only (provider_fixture_id -> CanonicalFixtureID).
//   - It preserves the original provider_fixture_id as an alias for live in-play linking and result settlement.
//   - It NEVER generates or modifies canonical league/competition IDs (Sportradar tournament IDs) or market IDs.
type Engine struct {
	repo  Repository
	cache *TieredIdentityCache
}

func NewEngine(repo Repository, cache *TieredIdentityCache) *Engine {
	return &Engine{
		repo:  repo,
		cache: cache,
	}
}

// GetCache returns the tiered identity cache.
func (e *Engine) GetCache() *TieredIdentityCache {
	return e.cache
}

// GetRepository returns the underlying persistent repository.
func (e *Engine) GetRepository() Repository {
	return e.repo
}

// NormalizeProviderFixtureID centralizes provider fixture ID normalization while preserving
// the original trimmed provider reference alongside the prefix-stripped form.
//
// Rules:
//   - Trims leading/trailing whitespace.
//   - Strips known Sportradar event prefixes ("sr:match:", "sr:sport_event:") in any order,
//     including repeated prefixes, case-insensitively.
//   - Rejects empty IDs or bare prefixes with no underlying identifier (returns "", "", nil).
//   - Returns deduplicated candidate IDs in lookup priority order: [originalTrimmed, cleanID].
func NormalizeProviderFixtureID(raw string) (originalTrimmed string, cleanID string, candidates []string) {
	originalTrimmed = strings.TrimSpace(raw)
	if originalTrimmed == "" {
		return "", "", nil
	}

	curr := originalTrimmed
	prefixes := []string{"sr:match:", "sr:sport_event:"}
	for {
		stripped := false
		lower := strings.ToLower(curr)
		for _, p := range prefixes {
			if strings.HasPrefix(lower, p) {
				curr = strings.TrimSpace(curr[len(p):])
				stripped = true
				break
			}
		}
		if !stripped {
			break
		}
	}

	if curr == "" {
		// Input consisted solely of prefixes (e.g. "sr:match:") with no actual fixture ID.
		return "", "", nil
	}

	cleanID = curr
	candidates = []string{originalTrimmed}
	if cleanID != originalTrimmed {
		candidates = append(candidates, cleanID)
	}
	return originalTrimmed, cleanID, candidates
}

// backfillAliasCache writes all candidate provider fixture IDs into the tiered cache
// under the canonical normalized provider key (and raw provider key if distinct).
func (e *Engine) backfillAliasCache(ctx context.Context, rawProvider, normProvider, canonicalID string, candidatePIDs ...string) {
	if e.cache == nil || canonicalID == "" {
		return
	}
	seen := make(map[string]struct{}, len(candidatePIDs))
	for _, pid := range candidatePIDs {
		pid = strings.TrimSpace(pid)
		if pid == "" {
			continue
		}
		if _, exists := seen[pid]; exists {
			continue
		}
		seen[pid] = struct{}{}
		e.cache.PutAlias(ctx, normProvider, pid, canonicalID)
		if rawProvider != "" && rawProvider != normProvider {
			e.cache.PutAlias(ctx, rawProvider, pid, canonicalID)
		}
	}
}

// ResolveCanonicalID resolves an external provider's fixture ID to the SDA canonical fixture ID.
// Normalizes provider names to canonical codes (e.g. "sportybet" -> "SPORTSXBET") while retaining
// backward-compatible fallback checks for existing database records.
//
// Hot lookup order:
//  1. L1 process-local cache
//  2. L2 Redis distributed cache
//  3. MySQL repository lookup (authoritative source of truth; backfills L2 and L1 on hit)
//
// Returns ErrAliasNotFound if the provider fixture ID is not mapped (never invents a fixture ID).
// Propagates unexpected repository errors immediately so cache misses never mask DB failures.
func (e *Engine) ResolveCanonicalID(ctx context.Context, provider, providerFixtureID string) (string, error) {
	rawProvider := strings.TrimSpace(provider)
	origPID, cleanPID, candidatePIDs := NormalizeProviderFixtureID(providerFixtureID)
	if rawProvider == "" || len(candidatePIDs) == 0 {
		return "", errors.New("provider and valid provider_fixture_id are required")
	}

	normProvider := canonical.NormalizeProvider(rawProvider)
	if normProvider == "" || normProvider == "UNKNOWN" {
		normProvider = strings.ToUpper(rawProvider)
	}

	// lookup checks L1 -> L2 -> Repository for a specific (providerKey, targetPID) pair.
	lookup := func(providerKey, targetPID string) (string, error) {
		if e.cache != nil {
			if cid, ok := e.cache.ResolveLocal(providerKey, targetPID); ok && cid != "" {
				e.backfillAliasCache(ctx, rawProvider, normProvider, cid, origPID, cleanPID, targetPID)
				return cid, nil
			}
			if cid, ok := e.cache.ResolveTiered(ctx, providerKey, targetPID); ok && cid != "" {
				e.backfillAliasCache(ctx, rawProvider, normProvider, cid, origPID, cleanPID, targetPID)
				return cid, nil
			}
		}
		if e.repo != nil {
			alias, err := e.repo.GetAlias(ctx, providerKey, targetPID)
			if err == nil && alias != nil && alias.CanonicalFixtureID != "" {
				e.backfillAliasCache(ctx, rawProvider, normProvider, alias.CanonicalFixtureID, origPID, cleanPID, targetPID)
				return alias.CanonicalFixtureID, nil
			}
			if err != nil && !errors.Is(err, ErrAliasNotFound) {
				return "", fmt.Errorf("alias lookup error: %w", err)
			}
		}
		return "", ErrAliasNotFound
	}

	for _, pid := range candidatePIDs {
		// 1. Primary lookup under canonical provider code (e.g. SPORTSXBET, ODIBETS, SPORTRADAR)
		cid, err := lookup(normProvider, pid)
		if err == nil && cid != "" {
			return cid, nil
		}
		if err != nil && !errors.Is(err, ErrAliasNotFound) {
			return "", err
		}

		// 2. Backward-compatibility fallback: check raw provider string if different (e.g. "sportybet")
		if rawProvider != normProvider {
			cid, err := lookup(rawProvider, pid)
			if err == nil && cid != "" {
				return cid, nil
			}
			if err != nil && !errors.Is(err, ErrAliasNotFound) {
				return "", err
			}
		}
	}

	return "", ErrAliasNotFound
}

// ResolveOrCreateFixture resolves an incoming match event to its canonical fixture identity.
// If the alias already exists, it returns the canonical fixture.
// If not, it derives natural identity (sport + normalized teams + kickoff) to find or create the canonical fixture,
// attaches both the original and prefix-cleaned provider aliases, and hydrates L1 and L2 caches.
func (e *Engine) ResolveOrCreateFixture(
	ctx context.Context,
	provider, providerFixtureID string,
	sportID int,
	homeTeam, awayTeam string,
	startTime time.Time,
	compID int64,
	compName string,
	mappingType canonical.MappingType,
	source string,
) (*CanonicalFixture, error) {
	rawProvider := strings.TrimSpace(provider)
	origPID, cleanPID, candidatePIDs := NormalizeProviderFixtureID(providerFixtureID)
	if rawProvider == "" || len(candidatePIDs) == 0 {
		return nil, errors.New("provider and providerFixtureID are required")
	}

	normProvider := canonical.NormalizeProvider(rawProvider)
	if normProvider == "" || normProvider == "UNKNOWN" {
		normProvider = strings.ToUpper(rawProvider)
	}

	// Step 1: Check if alias already exists (checks canonical code + candidate IDs + legacy aliases)
	cid, err := e.ResolveCanonicalID(ctx, rawProvider, origPID)
	if err == nil && cid != "" {
		return e.GetCanonicalFixture(ctx, cid)
	}
	if err != nil && !errors.Is(err, ErrAliasNotFound) {
		return nil, err
	}

	// Step 2: Natural identity resolution (concurrency-safe creation/lookup)
	f, err := e.repo.GetOrCreateCanonicalFixture(ctx, sportID, homeTeam, awayTeam, startTime, compID, compName)
	if err != nil {
		return nil, fmt.Errorf("get or create canonical fixture: %w", err)
	}

	// Step 3: Attach provider alias to the canonical fixture under canonical provider code.
	// If the authoritative DB rejects the alias (e.g. conflict), fail closed without polluting cache.
	_, err = e.repo.AttachAlias(ctx, f.CanonicalFixtureID, normProvider, origPID, mappingType, source)
	if err != nil {
		return nil, fmt.Errorf("attach provider alias: %w", err)
	}
	if cleanPID != "" && cleanPID != origPID {
		_, _ = e.repo.AttachAlias(ctx, f.CanonicalFixtureID, normProvider, cleanPID, mappingType, source)
	}

	// Also attach raw provider alias if different for backward-compatibility lookup in legacy DB
	if normProvider != rawProvider {
		_, _ = e.repo.AttachAlias(ctx, f.CanonicalFixtureID, rawProvider, origPID, mappingType, source)
	}

	// Step 4: Populate L1 and L2 caches after authoritative DB persistence succeeds
	if e.cache != nil {
		e.backfillAliasCache(ctx, rawProvider, normProvider, f.CanonicalFixtureID, origPID, cleanPID)
		e.cache.PutFixture(ctx, f)
	}

	return f, nil
}

// AttachAlias explicitly maps an external provider ID (e.g. Odibets Live ID or AIScore Result ID)
// to an existing canonical fixture and updates the tiered cache.
func (e *Engine) AttachAlias(
	ctx context.Context,
	canonicalID, provider, providerFixtureID string,
	mappingType canonical.MappingType,
	source string,
) (*FixtureAlias, error) {
	rawProvider := strings.TrimSpace(provider)
	origPID, cleanPID, candidatePIDs := NormalizeProviderFixtureID(providerFixtureID)
	if rawProvider == "" || len(candidatePIDs) == 0 {
		return nil, errors.New("provider and providerFixtureID are required")
	}
	if err := canonical.ValidateCanonicalFixtureID(canonicalID); err != nil {
		return nil, err
	}

	normProvider := canonical.NormalizeProvider(rawProvider)
	if normProvider == "" || normProvider == "UNKNOWN" {
		normProvider = strings.ToUpper(rawProvider)
	}

	alias, err := e.repo.AttachAlias(ctx, canonicalID, normProvider, origPID, mappingType, source)
	if err != nil {
		return nil, err
	}
	if cleanPID != "" && cleanPID != origPID {
		_, _ = e.repo.AttachAlias(ctx, canonicalID, normProvider, cleanPID, mappingType, source)
	}
	if rawProvider != normProvider {
		_, _ = e.repo.AttachAlias(ctx, canonicalID, rawProvider, origPID, mappingType, source)
	}

	if e.cache != nil {
		e.backfillAliasCache(ctx, rawProvider, normProvider, canonicalID, origPID, cleanPID)
	}

	return alias, nil
}

// GetCanonicalFixture fetches a canonical fixture by its canonical ID (L1 -> L2 -> DB).
func (e *Engine) GetCanonicalFixture(ctx context.Context, canonicalID string) (*CanonicalFixture, error) {
	if canonicalID == "" {
		return nil, errors.New("canonicalID is required")
	}

	// 1. Check L1 cache
	if e.cache != nil {
		if f, ok := e.cache.l1.GetFixture(canonicalID); ok && f != nil {
			return f, nil
		}
	}

	// 2. Query repository
	f, err := e.repo.GetCanonicalFixture(ctx, canonicalID)
	if err != nil {
		return nil, err
	}

	// Backfill cache
	if e.cache != nil && f != nil {
		e.cache.PutFixture(ctx, f)
	}

	return f, nil
}

// ResolveJITLiveEvent performs Grade-A in-play identity resolution for live events (including cold boots).
// Hierarchy: L1 Local RAM -> L2 Redis -> L3 MySQL Alias -> Deterministic Natural Candidate Evaluation.
// Guarantees:
// 1. One canonical ID per real match.
// 2. Dual providers attach to the same canonical ID.
// 3. Ambiguous fixtures are quarantined (cannot be opened for betting).
// 4. Atomic transaction prevents race conditions and duplicate creation under concurrency.
func (e *Engine) ResolveJITLiveEvent(
	ctx context.Context,
	candidate canonical.NaturalCandidate,
	provider, providerFixtureID string,
	source string,
) (*CanonicalFixture, *FixtureAlias, error) {
	rawProvider := strings.TrimSpace(provider)
	origPID, cleanPID, candidatePIDs := NormalizeProviderFixtureID(providerFixtureID)
	if rawProvider == "" || len(candidatePIDs) == 0 {
		return nil, nil, errors.New("provider and providerFixtureID are required")
	}

	normProvider := canonical.NormalizeProvider(rawProvider)
	if normProvider == "" || normProvider == "UNKNOWN" {
		normProvider = strings.ToUpper(rawProvider)
	}

	// 1. Hot path: check L1/L2/L3 alias mapping
	cid, err := e.ResolveCanonicalID(ctx, rawProvider, origPID)
	if err == nil && cid != "" {
		f, fErr := e.GetCanonicalFixture(ctx, cid)
		if fErr != nil {
			return nil, nil, fErr
		}
		alias, aErr := e.repo.GetAlias(ctx, normProvider, origPID)
		if aErr != nil && cleanPID != origPID {
			alias, aErr = e.repo.GetAlias(ctx, normProvider, cleanPID)
		}
		if aErr != nil && rawProvider != normProvider {
			alias, aErr = e.repo.GetAlias(ctx, rawProvider, origPID)
		}
		if aErr != nil {
			return f, nil, nil
		}
		return f, alias, nil
	}
	if err != nil && !errors.Is(err, ErrAliasNotFound) {
		return nil, nil, err
	}

	// 2. Evaluate candidate attributes
	candidate.CleanAndClassifyCandidate()
	outcome, existingFixture, rErr := e.repo.ResolveCandidate(ctx, candidate)
	if rErr != nil {
		return nil, nil, fmt.Errorf("resolve candidate: %w", rErr)
	}

	switch outcome {
	case OutcomeAmbiguous:
		// Settleability & Identity Invariant: Ambiguous fixtures are quarantined immediately
		return nil, nil, ErrAmbiguousFixture

	case OutcomeExistingMatch:
		// Natural candidate matched an existing canonical fixture
		if existingFixture == nil {
			return nil, nil, errors.New("existing fixture match returned nil fixture")
		}
		alias, aErr := e.repo.AttachAlias(ctx, existingFixture.CanonicalFixtureID, normProvider, origPID, canonical.MappingAutomatic, source)
		if aErr != nil {
			return nil, nil, fmt.Errorf("attach alias to existing fixture: %w", aErr)
		}
		if cleanPID != "" && cleanPID != origPID {
			_, _ = e.repo.AttachAlias(ctx, existingFixture.CanonicalFixtureID, normProvider, cleanPID, canonical.MappingAutomatic, source)
		}
		if e.cache != nil {
			e.backfillAliasCache(ctx, rawProvider, normProvider, existingFixture.CanonicalFixtureID, origPID, cleanPID)
			e.cache.PutFixture(ctx, existingFixture)
		}
		return existingFixture, alias, nil

	case OutcomeConfidentNewMatch:
		// Confident new match: execute atomic transaction to create canonical fixture and attach alias
		f, alias, cErr := e.repo.CreateJITCanonicalFixtureTx(ctx, candidate, normProvider, origPID, canonical.MappingAutomatic, source)
		if cErr != nil {
			return nil, nil, fmt.Errorf("create jit canonical fixture tx: %w", cErr)
		}
		if cleanPID != "" && cleanPID != origPID && f != nil {
			_, _ = e.repo.AttachAlias(ctx, f.CanonicalFixtureID, normProvider, cleanPID, canonical.MappingAutomatic, source)
		}
		if e.cache != nil && f != nil {
			e.backfillAliasCache(ctx, rawProvider, normProvider, f.CanonicalFixtureID, origPID, cleanPID)
			e.cache.PutFixture(ctx, f)
		}
		return f, alias, nil

	default:
		return nil, nil, fmt.Errorf("unsupported resolution outcome: %s", outcome)
	}
}

// EvaluateSettleability checks whether a canonical fixture meets the Grade-A Settleability Gate.
// If Settleable: returns true, BettingStateBettable.
// If Not Settleable: returns false, BettingStateLiveBroadcastOnly.
func (e *Engine) EvaluateSettleability(ctx context.Context, canonicalID string) (bool, canonical.BettingState, error) {
	if canonicalID == "" {
		return false, canonical.BettingSuspended, errors.New("canonicalID is required")
	}

	f, err := e.GetCanonicalFixture(ctx, canonicalID)
	if err != nil {
		return false, canonical.BettingSuspended, err
	}

	if f == nil {
		return false, canonical.BettingSuspended, ErrFixtureNotFound
	}

	// 1. Quarantined check
	if f.IdentityState == canonical.IdentityQuarantined {
		return false, canonical.BettingLiveBroadcastOnly, nil
	}

	// 2. Settleability gate: Must have SettlementCapable = true and a valid ResultResolutionKey
	if !f.SettlementCapable || f.ResultResolutionKey == "" {
		return false, canonical.BettingLiveBroadcastOnly, nil
	}

	// 3. Must have at least one active alias with a valid result resolution path
	aliases, err := e.repo.GetAliasesForCanonical(ctx, canonicalID)
	if err == nil && len(aliases) > 0 {
		hasActivePath := false
		for _, a := range aliases {
			if a.IsActive && a.ResultResolutionPath != "" {
				hasActivePath = true
				break
			}
		}
		if !hasActivePath {
			return false, canonical.BettingLiveBroadcastOnly, nil
		}
	}

	return true, canonical.BettingBettable, nil
}
