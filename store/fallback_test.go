package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

type errStore struct{}

func (e *errStore) Get(ctx context.Context, key string) (*State, error) {
	return nil, errors.New("redis connection failed")
}
func (e *errStore) Set(ctx context.Context, key string, state *State, ttl time.Duration) error {
	return errors.New("redis connection failed")
}
func (e *errStore) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	return 0, errors.New("redis connection failed")
}
func (e *errStore) Reset(ctx context.Context, key string) error {
	return errors.New("redis connection failed")
}
func (e *errStore) AllowTokenBucket(ctx context.Context, key string, rate float64, capacity int64, n int64, ttl time.Duration) (*EvalResult, error) {
	return nil, errors.New("redis connection failed")
}
func (e *errStore) AllowFixedWindow(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (*EvalResult, error) {
	return nil, errors.New("redis connection failed")
}
func (e *errStore) AllowSlidingWindowCounter(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (*EvalResult, error) {
	return nil, errors.New("redis connection failed")
}
func (e *errStore) AllowSlidingWindowLog(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (*EvalResult, error) {
	return nil, errors.New("redis connection failed")
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

	// Test Get & Set fallback
	err = fb.Set(ctx, "key1", &State{Count: 5}, time.Minute)
	if err != nil {
		t.Fatalf("unexpected fallback Set error: %v", err)
	}

	state, err := fb.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("unexpected fallback Get error: %v", err)
	}
	if state.Count != 5 {
		t.Fatalf("expected state count 5, got %d", state.Count)
	}

	// Test Reset fallback
	err = fb.Reset(ctx, "key1")
	if err != nil {
		t.Fatalf("unexpected fallback Reset error: %v", err)
	}

	// Test Increment fallback
	cnt, err := fb.Increment(ctx, "key2", time.Minute)
	if err != nil || cnt != 1 {
		t.Fatalf("unexpected fallback Increment result: cnt=%d, err=%v", cnt, err)
	}

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
}
