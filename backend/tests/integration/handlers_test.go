package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nutrition-platform/handlers"
	"nutrition-platform/utils"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNutritionDataHandler_GetRecipes(t *testing.T) {
	// Setup
	e := echo.New()
	h := handlers.NewNutritionDataHandler(nil, "../../nutrition data json")

	// Test cases
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Get recipes with pagination",
			queryParams:    "?page=1&limit=10",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.APIResponse
				json.Unmarshal(rec.Body.Bytes(), &response)
				assert.Equal(t, "success", response.Status)
				assert.NotNil(t, response.Items)
				assert.NotNil(t, response.Pagination)
			},
		},
		{
			name:           "Get recipes with invalid page",
			queryParams:    "?page=-1&limit=10",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Get recipes with invalid limit",
			queryParams:    "?page=1&limit=1000",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Get recipes with search query",
			queryParams:    "?q=chicken",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.APIResponse
				json.Unmarshal(rec.Body.Bytes(), &response)
				assert.Equal(t, "success", response.Status)
				assert.NotNil(t, response.Items)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition-data/recipes"+tt.queryParams, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/v1/nutrition-data/recipes")

			err := h.GetRecipes(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.validateFunc != nil {
				tt.validateFunc(t, rec)
			}
		})
	}
}

func TestNutritionDataHandler_GetWorkouts(t *testing.T) {
	// Setup
	e := echo.New()
	h := handlers.NewNutritionDataHandler(nil, "../../nutrition data json")

	// Test cases
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Get workouts with pagination",
			queryParams:    "?page=1&limit=5",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.APIResponse
				json.Unmarshal(rec.Body.Bytes(), &response)
				assert.Equal(t, "success", response.Status)
				assert.NotNil(t, response.Items)
				assert.NotNil(t, response.Pagination)
			},
		},
		{
			name:           "Get workouts with goal filter",
			queryParams:    "?goal=weight_loss",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.APIResponse
				json.Unmarshal(rec.Body.Bytes(), &response)
				assert.Equal(t, "success", response.Status)
				assert.NotNil(t, response.Items)
			},
		},
		{
			name:           "Get workouts with experience level",
			queryParams:    "?level=beginner",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.APIResponse
				json.Unmarshal(rec.Body.Bytes(), &response)
				assert.Equal(t, "success", response.Status)
				assert.NotNil(t, response.Items)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition-data/workouts"+tt.queryParams, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/v1/nutrition-data/workouts")

			err := h.GetWorkouts(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.validateFunc != nil {
				tt.validateFunc(t, rec)
			}
		})
	}
}

func TestNutritionDataHandler_GetComplaints(t *testing.T) {
	// Setup
	e := echo.New()
	h := handlers.NewNutritionDataHandler(nil, "../../nutrition data json")

	// Test cases
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Get complaints with pagination",
			queryParams:    "?page=1&limit=5",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.APIResponse
				json.Unmarshal(rec.Body.Bytes(), &response)
				assert.Equal(t, "success", response.Status)
				assert.NotNil(t, response.Items)
				assert.NotNil(t, response.Pagination)
			},
		},
		{
			name:           "Get complaints with search",
			queryParams:    "?q=headache",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.APIResponse
				json.Unmarshal(rec.Body.Bytes(), &response)
				assert.Equal(t, "success", response.Status)
				assert.NotNil(t, response.Items)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition-data/complaints"+tt.queryParams, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/v1/nutrition-data/complaints")

			err := h.GetComplaints(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.validateFunc != nil {
				tt.validateFunc(t, rec)
			}
		})
	}
}

func TestNutritionDataHandler_GetMetabolism(t *testing.T) {
	// Setup
	e := echo.New()
	h := handlers.NewNutritionDataHandler(nil, "../../nutrition data json")

	// Test cases
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Get metabolism with pagination",
			queryParams:    "?page=1&limit=5",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.APIResponse
				json.Unmarshal(rec.Body.Bytes(), &response)
				assert.Equal(t, "success", response.Status)
				assert.NotNil(t, response.Items)
				assert.NotNil(t, response.Pagination)
			},
		},
		{
			name:           "Get metabolism with search",
			queryParams:    "?q=fast",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.APIResponse
				json.Unmarshal(rec.Body.Bytes(), &response)
				assert.Equal(t, "success", response.Status)
				assert.NotNil(t, response.Items)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition-data/metabolism"+tt.queryParams, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/v1/nutrition-data/metabolism")

			err := h.GetMetabolism(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.validateFunc != nil {
				tt.validateFunc(t, rec)
			}
		})
	}
}

func TestNutritionDataHandler_GetDrugsNutrition(t *testing.T) {
	// Setup
	e := echo.New()
	h := handlers.NewNutritionDataHandler(nil, "../../nutrition data json")

	// Test cases
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Get drugs nutrition with pagination",
			queryParams:    "?page=1&limit=5",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.APIResponse
				json.Unmarshal(rec.Body.Bytes(), &response)
				assert.Equal(t, "success", response.Status)
				assert.NotNil(t, response.Items)
				assert.NotNil(t, response.Pagination)
			},
		},
		{
			name:           "Get drugs nutrition with search",
			queryParams:    "?q=aspirin",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var response utils.APIResponse
				json.Unmarshal(rec.Body.Bytes(), &response)
				assert.Equal(t, "success", response.Status)
				assert.NotNil(t, response.Items)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition-data/drugs-nutrition"+tt.queryParams, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/v1/nutrition-data/drugs-nutrition")

			err := h.GetDrugsNutrition(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.validateFunc != nil {
				tt.validateFunc(t, rec)
			}
		})
	}
}

// Test error handling
func TestNutritionDataHandler_ErrorHandling(t *testing.T) {
	// Setup
	e := echo.New()
	h := handlers.NewNutritionDataHandler(nil, "../../nutrition data json")

	// Test cases
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "Invalid pagination parameters",
			queryParams:    "?page=invalid&limit=invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Negative pagination values",
			queryParams:    "?page=-1&limit=-1",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Limit too high",
			queryParams:    "?page=1&limit=1000",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Zero limit",
			queryParams:    "?page=1&limit=0",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition-data/recipes"+tt.queryParams, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/v1/nutrition-data/recipes")

			err := h.GetRecipes(c)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}
