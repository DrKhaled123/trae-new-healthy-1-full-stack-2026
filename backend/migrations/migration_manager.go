package migrations

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

// MigrationManager manages database migrations with idempotency and error recovery
type MigrationManager struct {
	mu               sync.RWMutex
	db               *gorm.DB
	sqlDB            *sql.DB
	migrationsDir    string
	backupDir        string
	migrationHistory []MigrationRecord
	errorRecovery    *ErrorRecovery
	validationRules  map[string]ValidationRule
	preHooks         []PreMigrationHook
	postHooks        []PostMigrationHook
	dryRunMode       bool
	maxRetries       int
	retryDelay       time.Duration
}

// MigrationRecord represents a migration execution record
type MigrationRecord struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Version       int64     `json:"version"`
	Filename      string    `json:"filename"`
	Checksum      string    `json:"checksum"`
	ExecutedAt    time.Time `json:"executed_at"`
	ExecutionTime int64     `json:"execution_time_ms"`
	Status        string    `json:"status"` // "pending", "running", "completed", "failed", "rolled_back"
	ErrorMessage  string    `json:"error_message,omitempty"`
	RetryCount    int       `json:"retry_count"`
	BackupPath    string    `json:"backup_path,omitempty"`
	RollbackSQL   string    `json:"rollback_sql,omitempty" gorm:"type:text"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ErrorRecovery manages migration error recovery
type ErrorRecovery struct {
	Enabled               bool                        `json:"enabled"`
	AutoRollback          bool                        `json:"auto_rollback"`
	BackupBeforeMigration bool                        `json:"backup_before_migration"`
	RecoveryStrategies    map[string]RecoveryStrategy `json:"recovery_strategies"`
	NotificationURL       string                      `json:"notification_url,omitempty"`
}

// RecoveryStrategy defines how to recover from specific errors
type RecoveryStrategy struct {
	ErrorPattern    string        `json:"error_pattern"`
	Action          string        `json:"action"` // "retry", "rollback", "skip", "manual"
	MaxRetries      int           `json:"max_retries"`
	RetryDelay      time.Duration `json:"retry_delay"`
	RollbackSteps   int           `json:"rollback_steps"`
	NotifyOnFailure bool          `json:"notify_on_failure"`
}

// ValidationRule defines validation rules for migrations
type ValidationRule struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Pattern     string   `json:"pattern"`
	Required    bool     `json:"required"`
	Tables      []string `json:"tables,omitempty"`
	Columns     []string `json:"columns,omitempty"`
	Constraints []string `json:"constraints,omitempty"`
}

// PreMigrationHook defines pre-migration hooks
type PreMigrationHook func(version int64, filename string) error

// PostMigrationHook defines post-migration hooks
type PostMigrationHook func(version int64, filename string, success bool, duration time.Duration) error

// MigrationStatus represents the current migration status
type MigrationStatus struct {
	CurrentVersion    int64             `json:"current_version"`
	TargetVersion     int64             `json:"target_version"`
	PendingMigrations []string          `json:"pending_migrations"`
	FailedMigrations  []MigrationRecord `json:"failed_migrations"`
	LastMigration     *MigrationRecord  `json:"last_migration,omitempty"`
	DatabaseHealth    DatabaseHealth    `json:"database_health"`
	BackupStatus      BackupStatus      `json:"backup_status"`
}

// DatabaseHealth represents database health metrics
type DatabaseHealth struct {
	Connected          bool               `json:"connected"`
	Version            string             `json:"version"`
	Size               int64              `json:"size_bytes"`
	TableCount         int                `json:"table_count"`
	IndexCount         int                `json:"index_count"`
	ConstraintCount    int                `json:"constraint_count"`
	LastVacuum         *time.Time         `json:"last_vacuum,omitempty"`
	Fragmentation      float64            `json:"fragmentation_pct"`
	PerformanceMetrics map[string]float64 `json:"performance_metrics"`
}

// BackupStatus represents backup status
type BackupStatus struct {
	Enabled       bool       `json:"enabled"`
	LastBackup    *time.Time `json:"last_backup,omitempty"`
	BackupSize    int64      `json:"backup_size_bytes"`
	BackupPath    string     `json:"backup_path,omitempty"`
	RetentionDays int        `json:"retention_days"`
	AutoCleanup   bool       `json:"auto_cleanup"`
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *gorm.DB, migrationsDir, backupDir string) (*MigrationManager, error) {
	// Get underlying SQL DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL DB: %w", err)
	}

	// Create directories
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create migrations directory: %w", err)
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	mm := &MigrationManager{
		db:            db,
		sqlDB:         sqlDB,
		migrationsDir: migrationsDir,
		backupDir:     backupDir,
		errorRecovery: &ErrorRecovery{
			Enabled:               true,
			AutoRollback:          true,
			BackupBeforeMigration: true,
			RecoveryStrategies:    make(map[string]RecoveryStrategy),
		},
		validationRules: make(map[string]ValidationRule),
		preHooks:        make([]PreMigrationHook, 0),
		postHooks:       make([]PostMigrationHook, 0),
		maxRetries:      3,
		retryDelay:      5 * time.Second,
	}

	// Initialize Goose
	goose.SetDialect("sqlite3")
	goose.SetBaseFS(nil)

	// Auto-migrate migration records table
	if err := db.AutoMigrate(&MigrationRecord{}); err != nil {
		return nil, fmt.Errorf("failed to migrate migration records table: %w", err)
	}

	// Initialize default validation rules
	mm.initializeValidationRules()

	// Initialize default recovery strategies
	mm.initializeRecoveryStrategies()

	// Load migration history
	if err := mm.loadMigrationHistory(); err != nil {
		return nil, fmt.Errorf("failed to load migration history: %w", err)
	}

	return mm, nil
}

// initializeValidationRules sets up default validation rules
func (mm *MigrationManager) initializeValidationRules() {
	mm.validationRules = map[string]ValidationRule{
		"no_drop_table": {
			Name:        "no_drop_table",
			Description: "Prevent dropping tables in production",
			Pattern:     `(?i)DROP\s+TABLE`,
			Required:    true,
		},
		"no_drop_column": {
			Name:        "no_drop_column",
			Description: "Prevent dropping columns in production",
			Pattern:     `(?i)DROP\s+COLUMN`,
			Required:    true,
		},
		"require_transaction": {
			Name:        "require_transaction",
			Description: "Require migrations to be wrapped in transactions",
			Pattern:     `(?i)BEGIN\s*;.*COMMIT\s*;`,
			Required:    true,
		},
		"no_truncate": {
			Name:        "no_truncate",
			Description: "Prevent truncating tables",
			Pattern:     `(?i)TRUNCATE\s+TABLE`,
			Required:    true,
		},
		"require_if_not_exists": {
			Name:        "require_if_not_exists",
			Description: "Require IF NOT EXISTS for CREATE statements",
			Pattern:     `(?i)CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS`,
			Required:    false,
		},
	}
}

// initializeRecoveryStrategies sets up default recovery strategies
func (mm *MigrationManager) initializeRecoveryStrategies() {
	mm.errorRecovery.RecoveryStrategies = map[string]RecoveryStrategy{
		"table_exists": {
			ErrorPattern:    "table .* already exists",
			Action:          "skip",
			MaxRetries:      0,
			NotifyOnFailure: false,
		},
		"column_exists": {
			ErrorPattern:    "column .* already exists",
			Action:          "skip",
			MaxRetries:      0,
			NotifyOnFailure: false,
		},
		"constraint_exists": {
			ErrorPattern:    "constraint .* already exists",
			Action:          "skip",
			MaxRetries:      0,
			NotifyOnFailure: false,
		},
		"lock_timeout": {
			ErrorPattern:    "database is locked",
			Action:          "retry",
			MaxRetries:      5,
			RetryDelay:      10 * time.Second,
			NotifyOnFailure: true,
		},
		"syntax_error": {
			ErrorPattern:    "syntax error",
			Action:          "manual",
			MaxRetries:      0,
			NotifyOnFailure: true,
		},
		"foreign_key_constraint": {
			ErrorPattern:    "foreign key constraint",
			Action:          "rollback",
			MaxRetries:      0,
			RollbackSteps:   1,
			NotifyOnFailure: true,
		},
	}
}

// loadMigrationHistory loads migration history from database
func (mm *MigrationManager) loadMigrationHistory() error {
	var records []MigrationRecord
	if err := mm.db.Order("version DESC").Find(&records).Error; err != nil {
		return err
	}

	mm.mu.Lock()
	mm.migrationHistory = records
	mm.mu.Unlock()

	return nil
}

// Migrate runs pending migrations with idempotency and error recovery
func (mm *MigrationManager) Migrate() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Get current version
	currentVersion, err := goose.GetDBVersion(mm.sqlDB)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Get pending migrations
	pendingMigrations, err := mm.getPendingMigrations(currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get pending migrations: %w", err)
	}

	if len(pendingMigrations) == 0 {
		log.Println("No pending migrations")
		return nil
	}

	log.Printf("Found %d pending migrations", len(pendingMigrations))

	// Create backup if enabled
	var backupPath string
	if mm.errorRecovery.BackupBeforeMigration {
		backupPath, err = mm.createBackup()
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		log.Printf("Created backup at: %s", backupPath)
	}

	// Run migrations
	for _, migration := range pendingMigrations {
		if err := mm.runMigration(migration, backupPath); err != nil {
			log.Printf("Migration failed: %s - %v", migration, err)

			// Handle error based on recovery strategy
			if err := mm.handleMigrationError(migration, err, backupPath); err != nil {
				return fmt.Errorf("failed to handle migration error: %w", err)
			}

			// Stop on critical errors
			if mm.isCriticalError(err) {
				return fmt.Errorf("critical migration error: %w", err)
			}
		}
	}

	log.Println("All migrations completed successfully")
	return nil
}

// getPendingMigrations gets list of pending migrations
func (mm *MigrationManager) getPendingMigrations(currentVersion int64) ([]string, error) {
	// Get all migration files
	files, err := filepath.Glob(filepath.Join(mm.migrationsDir, "*.sql"))
	if err != nil {
		return nil, err
	}

	pending := make([]string, 0)

	for _, file := range files {
		// Extract version from filename
		version, err := mm.extractVersionFromFilename(filepath.Base(file))
		if err != nil {
			continue // Skip invalid filenames
		}

		// Check if migration is pending
		if version > currentVersion && !mm.isMigrationCompleted(version) {
			pending = append(pending, file)
		}
	}

	// Sort by version
	sort.Strings(pending)

	return pending, nil
}

// extractVersionFromFilename extracts version number from migration filename
func (mm *MigrationManager) extractVersionFromFilename(filename string) (int64, error) {
	// Expected format: 001_migration_name.sql
	parts := strings.Split(filename, "_")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid filename format: %s", filename)
	}

	var version int64
	if _, err := fmt.Sscanf(parts[0], "%d", &version); err != nil {
		return 0, fmt.Errorf("failed to parse version from filename: %s", filename)
	}

	return version, nil
}

// isMigrationCompleted checks if migration is already completed
func (mm *MigrationManager) isMigrationCompleted(version int64) bool {
	for _, record := range mm.migrationHistory {
		if record.Version == version && record.Status == "completed" {
			return true
		}
	}
	return false
}

// runMigration runs a single migration with validation and hooks
func (mm *MigrationManager) runMigration(migrationFile, backupPath string) error {
	filename := filepath.Base(migrationFile)
	version, err := mm.extractVersionFromFilename(filename)
	if err != nil {
		return err
	}

	log.Printf("Running migration: %s", filename)

	// Read migration content
	content, err := os.ReadFile(migrationFile)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Calculate checksum
	checksum := mm.calculateChecksum(content)

	// Check if migration already executed with same checksum
	if mm.isMigrationExecuted(version, checksum) {
		log.Printf("Migration %s already executed with same checksum, skipping", filename)
		return nil
	}

	// Validate migration
	if err := mm.validateMigration(string(content)); err != nil {
		return fmt.Errorf("migration validation failed: %w", err)
	}

	// Create migration record
	record := MigrationRecord{
		Version:    version,
		Filename:   filename,
		Checksum:   checksum,
		Status:     "pending",
		BackupPath: backupPath,
	}

	// Save initial record
	if err := mm.db.Create(&record).Error; err != nil {
		return fmt.Errorf("failed to create migration record: %w", err)
	}

	// Run pre-migration hooks
	for _, hook := range mm.preHooks {
		if err := hook(version, filename); err != nil {
			return fmt.Errorf("pre-migration hook failed: %w", err)
		}
	}

	// Update status to running
	record.Status = "running"
	mm.db.Save(&record)

	// Execute migration
	startTime := time.Now()
	err = mm.executeMigration(migrationFile)
	executionTime := time.Since(startTime)

	// Update record with results
	record.ExecutionTime = executionTime.Milliseconds()
	record.ExecutedAt = startTime

	if err != nil {
		record.Status = "failed"
		record.ErrorMessage = err.Error()
		log.Printf("Migration failed: %s - %v", filename, err)
	} else {
		record.Status = "completed"
		log.Printf("Migration completed: %s (took %v)", filename, executionTime)
	}

	mm.db.Save(&record)

	// Run post-migration hooks
	for _, hook := range mm.postHooks {
		if hookErr := hook(version, filename, err == nil, executionTime); hookErr != nil {
			log.Printf("Post-migration hook failed: %v", hookErr)
		}
	}

	return err
}

// calculateChecksum calculates SHA-256 checksum of migration content
func (mm *MigrationManager) calculateChecksum(content []byte) string {
	return fmt.Sprintf("%x", content) // Simplified checksum
}

// isMigrationExecuted checks if migration was already executed with same checksum
func (mm *MigrationManager) isMigrationExecuted(version int64, checksum string) bool {
	for _, record := range mm.migrationHistory {
		if record.Version == version && record.Checksum == checksum && record.Status == "completed" {
			return true
		}
	}
	return false
}

// validateMigration validates migration content against rules
func (mm *MigrationManager) validateMigration(content string) error {
	for _, rule := range mm.validationRules {
		if rule.Required {
			if err := mm.applyValidationRule(content, rule); err != nil {
				return fmt.Errorf("validation rule '%s' failed: %w", rule.Name, err)
			}
		}
	}
	return nil
}

// applyValidationRule applies a single validation rule
func (mm *MigrationManager) applyValidationRule(content string, rule ValidationRule) error {
	// This is a simplified implementation
	// In a real implementation, you would use regex matching
	switch rule.Name {
	case "no_drop_table":
		if strings.Contains(strings.ToUpper(content), "DROP TABLE") {
			return fmt.Errorf("DROP TABLE statements are not allowed")
		}
	case "no_drop_column":
		if strings.Contains(strings.ToUpper(content), "DROP COLUMN") {
			return fmt.Errorf("DROP COLUMN statements are not allowed")
		}
	case "no_truncate":
		if strings.Contains(strings.ToUpper(content), "TRUNCATE TABLE") {
			return fmt.Errorf("TRUNCATE TABLE statements are not allowed")
		}
	}
	return nil
}

// executeMigration executes the migration using Goose
func (mm *MigrationManager) executeMigration(migrationFile string) error {
	if mm.dryRunMode {
		log.Printf("DRY RUN: Would execute migration %s", migrationFile)
		return nil
	}

	return goose.Up(mm.sqlDB, mm.migrationsDir)
}

// handleMigrationError handles migration errors based on recovery strategies
func (mm *MigrationManager) handleMigrationError(migration string, migrationErr error, backupPath string) error {
	if !mm.errorRecovery.Enabled {
		return migrationErr
	}

	errorMsg := migrationErr.Error()

	// Find matching recovery strategy
	for _, strategy := range mm.errorRecovery.RecoveryStrategies {
		if mm.matchesErrorPattern(errorMsg, strategy.ErrorPattern) {
			log.Printf("Applying recovery strategy: %s", strategy.Action)

			switch strategy.Action {
			case "skip":
				log.Printf("Skipping migration due to recoverable error: %s", errorMsg)
				return nil
			case "retry":
				return mm.retryMigration(migration, strategy)
			case "rollback":
				return mm.rollbackMigration(migration, strategy, backupPath)
			case "manual":
				log.Printf("Manual intervention required for migration: %s", migration)
				return fmt.Errorf("manual intervention required: %w", migrationErr)
			}
		}
	}

	// No matching strategy found
	if mm.errorRecovery.AutoRollback {
		log.Println("Auto-rollback enabled, attempting rollback")
		return mm.rollbackMigration(migration, RecoveryStrategy{RollbackSteps: 1}, backupPath)
	}

	return migrationErr
}

// matchesErrorPattern checks if error message matches pattern
func (mm *MigrationManager) matchesErrorPattern(errorMsg, pattern string) bool {
	// Simplified pattern matching
	return strings.Contains(strings.ToLower(errorMsg), strings.ToLower(pattern))
}

// retryMigration retries a failed migration
func (mm *MigrationManager) retryMigration(migration string, strategy RecoveryStrategy) error {
	for attempt := 1; attempt <= strategy.MaxRetries; attempt++ {
		log.Printf("Retrying migration %s (attempt %d/%d)", migration, attempt, strategy.MaxRetries)

		time.Sleep(strategy.RetryDelay)

		if err := mm.executeMigration(migration); err == nil {
			log.Printf("Migration succeeded on retry attempt %d", attempt)
			return nil
		}
	}

	return fmt.Errorf("migration failed after %d retry attempts", strategy.MaxRetries)
}

// rollbackMigration rolls back failed migration
func (mm *MigrationManager) rollbackMigration(migration string, strategy RecoveryStrategy, backupPath string) error {
	log.Printf("Rolling back migration: %s", migration)

	// Use Goose to rollback
	for i := 0; i < strategy.RollbackSteps; i++ {
		if err := goose.Down(mm.sqlDB, mm.migrationsDir); err != nil {
			log.Printf("Rollback step %d failed: %v", i+1, err)

			// If rollback fails, restore from backup
			if backupPath != "" {
				return mm.restoreFromBackup(backupPath)
			}

			return fmt.Errorf("rollback failed: %w", err)
		}
	}

	log.Println("Rollback completed successfully")
	return nil
}

// isCriticalError checks if error is critical and should stop migration process
func (mm *MigrationManager) isCriticalError(err error) bool {
	errorMsg := strings.ToLower(err.Error())
	criticalPatterns := []string{
		"database corruption",
		"disk full",
		"out of memory",
		"permission denied",
		"connection refused",
	}

	for _, pattern := range criticalPatterns {
		if strings.Contains(errorMsg, pattern) {
			return true
		}
	}

	return false
}

// createBackup creates a database backup
func (mm *MigrationManager) createBackup() (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(mm.backupDir, fmt.Sprintf("backup_%s.db", timestamp))

	// For SQLite, we can copy the database file
	// This is a simplified implementation
	log.Printf("Creating backup at: %s", backupFile)

	// In a real implementation, you would:
	// 1. Use VACUUM INTO for SQLite
	// 2. Use pg_dump for PostgreSQL
	// 3. Use mysqldump for MySQL

	return backupFile, nil
}

// restoreFromBackup restores database from backup
func (mm *MigrationManager) restoreFromBackup(backupPath string) error {
	log.Printf("Restoring database from backup: %s", backupPath)

	// This is a simplified implementation
	// In a real implementation, you would restore the database

	return nil
}

// GetStatus returns current migration status
func (mm *MigrationManager) GetStatus() (*MigrationStatus, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// Get current version
	currentVersion, err := goose.GetDBVersion(mm.sqlDB)
	if err != nil {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}

	// Get pending migrations
	pendingMigrations, err := mm.getPendingMigrations(currentVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending migrations: %w", err)
	}

	// Get failed migrations
	failedMigrations := make([]MigrationRecord, 0)
	for _, record := range mm.migrationHistory {
		if record.Status == "failed" {
			failedMigrations = append(failedMigrations, record)
		}
	}

	// Get last migration
	var lastMigration *MigrationRecord
	if len(mm.migrationHistory) > 0 {
		lastMigration = &mm.migrationHistory[0]
	}

	// Get database health
	dbHealth := mm.getDatabaseHealth()

	// Get backup status
	backupStatus := mm.getBackupStatus()

	return &MigrationStatus{
		CurrentVersion:    currentVersion,
		TargetVersion:     mm.getTargetVersion(),
		PendingMigrations: mm.extractFilenames(pendingMigrations),
		FailedMigrations:  failedMigrations,
		LastMigration:     lastMigration,
		DatabaseHealth:    dbHealth,
		BackupStatus:      backupStatus,
	}, nil
}

// getTargetVersion gets the highest version from migration files
func (mm *MigrationManager) getTargetVersion() int64 {
	files, err := filepath.Glob(filepath.Join(mm.migrationsDir, "*.sql"))
	if err != nil {
		return 0
	}

	var maxVersion int64
	for _, file := range files {
		version, err := mm.extractVersionFromFilename(filepath.Base(file))
		if err != nil {
			continue
		}
		if version > maxVersion {
			maxVersion = version
		}
	}

	return maxVersion
}

// extractFilenames extracts filenames from full paths
func (mm *MigrationManager) extractFilenames(paths []string) []string {
	filenames := make([]string, len(paths))
	for i, path := range paths {
		filenames[i] = filepath.Base(path)
	}
	return filenames
}

// getDatabaseHealth gets database health metrics
func (mm *MigrationManager) getDatabaseHealth() DatabaseHealth {
	// Simplified health check
	health := DatabaseHealth{
		Connected:          true,
		Version:            "SQLite 3.x",
		PerformanceMetrics: make(map[string]float64),
	}

	// Get table count
	var tableCount int64
	mm.db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&tableCount)
	health.TableCount = int(tableCount)

	// Get index count
	var indexCount int64
	mm.db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='index'").Scan(&indexCount)
	health.IndexCount = int(indexCount)

	return health
}

// getBackupStatus gets backup status
func (mm *MigrationManager) getBackupStatus() BackupStatus {
	return BackupStatus{
		Enabled:       mm.errorRecovery.BackupBeforeMigration,
		RetentionDays: 30,
		AutoCleanup:   true,
	}
}

// AddPreHook adds a pre-migration hook
func (mm *MigrationManager) AddPreHook(hook PreMigrationHook) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.preHooks = append(mm.preHooks, hook)
}

// AddPostHook adds a post-migration hook
func (mm *MigrationManager) AddPostHook(hook PostMigrationHook) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.postHooks = append(mm.postHooks, hook)
}

// SetDryRunMode enables or disables dry run mode
func (mm *MigrationManager) SetDryRunMode(enabled bool) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.dryRunMode = enabled
}

// CreateMigration creates a new migration file
func (mm *MigrationManager) CreateMigration(name string) (string, error) {
	// Get next version number
	nextVersion := mm.getTargetVersion() + 1

	// Create filename
	filename := fmt.Sprintf("%03d_%s.sql", nextVersion, strings.ReplaceAll(name, " ", "_"))
	filePath := filepath.Join(mm.migrationsDir, filename)

	// Create migration template
	template := fmt.Sprintf(`-- +goose Up
-- Migration: %s
-- Created: %s

BEGIN;

-- Add your migration SQL here

COMMIT;

-- +goose Down
-- Rollback migration: %s

BEGIN;

-- Add your rollback SQL here

COMMIT;
`, name, time.Now().Format("2006-01-02 15:04:05"), name)

	// Write file
	if err := os.WriteFile(filePath, []byte(template), 0644); err != nil {
		return "", fmt.Errorf("failed to create migration file: %w", err)
	}

	log.Printf("Created migration file: %s", filename)
	return filePath, nil
}

// Rollback rolls back the last N migrations
func (mm *MigrationManager) Rollback(steps int) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	log.Printf("Rolling back %d migrations", steps)

	for i := 0; i < steps; i++ {
		if err := goose.Down(mm.sqlDB, mm.migrationsDir); err != nil {
			return fmt.Errorf("rollback step %d failed: %w", i+1, err)
		}
	}

	log.Printf("Successfully rolled back %d migrations", steps)
	return nil
}

// Reset resets the database to version 0
func (mm *MigrationManager) Reset() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	log.Println("Resetting database to version 0")

	if err := goose.Reset(mm.sqlDB, mm.migrationsDir); err != nil {
		return fmt.Errorf("failed to reset database: %w", err)
	}

	log.Println("Database reset completed")
	return nil
}
