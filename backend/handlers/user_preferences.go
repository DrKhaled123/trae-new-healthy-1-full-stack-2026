package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// UserPreferencesHandler handles user preferences endpoints
type UserPreferencesHandler struct {
	// Add dependencies here (e.g., userService, db)
}

// NewUserPreferencesHandler creates a new user preferences handler
func NewUserPreferencesHandler() *UserPreferencesHandler {
	return &UserPreferencesHandler{}
}

// GetPreferences returns the current user's preferences
func (h *UserPreferencesHandler) GetPreferences(c echo.Context) error {
	// Get user ID from context (set by JWT middleware)
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Stub implementation - in production, fetch from database
	preferences := map[string]interface{}{
		"user_id":                userID,
		"language":               "en",
		"timezone":               "UTC",
		"notifications_enabled":  true,
		"email_notifications":    true,
		"push_notifications":     true,
		"units":                  "metric",
		"dark_mode":              false,
		"theme":                  "light",
		"meal_reminders":         true,
		"water_reminders":        true,
		"workout_reminders":      true,
	}

	return c.JSON(http.StatusOK, preferences)
}

// UpdatePreferences updates the current user's preferences
func (h *UserPreferencesHandler) UpdatePreferences(c echo.Context) error {
	// Get user ID from context
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	var req map[string]interface{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Stub implementation - in production:
	// 1. Validate preferences
	// 2. Update database
	// 3. Return updated preferences

	// Merge with defaults
	preferences := map[string]interface{}{
		"user_id":                userID,
		"language":               "en",
		"timezone":               "UTC",
		"notifications_enabled":  true,
		"email_notifications":    true,
		"push_notifications":     true,
		"units":                  "metric",
		"dark_mode":              false,
		"theme":                  "light",
		"meal_reminders":         true,
		"water_reminders":        true,
		"workout_reminders":      true,
	}

	// Update with request values
	for k, v := range req {
		preferences[k] = v
	}

	return c.JSON(http.StatusOK, preferences)
}

