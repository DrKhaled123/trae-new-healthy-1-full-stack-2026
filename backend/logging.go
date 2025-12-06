// logging.go
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/labstack/echo/v4"
)

func InitLogging() error {
	// Initialize structured logger from structured_logger.go
	return InitStructuredLogger()
}

// RequestLoggerMiddleware logs HTTP requests
func RequestLoggerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			path := c.Request().URL.Path
			method := c.Request().Method
			clientIP := c.RealIP()

			// Get user ID if available (fix the assignment)
			userID := ""
			if id := c.Get("userId"); id != nil {
				if str, ok := id.(string); ok {
					userID = str
				}
			}

			// Get API key from header
			apiKey := c.Request().Header.Get("X-API-Key")

			// Process request
			err := next(c)

			// Calculate latency
			latency := time.Since(start)
			statusCode := c.Response().Status

			// Simple logging format
			logEntry := fmt.Sprintf("[%s] %s %s %d %v client=%s",
				time.Now().Format(time.RFC3339),
				method,
				path,
				statusCode,
				latency,
				clientIP,
			)

			if userID != "" {
				logEntry += fmt.Sprintf(" user=%s", userID)
			}

			if apiKey != "" {
				logEntry += fmt.Sprintf(" api_key=%s...", apiKey[:min(8, len(apiKey))])
			}

			// Log based on status code
			if statusCode >= 500 {
				log.Printf("ERROR: %s", logEntry)
			} else if statusCode >= 400 {
				log.Printf("WARN: %s", logEntry)
			} else if os.Getenv("LOG_LEVEL") != "error" {
				log.Printf("INFO: %s", logEntry)
			}

			return err
		}
	}
}

// ErrorLogger logs errors with context
func ErrorLogger(c echo.Context, err error, message string) {
	log.Printf("ERROR [%s]: %s - %v (client: %s, path: %s)",
		time.Now().Format(time.RFC3339),
		message,
		err,
		c.RealIP(),
		c.Request().URL.Path,
	)
}

// AuditLogger logs security-related events
func AuditLogger(action, resourceType, resourceID, clientIP, userAgent string, userID *string) {
	uid := ""
	if userID != nil {
		uid = *userID
	}

	log.Printf("AUDIT [%s]: action=%s resource=%s:%s client=%s user=%s user_agent=%s",
		time.Now().Format(time.RFC3339),
		action,
		resourceType,
		resourceID,
		clientIP,
		uid,
		userAgent,
	)
}

// MetricsLogger logs performance metrics
func MetricsLogger(metricName string, value float64, tags map[string]string) {
	if os.Getenv("LOG_LEVEL") != "error" {
		tagStr := ""
		for k, v := range tags {
			tagStr += k + "=" + v + " "
		}

		log.Printf("METRICS [%s]: %s=%.2f %s",
			time.Now().Format(time.RFC3339),
			metricName,
			value,
			tagStr,
		)
	}
}

// DatabaseLogger logs database operations
func DatabaseLogger(operation, table string, duration time.Duration, success bool, errorMsg string) {
	if success {
		if os.Getenv("LOG_LEVEL") == "debug" {
			log.Printf("DB [%s]: %s on %s took %v",
				time.Now().Format(time.RFC3339),
				operation,
				table,
				duration,
			)
		}
	} else {
		log.Printf("DB ERROR [%s]: %s on %s failed (%v) - %s",
			time.Now().Format(time.RFC3339),
			operation,
			table,
			duration,
			errorMsg,
		)
	}
}

// SecurityLogger logs security events
func SecurityLogger(eventType, clientIP, userAgent, details string) {
	log.Printf("SECURITY [%s]: %s from %s (%s) - %s",
		time.Now().Format(time.RFC3339),
		eventType,
		clientIP,
		userAgent,
		details,
	)
}

// PerformanceLogger logs performance issues
func PerformanceLogger(slowOperation string, threshold time.Duration, actual time.Duration) {
	log.Printf("PERFORMANCE [%s]: %s exceeded threshold (actual: %v, threshold: %v, ratio: %.2f)",
		time.Now().Format(time.RFC3339),
		slowOperation,
		actual,
		threshold,
		float64(actual)/float64(threshold),
	)
}

// HealthLogger logs health check results
func HealthLogger(service string, status, message string, latency time.Duration) {
	if status == "unhealthy" {
		log.Printf("HEALTH ERROR [%s]: %s - %s (latency: %v)",
			time.Now().Format(time.RFC3339),
			service,
			message,
			latency,
		)
	} else if os.Getenv("LOG_LEVEL") != "error" {
		log.Printf("HEALTH OK [%s]: %s - %s (latency: %v)",
			time.Now().Format(time.RFC3339),
			service,
			message,
			latency,
		)
	}
}

// APILogger logs API usage
func APILogger(method, path, userID, apiKey string, statusCode int, latency time.Duration, clientIP string) {
	if os.Getenv("LOG_LEVEL") == "debug" {
		keySnippet := apiKey[:min(8, len(apiKey))] + "..."
		log.Printf("API [%s]: %s %s %d %v client=%s user=%s key=%s",
			time.Now().Format(time.RFC3339),
			method,
			path,
			statusCode,
			latency,
			clientIP,
			userID,
			keySnippet,
		)
	}
}

// CloseLogger placeholder for compatibility
func CloseLogger() {
	// No cleanup needed for basic logging
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
