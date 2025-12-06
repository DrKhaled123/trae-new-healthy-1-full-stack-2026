package utils

import (
	"fmt"
	"strings"
)

// SortConfig represents sorting configuration
type SortConfig struct {
	Field string
	Order string // "asc" or "desc"
}

// ParseSortString parses a sort string like "field" or "field:asc"
func ParseSortString(sortStr string) *SortConfig {
	parts := strings.Split(sortStr, ":")
	config := &SortConfig{
		Field: parts[0],
		Order: "asc", // default order
	}

	if len(parts) > 1 {
		order := strings.ToLower(parts[1])
		if order == "desc" || order == "descending" {
			config.Order = "desc"
		}
	}

	return config
}

// BuildOrderByClause builds an ORDER BY clause
func BuildOrderByClause(sortConfig *SortConfig, allowedFields map[string]bool, defaultField string) (string, error) {
	if sortConfig == nil || sortConfig.Field == "" {
		if defaultField != "" {
			return fmt.Sprintf("ORDER BY %s ASC", defaultField), nil
		}
		return "", nil
	}

	// Validate field is allowed
	if allowedFields != nil && !allowedFields[sortConfig.Field] {
		return "", fmt.Errorf("field '%s' is not allowed for sorting", sortConfig.Field)
	}

	order := strings.ToUpper(sortConfig.Order)
	if order != "ASC" && order != "DESC" {
		order = "ASC"
	}

	return fmt.Sprintf("ORDER BY %s %s", sortConfig.Field, order), nil
}

// ValidateSortField validates that a sort field is allowed
func ValidateSortField(field string, allowedFields map[string]bool) error {
	if allowedFields == nil {
		return nil // No restrictions
	}
	if !allowedFields[field] {
		return fmt.Errorf("field '%s' is not allowed for sorting", field)
	}
	return nil
}

