package middleware

import (
	"net/http"

	"github.com/bshome19/rateshield"
)

// Stdlib returns a middleware for the standard net/http library.
func Stdlib(cfg Config) func(http.Handler) http.Handler {
	if cfg.KeyExtractor == nil {
		cfg.KeyExtractor = rateshield.KeyByIP
	}
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = defaultErrorHandler
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			key := cfg.KeyExtractor(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			result, err := cfg.Limiter.Allow(r.Context(), key)
			if err != nil {
				if cfg.FailStrategy == rateshield.FailClosed {
					http.Error(w, "rate limit store error", http.StatusInternalServerError)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			setRateLimitHeaders(w, result)

			if !result.Allowed {
				cfg.ErrorHandler(w, r, result)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// StdlibSimple is a simplified middleware that uses sensible defaults.
func StdlibSimple(limiter rateshield.Limiter) func(http.Handler) http.Handler {
	return Stdlib(Config{
		Limiter: limiter,
	})
}