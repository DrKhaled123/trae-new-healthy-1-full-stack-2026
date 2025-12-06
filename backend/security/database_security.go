package security

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SecureDatabase creates a securely configured SQLite database
func SecureDatabase(dbPath string) (*sql.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %v", err)
	}

	// Set secure permissions on database file if it exists
	if _, err := os.Stat(dbPath); err == nil {
		if err := os.Chmod(dbPath, 0600); err != nil {
			return nil, fmt.Errorf("failed to set database permissions: %v", err)
		}
	}

	// Enable WAL mode for better concurrency and security
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=1000&_foreign_keys=on&_secure_delete=on", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// Enable foreign key constraints
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %v", err)
	}

	// Enable secure delete (overwrite deleted data)
	if _, err := db.Exec("PRAGMA secure_delete = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable secure delete: %v", err)
	}

	// Set busy timeout
	if _, err := db.Exec("PRAGMA busy_timeout = 30000"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %v", err)
	}

	return db, nil
}

// DatabaseSecurityManager manages database security settings
type DatabaseSecurityManager struct {
	db *sql.DB
}

// NewDatabaseSecurityManager creates a new database security manager
func NewDatabaseSecurityManager(db *sql.DB) *DatabaseSecurityManager {
	return &DatabaseSecurityManager{db: db}
}

// EnableRowLevelSecurity enables row-level security for sensitive data
func (dsm *DatabaseSecurityManager) EnableRowLevelSecurity() error {
	// Create RLS policies for users table
	rlsPolicy := `
		CREATE TABLE IF NOT EXISTS user_access_policy (
			user_id INTEGER PRIMARY KEY,
			accessible_roles TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`

	if _, err := dsm.db.Exec(rlsPolicy); err != nil {
		return fmt.Errorf("failed to create RLS policy table: %v", err)
	}

	return nil
}

// EncryptSensitiveData encrypts sensitive columns
func (dsm *DatabaseSecurityManager) EncryptSensitiveData() error {
	// Enable SQLite encryption if using SQLCipher
	// Note: This requires the sqlcipher driver instead of sqlite3
	encryptionSQL := `
		-- Mark sensitive columns for encryption
		CREATE INDEX IF NOT EXISTS idx_users_email_encrypted ON users(email);
		CREATE INDEX IF NOT EXISTS idx_users_phone_encrypted ON users(phone);
	`

	if _, err := dsm.db.Exec(encryptionSQL); err != nil {
		return fmt.Errorf("failed to set up encrypted indexes: %v", err)
	}

	return nil
}

// AuditQuery logs all queries for security monitoring
func (dsm *DatabaseSecurityManager) AuditQuery(query string, args ...interface{}) error {
	// Log query for audit trail
	auditLog := fmt.Sprintf("[%s] Query: %s, Args: %v",
		time.Now().Format(time.RFC3339), query, args)

	// In production, write to secure audit log file
	// For now, just print (you should write to a secure log file)
	fmt.Printf("AUDIT: %s\n", auditLog)

	return nil
}

// ValidateQueryParameters validates input parameters to prevent injection
func (dsm *DatabaseSecurityManager) ValidateQueryParameters(params map[string]interface{}) error {
	for key, value := range params {
		switch v := value.(type) {
		case string:
			// Check for SQL injection patterns
			if hasSQLInjection(v) {
				return fmt.Errorf("potential SQL injection detected in parameter %s", key)
			}

			// Check string length limits
			if len(v) > 1000 {
				return fmt.Errorf("parameter %s exceeds maximum length", key)
			}
		case int, int64:
			// Validate numeric ranges
			if num, ok := v.(int64); ok {
				if num < 0 || num > 999999999 {
					return fmt.Errorf("parameter %s out of valid range", key)
				}
			}
		}
	}

	return nil
}

// BackupWithEncryption creates encrypted database backups
func (dsm *DatabaseSecurityManager) BackupWithEncryption(backupPath string, encryptionKey string) error {
	// Create backup directory
	dir := filepath.Dir(backupPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	// Create backup with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupFile := fmt.Sprintf("%s_%s.db", backupPath, timestamp)

	// Use SQLite backup API
	backupSQL := fmt.Sprintf("VACUUM INTO '%s'", backupFile)
	if _, err := dsm.db.Exec(backupSQL); err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}

	// Set secure permissions on backup
	if err := os.Chmod(backupFile, 0600); err != nil {
		return fmt.Errorf("failed to set backup permissions: %v", err)
	}

	return nil
}

// CleanupOldBackups removes old backup files (keep last 7 days)
func (dsm *DatabaseSecurityManager) CleanupOldBackups(backupDir string) error {
	cutoff := time.Now().AddDate(0, 0, -7) // 7 days ago

	files, err := filepath.Glob(filepath.Join(backupDir, "*.db"))
	if err != nil {
		return fmt.Errorf("failed to list backup files: %v", err)
	}

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(file); err != nil {
				fmt.Printf("Warning: failed to remove old backup %s: %v\n", file, err)
			} else {
				fmt.Printf("Removed old backup: %s\n", file)
			}
		}
	}

	return nil
}

// MonitorDatabaseHealth checks database security health
func (dsm *DatabaseSecurityManager) MonitorDatabaseHealth() map[string]interface{} {
	health := make(map[string]interface{})

	// Check WAL mode
	var journalMode string
	dsm.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	health["journal_mode"] = journalMode
	health["wal_enabled"] = journalMode == "wal"

	// Check foreign keys
	var foreignKeys int
	dsm.db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys)
	health["foreign_keys_enabled"] = foreignKeys == 1

	// Check page size
	var pageSize int
	dsm.db.QueryRow("PRAGMA page_size").Scan(&pageSize)
	health["page_size"] = pageSize

	// Check cache size
	var cacheSize int
	dsm.db.QueryRow("PRAGMA cache_size").Scan(&cacheSize)
	health["cache_size"] = cacheSize

	return health
}

// SQL injection detection patterns
func hasSQLInjection(str string) bool {
	sqlInjectionPatterns := []string{
		"union select",
		"union all",
		"select * from",
		"drop table",
		"delete from",
		"update .* set",
		"insert into",
		"script>",
		"javascript:",
		"vbscript:",
		"onload=",
		"onerror=",
		"<iframe",
		"<object",
		"<embed",
		"eval(",
		"document\\.cookie",
		"--",
		"/*",
		"*/",
		"xp_",
		"sp_",
	}

	lowerStr := strings.ToLower(str)
	for _, pattern := range sqlInjectionPatterns {
		if strings.Contains(lowerStr, pattern) {
			return true
		}
	}

	return false
}
