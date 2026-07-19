package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/bshome19/rateshield"
	"github.com/bshome19/rateshield/algorithms"
	"github.com/bshome19/rateshield/store"
)

func createTestLimiter(t *testing.T, capacity int64) (rateshield.Limiter, *store.Memory) {
	memStore := store.NewMemory()

	limiter, err := algorithms.NewTokenBucket(algorithms.TokenBucketConfig{
		Rate:     10,
		Capacity: capacity,
		Store:    memStore,
	})
	if err != nil {
		t.Fatalf("Failed to create limiter: %v", err)
	}

	return limiter, memStore
}

func TestStdlib_AllowedRequest(t *testing.T) {
	limiter, memStore := createTestLimiter(t, 10)
	defer memStore.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	middleware := Stdlib(Config{
		Limiter: limiter,
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	if body != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", body)
	}
}

func TestStdlib_RateLimitHeaders(t *testing.T) {
	limiter, memStore := createTestLimiter(t, 10)
	defer memStore.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Stdlib(Config{
		Limiter: limiter,
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rr, req)

	// Check all rate limit headers exist
	if rr.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("Missing X-RateLimit-Limit header")
	}
	if rr.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("Missing X-RateLimit-Remaining header")
	}
	if rr.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("Missing X-RateLimit-Reset header")
	}
}

func TestStdlib_RateLimited(t *testing.T) {
	limiter, memStore := createTestLimiter(t, 2)
	defer memStore.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Stdlib(Config{
		Limiter: limiter,
	})

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()
		middleware(handler).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Request %d: Expected 200, got %d", i+1, rr.Code)
		}
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", rr.Code)
	}

	// Check Retry-After header
	if rr.Header().Get("Retry-After") == "" {
		t.Error("Missing Retry-After header")
	}
}

func TestStdlib_RateLimitedResponseBody(t *testing.T) {
	limiter, memStore := createTestLimiter(t, 1)
	defer memStore.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Stdlib(Config{
		Limiter: limiter,
	})

	// First request
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	// Second request (rate limited)
	rr = httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	// Check response body contains error
	body := rr.Body.String()
	if body == "" {
		t.Error("Expected non-empty response body")
	}
}

func TestStdlib_Skip(t *testing.T) {
	limiter, memStore := createTestLimiter(t, 1)
	defer memStore.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Stdlib(Config{
		Limiter: limiter,
		Skip: func(r *http.Request) bool {
			return r.URL.Path == "/health"
		},
	})

	// Exhaust limit on regular endpoint
	req := httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	// Second request should be limited
	rr = httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429 on /api, got %d", rr.Code)
	}

	// Health endpoint should always work (skipped)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()
		middleware(handler).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Health request %d: Expected 200, got %d", i+1, rr.Code)
		}
	}
}

func TestStdlib_SkipMultiplePaths(t *testing.T) {
	limiter, memStore := createTestLimiter(t, 1)
	defer memStore.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Stdlib(Config{
		Limiter: limiter,
		Skip: func(r *http.Request) bool {
			return r.URL.Path == "/health" || r.URL.Path == "/metrics"
		},
	})

	// Both /health and /metrics should be skipped
	paths := []string{"/health", "/metrics"}
	for _, path := range paths {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", path, nil)
			req.RemoteAddr = "192.168.1.1:12345"
			rr := httptest.NewRecorder()
			middleware(handler).ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Path %s request %d: Expected 200, got %d", path, i+1, rr.Code)
			}
		}
	}
}

func TestStdlib_CustomKeyExtractor(t *testing.T) {
	limiter, memStore := createTestLimiter(t, 2)
	defer memStore.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Stdlib(Config{
		Limiter:      limiter,
		KeyExtractor: rateshield.KeyByHeader("X-API-Key"),
	})

	// Exhaust limit for API key 1
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-API-Key", "key1")
		rr := httptest.NewRecorder()
		middleware(handler).ServeHTTP(rr, req)
	}

	// API key 1 should be limited
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "key1")
	rr := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429 for key1, got %d", rr.Code)
	}

	// API key 2 should still work
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "key2")
	rr = httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for key2, got %d", rr.Code)
	}
}

func TestStdlib_CustomErrorHandler(t *testing.T) {
	limiter, memStore := createTestLimiter(t, 1)
	defer memStore.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	customError := map[string]interface{}{
		"custom": true,
		"error":  "custom rate limit error",
		"code":   429,
	}

	middleware := Stdlib(Config{
		Limiter: limiter,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, result *rateshield.Result) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Custom-Header", "custom-value")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(customError)
		},
	})

	// First request
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	// Second request (should use custom error handler)
	rr = httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", rr.Code)
	}

	// Check custom header
	if rr.Header().Get("X-Custom-Header") != "custom-value" {
		t.Error("Custom header not set")
	}

	var response map[string]interface{}
	_ = json.NewDecoder(rr.Body).Decode(&response)

	if response["custom"] != true {
		t.Error("Custom error handler not called")
	}
}

func TestStdlib_HeaderValues(t *testing.T) {
	limiter, memStore := createTestLimiter(t, 5)
	defer memStore.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Stdlib(Config{
		Limiter: limiter,
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()
	middleware(handler).ServeHTTP(rr, req)

	// Check header values
	limit, err := strconv.ParseInt(rr.Header().Get("X-RateLimit-Limit"), 10, 64)
	if err != nil {
		t.Fatalf("Failed to parse X-RateLimit-Limit: %v", err)
	}
	if limit != 5 {
		t.Errorf("Expected limit 5, got %d", limit)
	}

	remaining, err := strconv.ParseInt(rr.Header().Get("X-RateLimit-Remaining"), 10, 64)
	if err != nil {
		t.Fatalf("Failed to parse X-RateLimit-Remaining: %v", err)
	}
	if remaining != 4 {
		t.Errorf("Expected remaining 4, got %d", remaining)
	}

	resetStr := rr.Header().Get("X-RateLimit-Reset")
	if resetStr == "" {
		t.Fatal("X-RateLimit-Reset header is empty")
	}

	reset, err := strconv.ParseInt(resetStr, 10, 64)
	if err != nil {
		t.Fatalf("Failed to parse X-RateLimit-Reset: %v", err)
	}

	now := time.Now().Unix()
	if reset < now-1 {
		t.Errorf("Reset time %d is too far in the past (now: %d)", reset, now)
	}
	if reset > now+60 {
		t.Errorf("Reset time %d is too far in the future (now: %d)", reset, now)
	}
}

func TestStdlib_DecrementingRemaining(t *testing.T) {
	limiter, memStore := createTestLimiter(t, 5)
	defer memStore.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Stdlib(Config{
		Limiter: limiter,
	})

	// Track remaining values
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()
		middleware(handler).ServeHTTP(rr, req)

		remaining, _ := strconv.ParseInt(rr.Header().Get("X-RateLimit-Remaining"), 10, 64)
		expected := int64(4 - i)
		if remaining != expected {
			t.Errorf("Request %d: Expected remaining %d, got %d", i+1, expected, remaining)
		}
	}
}

func TestStdlibSimple(t *testing.T) {
	limiter, memStore := createTestLimiter(t, 5)
	defer memStore.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := StdlibSimple(limiter)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()

	middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}

	// Should have rate limit headers
	if rr.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("Missing rate limit headers in simple middleware")
	}
}

func TestStdlib_EmptyKey(t *testing.T) {
	limiter, memStore := createTestLimiter(t, 1)
	defer memStore.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Key extractor that returns empty string
	middleware := Stdlib(Config{
		Limiter: limiter,
		KeyExtractor: func(r *http.Request) string {
			return "" // Empty key
		},
	})

	// Should skip rate limiting when key is empty
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		middleware(handler).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Request %d: Expected 200, got %d", i+1, rr.Code)
		}
	}
}

func TestStdlib_DefaultKeyExtractor(t *testing.T) {
	limiter, memStore := createTestLimiter(t, 2)
	defer memStore.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// No custom key extractor - should use KeyByIP
	middleware := Stdlib(Config{
		Limiter: limiter,
	})

	// Different IPs should have separate limits
	ips := []string{"1.1.1.1:1234", "2.2.2.2:1234"}
	for _, ip := range ips {
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = ip
			rr := httptest.NewRecorder()
			middleware(handler).ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("IP %s request %d: Expected 200, got %d", ip, i+1, rr.Code)
			}
		}
	}
}

func TestStdlib_MethodsAndPaths(t *testing.T) {
	// Use a large capacity so we don't hit the rate limit
	limiter, memStore := createTestLimiter(t, 100)
	defer memStore.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Stdlib(Config{
		Limiter: limiter,
	})

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	paths := []string{"/", "/api", "/api/users", "/api/users/123"}

	for _, method := range methods {
		for _, path := range paths {
			req := httptest.NewRequest(method, path, nil)
			req.RemoteAddr = "192.168.1.1:12345"
			rr := httptest.NewRecorder()
			middleware(handler).ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("%s %s: Expected 200, got %d", method, path, rr.Code)
			}
		}
	}
}

func TestConfig_Defaults(t *testing.T) {
	limiter, memStore := createTestLimiter(t, 5)
	defer memStore.Close()

	// Empty config except limiter
	cfg := Config{
		Limiter: limiter,
	}

	if cfg.KeyExtractor != nil {
		t.Error("KeyExtractor should be nil by default")
	}
	if cfg.ErrorHandler != nil {
		t.Error("ErrorHandler should be nil by default")
	}
	if cfg.Skip != nil {
		t.Error("Skip should be nil by default")
	}
}

func TestSetRateLimitHeaders(t *testing.T) {
	result := &rateshield.Result{
		Limit:     100,
		Remaining: 95,
		ResetAt:   time.Now().Add(time.Minute),
	}

	rr := httptest.NewRecorder()
	setRateLimitHeaders(rr, result)

	if rr.Header().Get("X-RateLimit-Limit") != "100" {
		t.Error("X-RateLimit-Limit not set correctly")
	}
	if rr.Header().Get("X-RateLimit-Remaining") != "95" {
		t.Error("X-RateLimit-Remaining not set correctly")
	}
	if rr.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("X-RateLimit-Reset not set")
	}
}

func TestDefaultErrorHandler(t *testing.T) {
	result := &rateshield.Result{
		RetryAfter: 30 * time.Second,
	}

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	defaultErrorHandler(rr, req, result)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", rr.Code)
	}

	if rr.Header().Get("Content-Type") != "application/json" {
		t.Error("Content-Type should be application/json")
	}

	if rr.Header().Get("Retry-After") != "30" {
		t.Errorf("Retry-After should be 30, got %s", rr.Header().Get("Retry-After"))
	}
}
