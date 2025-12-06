package tests

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// TestConfig holds configuration for test environment
type TestConfig struct {
	DatabaseURL    string
	TestDataDir    string
	LogLevel       string
	Timeout        time.Duration
	CleanupOnExit  bool
	UseInMemoryDB  bool
	SeedData       bool
	ParallelTests  bool
	CoverageReport bool
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		DatabaseURL:    ":memory:",
		TestDataDir:    "./testdata",
		LogLevel:       "error",
		Timeout:        30 * time.Second,
		CleanupOnExit:  true,
		UseInMemoryDB:  true,
		SeedData:       true,
		ParallelTests:  true,
		CoverageReport: false,
	}
}

// LoadTestConfig loads test configuration from environment variables
func LoadTestConfig() *TestConfig {
	config := DefaultTestConfig()

	if dbURL := os.Getenv("TEST_DATABASE_URL"); dbURL != "" {
		config.DatabaseURL = dbURL
		config.UseInMemoryDB = false
	}

	if testDir := os.Getenv("TEST_DATA_DIR"); testDir != "" {
		config.TestDataDir = testDir
	}

	if logLevel := os.Getenv("TEST_LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}

	if os.Getenv("TEST_CLEANUP") == "false" {
		config.CleanupOnExit = false
	}

	if os.Getenv("TEST_SEED_DATA") == "false" {
		config.SeedData = false
	}

	if os.Getenv("TEST_PARALLEL") == "false" {
		config.ParallelTests = false
	}

	if os.Getenv("TEST_COVERAGE") == "true" {
		config.CoverageReport = true
	}

	return config
}

// TestDatabase manages test database lifecycle
type TestDatabase struct {
	DB       *sql.DB
	Config   *TestConfig
	TempFile string
	cleanup  []func() error
}

// NewTestDatabase creates a new test database instance
func NewTestDatabase(config *TestConfig) (*TestDatabase, error) {
	td := &TestDatabase{
		Config:  config,
		cleanup: make([]func() error, 0),
	}

	if err := td.Setup(); err != nil {
		return nil, fmt.Errorf("failed to setup test database: %w", err)
	}

	return td, nil
}

// Setup initializes the test database
func (td *TestDatabase) Setup() error {
	var dbURL string

	if td.Config.UseInMemoryDB {
		dbURL = ":memory:"
	} else {
		// Create temporary database file
		tempFile, err := os.CreateTemp("", "test_db_*.sqlite")
		if err != nil {
			return fmt.Errorf("failed to create temp database file: %w", err)
		}
		tempFile.Close()

		td.TempFile = tempFile.Name()
		dbURL = td.TempFile

		// Add cleanup for temp file
		td.cleanup = append(td.cleanup, func() error {
			return os.Remove(td.TempFile)
		})
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbURL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	td.DB = db

	// Add cleanup for database connection
	td.cleanup = append(td.cleanup, func() error {
		return td.DB.Close()
	})

	// Configure database
	if err := td.configureDatabase(); err != nil {
		return fmt.Errorf("failed to configure database: %w", err)
	}

	// Create schema
	if err := td.createSchema(); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Seed data if enabled
	if td.Config.SeedData {
		if err := td.seedData(); err != nil {
			return fmt.Errorf("failed to seed data: %w", err)
		}
	}

	return nil
}

// configureDatabase sets database configuration
func (td *TestDatabase) configureDatabase() error {
	configs := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = 1000",
		"PRAGMA temp_store = memory",
	}

	for _, config := range configs {
		if _, err := td.DB.Exec(config); err != nil {
			return fmt.Errorf("failed to execute config '%s': %w", config, err)
		}
	}

	return nil
}

// createSchema creates database tables
func (td *TestDatabase) createSchema() error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			first_name TEXT,
			last_name TEXT,
			age INTEGER,
			gender TEXT CHECK(gender IN ('male', 'female', 'other')),
			height REAL CHECK(height > 0),
			weight REAL CHECK(weight > 0),
			activity_level TEXT CHECK(activity_level IN ('sedentary', 'lightly_active', 'moderately_active', 'very_active', 'extremely_active')),
			goal TEXT CHECK(goal IN ('lose_weight', 'maintain_weight', 'gain_weight', 'build_muscle')),
			dietary_restrictions TEXT,
			allergies TEXT,
			is_active BOOLEAN DEFAULT 1,
			email_verified BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS foods (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			brand TEXT,
			category TEXT,
			barcode TEXT UNIQUE,
			serving_size REAL,
			serving_unit TEXT,
			calories INTEGER,
			protein REAL,
			carbs REAL,
			fat REAL,
			fiber REAL,
			sugar REAL,
			sodium INTEGER,
			potassium INTEGER,
			vitamin_a REAL,
			vitamin_c REAL,
			vitamin_d REAL,
			vitamin_e REAL,
			vitamin_k REAL,
			thiamin REAL,
			riboflavin REAL,
			niacin REAL,
			vitamin_b6 REAL,
			folate REAL,
			vitamin_b12 REAL,
			calcium INTEGER,
			iron REAL,
			magnesium INTEGER,
			phosphorus INTEGER,
			zinc REAL,
			copper REAL,
			manganese REAL,
			selenium REAL,
			is_verified BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS exercises (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			category TEXT,
			muscle_group TEXT,
			equipment TEXT,
			difficulty TEXT CHECK(difficulty IN ('beginner', 'intermediate', 'advanced')),
			calories_per_hour INTEGER,
			met_value REAL,
			instructions TEXT,
			video_url TEXT,
			image_url TEXT,
			is_active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS user_food_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			food_id INTEGER NOT NULL,
			quantity REAL NOT NULL CHECK(quantity > 0),
			meal_type TEXT NOT NULL CHECK(meal_type IN ('breakfast', 'lunch', 'dinner', 'snack')),
			logged_at DATETIME NOT NULL,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (food_id) REFERENCES foods(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS user_exercise_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			exercise_id INTEGER NOT NULL,
			duration INTEGER NOT NULL CHECK(duration > 0),
			sets INTEGER CHECK(sets > 0),
			reps INTEGER CHECK(reps > 0),
			weight REAL CHECK(weight >= 0),
			distance REAL CHECK(distance >= 0),
			calories_burned INTEGER,
			logged_at DATETIME NOT NULL,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (exercise_id) REFERENCES exercises(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS meal_plans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			target_calories INTEGER,
			target_protein REAL,
			target_carbs REAL,
			target_fat REAL,
			duration_days INTEGER,
			is_active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS workout_plans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			difficulty TEXT CHECK(difficulty IN ('beginner', 'intermediate', 'advanced')),
			duration_weeks INTEGER,
			sessions_per_week INTEGER,
			is_active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			key_hash TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			permissions TEXT,
			rate_limit INTEGER DEFAULT 1000,
			expires_at DATETIME,
			last_used_at DATETIME,
			is_active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS system_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			metric_name TEXT NOT NULL,
			metric_value REAL NOT NULL,
			metric_type TEXT NOT NULL,
			tags TEXT,
			recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	// Create indexes for better performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)",
		"CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)",
		"CREATE INDEX IF NOT EXISTS idx_foods_name ON foods(name)",
		"CREATE INDEX IF NOT EXISTS idx_foods_category ON foods(category)",
		"CREATE INDEX IF NOT EXISTS idx_foods_barcode ON foods(barcode)",
		"CREATE INDEX IF NOT EXISTS idx_exercises_name ON exercises(name)",
		"CREATE INDEX IF NOT EXISTS idx_exercises_category ON exercises(category)",
		"CREATE INDEX IF NOT EXISTS idx_user_food_logs_user_id ON user_food_logs(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_user_food_logs_logged_at ON user_food_logs(logged_at)",
		"CREATE INDEX IF NOT EXISTS idx_user_exercise_logs_user_id ON user_exercise_logs(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_user_exercise_logs_logged_at ON user_exercise_logs(logged_at)",
		"CREATE INDEX IF NOT EXISTS idx_meal_plans_user_id ON meal_plans(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_workout_plans_user_id ON workout_plans(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash)",
		"CREATE INDEX IF NOT EXISTS idx_system_metrics_name ON system_metrics(metric_name)",
		"CREATE INDEX IF NOT EXISTS idx_system_metrics_recorded_at ON system_metrics(recorded_at)",
	}

	// Execute schema creation
	for _, stmt := range schema {
		if _, err := td.DB.Exec(stmt); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Execute index creation
	for _, stmt := range indexes {
		if _, err := td.DB.Exec(stmt); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// seedData inserts test data
func (td *TestDatabase) seedData() error {
	// Insert test users
	users := [][]interface{}{
		{"testuser1", "test1@example.com", "$2a$10$hash1", "John", "Doe", 25, "male", 175.0, 70.0, "moderately_active", "maintain_weight", "", "", 1, 1},
		{"testuser2", "test2@example.com", "$2a$10$hash2", "Jane", "Smith", 30, "female", 165.0, 60.0, "lightly_active", "lose_weight", "vegetarian", "nuts", 1, 1},
		{"testuser3", "test3@example.com", "$2a$10$hash3", "Bob", "Johnson", 35, "male", 180.0, 80.0, "very_active", "build_muscle", "", "", 1, 0},
	}

	for _, user := range users {
		_, err := td.DB.Exec(`
			INSERT INTO users (username, email, password_hash, first_name, last_name, age, gender, height, weight, activity_level, goal, dietary_restrictions, allergies, is_active, email_verified)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, user...)
		if err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}
	}

	// Insert test foods
	foods := [][]interface{}{
		{"Apple", "Generic", "Fruit", "", 100.0, "g", 52, 0.3, 14.0, 0.2, 2.4, 10.4, 1, 107, 54.0, 4.6, 0.0, 0.18, 2.2, 0.031, 0.026, 0.091, 0.041, 3.0, 0.0, 6, 0.12, 5, 11, 0.04, 0.027, 0.035, 0.0, 1},
		{"Banana", "Generic", "Fruit", "", 100.0, "g", 89, 1.1, 23.0, 0.3, 2.6, 12.2, 1, 358, 64.0, 8.7, 0.0, 0.10, 0.5, 0.031, 0.073, 0.665, 0.367, 20.0, 0.0, 5, 0.26, 27, 22, 0.15, 0.078, 0.270, 1.0, 1},
		{"Chicken Breast", "Generic", "Meat", "", 100.0, "g", 165, 31.0, 0.0, 3.6, 0.0, 0.0, 74, 256, 6.0, 0.0, 0.1, 0.5, 0.3, 0.070, 0.120, 14.8, 0.6, 4.0, 0.3, 15, 0.9, 29, 228, 1.0, 0.048, 0.018, 27.6, 1},
		{"Brown Rice", "Generic", "Grain", "", 100.0, "g", 111, 2.6, 23.0, 0.9, 1.8, 0.4, 5, 43, 0.0, 0.0, 0.0, 0.4, 0.0, 0.101, 0.093, 2.996, 0.149, 4.0, 0.0, 10, 0.56, 44, 77, 0.6, 0.196, 1.180, 7.5, 1},
		{"Salmon", "Generic", "Fish", "", 100.0, "g", 208, 25.4, 0.0, 12.4, 0.0, 0.0, 59, 363, 40.0, 0.0, 11.3, 3.2, 0.5, 0.226, 0.155, 8.5, 0.8, 29.0, 2.8, 9, 0.3, 29, 252, 0.4, 0.049, 0.016, 36.5, 1},
	}

	for _, food := range foods {
		_, err := td.DB.Exec(`
			INSERT INTO foods (name, brand, category, barcode, serving_size, serving_unit, calories, protein, carbs, fat, fiber, sugar, sodium, potassium, vitamin_a, vitamin_c, vitamin_d, vitamin_e, vitamin_k, thiamin, riboflavin, niacin, vitamin_b6, folate, vitamin_b12, calcium, iron, magnesium, phosphorus, zinc, copper, manganese, selenium, is_verified)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, food...)
		if err != nil {
			return fmt.Errorf("failed to insert food: %w", err)
		}
	}

	// Insert test exercises
	exercises := [][]interface{}{
		{"Push-ups", "Strength", "Chest", "None", "beginner", 300, 3.8, "Perform push-ups with proper form", "", "", 1},
		{"Running", "Cardio", "Full Body", "None", "intermediate", 600, 8.0, "Run at moderate pace", "", "", 1},
		{"Squats", "Strength", "Legs", "None", "beginner", 250, 3.5, "Perform squats with proper form", "", "", 1},
		{"Deadlifts", "Strength", "Back", "Barbell", "advanced", 400, 6.0, "Perform deadlifts with proper form", "", "", 1},
		{"Cycling", "Cardio", "Legs", "Bicycle", "intermediate", 500, 7.5, "Cycle at moderate intensity", "", "", 1},
	}

	for _, exercise := range exercises {
		_, err := td.DB.Exec(`
			INSERT INTO exercises (name, category, muscle_group, equipment, difficulty, calories_per_hour, met_value, instructions, video_url, image_url, is_active)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, exercise...)
		if err != nil {
			return fmt.Errorf("failed to insert exercise: %w", err)
		}
	}

	return nil
}

// Cleanup cleans up test database resources
func (td *TestDatabase) Cleanup() error {
	var errors []error

	// Run cleanup functions in reverse order
	for i := len(td.cleanup) - 1; i >= 0; i-- {
		if err := td.cleanup[i](); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// TestHelper provides utility functions for tests
type TestHelper struct {
	DB     *sql.DB
	Config *TestConfig
}

// NewTestHelper creates a new test helper
func NewTestHelper(db *sql.DB, config *TestConfig) *TestHelper {
	return &TestHelper{
		DB:     db,
		Config: config,
	}
}

// TruncateTable truncates a table for test isolation
func (th *TestHelper) TruncateTable(tableName string) error {
	_, err := th.DB.Exec(fmt.Sprintf("DELETE FROM %s", tableName))
	if err != nil {
		return fmt.Errorf("failed to truncate table %s: %w", tableName, err)
	}

	// Reset auto-increment counter for SQLite
	_, err = th.DB.Exec(fmt.Sprintf("DELETE FROM sqlite_sequence WHERE name='%s'", tableName))
	if err != nil {
		// Ignore error if sqlite_sequence doesn't exist or table doesn't use auto-increment
		log.Printf("Warning: failed to reset auto-increment for table %s: %v", tableName, err)
	}

	return nil
}

// TruncateAllTables truncates all tables
func (th *TestHelper) TruncateAllTables() error {
	tables := []string{
		"system_metrics",
		"api_keys",
		"workout_plans",
		"meal_plans",
		"user_exercise_logs",
		"user_food_logs",
		"exercises",
		"foods",
		"users",
	}

	for _, table := range tables {
		if err := th.TruncateTable(table); err != nil {
			return err
		}
	}

	return nil
}

// CreateTestUser creates a test user and returns the ID
func (th *TestHelper) CreateTestUser(username, email string) (int, error) {
	result, err := th.DB.Exec(`
		INSERT INTO users (username, email, password_hash, first_name, last_name, age, gender, height, weight, activity_level, goal, is_active, email_verified)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, username, email, "$2a$10$testhash", "Test", "User", 25, "male", 175.0, 70.0, "moderately_active", "maintain_weight", 1, 1)
	if err != nil {
		return 0, fmt.Errorf("failed to create test user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get user ID: %w", err)
	}

	return int(id), nil
}

// CreateTestFood creates a test food and returns the ID
func (th *TestHelper) CreateTestFood(name, category string, calories int) (int, error) {
	result, err := th.DB.Exec(`
		INSERT INTO foods (name, category, serving_size, serving_unit, calories, protein, carbs, fat, is_verified)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, name, category, 100.0, "g", calories, 5.0, 20.0, 2.0, 1)
	if err != nil {
		return 0, fmt.Errorf("failed to create test food: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get food ID: %w", err)
	}

	return int(id), nil
}

// CreateTestExercise creates a test exercise and returns the ID
func (th *TestHelper) CreateTestExercise(name, category string, caloriesPerHour int) (int, error) {
	result, err := th.DB.Exec(`
		INSERT INTO exercises (name, category, muscle_group, equipment, difficulty, calories_per_hour, met_value, instructions, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, name, category, "Full Body", "None", "beginner", caloriesPerHour, 5.0, "Test exercise instructions", 1)
	if err != nil {
		return 0, fmt.Errorf("failed to create test exercise: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get exercise ID: %w", err)
	}

	return int(id), nil
}

// AssertTableCount asserts the number of rows in a table
func (th *TestHelper) AssertTableCount(t *testing.T, tableName string, expected int) {
	var count int
	err := th.DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rows in table %s: %v", tableName, err)
	}

	if count != expected {
		t.Errorf("Expected %d rows in table %s, got %d", expected, tableName, count)
	}
}

// GetTestDataPath returns the path to test data files
func (th *TestHelper) GetTestDataPath(filename string) string {
	return filepath.Join(th.Config.TestDataDir, filename)
}

// LoadTestDataFile loads test data from a file
func (th *TestHelper) LoadTestDataFile(filename string) ([]byte, error) {
	path := th.GetTestDataPath(filename)
	return os.ReadFile(path)
}
