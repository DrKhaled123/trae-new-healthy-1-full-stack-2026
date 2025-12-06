package errors

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// ErrorCode represents different types of errors
type ErrorCode string

const (
	// Authentication errors
	ErrInvalidAPIKey     ErrorCode = "INVALID_API_KEY"
	ErrExpiredAPIKey     ErrorCode = "EXPIRED_API_KEY"
	ErrRevokedAPIKey     ErrorCode = "REVOKED_API_KEY"
	ErrMissingAPIKey     ErrorCode = "MISSING_API_KEY"
	ErrInsufficientScope ErrorCode = "INSUFFICIENT_SCOPE"

	// Rate limiting errors
	ErrRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrQuotaExceeded     ErrorCode = "QUOTA_EXCEEDED"

	// Validation errors
	ErrInvalidInput     ErrorCode = "INVALID_INPUT"
	ErrMissingParameter ErrorCode = "MISSING_PARAMETER"
	ErrInvalidFormat    ErrorCode = "INVALID_FORMAT"
	ErrInvalidRange     ErrorCode = "INVALID_RANGE"

	// Resource errors
	ErrResourceNotFound ErrorCode = "RESOURCE_NOT_FOUND"
	ErrResourceExists   ErrorCode = "RESOURCE_EXISTS"
	ErrResourceLocked   ErrorCode = "RESOURCE_LOCKED"

	// Database errors
	ErrDatabaseConnection ErrorCode = "DATABASE_CONNECTION"
	ErrDatabaseQuery      ErrorCode = "DATABASE_QUERY"
	ErrDatabaseTimeout    ErrorCode = "DATABASE_TIMEOUT"

	// Security errors
	ErrSecurityViolation  ErrorCode = "SECURITY_VIOLATION"
	ErrSuspiciousActivity ErrorCode = "SUSPICIOUS_ACTIVITY"
	ErrIPBlocked          ErrorCode = "IP_BLOCKED"

	// Internal errors
	ErrInternalServer     ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrTimeout            ErrorCode = "TIMEOUT"
)

// APIError represents a structured API error
type APIError struct {
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id,omitempty"`
	Path      string    `json:"path,omitempty"`
	Method    string    `json:"method,omitempty"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// HTTPStatus returns the appropriate HTTP status code for the error
func (e *APIError) HTTPStatus() int {
	switch e.Code {
	case ErrInvalidAPIKey, ErrExpiredAPIKey, ErrRevokedAPIKey, ErrMissingAPIKey:
		return http.StatusUnauthorized
	case ErrInsufficientScope:
		return http.StatusForbidden
	case ErrRateLimitExceeded, ErrQuotaExceeded:
		return http.StatusTooManyRequests
	case ErrInvalidInput, ErrMissingParameter, ErrInvalidFormat, ErrInvalidRange:
		return http.StatusBadRequest
	case ErrResourceNotFound:
		return http.StatusNotFound
	case ErrResourceExists:
		return http.StatusConflict
	case ErrResourceLocked:
		return http.StatusLocked
	case ErrSecurityViolation, ErrSuspiciousActivity, ErrIPBlocked:
		return http.StatusForbidden
	case ErrServiceUnavailable:
		return http.StatusServiceUnavailable
	case ErrTimeout:
		return http.StatusRequestTimeout
	default:
		return http.StatusInternalServerError
	}
}

// NewAPIError creates a new API error
func NewAPIError(code ErrorCode, message string, details ...string) *APIError {
	err := &APIError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// WithContext adds request context to the error
func (e *APIError) WithContext(c echo.Context) *APIError {
	e.RequestID = c.Response().Header().Get(echo.HeaderXRequestID)
	e.Path = c.Request().URL.Path
	e.Method = c.Request().Method
	return e
}

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Error     *APIError `json:"error"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(err *APIError) *ErrorResponse {
	return &ErrorResponse{
		Error:     err,
		Success:   false,
		Timestamp: time.Now().UTC(),
	}
}

// Common error constructors
func ErrInvalidAPIKeyError(details string) *APIError {
	return NewAPIError(ErrInvalidAPIKey, "Invalid API key provided", details)
}

func ErrExpiredAPIKeyError() *APIError {
	return NewAPIError(ErrExpiredAPIKey, "API key has expired")
}

func ErrRevokedAPIKeyError() *APIError {
	return NewAPIError(ErrRevokedAPIKey, "API key has been revoked")
}

func ErrMissingAPIKeyError() *APIError {
	return NewAPIError(ErrMissingAPIKey, "API key is required")
}

func ErrInsufficientScopeError(required string) *APIError {
	return NewAPIError(ErrInsufficientScope, "Insufficient permissions", fmt.Sprintf("Required scope: %s", required))
}

func ErrRateLimitExceededError(limit int, window string) *APIError {
	return NewAPIError(ErrRateLimitExceeded, "Rate limit exceeded", fmt.Sprintf("Limit: %d requests per %s", limit, window))
}

func ErrInvalidInputError(field string) *APIError {
	return NewAPIError(ErrInvalidInput, "Invalid input provided", fmt.Sprintf("Field: %s", field))
}

func ErrResourceNotFoundError(resource string) *APIError {
	return NewAPIError(ErrResourceNotFound, "Resource not found", fmt.Sprintf("Resource: %s", resource))
}

func ErrInternalServerError(details string) *APIError {
	return NewAPIError(ErrInternalServer, "Internal server error", details)
}

func ErrSecurityViolationError(violation string) *APIError {
	return NewAPIError(ErrSecurityViolation, "Security violation detected", violation)
}

// ValidationError represents field validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// Error implements the error interface
func (v *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", v.Field, v.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// Error implements the error interface
func (v *ValidationErrors) Error() string {
	return fmt.Sprintf("Validation failed: %d errors", len(v.Errors))
}

// Add adds a validation error
func (v *ValidationErrors) Add(field, message, value string) {
	v.Errors = append(v.Errors, ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// HasErrors returns true if there are validation errors
func (v *ValidationErrors) HasErrors() bool {
	return len(v.Errors) > 0
}

// NewValidationErrors creates a new validation errors collection
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		Errors: make([]ValidationError, 0),
	}
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	Type         string            `json:"type"`
	Severity     string            `json:"severity"`
	Message      string            `json:"message"`
	Details      map[string]string `json:"details"`
	Timestamp    time.Time         `json:"timestamp"`
	IPAddress    string            `json:"ip_address"`
	UserAgent    string            `json:"user_agent"`
	APIKeyID     string            `json:"api_key_id,omitempty"`
	Endpoint     string            `json:"endpoint"`
	Method       string            `json:"method"`
	StatusCode   int               `json:"status_code"`
	ResponseTime time.Duration     `json:"response_time"`
}

// NewSecurityEvent creates a new security event
func NewSecurityEvent(eventType, severity, message string) *SecurityEvent {
	return &SecurityEvent{
		Type:      eventType,
		Severity:  severity,
		Message:   message,
		Details:   make(map[string]string),
		Timestamp: time.Now().UTC(),
	}
}

// WithContext adds request context to the security event
func (s *SecurityEvent) WithContext(c echo.Context) *SecurityEvent {
	s.IPAddress = c.RealIP()
	s.UserAgent = c.Request().UserAgent()
	s.Endpoint = c.Request().URL.Path
	s.Method = c.Request().Method
	return s
}

// AddDetail adds a detail to the security event
func (s *SecurityEvent) AddDetail(key, value string) *SecurityEvent {
	s.Details[key] = value
	return s
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
