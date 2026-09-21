package rateshield

import (
	"context"
	"time"

	"github.com/bshome19/rateshield/store"
)

// ==========================================
// Token Bucket Limiter
// ==========================================

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

// TokenBucket implements the token bucket rate limiting algorithm.
type TokenBucket struct {
	rate     float64
	capacity int64
	store    store.Store
	ttl      time.Duration
}

// NewTokenBucket creates a new TokenBucket rate limiter.
func NewTokenBucket(cfg TokenBucketConfig) (*TokenBucket, error) {
	if cfg.Rate <= 0 || cfg.Capacity <= 0 || cfg.Store == nil {
		return nil, ErrInvalidConfig
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
func (tb *TokenBucket) Allow(ctx context.Context, key string) (*Result, error) {
	return tb.AllowN(ctx, key, 1)
}

// AllowN checks if n requests are allowed atomically.
func (tb *TokenBucket) AllowN(ctx context.Context, key string, n int64) (*Result, error) {
	res, err := tb.AllowNFast(ctx, key, n)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// AllowFast checks if a single request is allowed without heap allocations.
func (tb *TokenBucket) AllowFast(ctx context.Context, key string) (Result, error) {
	return tb.AllowNFast(ctx, key, 1)
}

// AllowNFast checks if n requests are allowed without heap allocations.
func (tb *TokenBucket) AllowNFast(ctx context.Context, key string, n int64) (Result, error) {
	if key == "" {
		return Result{}, ErrInvalidKey
	}

	evalRes, err := tb.store.AllowTokenBucket(ctx, key, tb.rate, tb.capacity, n, tb.ttl)
	if err != nil {
		return Result{}, err
	}

	return Result{
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
		return ErrInvalidKey
	}
	return tb.store.Reset(ctx, key)
}

// Close releases any resources associated with the limiter store.
func (tb *TokenBucket) Close() error {
	return tb.store.Close()
}

// ==========================================
// Fixed Window Limiter
// ==========================================

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

// FixedWindow implements the fixed window counter rate limiting algorithm.
type FixedWindow struct {
	limit  int64
	window time.Duration
	store  store.Store
	ttl    time.Duration
}

// NewFixedWindow creates a new FixedWindow rate limiter.
func NewFixedWindow(cfg FixedWindowConfig) (*FixedWindow, error) {
	if cfg.Limit <= 0 || cfg.Window <= 0 || cfg.Store == nil {
		return nil, ErrInvalidConfig
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
func (fw *FixedWindow) Allow(ctx context.Context, key string) (*Result, error) {
	return fw.AllowN(ctx, key, 1)
}

// AllowN checks if n requests are allowed atomically.
func (fw *FixedWindow) AllowN(ctx context.Context, key string, n int64) (*Result, error) {
	res, err := fw.AllowNFast(ctx, key, n)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// AllowFast checks if a single request is allowed without heap allocations.
func (fw *FixedWindow) AllowFast(ctx context.Context, key string) (Result, error) {
	return fw.AllowNFast(ctx, key, 1)
}

// AllowNFast checks if n requests are allowed without heap allocations.
func (fw *FixedWindow) AllowNFast(ctx context.Context, key string, n int64) (Result, error) {
	if key == "" {
		return Result{}, ErrInvalidKey
	}

	evalRes, err := fw.store.AllowFixedWindow(ctx, key, fw.limit, fw.window, n, fw.ttl)
	if err != nil {
		return Result{}, err
	}

	return Result{
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
		return ErrInvalidKey
	}
	return fw.store.Reset(ctx, key)
}

// Close releases any resources associated with the limiter store.
func (fw *FixedWindow) Close() error {
	return fw.store.Close()
}

// ==========================================
// Sliding Window Counter Limiter
// ==========================================

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

// SlidingWindowCounter implements the sliding window counter rate limiting algorithm.
type SlidingWindowCounter struct {
	limit  int64
	window time.Duration
	store  store.Store
	ttl    time.Duration
}

// NewSlidingWindowCounter creates a new SlidingWindowCounter rate limiter.
func NewSlidingWindowCounter(cfg SlidingWindowCounterConfig) (*SlidingWindowCounter, error) {
	if cfg.Limit <= 0 || cfg.Window <= 0 || cfg.Store == nil {
		return nil, ErrInvalidConfig
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
func (swc *SlidingWindowCounter) Allow(ctx context.Context, key string) (*Result, error) {
	return swc.AllowN(ctx, key, 1)
}

// AllowN checks if n requests are allowed atomically.
func (swc *SlidingWindowCounter) AllowN(ctx context.Context, key string, n int64) (*Result, error) {
	res, err := swc.AllowNFast(ctx, key, n)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// AllowFast checks if a single request is allowed without heap allocations.
func (swc *SlidingWindowCounter) AllowFast(ctx context.Context, key string) (Result, error) {
	return swc.AllowNFast(ctx, key, 1)
}

// AllowNFast checks if n requests are allowed without heap allocations.
func (swc *SlidingWindowCounter) AllowNFast(ctx context.Context, key string, n int64) (Result, error) {
	if key == "" {
		return Result{}, ErrInvalidKey
	}

	evalRes, err := swc.store.AllowSlidingWindowCounter(ctx, key, swc.limit, swc.window, n, swc.ttl)
	if err != nil {
		return Result{}, err
	}

	return Result{
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
		return ErrInvalidKey
	}
	return swc.store.Reset(ctx, key)
}

// Close releases any resources associated with the limiter store.
func (swc *SlidingWindowCounter) Close() error {
	return swc.store.Close()
}

// ==========================================
// Sliding Window Log Limiter
// ==========================================

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

// SlidingWindow implements the sliding window log rate limiting algorithm.
type SlidingWindow struct {
	limit  int64
	window time.Duration
	store  store.Store
	ttl    time.Duration
}

// NewSlidingWindow creates a new SlidingWindow rate limiter.
func NewSlidingWindow(cfg SlidingWindowConfig) (*SlidingWindow, error) {
	if cfg.Limit <= 0 || cfg.Window <= 0 || cfg.Store == nil {
		return nil, ErrInvalidConfig
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
func (sw *SlidingWindow) Allow(ctx context.Context, key string) (*Result, error) {
	return sw.AllowN(ctx, key, 1)
}

// AllowN checks if n requests are allowed atomically.
func (sw *SlidingWindow) AllowN(ctx context.Context, key string, n int64) (*Result, error) {
	res, err := sw.AllowNFast(ctx, key, n)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// AllowFast checks if a single request is allowed without heap allocations.
func (sw *SlidingWindow) AllowFast(ctx context.Context, key string) (Result, error) {
	return sw.AllowNFast(ctx, key, 1)
}

// AllowNFast checks if n requests are allowed without heap allocations.
func (sw *SlidingWindow) AllowNFast(ctx context.Context, key string, n int64) (Result, error) {
	if key == "" {
		return Result{}, ErrInvalidKey
	}

	evalRes, err := sw.store.AllowSlidingWindowLog(ctx, key, sw.limit, sw.window, n, sw.ttl)
	if err != nil {
		return Result{}, err
	}

	return Result{
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
		return ErrInvalidKey
	}
	return sw.store.Reset(ctx, key)
}

// Close releases any resources associated with the limiter store.
func (sw *SlidingWindow) Close() error {
	return sw.store.Close()
}
