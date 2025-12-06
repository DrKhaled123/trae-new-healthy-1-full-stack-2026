package tests

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"nutrition-platform/config"
	"nutrition-platform/models"

	_ "github.com/mattn/go-sqlite3"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	// Setup test database
	setupTestDB()

	// Run tests
	code := m.Run()

	// Cleanup
	teardownTestDB()

	os.Exit(code)
}

func setupTestDB() {
	var err error
	testDB, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("Failed to create test database: %v", err)
	}

	// Set the global DB for testing
	models.DB = testDB

	// Create test tables
	createTestTables()
}

func teardownTestDB() {
	if testDB != nil {
		testDB.Close()
	}
}

func createTestTables() {
	queries := []string{
		`CREATE TABLE api_keys (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			key_hash TEXT UNIQUE NOT NULL,
			prefix TEXT NOT NULL,
			user_id TEXT,
			status TEXT DEFAULT 'active',
			scopes TEXT NOT NULL,
			rate_limit INTEGER DEFAULT 100,
			expires_at DATETIME,
			last_used_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			metadata TEXT DEFAULT '{}'
		)`,
		`CREATE TABLE api_key_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api_key_id TEXT REFERENCES api_keys(id) ON DELETE CASCADE,
			endpoint TEXT NOT NULL,
			method TEXT NOT NULL,
			status_code INTEGER NOT NULL,
			response_time INTEGER NOT NULL,
			ip_address TEXT,
			user_agent TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		if _, err := testDB.Exec(query); err != nil {
			log.Fatalf("Failed to create test table: %v", err)
		}
	}
}

func getTestConfig() *config.Config {
	return &config.Config{
		Environment:       "test",
		ServerPort:        8080,
		ServerHost:        "localhost",
		Debug:             true,
		RateLimitRequests: 100,
		SecurityHeaders:   true,
		DataPath:          "./testdata",
		NutritionDataPath: "./testdata",
	}
}

func resetTestDB() {
	queries := []string{
		"DELETE FROM api_key_usage",
		"DELETE FROM api_keys",
		"DELETE FROM users",
	}

	for _, query := range queries {
		if _, err := testDB.Exec(query); err != nil {
			log.Printf("Failed to reset test table: %v", err)
		}
	}
}
