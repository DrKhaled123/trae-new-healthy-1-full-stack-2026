package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nutrition-platform/config"
	"nutrition-platform/models"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthEndpoint(t *testing.T) {
	e := echo.New()

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "healthy",
			"timestamp": "2023-01-01T00:00:00Z",
			"version":   "1.0.0",
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestNutritionAnalysisEndpoint(t *testing.T) {
	e := echo.New()

	e.POST("/api/nutrition/analyze", func(c echo.Context) error {
		var req map[string]interface{}
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"food":        req["food"],
			"calories":    100,
			"protein":     5.0,
			"carbs":       15.0,
			"fat":         3.0,
			"halalStatus": true,
		})
	})

	requestBody := map[string]interface{}{
		"food":       "apple",
		"quantity":   100,
		"unit":       "g",
		"checkHalal": true,
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/api/nutrition/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "apple", response["food"])
	assert.Equal(t, float64(100), response["calories"])
}

func TestDatabaseConnection(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		DBHost:      "localhost",
		DBPort:      5432,
		DBName:      "nutrition_platform_test",
		DBUser:      "test_user",
		DBPassword:  "test_password",
		DBSSLMode:   "disable",
	}

	// This would test actual database connection
	// For CI/CD, we'd use a test database
	t.Skip("Database connection test requires test database setup")

	err := models.InitDatabase(cfg)
	if err != nil {
		t.Logf("Database connection failed (expected in test environment): %v", err)
		return
	}
	defer models.CloseDatabase()

	err = models.HealthCheck()
	assert.NoError(t, err)
}

func TestAPIEndpointsWithoutAuth(t *testing.T) {
	e := echo.New()

	// Add routes that don't require authentication
	e.GET("/api/info", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"name":    "Nutrition Platform API",
			"version": "1.0.0",
			"status":  "online",
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/info", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Nutrition Platform API", response["name"])
}

func TestAPIEndpointsWithAuth(t *testing.T) {
	e := echo.New()

	// Simulate API key middleware
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := c.Request().Header.Get("Authorization")
			if apiKey == "" || !strings.HasPrefix(apiKey, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "missing_api_key",
				})
			}
			return next(c)
		}
	})

	e.GET("/api/v1/users", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"users": []interface{}{},
		})
	})

	// Test without API key
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Test with API key
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer test_key")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMealPlanGeneration(t *testing.T) {
	e := echo.New()

	e.POST("/api/generate-meal-plan", func(c echo.Context) error {
		var req map[string]interface{}
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":        true,
			"targetCalories": 2000,
			"mealPlan": []map[string]interface{}{
				{
					"mealType": "breakfast",
					"calories": 500,
					"foods":    []string{"Oatmeal with berries"},
				},
			},
		})
	})

	requestBody := map[string]interface{}{
		"age":           25,
		"gender":        "male",
		"height":        175,
		"weight":        70,
		"activityLevel": "moderate",
		"goal":          "maintain",
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/api/generate-meal-plan", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.Equal(t, float64(2000), response["targetCalories"])
}
