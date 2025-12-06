package utils

import (
	"fmt"
	"strings"
)

// Filter represents a single filter condition
type Filter struct {
	Field    string
	Operator string // "eq", "ne", "gt", "gte", "lt", "lte", "like", "in", "between"
	Value    interface{}
}

// ParseFilterString parses a filter string like "field:value" or "field:operator:value"
func ParseFilterString(filterStr string) (*Filter, error) {
	parts := strings.Split(filterStr, ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid filter format: %s", filterStr)
	}

	filter := &Filter{
		Field:    parts[0],
		Operator: "eq", // default operator
		Value:    parts[1],
	}

	// Check for operator
	if len(parts) == 3 {
		filter.Operator = parts[1]
		filter.Value = parts[2]
	}

	// Handle range filters (e.g., "calories:1300-1500")
	if strings.Contains(filterStr, "-") && len(parts) == 2 {
		rangeParts := strings.Split(parts[1], "-")
		if len(rangeParts) == 2 {
			filter.Operator = "between"
			filter.Value = []string{rangeParts[0], rangeParts[1]}
		}
	}

	return filter, nil
}

// ParseMultipleFilters parses multiple filter strings
func ParseMultipleFilters(filterStrs []string) ([]*Filter, error) {
	var filters []*Filter
	for _, filterStr := range filterStrs {
		filter, err := ParseFilterString(filterStr)
		if err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

// BuildWhereClause builds a WHERE clause from filters
func BuildWhereClause(filters []*Filter, allowedFields map[string]bool) (string, []interface{}, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}

	var conditions []string
	var args []interface{}

	for _, filter := range filters {
		// Validate field is allowed
		if allowedFields != nil && !allowedFields[filter.Field] {
			return "", nil, fmt.Errorf("field '%s' is not allowed for filtering", filter.Field)
		}

		switch filter.Operator {
		case "eq":
			conditions = append(conditions, fmt.Sprintf("%s = ?", filter.Field))
			args = append(args, filter.Value)
		case "ne":
			conditions = append(conditions, fmt.Sprintf("%s != ?", filter.Field))
			args = append(args, filter.Value)
		case "gt":
			conditions = append(conditions, fmt.Sprintf("%s > ?", filter.Field))
			args = append(args, filter.Value)
		case "gte":
			conditions = append(conditions, fmt.Sprintf("%s >= ?", filter.Field))
			args = append(args, filter.Value)
		case "lt":
			conditions = append(conditions, fmt.Sprintf("%s < ?", filter.Field))
			args = append(args, filter.Value)
		case "lte":
			conditions = append(conditions, fmt.Sprintf("%s <= ?", filter.Field))
			args = append(args, filter.Value)
		case "like":
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", filter.Field))
			args = append(args, "%"+fmt.Sprintf("%v", filter.Value)+"%")
		case "in":
			// Handle array values
			if arr, ok := filter.Value.([]interface{}); ok {
				placeholders := strings.Repeat("?,", len(arr))
				placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma
				conditions = append(conditions, fmt.Sprintf("%s IN (%s)", filter.Field, placeholders))
				for _, v := range arr {
					args = append(args, v)
				}
			} else {
				return "", nil, fmt.Errorf("IN operator requires array value")
			}
		case "between":
			if arr, ok := filter.Value.([]string); ok && len(arr) == 2 {
				conditions = append(conditions, fmt.Sprintf("%s BETWEEN ? AND ?", filter.Field))
				args = append(args, arr[0], arr[1])
			} else {
				return "", nil, fmt.Errorf("BETWEEN operator requires array of 2 values")
			}
		default:
			return "", nil, fmt.Errorf("unsupported operator: %s", filter.Operator)
		}
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")
	return whereClause, args, nil
}

