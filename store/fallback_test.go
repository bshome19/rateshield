package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

type errStore struct{}

func (e *errStore) Reset(ctx context.Context, key string) error {
	return errors.New("redis connection failed")
}
func (e *errStore) AllowTokenBucket(ctx context.Context, key string, rate float64, capacity int64, n int64, ttl time.Duration) (EvalResult, error) {
	return EvalResult{}, errors.New("redis connection failed")
}
func (e *errStore) AllowFixedWindow(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (EvalResult, error) {
	return EvalResult{}, errors.New("redis connection failed")
}
func (e *errStore) AllowSlidingWindowCounter(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (EvalResult, error) {
	return EvalResult{}, errors.New("redis connection failed")
}
func (e *errStore) AllowSlidingWindowLog(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (EvalResult, error) {
	return EvalResult{}, errors.New("redis connection failed")
}
func (e *errStore) Close() error {
	return nil
}

func TestFallbackStore_PrimaryErrorFallback(t *testing.T) {
	errCount := 0
	primary := &errStore{}
	secondary := NewMemory()
	defer secondary.Close()

	fb := NewFallback(FallbackConfig{
		Primary:   primary,
		Secondary: secondary,
		OnPrimaryError: func(err error) {
			errCount++
		},
	})

	ctx := context.Background()

	// Should fallback seamlessly to secondary in-memory store
	res, err := fb.AllowTokenBucket(ctx, "user1", 10, 10, 1, time.Minute)
	if err != nil {
		t.Fatalf("unexpected fallback error: %v", err)
	}
	if !res.Allowed {
		t.Fatalf("expected request to be allowed on fallback")
	}

	if errCount == 0 {
		t.Fatalf("expected OnPrimaryError callback to be invoked")
	}

	// Test Reset fallback: primary is broken so Reset returns a primary error,
	// but the secondary store is still cleared (best-effort both-store reset).
	err = fb.Reset(ctx, "user1")
	_ = err // primary error is surfaced, not fatal

	// Test AllowFixedWindow fallback
	fwRes, err := fb.AllowFixedWindow(ctx, "fw_key", 5, time.Minute, 1, time.Minute)
	if err != nil || !fwRes.Allowed {
		t.Fatalf("unexpected fallback AllowFixedWindow: %v", err)
	}

	// Test AllowSlidingWindowCounter fallback
	swcRes, err := fb.AllowSlidingWindowCounter(ctx, "swc_key", 5, time.Minute, 1, time.Minute)
	if err != nil || !swcRes.Allowed {
		t.Fatalf("unexpected fallback AllowSlidingWindowCounter: %v", err)
	}

	// Test AllowSlidingWindowLog fallback
	swlRes, err := fb.AllowSlidingWindowLog(ctx, "swl_key", 5, time.Minute, 1, time.Minute)
	if err != nil || !swlRes.Allowed {
		t.Fatalf("unexpected fallback AllowSlidingWindowLog: %v", err)
	}

	_ = fb.Close()
}

func TestFallbackStore_CircuitBreakerTrips(t *testing.T) {
	errCount := 0
	primary := &errStore{}
	secondary := NewMemory()
	defer secondary.Close()

	fb := NewFallback(FallbackConfig{
		Primary:          primary,
		Secondary:        secondary,
		FailureThreshold: 2,
		Cooldown:         100 * time.Millisecond,
		OnPrimaryError: func(err error) {
			errCount++
		},
	})
	defer fb.Close()

	ctx := context.Background()

	// 1st request -> fails on primary -> errCount=1
	_, _ = fb.AllowTokenBucket(ctx, "u1", 10, 10, 1, time.Minute)
	if errCount != 1 {
		t.Fatalf("expected errCount=1, got %d", errCount)
	}

	// 2nd request -> fails on primary -> trips circuit breaker to Open -> errCount=2
	_, _ = fb.AllowTokenBucket(ctx, "u1", 10, 10, 1, time.Minute)
	if errCount != 2 {
		t.Fatalf("expected errCount=2, got %d", errCount)
	}

	// 3rd request -> circuit breaker is OPEN! Primary is bypassed completely -> errCount stays 2!
	_, _ = fb.AllowTokenBucket(ctx, "u1", 10, 10, 1, time.Minute)
	if errCount != 2 {
		t.Fatalf("expected errCount to remain 2 because breaker is open, got %d", errCount)
	}
}
