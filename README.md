# 🚦 rateshield

A high-performance, distributed, zero-allocation rate limiting library for Go.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/bshome19/rateshield.svg)](https://pkg.go.dev/github.com/bshome19/rateshield)
[![Go Report Card](https://goreportcard.com/badge/github.com/bshome19/rateshield)](https://goreportcard.com/report/github.com/bshome19/rateshield)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## 🚀 Features

- ⚡ **Ultra-Fast & Zero-Allocation** — Sub-40ns execution time and `0 B/op` allocations for Token Bucket, Fixed Window, and Sliding Window Counter.
- 🔒 **Sharded Mutex Memory Engine** — 64-shard FNV-1a hash-partitioned in-memory store eliminates lock contention across CPU cores.
- 🔴 **Atomic Distributed Redis** — Single RTT atomic Lua script execution prevents race conditions across distributed cluster nodes.
- 🛡️ **Multi-Tier Resilient Failover** — Automatic fallback from Primary (e.g. Redis) to Secondary (e.g. In-Memory) when Redis connection times out or fails.
- 🎯 **Multiple Rate Limiting Algorithms**:
  - **Token Bucket** — Burst handling & smooth refill rate.
  - **Sliding Window Counter** — $O(1)$ memory complexity, Cloudflare-standard weighted window estimation.
  - **Fixed Window Counter** — Simple, fast window-based limiting.
  - **Sliding Window Log** — Exact request timestamp log.
- 🔌 **First-Class HTTP Middleware** — Out-of-the-box support for `net/http`, **Gin**, **Fiber**, **Echo**, and **Chi**.
- 📊 **Modern IETF RateLimit Headers** — Supports both standard legacy `X-RateLimit-*` and modern IETF draft `RateLimit-*` headers.

---

## ⚡ Performance Benchmarks

Tested on **12th Gen Intel(R) Core(TM) i5-12450H** (`go test -bench=. -benchmem ./algorithms`):

| Algorithm | ns/op | Memory (B/op) | Allocations | Complexity |
| :--- | :---: | :---: | :---: | :---: |
| **Token Bucket** | **34.84 ns** | **0 B/op** | **0 allocs** | $O(1)$ |
| **Fixed Window** | **35.61 ns** | **0 B/op** | **0 allocs** | $O(1)$ |
| **Sliding Window Counter** | **38.16 ns** | **0 B/op** | **0 allocs** | $O(1)$ |
| **Sliding Window Log** | **175.70 ns** | **48 B/op** | **1 alloc** | $O(N)$ |

---

## 🏗️ Architecture

`rateshield` follows the **Strategy Pattern** combined with **Dependency Inversion**:

```
 ┌──────────────────────────────────────────────────────────────────┐
 │                        HTTP Request / API                        │
 └────────────────────────────────┬─────────────────────────────────┘
                                  │
                                  ▼
 ┌──────────────────────────────────────────────────────────────────┐
 │                   Middleware / Key Extractor                     │
 │          (net/http, Gin, Fiber, Echo, Chi | IP / User)           │
 └────────────────────────────────┬─────────────────────────────────┘
                                  │
                                  ▼
 ┌──────────────────────────────────────────────────────────────────┐
 │                        rateshield.Limiter                            │
 └────────────────────────────────┬─────────────────────────────────┘
                                  │
         ┌────────────────────────┼────────────────────────┐
         ▼                        ▼                        ▼
 ┌───────────────┐        ┌───────────────┐        ┌───────────────┐
 │ Token Bucket  │        │ Fixed Window  │        │Sliding Window │
 └───────┬───────┘        └───────┬───────┘        └───────┬───────┘
         │                        │                        │
         └────────────────────────┼────────────────────────┘
                                  │
                                  ▼
 ┌──────────────────────────────────────────────────────────────────┐
 │                         store.Store                              │
 │   (Sharded Memory | Atomic Redis Lua | Resilient Fallback)       │
 └──────────────────────────────────────────────────────────────────┘
```

---

## 📦 Installation

```bash
go get github.com/bshome19/rateshield
```

**Requirements:** Go 1.22 or higher.

---

## 🚦 Quick Start

### 1. Basic Usage (Unified Options API)

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bshome19/rateshield"
)

func main() {
	// Create a limiter using Token Bucket algorithm (10 req/sec, capacity of 20)
	limiter, err := rateshield.New(
		rateshield.WithAlgorithm(rateshield.TokenBucketAlgorithm),
		rateshield.WithRate(10),
		rateshield.WithCapacity(20),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	res, err := limiter.Allow(ctx, "user:123")
	if err != nil {
		log.Fatal(err)
	}

	if res.Allowed {
		fmt.Printf("✅ Request allowed! Remaining: %d/%d\n", res.Remaining, res.Limit)
	} else {
		fmt.Printf("❌ Exceeded! Retry after: %v\n", res.RetryAfter)
	}
}
```

---

### 2. Distributed Redis Storage

```go
package main

import (
	"context"
	"time"

	"github.com/bshome19/rateshield"
	"github.com/bshome19/rateshield/store"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	redisStore, err := store.NewRedis(store.RedisConfig{
		Client: rdb,
		Prefix: "api:rateshield:",
	})
	if err != nil {
		panic(err)
	}

	// Sliding Window Counter limiter backed by Redis
	limiter, _ := rateshield.New(
		rateshield.WithAlgorithm(rateshield.SlidingWindowCounterAlgorithm),
		rateshield.WithLimit(100),
		rateshield.WithWindow(time.Minute),
		rateshield.WithStore(redisStore),
	)

	_ = limiter
}
```

---

### 3. Multi-Tier Resilient Storage Fallback

Automatically falls back to local in-memory rate limiting if Redis connection drops or times out:

```go
primaryRedis, _ := store.NewRedis(store.RedisConfig{Client: redisClient})
secondaryMem := store.NewMemory()

resilientStore := store.NewFallback(store.FallbackConfig{
	Primary:   primaryRedis,
	Secondary: secondaryMem,
	OnPrimaryError: func(err error) {
		log.Printf("⚠️ Redis error, falling back to local memory: %v", err)
	},
})

limiter, _ := rateshield.New(
	rateshield.WithAlgorithm(rateshield.TokenBucketAlgorithm),
	rateshield.WithStore(resilientStore),
)
```

---

## 🔌 Framework Support

### Standard `net/http`
```go
limiter, _ := rateshield.New(
    rateshield.WithAlgorithm(rateshield.SlidingWindowCounterAlgorithm),
    rateshield.WithLimit(100),
    rateshield.WithWindow(time.Minute),
)

handler := middleware.StdlibSimple(limiter)(mux)
http.ListenAndServe(":8080", handler)
```

### Gin
```go
import "github.com/bshome19/rateshield/middleware"

r := gin.Default()
r.Use(middleware.GinSimple(limiter))
```

### Fiber
```go
import "github.com/bshome19/rateshield/middleware"

app := fiber.New()
app.Use(middleware.FiberSimple(limiter))
```

### Echo
```go
import "github.com/bshome19/rateshield/middleware"

e := echo.New()
e.Use(middleware.EchoSimple(limiter))
```

---

## 📊 Algorithm Comparison

| Algorithm | Recommended Use Case | Pros | Cons |
| :--- | :--- | :--- | :--- |
| **Token Bucket** | API Rate Limiting, Bursty Traffic | Handles sudden bursts smoothly; high precision. | Requires tracking rate & capacity. |
| **Sliding Window Counter** | High-Scale Distributed Web Services | $O(1)$ memory; smooth boundary transition; Cloudflare standard. | Approximate estimation near boundary (~99.9% accuracy). |
| **Fixed Window** | Simple APIs, Fixed Quotas (e.g. 1000/day) | Extremely simple, low overhead. | Traffic spike at window boundaries. |
| **Sliding Window Log** | Audit Compliance, Strict Zero-Burst APIs | 100% exact request log tracking. | $O(N)$ memory growth under heavy QPS. |

---

## ⚙️ Options Reference

| Option | Description | Default |
| :--- | :--- | :--- |
| `WithAlgorithm(alg)` | Sets rate limiting algorithm | `TokenBucketAlgorithm` |
| `WithRate(rate)` | Token refill rate per second | `10` |
| `WithCapacity(cap)` | Maximum burst capacity | `20` |
| `WithLimit(limit)` | Maximum requests per window | `100` |
| `WithWindow(win)` | Window duration | `1 * time.Minute` |
| `WithStore(store)` | Storage backend | `store.NewMemory()` |
| `WithFailStrategy(strategy)`| Behavior on store error (`FailOpen` / `FailClosed`) | `FailOpen` |
| `WithKeyExtractor(fn)` | Custom HTTP key extractor | `KeyByIP` |

---

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.