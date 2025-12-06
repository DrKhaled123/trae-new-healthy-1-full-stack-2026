package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"nutrition-platform/database"
	"nutrition-platform/models"
	"nutrition-platform/repositories"

	"github.com/labstack/echo/v4"
)

// NutritionGoalHandler handles nutrition goal CRUD operations
type NutritionGoalHandler struct {
	userRepo *repositories.UserRepository
}

// NewNutritionGoalHandler creates a new nutrition goal handler
func NewNutritionGoalHandler(db *sql.DB) *NutritionGoalHandler {
	dbWrapper := database.NewDatabase(db)
	return &NutritionGoalHandler{
		userRepo: repositories.NewUserRepository(dbWrapper),
	}
}

// GetGoals returns all nutrition goals for the current user
func (h *NutritionGoalHandler) GetGoals(c echo.Context) error {
	// Get user ID from context
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Convert userID to uint
	var userIDUint uint
	switch v := userID.(type) {
	case uint:
		userIDUint = v
	case string:
		if id, err := strconv.ParseUint(v, 10, 32); err == nil {
			userIDUint = uint(id)
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid user ID",
			})
		}
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	// Get active goals
	goals, err := h.userRepo.GetActiveNutritionGoals(userIDUint)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch goals: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   goals,
	})
}

// GetGoal returns a specific nutrition goal by ID
func (h *NutritionGoalHandler) GetGoal(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	goalID := c.Param("id")
	goalIDUint, err := strconv.ParseUint(goalID, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid goal ID",
		})
	}

	// Get all goals and find the one matching ID
	goals, err := h.userRepo.GetActiveNutritionGoals(userID.(uint))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch goals",
		})
	}

	for _, goal := range goals {
		if goal.ID == int(goalIDUint) {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"status": "success",
				"data":   goal,
			})
		}
	}

	return c.JSON(http.StatusNotFound, map[string]string{
		"error": "Goal not found",
	})
}

// CreateGoal creates a new nutrition goal
func (h *NutritionGoalHandler) CreateGoal(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Convert userID to uint
	var userIDUint uint
	switch v := userID.(type) {
	case uint:
		userIDUint = v
	case string:
		if id, err := strconv.ParseUint(v, 10, 32); err == nil {
			userIDUint = uint(id)
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid user ID",
			})
		}
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	var req struct {
		DailyCalories *int       `json:"daily_calories"`
		ProteinGrams  *float64   `json:"protein_grams"`
		CarbsGrams    *float64   `json:"carbs_grams"`
		FatGrams      *float64   `json:"fat_grams"`
		FiberGrams    *float64   `json:"fiber_grams"`
		SugarGrams    *float64   `json:"sugar_grams"`
		SodiumMg      *int       `json:"sodium_mg"`
		WaterMl       *int       `json:"water_ml"`
		IsActive      bool       `json:"is_active"`
		StartDate     *time.Time `json:"start_date"`
		EndDate       *time.Time `json:"end_date"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Create goal
	goal := &models.NutritionGoal{
		UserID:        userIDUint,
		DailyCalories: req.DailyCalories,
		ProteinGrams:  req.ProteinGrams,
		CarbsGrams:    req.CarbsGrams,
		FatGrams:      req.FatGrams,
		FiberGrams:    req.FiberGrams,
		SugarGrams:    req.SugarGrams,
		SodiumMg:      req.SodiumMg,
		WaterMl:       req.WaterMl,
		IsActive:      req.IsActive,
		StartDate:     req.StartDate,
		EndDate:       req.EndDate,
	}

	err := h.userRepo.CreateNutritionGoal(goal)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create goal: " + err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": "Nutrition goal created successfully",
		"data":    goal,
	})
}

// UpdateGoal updates an existing nutrition goal
func (h *NutritionGoalHandler) UpdateGoal(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	goalID := c.Param("id")
	goalIDUint, err := strconv.ParseUint(goalID, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid goal ID",
		})
	}

	var req struct {
		DailyCalories *int       `json:"daily_calories"`
		ProteinGrams  *float64   `json:"protein_grams"`
		CarbsGrams    *float64   `json:"carbs_grams"`
		FatGrams      *float64   `json:"fat_grams"`
		FiberGrams    *float64   `json:"fiber_grams"`
		SugarGrams    *float64   `json:"sugar_grams"`
		SodiumMg      *int       `json:"sodium_mg"`
		WaterMl       *int       `json:"water_ml"`
		IsActive      *bool      `json:"is_active"`
		StartDate     *time.Time `json:"start_date"`
		EndDate       *time.Time `json:"end_date"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Get existing goal
	goals, err := h.userRepo.GetActiveNutritionGoals(userID.(uint))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch goals",
		})
	}

	var existingGoal *models.NutritionGoal
	for _, goal := range goals {
		if goal.ID == int(goalIDUint) {
			existingGoal = goal
			break
		}
	}

	if existingGoal == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Goal not found",
		})
	}

	// Update fields
	if req.DailyCalories != nil {
		existingGoal.DailyCalories = req.DailyCalories
	}
	if req.ProteinGrams != nil {
		existingGoal.ProteinGrams = req.ProteinGrams
	}
	if req.CarbsGrams != nil {
		existingGoal.CarbsGrams = req.CarbsGrams
	}
	if req.FatGrams != nil {
		existingGoal.FatGrams = req.FatGrams
	}
	if req.FiberGrams != nil {
		existingGoal.FiberGrams = req.FiberGrams
	}
	if req.SugarGrams != nil {
		existingGoal.SugarGrams = req.SugarGrams
	}
	if req.SodiumMg != nil {
		existingGoal.SodiumMg = req.SodiumMg
	}
	if req.WaterMl != nil {
		existingGoal.WaterMl = req.WaterMl
	}
	if req.IsActive != nil {
		existingGoal.IsActive = *req.IsActive
	}
	if req.StartDate != nil {
		existingGoal.StartDate = req.StartDate
	}
	if req.EndDate != nil {
		existingGoal.EndDate = req.EndDate
	}

	// Update in database
	err = h.userRepo.UpdateNutritionGoal(existingGoal)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update goal: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Nutrition goal updated successfully",
		"data":    existingGoal,
	})
}

// DeleteGoal deletes a nutrition goal
func (h *NutritionGoalHandler) DeleteGoal(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	goalID := c.Param("id")
	goalIDUint, err := strconv.ParseUint(goalID, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid goal ID",
		})
	}

	// Get existing goal and deactivate it
	goals, err := h.userRepo.GetActiveNutritionGoals(userID.(uint))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch goals",
		})
	}

	var existingGoal *models.NutritionGoal
	for _, goal := range goals {
		if goal.ID == int(goalIDUint) {
			existingGoal = goal
			break
		}
	}

	if existingGoal == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Goal not found",
		})
	}

	// Delete goal (deactivate)
	err = h.userRepo.DeleteNutritionGoal(int(goalIDUint), userID.(uint))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete goal: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Nutrition goal deleted successfully",
	})
}
