package middleware

import (
	"net/http"

	"github.com/bshome19/rateshield"
	"github.com/labstack/echo/v4"
)

// EchoConfig holds the configuration for Echo rate limiting middleware.
type EchoConfig struct {
	// Limiter is the rate limiter instance.
	Limiter rateshield.Limiter

	// KeyExtractor extracts the rate limit key from the request.
	// Defaults to using client IP if not provided.
	KeyExtractor func(c echo.Context) string

	// ErrorHandler is called when rate limit is exceeded.
	ErrorHandler func(c echo.Context, result *rateshield.Result) error

	// Skip is a function to determine if rate limiting should be skipped.
	Skip func(c echo.Context) bool

	// Skipper is an alias for Skip (Echo convention).
	Skipper func(c echo.Context) bool
}

// Echo returns an Echo middleware for rate limiting.
func Echo(cfg EchoConfig) echo.MiddlewareFunc {
	// Set defaults
	if cfg.KeyExtractor == nil {
		cfg.KeyExtractor = func(c echo.Context) string {
			return c.RealIP()
		}
	}
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = defaultEchoErrorHandler
	}
	// Support both Skip and Skipper (Echo convention)
	skipper := cfg.Skip
	if skipper == nil {
		skipper = cfg.Skipper
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if we should skip rate limiting
			if skipper != nil && skipper(c) {
				return next(c)
			}

			// Extract the rate limit key
			key := cfg.KeyExtractor(c)
			if key == "" {
				return next(c)
			}

			// Check rate limit
			result, err := cfg.Limiter.Allow(c.Request().Context(), key)
			if err != nil {
				// On error, allow the request (fail open)
				return next(c)
			}

			// Set rate limit headers (legacy X-RateLimit-* + modern IETF RateLimit-*)
			setRateLimitHeaders(c.Response().Writer, result)

			// Check if rate limit exceeded
			if !result.Allowed {
				return cfg.ErrorHandler(c, result)
			}

			return next(c)
		}
	}
}

// defaultEchoErrorHandler is the default error handler for Echo.
func defaultEchoErrorHandler(c echo.Context, result *rateshield.Result) error {
	return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
		"error":       "rate limit exceeded",
		"retry_after": result.RetryAfter.Seconds(),
	})
}

// EchoSimple is a simplified Echo middleware that uses sensible defaults.
func EchoSimple(limiter rateshield.Limiter) echo.MiddlewareFunc {
	return Echo(EchoConfig{
		Limiter: limiter,
	})
}

// EchoKeyByIP returns a key extractor that uses the client IP.
func EchoKeyByIP() func(c echo.Context) string {
	return func(c echo.Context) string {
		return c.RealIP()
	}
}

// EchoKeyByHeader returns a key extractor that uses a header value.
func EchoKeyByHeader(header string) func(c echo.Context) string {
	return func(c echo.Context) string {
		return c.Request().Header.Get(header)
	}
}

// EchoKeyByPath returns a key extractor that uses the request path.
func EchoKeyByPath() func(c echo.Context) string {
	return func(c echo.Context) string {
		return c.Request().Method + ":" + c.Path()
	}
}

// EchoKeyByIPAndPath returns a key extractor combining IP and path.
func EchoKeyByIPAndPath() func(c echo.Context) string {
	return func(c echo.Context) string {
		return c.RealIP() + ":" + c.Request().Method + ":" + c.Path()
	}
}

// EchoKeyByUserID returns a key extractor using user ID from header.
func EchoKeyByUserID(header string) func(c echo.Context) string {
	return func(c echo.Context) string {
		userID := c.Request().Header.Get(header)
		if userID == "" {
			return c.RealIP() // Fallback to IP
		}
		return "user:" + userID
	}
}

// EchoKeyByParam returns a key extractor using a URL parameter.
func EchoKeyByParam(param string) func(c echo.Context) string {
	return func(c echo.Context) string {
		return c.Param(param)
	}
}
