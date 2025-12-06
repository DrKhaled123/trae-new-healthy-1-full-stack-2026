package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// ValidateQueryParams validates common query parameters
func ValidateQueryParams(pageStr, limitStr string) (int, int, error) {
	page := 1
	limit := 20

	if pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p < 1 {
			return 0, 0, fmt.Errorf("invalid page parameter: %s", pageStr)
		}
		page = p
	}

	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil || l < 1 {
			return 0, 0, fmt.Errorf("invalid limit parameter: %s", limitStr)
		}
		if l > 100 {
			return 0, 0, fmt.Errorf("limit cannot exceed 100")
		}
		limit = l
	}

	return page, limit, nil
}

// ValidateSortOrder validates sort order parameter
func ValidateSortOrder(order string) (string, error) {
	order = strings.ToLower(order)
	if order == "" {
		return "asc", nil
	}
	if order != "asc" && order != "desc" {
		return "", fmt.Errorf("invalid sort order: %s (must be 'asc' or 'desc')", order)
	}
	return order, nil
}

// ValidateFieldSelection validates field selection parameter
func ValidateFieldSelection(fieldsStr string, allowedFields map[string]bool) ([]string, error) {
	if fieldsStr == "" {
		return nil, nil
	}

	fields := strings.Split(fieldsStr, ",")
	var validatedFields []string

	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		// If allowedFields is nil, allow all fields
		if allowedFields != nil && !allowedFields[field] {
			return nil, fmt.Errorf("field '%s' is not allowed", field)
		}

		validatedFields = append(validatedFields, field)
	}

	return validatedFields, nil
}

// ExtractQueryParams extracts and validates query parameters from echo context
func ExtractQueryParams(c interface {
	QueryParam(string) string
}) (page int, limit int, search string, sort string, order string, err error) {
	pageStr := c.QueryParam("page")
	limitStr := c.QueryParam("limit")
	search = c.QueryParam("search")
	sort = c.QueryParam("sort")
	order = c.QueryParam("order")

	page, limit, err = ValidateQueryParams(pageStr, limitStr)
	if err != nil {
		return 0, 0, "", "", "", err
	}

	order, err = ValidateSortOrder(order)
	if err != nil {
		return 0, 0, "", "", "", err
	}

	return page, limit, search, sort, order, nil
}

