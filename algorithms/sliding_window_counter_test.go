package algorithms

import (
	"context"
	"testing"
	"time"

	"github.com/bshome19/rateshield/store"
)

func TestSlidingWindowCounter(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	swc, err := NewSlidingWindowCounter(SlidingWindowCounterConfig{
		Limit:  5,
		Window: 100 * time.Millisecond,
		Store:  memStore,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()

	// First 5 requests should be allowed
	for i := 0; i < 5; i++ {
		res, err := swc.Allow(ctx, "user1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.Allowed {
			t.Fatalf("expected request %d to be allowed", i+1)
		}
	}

	// 6th request should be blocked
	res, err := swc.Allow(ctx, "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Allowed {
		t.Fatalf("expected 6th request to be blocked")
	}

	// Wait for window to slide
	time.Sleep(120 * time.Millisecond)

	// Test AllowN
	resN, err := swc.AllowN(ctx, "userN", 2)
	if err != nil || !resN.Allowed {
		t.Fatalf("AllowN failed: %v", err)
	}

	// Test Reset
	err = swc.Reset(ctx, "user1")
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
}

func TestSlidingWindowCounter_InvalidConfig(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	_, err := NewSlidingWindowCounter(SlidingWindowCounterConfig{Limit: 0, Window: time.Minute, Store: memStore})
	if err == nil {
		t.Error("expected error on zero limit")
	}

	_, err = NewSlidingWindowCounter(SlidingWindowCounterConfig{Limit: 10, Window: 0, Store: memStore})
	if err == nil {
		t.Error("expected error on zero window")
	}

	_, err = NewSlidingWindowCounter(SlidingWindowCounterConfig{Limit: 10, Window: time.Minute, Store: nil})
	if err == nil {
		t.Error("expected error on nil store")
	}
}
