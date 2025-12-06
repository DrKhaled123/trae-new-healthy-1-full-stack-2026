package tests

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"nutrition-platform/errors"
)

// ErrorHandlingTestSuite tests error handling functionality
type ErrorHandlingTestSuite struct {
	suite.Suite
	echo         *echo.Echo
	errorHandler *errors.ErrorHandler
	dbHandler    *errors.DatabaseErrorHandler
	authHandler  *errors.AuthErrorHandler
	mockDB       *MockDB
}

// MockDB mocks database operations
type MockDB struct {
	mock.Mock
	*sql.DB
}

func (m *MockDB) PingContext(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDB) Stats() sql.DBStats {
	args := m.Called()
	return args.Get(0).(sql.DBStats)
}

// SetupSuite initializes the test suite
func (suite *ErrorHandlingTestSuite) SetupSuite() {
	suite.echo = echo.New()
	suite.mockDB = &MockDB{}

	// Initialize error handlers
	suite.errorHandler = errors.NewErrorHandler(&errors.ErrorHandlerConfig{
		DebugMode:             true,
		LogStackTrace:         true,
		NotifyOnPanic:         true,
		SanitizeErrors:        false,
		MaxRetries:            3,
		RetryDelay:            time.Millisecond * 100,
		CircuitBreakerEnabled: true,
		HealthCheckInterval:   time.Second * 30,
		AlertThreshold:        5,
		NotificationEnabled:   false,
	})

	suite.authHandler = errors.NewAuthErrorHandler(&errors.AuthConfig{
		JWTSecret:          "test-secret",
		JWTExpiration:      time.Hour * 24,
		APIKeySecret:       "api-secret",
		RateLimitRequests:  100,
		RateLimitWindow:    time.Minute,
		MaxLoginAttempts:   5,
		LockoutDuration:    time.Minute * 15,
		CleanupInterval:    time.Hour,
		EnableMetrics:      true,
		LogFailedAttempts:  true,
		BlockSuspiciousIPs: true,
	})
}

// TearDownSuite cleans up after tests
func (suite *ErrorHandlingTestSuite) TearDownSuite() {
	// Cleanup resources
}

// TestAPIErrorCreation tests API error creation
func (suite *ErrorHandlingTestSuite) TestAPIErrorCreation() {
	err := errors.NewAPIError(errors.ErrInvalidInput, "Test error", "Test details")
	assert.Equal(suite.T(), errors.ErrInvalidInput, err.Code)
	assert.Equal(suite.T(), "Test error", err.Message)
	assert.Equal(suite.T(), "Test details", err.Details)
	assert.Equal(suite.T(), http.StatusBadRequest, err.HTTPStatus())
}

// TestErrorHandlerProcessing tests error handler processing
func (suite *ErrorHandlingTestSuite) TestErrorHandlerProcessing() {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := suite.echo.NewContext(req, rec)

	err := errors.NewAPIError(errors.ErrInternalServer, "Test internal error", "Details")
	suite.errorHandler.HandleError(err, c)

	assert.Equal(suite.T(), http.StatusInternalServerError, rec.Code)
}

// TestErrorSanitization tests error sanitization
func (suite *ErrorHandlingTestSuite) TestErrorSanitization() {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := suite.echo.NewContext(req, rec)

	// Create handler with sanitization enabled
	config := &errors.ErrorHandlerConfig{
		DebugMode:             false,
		LogStackTrace:         false,
		CircuitBreakerEnabled: false,
		SanitizeErrors:        true,
	}
	sanitizingHandler := errors.NewErrorHandler(config)

	err := errors.NewAPIError(errors.ErrInternalServer, "Sensitive error", "Sensitive details")
	sanitizingHandler.HandleError(err, c)

	assert.Equal(suite.T(), http.StatusInternalServerError, rec.Code)
}

// TestRetryMechanism tests retry functionality
func (suite *ErrorHandlingTestSuite) TestRetryMechanism() {
	ctx := context.Background()
	attempts := 0

	err := suite.errorHandler.WithRetry(ctx, "test-operation", func() error {
		attempts++
		if attempts < 3 {
			return errors.NewAPIError(errors.ErrDatabaseTimeout, "Timeout", "")
		}
		return nil
	})

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 3, attempts)
}

// TestCircuitBreaker tests circuit breaker functionality
func (suite *ErrorHandlingTestSuite) TestCircuitBreaker() {
	cb := suite.errorHandler.GetCircuitBreaker("test-service")
	assert.NotNil(suite.T(), cb)

	// Test that we get the same circuit breaker for the same service
	cb2 := suite.errorHandler.GetCircuitBreaker("test-service")
	assert.Equal(suite.T(), cb, cb2)
}

// TestAuthErrorHandling tests authentication error handling
func (suite *ErrorHandlingTestSuite) TestAuthErrorHandling() {
	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := suite.echo.NewContext(req, rec)

	// Test ValidateAPIKey with invalid key
	keyInfo, err := suite.authHandler.ValidateAPIKey("invalid-key", c)
	assert.Nil(suite.T(), keyInfo)
	assert.NotNil(suite.T(), err)
	assert.Equal(suite.T(), errors.ErrInvalidAPIKey, err.Code)

	// Test CheckRateLimit
	err = suite.authHandler.CheckRateLimit("127.0.0.1", c)
	assert.Nil(suite.T(), err) // Should pass for first request
}

// TestErrorResponse tests error response formatting
func (suite *ErrorHandlingTestSuite) TestErrorResponse() {
	apiErr := errors.NewAPIError(errors.ErrInvalidAPIKey, "Invalid API key", "Key validation failed")

	// Test error response structure
	response := errors.NewErrorResponse(apiErr)

	assert.False(suite.T(), response.Success)
	assert.NotNil(suite.T(), response.Error)
	assert.Equal(suite.T(), errors.ErrInvalidAPIKey, response.Error.Code)

	// Test auth error handler with proper config
	authConfig := &errors.AuthConfig{
		JWTSecret:          "test-secret",
		JWTExpiration:      time.Hour,
		APIKeySecret:       "api-secret",
		RateLimitRequests:  100,
		RateLimitWindow:    time.Minute,
		MaxLoginAttempts:   5,
		LockoutDuration:    time.Minute * 15,
		CleanupInterval:    time.Minute * 5,
		EnableMetrics:      true,
		LogFailedAttempts:  false,
		BlockSuspiciousIPs: false,
		WhitelistedIPs:     []string{},
		BlacklistedIPs:     []string{},
	}
	authHandler := errors.NewAuthErrorHandler(authConfig)

	// Create a test request for context
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	// Test ValidateAPIKey method
	keyInfo, err := authHandler.ValidateAPIKey("invalid-key", c)
	assert.Nil(suite.T(), keyInfo)
	assert.NotNil(suite.T(), err)

	// Test CheckRateLimit method
	err = authHandler.CheckRateLimit("127.0.0.1", c)
	assert.Nil(suite.T(), err) // Should pass for first request
}

// TestErrorHandlingTestSuite runs the test suite
func TestErrorHandlingTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorHandlingTestSuite))
}

// BenchmarkErrorHandlerProcessing benchmarks error handler processing
func BenchmarkErrorHandlerProcessing(b *testing.B) {
	echo := echo.New()
	handler := errors.NewErrorHandler(&errors.ErrorHandlerConfig{
		DebugMode:      false,
		SanitizeErrors: true,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := echo.NewContext(req, rec)

		err := errors.NewAPIError(errors.ErrInternalServer, "Test error", "")
		handler.HandleError(err, c)
	}
}
