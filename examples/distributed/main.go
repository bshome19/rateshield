package main

import (
	"context"
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
	"github.com/redis/go-redis/v9"
)

func main() {
	// Get Redis URL from environment or use default
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	var limiter rateshield.Limiter
	var storeName string

	// Try to connect to Redis with minimal retries
	rdb := redis.NewClient(&redis.Options{
		Addr:         redisURL,
		DialTimeout:  1 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		PoolSize:     5,
		MaxRetries:   0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	err := rdb.Ping(ctx).Err()
	cancel()

	if err != nil {
		fmt.Println("⚠️  Redis not available")
		fmt.Println("📦 Falling back to in-memory store")
		fmt.Println()
		fmt.Println("💡 To use Redis, either:")
		fmt.Println("   1. Install: sudo apt install redis-server && sudo systemctl start redis-server")
		fmt.Println("   2. Docker:  docker run -d -p 6379:6379 redis:latest")
		fmt.Println()

		rdb.Close()

		memStore := store.NewMemory()
		storeName = "In-Memory"

		limiter, err = algorithms.NewSlidingWindow(algorithms.SlidingWindowConfig{
			Limit:  100,
			Window: time.Minute,
			Store:  memStore,
		})
		if err != nil {
			log.Fatalf("Failed to create limiter: %v", err)
		}
	} else {
		fmt.Println("✅ Connected to Redis at", redisURL)
		storeName = "Redis"

		redisStore, err := store.NewRedis(store.RedisConfig{
			Client: rdb,
			Prefix: "rateshield:demo:",
		})
		if err != nil {
			log.Fatalf("Failed to create Redis store: %v", err)
		}

		limiter, err = algorithms.NewSlidingWindow(algorithms.SlidingWindowConfig{
			Limit:  100,
			Window: time.Minute,
			Store:  redisStore,
		})
		if err != nil {
			log.Fatalf("Failed to create limiter: %v", err)
		}
	}

	// Create HTTP server
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"message": "Hello from rateshield!", "store": "%s"}`, storeName)))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
	})

	// Apply middleware
	handler := middleware.Stdlib(middleware.Config{
		Limiter:      limiter,
		KeyExtractor: rateshield.KeyByIP,
		Skip: func(r *http.Request) bool {
			return r.URL.Path == "/health"
		},
	})(mux)

	// Create listener FIRST to check if port is available
	addr := ":" + port
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// Port is in use, try to find another
		fmt.Printf("⚠️  Port %s is in use, finding available port...\n", port)
		listener, err = net.Listen("tcp", ":0") // :0 means any available port
		if err != nil {
			log.Fatalf("Failed to find available port: %v", err)
		}
	}

	// Get the actual port we're listening on
	actualPort := listener.Addr().(*net.TCPAddr).Port

	fmt.Println()
	fmt.Printf("🚦 Rate limiter starting with %s store\n", storeName)
	fmt.Printf("🌐 Server running on http://localhost:%d\n", actualPort)
	fmt.Println()
	fmt.Println("Try these commands:")
	fmt.Printf("  curl http://localhost:%d/\n", actualPort)
	fmt.Printf("  curl http://localhost:%d/health\n", actualPort)
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop the server")
	fmt.Println()

	// Use the listener we already created
	if err := http.Serve(listener, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
