package rateshield

import (
	"net"
	"net/http"
	"strings"
)

// KeyExtractor extracts a rate limit key from an HTTP request.
type KeyExtractor func(r *http.Request) string

// KeyByIP extracts the client IP address as the rate limit key.
func KeyByIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// KeyByHeader returns a KeyExtractor that uses a specific header value.
func KeyByHeader(header string) KeyExtractor {
	return func(r *http.Request) string {
		return r.Header.Get(header)
	}
}

// KeyByEndpoint extracts the HTTP method and path as the rate limit key.
func KeyByEndpoint(r *http.Request) string {
	return r.Method + ":" + r.URL.Path
}

// KeyByIPAndEndpoint combines IP and endpoint for the rate limit key.
func KeyByIPAndEndpoint(r *http.Request) string {
	return KeyByIP(r) + ":" + KeyByEndpoint(r)
}

// KeyByUserID returns a KeyExtractor that uses a user ID from a header.
func KeyByUserID(header string) KeyExtractor {
	return func(r *http.Request) string {
		userID := r.Header.Get(header)
		if userID == "" {
			return KeyByIP(r) // Fallback to IP if no user ID
		}
		return "user:" + userID
	}
}

// KeyByUserAndEndpoint returns a KeyExtractor combining user ID and endpoint.
func KeyByUserAndEndpoint(userHeader string) KeyExtractor {
	return func(r *http.Request) string {
		userID := r.Header.Get(userHeader)
		if userID == "" {
			userID = KeyByIP(r)
		}
		return userID + ":" + r.Method + ":" + r.URL.Path
	}
}
