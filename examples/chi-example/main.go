package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/bshome19/rateshield"
	"github.com/bshome19/rateshield/algorithms"
	"github.com/bshome19/rateshield/middleware"
	"github.com/bshome19/rateshield/store"
	"github.com/go-chi/chi/v5"
)

func main() {
	// Create store
	memStore := store.NewMemory()

	// Create limiter: 30 requests per minute
	limiter, err := algorithms.NewSlidingWindow(algorithms.SlidingWindowConfig{
		Limit:  30,
		Window: time.Minute,
		Store:  memStore,
	})
	if err != nil {
		log.Fatalf("Failed to create limiter: %v", err)
	}

	// Create Chi router
	r := chi.NewRouter()

	// Apply global rate limiting
	r.Use(middleware.Chi(middleware.ChiConfig{
		Limiter:      limiter,
		KeyExtractor: rateshield.KeyByIP,
		Skip: func(req *http.Request) bool {
			return req.URL.Path == "/health"
		},
	}))

	// Root endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Welcome to rateshield with Chi!",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Health check (skipped by rate limiter)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// API routes with stricter per-endpoint limiting
	r.Route("/api", func(r chi.Router) {
		apiLimiter, _ := algorithms.NewTokenBucket(algorithms.TokenBucketConfig{
			Rate:     3,
			Capacity: 6,
			Store:    memStore,
		})

		r.Use(middleware.Chi(middleware.ChiConfig{
			Limiter:      apiLimiter,
			KeyExtractor: rateshield.KeyByIPAndEndpoint,
		}))

		r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"users": []map[string]string{
					{"id": "1", "name": "Alice"},
					{"id": "2", "name": "Bob"},
					{"id": "3", "name": "Charlie"},
				},
			})
		})

		r.Get("/users/{userID}", func(w http.ResponseWriter, r *http.Request) {
			userID := chi.URLParam(r, "userID")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":    userID,
				"name":  "User " + userID,
				"email": "user" + userID + "@example.com",
			})
		})
	})

	// Get port from environment or find available one
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Try to get a listener on the port
	addr := ":" + port
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("⚠️  Port %s is in use, finding available port...\n", port)
		listener, err = net.Listen("tcp", ":0")
		if err != nil {
			log.Fatalf("Failed to find available port: %v", err)
		}
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port

	fmt.Println()
	fmt.Println("🚦 Chi server starting with rateshield rate limiting")
	fmt.Printf("🌐 Server running on http://localhost:%d\n", actualPort)
	fmt.Println()
	fmt.Printf("Try: curl http://localhost:%d/\n", actualPort)
	fmt.Printf("Try: curl http://localhost:%d/api/users\n", actualPort)
	fmt.Printf("Try: curl http://localhost:%d/api/users/123\n", actualPort)
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop the server")
	fmt.Println()

	// Serve using the listener
	if err := http.Serve(listener, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}