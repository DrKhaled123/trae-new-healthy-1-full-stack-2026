package utils

import (
	"strconv"

	"github.com/labstack/echo/v4"
)

// PaginationParams represents parsed pagination parameters
type PaginationParams struct {
	Page  int
	Limit int
}

// PaginationMeta represents pagination metadata (legacy)
type PaginationMeta struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// ParsePagination extracts and validates pagination params from request
func ParsePagination(c echo.Context) (PaginationParams, error) {
	page := DefaultPage
	limit := DefaultLimit

	// Parse page
	if pageStr := c.QueryParam("page"); pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p < 1 {
			return PaginationParams{}, echo.NewHTTPError(400, "Invalid page parameter: must be positive integer")
		}
		page = p
	}

	// Parse limit
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil || l < 1 || l > MaxLimit {
			return PaginationParams{}, echo.NewHTTPError(400, "Invalid limit parameter: must be between 1 and 100")
		}
		limit = l
	}

	return PaginationParams{Page: page, Limit: limit}, nil
}

// CalculateOffset calculates database offset from page and limit
func CalculateOffset(page, limit int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * limit
}

// CalculatePagination calculates pagination metadata (legacy)
func CalculatePagination(page, limit, total int) PaginationMeta {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	totalPages := (total + limit - 1) / limit // Ceiling division
	if totalPages == 0 {
		totalPages = 1
	}

	return PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// GetOffset calculates the offset for SQL queries (legacy)
func GetOffset(page, limit int) int {
	return CalculateOffset(page, limit)
}

// ValidatePaginationParams validates and normalizes pagination parameters (legacy)
func ValidatePaginationParams(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
