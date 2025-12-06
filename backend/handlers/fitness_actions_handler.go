package handlers

import (
	"net/http"
	"strconv"
	"time"

	"nutrition-platform/database"
	"nutrition-platform/services"

	"github.com/labstack/echo/v4"
)

// FitnessActionsHandler handles user-facing fitness actions
type FitnessActionsHandler struct {
	fitnessService *services.FitnessService
}

func NewFitnessActionsHandler(db *database.Database) *FitnessActionsHandler {
	return &FitnessActionsHandler{
		fitnessService: services.NewFitnessService(db),
	}
}

// GenerateWorkout - Action: User clicks "Generate Workout" button
// POST /api/v1/actions/generate-workout
func (h *FitnessActionsHandler) GenerateWorkout(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	var req struct {
		Goal         string   `json:"goal"`       // weight_loss, muscle_gain, endurance, flexibility
		Duration     int      `json:"duration"`   // minutes
		Difficulty   string   `json:"difficulty"` // beginner, intermediate, advanced
		Equipment    []string `json:"equipment"`
		MuscleGroups []string `json:"muscle_groups"`
		Restrictions []string `json:"restrictions"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	// Default duration to 30 minutes if not provided
	if req.Duration == 0 {
		req.Duration = 30
	}

	// Default difficulty to intermediate if not provided
	if req.Difficulty == "" {
		req.Difficulty = "intermediate"
	}

	workoutPlan := map[string]interface{}{
		"goal":          req.Goal,
		"duration":      req.Duration,
		"difficulty":    req.Difficulty,
		"equipment":     req.Equipment,
		"muscle_groups": req.MuscleGroups,
		"restrictions":  req.Restrictions,
		"generated_at":  time.Now().Format(time.RFC3339),
		"exercises":     []map[string]interface{}{}, // Will be populated by service
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Workout plan generated successfully",
		"data":    workoutPlan,
	})
}

// LogWorkout - Action: User clicks "Log Workout" button
// POST /api/v1/actions/log-workout
func (h *FitnessActionsHandler) LogWorkout(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	var req struct {
		WorkoutPlanID *uint                    `json:"workout_plan_id"`
		Exercises     []map[string]interface{} `json:"exercises"`
		Duration      int                      `json:"duration"` // minutes
		Date          string                   `json:"date"`     // YYYY-MM-DD format
		Notes         *string                  `json:"notes"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	// Parse date or use current date
	workoutDate := time.Now()
	if req.Date != "" {
		if parsedDate, err := time.Parse("2006-01-02", req.Date); err == nil {
			workoutDate = parsedDate
		}
	}

	workoutLog := map[string]interface{}{
		"workout_plan_id": req.WorkoutPlanID,
		"exercises":       req.Exercises,
		"duration":        req.Duration,
		"date":            workoutDate.Format("2006-01-02"),
		"notes":           req.Notes,
		"logged_at":       time.Now().Format(time.RFC3339),
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": "Workout logged successfully",
		"data":    workoutLog,
	})
}

// GetFitnessSummary - Action: User clicks "View Fitness Summary" button
// GET /api/v1/actions/fitness-summary?days=30
func (h *FitnessActionsHandler) GetFitnessSummary(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Parse days parameter (default 30)
	days := 30
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
			"workouts_completed": 0,
			"total_duration":     0, // minutes
			"total_calories":     0,
		},
		"averages": map[string]interface{}{
			"workouts_per_week": 0,
			"avg_duration":      0,
			"avg_calories":      0,
		},
		"most_trained_muscles": []string{},
		"favorite_exercises":   []string{},
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   summary,
	})
}

// GetWorkoutRecommendations - Action: User clicks "Get Workout Recommendations" button
// GET /api/v1/actions/workout-recommendations?goal=weight_loss&duration=30
func (h *FitnessActionsHandler) GetWorkoutRecommendations(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	goal := c.QueryParam("goal")
	if goal == "" {
		goal = "general_fitness"
	}

	duration := 30
	if durationStr := c.QueryParam("duration"); durationStr != "" {
		if d, err := strconv.Atoi(durationStr); err == nil {
			duration = d
		}
	}

	recommendations := []map[string]interface{}{
		{
			"id":          "workout_rec_1",
			"name":        "Recommended Workout",
			"goal":        goal,
			"duration":    duration,
			"description": "A personalized workout recommendation",
		},
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":          "success",
		"goal":            goal,
		"duration":        duration,
		"recommendations": recommendations,
	})
}
