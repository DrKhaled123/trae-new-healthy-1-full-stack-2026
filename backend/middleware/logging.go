package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// LoggingConfig defines configuration for request logging
type LoggingConfig struct {
	// SkipPaths defines paths to skip logging
	SkipPaths []string
	// SkipHealth determines if health checks should be skipped
	SkipHealth bool
	// LogRequestBody determines if request bodies should be logged
	LogRequestBody bool
	// LogResponseBody determines if response bodies should be logged
	LogResponseBody bool
	// MaxBodySize defines maximum body size to log (in bytes)
	MaxBodySize int
	// Logger defines custom logger
	Logger *logrus.Logger
}

// DefaultLoggingConfig returns default logging configuration
func DefaultLoggingConfig() *LoggingConfig {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	return &LoggingConfig{
		SkipPaths:       []string{"/health", "/metrics"},
		SkipHealth:      true,
		LogRequestBody:  false, // Don't log sensitive data by default
		LogResponseBody: false,
		MaxBodySize:     1024 * 1024, // 1MB
		Logger:          logger,
	}
}

// RequestLogger creates a request logging middleware
func RequestLogger(config *LoggingConfig) echo.MiddlewareFunc {
	if config == nil {
		config = DefaultLoggingConfig()
	}

	logger := config.Logger
	if logger == nil {
		logger = logrus.New()
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip logging for specified paths
			if shouldSkipLogging(c.Request().URL.Path, config) {
				return next(c)
			}

			start := time.Now()

			// Capture request details
			req := c.Request()
			requestID := c.Response().Header().Get(echo.HeaderXRequestID)
			if requestID == "" {
				requestID = generateRequestID()
				c.Response().Header().Set(echo.HeaderXRequestID, requestID)
			}

			// Read and potentially capture request body
			var requestBody []byte
			if config.LogRequestBody && req.Body != nil {
				requestBody, _ = io.ReadAll(io.LimitReader(req.Body, int64(config.MaxBodySize)))
				req.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}

			// Create custom response writer to capture response
			responseWriter := &responseBodyWriter{
				ResponseWriter: c.Response().Writer,
				body:           &bytes.Buffer{},
			}
			c.Response().Writer = responseWriter

			// Process request
			err := next(c)

			// Calculate duration
			duration := time.Since(start)

			// Log the request
			logEntry := logger.WithFields(logrus.Fields{
				"request_id":     requestID,
				"method":         req.Method,
				"path":           req.URL.Path,
				"query":          req.URL.RawQuery,
				"user_agent":     req.UserAgent(),
				"remote_addr":    c.RealIP(),
				"host":           req.Host,
				"status_code":    c.Response().Status,
				"duration_ms":    duration.Milliseconds(),
				"content_type":   c.Response().Header().Get("Content-Type"),
				"content_length": c.Response().Size,
			})

			// Add authentication info if available
			if user := c.Get("user"); user != nil {
				logEntry = logEntry.WithField("user_id", user)
			}

			// Add request body if configured (be careful with sensitive data)
			if config.LogRequestBody && len(requestBody) > 0 {
				if len(requestBody) > config.MaxBodySize {
					logEntry = logEntry.WithField("request_body", "[BODY TOO LARGE - TRUNCATED]")
				} else {
					logEntry = logEntry.WithField("request_body", string(requestBody))
				}
			}

			// Add response body if configured
			if config.LogResponseBody && responseWriter.body.Len() > 0 {
				responseBody := responseWriter.body.Bytes()
				if len(responseBody) > config.MaxBodySize {
					logEntry = logEntry.WithField("response_body", "[BODY TOO LARGE - TRUNCATED]")
				} else {
					logEntry = logEntry.WithField("response_body", string(responseBody))
				}
			}

			// Add error information if available
			if err != nil {
				logEntry = logEntry.WithFields(logrus.Fields{
					"error":      err.Error(),
					"error_type": getErrorType(err),
				})
				logEntry.Error("Request failed")
			} else {
				logEntry.Info("Request completed")
			}

			return err
		}
	}
}

// responseBodyWriter wraps http.ResponseWriter to capture response body
type responseBodyWriter struct {
	http.ResponseWriter
	body *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

// shouldSkipLogging determines if logging should be skipped for this path
func shouldSkipLogging(path string, config *LoggingConfig) bool {
	// Skip health checks if configured
	if config.SkipHealth && (path == "/health" || path == "/health/") {
		return true
	}

	// Skip configured paths
	for _, skipPath := range config.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}

	return false
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().Nanosecond()%len(charset)]
	}
	return string(b)
}

// getErrorType categorizes the error type
func getErrorType(err error) string {
	if err == nil {
		return "none"
	}

	switch {
	case strings.Contains(err.Error(), "validation"):
		return "validation"
	case strings.Contains(err.Error(), "timeout"):
		return "timeout"
	case strings.Contains(err.Error(), "database"):
		return "database"
	case strings.Contains(err.Error(), "network"):
		return "network"
	case strings.Contains(err.Error(), "permission"):
		return "permission"
	default:
		return "unknown"
	}
}

// SlowRequestLogger creates a middleware that logs slow requests
func SlowRequestLogger(threshold time.Duration, logger *logrus.Logger) echo.MiddlewareFunc {
	if logger == nil {
		logger = logrus.New()
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start)
			if duration > threshold {
				logger.WithFields(logrus.Fields{
					"method":            c.Request().Method,
					"path":              c.Request().URL.Path,
					"duration_ms":       duration.Milliseconds(),
					"status_code":       c.Response().Status,
					"remote_addr":       c.RealIP(),
					"slow_threshold_ms": threshold.Milliseconds(),
				}).Warn("Slow request detected")
			}

			return err
		}
	}
}

// ErrorLogger creates a middleware that logs errors with detailed context
func ErrorLogger(logger *logrus.Logger) echo.MiddlewareFunc {
	if logger == nil {
		logger = logrus.New()
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)

			if err != nil {
				logger.WithFields(logrus.Fields{
					"error":       err.Error(),
					"error_type":  getErrorType(err),
					"method":      c.Request().Method,
					"path":        c.Request().URL.Path,
					"query":       c.Request().URL.RawQuery,
					"user_agent":  c.Request().UserAgent(),
					"remote_addr": c.RealIP(),
					"status_code": c.Response().Status,
					"request_id":  c.Response().Header().Get(echo.HeaderXRequestID),
				}).Error("Request failed with error")
			}

			return err
		}
	}
}

// SecurityEventLogger creates a middleware that logs security-related events
func SecurityEventLogger(logger *logrus.Logger) echo.MiddlewareFunc {
	if logger == nil {
		logger = logrus.New()
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()

			// Log suspicious patterns
			suspiciousPatterns := []string{
				"../",     // Path traversal
				"<script", // XSS
				"union",   // SQL injection
				"exec",    // Command injection
				"eval",    // Code injection
			}

			for _, pattern := range suspiciousPatterns {
				if strings.Contains(req.URL.RawQuery, pattern) ||
					strings.Contains(req.URL.Path, pattern) {
					logger.WithFields(logrus.Fields{
						"alert_type":  "suspicious_pattern",
						"pattern":     pattern,
						"method":      req.Method,
						"path":        req.URL.Path,
						"query":       req.URL.RawQuery,
						"user_agent":  req.UserAgent(),
						"remote_addr": c.RealIP(),
						"request_id":  c.Response().Header().Get(echo.HeaderXRequestID),
					}).Warn("Suspicious request pattern detected")
				}
			}

			err := next(c)

			// Log authentication failures
			if err != nil && c.Response().Status == 401 {
				logger.WithFields(logrus.Fields{
					"alert_type":  "authentication_failure",
					"error":       err.Error(),
					"method":      req.Method,
					"path":        req.URL.Path,
					"remote_addr": c.RealIP(),
					"user_agent":  req.UserAgent(),
					"request_id":  c.Response().Header().Get(echo.HeaderXRequestID),
				}).Warn("Authentication failed")
			}

			// Log authorization failures
			if err != nil && c.Response().Status == 403 {
				logger.WithFields(logrus.Fields{
					"alert_type":  "authorization_failure",
					"error":       err.Error(),
					"method":      req.Method,
					"path":        req.URL.Path,
					"remote_addr": c.RealIP(),
					"user_agent":  req.UserAgent(),
					"request_id":  c.Response().Header().Get(echo.HeaderXRequestID),
				}).Warn("Authorization failed")
			}

			return err
		}
	}
}

// MetricsLogger creates a middleware that logs basic metrics
func MetricsLogger(logger *logrus.Logger) echo.MiddlewareFunc {
	if logger == nil {
		logger = logrus.New()
	}

	var requestCount int64
	var errorCount int64
	var totalDuration time.Duration

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start)
			requestCount++
			totalDuration += duration

			if err != nil {
				errorCount++
			}

			// Log metrics every 100 requests
			if requestCount%100 == 0 {
				logger.WithFields(logrus.Fields{
					"total_requests":  requestCount,
					"total_errors":    errorCount,
					"avg_duration_ms": float64(totalDuration.Nanoseconds()) / float64(requestCount) / 1000000,
					"error_rate":      float64(errorCount) / float64(requestCount) * 100,
				}).Info("API metrics")
			}

			return err
		}
	}
}

// StructuredLogger creates a middleware that logs in structured format
func StructuredLogger(config *LoggingConfig) echo.MiddlewareFunc {
	return RequestLogger(config)
}

// DevelopmentLogger creates a middleware for development environment
func DevelopmentLogger() echo.MiddlewareFunc {
	config := &LoggingConfig{
		SkipPaths:       []string{"/health"},
		SkipHealth:      false,
		LogRequestBody:  true,
		LogResponseBody: true,
		MaxBodySize:     1024, // Smaller for dev
		Logger:          nil,
	}

	// Use text formatter for development
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logger.SetLevel(logrus.DebugLevel)
	config.Logger = logger

	return RequestLogger(config)
}

// ProductionLogger creates a middleware for production environment
func ProductionLogger() echo.MiddlewareFunc {
	config := &LoggingConfig{
		SkipPaths:       []string{"/health", "/metrics", "/favicon.ico"},
		SkipHealth:      true,
		LogRequestBody:  false, // Don't log sensitive data in production
		LogResponseBody: false,
		MaxBodySize:     512 * 1024, // 512KB
		Logger:          nil,
	}

	// Use JSON formatter for production
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)
	config.Logger = logger

	return RequestLogger(config)
}
