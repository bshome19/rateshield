package main

import (
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
	"github.com/gin-gonic/gin"
)

func main() {
	// Create store
	memStore := store.NewMemory()

	// Create a limiter: 10 requests per minute
	limiter, err := algorithms.NewSlidingWindow(algorithms.SlidingWindowConfig{
		Limit:  10,
		Window: time.Minute,
		Store:  memStore,
	})
	if err != nil {
		log.Fatalf("Failed to create limiter: %v", err)
	}

	// Set Gin to release mode to reduce noise
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	r := gin.New()
	r.Use(gin.Recovery())

	// Apply global rate limiting by IP
	r.Use(middleware.Gin(middleware.GinConfig{
		Limiter:      limiter,
		KeyExtractor: rateshield.KeyByIP,
	}))

	// Public endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to rateshield with Gin!",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// API endpoint with stricter limits
	apiLimiter, _ := algorithms.NewTokenBucket(algorithms.TokenBucketConfig{
		Rate:     2,
		Capacity: 5,
		Store:    memStore,
	})

	api := r.Group("/api")
	api.Use(middleware.Gin(middleware.GinConfig{
		Limiter:      apiLimiter,
		KeyExtractor: rateshield.KeyByIPAndEndpoint,
	}))

	api.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": "Here is your data!",
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
	fmt.Println("🚦 Gin server starting with rateshield rate limiting")
	fmt.Printf("🌐 Server running on http://localhost:%d\n", actualPort)
	fmt.Println()
	fmt.Printf("Try: curl http://localhost:%d/\n", actualPort)
	fmt.Printf("Try: curl http://localhost:%d/api/data\n", actualPort)
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop the server")
	fmt.Println()

	// Serve using the listener
	if err := http.Serve(listener, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
