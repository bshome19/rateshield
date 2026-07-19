package store

import (
	"context"
	"sync"
	"time"
)

const (
	numShards = 64
	offset64  = 14695981039346656037
	prime64   = 1099511628211
)

func fnv1a(key string) uint64 {
	var hash uint64 = offset64
	for i := 0; i < len(key); i++ {
		hash ^= uint64(key[i])
		hash *= prime64
	}
	return hash
}

// Memory is a high-performance, sharded in-memory implementation of Store.
type Memory struct {
	shards  [numShards]*shard
	closeCh chan struct{}
}

type shard struct {
	mu   sync.RWMutex
	data map[string]*memEntry
}

type memEntry struct {
	tokens    float64
	lastUpd   time.Time
	count     int64
	prevCount int64
	windowNum int64
	requests  []time.Time
	expiresAt time.Time
}

// NewMemory creates a new sharded in-memory store.
func NewMemory() *Memory {
	m := &Memory{
		closeCh: make(chan struct{}),
	}
	for i := 0; i < numShards; i++ {
		m.shards[i] = &shard{
			data: make(map[string]*memEntry),
		}
	}

	go m.cleanup()

	return m
}

func (m *Memory) getShard(key string) *shard {
	idx := fnv1a(key) % numShards
	return m.shards[idx]
}

// Get retrieves the state for a key.
func (m *Memory) Get(ctx context.Context, key string) (*State, error) {
	s := m.getShard(key)
	s.mu.RLock()
	defer s.mu.RUnlock()

	if e, ok := s.data[key]; ok {
		if time.Now().Before(e.expiresAt) {
			reqsCopy := make([]time.Time, len(e.requests))
			copy(reqsCopy, e.requests)

			return &State{
				Tokens:      e.tokens,
				LastUpdate:  e.lastUpd,
				Count:       e.count,
				WindowStart: time.Unix(0, 0),
				Requests:    reqsCopy,
			}, nil
		}
	}

	return &State{
		Tokens:   0,
		Requests: make([]time.Time, 0),
	}, nil
}

// Set saves the state for a key with a TTL.
func (m *Memory) Set(ctx context.Context, key string, state *State, ttl time.Duration) error {
	s := m.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	reqsCopy := make([]time.Time, len(state.Requests))
	copy(reqsCopy, state.Requests)

	s.data[key] = &memEntry{
		tokens:    state.Tokens,
		lastUpd:   state.LastUpdate,
		count:     state.Count,
		requests:  reqsCopy,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Increment atomically increments the count for a key.
func (m *Memory) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	s := m.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	if e, ok := s.data[key]; ok && now.Before(e.expiresAt) {
		e.count++
		e.lastUpd = now
		return e.count, nil
	}

	s.data[key] = &memEntry{
		count:     1,
		lastUpd:   now,
		expiresAt: now.Add(ttl),
	}

	return 1, nil
}

// Reset removes the state for a key.
func (m *Memory) Reset(ctx context.Context, key string) error {
	s := m.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	return nil
}

// Close stops the cleanup goroutine.
func (m *Memory) Close() error {
	select {
	case <-m.closeCh:
		return nil
	default:
		close(m.closeCh)
	}
	return nil
}

// AllowTokenBucket performs an atomic token bucket evaluation.
func (m *Memory) AllowTokenBucket(ctx context.Context, key string, rate float64, capacity int64, n int64, ttl time.Duration) (*EvalResult, error) {
	s := m.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	e, ok := s.data[key]
	if !ok || now.After(e.expiresAt) {
		e = &memEntry{
			tokens:  float64(capacity),
			lastUpd: now,
		}
		s.data[key] = e
	} else if e.lastUpd.IsZero() {
		e.tokens = float64(capacity)
		e.lastUpd = now
	} else {
		elapsed := now.Sub(e.lastUpd).Seconds()
		e.tokens += elapsed * rate
		if e.tokens > float64(capacity) {
			e.tokens = float64(capacity)
		}
		e.lastUpd = now
	}

	e.expiresAt = now.Add(ttl)

	allowed := e.tokens >= float64(n)
	if allowed {
		e.tokens -= float64(n)
	}

	remaining := int64(e.tokens)
	if remaining < 0 {
		remaining = 0
	}

	var retryAfter time.Duration
	if !allowed {
		tokensNeeded := float64(n) - e.tokens
		retryAfter = time.Duration((tokensNeeded / rate) * float64(time.Second))
		if retryAfter < 0 {
			retryAfter = 0
		}
	}

	resetSecs := float64(capacity) / rate
	resetAt := now.Add(time.Duration(resetSecs * float64(time.Second)))

	return &EvalResult{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetAt:    resetAt,
		RetryAfter: retryAfter,
	}, nil
}

// AllowFixedWindow performs an atomic fixed window evaluation.
func (m *Memory) AllowFixedWindow(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (*EvalResult, error) {
	s := m.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	windowMs := window.Milliseconds()
	if windowMs <= 0 {
		windowMs = 1
	}
	currentWindowNum := now.UnixMilli() / windowMs
	windowEnd := time.UnixMilli((currentWindowNum + 1) * windowMs)

	e, ok := s.data[key]
	if !ok || now.After(e.expiresAt) {
		e = &memEntry{
			count:     0,
			windowNum: currentWindowNum,
			lastUpd:   now,
		}
		s.data[key] = e
	} else if e.windowNum != currentWindowNum {
		e.count = 0
		e.windowNum = currentWindowNum
		e.lastUpd = now
	}

	e.expiresAt = windowEnd.Add(window)

	allowed := (e.count + n) <= limit
	if allowed {
		e.count += n
	}

	remaining := limit - e.count
	if remaining < 0 {
		remaining = 0
	}

	var retryAfter time.Duration
	if !allowed {
		retryAfter = windowEnd.Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}
	}

	return &EvalResult{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetAt:    windowEnd,
		RetryAfter: retryAfter,
	}, nil
}

// AllowSlidingWindowCounter performs an atomic sliding window counter evaluation (Cloudflare style).
func (m *Memory) AllowSlidingWindowCounter(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (*EvalResult, error) {
	s := m.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	windowMs := window.Milliseconds()
	if windowMs <= 0 {
		windowMs = 1
	}
	currentWindowNum := now.UnixMilli() / windowMs
	windowStartMs := currentWindowNum * windowMs
	windowEnd := time.UnixMilli((currentWindowNum + 1) * windowMs)

	e, ok := s.data[key]
	if !ok || now.After(e.expiresAt) {
		e = &memEntry{
			count:     0,
			prevCount: 0,
			windowNum: currentWindowNum,
			lastUpd:   now,
		}
		s.data[key] = e
	} else if e.windowNum != currentWindowNum {
		if currentWindowNum == e.windowNum+1 {
			e.prevCount = e.count
		} else {
			e.prevCount = 0
		}
		e.count = 0
		e.windowNum = currentWindowNum
		e.lastUpd = now
	}

	e.expiresAt = windowEnd.Add(window)

	timeIntoWindowMs := now.UnixMilli() - windowStartMs
	weight := 1.0 - (float64(timeIntoWindowMs) / float64(windowMs))
	if weight < 0 {
		weight = 0
	}

	estimatedCount := int64(float64(e.prevCount)*weight + float64(e.count))
	allowed := (estimatedCount + n) <= limit
	if allowed {
		e.count += n
		estimatedCount += n
	}

	remaining := limit - estimatedCount
	if remaining < 0 {
		remaining = 0
	}

	var retryAfter time.Duration
	if !allowed {
		retryAfter = windowEnd.Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}
	}

	return &EvalResult{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetAt:    windowEnd,
		RetryAfter: retryAfter,
	}, nil
}

// AllowSlidingWindowLog performs an atomic sliding window log evaluation.
func (m *Memory) AllowSlidingWindowLog(ctx context.Context, key string, limit int64, window time.Duration, n int64, ttl time.Duration) (*EvalResult, error) {
	s := m.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-window)

	e, ok := s.data[key]
	if !ok || now.After(e.expiresAt) {
		e = &memEntry{
			requests: make([]time.Time, 0, 16),
			lastUpd:  now,
		}
		s.data[key] = e
	}

	e.expiresAt = now.Add(ttl)

	// Filter out expired timestamps in-place
	validCount := 0
	for _, reqTime := range e.requests {
		if reqTime.After(windowStart) {
			e.requests[validCount] = reqTime
			validCount++
		}
	}
	e.requests = e.requests[:validCount]

	currentCount := int64(validCount)
	allowed := (currentCount + n) <= limit

	if allowed {
		for i := int64(0); i < n; i++ {
			e.requests = append(e.requests, now)
		}
		currentCount += n
	}

	remaining := limit - currentCount
	if remaining < 0 {
		remaining = 0
	}

	var resetAt time.Time
	var retryAfter time.Duration
	if len(e.requests) > 0 {
		resetAt = e.requests[0].Add(window)
		if !allowed {
			retryAfter = resetAt.Sub(now)
			if retryAfter < 0 {
				retryAfter = 0
			}
		}
	} else {
		resetAt = now.Add(window)
	}

	return &EvalResult{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetAt:    resetAt,
		RetryAfter: retryAfter,
	}, nil
}

func (m *Memory) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.removeExpired()
		case <-m.closeCh:
			return
		}
	}
}

func (m *Memory) removeExpired() {
	now := time.Now()
	for i := 0; i < numShards; i++ {
		s := m.shards[i]
		s.mu.Lock()
		for key, e := range s.data {
			if now.After(e.expiresAt) {
				delete(s.data, key)
			}
		}
		s.mu.Unlock()
	}
}