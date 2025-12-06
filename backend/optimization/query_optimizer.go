package optimization

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// QueryOptimizer provides database query optimization and monitoring
type QueryOptimizer struct {
	db      *sql.DB
	metrics *QueryMetrics
	config  *OptimizerConfig
	mu      sync.RWMutex
	cache   map[string]*QueryPlan
}

// OptimizerConfig holds configuration for the query optimizer
type OptimizerConfig struct {
	SlowQueryThreshold time.Duration
	CacheSize          int
	EnableMetrics      bool
	LogSlowQueries     bool
	MaxRetries         int
}

// QueryMetrics holds Prometheus metrics for query performance
type QueryMetrics struct {
	QueryDuration  *prometheus.HistogramVec
	SlowQueries    *prometheus.CounterVec
	CacheHits      *prometheus.CounterVec
	QueryErrors    *prometheus.CounterVec
	ConnectionPool *prometheus.GaugeVec
}

// QueryPlan represents an optimized query execution plan
type QueryPlan struct {
	Query       string
	Parameters  []interface{}
	Indexes     []string
	EstimatedMs float64
	CachedAt    time.Time
}

// QueryResult holds the result of an optimized query
type QueryResult struct {
	Rows      *sql.Rows
	Duration  time.Duration
	FromCache bool
	Plan      *QueryPlan
}

// NewQueryOptimizer creates a new query optimizer instance
func NewQueryOptimizer(db *sql.DB, config *OptimizerConfig) *QueryOptimizer {
	metrics := &QueryMetrics{
		QueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.001, 0.01, 0.1, 1, 5, 10},
			},
			[]string{"query_type", "table", "optimized"},
		),
		SlowQueries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_slow_queries_total",
				Help: "Total number of slow database queries",
			},
			[]string{"query_type", "table"},
		),
		CacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_query_cache_hits_total",
				Help: "Total number of query plan cache hits",
			},
			[]string{"hit_type"},
		),
		QueryErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_query_errors_total",
				Help: "Total number of database query errors",
			},
			[]string{"error_type"},
		),
		ConnectionPool: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "db_connection_pool_stats",
				Help: "Database connection pool statistics",
			},
			[]string{"stat_type"},
		),
	}

	return &QueryOptimizer{
		db:      db,
		metrics: metrics,
		config:  config,
		cache:   make(map[string]*QueryPlan),
	}
}

// OptimizeQuery analyzes and optimizes a SQL query
func (qo *QueryOptimizer) OptimizeQuery(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	start := time.Now()

	// Check cache first
	cacheKey := qo.getCacheKey(query, args...)
	if plan := qo.getCachedPlan(cacheKey); plan != nil {
		qo.metrics.CacheHits.WithLabelValues("hit").Inc()
		return qo.executeWithPlan(ctx, plan)
	}

	qo.metrics.CacheHits.WithLabelValues("miss").Inc()

	// Analyze query for optimization opportunities
	plan, err := qo.analyzeQuery(ctx, query, args...)
	if err != nil {
		qo.metrics.QueryErrors.WithLabelValues("analysis_error").Inc()
		return nil, fmt.Errorf("query analysis failed: %w", err)
	}

	// Cache the plan
	qo.cachePlan(cacheKey, plan)

	// Execute optimized query
	result, err := qo.executeWithPlan(ctx, plan)
	if err != nil {
		qo.metrics.QueryErrors.WithLabelValues("execution_error").Inc()
		return nil, err
	}

	duration := time.Since(start)

	// Record metrics
	queryType := qo.getQueryType(query)
	tableName := qo.extractTableName(query)

	qo.metrics.QueryDuration.WithLabelValues(queryType, tableName, "true").Observe(duration.Seconds())

	if duration > qo.config.SlowQueryThreshold {
		qo.metrics.SlowQueries.WithLabelValues(queryType, tableName).Inc()
		if qo.config.LogSlowQueries {
			log.Printf("Slow query detected: %s (%.2fms)", query, duration.Seconds()*1000)
		}
	}

	result.Duration = duration
	return result, nil
}

// analyzeQuery analyzes a query and creates an optimization plan
func (qo *QueryOptimizer) analyzeQuery(ctx context.Context, query string, args ...interface{}) (*QueryPlan, error) {
	plan := &QueryPlan{
		Query:      query,
		Parameters: args,
		CachedAt:   time.Now(),
	}

	// Check if query can benefit from indexes
	indexes := qo.suggestIndexes(query)
	plan.Indexes = indexes

	// Estimate query cost
	plan.EstimatedMs = qo.estimateQueryCost(query)

	// Apply query optimizations
	optimizedQuery := qo.optimizeQueryString(query)
	plan.Query = optimizedQuery

	return plan, nil
}

// suggestIndexes suggests indexes that could improve query performance
func (qo *QueryOptimizer) suggestIndexes(query string) []string {
	var indexes []string
	queryLower := strings.ToLower(query)

	// Common patterns that benefit from indexes
	patterns := map[string]string{
		"where user_id =":        "idx_user_id",
		"where api_key_id =":     "idx_api_key_id",
		"where created_at >":     "idx_created_at",
		"where updated_at >":     "idx_updated_at",
		"where status =":         "idx_status",
		"where email =":          "idx_email",
		"order by created_at":    "idx_created_at",
		"order by updated_at":    "idx_updated_at",
		"join on user_id":        "idx_user_id",
		"where name like":        "idx_name_fts",
		"where description like": "idx_description_fts",
	}

	for pattern, index := range patterns {
		if strings.Contains(queryLower, pattern) {
			indexes = append(indexes, index)
		}
	}

	return indexes
}

// optimizeQueryString applies string-level optimizations to the query
func (qo *QueryOptimizer) optimizeQueryString(query string) string {
	// Remove unnecessary whitespace
	query = strings.TrimSpace(query)

	// Add query hints for better performance
	if strings.Contains(strings.ToLower(query), "select") && !strings.Contains(strings.ToLower(query), "limit") {
		// Add reasonable limits to prevent runaway queries
		if !strings.Contains(strings.ToLower(query), "count(") {
			query += " LIMIT 1000"
		}
	}

	return query
}

// estimateQueryCost provides a rough estimate of query execution time
func (qo *QueryOptimizer) estimateQueryCost(query string) float64 {
	queryLower := strings.ToLower(query)
	cost := 1.0 // Base cost in milliseconds

	// Increase cost for complex operations
	if strings.Contains(queryLower, "join") {
		cost *= 2.0
	}
	if strings.Contains(queryLower, "group by") {
		cost *= 1.5
	}
	if strings.Contains(queryLower, "order by") {
		cost *= 1.3
	}
	if strings.Contains(queryLower, "like") {
		cost *= 3.0
	}
	if strings.Contains(queryLower, "distinct") {
		cost *= 2.0
	}

	return cost
}

// executeWithPlan executes a query using the optimized plan
func (qo *QueryOptimizer) executeWithPlan(ctx context.Context, plan *QueryPlan) (*QueryResult, error) {
	rows, err := qo.db.QueryContext(ctx, plan.Query, plan.Parameters...)
	if err != nil {
		return nil, err
	}

	return &QueryResult{
		Rows:      rows,
		FromCache: false,
		Plan:      plan,
	}, nil
}

// getCacheKey generates a cache key for a query and its parameters
func (qo *QueryOptimizer) getCacheKey(query string, args ...interface{}) string {
	key := query
	for _, arg := range args {
		key += fmt.Sprintf("_%v", arg)
	}
	return key
}

// getCachedPlan retrieves a cached query plan
func (qo *QueryOptimizer) getCachedPlan(key string) *QueryPlan {
	qo.mu.RLock()
	defer qo.mu.RUnlock()

	plan, exists := qo.cache[key]
	if !exists {
		return nil
	}

	// Check if plan is still valid (not too old)
	if time.Since(plan.CachedAt) > 1*time.Hour {
		delete(qo.cache, key)
		return nil
	}

	return plan
}

// cachePlan stores a query plan in the cache
func (qo *QueryOptimizer) cachePlan(key string, plan *QueryPlan) {
	qo.mu.Lock()
	defer qo.mu.Unlock()

	// Implement LRU eviction if cache is full
	if len(qo.cache) >= qo.config.CacheSize {
		qo.evictOldestPlan()
	}

	qo.cache[key] = plan
}

// evictOldestPlan removes the oldest plan from cache
func (qo *QueryOptimizer) evictOldestPlan() {
	var oldestKey string
	var oldestTime time.Time

	for key, plan := range qo.cache {
		if oldestKey == "" || plan.CachedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = plan.CachedAt
		}
	}

	if oldestKey != "" {
		delete(qo.cache, oldestKey)
	}
}

// getQueryType extracts the query type (SELECT, INSERT, UPDATE, DELETE)
func (qo *QueryOptimizer) getQueryType(query string) string {
	queryLower := strings.ToLower(strings.TrimSpace(query))

	if strings.HasPrefix(queryLower, "select") {
		return "SELECT"
	} else if strings.HasPrefix(queryLower, "insert") {
		return "INSERT"
	} else if strings.HasPrefix(queryLower, "update") {
		return "UPDATE"
	} else if strings.HasPrefix(queryLower, "delete") {
		return "DELETE"
	}

	return "OTHER"
}

// extractTableName extracts the primary table name from a query
func (qo *QueryOptimizer) extractTableName(query string) string {
	queryLower := strings.ToLower(query)

	// Simple table name extraction
	if strings.Contains(queryLower, "from ") {
		parts := strings.Split(queryLower, "from ")
		if len(parts) > 1 {
			tablePart := strings.TrimSpace(parts[1])
			tableWords := strings.Fields(tablePart)
			if len(tableWords) > 0 {
				return tableWords[0]
			}
		}
	}

	return "unknown"
}

// MonitorConnectionPool monitors database connection pool health
func (qo *QueryOptimizer) MonitorConnectionPool() {
	if !qo.config.EnableMetrics {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := qo.db.Stats()

		qo.metrics.ConnectionPool.WithLabelValues("open_connections").Set(float64(stats.OpenConnections))
		qo.metrics.ConnectionPool.WithLabelValues("in_use").Set(float64(stats.InUse))
		qo.metrics.ConnectionPool.WithLabelValues("idle").Set(float64(stats.Idle))
		qo.metrics.ConnectionPool.WithLabelValues("wait_count").Set(float64(stats.WaitCount))
		qo.metrics.ConnectionPool.WithLabelValues("wait_duration_ms").Set(float64(stats.WaitDuration.Milliseconds()))
	}
}

// GetOptimizationReport generates a report of optimization opportunities
func (qo *QueryOptimizer) GetOptimizationReport() map[string]interface{} {
	qo.mu.RLock()
	defer qo.mu.RUnlock()

	report := map[string]interface{}{
		"cached_plans":      len(qo.cache),
		"cache_size_limit":  qo.config.CacheSize,
		"slow_threshold_ms": qo.config.SlowQueryThreshold.Milliseconds(),
	}

	// Analyze cached plans for common patterns
	indexSuggestions := make(map[string]int)
	for _, plan := range qo.cache {
		for _, index := range plan.Indexes {
			indexSuggestions[index]++
		}
	}

	report["suggested_indexes"] = indexSuggestions

	return report
}
