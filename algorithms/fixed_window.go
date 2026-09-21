package algorithms

import (
	"github.com/bshome19/rateshield"
)

// FixedWindow implements the fixed window counter rate limiting algorithm.
type FixedWindow = rateshield.FixedWindow

// FixedWindowConfig holds the configuration for a FixedWindow limiter.
type FixedWindowConfig = rateshield.FixedWindowConfig

// NewFixedWindow creates a new FixedWindow rate limiter.
func NewFixedWindow(cfg FixedWindowConfig) (*FixedWindow, error) {
	return rateshield.NewFixedWindow(cfg)
}
