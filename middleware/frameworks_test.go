package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bshome19/rateshield/algorithms"
	"github.com/bshome19/rateshield/store"
	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/labstack/echo/v4"
)

// TestRawGo_WithoutFramework tests rateshield directly in plain Go code (no HTTP/web framework).
func TestRawGo_WithoutFramework(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	limiter, err := algorithms.NewTokenBucket(algorithms.TokenBucketConfig{
		Rate:     5,
		Capacity: 3,
		Store:    memStore,
	})
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	ctx := context.Background()
	key := "user_raw_go_123"

	// First 3 requests must succeed
	for i := 0; i < 3; i++ {
		res, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("unexpected error on request %d: %v", i+1, err)
		}
		if !res.Allowed {
			t.Fatalf("expected request %d to be allowed", i+1)
		}
	}

	// 4th request must be blocked
	res, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Allowed {
		t.Fatalf("expected 4th request to be blocked")
	}
}

// TestGinMiddleware tests Gin integration.
func TestGinMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	memStore := store.NewMemory()
	defer memStore.Close()

	limiter, _ := algorithms.NewFixedWindow(algorithms.FixedWindowConfig{
		Limit:  2,
		Window: time.Minute,
		Store:  memStore,
	})

	r := gin.New()
	r.Use(GinSimple(limiter))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Req 1 & 2 allowed
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Gin: expected 200, got %d", w.Code)
		}
	}

	// Req 3 rate limited (429)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("Gin: expected 429, got %d", w.Code)
	}
}

// TestFiberMiddleware tests Fiber integration.
func TestFiberMiddleware(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	limiter, _ := algorithms.NewFixedWindow(algorithms.FixedWindowConfig{
		Limit:  2,
		Window: time.Minute,
		Store:  memStore,
	})

	app := fiber.New()
	app.Use(FiberSimple(limiter))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Req 1 & 2 allowed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Fiber test error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Fiber: expected 200, got %d", resp.StatusCode)
		}
	}

	// Req 3 blocked
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("Fiber: expected 429, got %d", resp.StatusCode)
	}
}

// TestEchoMiddleware tests Echo integration.
func TestEchoMiddleware(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	limiter, _ := algorithms.NewFixedWindow(algorithms.FixedWindowConfig{
		Limit:  2,
		Window: time.Minute,
		Store:  memStore,
	})

	e := echo.New()
	e.Use(EchoSimple(limiter))
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.3:1234"
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Echo: expected 200, got %d", rec.Code)
		}
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.3:1234"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("Echo: expected 429, got %d", rec.Code)
	}
}

// TestChiMiddleware tests Chi integration.
func TestChiMiddleware(t *testing.T) {
	memStore := store.NewMemory()
	defer memStore.Close()

	limiter, _ := algorithms.NewFixedWindow(algorithms.FixedWindowConfig{
		Limit:  2,
		Window: time.Minute,
		Store:  memStore,
	})

	r := chi.NewRouter()
	r.Use(ChiSimple(limiter))
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.4:1234"
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Chi: expected 200, got %d", rec.Code)
		}
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.4:1234"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("Chi: expected 429, got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if len(body) == 0 {
		t.Fatalf("Chi: expected response body on 429")
	}
}
