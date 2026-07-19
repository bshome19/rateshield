package algorithms

import (
	"context"
	"time"

	"github.com/bshome19/rateshield"
	"github.com/bshome19/rateshield/store"
)

// SlidingWindowCounter implements the sliding window counter rate limiting algorithm (O(1) memory complexity).
type SlidingWindowCounter struct {
	limit  int64         // Maximum requests per window
	window time.Duration // Window duration
	store  store.Store   // Storage backend
	ttl    time.Duration
}

// SlidingWindowCounterConfig holds the configuration for a SlidingWindowCounter limiter.
type SlidingWindowCounterConfig struct {
	// Limit is the maximum number of requests allowed per window.
	Limit int64

	// Window is the duration of the sliding window.
	Window time.Duration

	// Store is the storage backend.
	Store store.Store

	// TTL is how long to keep state in storage (default: 2x window).
	TTL time.Duration
}

// NewSlidingWindowCounter creates a new SlidingWindowCounter rate limiter.
func NewSlidingWindowCounter(cfg SlidingWindowCounterConfig) (*SlidingWindowCounter, error) {
	if cfg.Limit <= 0 {
		return nil, rateshield.ErrInvalidConfig
	}
	if cfg.Window <= 0 {
		return nil, rateshield.ErrInvalidConfig
	}
	if cfg.Store == nil {
		return nil, rateshield.ErrInvalidConfig
	}

	ttl := cfg.TTL
	if ttl == 0 {
		ttl = cfg.Window * 2
	}

	return &SlidingWindowCounter{
		limit:  cfg.Limit,
		window: cfg.Window,
		store:  cfg.Store,
		ttl:    ttl,
	}, nil
}

// Allow checks if a single request is allowed.
func (swc *SlidingWindowCounter) Allow(ctx context.Context, key string) (*rateshield.Result, error) {
	return swc.AllowN(ctx, key, 1)
}

// AllowN checks if n requests are allowed atomically.
func (swc *SlidingWindowCounter) AllowN(ctx context.Context, key string, n int64) (*rateshield.Result, error) {
	if key == "" {
		return nil, rateshield.ErrInvalidKey
	}

	evalRes, err := swc.store.AllowSlidingWindowCounter(ctx, key, swc.limit, swc.window, n, swc.ttl)
	if err != nil {
		return nil, err
	}

	return &rateshield.Result{
		Allowed:    evalRes.Allowed,
		Limit:      swc.limit,
		Remaining:  evalRes.Remaining,
		ResetAt:    evalRes.ResetAt,
		RetryAfter: evalRes.RetryAfter,
	}, nil
}

// Reset resets the rate limit for a key.
func (swc *SlidingWindowCounter) Reset(ctx context.Context, key string) error {
	if key == "" {
		return rateshield.ErrInvalidKey
	}
	return swc.store.Reset(ctx, key)
}
