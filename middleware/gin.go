package middleware

import (
	"net/http"

	"github.com/bshome19/rateshield"
	"github.com/gin-gonic/gin"
)

// GinConfig holds the configuration for Gin rate limiting middleware.
type GinConfig struct {
	// Limiter is the rate limiter instance.
	Limiter rateshield.Limiter

	// KeyExtractor extracts the rate limit key from the request.
	// Defaults to KeyByIP if not provided.
	KeyExtractor rateshield.KeyExtractor

	// ErrorHandler is called when rate limit is exceeded.
	ErrorHandler func(c *gin.Context, result *rateshield.Result)

	// Skip is a function to determine if rate limiting should be skipped.
	Skip func(c *gin.Context) bool
}

// Gin returns a Gin middleware for rate limiting.
func Gin(cfg GinConfig) gin.HandlerFunc {
	// Set defaults
	if cfg.KeyExtractor == nil {
		cfg.KeyExtractor = rateshield.KeyByIP
	}
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = defaultGinErrorHandler
	}

	return func(c *gin.Context) {
		// Check if we should skip rate limiting
		if cfg.Skip != nil && cfg.Skip(c) {
			c.Next()
			return
		}

		// Extract the rate limit key
		key := cfg.KeyExtractor(c.Request)
		if key == "" {
			c.Next()
			return
		}

		// Check rate limit
		result, err := cfg.Limiter.Allow(c.Request.Context(), key)
		if err != nil {
			// On error, allow the request (fail open)
			c.Next()
			return
		}

		// Set rate limit headers (legacy X-RateLimit-* + modern IETF RateLimit-*)
		setRateLimitHeaders(c.Writer, result)

		// Check if rate limit exceeded
		if !result.Allowed {
			cfg.ErrorHandler(c, result)
			c.Abort()
			return
		}

		c.Next()
	}
}

// defaultGinErrorHandler is the default error handler for Gin.
func defaultGinErrorHandler(c *gin.Context, result *rateshield.Result) {
	c.JSON(http.StatusTooManyRequests, gin.H{
		"error":       "rate limit exceeded",
		"retry_after": result.RetryAfter.Seconds(),
	})
}

// GinSimple is a simplified Gin middleware that uses sensible defaults.
func GinSimple(limiter rateshield.Limiter) gin.HandlerFunc {
	return Gin(GinConfig{
		Limiter: limiter,
	})
}
