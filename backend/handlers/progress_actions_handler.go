package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"nutrition-platform/models"
	"nutrition-platform/services"

	"github.com/labstack/echo/v4"
)

// ProgressActionsHandler handles user-facing progress tracking actions
type ProgressActionsHandler struct {
	progressService *services.ProgressService
}

func NewProgressActionsHandler(db *sql.DB) *ProgressActionsHandler {
	return &ProgressActionsHandler{
		progressService: services.NewProgressService(db),
	}
}

// TrackMeasurement - Action: User clicks "Log Measurement" button
// POST /api/v1/actions/track-measurement
func (h *ProgressActionsHandler) TrackMeasurement(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Convert userID to int64
	var userIDInt int64
	switch v := userID.(type) {
	case uint:
		userIDInt = int64(v)
	case int:
		userIDInt = int64(v)
	case int64:
		userIDInt = v
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			userIDInt = id
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
			"error": "Invalid request format: " + err.Error(),
		})
	}

	// Use service layer (business logic)
	measurement := &models.BodyMeasurement{
		MeasurementDate:   time.Now(),
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
	}

	if req.MeasurementDate != nil {
		measurement.MeasurementDate = *req.MeasurementDate
	}

	result, err := h.progressService.LogMeasurement(c.Request().Context(), uint(userIDInt), measurement)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": "Measurement logged successfully",
		"data":    result,
	})
}

// GetProgressSummary - Action: User clicks "View Progress" button
// GET /api/v1/actions/progress-summary?days=30
func (h *ProgressActionsHandler) GetProgressSummary(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Convert userID to int64
	var userIDInt int64
	switch v := userID.(type) {
	case uint:
		userIDInt = int64(v)
	case int:
		userIDInt = int64(v)
	case int64:
		userIDInt = v
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			userIDInt = id
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

	// Parse days parameter (default 30)
	days := 30
	if daysStr := c.QueryParam("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	summary, err := h.progressService.GetProgressSummary(c.Request().Context(), uint(userIDInt), days)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get progress summary: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   summary,
	})
}

// GetMeasurementHistory - Action: User views measurement history
// GET /api/v1/actions/measurement-history?page=1&limit=20&start_date=2024-01-01&end_date=2024-12-31
func (h *ProgressActionsHandler) GetMeasurementHistory(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Convert userID to int64
	var userIDInt int64
	switch v := userID.(type) {
	case uint:
		userIDInt = int64(v)
	case int:
		userIDInt = int64(v)
	case int64:
		userIDInt = v
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			userIDInt = id
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

	// Parse date filters
	var startDate, endDate *time.Time
	if startDateStr := c.QueryParam("start_date"); startDateStr != "" {
		if sd, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = &sd
		}
	}
	if endDateStr := c.QueryParam("end_date"); endDateStr != "" {
		if ed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = &ed
		}
	}

	measurements, total, err := h.progressService.GetMeasurementHistory(c.Request().Context(), uint(userIDInt), page, limit, startDate, endDate)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get measurement history: " + err.Error(),
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

// GetProgressCharts - Action: User views progress charts
// GET /api/v1/actions/progress-charts?days=30
func (h *ProgressActionsHandler) GetProgressCharts(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Convert userID to int64
	var userIDInt int64
	switch v := userID.(type) {
	case uint:
		userIDInt = int64(v)
	case int:
		userIDInt = int64(v)
	case int64:
		userIDInt = v
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			userIDInt = id
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

	// Parse days parameter (default 30)
	days := 30
	if daysStr := c.QueryParam("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	charts, err := h.progressService.GetProgressCharts(c.Request().Context(), uint(userIDInt), days)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get progress charts: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   charts,
	})
}

// CompareMeasurements - Action: User compares measurements between dates
// POST /api/v1/actions/compare-measurements
func (h *ProgressActionsHandler) CompareMeasurements(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Convert userID to int64
	var userIDInt int64
	switch v := userID.(type) {
	case uint:
		userIDInt = int64(v)
	case int:
		userIDInt = int64(v)
	case int64:
		userIDInt = v
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			userIDInt = id
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
		StartDate string `json:"start_date" validate:"required"`
		EndDate   string `json:"end_date" validate:"required"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid start_date format. Use YYYY-MM-DD",
		})
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid end_date format. Use YYYY-MM-DD",
		})
	}

	comparison, err := h.progressService.CompareMeasurements(c.Request().Context(), uint(userIDInt), startDate, endDate)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to compare measurements: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   comparison,
	})
}

// UploadProgressPhoto - Action: User clicks "Upload Progress Photo" button
// POST /api/v1/actions/upload-progress-photo
func (h *ProgressActionsHandler) UploadProgressPhoto(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Convert userID to int64
	var userIDInt int64
	switch v := userID.(type) {
	case uint:
		userIDInt = int64(v)
	case int:
		userIDInt = int64(v)
	case int64:
		userIDInt = v
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			userIDInt = id
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
		PhotoURL     string    `json:"photo_url" validate:"required"`
		ThumbnailURL string    `json:"thumbnail_url"`
		Date         time.Time `json:"date"`
		Weight       *float64  `json:"weight"`
		Notes        *string   `json:"notes"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	// Use current date if not provided
	if req.Date.IsZero() {
		req.Date = time.Now()
	}

	photo := map[string]interface{}{
		"user_id":       userIDInt,
		"photo_url":     req.PhotoURL,
		"thumbnail_url": req.ThumbnailURL,
		"date":          req.Date.Format("2006-01-02"),
		"weight":        req.Weight,
		"notes":         req.Notes,
		"uploaded_at":   time.Now().Format(time.RFC3339),
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": "Progress photo uploaded successfully",
		"data":    photo,
	})
}

// GetPhotoHistory - Action: User views photo gallery
// GET /api/v1/actions/photo-history?page=1&limit=20
func (h *ProgressActionsHandler) GetPhotoHistory(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Convert userID to int64
	var userIDInt int64
	switch v := userID.(type) {
	case uint:
		userIDInt = int64(v)
	case int:
		userIDInt = int64(v)
	case int64:
		userIDInt = v
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			userIDInt = id
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

	// Use the service to get photo history
	photos, total, err := h.progressService.GetPhotoHistory(c.Request().Context(), uint(userIDInt), page, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch photo history: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   photos,
		"pagination": map[string]interface{}{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}
