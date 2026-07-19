package e2e

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bshome19/rateshield/algorithms"
	"github.com/bshome19/rateshield/store"
	"github.com/redis/go-redis/v9"
)

// TestE2E_HighConcurrencyStress tests 10,000 concurrent requests against TokenBucket.
func TestE2E_HighConcurrencyStress(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	// Capacity 5000, 1000/sec refill rate
	limiter, err := algorithms.NewTokenBucket(algorithms.TokenBucketConfig{
		Rate:     1000,
		Capacity: 5000,
		Store:    memStore,
	})
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	const totalGoroutines = 10000
	var allowedCount int64
	var blockedCount int64
	var wg sync.WaitGroup

	ctx := context.Background()
	startTime := time.Now()

	for i := 0; i < totalGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "stress_user"
			res, err := limiter.Allow(ctx, key)
			if err != nil {
				t.Errorf("unexpected error for worker %d: %v", id, err)
				return
			}
			if res.Allowed {
				atomic.AddInt64(&allowedCount, 1)
			} else {
				atomic.AddInt64(&blockedCount, 1)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("⚡ 10,000 Concurrent Requests Executed in %v", duration)
	t.Logf("Allowed: %d, Blocked: %d", allowedCount, blockedCount)

	if allowedCount+blockedCount != totalGoroutines {
		t.Errorf("Expected total %d requests, got %d", totalGoroutines, allowedCount+blockedCount)
	}
	maxAllowed := 5000 + int64(duration.Seconds()*1000) + 100
	if allowedCount > maxAllowed {
		t.Errorf("Allowed count %d exceeded max allowed %d for duration %v", allowedCount, maxAllowed, duration)
	}
}

// TestE2E_DistributedRedisStress tests concurrent requests against miniredis Lua scripts.
func TestE2E_DistributedRedisStress(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rStore, err := store.NewRedis(store.RedisConfig{
		Client: rClient,
		Prefix: "stress:",
	})
	if err != nil {
		t.Fatalf("failed to create redis store: %v", err)
	}

	limiter, _ := algorithms.NewSlidingWindowCounter(algorithms.SlidingWindowCounterConfig{
		Limit:  1000,
		Window: time.Minute,
		Store:  rStore,
	})

	const totalWorkers = 2000
	var allowedCount int64
	var blockedCount int64
	var wg sync.WaitGroup

	ctx := context.Background()

	for i := 0; i < totalWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := limiter.Allow(ctx, "redis_user")
			if err != nil {
				t.Errorf("redis error: %v", err)
				return
			}
			if res.Allowed {
				atomic.AddInt64(&allowedCount, 1)
			} else {
				atomic.AddInt64(&blockedCount, 1)
			}
		}()
	}

	wg.Wait()

	if allowedCount != 1000 {
		t.Errorf("Expected exactly 1000 allowed requests in Redis sliding window counter, got %d", allowedCount)
	}
	if blockedCount != 1000 {
		t.Errorf("Expected exactly 1000 blocked requests, got %d", blockedCount)
	}
}
