package rateshield

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestResult_Allowed(t *testing.T) {
	now := time.Now()
	result := &Result{
		Allowed:    true,
		Limit:      100,
		Remaining:  95,
		ResetAt:    now.Add(time.Minute),
		RetryAfter: 0,
	}

	if !result.Allowed {
		t.Error("Expected Allowed to be true")
	}

	if result.Limit != 100 {
		t.Errorf("Expected Limit 100, got %d", result.Limit)
	}

	if result.Remaining != 95 {
		t.Errorf("Expected Remaining 95, got %d", result.Remaining)
	}

	if result.ResetAt.IsZero() {
		t.Error("Expected ResetAt to be set")
	}

	if result.RetryAfter != 0 {
		t.Errorf("Expected RetryAfter 0, got %v", result.RetryAfter)
	}
}

func TestResult_Denied(t *testing.T) {
	now := time.Now()
	result := &Result{
		Allowed:    false,
		Limit:      100,
		Remaining:  0,
		ResetAt:    now.Add(time.Minute),
		RetryAfter: 30 * time.Second,
	}

	if result.Allowed {
		t.Error("Expected Allowed to be false")
	}

	if result.Remaining != 0 {
		t.Errorf("Expected Remaining 0, got %d", result.Remaining)
	}

	if result.RetryAfter != 30*time.Second {
		t.Errorf("Expected RetryAfter 30s, got %v", result.RetryAfter)
	}
}

func TestResult_ZeroValues(t *testing.T) {
	result := &Result{}

	if result.Allowed {
		t.Error("Zero value Allowed should be false")
	}

	if result.Limit != 0 {
		t.Errorf("Zero value Limit should be 0, got %d", result.Limit)
	}

	if result.Remaining != 0 {
		t.Errorf("Zero value Remaining should be 0, got %d", result.Remaining)
	}

	if !result.ResetAt.IsZero() {
		t.Error("Zero value ResetAt should be zero time")
	}

	if result.RetryAfter != 0 {
		t.Errorf("Zero value RetryAfter should be 0, got %v", result.RetryAfter)
	}
}

// mockLimiter is a mock implementation of the Limiter interface for testing
type mockLimiter struct {
	allowResult *Result
	allowErr    error
	resetErr    error
	allowCalls  int
	resetCalls  int
	lastKey     string
	lastN       int64
}

func newMockLimiter(result *Result, err error) *mockLimiter {
	return &mockLimiter{
		allowResult: result,
		allowErr:    err,
	}
}

func (m *mockLimiter) Allow(ctx context.Context, key string) (*Result, error) {
	m.allowCalls++
	m.lastKey = key
	m.lastN = 1
	return m.allowResult, m.allowErr
}

func (m *mockLimiter) AllowN(ctx context.Context, key string, n int64) (*Result, error) {
	m.allowCalls++
	m.lastKey = key
	m.lastN = n
	return m.allowResult, m.allowErr
}

func (m *mockLimiter) Reset(ctx context.Context, key string) error {
	m.resetCalls++
	m.lastKey = key
	return m.resetErr
}

func TestLimiterInterface(t *testing.T) {
	// Verify mockLimiter implements Limiter interface
	var _ Limiter = (*mockLimiter)(nil)
}

func TestMockLimiter_Allow(t *testing.T) {
	expectedResult := &Result{
		Allowed:   true,
		Limit:     100,
		Remaining: 99,
	}

	mock := newMockLimiter(expectedResult, nil)
	ctx := context.Background()

	result, err := mock.Allow(ctx, "test-key")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != expectedResult {
		t.Error("Result does not match expected")
	}

	if mock.allowCalls != 1 {
		t.Errorf("Expected 1 Allow call, got %d", mock.allowCalls)
	}

	if mock.lastKey != "test-key" {
		t.Errorf("Expected key 'test-key', got '%s'", mock.lastKey)
	}

	if mock.lastN != 1 {
		t.Errorf("Expected n=1, got %d", mock.lastN)
	}
}

func TestMockLimiter_AllowError(t *testing.T) {
	expectedErr := errors.New("storage error")
	mock := newMockLimiter(nil, expectedErr)
	ctx := context.Background()

	result, err := mock.Allow(ctx, "test-key")

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	if result != nil {
		t.Error("Expected nil result on error")
	}
}

func TestMockLimiter_AllowN(t *testing.T) {
	expectedResult := &Result{
		Allowed:   true,
		Limit:     100,
		Remaining: 90,
	}

	mock := newMockLimiter(expectedResult, nil)
	ctx := context.Background()

	result, err := mock.AllowN(ctx, "test-key", 10)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != expectedResult {
		t.Error("Result does not match expected")
	}

	if mock.lastN != 10 {
		t.Errorf("Expected n=10, got %d", mock.lastN)
	}
}

func TestMockLimiter_Reset(t *testing.T) {
	mock := &mockLimiter{}
	ctx := context.Background()

	err := mock.Reset(ctx, "test-key")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if mock.resetCalls != 1 {
		t.Errorf("Expected 1 Reset call, got %d", mock.resetCalls)
	}

	if mock.lastKey != "test-key" {
		t.Errorf("Expected key 'test-key', got '%s'", mock.lastKey)
	}
}

func TestMockLimiter_ResetError(t *testing.T) {
	expectedErr := errors.New("reset error")
	mock := &mockLimiter{resetErr: expectedErr}
	ctx := context.Background()

	err := mock.Reset(ctx, "test-key")

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestLimiterInterface_Compatibility(t *testing.T) {
	// Test that we can use Limiter interface polymorphically
	limiter := Limiter(newMockLimiter(&Result{Allowed: true}, nil))
	ctx := context.Background()

	result, err := limiter.Allow(ctx, "key")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("Expected request to be allowed")
	}
}

func TestResult_ResetAtInFuture(t *testing.T) {
	now := time.Now()
	resetAt := now.Add(5 * time.Minute)

	result := &Result{
		Allowed: true,
		ResetAt: resetAt,
	}

	if result.ResetAt.Before(now) {
		t.Error("ResetAt should be in the future")
	}

	duration := result.ResetAt.Sub(now)
	if duration < 4*time.Minute || duration > 6*time.Minute {
		t.Errorf("ResetAt duration unexpected: %v", duration)
	}
}

func TestResult_RetryAfterDuration(t *testing.T) {
	tests := []struct {
		name       string
		retryAfter time.Duration
	}{
		{"zero", 0},
		{"one second", time.Second},
		{"30 seconds", 30 * time.Second},
		{"one minute", time.Minute},
		{"sub-second", 500 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{
				Allowed:    false,
				RetryAfter: tt.retryAfter,
			}

			if result.RetryAfter != tt.retryAfter {
				t.Errorf("Expected RetryAfter %v, got %v", tt.retryAfter, result.RetryAfter)
			}
		})
	}
}

// Test errors
func TestErrors(t *testing.T) {
	if ErrRateLimitExceeded == nil {
		t.Error("ErrRateLimitExceeded should not be nil")
	}
	if ErrRateLimitExceeded.Error() == "" {
		t.Error("ErrRateLimitExceeded should have a message")
	}

	if ErrInvalidKey == nil {
		t.Error("ErrInvalidKey should not be nil")
	}
	if ErrInvalidKey.Error() == "" {
		t.Error("ErrInvalidKey should have a message")
	}

	if ErrStoreUnavailable == nil {
		t.Error("ErrStoreUnavailable should not be nil")
	}
	if ErrStoreUnavailable.Error() == "" {
		t.Error("ErrStoreUnavailable should have a message")
	}

	if ErrInvalidConfig == nil {
		t.Error("ErrInvalidConfig should not be nil")
	}
	if ErrInvalidConfig.Error() == "" {
		t.Error("ErrInvalidConfig should have a message")
	}
}

func TestErrors_Unique(t *testing.T) {
	// Ensure all errors are unique
	errs := []error{
		ErrRateLimitExceeded,
		ErrInvalidKey,
		ErrStoreUnavailable,
		ErrInvalidConfig,
	}

	for i := 0; i < len(errs); i++ {
		for j := i + 1; j < len(errs); j++ {
			if errs[i] == errs[j] {
				t.Errorf("Errors at index %d and %d should be different", i, j)
			}
			if errs[i].Error() == errs[j].Error() {
				t.Errorf("Error messages at index %d and %d should be different", i, j)
			}
		}
	}
}

func TestErrors_Is(t *testing.T) {
	// Test that errors can be compared with errors.Is
	if !errors.Is(ErrRateLimitExceeded, ErrRateLimitExceeded) {
		t.Error("ErrRateLimitExceeded should match itself")
	}

	if errors.Is(ErrRateLimitExceeded, ErrInvalidKey) {
		t.Error("ErrRateLimitExceeded should not match ErrInvalidKey")
	}
}
