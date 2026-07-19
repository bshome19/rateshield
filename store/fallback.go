package store

import (
	"context"
	"time"
)

// FallbackStore wraps a primary store (e.g. Redis) and a secondary store (e.g. In-Memory).
// If the primary store encounters an error (e.g. connection timeout/down), it automatically
// falls back to the secondary store.
type FallbackStore struct {
	primary        Store
	secondary      Store
	onPrimaryError func(err error)
}

// FallbackConfig holds configuration for FallbackStore.
type FallbackConfig struct {
	Primary        Store
	Secondary      Store
	OnPrimaryError func(err error)
}

// NewFallback creates a new FallbackStore.
func NewFallback(cfg FallbackConfig) *FallbackStore {
	sec := cfg.Secondary
	if sec == nil {
		sec = NewMemory()
	}
	return &FallbackStore{
		primary:        cfg.Primary,
		secondary:      sec,
		onPrimaryError: cfg.OnPrimaryError,
	}
}

func (fs *FallbackStore) handlePrimaryError(err error) {
	if err != nil && fs.onPrimaryError != nil {
		fs.onPrimaryError(err)
	}
}

func (fs *FallbackStore) Get(ctx context.Context, key string) (*State, error) {
	if fs.primary != nil {
		state, err := fs.primary.Get(ctx, key)
		if err == nil {
			return state, nil
		}
		fs.handlePrimaryError(err)
	}
	return fs.secondary.Get(ctx, key)
}

func (fs *FallbackStore) Set(ctx context.Context, key string, state *State, ttl time.Duration) error {
	if fs.primary != nil {
		err := fs.primary.Set(ctx, key, state, ttl)
		if err == nil {
			return nil
		}
		fs.handlePrimaryError(err)
	}
	return fs.secondary.Set(ctx, key, state, ttl)
}

func (fs *FallbackStore) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if fs.primary != nil {
		val, err := fs.primary.Increment(ctx, key, ttl)
		if err == nil {
			return val, nil
		}
		fs.handlePrimaryError(err)
	}
	return fs.secondary.Increment(ctx, key, ttl)
}

func (fs *FallbackStore) Reset(ctx context.Context, key string) error {
	if fs.primary != nil {
		err := fs.primary.Reset(ctx, key)
		if err == nil {
			return nil
		}
		fs.handlePrimaryError(err)
	}
	return fs.secondary.Reset(ctx, key)
}

func (fs *FallbackStore) AllowTokenBucket(ctx context.Context, key string, rate float64, capacity int64, n int64, ttl time.Duration) (*EvalResult, error) {
	if fs.primary != nil {
		res, err := fs.primary.AllowTokenBucket(ctx, key, rate, capacity, n, ttl)
		if err == nil {
			return res, nil
		}
		fs.handlePrimaryError(err)
	}
	return fs.secondary.AllowTokenBucket(ctx, key, rate, capacity, n, ttl)
}

func (fs *FallbackStore) AllowFixedWindow(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (*EvalResult, error) {
	if fs.primary != nil {
		res, err := fs.primary.AllowFixedWindow(ctx, key, limit, window, n, ttl)
		if err == nil {
			return res, nil
		}
		fs.handlePrimaryError(err)
	}
	return fs.secondary.AllowFixedWindow(ctx, key, limit, window, n, ttl)
}

func (fs *FallbackStore) AllowSlidingWindowCounter(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (*EvalResult, error) {
	if fs.primary != nil {
		res, err := fs.primary.AllowSlidingWindowCounter(ctx, key, limit, window, n, ttl)
		if err == nil {
			return res, nil
		}
		fs.handlePrimaryError(err)
	}
	return fs.secondary.AllowSlidingWindowCounter(ctx, key, limit, window, n, ttl)
}

func (fs *FallbackStore) AllowSlidingWindowLog(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (*EvalResult, error) {
	if fs.primary != nil {
		res, err := fs.primary.AllowSlidingWindowLog(ctx, key, limit, window, n, ttl)
		if err == nil {
			return res, nil
		}
		fs.handlePrimaryError(err)
	}
	return fs.secondary.AllowSlidingWindowLog(ctx, key, limit, window, n, ttl)
}
