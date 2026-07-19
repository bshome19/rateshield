// Package middleware provides HTTP middleware for rate limiting.
package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/bshome19/rateshield"
)

// Config holds the configuration for rate limiting middleware.
type Config struct {
	// Limiter is the rate limiter instance.
	Limiter rateshield.Limiter

	// KeyExtractor extracts the rate limit key from the request.
	// Defaults to KeyByIP if not provided.
	KeyExtractor rateshield.KeyExtractor

	// ErrorHandler is called when rate limit is exceeded.
	// Defaults to a JSON error response if not provided.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, result *rateshield.Result)

	// FailStrategy determines behavior when limiter store fails (default: FailOpen).
	FailStrategy rateshield.FailStrategy

	// SkipSuccessfulRequests skips rate limiting for successful responses.
	SkipSuccessfulRequests bool

	// SkipFailedRequests skips rate limiting for failed responses.
	SkipFailedRequests bool

	// Skip is a function to determine if rate limiting should be skipped.
	Skip func(r *http.Request) bool
}

// setRateLimitHeaders sets standard and IETF rate limit headers.
func setRateLimitHeaders(w http.ResponseWriter, result *rateshield.Result) {
	limitStr := strconv.FormatInt(result.Limit, 10)
	remainingStr := strconv.FormatInt(result.Remaining, 10)
	resetUnix := strconv.FormatInt(result.ResetAt.Unix(), 10)

	// Standard legacy headers
	w.Header().Set("X-RateLimit-Limit", limitStr)
	w.Header().Set("X-RateLimit-Remaining", remainingStr)
	w.Header().Set("X-RateLimit-Reset", resetUnix)

	// Modern IETF Draft headers
	w.Header().Set("RateLimit-Limit", limitStr)
	w.Header().Set("RateLimit-Remaining", remainingStr)

	resetDelta := int64(time.Until(result.ResetAt).Seconds())
	if resetDelta < 0 {
		resetDelta = 0
	}
	w.Header().Set("RateLimit-Reset", strconv.FormatInt(resetDelta, 10))
}

// defaultErrorHandler is the default handler for rate limit exceeded errors.
func defaultErrorHandler(w http.ResponseWriter, r *http.Request, result *rateshield.Result) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(`{"error":"rate limit exceeded","retry_after":` +
		strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10) + `}`))
}
