package algorithms

import (
	"context"
	"testing"
	"time"

	"github.com/bshome19/rateshield/store"
)

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

func BenchmarkSlidingWindowLog_Memory(b *testing.B) {
	memStore := store.NewMemory()
	defer memStore.Close()

	sw, err := NewSlidingWindow(SlidingWindowConfig{
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
			_, _ = sw.Allow(ctx, "bench_key")
		}
	})
}
