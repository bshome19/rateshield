package algorithms

import (
	"github.com/bshome19/rateshield"
)

// SlidingWindowCounter implements the sliding window counter rate limiting algorithm.
type SlidingWindowCounter = rateshield.SlidingWindowCounter

// SlidingWindowCounterConfig holds the configuration for a SlidingWindowCounter limiter.
type SlidingWindowCounterConfig = rateshield.SlidingWindowCounterConfig

// NewSlidingWindowCounter creates a new SlidingWindowCounter rate limiter.
func NewSlidingWindowCounter(cfg SlidingWindowCounterConfig) (*SlidingWindowCounter, error) {
	return rateshield.NewSlidingWindowCounter(cfg)
}
