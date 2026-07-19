package middleware

import (
	"net/http"
	"strconv"

	"github.com/bshome19/rateshield"
)

// ChiConfig holds the configuration for Chi rate limiting middleware.
type ChiConfig struct {
	// Limiter is the rate limiter instance.
	Limiter rateshield.Limiter

	// KeyExtractor extracts the rate limit key from the request.
	// Defaults to KeyByIP if not provided.
	KeyExtractor rateshield.KeyExtractor

	// ErrorHandler is called when rate limit is exceeded.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, result *rateshield.Result)

	// Skip is a function to determine if rate limiting should be skipped.
	Skip func(r *http.Request) bool
}

// Chi returns a Chi-compatible middleware for rate limiting.
// Chi uses the standard http.Handler interface, so this is similar to Stdlib
// but provides Chi-specific conveniences.
func Chi(cfg ChiConfig) func(http.Handler) http.Handler {
	// Set defaults
	if cfg.KeyExtractor == nil {
		cfg.KeyExtractor = rateshield.KeyByIP
	}
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = defaultChiErrorHandler
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if we should skip rate limiting
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract the rate limit key
			key := cfg.KeyExtractor(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check rate limit
			result, err := cfg.Limiter.Allow(r.Context(), key)
			if err != nil {
				// On error, allow the request (fail open)
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

			// Check if rate limit exceeded
			if !result.Allowed {
				w.Header().Set("Retry-After", strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
				cfg.ErrorHandler(w, r, result)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// defaultChiErrorHandler is the default error handler for Chi.
func defaultChiErrorHandler(w http.ResponseWriter, r *http.Request, result *rateshield.Result) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte(`{"error":"rate limit exceeded","retry_after":` +
		strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10) + `}`))
}

// ChiSimple is a simplified Chi middleware that uses sensible defaults.
func ChiSimple(limiter rateshield.Limiter) func(http.Handler) http.Handler {
	return Chi(ChiConfig{
		Limiter: limiter,
	})
}

// ChiKeyByURLParam returns a key extractor that uses a URL parameter.
// Note: This requires chi's URLParam function to be used within the handler.
func ChiKeyByURLParam(param string) rateshield.KeyExtractor {
	return func(r *http.Request) string {
		// Chi stores URL params in the request context
		// This is a simplified version - in real usage you'd use chi.URLParam
		return r.URL.Query().Get(param)
	}
}

// ChiRateLimit is an alternative constructor that follows Chi's middleware conventions.
func ChiRateLimit(limiter rateshield.Limiter, opts ...ChiOption) func(http.Handler) http.Handler {
	cfg := ChiConfig{
		Limiter: limiter,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return Chi(cfg)
}

// ChiOption is a function that configures ChiConfig.
type ChiOption func(*ChiConfig)

// WithChiKeyExtractor sets a custom key extractor.
func WithChiKeyExtractor(ke rateshield.KeyExtractor) ChiOption {
	return func(cfg *ChiConfig) {
		cfg.KeyExtractor = ke
	}
}

// WithChiErrorHandler sets a custom error handler.
func WithChiErrorHandler(eh func(w http.ResponseWriter, r *http.Request, result *rateshield.Result)) ChiOption {
	return func(cfg *ChiConfig) {
		cfg.ErrorHandler = eh
	}
}

// WithChiSkip sets a skip function.
func WithChiSkip(skip func(r *http.Request) bool) ChiOption {
	return func(cfg *ChiConfig) {
		cfg.Skip = skip
	}
}
