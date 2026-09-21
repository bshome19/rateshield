package algorithms

import (
	"github.com/bshome19/rateshield"
)

// SlidingWindow implements the sliding window log rate limiting algorithm.
type SlidingWindow = rateshield.SlidingWindow

// SlidingWindowConfig holds the configuration for a SlidingWindow limiter.
type SlidingWindowConfig = rateshield.SlidingWindowConfig

// NewSlidingWindow creates a new SlidingWindow rate limiter.
func NewSlidingWindow(cfg SlidingWindowConfig) (*SlidingWindow, error) {
	return rateshield.NewSlidingWindow(cfg)
}
