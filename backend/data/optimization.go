package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/xeipuuv/gojsonschema"
)

// DataOptimizer provides data operation optimization features
type DataOptimizer struct {
	db          *sql.DB
	redisClient *redis.Client
	schemas     map[string]*gojsonschema.Schema
	mu          sync.RWMutex
	cacheConfig CacheConfig
	metrics     *OptimizationMetrics
}

// CacheConfig holds Redis caching configuration
type CacheConfig struct {
	DefaultTTL         time.Duration
	MaxKeySize         int
	MaxValueSize       int
	CompressionEnabled bool
	PrefixNamespace    string
	ClusterMode        bool
}

// OptimizationMetrics tracks data optimization metrics
type OptimizationMetrics struct {
	CacheHits          int64
	CacheMisses        int64
	ValidationErrors   int64
	QueryOptimizations int64
	PaginationRequests int64
	mu                 sync.RWMutex
}

// PaginationConfig defines pagination parameters
type PaginationConfig struct {
	Page    int                    `json:"page"`
	Limit   int                    `json:"limit"`
	SortBy  string                 `json:"sort_by"`
	SortDir string                 `json:"sort_dir"`
	Filters map[string]interface{} `json:"filters"`
	Search  string                 `json:"search"`
}

// PaginationResult contains paginated results
type PaginationResult struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int         `json:"total_pages"`
	HasNext    bool        `json:"has_next"`
	HasPrev    bool        `json:"has_prev"`
}

// ValidationResult contains schema validation results
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// QueryOptimization contains query optimization settings
type QueryOptimization struct {
	EagerLoading  []string               `json:"eager_loading"`
	SelectFields  []string               `json:"select_fields"`
	IndexHints    []string               `json:"index_hints"`
	JoinStrategy  string                 `json:"join_strategy"`
	CacheStrategy string                 `json:"cache_strategy"`
	BatchSize     int                    `json:"batch_size"`
	CustomFilters map[string]interface{} `json:"custom_filters"`
}

// NewDataOptimizer creates a new data optimizer
func NewDataOptimizer(db *sql.DB, redisClient *redis.Client, cacheConfig CacheConfig) *DataOptimizer {
	return &DataOptimizer{
		db:          db,
		redisClient: redisClient,
		schemas:     make(map[string]*gojsonschema.Schema),
		cacheConfig: cacheConfig,
		metrics:     &OptimizationMetrics{},
	}
}

// RegisterSchema registers a JSON schema for validation
func (do *DataOptimizer) RegisterSchema(name string, schemaJSON string) error {
	schemaLoader := gojsonschema.NewStringLoader(schemaJSON)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return fmt.Errorf("failed to compile schema %s: %w", name, err)
	}

	do.mu.Lock()
	do.schemas[name] = schema
	do.mu.Unlock()

	return nil
}

// ValidateJSON validates JSON data against a registered schema
func (do *DataOptimizer) ValidateJSON(schemaName string, data interface{}) ValidationResult {
	do.mu.RLock()
	schema, exists := do.schemas[schemaName]
	do.mu.RUnlock()

	if !exists {
		do.metrics.mu.Lock()
		do.metrics.ValidationErrors++
		do.metrics.mu.Unlock()
		return ValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("Schema %s not found", schemaName)},
		}
	}

	// Convert data to JSON for validation
	dataJSON, err := json.Marshal(data)
	if err != nil {
		do.metrics.mu.Lock()
		do.metrics.ValidationErrors++
		do.metrics.mu.Unlock()
		return ValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("Failed to marshal data: %v", err)},
		}
	}

	documentLoader := gojsonschema.NewBytesLoader(dataJSON)
	result, err := schema.Validate(documentLoader)
	if err != nil {
		do.metrics.mu.Lock()
		do.metrics.ValidationErrors++
		do.metrics.mu.Unlock()
		return ValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("Validation error: %v", err)},
		}
	}

	if result.Valid() {
		return ValidationResult{Valid: true}
	}

	// Collect validation errors
	errors := make([]string, 0, len(result.Errors()))
	for _, err := range result.Errors() {
		errors = append(errors, err.String())
	}

	do.metrics.mu.Lock()
	do.metrics.ValidationErrors++
	do.metrics.mu.Unlock()

	return ValidationResult{
		Valid:  false,
		Errors: errors,
	}
}

// CacheGet retrieves data from Redis cache
func (do *DataOptimizer) CacheGet(ctx context.Context, key string, dest interface{}) error {
	fullKey := do.buildCacheKey(key)

	val, err := do.redisClient.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			do.metrics.mu.Lock()
			do.metrics.CacheMisses++
			do.metrics.mu.Unlock()
			return fmt.Errorf("cache miss")
		}
		return fmt.Errorf("cache error: %w", err)
	}

	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		return fmt.Errorf("failed to unmarshal cached data: %w", err)
	}

	do.metrics.mu.Lock()
	do.metrics.CacheHits++
	do.metrics.mu.Unlock()

	return nil
}

// CacheSet stores data in Redis cache
func (do *DataOptimizer) CacheSet(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	fullKey := do.buildCacheKey(key)

	// Validate key and value sizes
	if len(fullKey) > do.cacheConfig.MaxKeySize {
		return fmt.Errorf("cache key too large: %d > %d", len(fullKey), do.cacheConfig.MaxKeySize)
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data for cache: %w", err)
	}

	if len(dataJSON) > do.cacheConfig.MaxValueSize {
		return fmt.Errorf("cache value too large: %d > %d", len(dataJSON), do.cacheConfig.MaxValueSize)
	}

	if ttl == 0 {
		ttl = do.cacheConfig.DefaultTTL
	}

	err = do.redisClient.Set(ctx, fullKey, dataJSON, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// CacheDelete removes data from Redis cache
func (do *DataOptimizer) CacheDelete(ctx context.Context, key string) error {
	fullKey := do.buildCacheKey(key)
	return do.redisClient.Del(ctx, fullKey).Err()
}

// CacheInvalidatePattern invalidates cache keys matching a pattern
func (do *DataOptimizer) CacheInvalidatePattern(ctx context.Context, pattern string) error {
	fullPattern := do.buildCacheKey(pattern)
	keys, err := do.redisClient.Keys(ctx, fullPattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys for pattern %s: %w", fullPattern, err)
	}

	if len(keys) > 0 {
		err = do.redisClient.Del(ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("failed to delete keys: %w", err)
		}
	}

	return nil
}

// buildCacheKey builds a full cache key with namespace
func (do *DataOptimizer) buildCacheKey(key string) string {
	if do.cacheConfig.PrefixNamespace != "" {
		return fmt.Sprintf("%s:%s", do.cacheConfig.PrefixNamespace, key)
	}
	return key
}

// PaginateQuery executes a paginated database query
func (do *DataOptimizer) PaginateQuery(ctx context.Context, baseQuery string, config PaginationConfig, optimization QueryOptimization) (*PaginationResult, error) {
	do.metrics.mu.Lock()
	do.metrics.PaginationRequests++
	do.metrics.mu.Unlock()

	// Validate pagination config
	if config.Page < 1 {
		config.Page = 1
	}
	if config.Limit < 1 || config.Limit > 1000 {
		config.Limit = 20 // Default limit
	}
	if config.SortDir != "ASC" && config.SortDir != "DESC" {
		config.SortDir = "ASC"
	}

	// Build optimized query
	optimizedQuery, countQuery := do.buildOptimizedQuery(baseQuery, config, optimization)

	// Get total count
	total, err := do.getQueryCount(ctx, countQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Calculate pagination info
	totalPages := int((total + int64(config.Limit) - 1) / int64(config.Limit))
	offset := (config.Page - 1) * config.Limit

	// Add LIMIT and OFFSET to query
	optimizedQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", config.Limit, offset)

	// Execute query
	rows, err := do.db.QueryContext(ctx, optimizedQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute paginated query: %w", err)
	}
	defer rows.Close()

	// Process results
	results, err := do.processQueryResults(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to process query results: %w", err)
	}

	return &PaginationResult{
		Data:       results,
		Total:      total,
		Page:       config.Page,
		Limit:      config.Limit,
		TotalPages: totalPages,
		HasNext:    config.Page < totalPages,
		HasPrev:    config.Page > 1,
	}, nil
}

// buildOptimizedQuery builds an optimized SQL query
func (do *DataOptimizer) buildOptimizedQuery(baseQuery string, config PaginationConfig, optimization QueryOptimization) (string, string) {
	do.metrics.mu.Lock()
	do.metrics.QueryOptimizations++
	do.metrics.mu.Unlock()

	query := baseQuery
	countQuery := do.buildCountQuery(baseQuery)

	// Apply SELECT field optimization
	if len(optimization.SelectFields) > 0 {
		query = do.optimizeSelectFields(query, optimization.SelectFields)
	}

	// Apply WHERE conditions from filters
	if len(config.Filters) > 0 || config.Search != "" {
		whereClause := do.buildWhereClause(config.Filters, config.Search)
		if whereClause != "" {
			if strings.Contains(strings.ToUpper(query), "WHERE") {
				query += " AND " + whereClause
				countQuery += " AND " + whereClause
			} else {
				query += " WHERE " + whereClause
				countQuery += " WHERE " + whereClause
			}
		}
	}

	// Apply ORDER BY
	if config.SortBy != "" {
		orderClause := fmt.Sprintf(" ORDER BY %s %s", config.SortBy, config.SortDir)
		query += orderClause
	}

	// Apply index hints
	if len(optimization.IndexHints) > 0 {
		query = do.applyIndexHints(query, optimization.IndexHints)
	}

	return query, countQuery
}

// buildCountQuery builds a count query from the base query
func (do *DataOptimizer) buildCountQuery(baseQuery string) string {
	// Simple approach: replace SELECT clause with COUNT(*)
	upper := strings.ToUpper(baseQuery)
	fromIndex := strings.Index(upper, "FROM")
	if fromIndex == -1 {
		return "SELECT COUNT(*) " + baseQuery
	}
	return "SELECT COUNT(*) " + baseQuery[fromIndex:]
}

// optimizeSelectFields optimizes SELECT fields
func (do *DataOptimizer) optimizeSelectFields(query string, fields []string) string {
	// Simple implementation: replace SELECT * with specific fields
	if strings.Contains(strings.ToUpper(query), "SELECT *") {
		fieldList := strings.Join(fields, ", ")
		return strings.Replace(query, "SELECT *", "SELECT "+fieldList, 1)
	}
	return query
}

// buildWhereClause builds WHERE clause from filters and search
func (do *DataOptimizer) buildWhereClause(filters map[string]interface{}, search string) string {
	conditions := make([]string, 0)

	// Add filter conditions
	for field, value := range filters {
		switch v := value.(type) {
		case string:
			conditions = append(conditions, fmt.Sprintf("%s = '%s'", field, v))
		case int, int64:
			conditions = append(conditions, fmt.Sprintf("%s = %v", field, v))
		case []interface{}:
			if len(v) > 0 {
				values := make([]string, len(v))
				for i, val := range v {
					values[i] = fmt.Sprintf("'%v'", val)
				}
				conditions = append(conditions, fmt.Sprintf("%s IN (%s)", field, strings.Join(values, ", ")))
			}
		}
	}

	// Add search condition (simplified)
	if search != "" {
		// This is a simplified search - in practice, you'd want to specify searchable fields
		conditions = append(conditions, fmt.Sprintf("(name ILIKE '%%%s%%' OR description ILIKE '%%%s%%')", search, search))
	}

	return strings.Join(conditions, " AND ")
}

// applyIndexHints applies database index hints
func (do *DataOptimizer) applyIndexHints(query string, hints []string) string {
	// This is database-specific. For PostgreSQL, you might use different syntax
	// For simplicity, this is a placeholder
	return query
}

// getQueryCount executes a count query
func (do *DataOptimizer) getQueryCount(ctx context.Context, countQuery string) (int64, error) {
	var count int64
	err := do.db.QueryRowContext(ctx, countQuery).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// processQueryResults processes SQL query results
func (do *DataOptimizer) processQueryResults(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0)

	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return nil, err
		}

		// Create a map for this row
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if val != nil {
				// Convert byte arrays to strings
				if b, ok := val.([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = val
				}
			} else {
				row[col] = nil
			}
		}
		results = append(results, row)
	}

	return results, nil
}

// BatchInsert performs optimized batch insert operations
func (do *DataOptimizer) BatchInsert(ctx context.Context, table string, data []map[string]interface{}, batchSize int) error {
	if len(data) == 0 {
		return nil
	}

	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}

	// Process data in batches
	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}

		batch := data[i:end]
		err := do.executeBatchInsert(ctx, table, batch)
		if err != nil {
			return fmt.Errorf("batch insert failed at batch %d: %w", i/batchSize, err)
		}
	}

	return nil
}

// executeBatchInsert executes a single batch insert
func (do *DataOptimizer) executeBatchInsert(ctx context.Context, table string, batch []map[string]interface{}) error {
	if len(batch) == 0 {
		return nil
	}

	// Get column names from the first record
	firstRecord := batch[0]
	columns := make([]string, 0, len(firstRecord))
	for col := range firstRecord {
		columns = append(columns, col)
	}

	// Build the INSERT query
	placeholders := make([]string, len(batch))
	values := make([]interface{}, 0, len(batch)*len(columns))

	for i, record := range batch {
		rowPlaceholders := make([]string, len(columns))
		for j, col := range columns {
			rowPlaceholders[j] = "$" + strconv.Itoa(i*len(columns)+j+1)
			values = append(values, record[col])
		}
		placeholders[i] = "(" + strings.Join(rowPlaceholders, ", ") + ")"
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := do.db.ExecContext(ctx, query, values...)
	return err
}

// GetOptimizationMetrics returns current optimization metrics
func (do *DataOptimizer) GetOptimizationMetrics() OptimizationMetrics {
	do.metrics.mu.RLock()
	defer do.metrics.mu.RUnlock()

	return OptimizationMetrics{
		CacheHits:          do.metrics.CacheHits,
		CacheMisses:        do.metrics.CacheMisses,
		ValidationErrors:   do.metrics.ValidationErrors,
		QueryOptimizations: do.metrics.QueryOptimizations,
		PaginationRequests: do.metrics.PaginationRequests,
	}
}

// ResetMetrics resets optimization metrics
func (do *DataOptimizer) ResetMetrics() {
	do.metrics.mu.Lock()
	defer do.metrics.mu.Unlock()

	do.metrics.CacheHits = 0
	do.metrics.CacheMisses = 0
	do.metrics.ValidationErrors = 0
	do.metrics.QueryOptimizations = 0
	do.metrics.PaginationRequests = 0
}

// GetCacheStats returns Redis cache statistics
func (do *DataOptimizer) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := do.redisClient.Info(ctx, "memory", "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	// Parse Redis INFO output (simplified)
	stats := make(map[string]interface{})
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				stats[parts[0]] = parts[1]
			}
		}
	}

	// Add our custom metrics
	metrics := do.GetOptimizationMetrics()
	stats["cache_hits"] = metrics.CacheHits
	stats["cache_misses"] = metrics.CacheMisses
	if metrics.CacheHits+metrics.CacheMisses > 0 {
		hitRate := float64(metrics.CacheHits) / float64(metrics.CacheHits+metrics.CacheMisses) * 100
		stats["hit_rate_percent"] = fmt.Sprintf("%.2f", hitRate)
	}

	return stats, nil
}

// PreventNPlusOne provides eager loading to prevent N+1 queries
func (do *DataOptimizer) PreventNPlusOne(ctx context.Context, baseQuery string, relations []string) ([]map[string]interface{}, error) {
	// Build query with JOINs for eager loading
	optimizedQuery := baseQuery

	for _, relation := range relations {
		// This is a simplified example - in practice, you'd need more sophisticated JOIN logic
		optimizedQuery += fmt.Sprintf(" LEFT JOIN %s ON %s", relation, "/* join condition */")
	}

	rows, err := do.db.QueryContext(ctx, optimizedQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute eager loading query: %w", err)
	}
	defer rows.Close()

	return do.processQueryResults(rows)
}

// AtomicUpdate performs atomic database updates with optimistic locking
func (do *DataOptimizer) AtomicUpdate(ctx context.Context, table string, id interface{}, updates map[string]interface{}, version int64) error {
	// Build UPDATE query with version check
	setClauses := make([]string, 0, len(updates))
	values := make([]interface{}, 0, len(updates)+2)
	paramIndex := 1

	for field, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, paramIndex))
		values = append(values, value)
		paramIndex++
	}

	// Add version increment
	setClauses = append(setClauses, fmt.Sprintf("version = $%d", paramIndex))
	values = append(values, version+1)
	paramIndex++

	// Add WHERE conditions
	values = append(values, id)
	idParam := paramIndex
	paramIndex++

	values = append(values, version)
	versionParam := paramIndex

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = $%d AND version = $%d",
		table,
		strings.Join(setClauses, ", "),
		idParam,
		versionParam,
	)

	result, err := do.db.ExecContext(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("atomic update failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("optimistic lock failed: record was modified by another transaction")
	}

	return nil
}
