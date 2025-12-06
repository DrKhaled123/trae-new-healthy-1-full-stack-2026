package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SQLiteConfig holds SQLite-specific configuration
type SQLiteConfig struct {
	DatabasePath      string        `json:"database_path"`
	WALMode           bool          `json:"wal_mode"`
	FTS5Enabled       bool          `json:"fts5_enabled"`
	JournalMode       string        `json:"journal_mode"`
	Synchronous       string        `json:"synchronous"`
	CacheSize         int           `json:"cache_size"`
	BusyTimeout       time.Duration `json:"busy_timeout"`
	MaxOpenConns      int           `json:"max_open_conns"`
	MaxIdleConns      int           `json:"max_idle_conns"`
	ConnMaxLifetime   time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime   time.Duration `json:"conn_max_idle_time"`
	ForeignKeys       bool          `json:"foreign_keys"`
	AutoVacuum        string        `json:"auto_vacuum"`
	TempStore         string        `json:"temp_store"`
	MmapSize          int64         `json:"mmap_size"`
	PageSize          int           `json:"page_size"`
	SecureDelete      bool          `json:"secure_delete"`
	RecursiveTriggers bool          `json:"recursive_triggers"`
	QueryOnly         bool          `json:"query_only"`
	BackupEnabled     bool          `json:"backup_enabled"`
	BackupInterval    time.Duration `json:"backup_interval"`
	BackupRetention   int           `json:"backup_retention_days"`
	EncryptionKey     string        `json:"encryption_key,omitempty"`
	CompressionLevel  int           `json:"compression_level"`
}

// SQLiteManager manages SQLite database connections and operations
type SQLiteManager struct {
	config    *SQLiteConfig
	db        *gorm.DB
	sqlDB     *sql.DB
	dbPath    string
	backupDir string
	metrics   *SQLiteMetrics
}

// SQLiteMetrics holds database performance metrics
type SQLiteMetrics struct {
	Connections      int64         `json:"connections"`
	Queries          int64         `json:"queries"`
	SlowQueries      int64         `json:"slow_queries"`
	CacheHits        int64         `json:"cache_hits"`
	CacheMisses      int64         `json:"cache_misses"`
	DatabaseSize     int64         `json:"database_size_bytes"`
	WALSize          int64         `json:"wal_size_bytes"`
	PageCount        int64         `json:"page_count"`
	FreePages        int64         `json:"free_pages"`
	Fragmentation    float64       `json:"fragmentation_pct"`
	LastVacuum       time.Time     `json:"last_vacuum"`
	LastBackup       time.Time     `json:"last_backup"`
	Uptime           time.Duration `json:"uptime"`
	AverageQueryTime float64       `json:"avg_query_time_ms"`
}

// FTSConfig holds Full-Text Search configuration
type FTSConfig struct {
	Enabled        bool                `json:"enabled"`
	Tables         []string            `json:"tables"`
	Tokenizer      string              `json:"tokenizer"`
	ContentTable   string              `json:"content_table"`
	ContentRowID   string              `json:"content_rowid"`
	Columns        []string            `json:"columns"`
	RankFunction   string              `json:"rank_function"`
	MinTokenLength int                 `json:"min_token_length"`
	MaxTokenLength int                 `json:"max_token_length"`
	StopWords      []string            `json:"stop_words"`
	Synonyms       map[string][]string `json:"synonyms"`
}

// ValidationResult holds database validation results
type ValidationResult struct {
	Valid           bool              `json:"valid"`
	WALModeEnabled  bool              `json:"wal_mode_enabled"`
	FTS5Available   bool              `json:"fts5_available"`
	ForeignKeys     bool              `json:"foreign_keys_enabled"`
	JournalMode     string            `json:"journal_mode"`
	Synchronous     string            `json:"synchronous"`
	CacheSize       int               `json:"cache_size"`
	PageSize        int               `json:"page_size"`
	AutoVacuum      string            `json:"auto_vacuum"`
	Encoding        string            `json:"encoding"`
	Version         string            `json:"sqlite_version"`
	Extensions      []string          `json:"loaded_extensions"`
	Pragmas         map[string]string `json:"pragma_values"`
	IntegrityCheck  []string          `json:"integrity_check"`
	QuickCheck      []string          `json:"quick_check"`
	Errors          []string          `json:"errors"`
	Warnings        []string          `json:"warnings"`
	Recommendations []string          `json:"recommendations"`
}

// NewSQLiteConfig creates a new SQLite configuration with defaults
func NewSQLiteConfig() *SQLiteConfig {
	return &SQLiteConfig{
		DatabasePath:      "./data/nutrition.db",
		WALMode:           true,
		FTS5Enabled:       true,
		JournalMode:       "WAL",
		Synchronous:       "NORMAL",
		CacheSize:         -64000, // 64MB cache
		BusyTimeout:       30 * time.Second,
		MaxOpenConns:      25,
		MaxIdleConns:      5,
		ConnMaxLifetime:   5 * time.Minute,
		ConnMaxIdleTime:   1 * time.Minute,
		ForeignKeys:       true,
		AutoVacuum:        "INCREMENTAL",
		TempStore:         "MEMORY",
		MmapSize:          268435456, // 256MB
		PageSize:          4096,
		SecureDelete:      true,
		RecursiveTriggers: true,
		QueryOnly:         false,
		BackupEnabled:     true,
		BackupInterval:    6 * time.Hour,
		BackupRetention:   7, // 7 days
		CompressionLevel:  6,
	}
}

// NewSQLiteManager creates a new SQLite manager
func NewSQLiteManager(config *SQLiteConfig) (*SQLiteManager, error) {
	if config == nil {
		config = NewSQLiteConfig()
	}

	// Ensure database directory exists
	dbDir := filepath.Dir(config.DatabasePath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Create backup directory
	backupDir := filepath.Join(dbDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	manager := &SQLiteManager{
		config:    config,
		dbPath:    config.DatabasePath,
		backupDir: backupDir,
		metrics:   &SQLiteMetrics{},
	}

	// Initialize database connection
	if err := manager.initializeDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return manager, nil
}

// initializeDatabase initializes the database connection with optimized settings
func (sm *SQLiteManager) initializeDatabase() error {
	// Build DSN with pragmas
	dsn := sm.buildDSN()

	// Configure GORM logger
	logLevel := logger.Info
	if os.Getenv("ENV") == "production" {
		logLevel = logger.Warn
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: false,
	}

	// Open database connection
	db, err := gorm.Open(sqlite.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	sm.db = db

	// Get underlying SQL DB
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}

	sm.sqlDB = sqlDB

	// Configure connection pool
	sqlDB.SetMaxOpenConns(sm.config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(sm.config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(sm.config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(sm.config.ConnMaxIdleTime)

	// Apply SQLite-specific pragmas
	if err := sm.applyPragmas(); err != nil {
		return fmt.Errorf("failed to apply pragmas: %w", err)
	}

	// Initialize FTS5 if enabled
	if sm.config.FTS5Enabled {
		if err := sm.initializeFTS5(); err != nil {
			log.Printf("Warning: Failed to initialize FTS5: %v", err)
			// Don't fail completely, just log the warning
		}
	}

	// Start background tasks
	go sm.startBackgroundTasks()

	return nil
}

// buildDSN builds the SQLite DSN with pragmas
func (sm *SQLiteManager) buildDSN() string {
	pragmas := []string{
		fmt.Sprintf("_journal_mode=%s", sm.config.JournalMode),
		fmt.Sprintf("_synchronous=%s", sm.config.Synchronous),
		fmt.Sprintf("_cache_size=%d", sm.config.CacheSize),
		fmt.Sprintf("_busy_timeout=%d", int(sm.config.BusyTimeout.Milliseconds())),
		fmt.Sprintf("_foreign_keys=%s", boolToString(sm.config.ForeignKeys)),
		fmt.Sprintf("_auto_vacuum=%s", sm.config.AutoVacuum),
		fmt.Sprintf("_temp_store=%s", sm.config.TempStore),
		fmt.Sprintf("_mmap_size=%d", sm.config.MmapSize),
		fmt.Sprintf("_page_size=%d", sm.config.PageSize),
		fmt.Sprintf("_secure_delete=%s", boolToString(sm.config.SecureDelete)),
		fmt.Sprintf("_recursive_triggers=%s", boolToString(sm.config.RecursiveTriggers)),
	}

	if sm.config.QueryOnly {
		pragmas = append(pragmas, "_query_only=true")
	}

	return fmt.Sprintf("%s?%s", sm.config.DatabasePath, strings.Join(pragmas, "&"))
}

// applyPragmas applies additional SQLite pragmas
func (sm *SQLiteManager) applyPragmas() error {
	pragmas := map[string]interface{}{
		"journal_mode":              sm.config.JournalMode,
		"synchronous":               sm.config.Synchronous,
		"cache_size":                sm.config.CacheSize,
		"foreign_keys":              boolToString(sm.config.ForeignKeys),
		"auto_vacuum":               sm.config.AutoVacuum,
		"temp_store":                sm.config.TempStore,
		"mmap_size":                 sm.config.MmapSize,
		"page_size":                 sm.config.PageSize,
		"secure_delete":             boolToString(sm.config.SecureDelete),
		"recursive_triggers":        boolToString(sm.config.RecursiveTriggers),
		"encoding":                  "UTF-8",
		"case_sensitive_like":       "false",
		"count_changes":             "false",
		"empty_result_callbacks":    "false",
		"legacy_alter_table":        "false",
		"reverse_unordered_selects": "false",
	}

	for pragma, value := range pragmas {
		query := fmt.Sprintf("PRAGMA %s = %v", pragma, value)
		if err := sm.db.Exec(query).Error; err != nil {
			log.Printf("Warning: Failed to set pragma %s: %v", pragma, err)
			// Continue with other pragmas
		}
	}

	return nil
}

// initializeFTS5 initializes Full-Text Search capabilities
func (sm *SQLiteManager) initializeFTS5() error {
	// Check if FTS5 is available
	var fts5Available bool
	err := sm.db.Raw("SELECT 1 FROM pragma_compile_options WHERE compile_options LIKE '%FTS5%'").Scan(&fts5Available).Error
	if err != nil || !fts5Available {
		return fmt.Errorf("FTS5 is not available in this SQLite build")
	}

	// Create FTS5 tables for nutrition data
	ftsQueries := []string{
		// Foods FTS table
		`CREATE VIRTUAL TABLE IF NOT EXISTS foods_fts USING fts5(
			name, description, brand, category, ingredients,
			content='foods', content_rowid='id',
			tokenize='porter ascii'
		)`,

		// Recipes FTS table
		`CREATE VIRTUAL TABLE IF NOT EXISTS recipes_fts USING fts5(
			title, description, instructions, tags, ingredients,
			content='recipes', content_rowid='id',
			tokenize='porter ascii'
		)`,

		// Meal plans FTS table
		`CREATE VIRTUAL TABLE IF NOT EXISTS meal_plans_fts USING fts5(
			name, description, goals, notes,
			content='meal_plans', content_rowid='id',
			tokenize='porter ascii'
		)`,

		// Create triggers to keep FTS tables in sync
		`CREATE TRIGGER IF NOT EXISTS foods_fts_insert AFTER INSERT ON foods BEGIN
			INSERT INTO foods_fts(rowid, name, description, brand, category, ingredients)
			VALUES (new.id, new.name, new.description, new.brand, new.category, new.ingredients);
		END`,

		`CREATE TRIGGER IF NOT EXISTS foods_fts_update AFTER UPDATE ON foods BEGIN
			UPDATE foods_fts SET
				name = new.name,
				description = new.description,
				brand = new.brand,
				category = new.category,
				ingredients = new.ingredients
			WHERE rowid = new.id;
		END`,

		`CREATE TRIGGER IF NOT EXISTS foods_fts_delete AFTER DELETE ON foods BEGIN
			DELETE FROM foods_fts WHERE rowid = old.id;
		END`,
	}

	for _, query := range ftsQueries {
		if err := sm.db.Exec(query).Error; err != nil {
			log.Printf("Warning: Failed to execute FTS query: %v", err)
			// Continue with other queries
		}
	}

	log.Println("FTS5 initialized successfully")
	return nil
}

// Validate validates the database configuration and health
func (sm *SQLiteManager) Validate() (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:           true,
		Pragmas:         make(map[string]string),
		Extensions:      make([]string, 0),
		IntegrityCheck:  make([]string, 0),
		QuickCheck:      make([]string, 0),
		Errors:          make([]string, 0),
		Warnings:        make([]string, 0),
		Recommendations: make([]string, 0),
	}

	// Check SQLite version
	var version string
	if err := sm.db.Raw("SELECT sqlite_version()").Scan(&version).Error; err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to get SQLite version: %v", err))
		result.Valid = false
	} else {
		result.Version = version
	}

	// Check journal mode
	var journalMode string
	if err := sm.db.Raw("PRAGMA journal_mode").Scan(&journalMode).Error; err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to get journal mode: %v", err))
	} else {
		result.JournalMode = journalMode
		result.WALModeEnabled = strings.ToUpper(journalMode) == "WAL"
		result.Pragmas["journal_mode"] = journalMode

		if sm.config.WALMode && !result.WALModeEnabled {
			result.Errors = append(result.Errors, "WAL mode is not enabled")
			result.Valid = false
		}
	}

	// Check FTS5 availability
	var fts5Count int
	if err := sm.db.Raw("SELECT COUNT(*) FROM pragma_compile_options WHERE compile_options LIKE '%FTS5%'").Scan(&fts5Count).Error; err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to check FTS5 availability: %v", err))
	} else {
		result.FTS5Available = fts5Count > 0

		if sm.config.FTS5Enabled && !result.FTS5Available {
			result.Errors = append(result.Errors, "FTS5 is not available in this SQLite build")
			result.Valid = false
		}
	}

	// Check foreign keys
	var foreignKeys string
	if err := sm.db.Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error; err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to check foreign keys: %v", err))
	} else {
		result.ForeignKeys = foreignKeys == "1"
		result.Pragmas["foreign_keys"] = foreignKeys
	}

	// Check other pragmas
	pragmasToCheck := []string{
		"synchronous", "cache_size", "page_size", "auto_vacuum",
		"temp_store", "secure_delete", "recursive_triggers", "encoding",
	}

	for _, pragma := range pragmasToCheck {
		var value string
		if err := sm.db.Raw(fmt.Sprintf("PRAGMA %s", pragma)).Scan(&value).Error; err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to check pragma %s: %v", pragma, err))
		} else {
			result.Pragmas[pragma] = value

			// Set specific fields
			switch pragma {
			case "synchronous":
				result.Synchronous = value
			case "cache_size":
				var cacheSize int
				fmt.Sscanf(value, "%d", &cacheSize)
				result.CacheSize = cacheSize
			case "page_size":
				var pageSize int
				fmt.Sscanf(value, "%d", &pageSize)
				result.PageSize = pageSize
			case "auto_vacuum":
				result.AutoVacuum = value
			case "encoding":
				result.Encoding = value
			}
		}
	}

	// Run integrity check
	var integrityResults []string
	rows, err := sm.db.Raw("PRAGMA integrity_check(10)").Rows()
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to run integrity check: %v", err))
	} else {
		defer rows.Close()
		for rows.Next() {
			var checkResult string
			rows.Scan(&checkResult)
			integrityResults = append(integrityResults, checkResult)

			if checkResult != "ok" {
				result.Errors = append(result.Errors, fmt.Sprintf("Integrity check failed: %s", checkResult))
				result.Valid = false
			}
		}
		result.IntegrityCheck = integrityResults
	}

	// Run quick check
	var quickResults []string
	rows, err = sm.db.Raw("PRAGMA quick_check(10)").Rows()
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to run quick check: %v", err))
	} else {
		defer rows.Close()
		for rows.Next() {
			var checkResult string
			rows.Scan(&checkResult)
			quickResults = append(quickResults, checkResult)

			if checkResult != "ok" {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Quick check warning: %s", checkResult))
			}
		}
		result.QuickCheck = quickResults
	}

	// Add recommendations
	if result.CacheSize < 32000 {
		result.Recommendations = append(result.Recommendations, "Consider increasing cache_size for better performance")
	}

	if result.PageSize < 4096 {
		result.Recommendations = append(result.Recommendations, "Consider using page_size of 4096 or higher")
	}

	if !result.WALModeEnabled {
		result.Recommendations = append(result.Recommendations, "Enable WAL mode for better concurrency")
	}

	if !result.FTS5Available && sm.config.FTS5Enabled {
		result.Recommendations = append(result.Recommendations, "Compile SQLite with FTS5 support for full-text search")
	}

	return result, nil
}

// GetMetrics returns current database metrics
func (sm *SQLiteManager) GetMetrics() (*SQLiteMetrics, error) {
	// Update metrics
	sm.updateMetrics()

	return sm.metrics, nil
}

// updateMetrics updates database metrics
func (sm *SQLiteManager) updateMetrics() {
	// Get database size
	if info, err := os.Stat(sm.dbPath); err == nil {
		sm.metrics.DatabaseSize = info.Size()
	}

	// Get WAL size
	walPath := sm.dbPath + "-wal"
	if info, err := os.Stat(walPath); err == nil {
		sm.metrics.WALSize = info.Size()
	}

	// Get page count
	var pageCount int64
	if err := sm.db.Raw("PRAGMA page_count").Scan(&pageCount).Error; err == nil {
		sm.metrics.PageCount = pageCount
	}

	// Get free pages
	var freePages int64
	if err := sm.db.Raw("PRAGMA freelist_count").Scan(&freePages).Error; err == nil {
		sm.metrics.FreePages = freePages
	}

	// Calculate fragmentation
	if sm.metrics.PageCount > 0 {
		sm.metrics.Fragmentation = float64(sm.metrics.FreePages) / float64(sm.metrics.PageCount) * 100
	}

	// Get connection stats
	stats := sm.sqlDB.Stats()
	sm.metrics.Connections = int64(stats.OpenConnections)
}

// startBackgroundTasks starts background maintenance tasks
func (sm *SQLiteManager) startBackgroundTasks() {
	// Auto-vacuum task
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			if sm.config.AutoVacuum == "INCREMENTAL" {
				if err := sm.db.Exec("PRAGMA incremental_vacuum(1000)").Error; err != nil {
					log.Printf("Auto-vacuum failed: %v", err)
				}
			}
		}
	}()

	// Backup task
	if sm.config.BackupEnabled {
		go func() {
			ticker := time.NewTicker(sm.config.BackupInterval)
			defer ticker.Stop()

			for range ticker.C {
				if err := sm.CreateBackup(); err != nil {
					log.Printf("Backup failed: %v", err)
				}
			}
		}()
	}

	// Metrics update task
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			sm.updateMetrics()
		}
	}()
}

// CreateBackup creates a database backup
func (sm *SQLiteManager) CreateBackup() error {
	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(sm.backupDir, fmt.Sprintf("nutrition_%s.db", timestamp))

	// Use VACUUM INTO for atomic backup
	query := fmt.Sprintf("VACUUM INTO '%s'", backupPath)
	if err := sm.db.Exec(query).Error; err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	sm.metrics.LastBackup = time.Now()
	log.Printf("Database backup created: %s", backupPath)

	// Cleanup old backups
	go sm.cleanupOldBackups()

	return nil
}

// cleanupOldBackups removes old backup files
func (sm *SQLiteManager) cleanupOldBackups() {
	files, err := filepath.Glob(filepath.Join(sm.backupDir, "nutrition_*.db"))
	if err != nil {
		log.Printf("Failed to list backup files: %v", err)
		return
	}

	cutoff := time.Now().AddDate(0, 0, -sm.config.BackupRetention)

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(file); err != nil {
				log.Printf("Failed to remove old backup %s: %v", file, err)
			} else {
				log.Printf("Removed old backup: %s", file)
			}
		}
	}
}

// Optimize performs database optimization
func (sm *SQLiteManager) Optimize() error {
	log.Println("Starting database optimization...")

	// Analyze tables for query optimization
	if err := sm.db.Exec("ANALYZE").Error; err != nil {
		log.Printf("ANALYZE failed: %v", err)
	}

	// Reindex all indexes
	if err := sm.db.Exec("REINDEX").Error; err != nil {
		log.Printf("REINDEX failed: %v", err)
	}

	// Incremental vacuum
	if sm.config.AutoVacuum == "INCREMENTAL" {
		if err := sm.db.Exec("PRAGMA incremental_vacuum").Error; err != nil {
			log.Printf("Incremental vacuum failed: %v", err)
		}
	}

	// Optimize FTS tables if enabled
	if sm.config.FTS5Enabled {
		ftsTables := []string{"foods_fts", "recipes_fts", "meal_plans_fts"}
		for _, table := range ftsTables {
			query := fmt.Sprintf("INSERT INTO %s(%s) VALUES('optimize')", table, table)
			if err := sm.db.Exec(query).Error; err != nil {
				log.Printf("FTS optimize failed for %s: %v", table, err)
			}
		}
	}

	log.Println("Database optimization completed")
	return nil
}

// Close closes the database connection
func (sm *SQLiteManager) Close() error {
	if sm.sqlDB != nil {
		return sm.sqlDB.Close()
	}
	return nil
}

// GetDB returns the GORM database instance
func (sm *SQLiteManager) GetDB() *gorm.DB {
	return sm.db
}

// GetSQLDB returns the underlying SQL database instance
func (sm *SQLiteManager) GetSQLDB() *sql.DB {
	return sm.sqlDB
}

// SearchFTS performs full-text search across FTS tables
func (sm *SQLiteManager) SearchFTS(query string, tables []string, limit int) (map[string][]map[string]interface{}, error) {
	if !sm.config.FTS5Enabled {
		return nil, fmt.Errorf("FTS5 is not enabled")
	}

	results := make(map[string][]map[string]interface{})

	for _, table := range tables {
		ftsTable := table + "_fts"

		// Prepare FTS query
		ftsQuery := fmt.Sprintf("SELECT rowid, rank FROM %s WHERE %s MATCH ? ORDER BY rank LIMIT ?", ftsTable, ftsTable)

		rows, err := sm.db.Raw(ftsQuery, query, limit).Rows()
		if err != nil {
			continue // Skip tables that don't exist or have errors
		}
		defer rows.Close()

		tableResults := make([]map[string]interface{}, 0)

		for rows.Next() {
			var rowid int64
			var rank float64
			rows.Scan(&rowid, &rank)

			// Get full record from original table
			var record map[string]interface{}
			if err := sm.db.Table(table).Where("id = ?", rowid).Take(&record).Error; err == nil {
				record["fts_rank"] = rank
				tableResults = append(tableResults, record)
			}
		}

		if len(tableResults) > 0 {
			results[table] = tableResults
		}
	}

	return results, nil
}

// boolToString converts boolean to SQLite string format
func boolToString(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}

// GetConfig returns the current SQLite configuration
func (sm *SQLiteManager) GetConfig() *SQLiteConfig {
	return sm.config
}

// UpdateConfig updates the SQLite configuration
func (sm *SQLiteManager) UpdateConfig(config *SQLiteConfig) error {
	sm.config = config

	// Reapply pragmas that can be changed at runtime
	runtimePragmas := map[string]interface{}{
		"cache_size":    config.CacheSize,
		"synchronous":   config.Synchronous,
		"temp_store":    config.TempStore,
		"secure_delete": boolToString(config.SecureDelete),
	}

	for pragma, value := range runtimePragmas {
		query := fmt.Sprintf("PRAGMA %s = %v", pragma, value)
		if err := sm.db.Exec(query).Error; err != nil {
			log.Printf("Warning: Failed to update pragma %s: %v", pragma, err)
		}
	}

	return nil
}
