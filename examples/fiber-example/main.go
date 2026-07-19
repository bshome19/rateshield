package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/bshome19/rateshield/algorithms"
	"github.com/bshome19/rateshield/middleware"
	"github.com/bshome19/rateshield/store"
	"github.com/gofiber/fiber/v2"
)

func main() {
	// Create store
	memStore := store.NewMemory()

	// Create limiter: 10 requests per minute
	limiter, err := algorithms.NewSlidingWindow(algorithms.SlidingWindowConfig{
		Limit:  10,
		Window: time.Minute,
		Store:  memStore,
	})
	if err != nil {
		log.Fatalf("Failed to create limiter: %v", err)
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "rateshield Fiber Example",
		DisableStartupMessage: true,
	})

	// Apply global rate limiting
	app.Use(middleware.Fiber(middleware.FiberConfig{
		Limiter:      limiter,
		KeyExtractor: middleware.FiberKeyByIP(),
	}))

	// Root endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to rateshield with Fiber!",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Health check (skip rate limiting)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// API group with stricter limits
	apiLimiter, _ := algorithms.NewTokenBucket(algorithms.TokenBucketConfig{
		Rate:     2,
		Capacity: 5,
		Store:    memStore,
	})

	api := app.Group("/api")
	api.Use(middleware.Fiber(middleware.FiberConfig{
		Limiter:      apiLimiter,
		KeyExtractor: middleware.FiberKeyByIPAndRoute(),
	}))

	api.Get("/users", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"users": []string{"alice", "bob", "charlie"},
		})
	})

	api.Get("/posts", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"posts": []string{"Hello World", "My First Post"},
		})
	})

	// Get port from environment or find available one
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
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
	fmt.Println("🚦 Fiber server starting with rateshield rate limiting")
	fmt.Printf("🌐 Server running on http://localhost:%d\n", actualPort)
	fmt.Println()
	fmt.Printf("Try: curl http://localhost:%d/\n", actualPort)
	fmt.Printf("Try: curl http://localhost:%d/api/users\n", actualPort)
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop the server")
	fmt.Println()

	// Serve using the listener
	if err := app.Listener(listener); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
