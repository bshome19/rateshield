package store

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func createTestRedisStore(t *testing.T) (*Redis, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	redisStore, err := NewRedis(RedisConfig{
		Client: client,
		Prefix: "test_rateshield:",
	})
	if err != nil {
		mr.Close()
		t.Fatalf("failed to create redis store: %v", err)
	}

	return redisStore, mr
}

func TestRedisStore_BasicOperations(t *testing.T) {
	s, mr := createTestRedisStore(t)
	defer s.Close()
	defer mr.Close()

	ctx := context.Background()
	key := "user1"

	// Set & Get
	st := &State{Tokens: 10, LastUpdate: time.Now(), Count: 5}
	err := s.Set(ctx, key, st, time.Hour)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := s.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Count != 5 {
		t.Errorf("Expected count 5, got %d", got.Count)
	}

	// Increment on fresh key
	incrKey := "user_incr"
	cnt, err := s.Increment(ctx, incrKey, time.Hour)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if cnt != 1 {
		t.Errorf("Expected count 1, got %d", cnt)
	}

	// Reset
	err = s.Reset(ctx, key)
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
}

func TestRedisStore_AtomicAlgorithms(t *testing.T) {
	s, mr := createTestRedisStore(t)
	defer s.Close()
	defer mr.Close()

	ctx := context.Background()

	// Token Bucket: rate, capacity, n, ttl
	tbRes, err := s.AllowTokenBucket(ctx, "tb_key", 10.0, 5, 1, time.Minute)
	if err != nil {
		t.Fatalf("AllowTokenBucket failed: %v", err)
	}
	if !tbRes.Allowed {
		t.Error("Expected token bucket allowed")
	}

	// Fixed Window: limit, window, n, ttl
	fwRes, err := s.AllowFixedWindow(ctx, "fw_key", 5, time.Minute, 1, time.Minute)
	if err != nil {
		t.Fatalf("AllowFixedWindow failed: %v", err)
	}
	if !fwRes.Allowed {
		t.Error("Expected fixed window allowed")
	}

	// Sliding Window Counter: limit, window, n, ttl
	swcRes, err := s.AllowSlidingWindowCounter(ctx, "swc_key", 5, time.Minute, 1, time.Minute)
	if err != nil {
		t.Fatalf("AllowSlidingWindowCounter failed: %v", err)
	}
	if !swcRes.Allowed {
		t.Error("Expected sliding window counter allowed")
	}

	// Sliding Window Log: limit, window, n, ttl
	swlRes, err := s.AllowSlidingWindowLog(ctx, "swl_key", 5, time.Minute, 1, time.Minute)
	if err != nil {
		t.Fatalf("AllowSlidingWindowLog failed: %v", err)
	}
	if !swlRes.Allowed {
		t.Error("Expected sliding window log allowed")
	}
}

func TestNewRedis_NilClient(t *testing.T) {
	_, err := NewRedis(RedisConfig{Client: nil})
	if err == nil {
		t.Error("Expected error when Client is nil")
	}
}
