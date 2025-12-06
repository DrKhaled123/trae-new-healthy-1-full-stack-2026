package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore_Allow(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	identifier := "test-user"
	max := 5
	window := time.Minute

	// Test first request
	allowed, count, resetTime, err := store.Allow(ctx, identifier, max, window)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 1, count)
	assert.True(t, time.Now().Before(resetTime))

	// Test multiple requests within limit
	for i := 2; i <= max; i++ {
		allowed, count, _, err := store.Allow(ctx, identifier, max, window)
		require.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, i, count)
	}

	// Test request exceeding limit
	allowed, count, _, err = store.Allow(ctx, identifier, max, window)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, max+1, count)
}

func TestMemoryStore_Reset(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	identifier := "test-user"

	// Make a request
	_, _, _, err := store.Allow(ctx, identifier, 5, time.Minute)
	require.NoError(t, err)

	// Reset
	err = store.Reset(ctx, identifier)
	require.NoError(t, err)

	// Next request should start fresh
	allowed, count, _, err := store.Allow(ctx, identifier, 5, time.Minute)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 1, count)
}

func TestRateLimiterWithConfig(t *testing.T) {
	e := echo.New()

	// Test configuration
	config := RateLimiterConfig{
		Store:               NewMemoryStore(),
		IdentifierExtractor: func(c echo.Context) string { return "test-user" },
		Max:                 3,
		Window:              time.Minute,
		Message:             "Rate limit exceeded",
		StatusCode:          http.StatusTooManyRequests,
		Headers:             true,
	}

	middleware := RateLimiterWithConfig(config)

	// Handler that always returns 200
	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	// Test requests within limit
	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middleware(handler)(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check rate limit headers
		assert.Equal(t, "3", rec.Header().Get("X-RateLimit-Limit"))
		assert.Equal(t, string(rune('0'+3-i)), rec.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
	}

	// Test request exceeding limit
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middleware(handler)(c)
	assert.Error(t, err)
	
	// Check if it's an HTTP error with correct status
	if httpErr, ok := err.(*echo.HTTPError); ok {
		assert.Equal(t, http.StatusTooManyRequests, httpErr.Code)
		assert.Equal(t, "Rate limit exceeded", httpErr.Message)
	}
}

func TestUserBasedRateLimiter(t *testing.T) {
	e := echo.New()
	middleware := UserBasedRateLimiter(2, time.Minute)

	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	// Test with user ID
	for i := 1; i <= 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user_id", "user123")

		err := middleware(handler)(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Third request should fail
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", "user123")

	err := middleware(handler)(c)
	assert.Error(t, err)
}

func TestAPIKeyRateLimiter(t *testing.T) {
	e := echo.New()
	middleware := APIKeyRateLimiter(2, time.Minute)

	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	// Test with API key
	for i := 1; i <= 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-API-Key", "test-key-123")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middleware(handler)(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Third request should fail
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", "test-key-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middleware(handler)(c)
	assert.Error(t, err)
}

func TestRateLimiterSkipper(t *testing.T) {
	e := echo.New()

	config := RateLimiterConfig{
		Skipper: func(c echo.Context) bool {
			return c.Request().URL.Path == "/health"
		},
		Store:  NewMemoryStore(),
		Max:    1,
		Window: time.Minute,
	}

	middleware := RateLimiterWithConfig(config)
	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	// Health check should be skipped
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Multiple requests to /health should all pass
	for i := 0; i < 5; i++ {
		err := middleware(handler)(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func BenchmarkMemoryStore_Allow(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Allow(ctx, "bench-user", 1000, time.Minute)
	}
}

func TestRateLimiterConcurrency(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	identifier := "concurrent-user"
	max := 10
	window := time.Minute

	// Run multiple goroutines concurrently
	results := make(chan bool, 20)
	
	for i := 0; i < 20; i++ {
		go func() {
			allowed, _, _, _ := store.Allow(ctx, identifier, max, window)
			results <- allowed
		}()
	}

	// Collect results
	allowedCount := 0
	for i := 0; i < 20; i++ {
		if <-results {
			allowedCount++
		}
	}

	// Should allow exactly 'max' requests
	assert.Equal(t, max, allowedCount, "Rate limiter should handle concurrent requests correctly")
}