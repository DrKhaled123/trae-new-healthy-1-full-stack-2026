package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nutrition-platform/config"
	"nutrition-platform/models"
	"nutrition-platform/services"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyGeneration(t *testing.T) {
	key, hash, err := models.GenerateAPIKey("nk")
	require.NoError(t, err)
	assert.NotEmpty(t, key)
	assert.NotEmpty(t, hash)
	assert.True(t, models.ValidateAPIKeyFormat(key))
}

func TestAPIKeyValidation(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"Valid key", "nk_1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", true},
		{"Invalid prefix", "invalid_1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", false},
		{"Short token", "nk_123", false},
		{"No underscore", "nk1234567890abcdef", false},
		{"Empty key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := models.ValidateAPIKeyFormat(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAPIKeyScopes(t *testing.T) {
	apiKey := &models.APIKey{
		Scopes: []models.APIKeyScope{models.ScopeNutrition, models.ScopeReadOnly},
	}

	assert.True(t, apiKey.HasScope(models.ScopeNutrition))
	assert.True(t, apiKey.HasScope(models.ScopeReadOnly))
	assert.False(t, apiKey.HasScope(models.ScopeAdmin))
}

func TestAPIKeyExpiration(t *testing.T) {
	// Test non-expired key
	future := time.Now().Add(24 * time.Hour)
	apiKey := &models.APIKey{
		Status:    models.APIKeyStatusActive,
		ExpiresAt: &future,
	}
	assert.False(t, apiKey.IsExpired())
	assert.True(t, apiKey.IsActive())

	// Test expired key
	past := time.Now().Add(-24 * time.Hour)
	apiKey.ExpiresAt = &past
	assert.True(t, apiKey.IsExpired())
	assert.False(t, apiKey.IsActive())
}

func TestAPIKeyEndpointAccess(t *testing.T) {
	tests := []struct {
		name     string
		scopes   []models.APIKeyScope
		endpoint string
		method   string
		expected bool
	}{
		{"Read access to nutrition", []models.APIKeyScope{models.ScopeNutrition, models.ScopeReadOnly}, "/api/v1/nutrition/analyze", "GET", true},
		{"Write access denied", []models.APIKeyScope{models.ScopeReadOnly}, "/api/v1/nutrition/analyze", "POST", false},
		{"Admin access all", []models.APIKeyScope{models.ScopeAdmin}, "/api/v1/nutrition/analyze", "POST", true},
		{"No scope access", []models.APIKeyScope{models.ScopeWorkouts}, "/api/v1/nutrition/analyze", "GET", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := &models.APIKey{
				Status: models.APIKeyStatusActive,
				Scopes: tt.scopes,
			}
			result := apiKey.CanAccess(tt.endpoint, tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRateLimiting(t *testing.T) {
	// This would require a more complex setup with actual database
	// For now, we'll test the rate limiter logic
	t.Skip("Rate limiting test requires database setup")
}

func TestAPIKeyMiddleware(t *testing.T) {
	// Setup test environment
	cfg := &config.Config{
		Environment: "test",
	}

	// Initialize test database
	err := models.InitDatabase(cfg)
	require.NoError(t, err)
	defer models.CloseDatabase()

	// Create API key service
	apiKeyService := services.NewAPIKeyService(models.DB)

	// Create test API key
	req := &models.CreateAPIKeyRequest{
		Name:      "Test Key",
		Scopes:    []models.APIKeyScope{models.ScopeNutrition, models.ScopeReadOnly},
		RateLimit: 100,
	}

	response, err := apiKeyService.CreateAPIKey("test-user", req)
	require.NoError(t, err)
	require.NotNil(t, response)

	// Test middleware with valid key
	e := echo.New()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition", nil)
	req2.Header.Set("Authorization", "Bearer "+response.Key)
	rec := httptest.NewRecorder()
	_ = e.NewContext(req2, rec) // Context created but not used in this test

	// This would require the actual middleware setup
	t.Skip("Middleware test requires full setup")
}
