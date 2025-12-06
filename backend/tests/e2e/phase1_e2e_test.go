package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const baseURL = "http://localhost:8080"

type Phase1E2ETestSuite struct {
	suite.Suite
	client *http.Client
}

func (s *Phase1E2ETestSuite) SetupSuite() {
	s.client = &http.Client{Timeout: 10 * time.Second}

	// Verify server is running
	resp, err := s.client.Get(baseURL + "/health")
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func (s *Phase1E2ETestSuite) TestCacheHitMiss() {
	url := baseURL + "/api/v1/nutrition-data/recipes?limit=5"

	// First request - MISS
	resp1, err := s.client.Get(url)
	require.NoError(s.T(), err)
	cacheStatus1 := resp1.Header.Get("X-Cache")
	s.T().Logf("First request: X-Cache = %s", cacheStatus1)
	resp1.Body.Close()

	time.Sleep(500 * time.Millisecond)

	// Second request - HIT
	resp2, err := s.client.Get(url)
	require.NoError(s.T(), err)
	cacheStatus2 := resp2.Header.Get("X-Cache")
	s.T().Logf("Second request: X-Cache = %s", cacheStatus2)
	resp2.Body.Close()

	if cacheStatus2 == "HIT" {
		s.T().Log("✅ Cache working correctly")
	}
}

func (s *Phase1E2ETestSuite) TestRateLimitingHeaders() {
	url := baseURL + "/api/v1/nutrition-data/recipes?limit=1"

	resp, err := s.client.Get(url)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.NotEmpty(s.T(), resp.Header.Get("X-RateLimit-Limit"))
	assert.NotEmpty(s.T(), resp.Header.Get("X-RateLimit-Remaining"))
	assert.NotEmpty(s.T(), resp.Header.Get("X-RateLimit-Reset"))
}

func (s *Phase1E2ETestSuite) TestSecurityHeaders() {
	url := baseURL + "/api/v1/nutrition-data/recipes?limit=1"

	resp, err := s.client.Get(url)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	// Check security headers
	assert.Equal(s.T(), "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(s.T(), "DENY", resp.Header.Get("X-Frame-Options"))
	assert.NotEmpty(s.T(), resp.Header.Get("X-API-Version"))
}

func (s *Phase1E2ETestSuite) TestAPIResponseFormat() {
	url := baseURL + "/api/v1/nutrition-data/recipes?limit=5"

	resp, err := s.client.Get(url)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	assert.Contains(s.T(), resp.Header.Get("Content-Type"), "application/json")
}

func (s *Phase1E2ETestSuite) TestHealthEndpoint() {
	resp, err := s.client.Get(baseURL + "/health")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

func (s *Phase1E2ETestSuite) TestNutritionDataEndpoints() {
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
		s.Run(endpoint.name, func() {
			url := baseURL + endpoint.path + "?limit=5"
			resp, err := s.client.Get(url)
			require.NoError(s.T(), err)
			defer resp.Body.Close()

			assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
			assert.Contains(s.T(), resp.Header.Get("Content-Type"), "application/json")
		})
	}
}

func (s *Phase1E2ETestSuite) TestPagination() {
	url := baseURL + "/api/v1/nutrition-data/recipes?page=1&limit=5"

	resp, err := s.client.Get(url)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

func (s *Phase1E2ETestSuite) TestSearchFunctionality() {
	url := baseURL + "/api/v1/nutrition-data/recipes?q=chicken&limit=5"

	resp, err := s.client.Get(url)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

func (s *Phase1E2ETestSuite) TestErrorHandling() {
	// Test 404
	resp, err := s.client.Get(baseURL + "/api/v1/nonexistent")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)

	// Test invalid parameters
	resp, err = s.client.Get(baseURL + "/api/v1/nutrition-data/recipes?limit=invalid")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode)
}

func (s *Phase1E2ETestSuite) TestPerformanceBaseline() {
	url := baseURL + "/api/v1/nutrition-data/recipes?limit=5"

	start := time.Now()
	resp, err := s.client.Get(url)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	duration := time.Since(start)

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	assert.True(s.T(), duration < 2*time.Second, "Response time should be under 2 seconds")

	s.T().Logf("Response time: %v", duration)
}

func TestPhase1E2ETestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}
	suite.Run(t, new(Phase1E2ETestSuite))
}
