package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ProductionConfig holds production database configuration
type ProductionConfig struct {
	// Primary database connection
	DatabaseURL string

	// Read replica configuration (optional)
	ReadReplicaURL string

	// Connection pool settings
	MaxOpenConnections    int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
	ConnectionMaxIdleTime time.Duration

	// Health check settings
	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration

	// Retry settings
	MaxRetries    int
	RetryInterval time.Duration

	// Logging
	LogLevel      logger.LogLevel
	SlowThreshold time.Duration
}

// DefaultProductionConfig returns default production database configuration
func DefaultProductionConfig() *ProductionConfig {
	return &ProductionConfig{
		MaxOpenConnections:    25,
		MaxIdleConnections:    10,
		ConnectionMaxLifetime: 5 * time.Minute,
		ConnectionMaxIdleTime: 5 * time.Minute,
		HealthCheckInterval:   30 * time.Second,
		HealthCheckTimeout:    5 * time.Second,
		MaxRetries:            3,
		RetryInterval:         1 * time.Second,
		LogLevel:              logger.Warn, // Only warnings and errors in production
		SlowThreshold:         200 * time.Millisecond,
	}
}

// ProductionDatabase manages production database connections with pooling and failover
type ProductionDatabase struct {
	primaryDB     *gorm.DB
	replicaDB     *gorm.DB
	sqlDB         *sql.DB
	config        *ProductionConfig
	healthChecker *HealthChecker
}

// HealthChecker monitors database health
type HealthChecker struct {
	db       *ProductionDatabase
	interval time.Duration
	timeout  time.Duration
	stop     chan bool
}

// NewProductionDatabase creates a new production database instance
func NewProductionDatabase(config *ProductionConfig) (*ProductionDatabase, error) {
	if config == nil {
		config = DefaultProductionConfig()
	}

	// Configure GORM logger
	gormConfig := &gorm.Config{
		Logger: logger.New(
			log.New(log.Writer(), "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             config.SlowThreshold,
				LogLevel:                  config.LogLevel,
				IgnoreRecordNotFoundError: true,
			},
		),
		PrepareStmt:                              true, // Preprepare statements for better performance
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	// Connect to primary database
	primaryDB, err := gorm.Open(postgres.Open(config.DatabaseURL), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to primary database: %w", err)
	}

	// Get underlying SQL DB for connection pool configuration
	sqlDB, err := primaryDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(config.MaxOpenConnections)
	sqlDB.SetMaxIdleConns(config.MaxIdleConnections)
	sqlDB.SetConnMaxLifetime(config.ConnectionMaxLifetime)
	sqlDB.SetConnMaxIdleTime(config.ConnectionMaxIdleTime)

	prodDB := &ProductionDatabase{
		primaryDB: primaryDB,
		sqlDB:     sqlDB,
		config:    config,
	}

	// Connect to read replica if configured
	if config.ReadReplicaURL != "" {
		replicaDB, err := gorm.Open(postgres.Open(config.ReadReplicaURL), gormConfig)
		if err != nil {
			log.Printf("Warning: failed to connect to read replica: %v", err)
		} else {
			prodDB.replicaDB = replicaDB

			// Configure replica connection pool
			if replicaSQLDB, err := replicaDB.DB(); err == nil {
				replicaSQLDB.SetMaxOpenConns(config.MaxOpenConnections)
				replicaSQLDB.SetMaxIdleConns(config.MaxIdleConnections)
				replicaSQLDB.SetConnMaxLifetime(config.ConnectionMaxLifetime)
				replicaSQLDB.SetConnMaxIdleTime(config.ConnectionMaxIdleTime)
			}
		}
	}

	// Start health checker
	healthChecker := &HealthChecker{
		db:       prodDB,
		interval: config.HealthCheckInterval,
		timeout:  config.HealthCheckTimeout,
		stop:     make(chan bool),
	}

	prodDB.healthChecker = healthChecker
	go healthChecker.Start()

	log.Println("✅ Production database connected successfully")
	if prodDB.replicaDB != nil {
		log.Println("✅ Read replica connected successfully")
	}

	return prodDB, nil
}

// GetReadDB returns the appropriate database for read operations
// Uses replica if available, falls back to primary
func (db *ProductionDatabase) GetReadDB() *gorm.DB {
	if db.replicaDB != nil {
		// Check if replica is healthy
		if sqlDB, err := db.replicaDB.DB(); err == nil {
			if err := sqlDB.Ping(); err == nil {
				return db.replicaDB
			}
			log.Printf("Read replica unhealthy, falling back to primary: %v", err)
		}
	}
	return db.primaryDB
}

// GetWriteDB returns the primary database for write operations
func (db *ProductionDatabase) GetWriteDB() *gorm.DB {
	return db.primaryDB
}

// GetDB returns the primary database (for backward compatibility)
func (db *ProductionDatabase) GetDB() *gorm.DB {
	return db.primaryDB
}

// Health performs health check on all database connections
func (db *ProductionDatabase) Health() error {
	// Check primary database
	if sqlDB, err := db.primaryDB.DB(); err == nil {
		if err := sqlDB.Ping(); err != nil {
			return fmt.Errorf("primary database unhealthy: %w", err)
		}
	} else {
		return fmt.Errorf("cannot access primary database: %w", err)
	}

	// Check replica if configured
	if db.replicaDB != nil {
		if sqlDB, err := db.replicaDB.DB(); err == nil {
			if err := sqlDB.Ping(); err != nil {
				log.Printf("Read replica health check failed: %v", err)
				// Don't return error, just log it
			}
		}
	}

	return nil
}

// Stats returns database connection pool statistics
func (db *ProductionDatabase) Stats() map[string]interface{} {
	stats := make(map[string]interface{})

	if sqlDB, err := db.primaryDB.DB(); err == nil {
		dbStats := sqlDB.Stats()
		stats["primary"] = map[string]interface{}{
			"open_connections":     dbStats.OpenConnections,
			"in_use":               dbStats.InUse,
			"idle":                 dbStats.Idle,
			"wait_count":           dbStats.WaitCount,
			"wait_duration":        dbStats.WaitDuration.String(),
			"max_idle_closed":      dbStats.MaxIdleClosed,
			"max_idle_time_closed": dbStats.MaxIdleTimeClosed,
			"max_lifetime_closed":  dbStats.MaxLifetimeClosed,
		}
	}

	if db.replicaDB != nil {
		if sqlDB, err := db.replicaDB.DB(); err == nil {
			dbStats := sqlDB.Stats()
			stats["replica"] = map[string]interface{}{
				"open_connections":     dbStats.OpenConnections,
				"in_use":               dbStats.InUse,
				"idle":                 dbStats.Idle,
				"wait_count":           dbStats.WaitCount,
				"wait_duration":        dbStats.WaitDuration.String(),
				"max_idle_closed":      dbStats.MaxIdleClosed,
				"max_idle_time_closed": dbStats.MaxIdleTimeClosed,
				"max_lifetime_closed":  dbStats.MaxLifetimeClosed,
			}
		}
	}

	return stats
}

// Close closes all database connections and stops health checker
func (db *ProductionDatabase) Close() error {
	// Stop health checker
	if db.healthChecker != nil {
		db.healthChecker.Stop()
	}

	var errors []error

	// Close primary database
	if db.sqlDB != nil {
		if err := db.sqlDB.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close primary database: %w", err))
		}
	}

	// Close replica database
	if db.replicaDB != nil {
		if replicaSQLDB, err := db.replicaDB.DB(); err == nil {
			if err := replicaSQLDB.Close(); err != nil {
				errors = append(errors, fmt.Errorf("failed to close replica database: %w", err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("database close errors: %v", errors)
	}

	log.Println("✅ Production database connections closed")
	return nil
}

// Start begins the health checking routine
func (hc *HealthChecker) Start() {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := hc.db.Health(); err != nil {
				log.Printf("Database health check failed: %v", err)
			}
		case <-hc.stop:
			return
		}
	}
}

// Stop stops the health checking routine
func (hc *HealthChecker) Stop() {
	close(hc.stop)
}

// RetryOperation retries a database operation with exponential backoff
func (db *ProductionDatabase) RetryOperation(operation func() error) error {
	var lastErr error

	for attempt := 0; attempt < db.config.MaxRetries; attempt++ {
		if err := operation(); err != nil {
			lastErr = err

			// Don't retry on certain errors
			if isNonRetryableError(err) {
				return err
			}

			if attempt < db.config.MaxRetries-1 {
				backoff := time.Duration(attempt+1) * db.config.RetryInterval
				log.Printf("Database operation failed (attempt %d/%d), retrying in %v: %v",
					attempt+1, db.config.MaxRetries, backoff, err)
				time.Sleep(backoff)
			}
		} else {
			return nil
		}
	}

	return fmt.Errorf("database operation failed after %d attempts: %w", db.config.MaxRetries, lastErr)
}

// isNonRetryableError checks if an error should not be retried
func isNonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	nonRetryableErrors := []string{
		"constraint violation",
		"unique constraint",
		"foreign key constraint",
		"invalid input syntax",
		"division by zero",
		"out of range",
	}

	for _, nonRetryable := range nonRetryableErrors {
		if contains(errStr, nonRetryable) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexOf(s, substr) >= 0))
}

// indexOf returns the index of a substring in a string
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Migrate performs database migrations with retry logic
func (db *ProductionDatabase) Migrate(models ...interface{}) error {
	return db.RetryOperation(func() error {
		return db.primaryDB.AutoMigrate(models...)
	})
}

// CreateTables creates tables with retry logic
func (db *ProductionDatabase) CreateTables(models ...interface{}) error {
	return db.RetryOperation(func() error {
		return db.primaryDB.Migrator().CreateTable(models...)
	})
}

// Transaction executes a function within a database transaction with retry logic
func (db *ProductionDatabase) Transaction(fn func(*gorm.DB) error) error {
	return db.primaryDB.Transaction(fn)
}

// ReplicaTransaction executes a read-only transaction on the replica
func (db *ProductionDatabase) ReplicaTransaction(fn func(*gorm.DB) error) error {
	readDB := db.GetReadDB()
	return readDB.Transaction(fn)
}
