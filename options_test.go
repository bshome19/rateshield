package rateshield

import (
	"net/http"
	"testing"
	"time"

	"github.com/bshome19/rateshield/store"
)

func TestOptions_Apply(t *testing.T) {
	mem := store.NewMemory()
	defer mem.Close()

	opts := DefaultOptions()
	opts.Apply(
		WithAlgorithm(SlidingWindowCounterAlgorithm),
		WithRate(50),
		WithCapacity(100),
		WithLimit(500),
		WithWindow(5*time.Minute),
		WithStore(mem),
		WithTTL(10*time.Minute),
		WithFailStrategy(FailClosed),
		WithKeyExtractor(func(r *http.Request) string { return "custom" }),
	)

	if opts.Algorithm != SlidingWindowCounterAlgorithm {
		t.Errorf("expected SlidingWindowCounterAlgorithm, got %v", opts.Algorithm)
	}
	if opts.Rate != 50 {
		t.Errorf("expected rate 50, got %f", opts.Rate)
	}
	if opts.Capacity != 100 {
		t.Errorf("expected capacity 100, got %d", opts.Capacity)
	}
	if opts.Limit != 500 {
		t.Errorf("expected limit 500, got %d", opts.Limit)
	}
	if opts.Window != 5*time.Minute {
		t.Errorf("expected window 5m, got %v", opts.Window)
	}
	if opts.Store != mem {
		t.Errorf("store mismatch")
	}
	if opts.TTL != 10*time.Minute {
		t.Errorf("expected TTL 10m, got %v", opts.TTL)
	}
	if opts.FailStrategy != FailClosed {
		t.Errorf("expected FailClosed strategy")
	}
}

func TestNew_Algorithms(t *testing.T) {
	ctx := t.Context()

	// 1. Token Bucket
	tb, err := New(
		WithAlgorithm(TokenBucketAlgorithm),
		WithRate(10),
		WithCapacity(20),
	)
	if err != nil {
		t.Fatalf("failed to create TokenBucket: %v", err)
	}
	defer tb.Close()

	res, err := tb.Allow(ctx, "user1")
	if err != nil || !res.Allowed {
		t.Fatalf("TokenBucket Allow failed: %v", err)
	}

	// 2. Fixed Window
	fw, err := New(
		WithAlgorithm(FixedWindowAlgorithm),
		WithLimit(100),
		WithWindow(time.Minute),
	)
	if err != nil {
		t.Fatalf("failed to create FixedWindow: %v", err)
	}
	defer fw.Close()

	res, err = fw.Allow(ctx, "user1")
	if err != nil || !res.Allowed {
		t.Fatalf("FixedWindow Allow failed: %v", err)
	}

	// 3. Sliding Window Counter
	swc, err := New(
		WithAlgorithm(SlidingWindowCounterAlgorithm),
		WithLimit(100),
		WithWindow(time.Minute),
	)
	if err != nil {
		t.Fatalf("failed to create SlidingWindowCounter: %v", err)
	}
	defer swc.Close()

	res, err = swc.Allow(ctx, "user1")
	if err != nil || !res.Allowed {
		t.Fatalf("SlidingWindowCounter Allow failed: %v", err)
	}

	// 4. Sliding Window Log
	swl, err := New(
		WithAlgorithm(SlidingWindowAlgorithm),
		WithLimit(100),
		WithWindow(time.Minute),
	)
	if err != nil {
		t.Fatalf("failed to create SlidingWindow: %v", err)
	}
	defer swl.Close()

	res, err = swl.Allow(ctx, "user1")
	if err != nil || !res.Allowed {
		t.Fatalf("SlidingWindow Allow failed: %v", err)
	}
}

func TestNew_InvalidAlgorithm(t *testing.T) {
	_, err := New(WithAlgorithm(Algorithm(999)))
	if err == nil {
		t.Error("expected error for invalid algorithm")
	}
}
