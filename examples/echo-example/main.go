package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/bshome19/rateshield/algorithms"
	"github.com/bshome19/rateshield/middleware"
	"github.com/bshome19/rateshield/store"
	"github.com/labstack/echo/v4"
)

func main() {
	// Create store
	memStore := store.NewMemory()

	// Create limiter: 20 requests per minute
	limiter, err := algorithms.NewSlidingWindow(algorithms.SlidingWindowConfig{
		Limit:  20,
		Window: time.Minute,
		Store:  memStore,
	})
	if err != nil {
		log.Fatalf("Failed to create limiter: %v", err)
	}

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Apply global rate limiting
	e.Use(middleware.Echo(middleware.EchoConfig{
		Limiter:      limiter,
		KeyExtractor: middleware.EchoKeyByIP(),
		Skipper: func(c echo.Context) bool {
			return c.Path() == "/health"
		},
	}))

	// Root endpoint
	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Welcome to rateshield with Echo!",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Health check (skipped by rate limiter)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// API group with per-user rate limiting
	apiLimiter, _ := algorithms.NewTokenBucket(algorithms.TokenBucketConfig{
		Rate:     5,
		Capacity: 10,
		Store:    memStore,
	})

	api := e.Group("/api")
	api.Use(middleware.Echo(middleware.EchoConfig{
		Limiter:      apiLimiter,
		KeyExtractor: middleware.EchoKeyByUserID("X-User-ID"),
	}))

	api.GET("/profile", func(c echo.Context) error {
		userID := c.Request().Header.Get("X-User-ID")
		if userID == "" {
			userID = "anonymous"
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"user_id": userID,
			"name":    "John Doe",
			"email":   "john@example.com",
		})
	})

	api.GET("/settings", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"theme":         "dark",
			"notifications": true,
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
	fmt.Println("🚦 Echo server starting with rateshield rate limiting")
	fmt.Printf("🌐 Server running on http://localhost:%d\n", actualPort)
	fmt.Println()
	fmt.Printf("Try: curl http://localhost:%d/\n", actualPort)
	fmt.Printf("Try: curl -H 'X-User-ID: user123' http://localhost:%d/api/profile\n", actualPort)
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop the server")
	fmt.Println()

	// Start server with the listener
	e.Listener = listener
	if err := e.Start(""); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
