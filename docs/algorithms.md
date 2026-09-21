# Rate Limiting Algorithms

`rateshield` provides four high-performance rate limiting algorithms, each optimized for different traffic characteristics and precision requirements.

---

## 1. Token Bucket

**Best for:** APIs with bursty traffic patterns, public endpoints.

### How it works:
- Tokens are continuously added to a bucket at a fixed rate per second.
- Each request consumes $N$ tokens (typically 1).
- Requests are permitted if sufficient tokens are available in the bucket.
- The bucket has a maximum capacity limit (burst size).

### Configuration:
```go
limiter, err := rateshield.New(
    rateshield.WithAlgorithm(rateshield.TokenBucketAlgorithm),
    rateshield.WithRate(10),     // 10 tokens added per second
    rateshield.WithCapacity(20), // Burst capacity of 20 tokens
)
```

### Characteristics:
- **Burst Handling:** Excellent (controlled bursts up to capacity).
- **Smoothness:** High.
- **Memory:** $O(1)$.
- **Performance:** Ultra-fast (<45ns in multi-key sharded memory).

---

## 2. Sliding Window Counter

**Best for:** High-scale distributed web applications, Cloudflare-standard rate limiting.

### How it works:
- Divides time into sliding windows and tracks request counts across the current and previous window boundaries.
- Uses a weighted estimation formula:
  $$\text{Count} = \text{Count}_{\text{prev}} \times \left(1 - \frac{t - t_{\text{start}}}{\text{window}}\right) + \text{Count}_{\text{curr}}$$
- Provides smooth transitions across window boundaries without storing timestamps.

### Configuration:
```go
limiter, err := rateshield.New(
    rateshield.WithAlgorithm(rateshield.SlidingWindowCounterAlgorithm),
    rateshield.WithLimit(100),
    rateshield.WithWindow(time.Minute),
)
```

### Characteristics:
- **Burst Handling:** Smooth boundary decay.
- **Accuracy:** ~99.9% estimation accuracy.
- **Memory:** $O(1)$ constant memory.
- **Performance:** Extremely fast.

---

## 3. Fixed Window Counter

**Best for:** Periodic quotas (e.g. 10,000 requests/day, tier-based billing quotas).

### How it works:
- Divides time into discrete, fixed windows (e.g., top of the hour or minute).
- Maintains a simple integer counter per window.
- Counter resets immediately when the window timestamp rolls over.

### Configuration:
```go
limiter, err := rateshield.New(
    rateshield.WithAlgorithm(rateshield.FixedWindowAlgorithm),
    rateshield.WithLimit(1000),
    rateshield.WithWindow(time.Hour),
)
```

### Characteristics:
- **Burst Handling:** Subject to boundary spikes (up to $2\times$ limit at window edges).
- **Accuracy:** Exact per fixed window.
- **Memory:** $O(1)$ lowest memory footprint.
- **Performance:** Fastest.

---

## 4. Sliding Window Log

**Best for:** Strict compliance auditing, financial transactions, zero-burst tolerance.

### How it works:
- Appends exact timestamp logs of every admitted request.
- On each evaluation, evicts timestamps outside the window range and computes the exact count.

### Configuration:
```go
limiter, err := rateshield.New(
    rateshield.WithAlgorithm(rateshield.SlidingWindowAlgorithm),
    rateshield.WithLimit(100),
    rateshield.WithWindow(time.Minute),
)
```

### Characteristics:
- **Burst Handling:** Zero burst allowed beyond strict limit.
- **Accuracy:** 100% mathematically exact.
- **Memory:** $O(N)$ where $N$ is the number of active requests in the window.
- **Performance:** Higher CPU overhead under very high QPS.

---

## Summary Comparison Matrix

| Algorithm | Burst Handling | Accuracy | Memory Complexity | CPU Complexity | Recommended Use Case |
| :--- | :---: | :---: | :---: | :---: | :--- |
| **Token Bucket** | Controlled burst | High | $O(1)$ | $O(1)$ | Public REST APIs, Bursty Traffic |
| **Sliding Window Counter** | Smooth decay | ~99.9% | $O(1)$ | $O(1)$ | Microservices, High-QPS Endpoints |
| **Fixed Window** | Spike at boundary | Exact per window | $O(1)$ | $O(1)$ | Tier Quotas (Daily/Monthly) |
| **Sliding Window Log** | Zero burst | 100% exact | $O(N)$ | $O(\log N)$ | Auditing, High-Security Endpoints |