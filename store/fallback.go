package store

import (
	"context"
	"time"

	"github.com/bshome19/rateshield/internal/circuit"
)

// FallbackStore wraps a primary store (e.g. Redis) and a secondary store (e.g. In-Memory).
// If the primary store encounters errors or goes down, an internal atomic circuit breaker
// trips and immediately diverts traffic to the secondary store without blocking on timeouts.
type FallbackStore struct {
	primary        Store
	secondary      Store
	onPrimaryError func(err error)
	breaker        *circuit.Breaker
}

// FallbackConfig holds configuration for FallbackStore.
type FallbackConfig struct {
	Primary        Store
	Secondary      Store
	OnPrimaryError func(err error)

	// FailureThreshold is the consecutive error count to trip the circuit (default: 5).
	FailureThreshold int64

	// SuccessThreshold is the consecutive successes in half-open state to reset (default: 2).
	SuccessThreshold int64

	// Cooldown is the duration to remain open before probing primary store (default: 5s).
	Cooldown time.Duration
}

// NewFallback creates a new FallbackStore equipped with an atomic circuit breaker.
func NewFallback(cfg FallbackConfig) *FallbackStore {
	sec := cfg.Secondary
	if sec == nil {
		sec = NewMemory()
	}

	cb := circuit.New(circuit.Config{
		FailureThreshold: cfg.FailureThreshold,
		SuccessThreshold: cfg.SuccessThreshold,
		Cooldown:         cfg.Cooldown,
	})

	return &FallbackStore{
		primary:        cfg.Primary,
		secondary:      sec,
		onPrimaryError: cfg.OnPrimaryError,
		breaker:        cb,
	}
}

// Breaker returns the internal circuit breaker instance.
func (fs *FallbackStore) Breaker() *circuit.Breaker {
	return fs.breaker
}

func (fs *FallbackStore) handlePrimaryError(err error) {
	if err != nil && fs.onPrimaryError != nil {
		fs.onPrimaryError(err)
	}
}

func (fs *FallbackStore) Reset(ctx context.Context, key string) error {
	var firstErr error
	if fs.primary != nil {
		if err := fs.primary.Reset(ctx, key); err != nil {
			fs.handlePrimaryError(err)
			firstErr = err
		}
	}
	// Always reset secondary too — prevents stale counters surviving a Redis
	// outage+recovery cycle where the secondary accumulated state during the outage.
	if err := fs.secondary.Reset(ctx, key); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (fs *FallbackStore) AllowTokenBucket(ctx context.Context, key string, rate float64, capacity int64, n int64, ttl time.Duration) (EvalResult, error) {
	if fs.primary != nil && fs.breaker.Allow() {
		res, err := fs.primary.AllowTokenBucket(ctx, key, rate, capacity, n, ttl)
		if err == nil {
			fs.breaker.ReportSuccess()
			return res, nil
		}
		fs.breaker.ReportFailure()
		fs.handlePrimaryError(err)
	}
	return fs.secondary.AllowTokenBucket(ctx, key, rate, capacity, n, ttl)
}

func (fs *FallbackStore) AllowFixedWindow(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (EvalResult, error) {
	if fs.primary != nil && fs.breaker.Allow() {
		res, err := fs.primary.AllowFixedWindow(ctx, key, limit, window, n, ttl)
		if err == nil {
			fs.breaker.ReportSuccess()
			return res, nil
		}
		fs.breaker.ReportFailure()
		fs.handlePrimaryError(err)
	}
	return fs.secondary.AllowFixedWindow(ctx, key, limit, window, n, ttl)
}

func (fs *FallbackStore) AllowSlidingWindowCounter(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (EvalResult, error) {
	if fs.primary != nil && fs.breaker.Allow() {
		res, err := fs.primary.AllowSlidingWindowCounter(ctx, key, limit, window, n, ttl)
		if err == nil {
			fs.breaker.ReportSuccess()
			return res, nil
		}
		fs.breaker.ReportFailure()
		fs.handlePrimaryError(err)
	}
	return fs.secondary.AllowSlidingWindowCounter(ctx, key, limit, window, n, ttl)
}

func (fs *FallbackStore) AllowSlidingWindowLog(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (EvalResult, error) {
	if fs.primary != nil && fs.breaker.Allow() {
		res, err := fs.primary.AllowSlidingWindowLog(ctx, key, limit, window, n, ttl)
		if err == nil {
			fs.breaker.ReportSuccess()
			return res, nil
		}
		fs.breaker.ReportFailure()
		fs.handlePrimaryError(err)
	}
	return fs.secondary.AllowSlidingWindowLog(ctx, key, limit, window, n, ttl)
}

// Close closes both primary and secondary stores if applicable.
func (fs *FallbackStore) Close() error {
	var firstErr error
	if fs.primary != nil {
		if err := fs.primary.Close(); err != nil {
			firstErr = err
		}
	}
	if fs.secondary != nil {
		if err := fs.secondary.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
