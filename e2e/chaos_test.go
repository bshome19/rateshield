package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bshome19/rateshield/algorithms"
	"github.com/bshome19/rateshield/store"
	"github.com/redis/go-redis/v9"
)

// TestE2E_ChaosRedisOutageFailover tests what happens when Redis crashes mid-traffic.
func TestE2E_ChaosRedisOutageFailover(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	rClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	primaryStore, err := store.NewRedis(store.RedisConfig{Client: rClient})
	if err != nil {
		t.Fatalf("failed to create redis store: %v", err)
	}

	secondaryStore := store.NewMemory()
	defer secondaryStore.Close()

	var errorCallbackInvoked bool
	fallbackStore := store.NewFallback(store.FallbackConfig{
		Primary:   primaryStore,
		Secondary: secondaryStore,
		OnPrimaryError: func(err error) {
			errorCallbackInvoked = true
		},
	})

	limiter, err := algorithms.NewFixedWindow(algorithms.FixedWindowConfig{
		Limit:  5,
		Window: time.Minute,
		Store:  fallbackStore,
	})
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	ctx := context.Background()

	// 1. Initial 2 requests while Redis is alive
	for i := 0; i < 2; i++ {
		res, err := limiter.Allow(ctx, "chaos_user")
		if err != nil || !res.Allowed {
			t.Fatalf("request %d failed on primary: %v", i+1, err)
		}
	}

	// 2. CHAOS EVENT: Shut down Redis!
	mr.Close()

	// 3. Requests should seamlessly fallback to memory store without throwing errors!
	for i := 0; i < 2; i++ {
		res, err := limiter.Allow(ctx, "chaos_user")
		if err != nil {
			t.Fatalf("request %d failed during Redis outage: %v", i+3, err)
		}
		if !res.Allowed {
			t.Fatalf("expected request %d to be allowed under fallback memory store", i+3)
		}
	}

	if !errorCallbackInvoked {
		t.Errorf("Expected OnPrimaryError callback to be triggered during chaos event")
	}
}
