// Package models provides data models for the nutrition platform
package models

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// DB is the global database connection
var DB *sql.DB

// InitDB initializes the database connection
func InitDB(databaseURL string) *sql.DB {
	// For testing, use in-memory database to avoid hanging
	dbPath := ":memory:"
	log.Printf("Using in-memory database for testing")

	// Open database
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Configure connection pool (not needed for in-memory but kept for consistency)
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)

	// Check if database is accessible
	if err := DB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Database initialized successfully (in-memory)")
	return DB
}

// runMigrations runs database migrations
func runMigrations() error {
	// Create migrations table if not exists
	migrationSQL := `
	CREATE TABLE IF NOT EXISTS migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := DB.Exec(migrationSQL); err != nil {
		return err
	}

	// List of migrations to run
	migrations := []Migration{
		{
			Name: "001_initial_schema",
			SQL: `
				CREATE TABLE IF NOT EXISTS users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					username TEXT NOT NULL UNIQUE,
					email TEXT NOT NULL UNIQUE,
					password_hash TEXT NOT NULL,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS api_keys (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					user_id INTEGER NOT NULL,
					key TEXT NOT NULL UNIQUE,
					quota_limit INTEGER DEFAULT 100,
					quota_used INTEGER DEFAULT 0,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users (id)
				);

				CREATE TABLE IF NOT EXISTS nutrition_plans (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					user_id INTEGER,
					name TEXT NOT NULL,
					description TEXT,
					target_calories INTEGER,
					protein_grams INTEGER,
					carb_grams INTEGER,
					fat_grams INTEGER,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users (id)
				);

				CREATE TABLE IF NOT EXISTS recipes (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL,
					description TEXT,
					instructions TEXT,
					prep_time INTEGER,
					cook_time INTEGER,
					servings INTEGER,
					calories_per_serving INTEGER,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS health_conditions (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL,
					description TEXT,
					nutrition_recommendations TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);`,
		},
		{
			Name: "002_add_indexes",
			SQL: `
				CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
				CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(key);
				CREATE INDEX IF NOT EXISTS idx_nutrition_plans_user_id ON nutrition_plans(user_id);
				CREATE INDEX IF NOT EXISTS idx_recipes_name ON recipes(name);`,
		},
		{
			Name: "003_add_audit_logs",
			SQL: `
				CREATE TABLE IF NOT EXISTS audit_logs (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					user_id INTEGER,
					action TEXT NOT NULL,
					resource_type TEXT NOT NULL,
					resource_id INTEGER,
					ip_address TEXT,
					user_agent TEXT,
					timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users (id)
				);`,
		},
		{
			Name: "006_nutrition_json_data",
			SQL: `
				CREATE TABLE IF NOT EXISTS diet_plans_json (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					diet_name TEXT NOT NULL,
					origin TEXT,
					principles TEXT,
					calorie_levels TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS workout_plans_json (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					api_version TEXT,
					language TEXT,
					purpose TEXT,
					goal TEXT,
					training_days_per_week INTEGER,
					training_split TEXT,
					experience_level TEXT,
					last_updated TEXT,
					license TEXT,
					scientific_references TEXT,
					weekly_plan TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS health_complaint_cases (
					id INTEGER PRIMARY KEY,
					condition_en TEXT NOT NULL,
					condition_ar TEXT NOT NULL,
					recommendations TEXT,
					enhanced_recommendations TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS metabolism_guides (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					section_id TEXT NOT NULL,
					title_en TEXT,
					title_ar TEXT,
					content TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS drug_nutrition_interactions (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					supported_languages TEXT,
					nutritional_recommendations TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE INDEX IF NOT EXISTS idx_diet_plans_name ON diet_plans_json(diet_name);
				CREATE INDEX IF NOT EXISTS idx_diet_plans_origin ON diet_plans_json(origin);
				CREATE INDEX IF NOT EXISTS idx_workout_plans_goal ON workout_plans_json(goal);
				CREATE INDEX IF NOT EXISTS idx_workout_plans_split ON workout_plans_json(training_split);
				CREATE INDEX IF NOT EXISTS idx_complaints_condition_en ON health_complaint_cases(condition_en);
				CREATE INDEX IF NOT EXISTS idx_complaints_condition_ar ON health_complaint_cases(condition_ar);
				CREATE INDEX IF NOT EXISTS idx_metabolism_section_id ON metabolism_guides(section_id);`,
		},
	}

	// Run each migration if not already applied
	for _, migration := range migrations {
		if err := applyMigration(migration); err != nil {
			return err
		}
	}

	return nil
}

// Migration represents a database migration
type Migration struct {
	Name string
	SQL  string
}

// applyMigration applies a single migration
func applyMigration(migration Migration) error {
	// Check if migration already applied
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM migrations WHERE name = ?", migration.Name).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Printf("Migration %s already applied, skipping", migration.Name)
		return nil // Migration already applied
	}

	log.Printf("Applying migration: %s", migration.Name)

	// Apply migration using transaction for safety
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.Exec(migration.SQL); err != nil {
		return err
	}

	// Record migration
	if _, err = tx.Exec("INSERT INTO migrations (name) VALUES (?)", migration.Name); err != nil {
		return err
	}

	return tx.Commit()
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		log.Println("Closing database connection")
		return DB.Close()
	}
	return nil
}

// InitTestDB initializes an in-memory SQLite database for testing
func InitTestDB() *sql.DB {
	// Use in-memory database for testing
	dbPath := ":memory:"

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open test database: %v", err)
	}

	// Configure connection pool for testing
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)

	// Check if database is accessible
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping test database: %v", err)
	}

	// Run test migrations
	if err := runTestMigrations(db); err != nil {
		log.Fatalf("Failed to run test migrations: %v", err)
	}

	log.Println("Test database initialized successfully (in-memory)")
	return db
}

// runTestMigrations runs essential migrations for testing
func runTestMigrations(db *sql.DB) error {
	// Create migrations table if not exists
	migrationSQL := `
	CREATE TABLE IF NOT EXISTS migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.Exec(migrationSQL); err != nil {
		return err
	}

	// List of essential test migrations
	migrations := []Migration{
		{
			Name: "001_initial_schema_test",
			SQL: `
				CREATE TABLE IF NOT EXISTS users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					username TEXT NOT NULL UNIQUE,
					email TEXT NOT NULL UNIQUE,
					password_hash TEXT NOT NULL,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS api_keys (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					user_id INTEGER NOT NULL,
					key TEXT NOT NULL UNIQUE,
					quota_limit INTEGER DEFAULT 100,
					quota_used INTEGER DEFAULT 0,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users (id)
				);

				CREATE TABLE IF NOT EXISTS nutrition_plans (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					user_id INTEGER,
					name TEXT NOT NULL,
					description TEXT,
					target_calories INTEGER,
					protein_grams INTEGER,
					carb_grams INTEGER,
					fat_grams INTEGER,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users (id)
				);

				CREATE TABLE IF NOT EXISTS recipes (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL,
					description TEXT,
					instructions TEXT,
					prep_time INTEGER,
					cook_time INTEGER,
					servings INTEGER,
					calories_per_serving INTEGER,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS health_conditions (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL,
					description TEXT,
					nutrition_recommendations TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);`,
		},
		{
			Name: "006_nutrition_json_data_test",
			SQL: `
				CREATE TABLE IF NOT EXISTS diet_plans_json (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					diet_name TEXT NOT NULL,
					origin TEXT,
					principles TEXT,
					calorie_levels TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS workout_plans_json (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					api_version TEXT,
					language TEXT,
					purpose TEXT,
					goal TEXT,
					training_days_per_week INTEGER,
					training_split TEXT,
					experience_level TEXT,
					last_updated TEXT,
					license TEXT,
					scientific_references TEXT,
					weekly_plan TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS health_complaint_cases (
					id INTEGER PRIMARY KEY,
					condition_en TEXT NOT NULL,
					condition_ar TEXT NOT NULL,
					recommendations TEXT,
					enhanced_recommendations TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS metabolism_guides (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					section_id TEXT NOT NULL,
					title_en TEXT,
					title_ar TEXT,
					content TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS drug_nutrition_interactions (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					supported_languages TEXT,
					nutritional_recommendations TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);`,
		},
	}

	// Run each migration if not already applied
	for _, migration := range migrations {
		if err := applyTestMigration(db, migration); err != nil {
			return err
		}
	}

	return nil
}

// applyTestMigration applies a single test migration
func applyTestMigration(db *sql.DB, migration Migration) error {
	// Check if migration already applied
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM migrations WHERE name = ?", migration.Name).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Printf("Test migration %s already applied, skipping", migration.Name)
		return nil // Migration already applied
	}

	log.Printf("Applying test migration: %s", migration.Name)

	// Apply migration using transaction for safety
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.Exec(migration.SQL); err != nil {
		return err
	}

	// Record migration
	if _, err = tx.Exec("INSERT INTO migrations (name) VALUES (?)", migration.Name); err != nil {
		return err
	}

	return tx.Commit()
}
