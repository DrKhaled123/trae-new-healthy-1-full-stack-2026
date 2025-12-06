package utils

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Pagination represents the new standard pagination format
type Pagination struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// StandardResponse represents the standardized API response
type StandardResponseNew struct {
	Status     string      `json:"status"`
	Data       interface{} `json:"data,omitempty"`
	Items      interface{} `json:"items,omitempty"`  // Frontend expects this
	Pagination *Pagination `json:"pagination,omitempty"`
	Filters    interface{} `json:"filters,omitempty"`
	Error      string      `json:"error,omitempty"`
	Message    string      `json:"message,omitempty"`
	Meta       interface{} `json:"meta,omitempty"`
}

// Core response functions
func Success(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, StandardResponseNew{
		Status: "success",
		Data:   data,
		Items:  data,  // For consistency with list endpoints
		Meta:   getDefaultMeta(c),
	})
}

func SuccessListWithPagination(c echo.Context, data interface{}, pagination *Pagination) error {
	return c.JSON(http.StatusOK, StandardResponseNew{
		Status:     "success",
		Data:       data,  // Backward compatibility
		Items:      data,  // Frontend expects this
		Pagination: pagination,
		Meta:       getDefaultMeta(c),
	})
}

func Error(c echo.Context, statusCode int, message string) error {
	return c.JSON(statusCode, StandardResponseNew{
		Status: "error",
		Error:  message,
		Meta:   getDefaultMeta(c),
	})
}

func BadRequest(c echo.Context, message string) error {
	return Error(c, http.StatusBadRequest, message)
}

func NotFound(c echo.Context, message string) error {
	return Error(c, http.StatusNotFound, message)
}

func InternalError(c echo.Context, message string) error {
	return Error(c, http.StatusInternalServerError, message)
}

func getDefaultMeta(c echo.Context) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": c.Get("timestamp"),
		"request_id": c.Response().Header().Get("X-Request-ID"),
	}
}

// StandardResponse represents legacy response format (for backward compatibility)
type StandardResponse struct {
	Status     string      `json:"status"`
	Data       interface{} `json:"data,omitempty"`
	Pagination interface{} `json:"pagination,omitempty"`
	Filters    interface{} `json:"filters,omitempty"`
	Error      string      `json:"error,omitempty"`
	Message    string      `json:"message,omitempty"`
}

// Legacy response functions (for backward compatibility)
func SuccessResponse(c echo.Context, data interface{}) error {
	return Success(c, data)
}

func SuccessResponseWithPagination(c echo.Context, data interface{}, pagination PaginationMeta, filters interface{}) error {
	// Convert legacy PaginationMeta to new Pagination
	newPagination := &Pagination{
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		Total:      pagination.Total,
		TotalPages: pagination.TotalPages,
	}
	return SuccessListWithPagination(c, data, newPagination)
}

func ErrorResponse(c echo.Context, statusCode int, message string) error {
	return Error(c, statusCode, message)
}

func BadRequestResponse(c echo.Context, message string) error {
	return BadRequest(c, message)
}

func NotFoundResponse(c echo.Context, message string) error {
	return NotFound(c, message)
}

func InternalServerErrorResponse(c echo.Context, message string) error {
	return InternalError(c, message)
}

func UnsupportedMediaType(c echo.Context, message string) error {
	return Error(c, http.StatusUnsupportedMediaType, message)
}
