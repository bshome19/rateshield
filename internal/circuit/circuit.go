// Package circuit provides an atomic, low-overhead circuit breaker for storage failover.
package circuit

import (
	"sync"
	"sync/atomic"
	"time"
)

// State represents the circuit breaker state.
type State int32

const (
	// StateClosed allows traffic to the primary store (normal operation).
	StateClosed State = iota

	// StateOpen diverts traffic away from the primary store immediately (failing).
	StateOpen

	// StateHalfOpen allows a limited probe to test if the primary store has recovered.
	StateHalfOpen
)

// Breaker implements a fast, lock-free on hot-path circuit breaker.
type Breaker struct {
	state          int32 // State atomic
	failureCount   int64
	successCount   int64
	openUntil      int64 // unix nano

	failureThreshold int64
	successThreshold int64
	cooldown         time.Duration

	mu sync.Mutex
}

// Config defines options for creating a Breaker.
type Config struct {
	// FailureThreshold is the number of consecutive errors to trip the circuit (default: 5).
	FailureThreshold int64

	// SuccessThreshold is the number of consecutive successes to close the circuit (default: 2).
	SuccessThreshold int64

	// Cooldown is the duration to remain open before probing in half-open state (default: 5s).
	Cooldown time.Duration
}

// New creates a new Breaker with the specified configuration.
func New(cfg Config) *Breaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = 2
	}
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = 5 * time.Second
	}

	return &Breaker{
		failureThreshold: cfg.FailureThreshold,
		successThreshold: cfg.SuccessThreshold,
		cooldown:         cfg.Cooldown,
	}
}

// Allow returns true if the primary store should be attempted.
func (b *Breaker) Allow() bool {
	st := State(atomic.LoadInt32(&b.state))
	if st == StateClosed {
		return true
	}

	now := time.Now().UnixNano()
	if st == StateOpen {
		openUntil := atomic.LoadInt64(&b.openUntil)
		if now < openUntil {
			return false
		}

		// Cooldown elapsed, attempt transition to half-open
		if atomic.CompareAndSwapInt32(&b.state, int32(StateOpen), int32(StateHalfOpen)) {
			atomic.StoreInt64(&b.successCount, 0)
			return true
		}
		return State(atomic.LoadInt32(&b.state)) == StateHalfOpen
	}

	// StateHalfOpen: allow probing
	return true
}

// ReportSuccess records a successful call to the primary store.
func (b *Breaker) ReportSuccess() {
	st := State(atomic.LoadInt32(&b.state))
	if st == StateClosed {
		atomic.StoreInt64(&b.failureCount, 0)
		return
	}

	if st == StateHalfOpen {
		succ := atomic.AddInt64(&b.successCount, 1)
		if succ >= b.successThreshold {
			b.mu.Lock()
			if State(b.state) == StateHalfOpen {
				atomic.StoreInt64(&b.failureCount, 0)
				atomic.StoreInt64(&b.successCount, 0)
				atomic.StoreInt32(&b.state, int32(StateClosed))
			}
			b.mu.Unlock()
		}
	}
}

// ReportFailure records a failed call to the primary store.
func (b *Breaker) ReportFailure() {
	st := State(atomic.LoadInt32(&b.state))

	if st == StateHalfOpen {
		// Probe failed, immediately trip back to open with full cooldown
		b.trip()
		return
	}

	if st == StateClosed {
		fails := atomic.AddInt64(&b.failureCount, 1)
		if fails >= b.failureThreshold {
			b.trip()
		}
	}
}

func (b *Breaker) trip() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now().UnixNano()
	atomic.StoreInt64(&b.openUntil, now+b.cooldown.Nanoseconds())
	atomic.StoreInt64(&b.failureCount, 0)
	atomic.StoreInt64(&b.successCount, 0)
	atomic.StoreInt32(&b.state, int32(StateOpen))
}

// State returns the current State of the circuit breaker.
func (b *Breaker) State() State {
	return State(atomic.LoadInt32(&b.state))
}

// Reset resets the circuit breaker to closed state.
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	atomic.StoreInt32(&b.state, int32(StateClosed))
	atomic.StoreInt64(&b.failureCount, 0)
	atomic.StoreInt64(&b.successCount, 0)
	atomic.StoreInt64(&b.openUntil, 0)
}
