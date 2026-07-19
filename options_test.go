package rateshield

import (
	"net/http"
	"testing"
	"time"

	"github.com/bshome19/rateshield/store"
)

func TestOptions_Apply(t *testing.T) {
	mem := store.NewMemory()
	defer mem.Close()

	opts := DefaultOptions()
	opts.Apply(
		WithAlgorithm(SlidingWindowCounterAlgorithm),
		WithRate(50),
		WithCapacity(100),
		WithLimit(500),
		WithWindow(5*time.Minute),
		WithStore(mem),
		WithTTL(10*time.Minute),
		WithFailStrategy(FailClosed),
		WithKeyExtractor(func(r *http.Request) string { return "custom" }),
		WithMetrics(nil),
	)

	if opts.Algorithm != SlidingWindowCounterAlgorithm {
		t.Errorf("expected SlidingWindowCounterAlgorithm, got %v", opts.Algorithm)
	}
	if opts.Rate != 50 {
		t.Errorf("expected rate 50, got %f", opts.Rate)
	}
	if opts.Capacity != 100 {
		t.Errorf("expected capacity 100, got %d", opts.Capacity)
	}
	if opts.Limit != 500 {
		t.Errorf("expected limit 500, got %d", opts.Limit)
	}
	if opts.Window != 5*time.Minute {
		t.Errorf("expected window 5m, got %v", opts.Window)
	}
	if opts.Store != mem {
		t.Errorf("store mismatch")
	}
	if opts.TTL != 10*time.Minute {
		t.Errorf("expected TTL 10m, got %v", opts.TTL)
	}
	if opts.FailStrategy != FailClosed {
		t.Errorf("expected FailClosed strategy")
	}
}
