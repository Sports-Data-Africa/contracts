package auth

import (
	"context"
	"strings"
	"time"
)

// AuthValidator defines the authoritative interface for verifying credentials and issuing tokens.
type AuthValidator interface {
	ValidateAPIKey(ctx context.Context, rawKey string, requiredScope string, sportID int32) (*ClientAuthContext, error)
	ValidateWSToken(ctx context.Context, token string) (*ClientAuthContext, error)
	IssueWSToken(ctx context.Context, rawKey string, ttlSeconds int32) (token string, expiresAt int64, err error)
	Invalidate(clientID string, keyHash string)
}

// CachedClient decorates any AuthValidator with an L1 in-memory cache to guarantee ~15ns lookups on hot paths.
type CachedClient struct {
	validator AuthValidator
	cache     *L1Cache
}

// NewCachedClient initializes a CachedClient wrapping the underlying validator.
func NewCachedClient(validator AuthValidator, l1TTL time.Duration) *CachedClient {
	return &CachedClient{
		validator: validator,
		cache:     NewL1Cache(l1TTL),
	}
}

// ValidateAPIKey checks the in-memory L1 cache first; on miss, delegates to the validator.
func (c *CachedClient) ValidateAPIKey(ctx context.Context, rawKey string, requiredScope string, sportID int32) (*ClientAuthContext, error) {
	if strings.TrimSpace(rawKey) == "" {
		return nil, ErrInvalidCredentials
	}

	keyHash := HashKey(rawKey)

	// Tier 1: Process-local L1 RAM cache lookup
	if cachedCtx, found := c.cache.Get(keyHash); found {
		if requiredScope != "" && !cachedCtx.HasScope(requiredScope) {
			return nil, ErrEntitlementMissing
		}
		if sportID > 0 && !cachedCtx.IsSportAllowed(sportID) {
			return nil, ErrSportNotAllowed
		}
		return cachedCtx, nil
	}

	// Tier 2: Delegate to underlying validator (gRPC or L2/L3)
	authCtx, err := c.validator.ValidateAPIKey(ctx, rawKey, requiredScope, sportID)
	if err != nil {
		return nil, err
	}

	// Populate L1 cache on success
	c.cache.Set(keyHash, authCtx)
	return authCtx, nil
}

// ValidateWSToken checks the in-memory cache first; on miss, delegates to the validator.
func (c *CachedClient) ValidateWSToken(ctx context.Context, token string) (*ClientAuthContext, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrInvalidToken
	}

	tokenHash := HashKey(token)
	if cachedCtx, found := c.cache.Get(tokenHash); found {
		return cachedCtx, nil
	}

	authCtx, err := c.validator.ValidateWSToken(ctx, token)
	if err != nil {
		return nil, err
	}

	c.cache.Set(tokenHash, authCtx)
	return authCtx, nil
}

// IssueWSToken delegates directly to validator (tokens are created on demand, never read from cache).
func (c *CachedClient) IssueWSToken(ctx context.Context, rawKey string, ttlSeconds int32) (string, int64, error) {
	return c.validator.IssueWSToken(ctx, rawKey, ttlSeconds)
}

// Invalidate purges cache entries for a client or key upon receiving a client.auth.invalidated event.
func (c *CachedClient) Invalidate(clientID string, keyHash string) {
	c.cache.Invalidate(clientID, keyHash)
	c.validator.Invalidate(clientID, keyHash)
}

// HandleInvalidationEvent processes an incoming invalidation event and purges local memory.
func (c *CachedClient) HandleInvalidationEvent(evt *ClientAuthInvalidatedEvent) {
	if evt == nil {
		return
	}
	c.Invalidate(evt.ClientID, evt.KeyHashHint)
}

// MockAuthValidator provides an in-memory validator for standalone unit testing without network dependencies.
type MockAuthValidator struct {
	keys   map[string]*ClientAuthContext // key: hash(raw_key)
	tokens map[string]*ClientAuthContext // key: token
}

// NewMockAuthValidator creates a mock validator for unit/integration tests.
func NewMockAuthValidator() *MockAuthValidator {
	return &MockAuthValidator{
		keys:   make(map[string]*ClientAuthContext),
		tokens: make(map[string]*ClientAuthContext),
	}
}

// RegisterKey registers a mock API key and its associated context.
func (m *MockAuthValidator) RegisterKey(rawKey string, authCtx *ClientAuthContext) {
	keyHash := HashKey(rawKey)
	m.keys[keyHash] = authCtx
}

// ValidateAPIKey validates the raw key against registered mock keys.
func (m *MockAuthValidator) ValidateAPIKey(ctx context.Context, rawKey string, requiredScope string, sportID int32) (*ClientAuthContext, error) {
	keyHash := HashKey(rawKey)
	authCtx, exists := m.keys[keyHash]
	if !exists {
		return nil, ErrInvalidCredentials
	}
	if requiredScope != "" && !authCtx.HasScope(requiredScope) {
		return nil, ErrEntitlementMissing
	}
	if sportID > 0 && !authCtx.IsSportAllowed(sportID) {
		return nil, ErrSportNotAllowed
	}
	return authCtx, nil
}

// ValidateWSToken validates token against registered mock tokens.
func (m *MockAuthValidator) ValidateWSToken(ctx context.Context, token string) (*ClientAuthContext, error) {
	authCtx, exists := m.tokens[token]
	if !exists {
		return nil, ErrInvalidToken
	}
	if time.Now().After(authCtx.ExpiresAt) {
		return nil, ErrTokenExpired
	}
	return authCtx, nil
}

// IssueWSToken creates a mock token valid for the key's client.
func (m *MockAuthValidator) IssueWSToken(ctx context.Context, rawKey string, ttlSeconds int32) (string, int64, error) {
	keyHash := HashKey(rawKey)
	authCtx, exists := m.keys[keyHash]
	if !exists {
		return "", 0, ErrInvalidCredentials
	}
	if ttlSeconds <= 0 || ttlSeconds > 60 {
		ttlSeconds = 60
	}
	exp := time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	token := "ws_tok_" + keyHash[:12] + "_" + time.Now().Format("150405")
	m.tokens[token] = &ClientAuthContext{
		ClientID:      authCtx.ClientID,
		ClientName:    authCtx.ClientName,
		Plan:          authCtx.Plan,
		Environment:   authCtx.Environment,
		Scopes:        authCtx.Scopes,
		AllowedSports: authCtx.AllowedSports,
		KeyID:         authCtx.KeyID,
		AuthMethod:    "WS_TOKEN",
		ExpiresAt:     exp,
	}
	return token, exp.Unix(), nil
}

// Invalidate removes mock keys or tokens.
func (m *MockAuthValidator) Invalidate(clientID string, keyHash string) {
	if keyHash != "" {
		delete(m.keys, keyHash)
	}
	if clientID != "" {
		for k, v := range m.keys {
			if v.ClientID == clientID {
				delete(m.keys, k)
			}
		}
		for t, v := range m.tokens {
			if v.ClientID == clientID {
				delete(m.tokens, t)
			}
		}
	}
}
