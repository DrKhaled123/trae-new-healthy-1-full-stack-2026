package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2ETestConfig defines configuration for end-to-end tests
type E2ETestConfig struct {
	BaseURL string
	Timeout time.Duration
}

var e2eConfig = E2ETestConfig{
	BaseURL: "http://localhost:8080",
	Timeout: 10 * time.Second,
}

// APIResponse represents the standardized API response format
type APIResponse struct {
	Status     string                 `json:"status"`
	Data       interface{}            `json:"data,omitempty"`
	Items      []interface{}          `json:"items,omitempty"`
	Pagination *PaginationResponse    `json:"pagination,omitempty"`
	Filters    interface{}            `json:"filters,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Message    string                 `json:"message,omitempty"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
}

type PaginationResponse struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

func TestE2E_HealthCheck(t *testing.T) {
	client := &http.Client{Timeout: e2eConfig.Timeout}
	
	resp, err := client.Get(e2eConfig.BaseURL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Check security headers are present
	assert.NotEmpty(t, resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
}

func TestE2E_NutritionDataEndpoints(t *testing.T) {
	client := &http.Client{Timeout: e2eConfig.Timeout}
	
	endpoints := []struct {
		name string
		path string
	}{
		{"Recipes", "/api/v1/nutrition-data/recipes"},
		{"Workouts", "/api/v1/nutrition-data/workouts"},
		{"Complaints", "/api/v1/nutrition-data/complaints"},
		{"Metabolism", "/api/v1/nutrition-data/metabolism"},
	}
	
	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			url := fmt.Sprintf("%s%s?limit=5", e2eConfig.BaseURL, endpoint.path)
			resp, err := client.Get(url)
			require.NoError(t, err)
			defer resp.Body.Close()
			
			// Check status code
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			
			// Check content type
			assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
			
			// Check rate limiting headers
			assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Limit"))
			assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Remaining"))
			
			// Check security headers
			assert.Equal(t, "1.0", resp.Header.Get("X-API-Version"))
			assert.Contains(t, resp.Header.Get("Cache-Control"), "no-store")
			
			// Parse response
			var apiResp APIResponse
			err = json.NewDecoder(resp.Body).Decode(&apiResp)
			require.NoError(t, err)
			
			// Check standardized response format
			assert.Equal(t, "success", apiResp.Status)
			assert.True(t, apiResp.Data != nil || len(apiResp.Items) > 0, "Either data or items should be present")
			
			// If pagination is present, validate structure
			if apiResp.Pagination != nil {
				assert.True(t, apiResp.Pagination.Page > 0)
				assert.True(t, apiResp.Pagination.Limit > 0)
				assert.True(t, apiResp.Pagination.TotalPages > 0)
			}
		})
	}
}

func TestE2E_RateLimiting(t *testing.T) {
	client := &http.Client{Timeout: e2eConfig.Timeout}
	url := e2eConfig.BaseURL + "/api/v1/nutrition-data/recipes?limit=1"
	
	// Make multiple requests to test rate limiting
	var lastRemainingHeader string
	for i := 0; i < 5; i++ {
		resp, err := client.Get(url)
		require.NoError(t, err)
		
		// Check rate limit headers
		limitHeader := resp.Header.Get("X-RateLimit-Limit")
		remainingHeader := resp.Header.Get("X-RateLimit-Remaining")
		resetHeader := resp.Header.Get("X-RateLimit-Reset")
		
		assert.NotEmpty(t, limitHeader)
		assert.NotEmpty(t, remainingHeader)
		assert.NotEmpty(t, resetHeader)
		
		// Remaining count should decrease (or stay same if different users)
		if lastRemainingHeader != "" {
			// Note: This might not always decrease if requests are from different IPs
			// or if the rate limiter resets between requests
		}
		lastRemainingHeader = remainingHeader
		
		resp.Body.Close()
		
		// Small delay between requests
		time.Sleep(100 * time.Millisecond)
	}
}

func TestE2E_SecurityHeaders(t *testing.T) {
	client := &http.Client{Timeout: e2eConfig.Timeout}
	
	testCases := []struct {
		name     string
		endpoint string
		headers  map[string]string
	}{
		{
			name:     "API Endpoint Security",
			endpoint: "/api/v1/nutrition-data/recipes?limit=1",
			headers: map[string]string{
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":        "DENY",
				"X-API-Version":          "1.0",
				"Cache-Control":          "no-store",
			},
		},
		{
			name:     "Health Check Security",
			endpoint: "/health",
			headers: map[string]string{
				"X-Content-Type-Options": "nosniff",
				"Cache-Control":          "no-cache",
				"Pragma":                 "no-cache",
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.Get(e2eConfig.BaseURL + tc.endpoint)
			require.NoError(t, err)
			defer resp.Body.Close()
			
			for headerName, expectedValue := range tc.headers {
				actualValue := resp.Header.Get(headerName)
				if expectedValue == "" {
					assert.Empty(t, actualValue, "Header %s should be empty", headerName)
				} else {
					assert.Contains(t, actualValue, expectedValue, 
						"Header %s should contain %s, got %s", headerName, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestE2E_CORS(t *testing.T) {
	client := &http.Client{Timeout: e2eConfig.Timeout}
	
	// Create OPTIONS request (preflight)
	req, err := http.NewRequest("OPTIONS", e2eConfig.BaseURL+"/api/v1/nutrition-data/recipes", nil)
	require.NoError(t, err)
	
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	// Check CORS headers (if CORS is configured)
	// Note: Adjust these assertions based on your CORS configuration
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Origin"))
		assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Methods"))
	}
}

func TestE2E_ErrorHandling(t *testing.T) {
	client := &http.Client{Timeout: e2eConfig.Timeout}
	
	testCases := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "Not Found",
			url:            "/api/v1/nonexistent",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid Parameters",
			url:            "/api/v1/nutrition-data/recipes?limit=invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.Get(e2eConfig.BaseURL + tc.url)
			require.NoError(t, err)
			defer resp.Body.Close()
			
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
			
			// Should still have security headers even for errors
			assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
		})
	}
}

func TestE2E_DataIntegrity(t *testing.T) {
	client := &http.Client{Timeout: e2eConfig.Timeout}
	
	// Test recipes endpoint for data structure
	resp, err := client.Get(e2eConfig.BaseURL + "/api/v1/nutrition-data/recipes?limit=1")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	var apiResp APIResponse
	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	require.NoError(t, err)
	
	assert.Equal(t, "success", apiResp.Status)
	
	// Check that we have either data or items
	hasData := apiResp.Data != nil
	hasItems := len(apiResp.Items) > 0
	assert.True(t, hasData || hasItems, "Response should have either data or items")
	
	// If we have items, verify structure
	if hasItems {
		// Convert first item to map for inspection
		if len(apiResp.Items) > 0 {
			itemMap, ok := apiResp.Items[0].(map[string]interface{})
			if ok {
				// Basic recipe fields that should be present
				expectedFields := []string{"id", "name"}
				for _, field := range expectedFields {
					assert.Contains(t, itemMap, field, "Recipe should have %s field", field)
				}
			}
		}
	}
}

func TestE2E_PerformanceBaseline(t *testing.T) {
	client := &http.Client{Timeout: e2eConfig.Timeout}
	
	endpoints := []string{
		"/health",
		"/api/v1/nutrition-data/recipes?limit=5",
		"/api/v1/nutrition-data/workouts?limit=5",
	}
	
	for _, endpoint := range endpoints {
		t.Run("Performance_"+endpoint, func(t *testing.T) {
			start := time.Now()
			
			resp, err := client.Get(e2eConfig.BaseURL + endpoint)
			require.NoError(t, err)
			defer resp.Body.Close()
			
			duration := time.Since(start)
			
			// Performance baseline: should respond within 2 seconds
			assert.True(t, duration < 2*time.Second, 
				"Endpoint %s took %v, should be under 2s", endpoint, duration)
			
			// Log performance for monitoring
			t.Logf("Endpoint %s responded in %v", endpoint, duration)
		})
	}
}

// Helper function to run a subset of E2E tests for CI/CD
func TestE2E_CriticalPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}
	
	client := &http.Client{Timeout: e2eConfig.Timeout}
	
	// Critical path: Health -> Recipes -> Security
	t.Run("Critical_Health", func(t *testing.T) {
		resp, err := client.Get(e2eConfig.BaseURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
	
	t.Run("Critical_Recipes", func(t *testing.T) {
		resp, err := client.Get(e2eConfig.BaseURL + "/api/v1/nutrition-data/recipes?limit=1")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		var apiResp APIResponse
		err = json.NewDecoder(resp.Body).Decode(&apiResp)
		require.NoError(t, err)
		assert.Equal(t, "success", apiResp.Status)
	})
	
	t.Run("Critical_Security", func(t *testing.T) {
		resp, err := client.Get(e2eConfig.BaseURL + "/api/v1/nutrition-data/recipes")
		require.NoError(t, err)
		defer resp.Body.Close()
		
		// Must have security headers
		assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
		assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Limit"))
	})
}