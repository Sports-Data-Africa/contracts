package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// RedisClient defines the minimal interface required for L2 distributed caching.
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

// L1Cache is an in-memory, thread-safe cache for sub-millisecond local process lookups.
type L1Cache struct {
	mu       sync.RWMutex
	aliases  map[string]string            // key: "provider:provider_fixture_id" -> canonical_fixture_id
	fixtures map[string]*CanonicalFixture // key: canonical_fixture_id -> CanonicalFixture
}

func NewL1Cache() *L1Cache {
	return &L1Cache{
		aliases:  make(map[string]string),
		fixtures: make(map[string]*CanonicalFixture),
	}
}

func (c *L1Cache) GetCanonicalID(provider, providerFixtureID string) (string, bool) {
	key := fmt.Sprintf("%s:%s", provider, providerFixtureID)
	c.mu.RLock()
	defer c.mu.RUnlock()
	cid, ok := c.aliases[key]
	return cid, ok
}

func (c *L1Cache) SetAlias(provider, providerFixtureID, canonicalID string) {
	key := fmt.Sprintf("%s:%s", provider, providerFixtureID)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aliases[key] = canonicalID
}

func (c *L1Cache) GetFixture(canonicalID string) (*CanonicalFixture, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	f, ok := c.fixtures[canonicalID]
	return f, ok
}

func (c *L1Cache) SetFixture(f *CanonicalFixture) {
	if f == nil || f.CanonicalFixtureID == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fixtures[f.CanonicalFixtureID] = f
}

func (c *L1Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aliases = make(map[string]string)
	c.fixtures = make(map[string]*CanonicalFixture)
}

// TieredIdentityCache coordinates L1 (local process memory) and L2 (Redis).
type TieredIdentityCache struct {
	l1       *L1Cache
	l2       RedisClient
	l2TTL    time.Duration
}

func NewTieredIdentityCache(l2 RedisClient, l2TTL time.Duration) *TieredIdentityCache {
	if l2TTL <= 0 {
		l2TTL = 24 * time.Hour
	}
	return &TieredIdentityCache{
		l1:    NewL1Cache(),
		l2:    l2,
		l2TTL: l2TTL,
	}
}

// ResolveLocal returns the canonical fixture ID directly from L1 process memory (zero I/O).
func (tc *TieredIdentityCache) ResolveLocal(provider, providerFixtureID string) (string, bool) {
	return tc.l1.GetCanonicalID(provider, providerFixtureID)
}

// ResolveTiered checks L1, then L2 (Redis). If L2 hits, it backfills L1.
func (tc *TieredIdentityCache) ResolveTiered(ctx context.Context, provider, providerFixtureID string) (string, bool) {
	// 1. Check L1 local cache
	if cid, found := tc.l1.GetCanonicalID(provider, providerFixtureID); found {
		return cid, true
	}

	// 2. Check L2 Redis cache if configured
	if tc.l2 != nil {
		key := fmt.Sprintf("identity:alias:%s:%s", provider, providerFixtureID)
		cid, err := tc.l2.Get(ctx, key)
		if err == nil && cid != "" {
			// Backfill L1
			tc.l1.SetAlias(provider, providerFixtureID, cid)
			return cid, true
		}
	}

	return "", false
}

// PutAlias stores the alias in both L1 and L2.
func (tc *TieredIdentityCache) PutAlias(ctx context.Context, provider, providerFixtureID, canonicalID string) {
	tc.l1.SetAlias(provider, providerFixtureID, canonicalID)
	if tc.l2 != nil {
		key := fmt.Sprintf("identity:alias:%s:%s", provider, providerFixtureID)
		_ = tc.l2.Set(ctx, key, canonicalID, tc.l2TTL)
	}
}

// PutFixture stores the full fixture in L1 and optionally L2.
func (tc *TieredIdentityCache) PutFixture(ctx context.Context, f *CanonicalFixture) {
	if f == nil || f.CanonicalFixtureID == "" {
		return
	}
	tc.l1.SetFixture(f)
	if tc.l2 != nil {
		key := fmt.Sprintf("identity:fixture:%s", f.CanonicalFixtureID)
		data, err := json.Marshal(f)
		if err == nil {
			_ = tc.l2.Set(ctx, key, string(data), tc.l2TTL)
		}
	}
}

// Invalidate removes an alias from L1 and L2.
func (tc *TieredIdentityCache) Invalidate(ctx context.Context, provider, providerFixtureID string) {
	key := fmt.Sprintf("%s:%s", provider, providerFixtureID)
	tc.l1.mu.Lock()
	delete(tc.l1.aliases, key)
	tc.l1.mu.Unlock()

	if tc.l2 != nil {
		_ = tc.l2.Del(ctx, fmt.Sprintf("identity:alias:%s:%s", provider, providerFixtureID))
	}
}
