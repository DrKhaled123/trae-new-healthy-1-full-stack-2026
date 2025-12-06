package utils

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// QueryCount returns total count for pagination
func QueryCount(db *sql.DB, query string, args ...interface{}) (int, error) {
	var count int
	err := db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("query count failed: %w", err)
	}
	return count, nil
}

// BuildWhereClauseFromMap constructs WHERE clause from filters map
func BuildWhereClauseFromMap(filters map[string]interface{}) (string, []interface{}) {
	if len(filters) == 0 {
		return "", nil
	}

	var conditions []string
	var args []interface{}
	argIndex := 1

	for key, value := range filters {
		if value != nil && value != "" {
			// Handle different value types
			switch v := value.(type) {
			case []interface{}:
				// Handle IN clause for arrays
				if len(v) > 0 {
					placeholders := make([]string, len(v))
					for i, item := range v {
						args = append(args, item)
						placeholders[i] = fmt.Sprintf("$%d", argIndex)
						argIndex++
					}
					conditions = append(conditions, fmt.Sprintf("%s IN (%s)", key, strings.Join(placeholders, ", ")))
				}
			case map[string]interface{}:
				// Handle range queries like {"min": 10, "max": 100}
				if minVal, hasMin := v["min"]; hasMin {
					conditions = append(conditions, fmt.Sprintf("%s >= $%d", key, argIndex))
					args = append(args, minVal)
					argIndex++
				}
				if maxVal, hasMax := v["max"]; hasMax {
					conditions = append(conditions, fmt.Sprintf("%s <= $%d", key, argIndex))
					args = append(args, maxVal)
					argIndex++
				}
			default:
				// Handle simple equality
				conditions = append(conditions, fmt.Sprintf("%s = $%d", key, argIndex))
				args = append(args, value)
				argIndex++
			}
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

// BuildOrderClause constructs ORDER BY clause from sort parameters
func BuildOrderClause(sortBy string, sortOrder string, allowedFields map[string]bool) string {
	if sortBy == "" {
		return ""
	}

	// Validate sort field
	if !allowedFields[sortBy] {
		return ""
	}

	// Default to ASC if invalid order
	if sortOrder != "DESC" && sortOrder != "ASC" {
		sortOrder = "ASC"
	}

	return fmt.Sprintf("ORDER BY %s %s", sortBy, sortOrder)
}

// BuildLimitOffset constructs LIMIT and OFFSET clause
func BuildLimitOffset(limit, offset int) string {
	if limit <= 0 {
		return ""
	}

	clause := fmt.Sprintf("LIMIT %d", limit)
	if offset > 0 {
		clause += fmt.Sprintf(" OFFSET %d", offset)
	}

	return clause
}

// BuildPaginationQuery constructs a complete paginated query
func BuildPaginationQuery(baseQuery string, filters map[string]interface{}, sortBy string, sortOrder string, page, limit int, allowedFields map[string]bool) (string, []interface{}) {
	var args []interface{}

	// Build WHERE clause
	whereClause, whereArgs := BuildWhereClauseFromMap(filters)
	args = append(args, whereArgs...)

	// Build ORDER BY clause
	orderClause := BuildOrderClause(sortBy, sortOrder, allowedFields)

	// Build LIMIT and OFFSET
	offset := CalculateOffset(page, limit)
	limitClause := BuildLimitOffset(limit, offset)

	// Combine all parts
	query := baseQuery
	if whereClause != "" {
		query += " " + whereClause
	}
	if orderClause != "" {
		query += " " + orderClause
	}
	if limitClause != "" {
		query += " " + limitClause
	}

	return query, args
}

// BuildSearchQuery constructs a search query with multiple search fields
func BuildSearchQuery(baseQuery string, searchTerm string, searchFields []string, filters map[string]interface{}, sortBy string, sortOrder string, page, limit int, allowedFields map[string]bool) (string, []interface{}) {
	var args []interface{}
	argIndex := 1

	// Build WHERE clause with search
	var conditions []string

	// Add search conditions
	if searchTerm != "" && len(searchFields) > 0 {
		searchConditions := make([]string, len(searchFields))
		for i, field := range searchFields {
			searchConditions[i] = fmt.Sprintf("%s ILIKE $%d", field, argIndex)
			args = append(args, "%"+searchTerm+"%")
			argIndex++
		}
		conditions = append(conditions, "("+strings.Join(searchConditions, " OR ")+")")
	}

	// Add filter conditions
	whereClause, whereArgs := BuildWhereClauseFromMap(filters)
	if whereClause != "" {
		// Remove "WHERE " prefix and add to conditions
		filterClause := strings.TrimPrefix(whereClause, "WHERE ")
		conditions = append(conditions, filterClause)

		// Adjust argument indices in whereArgs for PostgreSQL
		for _, arg := range whereArgs {
			args = append(args, arg)
		}
	}

	// Combine conditions
	var wherePart string
	if len(conditions) > 0 {
		wherePart = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build ORDER BY clause
	orderClause := BuildOrderClause(sortBy, sortOrder, allowedFields)

	// Build LIMIT and OFFSET
	offset := CalculateOffset(page, limit)
	limitClause := BuildLimitOffset(limit, offset)

	// Combine all parts
	query := baseQuery
	if wherePart != "" {
		query += " " + wherePart
	}
	if orderClause != "" {
		query += " " + orderClause
	}
	if limitClause != "" {
		query += " " + limitClause
	}

	return query, args
}

// ExecutePaginatedQuery executes a paginated query and returns results with count
func ExecutePaginatedQuery(db *sql.DB, query string, args []interface{}, scanFunc func(*sql.Rows) (interface{}, error)) ([]interface{}, int, error) {
	// Execute main query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("execute query failed: %w", err)
	}
	defer rows.Close()

	// Scan results
	var results []interface{}
	for rows.Next() {
		result, err := scanFunc(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan row failed: %w", err)
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	// Get total count
	countQuery, countArgs := buildCountQuery(query, args)
	total, err := QueryCount(db, countQuery, countArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("get count failed: %w", err)
	}

	return results, total, nil
}

// buildCountQuery converts a SELECT query to a COUNT query
func buildCountQuery(query string, args []interface{}) (string, []interface{}) {
	// Find SELECT and FROM to replace selection with COUNT
	lowerQuery := strings.ToLower(query)
	selectIndex := strings.Index(lowerQuery, "select")
	fromIndex := strings.Index(lowerQuery, "from")

	if selectIndex == -1 || fromIndex == -1 {
		return "SELECT COUNT(*)", nil
	}

	// Extract ORDER BY, LIMIT, OFFSET parts
	orderIndex := strings.LastIndex(lowerQuery, "order by")
	limitIndex := strings.LastIndex(lowerQuery, "limit")
	offsetIndex := strings.LastIndex(lowerQuery, "offset")

	// Find the earliest of ORDER BY, LIMIT, OFFSET
	cutIndex := len(query)
	if orderIndex != -1 && orderIndex < cutIndex {
		cutIndex = orderIndex
	}
	if limitIndex != -1 && limitIndex < cutIndex {
		cutIndex = limitIndex
	}
	if offsetIndex != -1 && offsetIndex < cutIndex {
		cutIndex = offsetIndex
	}

	baseQuery := query[:cutIndex]

	// Build COUNT query
	countQuery := strings.Replace(baseQuery, query[selectIndex:fromIndex], "SELECT COUNT(*)", 1)

	// Remove any ORDER BY from COUNT query (it doesn't make sense with COUNT)
	countQuery = strings.Split(countQuery, "ORDER BY")[0]

	return countQuery, args[:len(args)-getLimitOffsetArgsCount(query)]
}

// getLimitOffsetArgsCount counts how many args are used for LIMIT and OFFSET
func getLimitOffsetArgsCount(query string) int {
	count := 0
	lowerQuery := strings.ToLower(query)

	if strings.Contains(lowerQuery, "limit") {
		count++
	}
	if strings.Contains(lowerQuery, "offset") {
		count++
	}

	return count
}

// TransactionOptions defines options for database transactions
type TransactionOptions struct {
	IsolationLevel sql.IsolationLevel
	ReadOnly       bool
}

// DefaultTransactionOptions returns default transaction options
func DefaultTransactionOptions() *TransactionOptions {
	return &TransactionOptions{
		IsolationLevel: sql.LevelSerializable,
		ReadOnly:       false,
	}
}

// WithTransaction executes a function within a database transaction
func WithTransaction(db *sql.DB, opts *TransactionOptions, fn func(*sql.Tx) error) error {
	var tx *sql.Tx
	var err error

	if opts != nil {
		txOptions := &sql.TxOptions{
			Isolation: opts.IsolationLevel,
			ReadOnly:  opts.ReadOnly,
		}
		tx, err = db.BeginTx(context.Background(), txOptions)
	} else {
		tx, err = db.Begin()
	}

	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Re-throw panic after rollback
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = fn(tx)
	return err
}

// ExecuteInTransaction executes multiple queries in a transaction
func ExecuteInTransaction(db *sql.DB, queries []string, args [][]interface{}) error {
	return WithTransaction(db, DefaultTransactionOptions(), func(tx *sql.Tx) error {
		for i, query := range queries {
			var queryArgs []interface{}
			if i < len(args) {
				queryArgs = args[i]
			}

			_, err := tx.Exec(query, queryArgs...)
			if err != nil {
				return fmt.Errorf("execute query %d failed: %w", i+1, err)
			}
		}
		return nil
	})
}
