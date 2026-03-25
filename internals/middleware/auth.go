package middleware

import (
	"context"
	"net/http"
	"strings"
	"url-shortener/internals/utils"
)

type contextKey string

const claimsKey contextKey = "auth_claims"

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := extractBearerToken(r)
		if !ok {
			http.Error(w, "Missing or malformed Authorization header", http.StatusUnauthorized)
			return
		}

		claims, err := utils.ValidateToken(token)
		if err != nil {

			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := utils.SetUserIDInContext(r.Context(), claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})

}

// Similar to require auth but doesn not block request if no token available
func OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token, ok := extractBearerToken(r); ok {
			if claims, err := utils.ValidateToken(token); err == nil {
				ctx := context.WithValue(r.Context(), claimsKey, claims)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})

}

func extractBearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", false
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}
