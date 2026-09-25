package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/Sports-Data-Africa/contracts/auth"
)

func TestAuthValidationAndL1Cache(t *testing.T) {
	mock := auth.NewMockAuthValidator()
	client := auth.NewCachedClient(mock, 60*time.Second)

	rawKey := "sda_live_secret_key_123"
	authCtx := &auth.ClientAuthContext{
		ClientID:      "betking_prod",
		ClientName:    "BetKing Africa",
		Plan:          auth.PlanFullPlatform,
		Environment:   auth.EnvProduction,
		Scopes:        []string{auth.ScopePrematchRead, auth.ScopeLiveStream, auth.ScopeBetPlacement},
		AllowedSports: []int32{1, 2, 3, 4}, // Football, IceHockey, Basketball, Tennis
		KeyID:         "KEY-001",
		AuthMethod:    "API_KEY",
	}
	mock.RegisterKey(rawKey, authCtx)

	ctx := context.Background()

	// 1. Valid API key authentication
	res, err := client.ValidateAPIKey(ctx, rawKey, auth.ScopePrematchRead, 1)
	if err != nil {
		t.Fatalf("Expected valid authentication, got error: %v", err)
	}
	if res.ClientID != "betking_prod" {
		t.Fatalf("Expected client betking_prod, got %s", res.ClientID)
	}

	// 2. L1 Cache Hit (Must retrieve from memory)
	resCached, err := client.ValidateAPIKey(ctx, rawKey, auth.ScopePrematchRead, 1)
	if err != nil || resCached.ClientID != "betking_prod" {
		t.Fatalf("L1 Cache retrieval failed")
	}

	// 3. Missing Scope Rejection
	_, err = client.ValidateAPIKey(ctx, rawKey, auth.ScopeKeyManage, 1)
	if err != auth.ErrEntitlementMissing {
		t.Fatalf("Expected ErrEntitlementMissing, got: %v", err)
	}

	// 4. Disallowed Sport Rejection (Sport 66 Cricket is not in allowed list [1, 2, 3, 4])
	_, err = client.ValidateAPIKey(ctx, rawKey, auth.ScopePrematchRead, 66)
	if err != auth.ErrSportNotAllowed {
		t.Fatalf("Expected ErrSportNotAllowed, got: %v", err)
	}

	// 5. Table Tennis (Sport 10) Policy Quarantine (Always denied regardless of client list)
	_, err = client.ValidateAPIKey(ctx, rawKey, auth.ScopePrematchRead, 10)
	if err != auth.ErrSportNotAllowed {
		t.Fatalf("Expected Table Tennis to be denied with ErrSportNotAllowed, got: %v", err)
	}
}

func TestL1CacheInvalidation(t *testing.T) {
	mock := auth.NewMockAuthValidator()
	client := auth.NewCachedClient(mock, 60*time.Second)

	rawKey := "sda_live_to_revoke"
	authCtx := &auth.ClientAuthContext{
		ClientID:    "alpha_client",
		ClientName:  "Alpha Sports",
		Plan:        auth.PlanFullPlatform,
		Environment: auth.EnvProduction,
		Scopes:      []string{auth.ScopePrematchRead},
		KeyID:       "KEY-ALPHA-1",
	}
	mock.RegisterKey(rawKey, authCtx)

	ctx := context.Background()

	// First query: populates L1 cache
	_, err := client.ValidateAPIKey(ctx, rawKey, auth.ScopePrematchRead, 1)
	if err != nil {
		t.Fatalf("First validation failed: %v", err)
	}

	// Simulate Key Revocation: auth-service removes key from authority and publishes invalidation
	mock.Invalidate("alpha_client", auth.HashKey(rawKey))
	client.HandleInvalidationEvent(&auth.ClientAuthInvalidatedEvent{
		ClientID:    "alpha_client",
		KeyHashHint: auth.HashKey(rawKey),
		Reason:      "KEY_REVOKED",
		EventID:     "evt-inv-1",
		OccurredAt:  time.Now(),
	})

	// Immediate next query: must be REJECTED because L1 was purged!
	_, err = client.ValidateAPIKey(ctx, rawKey, auth.ScopePrematchRead, 1)
	if err != auth.ErrInvalidCredentials {
		t.Fatalf("Expected ErrInvalidCredentials immediately after revocation event, got: %v", err)
	}
}

func TestWSTokenLifecycle(t *testing.T) {
	mock := auth.NewMockAuthValidator()
	client := auth.NewCachedClient(mock, 60*time.Second)

	rawKey := "sda_live_ws_key"
	authCtx := &auth.ClientAuthContext{
		ClientID:    "beta_client",
		ClientName:  "Beta Games",
		Plan:        auth.PlanFullPlatform,
		Environment: auth.EnvProduction,
		Scopes:      []string{auth.ScopeLiveStream},
		KeyID:       "KEY-WS-1",
	}
	mock.RegisterKey(rawKey, authCtx)

	ctx := context.Background()

	// 1. Issue WS Token
	token, exp, err := client.IssueWSToken(ctx, rawKey, 60)
	if err != nil {
		t.Fatalf("IssueWSToken failed: %v", err)
	}
	if token == "" || exp <= time.Now().Unix() {
		t.Fatalf("Invalid token or expiry generated: token=%s, exp=%d", token, exp)
	}

	// 2. Validate WS Token
	wsCtx, err := client.ValidateWSToken(ctx, token)
	if err != nil {
		t.Fatalf("ValidateWSToken failed: %v", err)
	}
	if wsCtx.ClientID != "beta_client" || wsCtx.AuthMethod != "WS_TOKEN" {
		t.Fatalf("Unexpected WS context: %+v", wsCtx)
	}

	// 3. Reject invalid/tampered token
	_, err = client.ValidateWSToken(ctx, "invalid_tampered_token")
	if err != auth.ErrInvalidToken {
		t.Fatalf("Expected ErrInvalidToken, got: %v", err)
	}
}
