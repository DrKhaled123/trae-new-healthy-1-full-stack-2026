package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// APIResponse represents a standardized API response for external integrations
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Timestamp string      `json:"timestamp"`
	RequestID string      `json:"request_id"`
	Meta      *APIMeta    `json:"meta,omitempty"`
}

// APIMeta provides API-specific metadata
type APIMeta struct {
	APIVersion    string `json:"api_version"`
	RateLimit     int    `json:"rate_limit"`
	RateRemaining int    `json:"rate_remaining"`
	RateReset     int64  `json:"rate_reset"`
	Total         int    `json:"total,omitempty"`
	Page          int    `json:"page,omitempty"`
	PerPage       int    `json:"per_page,omitempty"`
	TotalPages    int    `json:"total_pages,omitempty"`
}

// GetMealsAPI returns meals data for external API consumers
func GetMealsAPI(c echo.Context) error {
	// Validate API key access
	if err := validateNutritionAccess(c, "read"); err != nil {
		return err
	}

	// Extract query parameters
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	category := c.QueryParam("category")
	cuisine := c.QueryParam("cuisine")
	dietary := c.QueryParam("dietary")
	lang := c.QueryParam("lang")
	if lang == "" {
		lang = "en"
	}

	// Mock meals data for API
	mealsData := []map[string]interface{}{
		{
			"id":          "api_meal_001",
			"name":        "Mediterranean Quinoa Bowl",
			"description": "Healthy bowl with quinoa, vegetables, and lean protein",
			"category":    "healthy",
			"cuisine":     "mediterranean",
			"prep_time":   15,
			"cook_time":   20,
			"servings":    2,
			"difficulty":  "easy",
			"calories":    450,
			"protein":     25.5,
			"carbs":       35.2,
			"fat":         18.7,
			"fiber":       8.3,
			"sodium":      680,
			"tags":        []string{"healthy", "protein-rich", "gluten-free"},
			"allergens":   []string{},
			"dietary":     []string{"halal", "gluten-free"},
			"created_at":  "2024-01-15T10:30:00Z",
			"updated_at":  "2024-01-15T10:30:00Z",
		},
		{
			"id":          "api_meal_002",
			"name":        "Grilled Salmon with Vegetables",
			"description": "Omega-3 rich salmon with seasonal vegetables",
			"category":    "healthy",
			"cuisine":     "international",
			"prep_time":   10,
			"cook_time":   25,
			"servings":    1,
			"difficulty":  "medium",
			"calories":    380,
			"protein":     32.0,
			"carbs":       12.5,
			"fat":         22.3,
			"fiber":       6.8,
			"sodium":      420,
			"tags":        []string{"high-protein", "omega-3", "low-carb"},
			"allergens":   []string{"fish"},
			"dietary":     []string{"halal", "keto-friendly"},
			"created_at":  "2024-01-15T11:00:00Z",
			"updated_at":  "2024-01-15T11:00:00Z",
		},
	}

	// Apply filters
	filteredMeals := filterMealsByAPI(mealsData, category, cuisine, dietary)

	// Apply pagination
	start := (page - 1) * perPage
	end := start + perPage
	if end > len(filteredMeals) {
		end = len(filteredMeals)
	}
	if start > len(filteredMeals) {
		start = len(filteredMeals)
	}

	paginatedMeals := filteredMeals[start:end]
	totalPages := (len(filteredMeals) + perPage - 1) / perPage

	// Get rate limit info from context
	rateLimit := c.Get("rate_limit")
	rateRemaining := c.Get("rate_remaining")
	rateReset := c.Get("rate_reset")

	meta := &APIMeta{
		APIVersion:    "1.0",
		RateLimit:     getIntFromContext(rateLimit, 1000),
		RateRemaining: getIntFromContext(rateRemaining, 999),
		RateReset:     getInt64FromContext(rateReset, time.Now().Add(time.Hour).Unix()),
		Total:         len(filteredMeals),
		Page:          page,
		PerPage:       perPage,
		TotalPages:    totalPages,
	}

	response := APIResponse{
		Success:   true,
		Message:   "Meals retrieved successfully",
		Data:      paginatedMeals,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
		Meta:      meta,
	}

	// Set rate limit headers
	c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(meta.RateLimit))
	c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(meta.RateRemaining))
	c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(meta.RateReset, 10))
	c.Response().Header().Set("Cache-Control", "public, max-age=300")

	return c.JSON(http.StatusOK, response)
}

// CreateMealAPI creates a new meal via API
func CreateMealAPI(c echo.Context) error {
	// Validate API key access
	if err := validateNutritionAccess(c, "write"); err != nil {
		return err
	}

	// Parse request body
	var mealData map[string]interface{}
	if err := c.Bind(&mealData); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Validate required fields
	requiredFields := []string{"name", "description", "category", "prep_time", "cook_time", "servings"}
	for _, field := range requiredFields {
		if _, exists := mealData[field]; !exists {
			return echo.NewHTTPError(http.StatusBadRequest, "Missing required field: "+field)
		}
	}

	// Generate ID and timestamps
	mealData["id"] = "api_meal_" + generateID()
	mealData["created_at"] = time.Now().UTC().Format(time.RFC3339)
	mealData["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	mealData["created_by_api"] = true
	mealData["api_key_id"] = c.Get("api_key_id")

	// In production, this would save to database
	// For now, return the created meal data

	response := APIResponse{
		Success:   true,
		Message:   "Meal created successfully",
		Data:      mealData,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
		Meta: &APIMeta{
			APIVersion: "1.0",
		},
	}

	return c.JSON(http.StatusCreated, response)
}

// GetMealAPI retrieves a specific meal by ID
func GetMealAPI(c echo.Context) error {
	// Validate API key access
	if err := validateNutritionAccess(c, "read"); err != nil {
		return err
	}

	mealID := c.Param("id")
	if mealID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Meal ID is required")
	}

	// Mock meal data - in production, fetch from database
	mealData := map[string]interface{}{
		"id":          mealID,
		"name":        "Mediterranean Quinoa Bowl",
		"description": "Healthy bowl with quinoa, vegetables, and lean protein",
		"category":    "healthy",
		"cuisine":     "mediterranean",
		"prep_time":   15,
		"cook_time":   20,
		"servings":    2,
		"difficulty":  "easy",
		"nutrition": map[string]interface{}{
			"calories": 450,
			"protein":  25.5,
			"carbs":    35.2,
			"fat":      18.7,
			"fiber":    8.3,
			"sodium":   680,
		},
		"ingredients": []map[string]interface{}{
			{
				"name":   "Quinoa",
				"amount": "1 cup",
				"unit":   "cup",
			},
			{
				"name":   "Chicken Breast",
				"amount": "200g",
				"unit":   "grams",
			},
		},
		"instructions": []string{
			"Cook quinoa according to package instructions",
			"Grill chicken breast until cooked through",
			"Combine all ingredients in a bowl",
		},
		"tags":       []string{"healthy", "protein-rich", "gluten-free"},
		"allergens":  []string{},
		"dietary":    []string{"halal", "gluten-free"},
		"created_at": "2024-01-15T10:30:00Z",
		"updated_at": "2024-01-15T10:30:00Z",
	}

	response := APIResponse{
		Success:   true,
		Message:   "Meal retrieved successfully",
		Data:      mealData,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
		Meta: &APIMeta{
			APIVersion: "1.0",
		},
	}

	c.Response().Header().Set("Cache-Control", "public, max-age=600")
	return c.JSON(http.StatusOK, response)
}

// UpdateMealAPI updates an existing meal
func UpdateMealAPI(c echo.Context) error {
	// Validate API key access
	if err := validateNutritionAccess(c, "write"); err != nil {
		return err
	}

	mealID := c.Param("id")
	if mealID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Meal ID is required")
	}

	// Parse request body
	var updateData map[string]interface{}
	if err := c.Bind(&updateData); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Add update timestamp
	updateData["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	updateData["updated_by_api"] = true
	updateData["api_key_id"] = c.Get("api_key_id")

	// In production, this would update the database record
	// For now, return the updated data
	updateData["id"] = mealID

	response := APIResponse{
		Success:   true,
		Message:   "Meal updated successfully",
		Data:      updateData,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
		Meta: &APIMeta{
			APIVersion: "1.0",
		},
	}

	return c.JSON(http.StatusOK, response)
}

// DeleteMealAPI deletes a meal
func DeleteMealAPI(c echo.Context) error {
	// Validate API key access
	if err := validateNutritionAccess(c, "write"); err != nil {
		return err
	}

	mealID := c.Param("id")
	if mealID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Meal ID is required")
	}

	// In production, this would delete from database
	// For now, return success response

	response := APIResponse{
		Success:   true,
		Message:   "Meal deleted successfully",
		Data:      map[string]string{"id": mealID, "status": "deleted"},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
		Meta: &APIMeta{
			APIVersion: "1.0",
		},
	}

	return c.JSON(http.StatusOK, response)
}

// Helper functions

// filterMealsByAPI applies filters to meals data for API responses
func filterMealsByAPI(meals []map[string]interface{}, category, cuisine, dietary string) []map[string]interface{} {
	filtered := make([]map[string]interface{}, 0)

	for _, meal := range meals {
		// Apply category filter
		if category != "" {
			if mealCategory, ok := meal["category"].(string); !ok || mealCategory != category {
				continue
			}
		}

		// Apply cuisine filter
		if cuisine != "" {
			if mealCuisine, ok := meal["cuisine"].(string); !ok || mealCuisine != cuisine {
				continue
			}
		}

		// Apply dietary filter
		if dietary != "" {
			if dietaryList, ok := meal["dietary"].([]string); ok {
				if !contains(dietaryList, dietary) {
					continue
				}
			} else {
				continue
			}
		}

		filtered = append(filtered, meal)
	}

	return filtered
}

// getIntFromContext safely extracts int from context
func getIntFromContext(value interface{}, defaultValue int) int {
	if value == nil {
		return defaultValue
	}
	if intVal, ok := value.(int); ok {
		return intVal
	}
	return defaultValue
}

// getInt64FromContext safely extracts int64 from context
func getInt64FromContext(value interface{}, defaultValue int64) int64 {
	if value == nil {
		return defaultValue
	}
	if int64Val, ok := value.(int64); ok {
		return int64Val
	}
	return defaultValue
}

// generateID generates a simple ID for demo purposes
func generateID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// validateNutritionAccess validates API key access for nutrition endpoints
func validateNutritionAccess(c echo.Context, permission string) error {
	// Get API key from header
	apiKey := c.Request().Header.Get("X-API-Key")
	if apiKey == "" {
		apiKey = c.QueryParam("api_key")
	}

	if apiKey == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "API key required")
	}

	// In production, validate API key against database
	// For now, accept any non-empty key for demo purposes
	if apiKey != "demo-nutrition-key" && apiKey != "prod-nutrition-key" {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API key")
	}

	// Set API key info in context
	c.Set("api_key_id", "key_12345")
	c.Set("rate_limit", 1000)
	c.Set("rate_remaining", 999)
	c.Set("rate_reset", time.Now().Add(time.Hour).Unix())

	return nil
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
