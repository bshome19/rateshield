package rateshield

import "errors"

var (
	// ErrRateLimitExceeded is returned when the rate limit is exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrInvalidKey is returned when an empty or invalid key is provided.
	ErrInvalidKey = errors.New("invalid rate limit key")

	// ErrStoreUnavailable is returned when the storage backend is unavailable.
	ErrStoreUnavailable = errors.New("rate limit store unavailable")

	// ErrInvalidConfig is returned when the limiter configuration is invalid.
	ErrInvalidConfig = errors.New("invalid rate limiter configuration")
)
