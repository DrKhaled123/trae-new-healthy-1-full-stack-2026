// frontend.go
package main

import (
	"net/http"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

// NutritionRequest represents a nutrition analysis request
type NutritionRequest struct {
	Food     string  `json:"food"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
	MealType string  `json:"meal_type"`
}

// NutritionResponse represents nutrition analysis response
type NutritionResponse struct {
	Food            string             `json:"food"`
	Quantity        float64            `json:"quantity"`
	Unit            string             `json:"unit"`
	Calories        float64            `json:"calories"`
	Protein         float64            `json:"protein"`
	Carbs           float64            `json:"carbs"`
	Fat             float64            `json:"fat"`
	Fiber           float64            `json:"fiber"`
	Sugar           float64            `json:"sugar"`
	Sodium          float64            `json:"sodium"`
	Vitamins        map[string]float64 `json:"vitamins"`
	Minerals        map[string]float64 `json:"minerals"`
	HealthScore     int                `json:"health_score"`
	Recommendations []string           `json:"recommendations"`
}

// MealPlanRequest represents a meal plan generation request
type MealPlanRequest struct {
	Goals               []string `json:"goals"`
	DietaryRestrictions []string `json:"dietary_restrictions"`
	CaloriesTarget      float64  `json:"calories_target"`
	MealsPerDay         int      `json:"meals_per_day"`
	Duration            int      `json:"duration_days"`
}

// generateNutritionAnalysis generates nutrition analysis for a food item
func generateNutritionAnalysis(req NutritionRequest) NutritionResponse {
	// This is a placeholder implementation
	// In a real application, you would:
	// 1. Look up the food in a nutrition database
	// 2. Calculate nutrition based on quantity
	// 3. Apply any relevant multipliers
	return NutritionResponse{
		Food:            req.Food,
		Quantity:        req.Quantity,
		Unit:            req.Unit,
		Calories:        250.0,
		Protein:         25.0,
		Carbs:           30.0,
		Fat:             10.0,
		Fiber:           5.0,
		Sugar:           8.0,
		Sodium:          300.0,
		Vitamins:        map[string]float64{"A": 1000, "C": 50, "D": 200},
		Minerals:        map[string]float64{"Iron": 8, "Calcium": 200},
		HealthScore:     85,
		Recommendations: []string{"Good source of protein", "Consider adding more vegetables"},
	}
}

// generateMealPlan generates a meal plan based on request
func generateMealPlan(c echo.Context) error {
	// This is a placeholder implementation
	// In a real application, you would:
	// 1. Analyze the user's requirements
	// 2. Generate appropriate meals
	// 3. Ensure nutritional balance
	// 4. Consider dietary restrictions
	return c.JSON(http.StatusOK, map[string]interface{}{
		"meal_plan": map[string]interface{}{
			"id":             "new-meal-plan-id",
			"name":           "Balanced Nutrition Plan",
			"description":    "A balanced meal plan for general health",
			"duration":       7,
			"meals_per_day":  3,
			"total_calories": 2100,
			"meals": []interface{}{
				map[string]interface{}{
					"day": 1,
					"meals": []interface{}{
						map[string]interface{}{
							"name":     "Breakfast",
							"calories": 700,
							"foods":    []string{"Oatmeal", "Banana", "Almonds"},
						},
						map[string]interface{}{
							"name":     "Lunch",
							"calories": 700,
							"foods":    []string{"Grilled Chicken", "Brown Rice", "Vegetables"},
						},
						map[string]interface{}{
							"name":     "Dinner",
							"calories": 700,
							"foods":    []string{"Salmon", "Quinoa", "Salad"},
						},
					},
				},
			},
		},
		"generated_at": "now",
	})
}

func SetupFrontendRoutes(e *echo.Echo, config Config) {
	// Serve static files (CSS, JS, images, etc.)
	e.Static("/assets", "./frontend/build/static")  // React build static files
	e.Static("/static", "./frontend/build/static")  // Alternative path
	e.Static("/css", "./frontend/build/static/css") // Specific CSS directory
	e.Static("/js", "./frontend/build/static/js")   // Specific JS directory
	e.Static("/images", "./frontend/public/images") // Images directory
	e.Static("/img", "./frontend/public/images")    // Alternative images path

	// Health check - must come before the catch-all route
	e.GET("/health", healthHandler)

	// SPA Catch-all handler for client-side routing
	// This must be the last route defined, as it will handle all unmatched routes
	spaHandler := createSPAHandler("./frontend/build/index.html", config)
	e.GET("/*", spaHandler)

	// Special case: handle API routes that should respond with 404 for SPA
	e.GET("/api", func(c echo.Context) error {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error":   "API endpoint not found",
			"message": "This looks like an API request but no matching endpoint was found",
		})
	})
}

func createSPAHandler(indexPath string, config Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skip API routes - they should be handled by other handlers
		if len(c.Request().URL.Path) >= 4 && c.Request().URL.Path[:4] == "/api" {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error":   "API endpoint not found",
				"message": "This API endpoint does not exist",
			})
		}

		// Skip static asset routes - they should be handled by Static middleware
		staticPaths := []string{"/static/", "/css/", "/js/", "/images/", "/img/", "/assets/"}
		path := c.Request().URL.Path
		for _, staticPath := range staticPaths {
			if len(path) >= len(staticPath) && path[:len(staticPath)] == staticPath {
				// This should not happen as static middleware should handle it
				return c.JSON(http.StatusNotFound, map[string]interface{}{
					"error": "Static file not found",
				})
			}
		}

		// Try to serve the index.html file for SPA routing
		if err := serveIndexFile(c, indexPath); err != nil {
			// If index.html doesn't exist or can't be read, return a helpful error
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "Frontend application not found",
				"message": "The frontend application is not built or accessible",
				"details": err.Error(),
			})
		}

		return nil
	}
}

func serveIndexFile(c echo.Context, indexPath string) error {
	// Check if the index file exists
	if _, err := filepath.Abs(indexPath); err != nil {
		return err
	}

	// Set appropriate headers for HTML
	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Response().Header().Set("Pragma", "no-cache")
	c.Response().Header().Set("Expires", "0")

	// Serve the file
	return c.File(indexPath)
}

// SetupAPIRoutes sets up all API routes
func SetupAPIRoutes(e *echo.Echo, config Config) {
	api := e.Group("/api/v1")

	// Apply security middleware
	api.Use(SecurityMiddleware())
	api.Use(RequestValidationMiddleware())

	// Apply rate limiting (but not to health checks)
	api.Use(GetRateLimiter().GetRateLimitMiddleware())

	// Apply request logging
	api.Use(RequestLoggerMiddleware())

	// Quote endpoints
	api.GET("/quote", getQuoteHandler)
	api.POST("/quote/analyze", analyzeQuoteHandler)

	// User management endpoints
	api.POST("/users", createUserHandler)
	api.GET("/users/:id", getUserHandler)
	api.PUT("/users/:id", updateUserHandler)
	api.DELETE("/users/:id", deleteUserHandler)

	// API key management
	api.POST("/auth/login", loginHandler)
	api.POST("/auth/register", registerHandler)
	api.POST("/auth/logout", logoutHandler)

	// Protected routes (require API key)
	protected := api.Group("", AuthMiddleware())

	// Nutrition analysis endpoints
	protected.POST("/nutrition/analyze", analyzeNutritionHandler)
	protected.POST("/nutrition/batch-analyze", batchNutritionAnalysisHandler)

	// Meal planning
	protected.POST("/meal-plans/generate", generateMealPlanHandler)
	protected.GET("/meal-plans", getMealPlansHandler)
	protected.GET("/meal-plans/:id", getMealPlanHandler)
	protected.PUT("/meal-plans/:id", updateMealPlanHandler)
	protected.DELETE("/meal-plans/:id", deleteMealPlanHandler)

	// Exercise planning
	protected.POST("/workouts/generate", generateWorkoutHandler)
	protected.GET("/workouts", getWorkoutsHandler)
	protected.GET("/workouts/:id", getWorkoutHandler)
	protected.PUT("/workouts/:id", updateWorkoutHandler)
	protected.DELETE("/workouts/:id", deleteWorkoutHandler)

	// Recipe management
	protected.POST("/recipes", createRecipeHandler)
	protected.GET("/recipes", getRecipesHandler)
	protected.GET("/recipes/:id", getRecipeHandler)
	protected.PUT("/recipes/:id", updateRecipeHandler)
	protected.DELETE("/recipes/:id", deleteRecipeHandler)
	protected.GET("/recipes/search", searchRecipesHandler)

	// Health tracking
	protected.POST("/health/log", logHealthDataHandler)
	protected.GET("/health/history", getHealthHistoryHandler)

	// Analytics and reporting
	protected.GET("/analytics/usage", getUsageAnalyticsHandler)
	protected.GET("/analytics/nutrition", getNutritionAnalyticsHandler)

	// Admin routes (require admin authentication)
	admin := api.Group("/admin", BasicAuthMiddleware("admin", config.JWTSecret))
	admin.GET("/users", getAllUsersHandler)
	admin.GET("/stats", getSystemStatsHandler)
	admin.POST("/config/reload", reloadConfigHandler)

	// File upload endpoints (protected)
	protected.POST("/upload/image", uploadImageHandler)
	protected.POST("/upload/document", uploadDocumentHandler)

	// Export endpoints
	protected.GET("/export/nutrition-data", exportNutritionDataHandler)
	protected.GET("/export/health-data", exportHealthDataHandler)
}

// Placeholder handlers (replace these with your actual implementations)
// These are kept simple for now - integrate with your existing handlers

func getQuoteHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"quote":    "Success is not final, failure is not fatal: it is the courage to continue that counts.",
		"author":   "Winston Churchill",
		"category": "motivation",
	})
}

func analyzeQuoteHandler(c echo.Context) error {
	req := make(map[string]interface{})
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"analysis":  "Quote has positive motivational impact",
		"sentiment": "positive",
		"themes":    []string{"success", "resilience", "courage"},
	})
}

func createUserHandler(c echo.Context) error {
	req := make(map[string]interface{})
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":      "new-user-id",
		"message": "User created successfully",
		"status":  "active",
	})
}

// Add more placeholder handlers as needed...

func getUserHandler(c echo.Context) error {
	id := c.Param("id")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id": id,
		"user": map[string]interface{}{
			"name":       "User " + id,
			"email":      "user" + id + "@example.com",
			"status":     "active",
			"created_at": "2023-01-01T00:00:00Z",
		},
	})
}

func updateUserHandler(c echo.Context) error {
	id := c.Param("id")
	req := make(map[string]interface{})
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "User updated successfully",
		"id":      id,
	})
}

func deleteUserHandler(c echo.Context) error {
	id := c.Param("id")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":      id,
		"deleted": true,
		"message": "User deleted successfully",
	})
}

func loginHandler(c echo.Context) error {
	req := make(map[string]interface{})
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token": "***jwt-token-here***",
		"token_type":   "Bearer",
		"expires_in":   3600,
		"user": map[string]interface{}{
			"id":    "user123",
			"email": "user@example.com",
			"name":  "John Doe",
		},
	})
}

func registerHandler(c echo.Context) error {
	req := make(map[string]interface{})
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message":               "User registered successfully",
		"verification_required": false,
	})
}

func logoutHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Logged out successfully",
	})
}

func analyzeNutritionHandler(c echo.Context) error {
	var req NutritionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	response := generateNutritionAnalysis(req)
	return c.JSON(http.StatusOK, response)
}

func batchNutritionAnalysisHandler(c echo.Context) error {
	req := make([]NutritionRequest, 0)
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	results := make([]NutritionResponse, 0, len(req))
	for _, item := range req {
		result := generateNutritionAnalysis(item)
		results = append(results, result)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}

func generateMealPlanHandler(c echo.Context) error {
	var req MealPlanRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	return generateMealPlan(c)
}

func generateWorkoutHandler(c echo.Context) error {
	wReq := make(map[string]interface{})
	if err := c.Bind(&wReq); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Workout plan generated successfully",
	})
}

// Continue adding placeholder handlers for all routes...
func getMealPlansHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"meal_plans": []interface{}{},
		"count":      0,
	})
}

func getMealPlanHandler(c echo.Context) error {
	id := c.Param("id")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":   id,
		"plan": map[string]interface{}{},
	})
}

func updateMealPlanHandler(c echo.Context) error {
	id := c.Param("id")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":      id,
		"updated": true,
	})
}

func deleteMealPlanHandler(c echo.Context) error {
	id := c.Param("id")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":      id,
		"deleted": true,
	})
}

func getWorkoutsHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"workouts": []interface{}{},
		"count":    0,
	})
}

func getWorkoutHandler(c echo.Context) error {
	id := c.Param("id")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":      id,
		"workout": map[string]interface{}{},
	})
}

func updateWorkoutHandler(c echo.Context) error {
	id := c.Param("id")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":      id,
		"updated": true,
	})
}

func deleteWorkoutHandler(c echo.Context) error {
	id := c.Param("id")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":      id,
		"deleted": true,
	})
}

// Recipe management
func createRecipeHandler(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":      "new-recipe-id",
		"created": true,
	})
}

func getRecipesHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"recipes": []interface{}{},
		"count":   0,
	})
}

func getRecipeHandler(c echo.Context) error {
	id := c.Param("id")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":     id,
		"recipe": map[string]interface{}{},
	})
}

func updateRecipeHandler(c echo.Context) error {
	id := c.Param("id")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":      id,
		"updated": true,
	})
}

func deleteRecipeHandler(c echo.Context) error {
	id := c.Param("id")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":      id,
		"deleted": true,
	})
}

func searchRecipesHandler(c echo.Context) error {
	query := c.QueryParam("q")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"query":   query,
		"results": []interface{}{},
		"count":   0,
	})
}

// Health tracking
func logHealthDataHandler(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":     "health-log-id",
		"logged": true,
	})
}

func getHealthHistoryHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"history": []interface{}{},
		"count":   0,
	})
}

// Analytics
func getUsageAnalyticsHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"analytics": map[string]interface{}{
			"total_requests":        0,
			"unique_users":          0,
			"most_popular_endpoint": "",
		},
	})
}

func getNutritionAnalyticsHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"analytics": map[string]interface{}{
			"popular_foods":         []interface{}{},
			"common_nutrients":      []interface{}{},
			"avg_calories_per_meal": 0,
		},
	})
}

// Admin handlers
func getAllUsersHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"users": []interface{}{},
		"count": 0,
	})
}

func getSystemStatsHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"stats": map[string]interface{}{
			"uptime":          "0s",
			"total_users":     0,
			"total_requests":  0,
			"database_status": "ok",
			"memory_usage":    "0MB",
			"disk_usage":      "0GB",
		},
	})
}

func reloadConfigHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":      "Configuration reloaded successfully",
		"reloaded_at":  "now",
		"new_settings": map[string]interface{}{},
	})
}

// File upload handlers
func uploadImageHandler(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":   "uploaded-file-id",
		"url":  "https://example.com/uploads/image.jpg",
		"size": 12345,
	})
}

func uploadDocumentHandler(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":   "uploaded-doc-id",
		"url":  "https://example.com/uploads/document.pdf",
		"size": 67890,
	})
}

// Export handlers
func exportNutritionDataHandler(c echo.Context) error {
	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=nutrition-data.json")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"export_data": []interface{}{},
		"exported_at": "now",
		"format":      "json",
	})
}

func exportHealthDataHandler(c echo.Context) error {
	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=health-data.json")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"export_data": []interface{}{},
		"exported_at": "now",
		"format":      "json",
	})
}
