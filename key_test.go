package rateshield

import (
	"net/http/httptest"
	"testing"
)

func TestKeyByIP(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		xRealIP        string
		expectedKey    string
	}{
		{
			name:        "simple remote addr",
			remoteAddr:  "192.168.1.1:12345",
			expectedKey: "192.168.1.1",
		},
		{
			name:        "remote addr without port",
			remoteAddr:  "192.168.1.1",
			expectedKey: "192.168.1.1",
		},
		{
			name:          "x-forwarded-for single",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.195",
			expectedKey:   "203.0.113.195",
		},
		{
			name:          "x-forwarded-for multiple",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.195, 70.41.3.18, 150.172.238.178",
			expectedKey:   "203.0.113.195",
		},
		{
			name:        "x-real-ip",
			remoteAddr:  "10.0.0.1:12345",
			xRealIP:     "203.0.113.195",
			expectedKey: "203.0.113.195",
		},
		{
			name:          "x-forwarded-for takes precedence",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "1.1.1.1",
			xRealIP:       "2.2.2.2",
			expectedKey:   "1.1.1.1",
		},
		{
			name:        "ipv6 address",
			remoteAddr:  "[::1]:12345",
			expectedKey: "::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			key := KeyByIP(req)
			if key != tt.expectedKey {
				t.Errorf("KeyByIP() = %q, want %q", key, tt.expectedKey)
			}
		})
	}
}

func TestKeyByEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		path        string
		expectedKey string
	}{
		{
			name:        "GET root",
			method:      "GET",
			path:        "/",
			expectedKey: "GET:/",
		},
		{
			name:        "POST api",
			method:      "POST",
			path:        "/api/users",
			expectedKey: "POST:/api/users",
		},
		{
			name:        "DELETE with id",
			method:      "DELETE",
			path:        "/api/users/123",
			expectedKey: "DELETE:/api/users/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			key := KeyByEndpoint(req)
			if key != tt.expectedKey {
				t.Errorf("KeyByEndpoint() = %q, want %q", key, tt.expectedKey)
			}
		})
	}
}

func TestKeyByHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "my-api-key-123")
	req.Header.Set("Authorization", "Bearer token123")

	extractor := KeyByHeader("X-API-Key")
	key := extractor(req)
	if key != "my-api-key-123" {
		t.Errorf("KeyByHeader() = %q, want %q", key, "my-api-key-123")
	}

	extractor = KeyByHeader("Authorization")
	key = extractor(req)
	if key != "Bearer token123" {
		t.Errorf("KeyByHeader() = %q, want %q", key, "Bearer token123")
	}

	extractor = KeyByHeader("Non-Existent")
	key = extractor(req)
	if key != "" {
		t.Errorf("KeyByHeader() = %q, want empty string", key)
	}
}

func TestKeyByIPAndEndpoint(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/data", nil)
	req.RemoteAddr = "192.168.1.100:54321"

	key := KeyByIPAndEndpoint(req)
	expected := "192.168.1.100:POST:/api/data"
	if key != expected {
		t.Errorf("KeyByIPAndEndpoint() = %q, want %q", key, expected)
	}
}

func TestKeyByUserID(t *testing.T) {
	tests := []struct {
		name        string
		headerValue string
		remoteAddr  string
		expectedKey string
	}{
		{
			name:        "with user id",
			headerValue: "user-456",
			remoteAddr:  "10.0.0.1:12345",
			expectedKey: "user:user-456",
		},
		{
			name:        "without user id fallback to ip",
			headerValue: "",
			remoteAddr:  "10.0.0.1:12345",
			expectedKey: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.headerValue != "" {
				req.Header.Set("X-User-ID", tt.headerValue)
			}

			extractor := KeyByUserID("X-User-ID")
			key := extractor(req)
			if key != tt.expectedKey {
				t.Errorf("KeyByUserID() = %q, want %q", key, tt.expectedKey)
			}
		})
	}
}

func TestKeyByUserAndEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/profile", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-User-ID", "john123")

	extractor := KeyByUserAndEndpoint("X-User-ID")
	key := extractor(req)
	expected := "john123:GET:/api/profile"
	if key != expected {
		t.Errorf("KeyByUserAndEndpoint() = %q, want %q", key, expected)
	}

	// Without user ID
	req2 := httptest.NewRequest("GET", "/api/profile", nil)
	req2.RemoteAddr = "10.0.0.1:12345"

	key = extractor(req2)
	expected = "10.0.0.1:GET:/api/profile"
	if key != expected {
		t.Errorf("KeyByUserAndEndpoint() = %q, want %q", key, expected)
	}
}