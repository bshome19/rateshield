package store

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewMemory(t *testing.T) {
	m := NewMemory()
	if m == nil {
		t.Fatal("NewMemory returned nil")
	}
	defer m.Close()

	if m.shards[0] == nil {
		t.Error("shards not initialized")
	}
	if m.closeCh == nil {
		t.Error("closeCh not initialized")
	}
}

func TestMemory_GetNewKey(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()

	// Get non-existent key should return empty state
	state, err := m.Get(ctx, "non-existent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !state.LastUpdate.IsZero() {
		t.Error("Expected zero LastUpdate for new key")
	}
	if state.Tokens != 0 {
		t.Errorf("Expected 0 tokens, got %f", state.Tokens)
	}
	if state.Count != 0 {
		t.Errorf("Expected 0 count, got %d", state.Count)
	}
	if state.Requests == nil {
		t.Error("Requests should be initialized")
	}
}

func TestMemory_SetAndGet(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()
	key := "test-key"

	now := time.Now()
	newState := &State{
		Tokens:      5.5,
		LastUpdate:  now,
		Count:       10,
		WindowStart: now.Add(-time.Minute),
		Requests:    []time.Time{now},
	}
	err := m.Set(ctx, key, newState, time.Hour)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get the value back
	state, err := m.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if state.Tokens != 5.5 {
		t.Errorf("Expected tokens 5.5, got %f", state.Tokens)
	}
	if state.Count != 10 {
		t.Errorf("Expected count 10, got %d", state.Count)
	}
	if len(state.Requests) != 1 {
		t.Errorf("Expected 1 request, got %d", len(state.Requests))
	}
}

func TestMemory_SetCopiesData(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()
	key := "test-key"

	// Create state with requests
	now := time.Now()
	requests := []time.Time{now, now.Add(time.Second)}
	state := &State{
		Tokens:   5,
		Requests: requests,
	}

	err := m.Set(ctx, key, state, time.Hour)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Modify original
	requests[0] = time.Time{}
	state.Tokens = 999

	// Get should return unmodified data
	retrieved, _ := m.Get(ctx, key)
	if retrieved.Requests[0].IsZero() {
		t.Error("Store should have copied the requests slice")
	}
	if retrieved.Tokens == 999 {
		t.Error("Store should have copied the tokens value")
	}
}

func TestMemory_GetCopiesData(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()
	key := "test-key"

	now := time.Now()
	state := &State{
		Tokens:   5,
		Requests: []time.Time{now},
	}
	_ = m.Set(ctx, key, state, time.Hour)

	// Get and modify
	retrieved, _ := m.Get(ctx, key)
	retrieved.Tokens = 999
	if len(retrieved.Requests) > 0 {
		retrieved.Requests[0] = time.Time{}
	}

	// Get again - should be unchanged
	retrieved2, _ := m.Get(ctx, key)
	if retrieved2.Tokens == 999 {
		t.Error("Get should return a copy (tokens)")
	}
	if len(retrieved2.Requests) > 0 && retrieved2.Requests[0].IsZero() {
		t.Error("Get should return a copy (requests)")
	}
}

func TestMemory_Increment(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()
	key := "test-key"

	// First increment
	count, err := m.Increment(ctx, key, time.Hour)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// Second increment
	count, err = m.Increment(ctx, key, time.Hour)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}

	// Third increment
	count, err = m.Increment(ctx, key, time.Hour)
	if err != nil {
		t.Fatalf("Increment failed: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}

func TestMemory_IncrementMultipleKeys(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()

	// Increment different keys
	count1, _ := m.Increment(ctx, "key1", time.Hour)
	count2, _ := m.Increment(ctx, "key2", time.Hour)
	count1b, _ := m.Increment(ctx, "key1", time.Hour)

	if count1 != 1 {
		t.Errorf("Expected key1 count 1, got %d", count1)
	}
	if count2 != 1 {
		t.Errorf("Expected key2 count 1, got %d", count2)
	}
	if count1b != 2 {
		t.Errorf("Expected key1 second increment to be 2, got %d", count1b)
	}
}

func TestMemory_Reset(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()
	key := "test-key"

	// Set a value
	state := &State{Tokens: 10, LastUpdate: time.Now()}
	_ = m.Set(ctx, key, state, time.Hour)

	// Verify it exists
	retrieved, _ := m.Get(ctx, key)
	if retrieved.LastUpdate.IsZero() {
		t.Error("State should exist before reset")
	}

	// Reset
	err := m.Reset(ctx, key)
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	// Should be gone
	retrieved, _ = m.Get(ctx, key)
	if !retrieved.LastUpdate.IsZero() {
		t.Error("Expected zero state after reset")
	}
}

func TestMemory_ResetNonExistent(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()

	// Reset non-existent key should not error
	err := m.Reset(ctx, "non-existent")
	if err != nil {
		t.Errorf("Reset of non-existent key should not error: %v", err)
	}
}

func TestMemory_TTLExpiry(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()
	key := "test-key"

	// Set with short TTL
	state := &State{Tokens: 10, LastUpdate: time.Now()}
	_ = m.Set(ctx, key, state, 50*time.Millisecond)

	// Should exist initially
	retrieved, _ := m.Get(ctx, key)
	if retrieved.LastUpdate.IsZero() {
		t.Error("Expected state to exist")
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	retrieved, _ = m.Get(ctx, key)
	if !retrieved.LastUpdate.IsZero() {
		t.Error("Expected state to be expired")
	}
}

func TestMemory_TTLRefresh(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()
	key := "test-key"

	// Set with short TTL
	state := &State{Tokens: 10, LastUpdate: time.Now()}
	_ = m.Set(ctx, key, state, 100*time.Millisecond)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Update with new TTL
	state.Tokens = 20
	_ = m.Set(ctx, key, state, 100*time.Millisecond)

	// Wait past original expiry
	time.Sleep(70 * time.Millisecond)

	// Should still exist (TTL was refreshed)
	retrieved, _ := m.Get(ctx, key)
	if retrieved.LastUpdate.IsZero() {
		t.Error("Expected state to exist (TTL was refreshed)")
	}
	if retrieved.Tokens != 20 {
		t.Errorf("Expected tokens 20, got %f", retrieved.Tokens)
	}
}

func TestMemory_ConcurrentIncrement(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()
	key := "concurrent-key"

	var wg sync.WaitGroup

	// Concurrent increments
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = m.Increment(ctx, key, time.Hour)
		}()
	}

	wg.Wait()

	// Verify count
	state, _ := m.Get(ctx, key)
	if state.Count != 100 {
		t.Errorf("Expected count 100, got %d", state.Count)
	}
}

func TestMemory_ConcurrentReadWrite(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()

	var wg sync.WaitGroup

	// Writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "key"
			state := &State{Tokens: float64(id), LastUpdate: time.Now()}
			_ = m.Set(ctx, key, state, time.Hour)
		}(i)
	}

	// Readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = m.Get(ctx, "key")
		}()
	}

	wg.Wait()
	// If we get here without a race condition, test passes
}

func TestMemory_ConcurrentDifferentKeys(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()

	var wg sync.WaitGroup

	// Concurrent operations on different keys
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := string(rune('a' + id%26))
			_, _ = m.Increment(ctx, key, time.Hour)
			_, _ = m.Get(ctx, key)
		}(i)
	}

	wg.Wait()
}

func TestMemory_Close(t *testing.T) {
	m := NewMemory()

	err := m.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestMemory_StateFields(t *testing.T) {
	// Test State struct fields
	now := time.Now()
	state := &State{
		Tokens:      10.5,
		LastUpdate:  now,
		Count:       5,
		WindowStart: now.Add(-time.Minute),
		Requests:    []time.Time{now, now.Add(time.Second)},
	}

	if state.Tokens != 10.5 {
		t.Errorf("Expected Tokens 10.5, got %f", state.Tokens)
	}
	if !state.LastUpdate.Equal(now) {
		t.Error("LastUpdate mismatch")
	}
	if state.Count != 5 {
		t.Errorf("Expected Count 5, got %d", state.Count)
	}
	if len(state.Requests) != 2 {
		t.Errorf("Expected 2 requests, got %d", len(state.Requests))
	}
}

func TestMemory_EmptyRequests(t *testing.T) {
	m := NewMemory()
	defer m.Close()

	ctx := context.Background()
	key := "test-key"

	// Set state with nil requests
	state := &State{
		Tokens: 5,
	}
	err := m.Set(ctx, key, state, time.Hour)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get should handle nil requests
	retrieved, err := m.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.Tokens != 5 {
		t.Errorf("Expected tokens 5, got %f", retrieved.Tokens)
	}
}
