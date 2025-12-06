package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// RequestSigningConfig holds configuration for request signing
type RequestSigningConfig struct {
	SecretKey       string
	TimestampHeader string
	SignatureHeader string
	MaxTimeDrift    time.Duration
	RequiredHeaders []string
}

// DefaultRequestSigningConfig returns default configuration
func DefaultRequestSigningConfig() RequestSigningConfig {
	return RequestSigningConfig{
		TimestampHeader: "X-Timestamp",
		SignatureHeader: "X-Signature",
		MaxTimeDrift:    5 * time.Minute,
		RequiredHeaders: []string{"content-type", "x-api-key"},
	}
}

// RequestSigningMiddleware creates middleware for request signature verification
func RequestSigningMiddleware(config RequestSigningConfig) echo.MiddlewareFunc {
	if config.SecretKey == "" {
		panic("RequestSigningMiddleware: SecretKey is required")
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip signing verification for certain endpoints
			if shouldSkipSigning(c.Request().URL.Path) {
				return next(c)
			}

			// Extract timestamp
			timestampStr := c.Request().Header.Get(config.TimestampHeader)
			if timestampStr == "" {
				return echo.NewHTTPError(http.StatusBadRequest, map[string]interface{}{
					"error":   "missing_timestamp",
					"message": fmt.Sprintf("Request must include %s header", config.TimestampHeader),
					"code":    "SIGN_001",
				})
			}

			// Parse timestamp
			timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, map[string]interface{}{
					"error":   "invalid_timestamp",
					"message": "Timestamp must be a valid Unix timestamp",
					"code":    "SIGN_002",
				})
			}

			// Check timestamp drift
			now := time.Now().Unix()
			if abs(now-timestamp) > int64(config.MaxTimeDrift.Seconds()) {
				return echo.NewHTTPError(http.StatusBadRequest, map[string]interface{}{
					"error":   "timestamp_drift",
					"message": fmt.Sprintf("Request timestamp is too old or too far in the future (max drift: %v)", config.MaxTimeDrift),
					"code":    "SIGN_003",
				})
			}

			// Extract signature
			signature := c.Request().Header.Get(config.SignatureHeader)
			if signature == "" {
				return echo.NewHTTPError(http.StatusBadRequest, map[string]interface{}{
					"error":   "missing_signature",
					"message": fmt.Sprintf("Request must include %s header", config.SignatureHeader),
					"code":    "SIGN_004",
				})
			}

			// Read request body
			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, map[string]interface{}{
					"error":   "body_read_error",
					"message": "Failed to read request body",
					"code":    "SIGN_005",
				})
			}

			// Restore body for next handlers
			c.Request().Body = io.NopCloser(strings.NewReader(string(body)))

			// Generate expected signature
			expectedSignature := generateSignature(
				config.SecretKey,
				c.Request().Method,
				c.Request().URL.Path,
				timestampStr,
				string(body),
				extractRequiredHeaders(c.Request(), config.RequiredHeaders),
			)

			// Verify signature
			if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]interface{}{
					"error":   "invalid_signature",
					"message": "Request signature verification failed",
					"code":    "SIGN_006",
				})
			}

			// Store signature verification status in context
			c.Set("signature_verified", true)
			c.Set("request_timestamp", timestamp)

			return next(c)
		}
	}
}

// generateSignature creates HMAC-SHA256 signature for the request
func generateSignature(secretKey, method, path, timestamp, body string, headers map[string]string) string {
	// Create canonical string
	var canonicalParts []string
	canonicalParts = append(canonicalParts, method)
	canonicalParts = append(canonicalParts, path)
	canonicalParts = append(canonicalParts, timestamp)
	canonicalParts = append(canonicalParts, body)

	// Add headers in sorted order
	for key, value := range headers {
		canonicalParts = append(canonicalParts, fmt.Sprintf("%s:%s", key, value))
	}

	canonicalString := strings.Join(canonicalParts, "\n")

	// Generate HMAC-SHA256
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(canonicalString))
	return hex.EncodeToString(h.Sum(nil))
}

// extractRequiredHeaders extracts specified headers from request
func extractRequiredHeaders(req *http.Request, requiredHeaders []string) map[string]string {
	headers := make(map[string]string)
	for _, header := range requiredHeaders {
		value := req.Header.Get(header)
		if value != "" {
			headers[strings.ToLower(header)] = value
		}
	}
	return headers
}

// shouldSkipSigning determines if signature verification should be skipped
func shouldSkipSigning(path string) bool {
	skipPaths := []string{
		"/health",
		"/metrics",
		"/api/info",
		"/api/nutrition/analyze", // Public endpoint
	}

	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// abs returns absolute value of int64
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// SignatureRequiredMiddleware ensures that sensitive endpoints require signature verification
func SignatureRequiredMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if this endpoint requires signature verification
			if requiresSignature(c.Request().URL.Path, c.Request().Method) {
				verified, ok := c.Get("signature_verified").(bool)
				if !ok || !verified {
					return echo.NewHTTPError(http.StatusUnauthorized, map[string]interface{}{
						"error":   "signature_required",
						"message": "This endpoint requires request signature verification",
						"code":    "SIGN_007",
					})
				}
			}

			return next(c)
		}
	}
}

// requiresSignature determines if an endpoint requires signature verification
func requiresSignature(path, method string) bool {
	// Define sensitive endpoints that require signature verification
	sensitiveEndpoints := map[string][]string{
		"/api/v1/users":         {"POST", "PUT", "DELETE"},
		"/api/v1/admin":         {"GET", "POST", "PUT", "DELETE"},
		"/api/v1/api-keys":      {"GET", "POST", "DELETE"},
		"/api/v1/meal-plans":    {"POST", "PUT", "DELETE"},
		"/api/v1/workout-plans": {"POST", "PUT", "DELETE"},
	}

	for endpointPath, methods := range sensitiveEndpoints {
		if strings.HasPrefix(path, endpointPath) {
			for _, m := range methods {
				if method == m {
					return true
				}
			}
		}
	}

	return false
}

// GenerateClientSignature is a helper function for clients to generate signatures
func GenerateClientSignature(secretKey, method, path, timestamp, body string, headers map[string]string) string {
	return generateSignature(secretKey, method, path, timestamp, body, headers)
}
