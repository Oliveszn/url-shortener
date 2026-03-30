package limiter

import (
	"net/http"
	"url-shortener/internals/utils"
)

// middleware returns a handler that enforces the token bucket limit defined by config
// if the request has a valid jwt, rate limit key is user id, if req is anonymous we use client ip
func Middleware(l *Limiter, anonCfg, authedCfg Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var key string
			var cfg Config

			userID, err := utils.GetUserIDFromContext(r.Context())
			if err == nil {
				key = UserKey(userID)
				cfg = authedCfg
			} else {
				key = IPKey(r)
				cfg = anonCfg
			}

			result, err := l.Allow(r.Context(), key, cfg)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			SetHeaders(w, result)

			if !result.Allowed {
				Deny(w, result)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Strictiponly always keys in on the ip regardless of aut status and its used for authenticated endpoints where we want to limit by ip even if user sends token
func StrictIPOnly(l *Limiter, cfg Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := IPKey(r)
			result, err := l.Allow(r.Context(), key, cfg)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			SetHeaders(w, result)
			if !result.Allowed {
				Deny(w, result)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
