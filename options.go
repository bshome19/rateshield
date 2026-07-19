package rateshield

import (
	"time"

	"github.com/bshome19/rateshield/store"
)

// Algorithm represents the rate limiting algorithm type.
type Algorithm int

const (
	// TokenBucketAlgorithm uses the token bucket algorithm.
	TokenBucketAlgorithm Algorithm = iota

	// SlidingWindowAlgorithm uses the sliding window log algorithm.
	SlidingWindowAlgorithm

	// FixedWindowAlgorithm uses the fixed window algorithm.
	FixedWindowAlgorithm

	// SlidingWindowCounterAlgorithm uses the high-performance O(1) memory sliding window counter algorithm.
	SlidingWindowCounterAlgorithm
)

// Options holds the configuration for a rate limiter.
type Options struct {
	// Algorithm is the rate limiting algorithm to use.
	Algorithm Algorithm

	// Rate is the number of requests allowed per second (for token bucket).
	Rate float64

	// Capacity is the maximum burst size (for token bucket).
	Capacity int64

	// Limit is the maximum requests per window (for window-based algorithms).
	Limit int64

	// Window is the time window for window-based algorithms.
	Window time.Duration

	// Store is the storage backend.
	Store store.Store

	// TTL is how long to keep state in storage.
	TTL time.Duration

	// KeyExtractor extracts the rate limit key from requests.
	KeyExtractor KeyExtractor

	// FailStrategy determines behavior on store error (default: FailOpen).
	FailStrategy FailStrategy

	// Metrics holds an optional metrics collector callback interface.
	Metrics MetricsCollector
}

// Option is a function that configures Options.
type Option func(*Options)

// WithAlgorithm sets the rate limiting algorithm.
func WithAlgorithm(alg Algorithm) Option {
	return func(o *Options) {
		o.Algorithm = alg
	}
}

// WithRate sets the token generation rate (tokens per second).
func WithRate(rate float64) Option {
	return func(o *Options) {
		o.Rate = rate
	}
}

// WithCapacity sets the maximum burst size.
func WithCapacity(capacity int64) Option {
	return func(o *Options) {
		o.Capacity = capacity
	}
}

// WithLimit sets the maximum requests per window.
func WithLimit(limit int64) Option {
	return func(o *Options) {
		o.Limit = limit
	}
}

// WithWindow sets the time window for window-based algorithms.
func WithWindow(window time.Duration) Option {
	return func(o *Options) {
		o.Window = window
	}
}

// WithStore sets the storage backend.
func WithStore(s store.Store) Option {
	return func(o *Options) {
		o.Store = s
	}
}

// WithTTL sets how long to keep state in storage.
func WithTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.TTL = ttl
	}
}

// WithKeyExtractor sets the key extractor function.
func WithKeyExtractor(ke KeyExtractor) Option {
	return func(o *Options) {
		o.KeyExtractor = ke
	}
}

// WithFailStrategy sets the failure strategy on store errors.
func WithFailStrategy(strategy FailStrategy) Option {
	return func(o *Options) {
		o.FailStrategy = strategy
	}
}

// WithMetrics sets the metrics collector instance.
func WithMetrics(m MetricsCollector) Option {
	return func(o *Options) {
		o.Metrics = m
	}
}

// DefaultOptions returns the default options.
func DefaultOptions() *Options {
	return &Options{
		Algorithm:    TokenBucketAlgorithm,
		Rate:         10,
		Capacity:     20,
		Limit:        100,
		Window:       time.Minute,
		TTL:          time.Hour,
		FailStrategy: FailOpen,
	}
}

// Apply applies the given options to the Options struct.
func (o *Options) Apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}
