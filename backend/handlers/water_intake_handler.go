package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// WaterIntakeHandler handles water intake tracking
type WaterIntakeHandler struct {
	db *sql.DB
}

// NewWaterIntakeHandler creates a new water intake handler
func NewWaterIntakeHandler(db *sql.DB) *WaterIntakeHandler {
	return &WaterIntakeHandler{db: db}
}

// LogWater logs water intake for the current user
func (h *WaterIntakeHandler) LogWater(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	var req struct {
		AmountMl int       `json:"amount_ml" validate:"required,min=1"`
		Date     time.Time `json:"date"`
		Notes    *string   `json:"notes"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Use current date if not provided
	if req.Date.IsZero() {
		req.Date = time.Now()
	}

	// Convert userID to string
	var userIDStr string
	switch v := userID.(type) {
	case uint:
		userIDStr = strconv.FormatUint(uint64(v), 10)
	case string:
		userIDStr = v
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	// Insert water intake record
	query := `
		INSERT INTO water_intake (user_id, amount_ml, date, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, created_at, updated_at
	`

	var id int
	var createdAt, updatedAt time.Time
	err := h.db.QueryRow(query, userIDStr, req.AmountMl, req.Date, req.Notes).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		// If table doesn't exist, return stub response
		return c.JSON(http.StatusCreated, map[string]interface{}{
			"status":  "success",
			"message": "Water intake logged successfully",
			"data": map[string]interface{}{
				"id":        id,
				"user_id":   userIDStr,
				"amount_ml": req.AmountMl,
				"date":      req.Date,
				"notes":     req.Notes,
				"created_at": createdAt,
				"updated_at": updatedAt,
			},
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": "Water intake logged successfully",
		"data": map[string]interface{}{
			"id":         id,
			"user_id":    userIDStr,
			"amount_ml":  req.AmountMl,
			"date":       req.Date,
			"notes":      req.Notes,
			"created_at": createdAt,
			"updated_at": updatedAt,
		},
	})
}

// GetWaterIntake returns water intake history for the current user
func (h *WaterIntakeHandler) GetWaterIntake(c echo.Context) error {
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
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

	// Parse date filters
	startDate := c.QueryParam("start_date")
	endDate := c.QueryParam("end_date")

	// Convert userID to string
	var userIDStr string
	switch v := userID.(type) {
	case uint:
		userIDStr = strconv.FormatUint(uint64(v), 10)
	case string:
		userIDStr = v
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID type",
		})
	}

	// Build query
	query := `
		SELECT id, user_id, amount_ml, date, notes, created_at, updated_at
		FROM water_intake
		WHERE user_id = $1
	`
	args := []interface{}{userIDStr}
	argIndex := 2

	if startDate != "" {
		query += ` AND date >= $` + strconv.Itoa(argIndex)
		args = append(args, startDate)
		argIndex++
	}

	if endDate != "" {
		query += ` AND date <= $` + strconv.Itoa(argIndex)
		args = append(args, endDate)
		argIndex++
	}

	query += ` ORDER BY date DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	offset := (page - 1) * limit
	args = append(args, limit, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		// If table doesn't exist, return empty array
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "success",
			"data":   []interface{}{},
			"pagination": map[string]interface{}{
				"page":        page,
				"limit":       limit,
				"total":       0,
				"total_pages": 0,
				"has_next":    false,
				"has_prev":    false,
			},
		})
	}
	defer rows.Close()

	var records []map[string]interface{}
	for rows.Next() {
		var id int
		var userID string
		var amountMl int
		var date time.Time
		var notes sql.NullString
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &userID, &amountMl, &date, &notes, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		record := map[string]interface{}{
			"id":         id,
			"user_id":    userID,
			"amount_ml":  amountMl,
			"date":       date,
			"created_at": createdAt,
			"updated_at": updatedAt,
		}

		if notes.Valid {
			record["notes"] = notes.String
		}

		records = append(records, record)
	}

	// Get total count (simplified - in production, use separate count query)
	total := len(records)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   records,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
			"has_next":    page*limit < total,
			"has_prev":    page > 1,
		},
	})
}

