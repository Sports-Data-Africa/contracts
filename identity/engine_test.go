package identity

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Sports-Data-Africa/contracts/canonical"
)

// MockRedis provides an in-memory implementation of RedisClient for testing.
type MockRedis struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewMockRedis() *MockRedis {
	return &MockRedis{data: make(map[string]string)}
}

func (m *MockRedis) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	if !ok {
		return "", errors.New("redis: nil")
	}
	return v, nil
}

func (m *MockRedis) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *MockRedis) Del(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

// In-memory mock repository for tests without MySQL
type MockRepository struct {
	mu        sync.RWMutex
	fixtures  map[string]*CanonicalFixture
	byHash    map[string]*CanonicalFixture
	aliases   map[string]*FixtureAlias
	conflicts []*IdentityConflict
	seq       int64
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		fixtures:  make(map[string]*CanonicalFixture),
		byHash:    make(map[string]*CanonicalFixture),
		aliases:   make(map[string]*FixtureAlias),
		conflicts: make([]*IdentityConflict, 0),
		seq:       100000,
	}
}

func (r *MockRepository) InitSchema(ctx context.Context) error { return nil }

func (r *MockRepository) GetCanonicalFixture(ctx context.Context, canonicalID string) (*CanonicalFixture, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.fixtures[canonicalID]
	if !ok {
		return nil, ErrFixtureNotFound
	}
	return f, nil
}

func (r *MockRepository) GetCanonicalFixtureByNaturalHash(ctx context.Context, hash string) (*CanonicalFixture, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.byHash[hash]
	if !ok {
		return nil, ErrFixtureNotFound
	}
	return f, nil
}

func (r *MockRepository) GetOrCreateCanonicalFixture(ctx context.Context, sportID int, homeTeam, awayTeam string, startTime time.Time, compID int64, compName string) (*CanonicalFixture, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	hash := canonical.NaturalIdentityHash(sportID, homeTeam, awayTeam, startTime)
	if existing, ok := r.byHash[hash]; ok {
		return existing, nil
	}

	r.seq++
	cid, err := canonical.FormatCanonicalFixtureID(sportID, r.seq)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	f := &CanonicalFixture{
		ID:                  r.seq,
		CanonicalFixtureID:  cid,
		SequenceNumber:      r.seq,
		SportID:             sportID,
		SportCode:           canonical.SportIDToCode[sportID],
		HomeTeam:            homeTeam,
		AwayTeam:            awayTeam,
		ScheduledStart:      startTime,
		CompetitionID:       compID,
		CompetitionName:     compName,
		NaturalIdentityHash: hash,
		IdentityState:       canonical.IdentityIdentified,
		VerificationState:   canonical.VerificationPending,
		FixtureState:        canonical.FixtureNotStarted,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	r.fixtures[cid] = f
	r.byHash[hash] = f
	return f, nil
}

func (r *MockRepository) GetAlias(ctx context.Context, provider, providerFixtureID string) (*FixtureAlias, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", provider, providerFixtureID)
	a, ok := r.aliases[key]
	if !ok {
		return nil, ErrAliasNotFound
	}
	return a, nil
}

func (r *MockRepository) AttachAlias(ctx context.Context, canonicalID, provider, providerFixtureID string, mappingType canonical.MappingType, source string) (*FixtureAlias, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s:%s", provider, providerFixtureID)
	if existing, ok := r.aliases[key]; ok {
		if existing.CanonicalFixtureID == canonicalID {
			existing.LastSeenAt = time.Now().UTC()
			return existing, nil
		}
		// Conflict!
		c := &IdentityConflict{
			ConflictType:        "ALIAS_REASSIGNMENT_ATTEMPT",
			Provider:            provider,
			ProviderFixtureID:   providerFixtureID,
			ExistingCanonicalID: existing.CanonicalFixtureID,
			ProposedCanonicalID: canonicalID,
			Status:              "UNRESOLVED",
			CreatedAt:           time.Now().UTC(),
		}
		r.conflicts = append(r.conflicts, c)
		return nil, fmt.Errorf("%w: already assigned to %s", ErrIdentityConflict, existing.CanonicalFixtureID)
	}

	now := time.Now().UTC()
	alias := &FixtureAlias{
		ID:                 int64(len(r.aliases) + 1),
		CanonicalFixtureID: canonicalID,
		Provider:           provider,
		ProviderFixtureID:  providerFixtureID,
		MappingType:        mappingType,
		Source:             source,
		FirstSeenAt:        now,
		LastSeenAt:         now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	r.aliases[key] = alias
	return alias, nil
}

func (r *MockRepository) GetAliasesForCanonical(ctx context.Context, canonicalID string) ([]FixtureAlias, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []FixtureAlias
	for _, a := range r.aliases {
		if a.CanonicalFixtureID == canonicalID {
			list = append(list, *a)
		}
	}
	return list, nil
}

func (r *MockRepository) RecordConflict(ctx context.Context, conflict *IdentityConflict) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conflicts = append(r.conflicts, conflict)
	return nil
}

func TestCanonicalFixtureCreationAndDeduplication(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository()
	cache := NewTieredIdentityCache(NewMockRedis(), time.Hour)
	engine := NewEngine(repo, cache)

	startTime := time.Date(2026, 9, 18, 19, 0, 0, 0, time.UTC)

	// Ingest Prematch Match from Paripesa
	f1, err := engine.ResolveOrCreateFixture(
		ctx, "paripesa", "123456",
		1, "Arsenal FC", "Chelsea FC", startTime,
		10, "Premier League", canonical.MappingPrematch, "feed_vzip",
	)
	if err != nil {
		t.Fatalf("unexpected error creating fixture 1: %v", err)
	}
	if f1.CanonicalFixtureID != "SPD-FB-100001" {
		t.Errorf("expected canonical ID SPD-FB-100001, got %s", f1.CanonicalFixtureID)
	}

	// Ingest Same Match from SportyBet (Different Provider & Provider ID)
	f2, err := engine.ResolveOrCreateFixture(
		ctx, "sportybet", "sr:match:51849281",
		1, "arsenal", "chelsea", startTime, // Variations in team names
		10, "English Premier League", canonical.MappingPrematch, "sportradar",
	)
	if err != nil {
		t.Fatalf("unexpected error resolving fixture 2: %v", err)
	}

	// Must resolve to the EXACT SAME canonical fixture!
	if f1.CanonicalFixtureID != f2.CanonicalFixtureID {
		t.Fatalf("expected both providers to map to same canonical ID, got f1=%s, f2=%s", f1.CanonicalFixtureID, f2.CanonicalFixtureID)
	}
}

func TestMultipleProviderAliasesOneCanonicalFixture(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository()
	cache := NewTieredIdentityCache(NewMockRedis(), time.Hour)
	engine := NewEngine(repo, cache)

	startTime := time.Date(2026, 9, 18, 21, 0, 0, 0, time.UTC)

	// 1. Prematch alias
	f, err := engine.ResolveOrCreateFixture(
		ctx, "paripesa", "123456",
		1, "Liverpool", "Manchester City", startTime,
		10, "Premier League", canonical.MappingPrematch, "prematch_line",
	)
	if err != nil {
		t.Fatalf("failed to create prematch: %v", err)
	}

	// 2. Attach Live alias (Paripesa live ID differs from prematch ID!)
	_, err = engine.AttachAlias(ctx, f.CanonicalFixtureID, "paripesa", "987654", canonical.MappingLive, "live_socket")
	if err != nil {
		t.Fatalf("failed to attach live alias: %v", err)
	}

	// 3. Attach Result alias (AIScore result ID)
	_, err = engine.AttachAlias(ctx, f.CanonicalFixtureID, "aiscore", "AS-88219", canonical.MappingResult, "aiscore_api")
	if err != nil {
		t.Fatalf("failed to attach result alias: %v", err)
	}

	// Verify all three resolve to the exact same canonical fixture
	cPrematch, err := engine.ResolveCanonicalID(ctx, "paripesa", "123456")
	if err != nil || cPrematch != f.CanonicalFixtureID {
		t.Errorf("prematch resolve failed: got %s, err %v", cPrematch, err)
	}

	cLive, err := engine.ResolveCanonicalID(ctx, "paripesa", "987654")
	if err != nil || cLive != f.CanonicalFixtureID {
		t.Errorf("live resolve failed: got %s, err %v", cLive, err)
	}

	cResult, err := engine.ResolveCanonicalID(ctx, "aiscore", "AS-88219")
	if err != nil || cResult != f.CanonicalFixtureID {
		t.Errorf("result resolve failed: got %s, err %v", cResult, err)
	}
}

func TestIdentityConflictRejection(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository()
	cache := NewTieredIdentityCache(NewMockRedis(), time.Hour)
	engine := NewEngine(repo, cache)

	startTime := time.Date(2026, 9, 18, 19, 0, 0, 0, time.UTC)

	// Match 1
	f1, _ := engine.ResolveOrCreateFixture(
		ctx, "paripesa", "111111",
		1, "Arsenal", "Chelsea", startTime,
		1, "PL", canonical.MappingPrematch, "feed",
	)

	// Match 2 (different match)
	f2, _ := engine.ResolveOrCreateFixture(
		ctx, "paripesa", "222222",
		1, "Real Madrid", "Barcelona", startTime,
		2, "La Liga", canonical.MappingPrematch, "feed",
	)

	// Attempting to reassign provider ID "111111" to Match 2 must trigger an identity conflict
	_, err := engine.AttachAlias(ctx, f2.CanonicalFixtureID, "paripesa", "111111", canonical.MappingLive, "malicious_feed")
	if err == nil {
		t.Fatal("expected error on alias reassignment conflict, got nil")
	}
	if !errors.Is(err, ErrIdentityConflict) {
		t.Errorf("expected ErrIdentityConflict, got %v", err)
	}

	if len(repo.conflicts) != 1 {
		t.Errorf("expected 1 conflict recorded, got %d", len(repo.conflicts))
	}
	if repo.conflicts[0].ExistingCanonicalID != f1.CanonicalFixtureID {
		t.Errorf("expected conflict existing ID %s, got %s", f1.CanonicalFixtureID, repo.conflicts[0].ExistingCanonicalID)
	}
}

func TestTieredCacheResolution(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository()
	redis := NewMockRedis()
	cache := NewTieredIdentityCache(redis, time.Hour)
	engine := NewEngine(repo, cache)

	startTime := time.Date(2026, 9, 18, 15, 0, 0, 0, time.UTC)
	f, _ := engine.ResolveOrCreateFixture(
		ctx, "sportybet", "sr:match:9999",
		1, "Bayern Munich", "Dortmund", startTime,
		3, "Bundesliga", canonical.MappingPrematch, "sr",
	)

	// 1. Hot Path: L1 hit (local RAM)
	cidL1, err := engine.ResolveCanonicalID(ctx, "sportybet", "sr:match:9999")
	if err != nil || cidL1 != f.CanonicalFixtureID {
		t.Fatalf("expected L1 hit %s, got %s, err %v", f.CanonicalFixtureID, cidL1, err)
	}

	// 2. Simulate process restart: wipe L1 local cache
	cache.l1.Clear()

	// 3. Hot Path: L2 hit (Redis)
	cidL2, err := engine.ResolveCanonicalID(ctx, "sportybet", "sr:match:9999")
	if err != nil || cidL2 != f.CanonicalFixtureID {
		t.Fatalf("expected L2 hit %s, got %s, err %v", f.CanonicalFixtureID, cidL2, err)
	}

	// 4. Verify that L2 hit backfilled L1
	if _, ok := cache.l1.GetCanonicalID("sportybet", "sr:match:9999"); !ok {
		t.Errorf("expected L1 to be backfilled from L2")
	}

	// 5. Simulate Redis eviction & L1 wipe: fall back to MySQL repo
	cache.l1.Clear()
	_ = redis.Del(ctx, "identity:alias:sportybet:sr:match:9999")

	cidDB, err := engine.ResolveCanonicalID(ctx, "sportybet", "sr:match:9999")
	if err != nil || cidDB != f.CanonicalFixtureID {
		t.Fatalf("expected DB fallback hit %s, got %s, err %v", f.CanonicalFixtureID, cidDB, err)
	}

	// 6. Verify backfill of both L1 and L2
	if _, ok := cache.l1.GetCanonicalID("sportybet", "sr:match:9999"); !ok {
		t.Errorf("expected L1 to be backfilled from DB")
	}
	if redisVal, rErr := redis.Get(ctx, "identity:alias:sportybet:sr:match:9999"); rErr != nil || redisVal != f.CanonicalFixtureID {
		t.Errorf("expected Redis to be backfilled from DB, got %s, err %v", redisVal, rErr)
	}
}
