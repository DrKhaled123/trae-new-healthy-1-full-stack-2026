package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nutrition-platform/errors"
	"nutrition-platform/validation"

	"github.com/labstack/echo/v4"
)

// BackendValidationMiddleware provides comprehensive request/response validation
type BackendValidationMiddleware struct {
	validator          *validation.BackendValidator
	config             *ValidationMiddlewareConfig
	securityLogger     SecurityLogger
	performanceTracker PerformanceTracker
}

// ValidationMiddlewareConfig holds middleware configuration
type ValidationMiddlewareConfig struct {
	EnableRequestValidation   bool
	EnableResponseValidation  bool
	EnableSecurityChecks      bool
	EnablePerformanceTracking bool
	EnableLogging             bool
	MaxRequestSize            int64
	AllowedContentTypes       []string
	BlockedUserAgents         []string
	BlockedIPs                []string
	RateLimitEnabled          bool
	RequestTimeout            time.Duration
	ValidationTimeout         time.Duration
}

// SecurityLogger interface for logging security events
type SecurityLogger interface {
	LogSecurityEvent(event *errors.SecurityEvent)
}

// PerformanceTracker interface for tracking performance metrics
type PerformanceTracker interface {
	TrackRequest(endpoint string, method string, duration time.Duration, statusCode int)
	TrackValidation(endpoint string, method string, duration time.Duration, valid bool)
}

// DefaultValidationMiddlewareConfig returns default configuration
func DefaultValidationMiddlewareConfig() *ValidationMiddlewareConfig {
	return &ValidationMiddlewareConfig{
		EnableRequestValidation:   true,
		EnableResponseValidation:  true,
		EnableSecurityChecks:      true,
		EnablePerformanceTracking: true,
		EnableLogging:             true,
		MaxRequestSize:            10 * 1024 * 1024, // 10MB
		AllowedContentTypes:       []string{"application/json", "application/xml", "multipart/form-data"},
		BlockedUserAgents:         []string{"bot", "crawler", "spider", "scanner"},
		RateLimitEnabled:          true,
		RequestTimeout:            30 * time.Second,
		ValidationTimeout:         5 * time.Second,
	}
}

// NewBackendValidationMiddleware creates new validation middleware
func NewBackendValidationMiddleware(validator *validation.BackendValidator, config *ValidationMiddlewareConfig) echo.MiddlewareFunc {
	if config == nil {
		config = DefaultValidationMiddlewareConfig()
	}

	m := &BackendValidationMiddleware{
		validator: validator,
		config:    config,
	}

	return m.Middleware()
}

// Middleware returns the Echo middleware function
func (m *BackendValidationMiddleware) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			requestID := c.Response().Header().Get(echo.HeaderXRequestID)

			// Set up validation context
			validationCtx, cancel := context.WithTimeout(c.Request().Context(), m.config.ValidationTimeout)
			defer cancel()

			// Perform request validation
			if m.config.EnableRequestValidation {
				if err := m.validateRequest(c, validationCtx); err != nil {
					m.logSecurityEvent("request_validation_failed", "medium", err.Error(), c)
					return err
				}
			}

			// Perform security checks
			if m.config.EnableSecurityChecks {
				if err := m.performSecurityChecks(c, validationCtx); err != nil {
					m.logSecurityEvent("security_check_failed", "high", err.Error(), c)
					return err
				}
			}

			// Create wrapped response writer to capture response
			if m.config.EnableResponseValidation {
				c.Response().Before(func() {
					// Pre-response validation
					m.validatePreResponse(c)
				})
			}

			// Execute the request
			err := next(c)

			// Track performance
			if m.config.EnablePerformanceTracking {
				duration := time.Since(start)
				m.trackPerformance(c, duration, c.Response().Status)
			}

			// Perform response validation
			if m.config.EnableResponseValidation && err == nil {
				if validationErr := m.validateResponse(c, validationCtx); validationErr != nil {
					m.logSecurityEvent("response_validation_failed", "medium", validationErr.Error(), c)
					// Don't return error if request was successful, just log it
				}
			}

			return err
		}
	}
}

// validateRequest validates incoming request
func (m *BackendValidationMiddleware) validateRequest(c echo.Context, ctx context.Context) error {
	// Validate request size
	if c.Request().ContentLength > m.config.MaxRequestSize {
		return errors.ErrInvalidInputError("Request size exceeds maximum allowed")
	}

	// Validate content type
	contentType := c.Request().Header.Get("Content-Type")
	if !m.isAllowedContentType(contentType) {
		return errors.ErrInvalidFormatError("Content-Type not allowed")
	}

	// Validate user agent
	userAgent := c.Request().UserAgent()
	if m.isBlockedUserAgent(userAgent) {
		return errors.ErrSecurityViolationError("Blocked user agent detected")
	}

	// Validate IP address
	ipAddress := c.RealIP()
	if m.isBlockedIP(ipAddress) {
		return errors.ErrIPBlockedError("IP address is blocked")
	}

	// Use backend validator for comprehensive validation
	if err := m.validator.ValidateRequest(c); err != nil {
		return err
	}

	return nil
}

// performSecurityChecks performs security validation
func (m *BackendValidationMiddleware) performSecurityChecks(c echo.Context, ctx context.Context) error {
	// Check for suspicious patterns in URL
	path := c.Request().URL.Path
	if m.containsSuspiciousPatterns(path) {
		return errors.ErrSecurityViolationError("Suspicious URL pattern detected")
	}

	// Check for SQL injection patterns in query parameters
	for key, values := range c.QueryParams() {
		for _, value := range values {
			if m.containsSQLInjectionPatterns(value) {
				return errors.ErrSecurityViolationError(fmt.Sprintf("Potential SQL injection in parameter %s", key))
			}
		}
	}

	// Check for XSS patterns in headers
	for key, values := range c.Request().Header {
		for _, value := range values {
			if m.containsXSSPatterns(value) {
				return errors.ErrSecurityViolationError(fmt.Sprintf("Potential XSS in header %s", key))
			}
		}
	}

	// Validate request rate
	if m.config.RateLimitEnabled {
		if err := m.validateRateLimit(c); err != nil {
			return err
		}
	}

	return nil
}

// validatePreResponse performs pre-response validation
func (m *BackendValidationMiddleware) validatePreResponse(c echo.Context) {
	// Add security headers
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
	c.Response().Header().Set("X-Frame-Options", "DENY")
	c.Response().Header().Set("X-XSS-Protection", "1; mode=block")
	c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Add validation headers
	c.Response().Header().Set("X-Validation-Status", "validated")
	c.Response().Header().Set("X-Validation-Timestamp", time.Now().UTC().Format(time.RFC3339))
}

// validateResponse validates outgoing response
func (m *BackendValidationMiddleware) validateResponse(c echo.Context, ctx context.Context) error {
	// Get response data if available
	responseData := c.Get("response_data")
	if responseData != nil {
		if err := m.validator.ValidateResponse(c, responseData); err != nil {
			return err
		}
	}

	// Validate response headers
	requiredHeaders := []string{
		"Content-Type",
		"X-Content-Type-Options",
	}

	for _, header := range requiredHeaders {
		if c.Response().Header().Get(header) == "" {
			return errors.ErrSecurityViolationError(fmt.Sprintf("Missing required response header: %s", header))
		}
	}

	return nil
}

// Helper validation methods

func (m *BackendValidationMiddleware) isAllowedContentType(contentType string) bool {
	for _, allowed := range m.config.AllowedContentTypes {
		if strings.Contains(contentType, allowed) {
			return true
		}
	}
	return false
}

func (m *BackendValidationMiddleware) isBlockedUserAgent(userAgent string) bool {
	lowerUserAgent := strings.ToLower(userAgent)
	for _, blocked := range m.config.BlockedUserAgents {
		if strings.Contains(lowerUserAgent, blocked) {
			return true
		}
	}
	return false
}

func (m *BackendValidationMiddleware) isBlockedIP(ipAddress string) bool {
	for _, blocked := range m.config.BlockedIPs {
		if ipAddress == blocked {
			return true
		}
	}
	return false
}

func (m *BackendValidationMiddleware) containsSuspiciousPatterns(input string) bool {
	suspiciousPatterns := []string{
		"../", "..\\", "%2e%2e%2f", "%2e%2e%5c",
		"<script", "javascript:", "vbscript:", "onload=", "onerror=",
		"union select", "drop table", "insert into", "delete from",
		"exec(", "xp_", "sp_", "load_file", "into outfile",
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}
	return false
}

func (m *BackendValidationMiddleware) containsSQLInjectionPatterns(input string) bool {
	sqlPatterns := []string{
		"' or '1'='1",
		"' or 1=1--",
		"union select",
		"drop table",
		"insert into",
		"delete from",
		"update set",
		"exec(",
		"xp_",
		"sp_",
		"load_file",
		"into outfile",
		"information_schema",
		"sysobjects",
		"syscolumns",
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range sqlPatterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}
	return false
}

func (m *BackendValidationMiddleware) containsXSSPatterns(input string) bool {
	xssPatterns := []string{
		"<script",
		"javascript:",
		"vbscript:",
		"onload=",
		"onerror=",
		"onclick=",
		"<iframe",
		"<object",
		"<embed",
		"<link",
		"<meta",
		"data:text/html",
		"expression(",
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range xssPatterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}
	return false
}

func (m *BackendValidationMiddleware) validateRateLimit(c echo.Context) error {
	// Basic rate limiting logic
	// In production, this would integrate with a proper rate limiting system
	ipAddress := c.RealIP()
	endpoint := c.Request().URL.Path
	method := c.Request().Method

	// Create a simple rate limit key
	rateLimitKey := fmt.Sprintf("rate_limit:%s:%s:%s", ipAddress, method, endpoint)

	// For demonstration, always allow
	_ = rateLimitKey

	return nil
}

func (m *BackendValidationMiddleware) logSecurityEvent(eventType, severity, message string, c echo.Context) {
	if !m.config.EnableLogging || m.securityLogger == nil {
		return
	}

	event := errors.NewSecurityEvent(eventType, severity, message)
	event.WithContext(c)

	// Add additional context
	event.AddDetail("validation_middleware", "true")
	event.AddDetail("request_path", c.Request().URL.Path)
	event.AddDetail("request_method", c.Request().Method)
	event.AddDetail("user_agent", c.Request().UserAgent())

	m.securityLogger.LogSecurityEvent(event)
}

func (m *BackendValidationMiddleware) trackPerformance(c echo.Context, duration time.Duration, statusCode int) {
	if !m.config.EnablePerformanceTracking || m.performanceTracker == nil {
		return
	}

	endpoint := c.Request().URL.Path
	method := c.Request().Method

	m.performanceTracker.TrackRequest(endpoint, method, duration, statusCode)
}

// SetSecurityLogger sets the security logger
func (m *BackendValidationMiddleware) SetSecurityLogger(logger SecurityLogger) {
	m.securityLogger = logger
}

// SetPerformanceTracker sets the performance tracker
func (m *BackendValidationMiddleware) SetPerformanceTracker(tracker PerformanceTracker) {
	m.performanceTracker = tracker
}

// RequestValidator returns a simple request validator middleware
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
