package circuit

import (
	"testing"
	"time"
)

func TestBreaker_Lifecycle(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Cooldown:         50 * time.Millisecond,
	})

	if cb.State() != StateClosed {
		t.Fatalf("expected closed, got %v", cb.State())
	}
	if !cb.Allow() {
		t.Fatal("expected allow on closed breaker")
	}

	// 2 failures should not trip
	cb.ReportFailure()
	cb.ReportFailure()
	if cb.State() != StateClosed {
		t.Fatalf("expected still closed after 2 failures")
	}

	// 3rd failure trips the breaker to Open
	cb.ReportFailure()
	if cb.State() != StateOpen {
		t.Fatalf("expected open after 3 failures, got %v", cb.State())
	}
	if cb.Allow() {
		t.Fatal("expected allow to be false when open")
	}

	// Wait for cooldown
	time.Sleep(60 * time.Millisecond)

	// Now should transition to HalfOpen and allow probe
	if !cb.Allow() {
		t.Fatal("expected allow on half-open breaker")
	}
	if cb.State() != StateHalfOpen {
		t.Fatalf("expected half-open after cooldown, got %v", cb.State())
	}

	// First success in half-open
	cb.ReportSuccess()
	if cb.State() != StateHalfOpen {
		t.Fatalf("expected half-open after 1 success, got %v", cb.State())
	}

	// Second success should close the breaker
	cb.ReportSuccess()
	if cb.State() != StateClosed {
		t.Fatalf("expected closed after 2 successes, got %v", cb.State())
	}

	// Reset test
	cb.ReportFailure()
	cb.ReportFailure()
	cb.ReportFailure()
	if cb.State() != StateOpen {
		t.Fatalf("expected open")
	}
	cb.Reset()
	if cb.State() != StateClosed {
		t.Fatalf("expected closed after reset")
	}
}

func TestBreaker_HalfOpenFailure(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Cooldown:         20 * time.Millisecond,
	})

	cb.ReportFailure()
	if cb.State() != StateOpen {
		t.Fatal("expected open")
	}

	time.Sleep(30 * time.Millisecond)
	if !cb.Allow() {
		t.Fatal("expected allow")
	}

	// Probe fails -> trips immediately back to open
	cb.ReportFailure()
	if cb.State() != StateOpen {
		t.Fatalf("expected open after failed probe, got %v", cb.State())
	}
}
