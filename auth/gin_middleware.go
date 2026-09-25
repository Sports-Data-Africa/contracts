package auth

import (
	"context"
	"net/http"
	"strings"
)

// HTTPAuthMiddleware is a framework-agnostic standard library http.Handler middleware wrapper.
func HTTPAuthMiddleware(validator AuthValidator, requiredScope string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		var rawSecret string
		if strings.HasPrefix(authHeader, "Bearer ") {
			rawSecret = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			rawSecret = r.Header.Get("X-API-Key")
		}

		if rawSecret == "" {
			http.Error(w, `{"success":false,"code":"MISSING_API_KEY","message":"Authorization header Bearer <key> or X-API-Key required."}`, http.StatusUnauthorized)
			return
		}

		authCtx, err := validator.ValidateAPIKey(r.Context(), rawSecret, requiredScope, 0)
		if err != nil {
			http.Error(w, `{"success":false,"code":"INVALID_API_KEY","message":"`+err.Error()+`"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "auth_context", authCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
