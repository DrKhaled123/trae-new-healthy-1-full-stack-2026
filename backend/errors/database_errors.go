package errors

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
)

// DatabaseErrorHandler handles database-specific errors and recovery
type DatabaseErrorHandler struct {
	db              *sql.DB
	config          *DatabaseConfig
	metrics         *DatabaseMetrics
	healthChecker   *DatabaseHealthChecker
	connectionPool  *ConnectionPool
	mu              sync.RWMutex
	lastHealthCheck time.Time
}

// DatabaseConfig holds database error handling configuration
type DatabaseConfig struct {
	MaxRetries          int
	RetryDelay          time.Duration
	HealthCheckInterval time.Duration
	ConnectionTimeout   time.Duration
	QueryTimeout        time.Duration
	MaxConnections      int
	MaxIdleConnections  int
	ConnectionLifetime  time.Duration
	EnableMetrics       bool
	LogSlowQueries      bool
	SlowQueryThreshold  time.Duration
}

// DatabaseMetrics holds Prometheus metrics for database operations
type DatabaseMetrics struct {
	ConnectionsActive   prometheus.Gauge
	ConnectionsIdle     prometheus.Gauge
	ConnectionsTotal    *prometheus.CounterVec
	QueryDuration       *prometheus.HistogramVec
	QueryErrors         *prometheus.CounterVec
	TransactionDuration *prometheus.HistogramVec
	Deadlocks           prometheus.Counter
	SlowQueries         prometheus.Counter
	HealthCheckStatus   prometheus.Gauge
}

// DatabaseHealthChecker monitors database health
type DatabaseHealthChecker struct {
	db           *sql.DB
	config       *DatabaseConfig
	metrics      *DatabaseMetrics
	lastCheck    time.Time
	isHealthy    bool
	failureCount int
	mu           sync.RWMutex
}

// ConnectionPool manages database connections with error handling
type ConnectionPool struct {
	db      *sql.DB
	config  *DatabaseConfig
	metrics *DatabaseMetrics
}

// NewDatabaseErrorHandler creates a new database error handler
func NewDatabaseErrorHandler(db *sql.DB, config *DatabaseConfig) *DatabaseErrorHandler {
	metrics := newDatabaseMetrics()

	handler := &DatabaseErrorHandler{
		db:             db,
		config:         config,
		metrics:        metrics,
		healthChecker:  NewDatabaseHealthChecker(db, config, metrics),
		connectionPool: NewConnectionPool(db, config, metrics),
	}

	// Start health monitoring
	go handler.startHealthMonitoring()

	return handler
}

// newDatabaseMetrics creates Prometheus metrics for database monitoring
func newDatabaseMetrics() *DatabaseMetrics {
	m := &DatabaseMetrics{
		ConnectionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nutrition_platform_db_connections_active",
				Help: "Number of active database connections",
			},
		),
		ConnectionsIdle: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nutrition_platform_db_connections_idle",
				Help: "Number of idle database connections",
			},
		),
		ConnectionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nutrition_platform_db_connections_total",
				Help: "Total number of database connections created",
			},
			[]string{"status"},
		),
		QueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nutrition_platform_db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
			},
			[]string{"operation", "table"},
		),
		QueryErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nutrition_platform_db_query_errors_total",
				Help: "Total number of database query errors",
			},
			[]string{"error_type", "operation"},
		),
		TransactionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nutrition_platform_db_transaction_duration_seconds",
				Help:    "Database transaction duration in seconds",
				Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0},
			},
			[]string{"status"},
		),
		Deadlocks: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "nutrition_platform_db_deadlocks_total",
				Help: "Total number of database deadlocks",
			},
		),
		SlowQueries: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "nutrition_platform_db_slow_queries_total",
				Help: "Total number of slow database queries",
			},
		),
		HealthCheckStatus: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nutrition_platform_db_health_status",
				Help: "Database health status (1=healthy, 0=unhealthy)",
			},
		),
	}

	// Register metrics
	prometheus.MustRegister(m.ConnectionsActive)
	prometheus.MustRegister(m.ConnectionsIdle)
	prometheus.MustRegister(m.ConnectionsTotal)
	prometheus.MustRegister(m.QueryDuration)
	prometheus.MustRegister(m.QueryErrors)
	prometheus.MustRegister(m.TransactionDuration)
	prometheus.MustRegister(m.Deadlocks)
	prometheus.MustRegister(m.SlowQueries)
	prometheus.MustRegister(m.HealthCheckStatus)

	return m
}

// ExecuteWithRetry executes a database operation with retry logic
func (deh *DatabaseErrorHandler) ExecuteWithRetry(ctx context.Context, operation string, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= deh.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(deh.config.RetryDelay * time.Duration(attempt)):
			}
		}

		start := time.Now()
		err := fn()
		duration := time.Since(start)

		// Record metrics
		if err != nil {
			errorType := deh.classifyError(err)
			deh.metrics.QueryErrors.WithLabelValues(errorType, operation).Inc()
			lastErr = err

			// Don't retry certain errors
			if !deh.shouldRetryError(err) {
				break
			}
		} else {
			// Success
			deh.metrics.QueryDuration.WithLabelValues(operation, "unknown").Observe(duration.Seconds())

			// Check for slow queries
			if deh.config.LogSlowQueries && duration > deh.config.SlowQueryThreshold {
				deh.metrics.SlowQueries.Inc()
				log.Printf("Slow query detected: %s took %v", operation, duration)
			}

			return nil
		}
	}

	return deh.wrapDatabaseError(lastErr)
}

// QueryWithTimeout executes a query with timeout
func (deh *DatabaseErrorHandler) QueryWithTimeout(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ctx, cancel := context.WithTimeout(ctx, deh.config.QueryTimeout)
	defer cancel()

	return deh.ExecuteQueryWithRetry(ctx, "SELECT", func() (*sql.Rows, error) {
		return deh.db.QueryContext(ctx, query, args...)
	})
}

// ExecWithTimeout executes a statement with timeout
func (deh *DatabaseErrorHandler) ExecWithTimeout(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, deh.config.QueryTimeout)
	defer cancel()

	return deh.ExecuteExecWithRetry(ctx, "EXEC", func() (sql.Result, error) {
		return deh.db.ExecContext(ctx, query, args...)
	})
}

// ExecuteQueryWithRetry executes a query with retry logic
func (deh *DatabaseErrorHandler) ExecuteQueryWithRetry(ctx context.Context, operation string, fn func() (*sql.Rows, error)) (*sql.Rows, error) {
	var lastErr error
	var result *sql.Rows

	for attempt := 0; attempt <= deh.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(deh.config.RetryDelay * time.Duration(attempt)):
			}
		}

		start := time.Now()
		rows, err := fn()
		duration := time.Since(start)

		if err != nil {
			errorType := deh.classifyError(err)
			deh.metrics.QueryErrors.WithLabelValues(errorType, operation).Inc()
			lastErr = err

			if !deh.shouldRetryError(err) {
				break
			}
		} else {
			deh.metrics.QueryDuration.WithLabelValues(operation, "unknown").Observe(duration.Seconds())
			result = rows
			break
		}
	}

	if lastErr != nil {
		return nil, deh.wrapDatabaseError(lastErr)
	}

	return result, nil
}

// ExecuteExecWithRetry executes a statement with retry logic
func (deh *DatabaseErrorHandler) ExecuteExecWithRetry(ctx context.Context, operation string, fn func() (sql.Result, error)) (sql.Result, error) {
	var lastErr error
	var result sql.Result

	for attempt := 0; attempt <= deh.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(deh.config.RetryDelay * time.Duration(attempt)):
			}
		}

		start := time.Now()
		res, err := fn()
		duration := time.Since(start)

		if err != nil {
			errorType := deh.classifyError(err)
			deh.metrics.QueryErrors.WithLabelValues(errorType, operation).Inc()
			lastErr = err

			if !deh.shouldRetryError(err) {
				break
			}
		} else {
			deh.metrics.QueryDuration.WithLabelValues(operation, "unknown").Observe(duration.Seconds())
			result = res
			break
		}
	}

	if lastErr != nil {
		return nil, deh.wrapDatabaseError(lastErr)
	}

	return result, nil
}

// ExecuteTransaction executes a transaction with error handling
func (deh *DatabaseErrorHandler) ExecuteTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		deh.metrics.TransactionDuration.WithLabelValues("completed").Observe(duration.Seconds())
	}()

	tx, err := deh.db.BeginTx(ctx, nil)
	if err != nil {
		deh.metrics.TransactionDuration.WithLabelValues("failed").Observe(time.Since(start).Seconds())
		return deh.wrapDatabaseError(err)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	err = fn(tx)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Printf("Failed to rollback transaction: %v", rollbackErr)
		}
		deh.metrics.TransactionDuration.WithLabelValues("rolled_back").Observe(time.Since(start).Seconds())
		return deh.wrapDatabaseError(err)
	}

	if err = tx.Commit(); err != nil {
		deh.metrics.TransactionDuration.WithLabelValues("commit_failed").Observe(time.Since(start).Seconds())
		return deh.wrapDatabaseError(err)
	}

	return nil
}

// Helper methods

func (deh *DatabaseErrorHandler) classifyError(err error) string {
	if pqErr, ok := err.(*pq.Error); ok {
		switch pqErr.Code {
		case "40001": // serialization_failure
			return "deadlock"
		case "40P01": // deadlock_detected
			deh.metrics.Deadlocks.Inc()
			return "deadlock"
		case "23505": // unique_violation
			return "constraint_violation"
		case "23503": // foreign_key_violation
			return "foreign_key_violation"
		case "23514": // check_violation
			return "check_violation"
		case "08000", "08003", "08006": // connection errors
			return "connection_error"
		case "57014": // query_canceled
			return "timeout"
		default:
			return "database_error"
		}
	}

	if strings.Contains(err.Error(), "timeout") {
		return "timeout"
	}
	if strings.Contains(err.Error(), "connection") {
		return "connection_error"
	}

	return "unknown_error"
}

func (deh *DatabaseErrorHandler) shouldRetryError(err error) bool {
	errorType := deh.classifyError(err)
	switch errorType {
	case "deadlock", "timeout", "connection_error":
		return true
	case "constraint_violation", "foreign_key_violation", "check_violation":
		return false
	default:
		return false
	}
}

func (deh *DatabaseErrorHandler) wrapDatabaseError(err error) *APIError {
	errorType := deh.classifyError(err)

	switch errorType {
	case "deadlock":
		return NewAPIError(ErrDatabaseQuery, "Database deadlock detected", err.Error())
	case "timeout":
		return NewAPIError(ErrDatabaseTimeout, "Database operation timed out", err.Error())
	case "connection_error":
		return NewAPIError(ErrDatabaseConnection, "Database connection error", err.Error())
	case "constraint_violation":
		return NewAPIError(ErrInvalidInput, "Data constraint violation", err.Error())
	default:
		return NewAPIError(ErrDatabaseQuery, "Database operation failed", err.Error())
	}
}

func (deh *DatabaseErrorHandler) startHealthMonitoring() {
	ticker := time.NewTicker(deh.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		deh.performHealthCheck()
	}
}

func (deh *DatabaseErrorHandler) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := deh.db.PingContext(ctx)
	if err != nil {
		log.Printf("Database health check failed: %v", err)
		deh.metrics.HealthCheckStatus.Set(0)
	} else {
		deh.metrics.HealthCheckStatus.Set(1)
	}

	// Update connection metrics
	stats := deh.db.Stats()
	deh.metrics.ConnectionsActive.Set(float64(stats.OpenConnections))
	deh.metrics.ConnectionsIdle.Set(float64(stats.Idle))
}

// NewDatabaseHealthChecker creates a new database health checker
func NewDatabaseHealthChecker(db *sql.DB, config *DatabaseConfig, metrics *DatabaseMetrics) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{
		db:        db,
		config:    config,
		metrics:   metrics,
		isHealthy: true,
	}
}

// NewConnectionPool creates a new connection pool manager
func NewConnectionPool(db *sql.DB, config *DatabaseConfig, metrics *DatabaseMetrics) *ConnectionPool {
	return &ConnectionPool{
		db:      db,
		config:  config,
		metrics: metrics,
	}
}

// IsHealthy returns the current health status
func (dhc *DatabaseHealthChecker) IsHealthy() bool {
	dhc.mu.RLock()
	defer dhc.mu.RUnlock()
	return dhc.isHealthy
}

// GetConnectionStats returns current connection statistics
func (cp *ConnectionPool) GetConnectionStats() sql.DBStats {
	return cp.db.Stats()
}
