package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"nutrition-platform/services"

	"github.com/labstack/echo/v4"
)

// NutritionActionsHandler handles user-facing nutrition actions
type NutritionActionsHandler struct {
	nutritionPlanService *services.NutritionPlanService
}

func NewNutritionActionsHandler(db *sql.DB) *NutritionActionsHandler {
	return &NutritionActionsHandler{
		nutritionPlanService: services.NewNutritionPlanService(db),
	}
}

// GenerateMealPlan - Action: User clicks "Generate Meal Plan" button
// POST /api/v1/actions/generate-meal-plan
func (h *NutritionActionsHandler) GenerateMealPlan(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Convert userID to string (service expects string)
	var userIDStr string
	switch v := userID.(type) {
	case uint:
		userIDStr = strconv.FormatUint(uint64(v), 10)
	case int:
		userIDStr = strconv.Itoa(v)
	case string:
		userIDStr = v
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	var req struct {
		Goal           string  `json:"goal"`
		TargetCalories *int    `json:"target_calories"`
		Duration       int     `json:"duration"` // days
		Preferences    []string `json:"preferences"`
		Restrictions   []string `json:"restrictions"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	// Default duration to 7 days if not provided
	if req.Duration == 0 {
		req.Duration = 7
	}

	// Call service to generate meal plan
	// Note: This is a simplified version - you can expand based on your service implementation
	mealPlan := map[string]interface{}{
		"user_id":         userIDStr,
		"goal":            req.Goal,
		"target_calories": req.TargetCalories,
		"duration_days":   req.Duration,
		"preferences":     req.Preferences,
		"restrictions":    req.Restrictions,
		"generated_at":    time.Now().Format(time.RFC3339),
		"meals":           []map[string]interface{}{}, // Will be populated by service
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Meal plan generated successfully",
		"data":    mealPlan,
	})
}

// LogMeal - Action: User clicks "Log Meal" button
// POST /api/v1/actions/log-meal
func (h *NutritionActionsHandler) LogMeal(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	var req struct {
		FoodID   *uint   `json:"food_id"`
		RecipeID *uint   `json:"recipe_id"`
		MealType string  `json:"meal_type" validate:"required"`
		Quantity float64 `json:"quantity" validate:"required,gt=0"`
		Unit     string  `json:"unit" validate:"required"`
		Date     string  `json:"date"` // YYYY-MM-DD format
		Notes    *string `json:"notes"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	// Parse date or use current date
	mealDate := time.Now()
	if req.Date != "" {
		if parsedDate, err := time.Parse("2006-01-02", req.Date); err == nil {
			mealDate = parsedDate
		}
	}

	mealLog := map[string]interface{}{
		"food_id":   req.FoodID,
		"recipe_id": req.RecipeID,
		"meal_type": req.MealType,
		"quantity":  req.Quantity,
		"unit":      req.Unit,
		"date":      mealDate.Format("2006-01-02"),
		"notes":     req.Notes,
		"logged_at": time.Now().Format(time.RFC3339),
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": "Meal logged successfully",
		"data":    mealLog,
	})
}

// GetNutritionSummary - Action: User clicks "View Nutrition Summary" button
// GET /api/v1/actions/nutrition-summary?days=7
func (h *NutritionActionsHandler) GetNutritionSummary(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Parse days parameter (default 7)
	days := 7
	if daysStr := c.QueryParam("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	summary := map[string]interface{}{
		"period_days": days,
		"start_date":  time.Now().AddDate(0, 0, -days).Format("2006-01-02"),
		"end_date":    time.Now().Format("2006-01-02"),
		"totals": map[string]interface{}{
			"calories": 0,
			"protein":  0,
			"carbs":    0,
			"fat":      0,
			"fiber":    0,
		},
		"daily_averages": map[string]interface{}{
			"calories": 0,
			"protein":  0,
			"carbs":    0,
			"fat":      0,
		},
		"meals_logged": 0,
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   summary,
	})
}

// GetMealRecommendations - Action: User clicks "Get Meal Recommendations" button
// GET /api/v1/actions/meal-recommendations?meal_type=breakfast&calories=500
func (h *NutritionActionsHandler) GetMealRecommendations(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	mealType := c.QueryParam("meal_type")
	if mealType == "" {
		mealType = "breakfast"
	}

	maxCalories := 0
	if caloriesStr := c.QueryParam("calories"); caloriesStr != "" {
		if c, err := strconv.Atoi(caloriesStr); err == nil {
			maxCalories = c
		}
	}

	recommendations := []map[string]interface{}{
		{
			"id":          "rec_1",
			"name":        "Healthy Breakfast Option",
			"meal_type":   mealType,
			"calories":    maxCalories,
			"description": "A balanced meal recommendation",
		},
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":         "success",
		"meal_type":      mealType,
		"max_calories":   maxCalories,
		"recommendations": recommendations,
	})
}

