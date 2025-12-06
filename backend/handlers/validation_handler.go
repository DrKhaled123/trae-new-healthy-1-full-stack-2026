package handlers

import (
	"net/http"
	"nutrition-platform/services"

	"github.com/labstack/echo/v4"
)

// ValidationHandler handles data validation requests
type ValidationHandler struct {
	validator *services.NutritionDataValidator
}

// NewValidationHandler creates a new validation handler
func NewValidationHandler(dataDir string) *ValidationHandler {
	return &ValidationHandler{
		validator: services.NewNutritionDataValidator(dataDir),
	}
}

// ValidateAll validates all nutrition data files
func (h *ValidationHandler) ValidateAll(c echo.Context) error {
	results, err := h.validator.ValidateAll()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "Validation failed",
			"message": err.Error(),
		})
	}

	// Count valid/invalid files
	validCount := 0
	invalidCount := 0
	for _, result := range results {
		if result.Valid {
			validCount++
		} else {
			invalidCount++
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":        "success",
		"valid_count":   validCount,
		"invalid_count": invalidCount,
		"total_files":   len(results),
		"results":       results,
	})
}

// ValidateFile validates a specific file
func (h *ValidationHandler) ValidateFile(c echo.Context) error {
	filename := c.Param("filename")
	if filename == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Filename is required",
		})
	}

	result := h.validator.ValidateFile(filename)

	statusCode := http.StatusOK
	if !result.Valid {
		statusCode = http.StatusBadRequest
	}

	return c.JSON(statusCode, map[string]interface{}{
		"status": "success",
		"result": result,
	})
}
