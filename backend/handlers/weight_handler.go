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

// WeightHandler handles weight tracking operations
type WeightHandler struct {
	weightRepo *repositories.WeightRepository
}

// NewWeightHandler creates a new weight handler
func NewWeightHandler(db *sql.DB) *WeightHandler {
	// Wrap *sql.DB in *database.Database for WeightRepository
	dbWrapper := database.NewDatabase(db)
	return &WeightHandler{
		weightRepo: repositories.NewWeightRepository(dbWrapper),
	}
}

// LogWeight logs a weight entry
func (h *WeightHandler) LogWeight(c echo.Context) error {
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
		Weight float64 `json:"weight" validate:"required,min=0"`
		Unit   string  `json:"unit"` // kg or lbs
		Notes  *string `json:"notes"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Default unit to kg if not provided
	if req.Unit == "" {
		req.Unit = "kg"
	}

	log := &models.WeightLog{
		UserID:    int(userIDUint),
		Weight:    req.Weight,
		Unit:      req.Unit,
		Notes:     req.Notes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := h.weightRepo.CreateWeightLog(log)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to log weight: " + err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": "Weight logged successfully",
		"data":    log,
	})
}

// GetWeightHistory returns weight history for current user
func (h *WeightHandler) GetWeightHistory(c echo.Context) error {
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

	// Parse pagination
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

	// Parse date filters (currently unused but keeping for future implementation)
	// var startDate, endDate *time.Time
	// if startDateStr := c.QueryParam("start_date"); startDateStr != "" {
	// 	if sd, err := time.Parse("2006-01-02", startDateStr); err == nil {
	// 		startDate = &sd
	// 	}
	// }
	// if endDateStr := c.QueryParam("end_date"); endDateStr != "" {
	// 	if ed, err := time.Parse("2006-01-02", endDateStr); err == nil {
	// 		endDate = &ed
	// 	}
	// }

	logs, total, err := h.weightRepo.GetWeightLogs(int(userIDUint), page, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch weight history: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   logs,
		"pagination": map[string]interface{}{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetWeightLog returns a specific weight log by ID
func (h *WeightHandler) GetWeightLog(c echo.Context) error {
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

	logID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid weight log ID",
		})
	}

	log, err := h.weightRepo.GetWeightLogByID(int(uint(logID)), int(userIDUint))
	if err != nil {
		if err.Error() == "weight log not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Weight log not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch weight log: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   log,
	})
}

// UpdateWeightLog updates a weight log entry
func (h *WeightHandler) UpdateWeightLog(c echo.Context) error {
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

	logID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid weight log ID",
		})
	}

	var req struct {
		Weight float64 `json:"weight" validate:"required,min=0"`
		Unit   string  `json:"unit"` // kg or lbs
		Notes  *string `json:"notes"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Get existing log
	existingLog, err := h.weightRepo.GetWeightLogByID(int(uint(logID)), int(userIDUint))
	if err != nil {
		if err.Error() == "weight log not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Weight log not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch weight log",
		})
	}

	// Update fields
	existingLog.Weight = req.Weight
	if req.Unit != "" {
		existingLog.Unit = req.Unit
	}
	if req.Notes != nil {
		existingLog.Notes = req.Notes
	}
	existingLog.UpdatedAt = time.Now()
	existingLog.UpdatedAt = time.Now()

	err = h.weightRepo.UpdateWeightLog(existingLog)
	if err != nil {
		if err.Error() == "weight log not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Weight log not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update weight log: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Weight log updated successfully",
		"data":    existingLog,
	})
}

// DeleteWeightLog deletes a weight log entry
func (h *WeightHandler) DeleteWeightLog(c echo.Context) error {
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

	logID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid weight log ID",
		})
	}

	err = h.weightRepo.DeleteWeightLog(int(uint(logID)), int(userIDUint))
	if err != nil {
		if err.Error() == "weight log not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Weight log not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete weight log: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Weight log deleted successfully",
	})
}
