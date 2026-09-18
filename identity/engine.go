package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sports-Data-Africa/contracts/canonical"
)

// Engine is the single authoritative identity resolution engine for the platform.
// It coordinates L1 local process memory, L2 Redis distributed caching, and MySQL persistence.
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

// ResolveCanonicalID is the hot-path identity lookup.
// 1. L1 local process memory lookup (zero I/O, sub-millisecond).
// 2. L2 Redis lookup (backfills L1 on hit).
// 3. MySQL repository lookup (backfills L2 and L1 on hit).
// Returns ErrAliasNotFound if the provider fixture ID is not mapped.
func (e *Engine) ResolveCanonicalID(ctx context.Context, provider, providerFixtureID string) (string, error) {
	if provider == "" || providerFixtureID == "" {
		return "", errors.New("provider and provider_fixture_id are required")
	}

	// 1. Check L1 local cache (hot path)
	if e.cache != nil {
		if cid, ok := e.cache.ResolveLocal(provider, providerFixtureID); ok && cid != "" {
			return cid, nil
		}

		// 2. Check L2 Redis cache
		if cid, ok := e.cache.ResolveTiered(ctx, provider, providerFixtureID); ok && cid != "" {
			return cid, nil
		}
	}

	// 3. Check MySQL repository
	if e.repo != nil {
		alias, err := e.repo.GetAlias(ctx, provider, providerFixtureID)
		if err == nil && alias != nil {
			if e.cache != nil {
				e.cache.PutAlias(ctx, provider, providerFixtureID, alias.CanonicalFixtureID)
			}
			return alias.CanonicalFixtureID, nil
		}
		if !errors.Is(err, ErrAliasNotFound) {
			return "", fmt.Errorf("alias lookup error: %w", err)
		}
	}

	return "", ErrAliasNotFound
}

// ResolveOrCreateFixture resolves an incoming match event to its canonical identity.
// If the alias already exists, it returns the canonical fixture.
// If not, it derives natural identity (sport + normalized teams + kickoff) to find or create the canonical fixture,
// attaches the provider alias, and hydrates L1 and L2 caches.
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
	if provider == "" || providerFixtureID == "" {
		return nil, errors.New("provider and providerFixtureID are required")
	}

	// Step 1: Check if alias already exists
	cid, err := e.ResolveCanonicalID(ctx, provider, providerFixtureID)
	if err == nil && cid != "" {
		// Existing mapping found; fetch full canonical fixture
		return e.GetCanonicalFixture(ctx, cid)
	}

	// Step 2: Natural identity resolution (concurrency-safe creation/lookup)
	f, err := e.repo.GetOrCreateCanonicalFixture(ctx, sportID, homeTeam, awayTeam, startTime, compID, compName)
	if err != nil {
		return nil, fmt.Errorf("get or create canonical fixture: %w", err)
	}

	// Step 3: Attach provider alias to the canonical fixture
	_, err = e.repo.AttachAlias(ctx, f.CanonicalFixtureID, provider, providerFixtureID, mappingType, source)
	if err != nil {
		return nil, fmt.Errorf("attach provider alias: %w", err)
	}

	// Step 4: Populate L1 and L2 caches
	if e.cache != nil {
		e.cache.PutAlias(ctx, provider, providerFixtureID, f.CanonicalFixtureID)
		e.cache.PutFixture(ctx, f)
	}

	return f, nil
}

// AttachAlias explicitly maps an external provider ID (e.g. Paripesa Live ID or AIScore Result ID)
// to an existing canonical fixture and updates the tiered cache.
func (e *Engine) AttachAlias(
	ctx context.Context,
	canonicalID, provider, providerFixtureID string,
	mappingType canonical.MappingType,
	source string,
) (*FixtureAlias, error) {
	if err := canonical.ValidateCanonicalFixtureID(canonicalID); err != nil {
		return nil, err
	}

	alias, err := e.repo.AttachAlias(ctx, canonicalID, provider, providerFixtureID, mappingType, source)
	if err != nil {
		return nil, err
	}

	if e.cache != nil {
		e.cache.PutAlias(ctx, provider, providerFixtureID, canonicalID)
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
