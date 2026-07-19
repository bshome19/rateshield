# Getting Started with rateshield

This guide will help you integrate rateshield into your Go application.

## Installation

```bash
go get github.com/bshome19/rateshield
```

## Basic Usage

### 1. Create a Store

First, create a storage backend:

```go
import "github.com/bshome19/rateshield/store"

// In-memory store (single instance)
memStore := store.NewMemory()
defer memStore.Close()

// Redis store (distributed)
redisStore, err := store.NewRedis(store.RedisConfig{
    Client: redisClient,
    Prefix: "ratelimit:",
})
```

### 2. Create a Limiter

Choose an algorithm and create a limiter:

```go
import "github.com/bshome19/rateshield/algorithms"

// Token Bucket: 10 req/sec, burst of 20
limiter, err := algorithms.NewTokenBucket(algorithms.TokenBucketConfig{
    Rate:     10,
    Capacity: 20,
    Store:    memStore,
})

// Sliding Window: 100 requests per minute
limiter, err := algorithms.NewSlidingWindow(algorithms.SlidingWindowConfig{
    Limit:  100,
    Window: time.Minute,
    Store:  memStore,
})

// Fixed Window: 1000 requests per hour
limiter, err := algorithms.NewFixedWindow(algorithms.FixedWindowConfig{
    Limit:  1000,
    Window: time.Hour,
    Store:  memStore,
})
```

### 3. Check Rate Limits

```go
result, err := limiter.Allow(ctx, "user:123")
if err != nil {
    // Handle error
}

if result.Allowed {
    // Process request
} else {
    // Reject request, retry after result.RetryAfter
}
```

### 4. Use Middleware

```go
import "github.com/bshome19/rateshield/middleware"

// For net/http
handler := middleware.Stdlib(middleware.Config{
    Limiter: limiter,
})(yourHandler)

// For Gin
router.Use(middleware.GinSimple(limiter))
```

## Next Steps

- Read [Algorithms Explained](algorithms.md) to understand the different algorithms
- Check out the [examples](../examples/) for complete working code