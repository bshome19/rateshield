// Package store provides storage backends for rateshield rate limiters.
package store

import (
	"context"
	"time"
)

// State represents the current state of a rate limit bucket.
type State struct {
	// Tokens is the current number of tokens (for token bucket).
	Tokens float64

	// LastUpdate is the last time the state was updated.
	LastUpdate time.Time

	// Count is the request count (for fixed/sliding window).
	Count int64

	// WindowStart is the start of the current window.
	WindowStart time.Time

	// Requests stores timestamps for sliding window log algorithm.
	Requests []time.Time
}

// EvalResult represents the outcome of an atomic rate limit store evaluation.
type EvalResult struct {
	Allowed    bool
	Remaining  int64
	ResetAt    time.Time
	RetryAfter time.Duration
}

// Store is the interface for rate limit state storage.
type Store interface {
	// Get retrieves the current state for a key.
	Get(ctx context.Context, key string) (*State, error)

	// Set saves the state for a key.
	Set(ctx context.Context, key string, state *State, ttl time.Duration) error

	// Increment atomically increments the count for a key.
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)

	// Reset removes the state for a key.
	Reset(ctx context.Context, key string) error

	// AllowTokenBucket performs an atomic token bucket evaluation.
	AllowTokenBucket(ctx context.Context, key string, rate float64, capacity int64, n int64, ttl time.Duration) (*EvalResult, error)

	// AllowFixedWindow performs an atomic fixed window evaluation.
	AllowFixedWindow(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (*EvalResult, error)

	// AllowSlidingWindowCounter performs an atomic sliding window counter evaluation.
	AllowSlidingWindowCounter(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (*EvalResult, error)

	// AllowSlidingWindowLog performs an atomic sliding window log evaluation.
	AllowSlidingWindowLog(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (*EvalResult, error)
}