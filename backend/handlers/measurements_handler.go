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

// MeasurementsHandler handles body measurement operations
type MeasurementsHandler struct {
	measurementRepo *repositories.BodyMeasurementRepository
}

// NewMeasurementsHandler creates a new measurements handler
func NewMeasurementsHandler(db *sql.DB) *MeasurementsHandler {
	// Wrap *sql.DB in *database.Database for BodyMeasurementRepository
	dbWrapper := database.NewDatabase(db)
	return &MeasurementsHandler{
		measurementRepo: repositories.NewBodyMeasurementRepository(dbWrapper),
	}
}

// LogMeasurement logs a body measurement entry
func (h *MeasurementsHandler) LogMeasurement(c echo.Context) error {
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
		MeasurementDate   *time.Time `json:"measurement_date"`
		Weight            *float64   `json:"weight,omitempty"`
		Height            *float64   `json:"height,omitempty"`
		BodyFatPercentage *float64   `json:"body_fat_percentage,omitempty"`
		MuscleMass        *float64   `json:"muscle_mass,omitempty"`
		Waist             *float64   `json:"waist,omitempty"`
		Chest             *float64   `json:"chest,omitempty"`
		LeftBicep         *float64   `json:"left_bicep,omitempty"`
		RightBicep        *float64   `json:"right_bicep,omitempty"`
		LeftForearm       *float64   `json:"left_forearm,omitempty"`
		RightForearm      *float64   `json:"right_forearm,omitempty"`
		LeftThigh         *float64   `json:"left_thigh,omitempty"`
		RightThigh        *float64   `json:"right_thigh,omitempty"`
		LeftCalf          *float64   `json:"left_calf,omitempty"`
		RightCalf         *float64   `json:"right_calf,omitempty"`
		Neck              *float64   `json:"neck,omitempty"`
		Hips              *float64   `json:"hips,omitempty"`
		Notes             *string    `json:"notes,omitempty"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Default measurement date to today if not provided
	measurementDate := time.Now()
	if req.MeasurementDate != nil {
		measurementDate = *req.MeasurementDate
	}

	measurement := &models.BodyMeasurement{
		UserID:            userIDUint,
		MeasurementDate:   measurementDate,
		Weight:            req.Weight,
		Height:            req.Height,
		BodyFatPercentage: req.BodyFatPercentage,
		MuscleMass:        req.MuscleMass,
		Waist:             req.Waist,
		Chest:             req.Chest,
		LeftBicep:         req.LeftBicep,
		RightBicep:        req.RightBicep,
		LeftForearm:       req.LeftForearm,
		RightForearm:      req.RightForearm,
		LeftThigh:         req.LeftThigh,
		RightThigh:        req.RightThigh,
		LeftCalf:          req.LeftCalf,
		RightCalf:         req.RightCalf,
		Neck:              req.Neck,
		Hips:              req.Hips,
		Notes:             req.Notes,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Validate the measurement
	if err := measurement.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	err := h.measurementRepo.CreateBodyMeasurement(c.Request().Context(), measurement)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to log measurement: " + err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": "Body measurement logged successfully",
		"data":    measurement,
	})
}

// GetMeasurements returns measurement history for current user
func (h *MeasurementsHandler) GetMeasurements(c echo.Context) error {
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

	offset := (page - 1) * limit

	// Get measurements
	measurements, err := h.measurementRepo.GetBodyMeasurementsByUserID(c.Request().Context(), int64(userIDUint), limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch measurements: " + err.Error(),
		})
	}

	// Get total count for pagination
	total, err := h.measurementRepo.GetMeasurementCountByUserID(c.Request().Context(), int64(userIDUint))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch measurement count: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   measurements,
		"pagination": map[string]interface{}{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetMeasurement returns a specific measurement by ID
func (h *MeasurementsHandler) GetMeasurement(c echo.Context) error {
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

	measurementID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid measurement ID",
		})
	}

	measurement, err := h.measurementRepo.GetBodyMeasurementByID(c.Request().Context(), int64(measurementID))
	if err != nil {
		if err.Error() == "body measurement not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Body measurement not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch measurement: " + err.Error(),
		})
	}

	// Check if the measurement belongs to the current user
	if measurement.UserID != userIDUint {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   measurement,
	})
}

// UpdateMeasurement updates a body measurement entry
func (h *MeasurementsHandler) UpdateMeasurement(c echo.Context) error {
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

	measurementID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid measurement ID",
		})
	}

	var req struct {
		MeasurementDate   *time.Time `json:"measurement_date"`
		Weight            *float64   `json:"weight,omitempty"`
		Height            *float64   `json:"height,omitempty"`
		BodyFatPercentage *float64   `json:"body_fat_percentage,omitempty"`
		MuscleMass        *float64   `json:"muscle_mass,omitempty"`
		Waist             *float64   `json:"waist,omitempty"`
		Chest             *float64   `json:"chest,omitempty"`
		LeftBicep         *float64   `json:"left_bicep,omitempty"`
		RightBicep        *float64   `json:"right_bicep,omitempty"`
		LeftForearm       *float64   `json:"left_forearm,omitempty"`
		RightForearm      *float64   `json:"right_forearm,omitempty"`
		LeftThigh         *float64   `json:"left_thigh,omitempty"`
		RightThigh        *float64   `json:"right_thigh,omitempty"`
		LeftCalf          *float64   `json:"left_calf,omitempty"`
		RightCalf         *float64   `json:"right_calf,omitempty"`
		Neck              *float64   `json:"neck,omitempty"`
		Hips              *float64   `json:"hips,omitempty"`
		Notes             *string    `json:"notes,omitempty"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Get existing measurement
	existingMeasurement, err := h.measurementRepo.GetBodyMeasurementByID(c.Request().Context(), int64(measurementID))
	if err != nil {
		if err.Error() == "body measurement not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Body measurement not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch measurement",
		})
	}

	// Check if the measurement belongs to the current user
	if existingMeasurement.UserID != userIDUint {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	// Update fields if provided
	if req.MeasurementDate != nil {
		existingMeasurement.MeasurementDate = *req.MeasurementDate
	}
	if req.Weight != nil {
		existingMeasurement.Weight = req.Weight
	}
	if req.Height != nil {
		existingMeasurement.Height = req.Height
	}
	if req.BodyFatPercentage != nil {
		existingMeasurement.BodyFatPercentage = req.BodyFatPercentage
	}
	if req.MuscleMass != nil {
		existingMeasurement.MuscleMass = req.MuscleMass
	}
	if req.Waist != nil {
		existingMeasurement.Waist = req.Waist
	}
	if req.Chest != nil {
		existingMeasurement.Chest = req.Chest
	}
	if req.LeftBicep != nil {
		existingMeasurement.LeftBicep = req.LeftBicep
	}
	if req.RightBicep != nil {
		existingMeasurement.RightBicep = req.RightBicep
	}
	if req.LeftForearm != nil {
		existingMeasurement.LeftForearm = req.LeftForearm
	}
	if req.RightForearm != nil {
		existingMeasurement.RightForearm = req.RightForearm
	}
	if req.LeftThigh != nil {
		existingMeasurement.LeftThigh = req.LeftThigh
	}
	if req.RightThigh != nil {
		existingMeasurement.RightThigh = req.RightThigh
	}
	if req.LeftCalf != nil {
		existingMeasurement.LeftCalf = req.LeftCalf
	}
	if req.RightCalf != nil {
		existingMeasurement.RightCalf = req.RightCalf
	}
	if req.Neck != nil {
		existingMeasurement.Neck = req.Neck
	}
	if req.Hips != nil {
		existingMeasurement.Hips = req.Hips
	}
	if req.Notes != nil {
		existingMeasurement.Notes = req.Notes
	}
	existingMeasurement.UpdatedAt = time.Now()

	// Validate the updated measurement
	if err := existingMeasurement.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	err = h.measurementRepo.UpdateBodyMeasurement(c.Request().Context(), existingMeasurement)
	if err != nil {
		if err.Error() == "body measurement not found or access denied" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Body measurement not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update measurement: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Body measurement updated successfully",
		"data":    existingMeasurement,
	})
}

// DeleteMeasurement deletes a body measurement entry
func (h *MeasurementsHandler) DeleteMeasurement(c echo.Context) error {
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

	measurementID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid measurement ID",
		})
	}

	// First check if the measurement exists and belongs to the user
	existingMeasurement, err := h.measurementRepo.GetBodyMeasurementByID(c.Request().Context(), int64(measurementID))
	if err != nil {
		if err.Error() == "body measurement not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Body measurement not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch measurement",
		})
	}

	// Check if the measurement belongs to the current user
	if existingMeasurement.UserID != userIDUint {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	err = h.measurementRepo.DeleteBodyMeasurement(c.Request().Context(), int64(measurementID), int64(userIDUint))
	if err != nil {
		if err.Error() == "body measurement not found or access denied" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Body measurement not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete measurement: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Body measurement deleted successfully",
	})
}
