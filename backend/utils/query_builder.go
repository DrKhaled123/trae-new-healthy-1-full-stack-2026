package utils

import (
	"fmt"
	"strings"
)

// QueryBuilder helps build SQL queries with filters, search, sorting, and pagination
type QueryBuilder struct {
	baseQuery    string
	whereClause  string
	orderByClause string
	limitClause  string
	args         []interface{}
	tablePrefix  string
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(baseQuery string) *QueryBuilder {
	return &QueryBuilder{
		baseQuery: baseQuery,
		args:      make([]interface{}, 0),
	}
}

// SetTablePrefix sets the table prefix for field names
func (qb *QueryBuilder) SetTablePrefix(prefix string) {
	qb.tablePrefix = prefix
}

// AddWhere adds a WHERE clause
func (qb *QueryBuilder) AddWhere(clause string, args ...interface{}) {
	if qb.whereClause == "" {
		qb.whereClause = "WHERE " + clause
	} else {
		qb.whereClause += " AND " + clause
	}
	qb.args = append(qb.args, args...)
}

// AddFilters adds filters using the filtering utilities
func (qb *QueryBuilder) AddFilters(filters []*Filter, allowedFields map[string]bool) error {
	if len(filters) == 0 {
		return nil
	}

	whereClause, args, err := BuildWhereClause(filters, allowedFields)
	if err != nil {
		return err
	}

	if whereClause != "" {
		// Remove "WHERE " prefix since AddWhere will add it
		clause := strings.TrimPrefix(whereClause, "WHERE ")
		qb.AddWhere(clause, args...)
	}

	return nil
}

// AddSearch adds search clause
func (qb *QueryBuilder) AddSearch(searchQuery SearchQuery) {
	clause, args := BuildSearchClause(searchQuery, qb.tablePrefix)
	if clause != "" {
		qb.AddWhere(clause, args...)
	}
}

// AddSort adds ORDER BY clause
func (qb *QueryBuilder) AddSort(sortConfig *SortConfig, allowedFields map[string]bool, defaultField string) error {
	orderBy, err := BuildOrderByClause(sortConfig, allowedFields, defaultField)
	if err != nil {
		return err
	}
	qb.orderByClause = orderBy
	return nil
}

// AddPagination adds LIMIT and OFFSET
func (qb *QueryBuilder) AddPagination(page, limit int) {
	offset := GetOffset(page, limit)
	qb.limitClause = fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
}

// Build builds the final SQL query
func (qb *QueryBuilder) Build() (string, []interface{}) {
	query := qb.baseQuery

	if qb.whereClause != "" {
		query += " " + qb.whereClause
	}

	if qb.orderByClause != "" {
		query += " " + qb.orderByClause
	}

	if qb.limitClause != "" {
		query += " " + qb.limitClause
	}

	return query, qb.args
}

// BuildCountQuery builds a COUNT query for pagination
func (qb *QueryBuilder) BuildCountQuery() (string, []interface{}) {
	// Extract table name from base query (simplified - assumes SELECT ... FROM table)
	countQuery := strings.Replace(qb.baseQuery, "SELECT *", "SELECT COUNT(*)", 1)
	countQuery = strings.Replace(countQuery, "SELECT ", "SELECT COUNT(*) ", 1)
	
	// Remove ORDER BY and LIMIT from count query
	if idx := strings.Index(countQuery, "ORDER BY"); idx != -1 {
		countQuery = countQuery[:idx]
	}
	if idx := strings.Index(countQuery, "LIMIT"); idx != -1 {
		countQuery = countQuery[:idx]
	}

	if qb.whereClause != "" {
		countQuery += " " + qb.whereClause
	}

	return strings.TrimSpace(countQuery), qb.args
}

