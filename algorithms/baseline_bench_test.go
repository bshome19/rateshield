package algorithms

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bshome19/rateshield/store"
)

// Single-key benchmarks (measuring single shard contention under concurrency)

func BenchmarkTokenBucket_Memory(b *testing.B) {
	memStore := store.NewMemory()
	defer memStore.Close()

	tb, err := NewTokenBucket(TokenBucketConfig{
		Rate:     100000000,
		Capacity: 100000000,
		Store:    memStore,
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = tb.Allow(ctx, "bench_key")
		}
	})
}

func BenchmarkTokenBucket_Memory_ZeroAlloc(b *testing.B) {
	memStore := store.NewMemory()
	defer memStore.Close()

	tb, err := NewTokenBucket(TokenBucketConfig{
		Rate:     100000000,
		Capacity: 100000000,
		Store:    memStore,
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = tb.AllowFast(ctx, "bench_key")
		}
	})
}

func BenchmarkTokenBucket_Memory_MultiKey_ZeroAlloc(b *testing.B) {
	memStore := store.NewMemory()
	defer memStore.Close()

	tb, err := NewTokenBucket(TokenBucketConfig{
		Rate:     100000000,
		Capacity: 100000000,
		Store:    memStore,
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	var counter int64
	keys := make([]string, 64)
	for i := 0; i < 64; i++ {
		keys[i] = "bench_key_" + strconv.Itoa(i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := atomic.AddInt64(&counter, 1) & 63
			_, _ = tb.AllowFast(ctx, keys[idx])
		}
	})
}

func BenchmarkFixedWindow_Memory(b *testing.B) {
	memStore := store.NewMemory()
	defer memStore.Close()

	fw, err := NewFixedWindow(FixedWindowConfig{
		Limit:  100000000,
		Window: time.Minute,
		Store:  memStore,
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = fw.Allow(ctx, "bench_key")
		}
	})
}

func BenchmarkFixedWindow_Memory_ZeroAlloc(b *testing.B) {
	memStore := store.NewMemory()
	defer memStore.Close()

	fw, err := NewFixedWindow(FixedWindowConfig{
		Limit:  100000000,
		Window: time.Minute,
		Store:  memStore,
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = fw.AllowFast(ctx, "bench_key")
		}
	})
}

func BenchmarkSlidingWindowCounter_Memory(b *testing.B) {
	memStore := store.NewMemory()
	defer memStore.Close()

	swc, err := NewSlidingWindowCounter(SlidingWindowCounterConfig{
		Limit:  100000000,
		Window: time.Minute,
		Store:  memStore,
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = swc.Allow(ctx, "bench_key")
		}
	})
}

func BenchmarkSlidingWindowCounter_Memory_ZeroAlloc(b *testing.B) {
	memStore := store.NewMemory()
	defer memStore.Close()

	swc, err := NewSlidingWindowCounter(SlidingWindowCounterConfig{
		Limit:  100000000,
		Window: time.Minute,
		Store:  memStore,
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = swc.AllowFast(ctx, "bench_key")
		}
	})
}

func BenchmarkSlidingWindowLog_Memory(b *testing.B) {
	memStore := store.NewMemory()
	defer memStore.Close()

	sw, err := NewSlidingWindow(SlidingWindowConfig{
		Limit:  10000,
		Window: 5 * time.Millisecond,
		Store:  memStore,
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = sw.Allow(ctx, "bench_key")
		}
	})
}

func BenchmarkSlidingWindowLog_Memory_ZeroAlloc(b *testing.B) {
	memStore := store.NewMemory()
	defer memStore.Close()

	sw, err := NewSlidingWindow(SlidingWindowConfig{
		Limit:  10000,
		Window: 5 * time.Millisecond,
		Store:  memStore,
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = sw.AllowFast(ctx, "bench_key")
		}
	})
}

func BenchmarkTokenBucket_Memory_MultiKey(b *testing.B) {
	memStore := store.NewMemory()
	defer memStore.Close()

	tb, err := NewTokenBucket(TokenBucketConfig{
		Rate:     100000000,
		Capacity: 100000000,
		Store:    memStore,
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	var counter int64
	keys := make([]string, 64)
	for i := 0; i < 64; i++ {
		keys[i] = "bench_key_" + strconv.Itoa(i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := atomic.AddInt64(&counter, 1) & 63
			_, _ = tb.Allow(ctx, keys[idx])
		}
	})
}

func BenchmarkFixedWindow_Memory_MultiKey_ZeroAlloc(b *testing.B) {
	memStore := store.NewMemory()
	defer memStore.Close()

	fw, err := NewFixedWindow(FixedWindowConfig{
		Limit:  100000000,
		Window: time.Minute,
		Store:  memStore,
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	var counter int64
	keys := make([]string, 64)
	for i := 0; i < 64; i++ {
		keys[i] = "bench_key_" + strconv.Itoa(i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := atomic.AddInt64(&counter, 1) & 63
			_, _ = fw.AllowFast(ctx, keys[idx])
		}
	})
}

func BenchmarkSlidingWindowCounter_Memory_MultiKey_ZeroAlloc(b *testing.B) {
	memStore := store.NewMemory()
	defer memStore.Close()

	swc, err := NewSlidingWindowCounter(SlidingWindowCounterConfig{
		Limit:  100000000,
		Window: time.Minute,
		Store:  memStore,
	})
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	var counter int64
	keys := make([]string, 64)
	for i := 0; i < 64; i++ {
		keys[i] = "bench_key_" + strconv.Itoa(i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := atomic.AddInt64(&counter, 1) & 63
			_, _ = swc.AllowFast(ctx, keys[idx])
		}
	})
}
