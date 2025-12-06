package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"nutrition-platform/database"
	"nutrition-platform/models"
	"nutrition-platform/repositories"

	"github.com/labstack/echo/v4"
)

// WorkoutHandler handles workout-related requests
type WorkoutHandler struct {
	workoutRepo *repositories.WorkoutRepository
}

// NewWorkoutHandler creates a new WorkoutHandler instance
func NewWorkoutHandler(db *sql.DB) *WorkoutHandler {
	dbWrapper := database.NewDatabase(db)
	return &WorkoutHandler{
		workoutRepo: repositories.NewWorkoutRepository(dbWrapper),
	}
}

// LogWorkout logs a new workout session
func (h *WorkoutHandler) LogWorkout(c echo.Context) error {
	// Get user ID from context (from JWT middleware)
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Convert userID to string
	var userIDStr string
	switch v := userID.(type) {
	case string:
		userIDStr = v
	case int64:
		userIDStr = strconv.FormatInt(v, 10)
	case int:
		userIDStr = strconv.Itoa(v)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	var req models.CreateUserWorkoutSessionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	// Validate required fields
	if req.Status == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Status is required",
		})
	}

	session := &models.UserWorkoutSession{
		UserID:             userIDStr,
		WorkoutSessionID:   req.WorkoutSessionID,
		WorkoutProgramID:   req.WorkoutProgramID,
		ScheduledDate:      req.ScheduledDate,
		CompletedDate:      req.CompletedDate,
		DurationMinutes:    req.DurationMinutes,
		CaloriesBurned:     req.CaloriesBurned,
		PerceivedExertion:  req.PerceivedExertion,
		MoodBefore:         req.MoodBefore,
		MoodAfter:          req.MoodAfter,
		ExercisesCompleted: req.ExercisesCompleted,
		ExercisesSkipped:   req.ExercisesSkipped,
		ModificationsUsed:  req.ModificationsUsed,
		Notes:              req.Notes,
		InjuriesReported:   req.InjuriesReported,
		Status:             req.Status,
	}

	if err := h.workoutRepo.CreateUserWorkoutSession(session); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to log workout: " + err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, session)
}

// GetWorkouts retrieves workout sessions for a user
func (h *WorkoutHandler) GetWorkouts(c echo.Context) error {
	// Get user ID from context (from JWT middleware)
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Convert userID to string
	var userIDStr string
	switch v := userID.(type) {
	case string:
		userIDStr = v
	case int64:
		userIDStr = strconv.FormatInt(v, 10)
	case int:
		userIDStr = strconv.Itoa(v)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	// Parse pagination parameters
	page := 1
	limit := 20
	if pageStr := c.QueryParam("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	sessions, total, err := h.workoutRepo.GetUserWorkoutSessions(userIDStr, page, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get workouts: " + err.Error(),
		})
	}

	totalPages := (int(total) + limit - 1) / limit
	return c.JSON(http.StatusOK, map[string]interface{}{
		"workouts":    sessions,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

// GetWorkout retrieves a specific workout session by ID
func (h *WorkoutHandler) GetWorkout(c echo.Context) error {
	id := c.Param("id")

	// Get user ID from context (from JWT middleware)
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Convert userID to string
	var userIDStr string
	switch v := userID.(type) {
	case string:
		userIDStr = v
	case int64:
		userIDStr = strconv.FormatInt(v, 10)
	case int:
		userIDStr = strconv.Itoa(v)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	session, err := h.workoutRepo.GetUserWorkoutSessionByID(id, userIDStr)
	if err != nil {
		if err.Error() == "workout session not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Workout session not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get workout: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, session)
}

// UpdateWorkout updates an existing workout session
func (h *WorkoutHandler) UpdateWorkout(c echo.Context) error {
	id := c.Param("id")

	// Get user ID from context (from JWT middleware)
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Convert userID to string
	var userIDStr string
	switch v := userID.(type) {
	case string:
		userIDStr = v
	case int64:
		userIDStr = strconv.FormatInt(v, 10)
	case int:
		userIDStr = strconv.Itoa(v)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	// Get existing session
	existing, err := h.workoutRepo.GetUserWorkoutSessionByID(id, userIDStr)
	if err != nil {
		if err.Error() == "workout session not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Workout session not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get workout: " + err.Error(),
		})
	}

	var req models.CreateUserWorkoutSessionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	// Update fields if provided
	if req.WorkoutSessionID != nil {
		existing.WorkoutSessionID = req.WorkoutSessionID
	}
	if req.WorkoutProgramID != nil {
		existing.WorkoutProgramID = req.WorkoutProgramID
	}
	if req.ScheduledDate != nil {
		existing.ScheduledDate = req.ScheduledDate
	}
	if req.CompletedDate != nil {
		existing.CompletedDate = req.CompletedDate
	}
	if req.DurationMinutes != nil {
		existing.DurationMinutes = req.DurationMinutes
	}
	if req.CaloriesBurned != nil {
		existing.CaloriesBurned = req.CaloriesBurned
	}
	if req.PerceivedExertion != nil {
		existing.PerceivedExertion = req.PerceivedExertion
	}
	if req.MoodBefore != nil {
		existing.MoodBefore = req.MoodBefore
	}
	if req.MoodAfter != nil {
		existing.MoodAfter = req.MoodAfter
	}
	if req.ExercisesCompleted != nil {
		existing.ExercisesCompleted = req.ExercisesCompleted
	}
	if req.ExercisesSkipped != nil {
		existing.ExercisesSkipped = req.ExercisesSkipped
	}
	if req.ModificationsUsed != nil {
		existing.ModificationsUsed = req.ModificationsUsed
	}
	if req.Notes != nil {
		existing.Notes = req.Notes
	}
	if req.InjuriesReported != nil {
		existing.InjuriesReported = req.InjuriesReported
	}
	if req.Status != "" {
		existing.Status = req.Status
	}

	if err := h.workoutRepo.UpdateUserWorkoutSession(existing); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update workout: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, existing)
}

// DeleteWorkout deletes a workout session
func (h *WorkoutHandler) DeleteWorkout(c echo.Context) error {
	id := c.Param("id")

	// Get user ID from context (from JWT middleware)
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Convert userID to string
	var userIDStr string
	switch v := userID.(type) {
	case string:
		userIDStr = v
	case int64:
		userIDStr = strconv.FormatInt(v, 10)
	case int:
		userIDStr = strconv.Itoa(v)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	// Check if session exists and belongs to user
	_, err := h.workoutRepo.GetUserWorkoutSessionByID(id, userIDStr)
	if err != nil {
		if err.Error() == "workout session not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Workout session not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get workout: " + err.Error(),
		})
	}

	if err := h.workoutRepo.DeleteUserWorkoutSession(id, userIDStr); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete workout: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Workout deleted successfully",
	})
}
