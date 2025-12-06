// database.go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)


var DB *sql.DB

func InitDB() error {
	// For now, use a simple SQLite database path
	// Get database path from environment or use default
	dbPath := "./nutrition_platform.db"

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %v", err)
	}

	// Open database
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	// Configure connection pool
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)

	// Check if database is accessible
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %v", err)
	}

	log.Println("Database initialized successfully")
	return nil
}

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
	}

	// Run each migration if not already applied
	for _, migration := range migrations {
		if err := applyMigration(migration); err != nil {
			return err
		}
	}

	return nil
}

type Migration struct {
	Name string
	SQL  string
}

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

func CloseDatabase() {
	if DB != nil {
		log.Println("Closing database connection")
		if err := DB.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
}
