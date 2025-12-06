package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nutrition-platform/handlers"
	"nutrition-platform/utils"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// TestAPIResponseContract verifies all endpoints return standardized format
func TestAPIResponseContract(t *testing.T) {
	e := echo.New()

	endpoints := []struct {
		name    string
		method  string
		path    string
		handler func(echo.Context) error
	}{
		{
			name:    "GetRecipes",
			method:  http.MethodGet,
			path:    "/api/v1/nutrition-data/recipes",
			handler: handlers.NewNutritionDataHandler(nil, "../../nutrition data json").GetRecipes,
		},
		{
			name:    "GetWorkouts",
			method:  http.MethodGet,
			path:    "/api/v1/nutrition-data/workouts",
			handler: handlers.NewNutritionDataHandler(nil, "../../nutrition data json").GetWorkouts,
		},
		{
			name:    "GetComplaints",
			method:  http.MethodGet,
			path:    "/api/v1/nutrition-data/complaints",
			handler: handlers.NewNutritionDataHandler(nil, "../../nutrition data json").GetComplaints,
		},
		{
			name:    "GetMetabolism",
			method:  http.MethodGet,
			path:    "/api/v1/nutrition-data/metabolism",
			handler: handlers.NewNutritionDataHandler(nil, "../../nutrition data json").GetMetabolism,
		},
		{
			name:    "GetDrugsNutrition",
			method:  http.MethodGet,
			path:    "/api/v1/nutrition-data/drugs-nutrition",
			handler: handlers.NewNutritionDataHandler(nil, "../../nutrition data json").GetDrugsNutrition,
		},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			req := httptest.NewRequest(endpoint.method, endpoint.path+"?page=1&limit=10", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath(endpoint.path)

			err := endpoint.handler(c)
			assert.NoError(t, err)

			var response utils.APIResponse
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Verify contract
			assert.Contains(t, []string{"success", "error"}, response.Status)
			if response.Status == "success" {
				assert.True(t, response.Items != nil || response.Data != nil)
			}
		})
	}
}

// TestPaginationContract verifies pagination response format
func TestPaginationContract(t *testing.T) {
	e := echo.New()
	h := handlers.NewNutritionDataHandler(nil, "../../nutrition data json")

	endpoints := []struct {
		name    string
		path    string
		handler func(echo.Context) error
	}{
		{
			name:    "Recipes Pagination",
			path:    "/api/v1/nutrition-data/recipes",
			handler: h.GetRecipes,
		},
		{
			name:    "Workouts Pagination",
			path:    "/api/v1/nutrition-data/workouts",
			handler: h.GetWorkouts,
		},
		{
			name:    "Complaints Pagination",
			path:    "/api/v1/nutrition-data/complaints",
			handler: h.GetComplaints,
		},
		{
			name:    "Metabolism Pagination",
			path:    "/api/v1/nutrition-data/metabolism",
			handler: h.GetMetabolism,
		},
		{
			name:    "DrugsNutrition Pagination",
			path:    "/api/v1/nutrition-data/drugs-nutrition",
			handler: h.GetDrugsNutrition,
		},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, endpoint.path+"?page=1&limit=5", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath(endpoint.path)

			err := endpoint.handler(c)
			assert.NoError(t, err)

			var response utils.APIResponse
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Verify pagination structure
			assert.Equal(t, "success", response.Status)
			assert.NotNil(t, response.Pagination)
			assert.NotNil(t, response.Items)

			// Verify pagination fields
			pagination := response.Pagination
			assert.NotNil(t, pagination.Page)
			assert.NotNil(t, pagination.Limit)
			assert.NotNil(t, pagination.Total)
			assert.NotNil(t, pagination.TotalPages)
		})
	}
}

// TestErrorResponseContract verifies error response format
func TestErrorResponseContract(t *testing.T) {
	e := echo.New()
	h := handlers.NewNutritionDataHandler(nil, "../../nutrition data json")

	errorCases := []struct {
		name        string
		path        string
		queryParams string
		handler     func(echo.Context) error
	}{
		{
			name:        "Recipes Invalid Page",
			path:        "/api/v1/nutrition-data/recipes",
			queryParams: "?page=-1&limit=10",
			handler:     h.GetRecipes,
		},
		{
			name:        "Recipes Invalid Limit",
			path:        "/api/v1/nutrition-data/recipes",
			queryParams: "?page=1&limit=1000",
			handler:     h.GetRecipes,
		},
		{
			name:        "Workouts Invalid Parameters",
			path:        "/api/v1/nutrition-data/workouts",
			queryParams: "?page=invalid&limit=invalid",
			handler:     h.GetWorkouts,
		},
	}

	for _, testCase := range errorCases {
		t.Run(testCase.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, testCase.path+testCase.queryParams, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath(testCase.path)

			err := testCase.handler(c)
			assert.NoError(t, err)

			var response utils.APIResponse
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Verify error response structure
			assert.Equal(t, "error", response.Status)
			assert.NotEmpty(t, response.Error)
		})
	}
}

// TestContentTypeHeaders verifies all endpoints return correct content type
func TestContentTypeHeaders(t *testing.T) {
	e := echo.New()
	h := handlers.NewNutritionDataHandler(nil, "../../nutrition data json")

	endpoints := []struct {
		name    string
		path    string
		handler func(echo.Context) error
	}{
		{
			name:    "Recipes Content Type",
			path:    "/api/v1/nutrition-data/recipes",
			handler: h.GetRecipes,
		},
		{
			name:    "Workouts Content Type",
			path:    "/api/v1/nutrition-data/workouts",
			handler: h.GetWorkouts,
		},
		{
			name:    "Complaints Content Type",
			path:    "/api/v1/nutrition-data/complaints",
			handler: h.GetComplaints,
		},
		{
			name:    "Metabolism Content Type",
			path:    "/api/v1/nutrition-data/metabolism",
			handler: h.GetMetabolism,
		},
		{
			name:    "DrugsNutrition Content Type",
			path:    "/api/v1/nutrition-data/drugs-nutrition",
			handler: h.GetDrugsNutrition,
		},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, endpoint.path+"?page=1&limit=5", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath(endpoint.path)

			err := endpoint.handler(c)
			assert.NoError(t, err)

			contentType := rec.Header().Get("Content-Type")
			assert.Contains(t, contentType, "application/json")
		})
	}
}

// TestResponseStructure verifies individual item structures
func TestResponseStructure(t *testing.T) {
	e := echo.New()
	h := handlers.NewNutritionDataHandler(nil, "../../nutrition data json")

	// Test recipe structure
	t.Run("Recipe Item Structure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition-data/recipes?limit=1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/v1/nutrition-data/recipes")

		err := h.GetRecipes(c)
		assert.NoError(t, err)

		var response utils.APIResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		items, ok := response.Items.([]interface{})
		if ok && len(items) > 0 {
			recipe := items[0].(map[string]interface{})
			assert.NotNil(t, recipe["name"])
		}
	})

	// Test workout structure
	t.Run("Workout Item Structure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition-data/workouts?limit=1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/v1/nutrition-data/workouts")

		err := h.GetWorkouts(c)
		assert.NoError(t, err)

		var response utils.APIResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		items, ok := response.Items.([]interface{})
		if ok && len(items) > 0 {
			workout := items[0].(map[string]interface{})
			assert.NotNil(t, workout["goal"])
		}
	})

	// Test complaint structure
	t.Run("Complaint Item Structure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition-data/complaints?limit=1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/v1/nutrition-data/complaints")

		err := h.GetComplaints(c)
		assert.NoError(t, err)

		var response utils.APIResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		items, ok := response.Items.([]interface{})
		if ok && len(items) > 0 {
			complaint := items[0].(map[string]interface{})
			assert.NotNil(t, complaint["name"])
		}
	})
}
