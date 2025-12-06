package utils

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// AppError represents a custom application error
type AppError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

// NewError creates a new application error
func NewError(code int, message string, details interface{}) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Predefined error types
var (
	ErrBadRequest         = NewError(http.StatusBadRequest, "Bad request", nil)
	ErrUnauthorized       = NewError(http.StatusUnauthorized, "Unauthorized", nil)
	ErrForbidden          = NewError(http.StatusForbidden, "Forbidden", nil)
	ErrNotFound           = NewError(http.StatusNotFound, "Resource not found", nil)
	ErrConflict           = NewError(http.StatusConflict, "Resource conflict", nil)
	ErrUnprocessable      = NewError(http.StatusUnprocessableEntity, "Unprocessable entity", nil)
	ErrTooManyRequests    = NewError(http.StatusTooManyRequests, "Too many requests", nil)
	ErrInternalServer     = NewError(http.StatusInternalServerError, "Internal server error", nil)
	ErrServiceUnavailable = NewError(http.StatusServiceUnavailable, "Service unavailable", nil)
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string, value interface{}) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	}
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (ve *ValidationErrors) Error() string {
	return fmt.Sprintf("validation failed: %d errors", len(ve.Errors))
}

// Add adds a validation error
func (ve *ValidationErrors) Add(field, message string, value interface{}) {
	ve.Errors = append(ve.Errors, ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// HasErrors returns true if there are validation errors
func (ve *ValidationErrors) HasErrors() bool {
	return len(ve.Errors) > 0
}

// ToAppError converts validation errors to AppError
func (ve *ValidationErrors) ToAppError() *AppError {
	if len(ve.Errors) == 0 {
		return nil
	}
	return NewError(http.StatusUnprocessableEntity, "Validation failed", ve.Errors)
}

// HandleError handles errors and returns appropriate HTTP responses
func HandleError(c echo.Context, err error) error {
	if appErr, ok := err.(*AppError); ok {
		return c.JSON(appErr.Code, map[string]interface{}{
			"status":  "error",
			"error":   appErr.Message,
			"details": appErr.Details,
			"meta":    getDefaultMeta(c),
		})
	}

	if valErr, ok := err.(*ValidationErrors); ok && valErr.HasErrors() {
		return c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
			"status":  "error",
			"error":   "Validation failed",
			"details": valErr.Errors,
			"meta":    getDefaultMeta(c),
		})
	}

	if echoErr, ok := err.(*echo.HTTPError); ok {
		return c.JSON(echoErr.Code, map[string]interface{}{
			"status": "error",
			"error":  echoErr.Message,
			"meta":   getDefaultMeta(c),
		})
	}

	// Default to 500 internal server error
	return c.JSON(http.StatusInternalServerError, map[string]interface{}{
		"status": "error",
		"error":  "Internal server error",
		"meta":   getDefaultMeta(c),
	})
}

// ErrorWithCode creates an error with specific HTTP status code
func ErrorWithCode(code int, message string) *AppError {
	return NewError(code, message, nil)
}

// ErrorWithDetails creates an error with details
func ErrorWithDetails(code int, message string, details interface{}) *AppError {
	return NewError(code, message, details)
}

// DatabaseError creates a database-related error
func DatabaseError(message string) *AppError {
	return NewError(http.StatusInternalServerError, fmt.Sprintf("Database error: %s", message), nil)
}

// ValidationErrorWithField creates a field-specific validation error
func ValidationErrorWithField(field, message string) *AppError {
	ve := &ValidationErrors{}
	ve.Add(field, message, nil)
	return NewError(http.StatusUnprocessableEntity, "Validation failed", ve.Errors)
}

// NotFoundError creates a not found error with optional entity name
func NotFoundError(entity string) *AppError {
	if entity == "" {
		return ErrNotFound
	}
	return NewError(http.StatusNotFound, fmt.Sprintf("%s not found", entity), nil)
}

// ConflictError creates a conflict error with optional details
func ConflictError(message string, details interface{}) *AppError {
	if message == "" {
		return ErrConflict
	}
	return NewError(http.StatusConflict, message, details)
}

// RateLimitError creates a rate limit error
func RateLimitError(limit int, window string) *AppError {
	message := fmt.Sprintf("Rate limit exceeded. Maximum %d requests per %s", limit, window)
	return NewError(http.StatusTooManyRequests, message, map[string]interface{}{
		"limit":  limit,
		"window": window,
	})
}

// ServiceError creates a service-related error
func ServiceError(service, message string) *AppError {
	fullMessage := fmt.Sprintf("Service error (%s): %s", service, message)
	return NewError(http.StatusServiceUnavailable, fullMessage, nil)
}

// TimeoutError creates a timeout error
func TimeoutError(operation string) *AppError {
	message := fmt.Sprintf("Operation timed out: %s", operation)
	return NewError(http.StatusRequestTimeout, message, nil)
}
