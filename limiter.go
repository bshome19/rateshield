// Package rateshield provides a fast, flexible rate limiting library for Go.
package rateshield

import (
	"context"
	"time"
)

// FailStrategy defines the behavior when storage or internal errors occur.
type FailStrategy int

const (
	// FailOpen allows requests when store errors occur (prevents service outage). Default.
	FailOpen FailStrategy = iota

	// FailClosed blocks requests when store errors occur (enforces strict rate limits).
	FailClosed
)

// MetricsCollector defines an interface for observing rate limiting events.
type MetricsCollector interface {
	OnAllowed(key string)
	OnBlocked(key string)
	OnError(key string, err error)
}

// Result represents the outcome of a rate limit check.
type Result struct {
	// Allowed indicates whether the request is permitted.
	Allowed bool

	// Limit is the maximum number of requests allowed in the window.
	Limit int64

	// Remaining is the number of requests remaining in the current window.
	Remaining int64

	// ResetAt is the time when the rate limit window resets.
	ResetAt time.Time

	// RetryAfter indicates how long to wait before retrying (when not allowed).
	RetryAfter time.Duration
}

// Limiter is the main interface for rate limiting.
type Limiter interface {
	// Allow checks if a single request identified by key is allowed.
	Allow(ctx context.Context, key string) (*Result, error)

	// AllowN checks if n requests identified by key are allowed.
	AllowN(ctx context.Context, key string, n int64) (*Result, error)

	// Reset resets the rate limit for a given key.
	Reset(ctx context.Context, key string) error
}
