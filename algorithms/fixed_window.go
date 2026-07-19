package algorithms

import (
	"context"
	"time"

	"github.com/bshome19/rateshield"
	"github.com/bshome19/rateshield/store"
)

// FixedWindow implements the fixed window counter rate limiting algorithm.
type FixedWindow struct {
	limit  int64         // Maximum requests per window
	window time.Duration // Window duration
	store  store.Store   // Storage backend
	ttl    time.Duration
}

// FixedWindowConfig holds the configuration for a FixedWindow limiter.
type FixedWindowConfig struct {
	// Limit is the maximum number of requests allowed per window.
	Limit int64

	// Window is the duration of the fixed window.
	Window time.Duration

	// Store is the storage backend.
	Store store.Store

	// TTL is how long to keep state in storage (default: 2x window).
	TTL time.Duration
}

// NewFixedWindow creates a new FixedWindow rate limiter.
func NewFixedWindow(cfg FixedWindowConfig) (*FixedWindow, error) {
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

	return &FixedWindow{
		limit:  cfg.Limit,
		window: cfg.Window,
		store:  cfg.Store,
		ttl:    ttl,
	}, nil
}

// Allow checks if a single request is allowed.
func (fw *FixedWindow) Allow(ctx context.Context, key string) (*rateshield.Result, error) {
	return fw.AllowN(ctx, key, 1)
}

// AllowN checks if n requests are allowed atomically.
func (fw *FixedWindow) AllowN(ctx context.Context, key string, n int64) (*rateshield.Result, error) {
	if key == "" {
		return nil, rateshield.ErrInvalidKey
	}

	evalRes, err := fw.store.AllowFixedWindow(ctx, key, fw.limit, fw.window, n, fw.ttl)
	if err != nil {
		return nil, err
	}

	return &rateshield.Result{
		Allowed:    evalRes.Allowed,
		Limit:      fw.limit,
		Remaining:  evalRes.Remaining,
		ResetAt:    evalRes.ResetAt,
		RetryAfter: evalRes.RetryAfter,
	}, nil
}

// Reset resets the rate limit for a key.
func (fw *FixedWindow) Reset(ctx context.Context, key string) error {
	if key == "" {
		return rateshield.ErrInvalidKey
	}
	return fw.store.Reset(ctx, key)
}