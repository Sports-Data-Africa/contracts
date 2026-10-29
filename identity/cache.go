package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Sports-Data-Africa/contracts/canonical"
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

func makeAliasKey(provider, providerFixtureID string) string {
	normProvider := canonical.NormalizeProvider(provider)
	return fmt.Sprintf("%s:%s", normProvider, strings.TrimSpace(providerFixtureID))
}

func (c *L1Cache) GetCanonicalID(provider, providerFixtureID string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	trimmedPID := strings.TrimSpace(providerFixtureID)
	if cid, ok := c.aliases[makeAliasKey(provider, trimmedPID)]; ok && cid != "" {
		return cid, true
	}
	rawProv := strings.TrimSpace(provider)
	if cid, ok := c.aliases[fmt.Sprintf("%s:%s", strings.ToUpper(rawProv), trimmedPID)]; ok && cid != "" {
		return cid, true
	}
	if cid, ok := c.aliases[fmt.Sprintf("%s:%s", rawProv, trimmedPID)]; ok && cid != "" {
		return cid, true
	}
	cid, ok := c.aliases[fmt.Sprintf("%s:%s", strings.ToLower(rawProv), trimmedPID)]
	return cid, ok && cid != ""
}

func (c *L1Cache) SetAlias(provider, providerFixtureID, canonicalID string) {
	trimmedPID := strings.TrimSpace(providerFixtureID)
	canonicalID = strings.TrimSpace(canonicalID)
	if trimmedPID == "" || canonicalID == "" {
		return
	}
	normProvider := canonical.NormalizeProvider(provider)
	rawProv := strings.TrimSpace(provider)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.aliases[fmt.Sprintf("%s:%s", normProvider, trimmedPID)] = canonicalID
	if rawProv != "" {
		c.aliases[fmt.Sprintf("%s:%s", strings.ToUpper(rawProv), trimmedPID)] = canonicalID
		c.aliases[fmt.Sprintf("%s:%s", rawProv, trimmedPID)] = canonicalID
		c.aliases[fmt.Sprintf("%s:%s", strings.ToLower(rawProv), trimmedPID)] = canonicalID
	}
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
	l1    *L1Cache
	l2    RedisClient
	l2TTL time.Duration
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
	trimmedPID := strings.TrimSpace(providerFixtureID)
	if trimmedPID == "" {
		return "", false
	}

	// 1. Check L1 local cache
	if cid, found := tc.l1.GetCanonicalID(provider, trimmedPID); found {
		return cid, true
	}

	// 2. Check L2 Redis cache if configured
	if tc.l2 != nil {
		normProvider := canonical.NormalizeProvider(provider)
		rawProv := strings.TrimSpace(provider)

		keys := []string{
			fmt.Sprintf("identity:alias:%s:%s", normProvider, trimmedPID),
		}
		if rawProv != "" && rawProv != normProvider {
			keys = append(keys, fmt.Sprintf("identity:alias:%s:%s", rawProv, trimmedPID))
		}
		upperProv := strings.ToUpper(rawProv)
		if upperProv != "" && upperProv != normProvider && upperProv != rawProv {
			keys = append(keys, fmt.Sprintf("identity:alias:%s:%s", upperProv, trimmedPID))
		}

		for _, key := range keys {
			if cid, err := tc.l2.Get(ctx, key); err == nil && cid != "" {
				tc.l1.SetAlias(normProvider, trimmedPID, cid)
				if rawProv != "" && rawProv != normProvider {
					tc.l1.SetAlias(rawProv, trimmedPID, cid)
				}
				return cid, true
			}
		}
	}

	return "", false
}

// PutAlias stores the alias in both L1 and L2 under the canonical normalized provider key.
func (tc *TieredIdentityCache) PutAlias(ctx context.Context, provider, providerFixtureID, canonicalID string) {
	trimmedPID := strings.TrimSpace(providerFixtureID)
	canonicalID = strings.TrimSpace(canonicalID)
	if trimmedPID == "" || canonicalID == "" {
		return
	}
	normProvider := canonical.NormalizeProvider(provider)
	tc.l1.SetAlias(normProvider, trimmedPID, canonicalID)
	rawProv := strings.TrimSpace(provider)
	if rawProv != "" && rawProv != normProvider {
		tc.l1.SetAlias(rawProv, trimmedPID, canonicalID)
	}
	if tc.l2 != nil {
		normKey := fmt.Sprintf("identity:alias:%s:%s", normProvider, trimmedPID)
		_ = tc.l2.Set(ctx, normKey, canonicalID, tc.l2TTL)
		if rawProv != "" && rawProv != normProvider {
			rawKey := fmt.Sprintf("identity:alias:%s:%s", rawProv, trimmedPID)
			_ = tc.l2.Set(ctx, rawKey, canonicalID, tc.l2TTL)
		}
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

// Invalidate removes an alias from L1 and L2 across normalized and raw provider keys.
func (tc *TieredIdentityCache) Invalidate(ctx context.Context, provider, providerFixtureID string) {
	trimmedPID := strings.TrimSpace(providerFixtureID)
	normProvider := canonical.NormalizeProvider(provider)
	rawProv := strings.TrimSpace(provider)

	tc.l1.mu.Lock()
	delete(tc.l1.aliases, fmt.Sprintf("%s:%s", normProvider, trimmedPID))
	delete(tc.l1.aliases, fmt.Sprintf("%s:%s", strings.ToUpper(rawProv), trimmedPID))
	delete(tc.l1.aliases, fmt.Sprintf("%s:%s", rawProv, trimmedPID))
	delete(tc.l1.aliases, fmt.Sprintf("%s:%s", strings.ToLower(rawProv), trimmedPID))
	tc.l1.mu.Unlock()

	if tc.l2 != nil {
		_ = tc.l2.Del(
			ctx,
			fmt.Sprintf("identity:alias:%s:%s", normProvider, trimmedPID),
			fmt.Sprintf("identity:alias:%s:%s", rawProv, trimmedPID),
			fmt.Sprintf("identity:alias:%s:%s", strings.ToUpper(rawProv), trimmedPID),
		)
	}
}
