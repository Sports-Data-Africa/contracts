package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("AUTHENTICATION_FAILED")
	ErrAccountInactive    = errors.New("ACCOUNT_INACTIVE")
	ErrAccountSuspended   = errors.New("ACCOUNT_SUSPENDED")
	ErrAccountTerminated  = errors.New("ACCOUNT_TERMINATED")
	ErrEntitlementMissing = errors.New("PRODUCT_ENTITLEMENT_REQUIRED")
	ErrSportNotAllowed    = errors.New("SPORT_ACCESS_DENIED")
	ErrTokenExpired       = errors.New("TOKEN_EXPIRED")
	ErrInvalidToken       = errors.New("INVALID_TOKEN")
	ErrAPIKeyRevoked      = errors.New("API_KEY_REVOKED")
	ErrAPIKeyGraceExpired = errors.New("API_KEY_GRACE_EXPIRED")
	ErrEnvMismatch        = errors.New("ENVIRONMENT_MISMATCH")
)

const (
	ScopePrematchRead   = "SPORTS_PREMATCH"
	ScopeLiveStream     = "SPORTS_LIVE"
	ScopeBetPlacement   = "BETTING"
	ScopeSettlementRead = "SETTLEMENT"
	ScopeKeyManage      = "API_KEYS_MANAGE"

	EnvProduction = "PRODUCTION"
	EnvStaging    = "STAGING"
	EnvSandbox    = "SANDBOX"

	KeyStatusActive  = "ACTIVE"
	KeyStatusGrace   = "GRACE_PERIOD"
	KeyStatusRevoked = "REVOKED"

	PlanFeedOnly     = "FEED_ONLY"
	PlanFullPlatform = "FULL_PLATFORM"
	PlanEnterprise   = "ENTERPRISE"
)

// ClientAuthContext is the canonical internal authorization object across all SDA services.
type ClientAuthContext struct {
	ClientID      string    `json:"client_id"`
	ClientName    string    `json:"client_name"`
	Plan          string    `json:"plan"`
	Environment   string    `json:"environment"`
	Scopes        []string  `json:"scopes"`
	AllowedSports []int32   `json:"allowed_sports"` // Empty or nil means all globally released sports
	KeyID         string    `json:"key_id"`
	AuthMethod    string    `json:"auth_method"` // "API_KEY" or "WS_TOKEN"
	ExpiresAt     time.Time `json:"expires_at"`
}


// HasScope checks whether the authenticated client has the required permission/scope.
func (c *ClientAuthContext) HasScope(required string) bool {
	if c == nil {
		return false
	}
	if c.Plan == PlanEnterprise {
		return true // Enterprise plan has universal access to platform scopes
	}
	for _, s := range c.Scopes {
		if strings.EqualFold(s, required) {
			return true
		}
	}
	return false
}

// IsSportAllowed checks whether the client is entitled to access the given sport ID.
func (c *ClientAuthContext) IsSportAllowed(sportID int32) bool {
	if c == nil {
		return false
	}
	// Table tennis (ID 10) is internal-only globally and always denied
	if sportID == 10 {
		return false
	}
	if len(c.AllowedSports) == 0 {
		return true // Default: entitled to all globally released client sports
	}
	for _, s := range c.AllowedSports {
		if s == sportID {
			return true
		}
	}
	return false
}

// ClientAuthInvalidatedEvent is published when a key is revoked, rotated, or client suspended.
// Downstream microservices consume this event to invalidate their local L1 caches immediately.
type ClientAuthInvalidatedEvent struct {
	ClientID    string    `json:"client_id"`
	KeyID       string    `json:"key_id,omitempty"`
	KeyHashHint string    `json:"key_hash_hint,omitempty"`
	Reason      string    `json:"reason"`
	EventID     string    `json:"event_id"`
	OccurredAt  time.Time `json:"occurred_at"`
}


// HashKey computes SHA-256 of raw API key string
func HashKey(rawKey string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(rawKey)))
	return hex.EncodeToString(sum[:])
}

// CacheEntry wraps ClientAuthContext with an expiration timestamp
type CacheEntry struct {
	Context   *ClientAuthContext
	ExpiresAt time.Time
}

// L1Cache provides a bounded, high-performance in-memory cache for API key validation results.
// Maximum capacity is 50,000 entries. On overflow, expired entries are evicted first;
// if still over capacity, a random sample of entries is evicted (prevents unbounded heap growth).
type L1Cache struct {
	mu       sync.RWMutex
	entries  map[string]CacheEntry // key: sha256(raw_key) or token string
	ttl      time.Duration
	maxSize  int
}

const defaultL1MaxSize = 50_000

// NewL1Cache initializes a bounded L1 in-memory auth cache with the specified TTL.
func NewL1Cache(ttl time.Duration) *L1Cache {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return &L1Cache{
		entries: make(map[string]CacheEntry),
		ttl:     ttl,
		maxSize: defaultL1MaxSize,
	}
}

// Get retrieves a cached ClientAuthContext if present and unexpired.
func (c *L1Cache) Get(keyHash string) (*ClientAuthContext, bool) {
	c.mu.RLock()
	entry, exists := c.entries[keyHash]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}
	if time.Now().After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.entries, keyHash)
		c.mu.Unlock()
		return nil, false
	}
	return entry.Context, true
}

// Set stores a ClientAuthContext in the cache, enforcing the capacity bound.
func (c *L1Cache) Set(keyHash string, ctx *ClientAuthContext) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If already at capacity, evict expired entries first, then random entries
	if len(c.entries) >= c.maxSize {
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.ExpiresAt) {
				delete(c.entries, k)
			}
			// Limit eviction loop to avoid long lock holds
			if len(c.entries) < c.maxSize {
				break
			}
		}
		// If still at capacity after expiry eviction, evict a random batch
		if len(c.entries) >= c.maxSize {
			evicted := 0
			for k := range c.entries {
				delete(c.entries, k)
				evicted++
				if evicted >= 500 {
					break
				}
			}
		}
	}

	c.entries[keyHash] = CacheEntry{
		Context:   ctx,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Invalidate removes entries for a specific keyHash or clientID.
func (c *L1Cache) Invalidate(clientID string, keyHash string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if keyHash != "" {
		delete(c.entries, keyHash)
	}
	if clientID != "" {
		for k, v := range c.entries {
			if v.Context != nil && v.Context.ClientID == clientID {
				delete(c.entries, k)
			}
		}
	}
}

// Clear flushes all entries from L1 cache.
func (c *L1Cache) Clear() {
	c.mu.Lock()
	c.entries = make(map[string]CacheEntry)
	c.mu.Unlock()
}

// Size returns the current number of cache entries.
func (c *L1Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
