// Package algorithms provides rate limiting algorithm implementations.
package algorithms

import (
	"context"
	"time"

	"github.com/bshome19/rateshield"
	"github.com/bshome19/rateshield/store"
)

// TokenBucket implements the token bucket rate limiting algorithm.
type TokenBucket struct {
	rate     float64     // Tokens added per second
	capacity int64       // Maximum tokens (burst size)
	store    store.Store // Storage backend
	ttl      time.Duration
}

// TokenBucketConfig holds the configuration for a TokenBucket limiter.
type TokenBucketConfig struct {
	// Rate is the number of tokens added per second.
	Rate float64

	// Capacity is the maximum number of tokens (burst size).
	Capacity int64

	// Store is the storage backend.
	Store store.Store

	// TTL is how long to keep state in storage (default: 1 hour).
	TTL time.Duration
}

// NewTokenBucket creates a new TokenBucket rate limiter.
func NewTokenBucket(cfg TokenBucketConfig) (*TokenBucket, error) {
	if cfg.Rate <= 0 {
		return nil, rateshield.ErrInvalidConfig
	}
	if cfg.Capacity <= 0 {
		return nil, rateshield.ErrInvalidConfig
	}
	if cfg.Store == nil {
		return nil, rateshield.ErrInvalidConfig
	}

	ttl := cfg.TTL
	if ttl == 0 {
		ttl = time.Hour
	}

	return &TokenBucket{
		rate:     cfg.Rate,
		capacity: cfg.Capacity,
		store:    cfg.Store,
		ttl:      ttl,
	}, nil
}

// Allow checks if a single request is allowed.
func (tb *TokenBucket) Allow(ctx context.Context, key string) (*rateshield.Result, error) {
	return tb.AllowN(ctx, key, 1)
}

// AllowN checks if n requests are allowed atomically.
func (tb *TokenBucket) AllowN(ctx context.Context, key string, n int64) (*rateshield.Result, error) {
	if key == "" {
		return nil, rateshield.ErrInvalidKey
	}

	evalRes, err := tb.store.AllowTokenBucket(ctx, key, tb.rate, tb.capacity, n, tb.ttl)
	if err != nil {
		return nil, err
	}

	return &rateshield.Result{
		Allowed:    evalRes.Allowed,
		Limit:      tb.capacity,
		Remaining:  evalRes.Remaining,
		ResetAt:    evalRes.ResetAt,
		RetryAfter: evalRes.RetryAfter,
	}, nil
}

// Reset resets the rate limit for a key.
func (tb *TokenBucket) Reset(ctx context.Context, key string) error {
	if key == "" {
		return rateshield.ErrInvalidKey
	}
	return tb.store.Reset(ctx, key)
}
