# Rate Limiting Algorithms

rateshield supports three popular rate limiting algorithms. Each has different characteristics and use cases.

## Token Bucket

**Best for:** APIs with bursty traffic patterns

### How it works:
- Tokens are added to a bucket at a fixed rate
- Each request consumes one token
- Requests are allowed if tokens are available
- Bucket has a maximum capacity (burst size)

### Configuration:
```go
limiter, err := algorithms.NewTokenBucket(algorithms.TokenBucketConfig{
    Rate:     10,    // 10 tokens per second
    Capacity: 20,    // Max 20 tokens (burst)
    Store:    store,
})
```

### Pros:
- Allows controlled bursts
- Smooth rate limiting
- Memory efficient

### Cons:
- More complex to understand
- Burst can cause issues if too large

---

## Sliding Window

**Best for:** Precise rate limiting, API quotas

### How it works:
- Tracks timestamps of all requests in the window
- Counts requests in the last N seconds/minutes
- Provides smooth, accurate limiting

### Configuration:
```go
limiter, err := algorithms.NewSlidingWindow(algorithms.SlidingWindowConfig{
    Limit:  100,           // 100 requests
    Window: time.Minute,   // per minute
    Store:  store,
})
```

### Pros:
- Most accurate
- No boundary issues
- Smooth distribution

### Cons:
- Higher memory usage (stores timestamps)
- More computation required

---

## Fixed Window

**Best for:** Simple rate limiting, high performance

### How it works:
- Divides time into fixed windows (e.g., every minute)
- Counts requests in each window
- Resets count when window changes

### Configuration:
```go
limiter, err := algorithms.NewFixedWindow(algorithms.FixedWindowConfig{
    Limit:  100,           // 100 requests
    Window: time.Minute,   // per minute
    Store:  store,
})
```

### Pros:
- Simple to understand
- Low memory usage
- Fast performance

### Cons:
- Boundary problem: 2x burst at window edges
- Less smooth than other algorithms

---

## Comparison Table

| Algorithm      | Burst Handling | Accuracy | Memory | Performance |
|----------------|----------------|----------|--------|-------------|
| Token Bucket   | Controlled     | Good     | Low    | Fast        |
| Sliding Window | None           | Best     | High   | Medium      |
| Fixed Window   | At boundaries  | Good     | Low    | Fastest     |

## Recommendations

- **Public APIs:** Sliding Window (most fair)
- **Internal Services:** Token Bucket (allows bursts)
- **High Traffic:** Fixed Window (best performance)
- **Premium Users:** Token Bucket with higher capacity