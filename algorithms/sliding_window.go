package algorithms

import (
	"context"
	"time"

	"github.com/bshome19/rateshield"
	"github.com/bshome19/rateshield/store"
)

// SlidingWindow implements the sliding window log rate limiting algorithm.
type SlidingWindow struct {
	limit  int64         // Maximum requests per window
	window time.Duration // Window duration
	store  store.Store   // Storage backend
	ttl    time.Duration
}

// SlidingWindowConfig holds the configuration for a SlidingWindow limiter.
type SlidingWindowConfig struct {
	// Limit is the maximum number of requests allowed per window.
	Limit int64

	// Window is the duration of the sliding window.
	Window time.Duration

	// Store is the storage backend.
	Store store.Store

	// TTL is how long to keep state in storage (default: 2x window).
	TTL time.Duration
}

// NewSlidingWindow creates a new SlidingWindow rate limiter.
func NewSlidingWindow(cfg SlidingWindowConfig) (*SlidingWindow, error) {
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

	return &SlidingWindow{
		limit:  cfg.Limit,
		window: cfg.Window,
		store:  cfg.Store,
		ttl:    ttl,
	}, nil
}

// Allow checks if a single request is allowed.
func (sw *SlidingWindow) Allow(ctx context.Context, key string) (*rateshield.Result, error) {
	return sw.AllowN(ctx, key, 1)
}

// AllowN checks if n requests are allowed atomically.
func (sw *SlidingWindow) AllowN(ctx context.Context, key string, n int64) (*rateshield.Result, error) {
	if key == "" {
		return nil, rateshield.ErrInvalidKey
	}

	evalRes, err := sw.store.AllowSlidingWindowLog(ctx, key, sw.limit, sw.window, n, sw.ttl)
	if err != nil {
		return nil, err
	}

	return &rateshield.Result{
		Allowed:    evalRes.Allowed,
		Limit:      sw.limit,
		Remaining:  evalRes.Remaining,
		ResetAt:    evalRes.ResetAt,
		RetryAfter: evalRes.RetryAfter,
	}, nil
}

// Reset resets the rate limit for a key.
func (sw *SlidingWindow) Reset(ctx context.Context, key string) error {
	if key == "" {
		return rateshield.ErrInvalidKey
	}
	return sw.store.Reset(ctx, key)
}
