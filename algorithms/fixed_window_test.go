package algorithms

import (
	"context"
	"testing"
	"time"

	"github.com/bshome19/rateshield/store"
)

func TestNewFixedWindow(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	tests := []struct {
		name    string
		cfg     FixedWindowConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: FixedWindowConfig{
				Limit:  100,
				Window: time.Minute,
				Store:  memStore,
			},
			wantErr: false,
		},
		{
			name: "zero limit",
			cfg: FixedWindowConfig{
				Limit:  0,
				Window: time.Minute,
				Store:  memStore,
			},
			wantErr: true,
		},
		{
			name: "zero window",
			cfg: FixedWindowConfig{
				Limit:  100,
				Window: 0,
				Store:  memStore,
			},
			wantErr: true,
		},
		{
			name: "nil store",
			cfg: FixedWindowConfig{
				Limit:  100,
				Window: time.Minute,
				Store:  nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFixedWindow(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFixedWindow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFixedWindow_Allow(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	limiter, err := NewFixedWindow(FixedWindowConfig{
		Limit:  5,
		Window: time.Minute, // Use longer window to avoid timing issues
		Store:  memStore,
	})
	if err != nil {
		t.Fatalf("Failed to create limiter: %v", err)
	}

	ctx := context.Background()
	key := "test-user"

	// First 5 requests should be allowed
	for i := 0; i < 5; i++ {
		result, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Request %d: unexpected error: %v", i+1, err)
		}
		if !result.Allowed {
			t.Errorf("Request %d: expected allowed, got denied", i+1)
		}
	}

	// 6th request should be denied
	result, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Request 6: unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("Request 6: expected denied, got allowed")
	}
}

func TestFixedWindow_AllowN(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	limiter, err := NewFixedWindow(FixedWindowConfig{
		Limit:  10,
		Window: time.Minute,
		Store:  memStore,
	})
	if err != nil {
		t.Fatalf("Failed to create limiter: %v", err)
	}

	ctx := context.Background()
	key := "test-user"

	// Request 7 at once
	result, _ := limiter.AllowN(ctx, key, 7)
	if !result.Allowed {
		t.Error("Expected 7 requests to be allowed")
	}

	// Request 3 more (should be allowed, total = 10)
	result, _ = limiter.AllowN(ctx, key, 3)
	if !result.Allowed {
		t.Error("Expected 3 more requests to be allowed")
	}

	// Request 1 more (should be denied)
	result, _ = limiter.AllowN(ctx, key, 1)
	if result.Allowed {
		t.Error("Expected request to be denied")
	}
}

func TestFixedWindow_WindowReset(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	// Use a short window for testing
	windowDuration := 100 * time.Millisecond

	limiter, err := NewFixedWindow(FixedWindowConfig{
		Limit:  3,
		Window: windowDuration,
		Store:  memStore,
	})
	if err != nil {
		t.Fatalf("Failed to create limiter: %v", err)
	}

	ctx := context.Background()

	// Use unique key to avoid any caching issues
	key := "test-user-window-reset"

	// Exhaust limit
	for i := 0; i < 3; i++ {
		result, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Request %d error: %v", i+1, err)
		}
		if !result.Allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should be denied now
	result, _ := limiter.Allow(ctx, key)
	if result.Allowed {
		t.Error("Expected to be denied after exhausting limit")
	}

	// Wait for at least 2 windows to pass to ensure we're in a new window
	time.Sleep(windowDuration*2 + 50*time.Millisecond)

	// Use a new key to ensure clean state in new window
	key2 := "test-user-window-reset-2"

	// Should be allowed in new window
	result, err = limiter.Allow(ctx, key2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("Expected to be allowed in new window with new key")
	}

	// Also verify the original key works in the new window
	// The window number should have changed
	result, err = limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("Expected to be allowed in new window with original key")
	}
}

func TestFixedWindow_DifferentKeys(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	limiter, err := NewFixedWindow(FixedWindowConfig{
		Limit:  2,
		Window: time.Minute,
		Store:  memStore,
	})
	if err != nil {
		t.Fatalf("Failed to create limiter: %v", err)
	}

	ctx := context.Background()

	// Exhaust user1
	limiter.Allow(ctx, "user1")
	limiter.Allow(ctx, "user1")
	result, _ := limiter.Allow(ctx, "user1")
	if result.Allowed {
		t.Error("user1 should be rate limited")
	}

	// user2 should be fine
	result, _ = limiter.Allow(ctx, "user2")
	if !result.Allowed {
		t.Error("user2 should not be rate limited")
	}
}

func TestFixedWindow_Reset(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	limiter, err := NewFixedWindow(FixedWindowConfig{
		Limit:  2,
		Window: time.Minute,
		Store:  memStore,
	})
	if err != nil {
		t.Fatalf("Failed to create limiter: %v", err)
	}

	ctx := context.Background()
	key := "test-user"

	// Exhaust limit
	limiter.Allow(ctx, key)
	limiter.Allow(ctx, key)
	result, _ := limiter.Allow(ctx, key)
	if result.Allowed {
		t.Error("Expected to be rate limited")
	}

	// Reset
	err = limiter.Reset(ctx, key)
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	// Should be allowed
	result, _ = limiter.Allow(ctx, key)
	if !result.Allowed {
		t.Error("Expected to be allowed after reset")
	}
}

func TestFixedWindow_RetryAfter(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	limiter, err := NewFixedWindow(FixedWindowConfig{
		Limit:  1,
		Window: time.Second,
		Store:  memStore,
	})
	if err != nil {
		t.Fatalf("Failed to create limiter: %v", err)
	}

	ctx := context.Background()
	key := "test-user"

	// First request
	limiter.Allow(ctx, key)

	// Second request should be denied with RetryAfter
	result, _ := limiter.Allow(ctx, key)
	if result.Allowed {
		t.Error("Expected to be denied")
	}
	if result.RetryAfter <= 0 {
		t.Error("Expected positive RetryAfter")
	}
	if result.RetryAfter > time.Second {
		t.Errorf("RetryAfter too large: %v", result.RetryAfter)
	}
}

func TestFixedWindow_Sequential(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	limit := int64(10)
	limiter, err := NewFixedWindow(FixedWindowConfig{
		Limit:  limit,
		Window: time.Minute,
		Store:  memStore,
	})
	if err != nil {
		t.Fatalf("Failed to create limiter: %v", err)
	}

	ctx := context.Background()
	key := "sequential-test"

	allowedCount := int64(0)

	// Sequential requests - should be exactly limited
	for i := 0; i < 20; i++ {
		result, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.Allowed {
			allowedCount++
		}
	}

	if allowedCount != limit {
		t.Errorf("Expected exactly %d allowed requests, got %d", limit, allowedCount)
	}
}

func TestFixedWindow_WindowNumber(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	windowDuration := 100 * time.Millisecond

	limiter, err := NewFixedWindow(FixedWindowConfig{
		Limit:  5,
		Window: windowDuration,
		Store:  memStore,
	})
	if err != nil {
		t.Fatalf("Failed to create limiter: %v", err)
	}

	ctx := context.Background()
	key := "window-number-test"

	// Make some requests
	result, _ := limiter.Allow(ctx, key)
	if !result.Allowed {
		t.Error("First request should be allowed")
	}

	result, _ = limiter.Allow(ctx, key)
	if !result.Allowed {
		t.Error("Second request should be allowed")
	}

	// Wait for new window
	time.Sleep(windowDuration + 20*time.Millisecond)

	// Should have full limit again in new window
	for i := 0; i < 5; i++ {
		result, _ = limiter.Allow(ctx, key)
		if !result.Allowed {
			t.Errorf("Request %d in new window should be allowed", i+1)
		}
	}

	// 6th request in new window should be denied
	result, _ = limiter.Allow(ctx, key)
	if result.Allowed {
		t.Error("6th request in new window should be denied")
	}
}
