package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bshome19/rateshield/algorithms"
	"github.com/bshome19/rateshield/store"
)

func main() {
	fmt.Println("🚦 rateshield - Basic Example")
	fmt.Println("==========================")

	// Create an in-memory store
	memStore := store.NewMemory()
	defer memStore.Close()

	// Create a token bucket limiter: 5 requests/second, burst of 10
	limiter, err := algorithms.NewTokenBucket(algorithms.TokenBucketConfig{
		Rate:     5,
		Capacity: 10,
		Store:    memStore,
	})
	if err != nil {
		log.Fatalf("Failed to create limiter: %v", err)
	}

	ctx := context.Background()
	userKey := "user:123"

	fmt.Println()
	fmt.Println("📊 Simulating 15 rapid requests...")
	fmt.Println()

	// Simulate rapid requests
	for i := 1; i <= 15; i++ {
		result, err := limiter.Allow(ctx, userKey)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		status := "✅ ALLOWED"
		if !result.Allowed {
			status = "❌ DENIED"
		}

		fmt.Printf("Request %2d: %s | Remaining: %d/%d",
			i, status, result.Remaining, result.Limit)

		if !result.Allowed {
			fmt.Printf(" | Retry after: %v", result.RetryAfter.Round(time.Millisecond))
		}
		fmt.Println()

		time.Sleep(50 * time.Millisecond)
	}

	fmt.Println()
	fmt.Println("⏳ Waiting 2 seconds for tokens to refill...")
	time.Sleep(2 * time.Second)

	fmt.Println()
	fmt.Println("📊 Trying 3 more requests after waiting...")
	fmt.Println()

	for i := 1; i <= 3; i++ {
		result, _ := limiter.Allow(ctx, userKey)
		status := "✅ ALLOWED"
		if !result.Allowed {
			status = "❌ DENIED"
		}
		fmt.Printf("Request %d: %s | Remaining: %d/%d\n",
			i, status, result.Remaining, result.Limit)
	}

	fmt.Println()
	fmt.Println("✨ Example complete!")
}
