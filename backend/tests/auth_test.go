package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nutrition-platform/handlers"
	"nutrition-platform/middleware"
	"nutrition-platform/security"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthHandler_Register(t *testing.T) {
	// Setup
	e := echo.New()
	jwtManager := security.NewJWTManager()
	authHandler := handlers.NewAuthHandler(nil, jwtManager)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
		validateFields map[string]bool
	}{
		{
			name: "Valid registration",
			requestBody: map[string]interface{}{
				"email":           "test@example.com",
				"password":        "password123",
				"confirm_password": "password123",
				"first_name":      "John",
				"last_name":       "Doe",
				"date_of_birth":   "1990-01-01",
				"gender":          "male",
				"language":        "en",
			},
			expectedStatus: http.StatusCreated,
			validateFields: map[string]bool{
				"access_token":  true,
				"refresh_token": true,
				"user":          true,
				"expires_in":    true,
			},
		},
		{
			name: "Invalid email format",
			requestBody: map[string]interface{}{
				"email":           "invalid-email",
				"password":        "password123",
				"confirm_password": "password123",
				"first_name":      "John",
				"last_name":       "Doe",
				"date_of_birth":   "1990-01-01",
				"gender":          "male",
				"language":        "en",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Validation failed",
		},
		{
			name: "Password too short",
			requestBody: map[string]interface{}{
				"email":           "test@example.com",
				"password":        "123",
				"confirm_password": "123",
				"first_name":      "John",
				"last_name":       "Doe",
				"date_of_birth":   "1990-01-01",
				"gender":          "male",
				"language":        "en",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Validation failed",
		},
		{
			name: "Passwords don't match",
			requestBody: map[string]interface{}{
				"email":           "test@example.com",
				"password":        "password123",
				"confirm_password": "different123",
				"first_name":      "John",
				"last_name":       "Doe",
				"date_of_birth":   "1990-01-01",
				"gender":          "male",
				"language":        "en",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Passwords do not match",
		},
		{
			name: "Invalid gender",
			requestBody: map[string]interface{}{
				"email":           "test@example.com",
				"password":        "password123",
				"confirm_password": "password123",
				"first_name":      "John",
				"last_name":       "Doe",
				"date_of_birth":   "1990-01-01",
				"gender":          "invalid",
				"language":        "en",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Validation failed",
		},
		{
			name: "Invalid language",
			requestBody: map[string]interface{}{
				"email":           "test@example.com",
				"password":        "password123",
				"confirm_password": "password123",
				"first_name":      "John",
				"last_name":       "Doe",
				"date_of_birth":   "1990-01-01",
				"gender":          "male",
				"language":        "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Validation failed",
		},
		{
			name: "Missing required fields",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := authHandler.Register(c)

			if tt.expectedStatus >= 400 {
				assert.Error(t, err)
				if echo.HTTPError, ok := err.(*echo.HTTPError); ok {
					assert.Equal(t, tt.expectedStatus, echo.HTTPError.Code)
					if tt.expectedError != "" {
						assert.Contains(t, echo.HTTPError.Message.(string), tt.expectedError)
					}
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)

				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				for field, shouldExist := range tt.validateFields {
					if shouldExist {
						assert.Contains(t, response, field)
					}
				}

				// Validate token format
				if accessToken, ok := response["access_token"].(string); ok && accessToken != "" {
					assert.True(t, len(accessToken) > 10, "Access token should be substantial")
				}
				if refreshToken, ok := response["refresh_token"].(string); ok && refreshToken != "" {
					assert.True(t, len(refreshToken) > 10, "Refresh token should be substantial")
				}
				if expiresIn, ok := response["expires_in"].(float64); ok {
					assert.Equal(t, float64(15*60), expiresIn, "Expires in should be 15 minutes")
				}
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	// Setup
	e := echo.New()
	jwtManager := security.NewJWTManager()
	authHandler := handlers.NewAuthHandler(nil, jwtManager)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid login",
			requestBody: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid email format",
			requestBody: map[string]interface{}{
				"email":    "invalid-email",
				"password": "password123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Validation failed",
		},
		{
			name: "Missing password",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Validation failed",
		},
		{
			name: "Invalid JSON",
			requestBody: "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := authHandler.Login(c)

			if tt.expectedStatus >= 400 {
				assert.Error(t, err)
				if echo.HTTPError, ok := err.(*echo.HTTPError); ok {
					assert.Equal(t, tt.expectedStatus, echo.HTTPError.Code)
					if tt.expectedError != "" {
						assert.Contains(t, echo.HTTPError.Message.(string), tt.expectedError)
					}
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)

				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Contains(t, response, "access_token")
				assert.Contains(t, response, "refresh_token")
				assert.Contains(t, response, "user")
				assert.Contains(t, response, "expires_in")
			}
		})
	}
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	// Setup
	e := echo.New()
	jwtManager := security.NewJWTManager()
	authHandler := handlers.NewAuthHandler(nil, jwtManager)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid refresh token request",
			requestBody: map[string]interface{}{
				"refresh_token": "valid_refresh_token",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Missing refresh token",
			requestBody: map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Validation failed",
		},
		{
			name:           "Invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := authHandler.RefreshToken(c)

			if tt.expectedStatus >= 400 {
				assert.Error(t, err)
				if echo.HTTPError, ok := err.(*echo.HTTPError); ok {
					assert.Equal(t, tt.expectedStatus, echo.HTTPError.Code)
					if tt.expectedError != "" {
						assert.Contains(t, echo.HTTPError.Message.(string), tt.expectedError)
					}
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)

				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Contains(t, response, "access_token")
				assert.Contains(t, response, "refresh_token")
				assert.Contains(t, response, "user")
				assert.Contains(t, response, "expires_in")
			}
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	// Setup
	e := echo.New()
	jwtManager := security.NewJWTManager()
	authHandler := handlers.NewAuthHandler(nil, jwtManager)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid logout with Bearer token",
			authHeader:     "Bearer valid_token_here",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Authorization header required",
		},
		{
			name:           "Invalid authorization format - no Bearer",
			authHeader:     "valid_token_here",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid authorization header format",
		},
		{
			name:           "Invalid authorization format - multiple parts",
			authHeader:     "Bearer token extra_part",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid authorization header format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := authHandler.Logout(c)

			if tt.expectedStatus >= 400 {
				assert.Error(t, err)
				if echo.HTTPError, ok := err.(*echo.HTTPError); ok {
					assert.Equal(t, tt.expectedStatus, echo.HTTPError.Code)
					if tt.expectedError != "" {
						assert.Contains(t, echo.HTTPError.Message.(string), tt.expectedError)
					}
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)

				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Contains(t, response, "message")
				assert.Equal(t, "Logged out successfully", response["message"])
			}
		})
	}
}

func TestJWTAuth_Middleware(t *testing.T) {
	// Setup
	e := echo.New()
	jwtAuth := middleware.JWTAuth()

	// Generate a valid token for testing
	validToken, err := middleware.GenerateToken("test-user-id", "test@example.com", "user", false)
	require.NoError(t, err)

	// Generate an expired token (simulate by creating an invalid token)
	expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedError  string
		setupContext   bool
	}{
		{
			name:           "Valid JWT token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			setupContext:   true,
		},
		{
			name:           "Missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Authorization header required",
		},
		{
			name:           "Invalid authorization format",
			authHeader:     "InvalidFormat " + validToken,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Bearer token required",
		},
		{
			name:           "Invalid JWT token",
			authHeader:     "Bearer invalid_token",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid token",
		},
		{
			name:           "Expired JWT token",
			authHeader:     "Bearer " + expiredToken,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid token",
		},
		{
			name:           "Malformed JWT token",
			authHeader:     "Bearer malformed.jwt.token",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Create a mock handler to test middleware
			nextHandler := func(c echo.Context) error {
				return c.JSON(http.StatusOK, map[string]string{"message": "success"})
			}

			err := jwtAuth(nextHandler)(c)

			if tt.expectedStatus >= 400 {
				assert.Error(t, err)
				if echo.HTTPError, ok := err.(*echo.HTTPError); ok {
					assert.Equal(t, tt.expectedStatus, echo.HTTPError.Code)
					if tt.expectedError != "" {
						assert.Contains(t, echo.HTTPError.Message.(string), tt.expectedError)
					}
				}
			} else {
				assert.NoError(t, err)
				if tt.setupContext {
					// Verify that user context is set
					userID := c.Get("user_id")
					assert.NotNil(t, userID)
					assert.Equal(t, "test-user-id", userID)
				}
			}
		})
	}
}

func TestAdminAuth_Middleware(t *testing.T) {
	// Setup
	e := echo.New()
	adminAuth := middleware.AdminAuth()

	tests := []struct {
		name           string
		isAdmin        bool
		adminInContext bool
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Admin user access",
			isAdmin:        true,
			adminInContext: true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Non-admin user access",
			isAdmin:        false,
			adminInContext: true,
			expectedStatus: http.StatusForbidden,
			expectedError:  "Admin access required",
		},
		{
			name:           "Missing admin context",
			isAdmin:        false,
			adminInContext: false,
			expectedStatus: http.StatusForbidden,
			expectedError:  "Admin access required",
		},
		{
			name:           "Invalid admin context type",
			isAdmin:        false,
			adminInContext: false,
			expectedStatus: http.StatusForbidden,
			expectedError:  "Admin access required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set up the context based on test case
			if tt.adminInContext {
				c.Set("is_admin", tt.isAdmin)
			} else {
				c.Set("is_admin", "invalid_type")
			}

			// Create a mock handler to test middleware
			nextHandler := func(c echo.Context) error {
				return c.JSON(http.StatusOK, map[string]string{"message": "admin success"})
			}

			err := adminAuth(nextHandler)(c)

			if tt.expectedStatus >= 400 {
				assert.Error(t, err)
				if echo.HTTPError, ok := err.(*echo.HTTPError); ok {
					assert.Equal(t, tt.expectedStatus, echo.HTTPError.Code)
					if tt.expectedError != "" {
						assert.Contains(t, echo.HTTPError.Message.(string), tt.expectedError)
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTokenGeneration(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		email    string
		role     string
		isAdmin  bool
		testType string
	}{
		{
			name:     "Generate access token for regular user",
			userID:   "user123",
			email:    "user@example.com",
			role:     "user",
			isAdmin:  false,
			testType: "access",
		},
		{
			name:     "Generate access token for admin user",
			userID:   "admin123",
			email:    "admin@example.com",
			role:     "admin",
			isAdmin:  true,
			testType: "access",
		},
		{
			name:     "Generate refresh token",
			userID:   "user123",
			email:    "user@example.com",
			role:     "user",
			isAdmin:  false,
			testType: "refresh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var token string
			var err error

			if tt.testType == "access" {
				token, err = middleware.GenerateToken(tt.userID, tt.email, tt.role, tt.isAdmin)
			} else {
				token, err = middleware.GenerateRefreshToken(tt.userID)
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, token)
			assert.True(t, len(token) > 10, "Token should be substantial")

			// Verify token contains three parts (header.payload.signature)
			parts := strings.Split(token, ".")
			assert.Len(t, parts, 3, "JWT token should have 3 parts separated by dots")
		})
	}
}

// Performance test for concurrent authentication requests
func TestAuth_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	e := echo.New()
	jwtManager := security.NewJWTManager()
	authHandler := handlers.NewAuthHandler(nil, jwtManager)

	// Test concurrent login requests
	concurrency := 50
	t.Run("ConcurrentLoginRequests", func(t *testing.T) {
		t.Parallel()
		
		done := make(chan bool, concurrency)
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			go func(id int) {
				defer func() { done <- true }()
				
				requestBody := map[string]interface{}{
					"email":    "test@example.com",
					"password": "password123",
				}
				body, _ := json.Marshal(requestBody)

				req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				err := authHandler.Login(c)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < concurrency; i++ {
			<-done
		}

		duration := time.Since(start)
		t.Logf("Concurrent login requests (%d) completed in %v", concurrency, duration)
		
		// Performance assertion - should complete within reasonable time
		assert.Less(t, duration, 5*time.Second, "Concurrent requests should complete within 5 seconds")
	})

	// Test concurrent token generation
	t.Run("ConcurrentTokenGeneration", func(t *testing.T) {
		t.Parallel()
		
		done := make(chan bool, concurrency)
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			go func(id int) {
				defer func() { done <- true }()
				
				token, err := middleware.GenerateToken(
					"user-"+string(rune(id)),
					"user@example.com",
					"user",
					false,
				)
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < concurrency; i++ {
			<-done
		}

		duration := time.Since(start)
		t.Logf("Concurrent token generation (%d) completed in %v", concurrency, duration)
		
		// Performance assertion - should complete within reasonable time
		assert.Less(t, duration, 2*time.Second, "Concurrent token generation should complete within 2 seconds")
	})
}
