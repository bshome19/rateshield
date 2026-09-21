// Package algorithms provides rate limiting algorithm implementations.
package algorithms

import (
	"github.com/bshome19/rateshield"
)

// TokenBucket implements the token bucket rate limiting algorithm.
type TokenBucket = rateshield.TokenBucket

// TokenBucketConfig holds the configuration for a TokenBucket limiter.
type TokenBucketConfig = rateshield.TokenBucketConfig

// NewTokenBucket creates a new TokenBucket rate limiter.
func NewTokenBucket(cfg TokenBucketConfig) (*TokenBucket, error) {
	return rateshield.NewTokenBucket(cfg)
}
