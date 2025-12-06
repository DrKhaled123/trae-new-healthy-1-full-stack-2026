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

// ExerciseHandler handles exercise-related requests
type ExerciseHandler struct {
	exerciseRepo *repositories.ExerciseRepository
}

// NewExerciseHandler creates a new ExerciseHandler instance
func NewExerciseHandler(db *sql.DB) *ExerciseHandler {
	dbWrapper := database.NewDatabase(db)
	return &ExerciseHandler{
		exerciseRepo: repositories.NewExerciseRepository(dbWrapper),
	}
}

// GetExercises retrieves exercises with pagination and filters
func (h *ExerciseHandler) GetExercises(c echo.Context) error {
	// Get user ID from context (from JWT middleware)
	userID := c.Get("user_id")

	// Parse query parameters
	req := &models.ListExercisesRequest{
		Search:      c.QueryParam("search"),
		MuscleGroup: c.QueryParam("muscle_group"),
		Equipment:   c.QueryParam("equipment"),
		Difficulty:  c.QueryParam("difficulty"),
		SortBy:      c.QueryParam("sort_by"),
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

	req.Limit = limit
	req.Offset = (page - 1) * limit

	// If user is not authenticated, only show public exercises
	if userID == nil {
		isPublic := true
		req.IsPublic = &isPublic
	} else {
		// Convert userID to int64 if present
		var userIDInt64 int64
		switch v := userID.(type) {
		case int64:
			userIDInt64 = v
		case string:
			if id, err := strconv.ParseInt(v, 10, 64); err == nil {
				userIDInt64 = id
			}
		case int:
			userIDInt64 = int64(v)
		}

		if userIDInt64 > 0 {
			req.CreatedBy = &userIDInt64
		}
	}

	exercises, total, err := h.exerciseRepo.GetExercises(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get exercises: " + err.Error(),
		})
	}

	totalPages := (int(total) + limit - 1) / limit
	response := &models.ExerciseListResponse{
		Exercises:  exercises,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	return c.JSON(http.StatusOK, response)
}

// SearchExercises searches exercises by name or description
func (h *ExerciseHandler) SearchExercises(c echo.Context) error {
	search := c.QueryParam("q")
	if search == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Search query is required",
		})
	}

	limit := 20
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	exercises, err := h.exerciseRepo.SearchExercises(search, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to search exercises: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"exercises": exercises,
		"count":     len(exercises),
	})
}

// GetExercise retrieves a specific exercise by ID
func (h *ExerciseHandler) GetExercise(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid exercise ID",
		})
	}

	exercise, err := h.exerciseRepo.GetExerciseByID(id)
	if err != nil {
		if err.Error() == "exercise not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Exercise not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get exercise: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, exercise)
}

// CreateExercise creates a new exercise
func (h *ExerciseHandler) CreateExercise(c echo.Context) error {
	// Get user ID from context (from JWT middleware)
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Convert userID to int64
	var userIDInt64 int64
	switch v := userID.(type) {
	case int64:
		userIDInt64 = v
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			userIDInt64 = id
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid user ID",
			})
		}
	case int:
		userIDInt64 = int64(v)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	var req struct {
		Name         string              `json:"name" validate:"required,min=1,max=200"`
		Description  string              `json:"description" validate:"max=1000"`
		MuscleGroups models.MuscleGroups `json:"muscle_groups"`
		Equipment    models.Equipment    `json:"equipment"`
		Difficulty   string              `json:"difficulty" validate:"oneof=beginner intermediate advanced"`
		Instructions string              `json:"instructions" validate:"max=2000"`
		Tips         string              `json:"tips" validate:"max=1000"`
		IsPublic     bool                `json:"is_public"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	// Validate required fields
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name is required",
		})
	}

	// Set default values
	if req.Difficulty == "" {
		req.Difficulty = "beginner"
	}

	exercise := &models.Exercise{
		Name:         req.Name,
		Description:  req.Description,
		MuscleGroups: req.MuscleGroups,
		Equipment:    req.Equipment,
		Difficulty:   req.Difficulty,
		Instructions: req.Instructions,
		Tips:         req.Tips,
		CreatedBy:    userIDInt64,
		IsPublic:     req.IsPublic,
	}

	if err := h.exerciseRepo.CreateExercise(exercise); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create exercise: " + err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, exercise)
}

// UpdateExercise updates an existing exercise
func (h *ExerciseHandler) UpdateExercise(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid exercise ID",
		})
	}

	// Get user ID from context (from JWT middleware)
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Convert userID to int64
	var userIDInt64 int64
	switch v := userID.(type) {
	case int64:
		userIDInt64 = v
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			userIDInt64 = id
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid user ID",
			})
		}
	case int:
		userIDInt64 = int64(v)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	// Get existing exercise
	existing, err := h.exerciseRepo.GetExerciseByID(id)
	if err != nil {
		if err.Error() == "exercise not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Exercise not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get exercise: " + err.Error(),
		})
	}

	// Check if user owns the exercise
	if existing.CreatedBy != userIDInt64 {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "You can only update your own exercises",
		})
	}

	var req struct {
		Name         *string              `json:"name,omitempty"`
		Description  *string              `json:"description,omitempty"`
		MuscleGroups *models.MuscleGroups `json:"muscle_groups,omitempty"`
		Equipment    *models.Equipment    `json:"equipment,omitempty"`
		Difficulty   *string              `json:"difficulty,omitempty"`
		Instructions *string              `json:"instructions,omitempty"`
		Tips         *string              `json:"tips,omitempty"`
		IsPublic     *bool                `json:"is_public,omitempty"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	// Update fields if provided
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.MuscleGroups != nil {
		existing.MuscleGroups = *req.MuscleGroups
	}
	if req.Equipment != nil {
		existing.Equipment = *req.Equipment
	}
	if req.Difficulty != nil {
		existing.Difficulty = *req.Difficulty
	}
	if req.Instructions != nil {
		existing.Instructions = *req.Instructions
	}
	if req.Tips != nil {
		existing.Tips = *req.Tips
	}
	if req.IsPublic != nil {
		existing.IsPublic = *req.IsPublic
	}

	if err := h.exerciseRepo.UpdateExercise(existing); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update exercise: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, existing)
}

// DeleteExercise deletes an exercise
func (h *ExerciseHandler) DeleteExercise(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid exercise ID",
		})
	}

	// Get user ID from context (from JWT middleware)
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Convert userID to int64
	var userIDInt64 int64
	switch v := userID.(type) {
	case int64:
		userIDInt64 = v
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			userIDInt64 = id
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid user ID",
			})
		}
	case int:
		userIDInt64 = int64(v)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	// Get existing exercise
	existing, err := h.exerciseRepo.GetExerciseByID(id)
	if err != nil {
		if err.Error() == "exercise not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Exercise not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get exercise: " + err.Error(),
		})
	}

	// Check if user owns the exercise
	if existing.CreatedBy != userIDInt64 {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "You can only delete your own exercises",
		})
	}

	if err := h.exerciseRepo.DeleteExercise(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete exercise: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Exercise deleted successfully",
	})
}
