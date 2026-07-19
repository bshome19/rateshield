package middleware

import (
	"strconv"

	"github.com/bshome19/rateshield"
	"github.com/gofiber/fiber/v2"
)

// FiberConfig holds the configuration for Fiber rate limiting middleware.
type FiberConfig struct {
	// Limiter is the rate limiter instance.
	Limiter rateshield.Limiter

	// KeyExtractor extracts the rate limit key from the request.
	// Defaults to using client IP if not provided.
	KeyExtractor func(c *fiber.Ctx) string

	// ErrorHandler is called when rate limit is exceeded.
	ErrorHandler func(c *fiber.Ctx, result *rateshield.Result) error

	// Skip is a function to determine if rate limiting should be skipped.
	Skip func(c *fiber.Ctx) bool

	// SkipSuccessfulRequests skips counting successful requests.
	SkipSuccessfulRequests bool

	// SkipFailedRequests skips counting failed requests.
	SkipFailedRequests bool
}

// Fiber returns a Fiber middleware for rate limiting.
func Fiber(cfg FiberConfig) fiber.Handler {
	// Set defaults
	if cfg.KeyExtractor == nil {
		cfg.KeyExtractor = func(c *fiber.Ctx) string {
			return c.IP()
		}
	}
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = defaultFiberErrorHandler
	}

	return func(c *fiber.Ctx) error {
		// Check if we should skip rate limiting
		if cfg.Skip != nil && cfg.Skip(c) {
			return c.Next()
		}

		// Extract the rate limit key
		key := cfg.KeyExtractor(c)
		if key == "" {
			return c.Next()
		}

		// Check rate limit
		result, err := cfg.Limiter.Allow(c.Context(), key)
		if err != nil {
			// On error, allow the request (fail open)
			return c.Next()
		}

		// Set rate limit headers
		c.Set("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
		c.Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))

		// Check if rate limit exceeded
		if !result.Allowed {
			c.Set("Retry-After", strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
			return cfg.ErrorHandler(c, result)
		}

		return c.Next()
	}
}

// defaultFiberErrorHandler is the default error handler for Fiber.
func defaultFiberErrorHandler(c *fiber.Ctx, result *rateshield.Result) error {
	return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
		"error":       "rate limit exceeded",
		"retry_after": result.RetryAfter.Seconds(),
	})
}

// FiberSimple is a simplified Fiber middleware that uses sensible defaults.
func FiberSimple(limiter rateshield.Limiter) fiber.Handler {
	return Fiber(FiberConfig{
		Limiter: limiter,
	})
}

// FiberKeyByIP returns a key extractor that uses the client IP.
func FiberKeyByIP() func(c *fiber.Ctx) string {
	return func(c *fiber.Ctx) string {
		return c.IP()
	}
}

// FiberKeyByHeader returns a key extractor that uses a header value.
func FiberKeyByHeader(header string) func(c *fiber.Ctx) string {
	return func(c *fiber.Ctx) string {
		return c.Get(header)
	}
}

// FiberKeyByRoute returns a key extractor that uses the route path.
func FiberKeyByRoute() func(c *fiber.Ctx) string {
	return func(c *fiber.Ctx) string {
		return c.Method() + ":" + c.Path()
	}
}

// FiberKeyByIPAndRoute returns a key extractor combining IP and route.
func FiberKeyByIPAndRoute() func(c *fiber.Ctx) string {
	return func(c *fiber.Ctx) string {
		return c.IP() + ":" + c.Method() + ":" + c.Path()
	}
}

// FiberKeyByUserID returns a key extractor that uses a user ID from header.
func FiberKeyByUserID(header string) func(c *fiber.Ctx) string {
	return func(c *fiber.Ctx) string {
		userID := c.Get(header)
		if userID == "" {
			return c.IP() // Fallback to IP
		}
		return "user:" + userID
	}
}
