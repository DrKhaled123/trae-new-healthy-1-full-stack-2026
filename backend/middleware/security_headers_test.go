package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityHeadersWithConfig(t *testing.T) {
	e := echo.New()

	config := SecurityHeadersConfig{
		ContentTypeNosniff:        true,
		FrameOptions:              "DENY",
		XSSProtection:             "1; mode=block",
		ContentSecurityPolicy:     "default-src 'self'",
		ReferrerPolicy:           "strict-origin",
		CrossOriginResourcePolicy: "cross-origin",
		HSTS: HSSTConfig{
			Enabled:           true,
			MaxAge:            31536000,
			IncludeSubdomains: true,
			Preload:           false,
		},
		CustomHeaders: map[string]string{
			"X-Test-Header": "test-value",
		},
	}

	middleware := SecurityHeadersWithConfig(config)
	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middleware(handler)(c)
	require.NoError(t, err)

	// Check security headers
	headers := rec.Header()
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"))
	assert.Equal(t, "default-src 'self'", headers.Get("Content-Security-Policy"))
	assert.Equal(t, "strict-origin", headers.Get("Referrer-Policy"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", headers.Get("Strict-Transport-Security"))
	assert.Equal(t, "cross-origin", headers.Get("Cross-Origin-Resource-Policy"))
	assert.Equal(t, "test-value", headers.Get("X-Test-Header"))
}

func TestSecurityHeaders(t *testing.T) {
	e := echo.New()
	middleware := SecurityHeaders()
	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middleware(handler)(c)
	require.NoError(t, err)

	// Check that default headers are set
	headers := rec.Header()
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"))
	assert.NotEmpty(t, headers.Get("Content-Security-Policy"))
	assert.NotEmpty(t, headers.Get("Strict-Transport-Security"))
}

func TestProductionSecurityHeaders(t *testing.T) {
	e := echo.New()
	domain := "example.com"
	middleware := ProductionSecurityHeaders(domain)
	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middleware(handler)(c)
	require.NoError(t, err)

	headers := rec.Header()
	
	// Check that production CSP includes domain-specific policies
	csp := headers.Get("Content-Security-Policy")
	assert.Contains(t, csp, "connect-src 'self' https://api.example.com")
	assert.Contains(t, csp, "upgrade-insecure-requests")
	
	// Check HSTS has longer max-age for production
	hsts := headers.Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=63072000") // 2 years
	assert.Contains(t, hsts, "preload")
}

func TestDevelopmentSecurityHeaders(t *testing.T) {
	e := echo.New()
	middleware := DevelopmentSecurityHeaders()
	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middleware(handler)(c)
	require.NoError(t, err)

	headers := rec.Header()
	
	// Check that development CSP is more permissive
	csp := headers.Get("Content-Security-Policy")
	assert.Contains(t, csp, "'unsafe-inline'")
	assert.Contains(t, csp, "'unsafe-eval'")
	assert.Contains(t, csp, "http:")
	
	// Check HSTS is disabled in development
	hsts := headers.Get("Strict-Transport-Security")
	assert.Empty(t, hsts)
	
	// Check frame options is more permissive
	frameOptions := headers.Get("X-Frame-Options")
	assert.Equal(t, "SAMEORIGIN", frameOptions)
}

func TestAPISecurityHeaders(t *testing.T) {
	e := echo.New()
	middleware := APISecurityHeaders()
	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"data": "api response"})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middleware(handler)(c)
	require.NoError(t, err)

	headers := rec.Header()
	
	// Check API-specific headers
	assert.Equal(t, "1.0", headers.Get("X-API-Version"))
	assert.Contains(t, headers.Get("Cache-Control"), "no-store")
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
	
	// CSP should be empty for API endpoints
	assert.Empty(t, headers.Get("Content-Security-Policy"))
}

func TestHealthCheckSecurityHeaders(t *testing.T) {
	e := echo.New()
	middleware := HealthCheckSecurityHeaders()
	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middleware(handler)(c)
	require.NoError(t, err)

	headers := rec.Header()
	
	// Check minimal headers for health checks
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Contains(t, headers.Get("Cache-Control"), "no-cache")
	assert.Equal(t, "no-cache", headers.Get("Pragma"))
	assert.Equal(t, "0", headers.Get("Expires"))
}

func TestRemoveServerHeader(t *testing.T) {
	e := echo.New()
	middleware := RemoveServerHeader()
	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middleware(handler)(c)
	require.NoError(t, err)

	// Server header should be empty
	assert.Empty(t, rec.Header().Get("Server"))
}

func TestCustomSecurityHeaders(t *testing.T) {
	e := echo.New()
	
	options := map[string]interface{}{
		"csp":              "default-src 'none'",
		"frame_options":    "SAMEORIGIN",
		"referrer_policy":  "no-referrer",
		"hsts_enabled":     false,
		"custom_headers": map[string]string{
			"X-Custom": "custom-value",
		},
	}
	
	middleware := CustomSecurityHeaders(options)
	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middleware(handler)(c)
	require.NoError(t, err)

	headers := rec.Header()
	
	// Check custom configuration
	assert.Equal(t, "default-src 'none'", headers.Get("Content-Security-Policy"))
	assert.Equal(t, "SAMEORIGIN", headers.Get("X-Frame-Options"))
	assert.Equal(t, "no-referrer", headers.Get("Referrer-Policy"))
	assert.Empty(t, headers.Get("Strict-Transport-Security")) // HSTS disabled
	assert.Equal(t, "custom-value", headers.Get("X-Custom"))
}

func TestSecurityHeadersSkipper(t *testing.T) {
	e := echo.New()

	config := SecurityHeadersConfig{
		Skipper: func(c echo.Context) bool {
			return c.Request().URL.Path == "/skip"
		},
		ContentTypeNosniff: true,
		FrameOptions:       "DENY",
	}

	middleware := SecurityHeadersWithConfig(config)
	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	// Test skipped path
	req := httptest.NewRequest(http.MethodGet, "/skip", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := middleware(handler)(c)
	require.NoError(t, err)

	// Headers should not be set for skipped path
	headers := rec.Header()
	assert.Empty(t, headers.Get("X-Content-Type-Options"))
	assert.Empty(t, headers.Get("X-Frame-Options"))

	// Test non-skipped path
	req = httptest.NewRequest(http.MethodGet, "/normal", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	err = middleware(handler)(c)
	require.NoError(t, err)

	// Headers should be set for normal path
	headers = rec.Header()
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
}

func BenchmarkSecurityHeaders(b *testing.B) {
	e := echo.New()
	middleware := SecurityHeaders()
	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		middleware(handler)(c)
	}
}