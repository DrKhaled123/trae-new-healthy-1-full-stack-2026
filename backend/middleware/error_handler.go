package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"nutrition-platform/errors"

	"github.com/labstack/echo/v4"
)

// ErrorHandlerConfig holds configuration for error handling
type ErrorHandlerConfig struct {
	Logger         echo.Logger
	DebugMode      bool
	LogStackTrace  bool
	SecurityLogger SecurityLogger
	NotifyOnPanic  bool
	SanitizeErrors bool
}

// SecurityLogger interface for logging security events
type SecurityLogger interface {
	LogSecurityEvent(event *errors.SecurityEvent)
}

// DefaultErrorHandlerConfig returns default configuration
func DefaultErrorHandlerConfig() ErrorHandlerConfig {
	return ErrorHandlerConfig{
		DebugMode:      false,
		LogStackTrace:  true,
		NotifyOnPanic:  true,
		SanitizeErrors: true,
	}
}

// ErrorHandler returns an Echo error handler middleware
func ErrorHandler(config ErrorHandlerConfig) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		start := time.Now()
		defer func() {
			if config.SecurityLogger != nil {
				event := createSecurityEvent(err, c, time.Since(start))
				config.SecurityLogger.LogSecurityEvent(event)
			}
		}()

		// Don't handle if response already sent
		if c.Response().Committed {
			return
		}

		var apiErr *errors.APIError
		var httpErr *echo.HTTPError
		var validationErr *errors.ValidationErrors

		// Handle different error types
		switch e := err.(type) {
		case *errors.APIError:
			apiErr = e.WithContext(c)
		case *echo.HTTPError:
			httpErr = e
			apiErr = convertHTTPError(httpErr)
		case *errors.ValidationErrors:
			validationErr = e
			apiErr = &errors.APIError{
				Code:      errors.ErrInvalidInput,
				Message:   "Validation failed",
				Details:   validationErr.Error(),
				Timestamp: time.Now().UTC(),
			}
		default:
			// Handle unexpected errors
			apiErr = handleUnexpectedError(err, config)
		}

		// Add request context
		apiErr = apiErr.WithContext(c)

		// Log the error
		logError(err, apiErr, c, config)

		// Sanitize error for production
		if config.SanitizeErrors && !config.DebugMode {
			apiErr = sanitizeError(apiErr)
		}

		// Create error response
		errorResponse := errors.NewErrorResponse(apiErr)

		// Add validation errors if present
		if validationErr != nil {
			errorResponse.Error.Details = fmt.Sprintf("Validation errors: %d", len(validationErr.Errors))
			// In debug mode, include validation details
			if config.DebugMode {
				validationData, _ := json.Marshal(validationErr.Errors)
				errorResponse.Error.Details = string(validationData)
			}
		}

		// Set security headers
		setSecurityHeaders(c)

		// Send error response
		statusCode := apiErr.HTTPStatus()
		if err := c.JSON(statusCode, errorResponse); err != nil {
			if config.Logger != nil {
				config.Logger.Error("Failed to send error response:", err)
			}
		}
	}
}

// convertHTTPError converts Echo HTTP error to API error
func convertHTTPError(httpErr *echo.HTTPError) *errors.APIError {
	var code errors.ErrorCode
	var message string

	switch httpErr.Code {
	case http.StatusBadRequest:
		code = errors.ErrInvalidInput
		message = "Bad request"
	case http.StatusUnauthorized:
		code = errors.ErrInvalidAPIKey
		message = "Unauthorized"
	case http.StatusForbidden:
		code = errors.ErrInsufficientScope
		message = "Forbidden"
	case http.StatusNotFound:
		code = errors.ErrResourceNotFound
		message = "Not found"
	case http.StatusTooManyRequests:
		code = errors.ErrRateLimitExceeded
		message = "Too many requests"
	case http.StatusInternalServerError:
		code = errors.ErrInternalServer
		message = "Internal server error"
	default:
		code = errors.ErrInternalServer
		message = "Unknown error"
	}

	details := ""
	if httpErr.Message != nil {
		details = fmt.Sprintf("%v", httpErr.Message)
	}

	return errors.NewAPIError(code, message, details)
}

// handleUnexpectedError handles unexpected errors
func handleUnexpectedError(err error, config ErrorHandlerConfig) *errors.APIError {
	// Log stack trace for debugging
	if config.LogStackTrace {
		log.Printf("Unexpected error: %v\nStack trace:\n%s", err, debug.Stack())
	}

	// Check for specific error patterns
	errorMsg := err.Error()
	switch {
	case strings.Contains(errorMsg, "connection refused"):
		return errors.NewAPIError(errors.ErrDatabaseConnection, "Database connection failed")
	case strings.Contains(errorMsg, "timeout"):
		return errors.NewAPIError(errors.ErrTimeout, "Request timeout")
	case strings.Contains(errorMsg, "context deadline exceeded"):
		return errors.NewAPIError(errors.ErrTimeout, "Request timeout")
	case strings.Contains(errorMsg, "sql: no rows"):
		return errors.NewAPIError(errors.ErrResourceNotFound, "Resource not found")
	default:
		return errors.NewAPIError(errors.ErrInternalServer, "Internal server error", errorMsg)
	}
}

// sanitizeError removes sensitive information from errors in production
func sanitizeError(apiErr *errors.APIError) *errors.APIError {
	// Create a copy to avoid modifying the original
	sanitized := *apiErr

	// Remove sensitive details for certain error types
	switch apiErr.Code {
	case errors.ErrInternalServer, errors.ErrDatabaseConnection, errors.ErrDatabaseQuery:
		sanitized.Details = "" // Remove internal details
		sanitized.Message = "An internal error occurred"
	case errors.ErrSecurityViolation:
		sanitized.Details = "" // Remove security details
	}

	return &sanitized
}

// logError logs the error with appropriate level
func logError(originalErr error, apiErr *errors.APIError, c echo.Context, config ErrorHandlerConfig) {
	if config.Logger == nil {
		return
	}

	// Determine log level based on error type
	logLevel := "ERROR"
	switch apiErr.Code {
	case errors.ErrInvalidInput, errors.ErrMissingParameter, errors.ErrResourceNotFound:
		logLevel = "WARN"
	case errors.ErrSecurityViolation, errors.ErrSuspiciousActivity:
		logLevel = "CRITICAL"
	}

	// Create log entry
	logEntry := map[string]interface{}{
		"level":      logLevel,
		"error_code": apiErr.Code,
		"message":    apiErr.Message,
		"method":     c.Request().Method,
		"path":       c.Request().URL.Path,
		"ip":         c.RealIP(),
		"user_agent": c.Request().UserAgent(),
		"request_id": c.Response().Header().Get(echo.HeaderXRequestID),
		"timestamp":  time.Now().UTC(),
	}

	// Add details in debug mode
	if config.DebugMode && apiErr.Details != "" {
		logEntry["details"] = apiErr.Details
	}

	// Add original error in debug mode
	if config.DebugMode && originalErr != nil {
		logEntry["original_error"] = originalErr.Error()
	}

	// Log based on level
	switch logLevel {
	case "WARN":
		config.Logger.Warn(logEntry)
	case "CRITICAL":
		config.Logger.Error(logEntry)
	default:
		config.Logger.Error(logEntry)
	}
}

// createSecurityEvent creates a security event from an error
func createSecurityEvent(err error, c echo.Context, responseTime time.Duration) *errors.SecurityEvent {
	eventType := "error"
	severity := "medium"
	message := "Request error occurred"

	// Determine event type and severity based on error
	if apiErr, ok := err.(*errors.APIError); ok {
		switch apiErr.Code {
		case errors.ErrSecurityViolation, errors.ErrSuspiciousActivity:
			eventType = "security_violation"
			severity = "high"
			message = "Security violation detected"
		case errors.ErrInvalidAPIKey, errors.ErrExpiredAPIKey, errors.ErrRevokedAPIKey:
			eventType = "authentication_failure"
			severity = "medium"
			message = "Authentication failure"
		case errors.ErrRateLimitExceeded:
			eventType = "rate_limit_exceeded"
			severity = "low"
			message = "Rate limit exceeded"
		}
	}

	event := errors.NewSecurityEvent(eventType, severity, message)
	event.WithContext(c)
	event.StatusCode = c.Response().Status
	event.ResponseTime = responseTime

	// Add API key ID if available
	if apiKeyID := c.Get("api_key_id"); apiKeyID != nil {
		event.APIKeyID = fmt.Sprintf("%v", apiKeyID)
	}

	return event
}

// setSecurityHeaders sets security headers on error responses
func setSecurityHeaders(c echo.Context) {
	header := c.Response().Header()
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
	header.Set("X-XSS-Protection", "1; mode=block")
	header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	header.Set("Pragma", "no-cache")
	header.Set("Expires", "0")
}

// PanicRecovery returns a middleware that recovers from panics
func PanicRecovery(config ErrorHandlerConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					// Log panic with stack trace
					if config.Logger != nil {
						config.Logger.Error(map[string]interface{}{
							"level":       "CRITICAL",
							"type":        "panic",
							"panic":       r,
							"stack_trace": string(debug.Stack()),
							"method":      c.Request().Method,
							"path":        c.Request().URL.Path,
							"ip":          c.RealIP(),
							"timestamp":   time.Now().UTC(),
						})
					}

					// Create security event for panic
					if config.SecurityLogger != nil {
						event := errors.NewSecurityEvent("panic", "critical", "Application panic occurred")
						event.WithContext(c)
						event.AddDetail("panic", fmt.Sprintf("%v", r))
						config.SecurityLogger.LogSecurityEvent(event)
					}

					// Create API error for panic
					apiErr := errors.NewAPIError(
						errors.ErrInternalServer,
						"Internal server error",
						fmt.Sprintf("Panic: %v", r),
					).WithContext(c)

					// Sanitize in production
					if config.SanitizeErrors && !config.DebugMode {
						apiErr = sanitizeError(apiErr)
					}

					// Send error response
					errorResponse := errors.NewErrorResponse(apiErr)
					setSecurityHeaders(c)
					c.JSON(http.StatusInternalServerError, errorResponse)
				}
			}()

			return next(c)
		}
	}
}

// RequestValidator validates common request parameters
func RequestValidator() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Validate request size
			if c.Request().ContentLength > 10*1024*1024 { // 10MB limit
				return errors.NewAPIError(
					errors.ErrInvalidInput,
					"Request too large",
					"Maximum request size is 10MB",
				)
			}

			// Validate content type for POST/PUT requests
			method := c.Request().Method
			if method == "POST" || method == "PUT" || method == "PATCH" {
				contentType := c.Request().Header.Get("Content-Type")
				if contentType != "" && !strings.Contains(contentType, "application/json") &&
					!strings.Contains(contentType, "multipart/form-data") &&
					!strings.Contains(contentType, "application/x-www-form-urlencoded") {
					return errors.NewAPIError(
						errors.ErrInvalidFormat,
						"Unsupported content type",
						fmt.Sprintf("Content-Type: %s", contentType),
					)
				}
			}

			return next(c)
		}
	}
}
