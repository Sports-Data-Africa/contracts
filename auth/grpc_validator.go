package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	authv1 "github.com/Sports-Data-Africa/contracts/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCAuthValidator implements AuthValidator over internal gRPC to auth-service.
type GRPCAuthValidator struct {
	client authv1.AuthServiceClient
	conn   *grpc.ClientConn
}

// NewGRPCAuthValidator creates a new gRPC-backed AuthValidator
func NewGRPCAuthValidator(conn *grpc.ClientConn) *GRPCAuthValidator {
	return &GRPCAuthValidator{
		client: authv1.NewAuthServiceClient(conn),
		conn:   conn,
	}
}

// ValidateAPIKey calls the authoritative auth-service ValidateAPIKey RPC
func (g *GRPCAuthValidator) ValidateAPIKey(ctx context.Context, rawKey string, requiredScope string, sportID int32) (*ClientAuthContext, error) {
	if strings.TrimSpace(rawKey) == "" {
		return nil, ErrInvalidCredentials
	}

	req := &authv1.ValidateAPIKeyRequest{
		RawApiKey:     rawKey,
		RequiredScope: requiredScope,
		SportId:       sportID,
	}

	resp, err := g.client.ValidateAPIKey(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.Unauthenticated:
				return nil, ErrInvalidCredentials
			case codes.PermissionDenied:
				return nil, ErrEntitlementMissing
			case codes.FailedPrecondition:
				return nil, ErrAccountInactive
			}
		}
		return nil, fmt.Errorf("auth-service gRPC error: %w", err)
	}

	if !resp.Valid {
		switch resp.ErrorCode {
		case "ACCOUNT_INACTIVE", "CLIENT_SUSPENDED":
			return nil, ErrAccountInactive
		case "ENTITLEMENT_MISSING":
			return nil, ErrEntitlementMissing
		case "SPORT_NOT_ALLOWED":
			return nil, ErrSportNotAllowed
		default:
			if resp.ErrorMessage != "" {
				return nil, errors.New(resp.ErrorMessage)
			}
			return nil, ErrInvalidCredentials
		}
	}

	// Double-check local scope and sport rules on received contract
	authCtx := &ClientAuthContext{
		ClientID:      resp.ClientId,
		ClientName:    resp.ClientName,
		Plan:          resp.Plan,
		Environment:   resp.Environment,
		Scopes:        resp.Scopes,
		AllowedSports: resp.AllowedSports,
		KeyID:         resp.KeyId,
		AuthMethod:    "API_KEY",
		ExpiresAt:     time.Unix(resp.ExpiresAtUnix, 0),
	}

	if requiredScope != "" && !authCtx.HasScope(requiredScope) {
		return nil, ErrEntitlementMissing
	}
	if sportID > 0 && !authCtx.IsSportAllowed(sportID) {
		return nil, ErrSportNotAllowed
	}

	return authCtx, nil
}

// ValidateWSToken calls the authoritative auth-service ValidateWSToken RPC
func (g *GRPCAuthValidator) ValidateWSToken(ctx context.Context, token string) (*ClientAuthContext, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrInvalidToken
	}

	req := &authv1.ValidateWSTokenRequest{
		Token: token,
	}

	resp, err := g.client.ValidateWSToken(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Unauthenticated {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("auth-service gRPC error: %w", err)
	}

	if !resp.Valid {
		if resp.ErrorCode == "TOKEN_EXPIRED" {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	return &ClientAuthContext{
		ClientID:      resp.ClientId,
		ClientName:    resp.ClientName,
		Plan:          resp.Plan,
		Environment:   resp.Environment,
		Scopes:        resp.Scopes,
		AllowedSports: resp.AllowedSports,
		KeyID:         resp.KeyId,
		AuthMethod:    "WS_TOKEN",
		ExpiresAt:     time.Unix(resp.ExpiresAtUnix, 0),
	}, nil
}

// IssueWSToken validates the raw key and generates a token context
func (g *GRPCAuthValidator) IssueWSToken(ctx context.Context, rawKey string, ttlSeconds int32) (string, int64, error) {
	authCtx, err := g.ValidateAPIKey(ctx, rawKey, ScopeLiveStream, 0)
	if err != nil {
		return "", 0, err
	}
	_ = authCtx

	if ttlSeconds <= 0 || ttlSeconds > 300 {
		ttlSeconds = 60
	}
	exp := time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	token := fmt.Sprintf("ws_%s_%d", HashKey(rawKey)[:16], exp.Unix())
	return token, exp.Unix(), nil
}

// Invalidate handles cluster or local invalidation
func (g *GRPCAuthValidator) Invalidate(clientID string, keyHash string) {
	// Remote state is authoritative in auth-service; local cache handled by CachedClient
}
