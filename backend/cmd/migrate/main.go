package main

import (
	"database/sql"
	"flag"
	"log"
	"os"

	"nutrition-platform/config"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	var (
		direction = flag.String("direction", "up", "Migration direction: up or down")
		steps     = flag.Int("steps", 0, "Number of migration steps (0 for all)")
		version   = flag.Int("version", -1, "Migrate to specific version")
	)
	flag.Parse()

	// Load configuration
	cfg := config.LoadConfig()

	// Connect to database
	db, err := sql.Open("postgres", cfg.GetDatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create migrate instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("Failed to create database driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	// Execute migration based on flags
	switch {
	case *version >= 0:
		log.Printf("Migrating to version %d", *version)
		err = m.Migrate(uint(*version))
	case *direction == "down":
		if *steps > 0 {
			log.Printf("Migrating down %d steps", *steps)
			err = m.Steps(-*steps)
		} else {
			log.Println("Migrating down all")
			err = m.Down()
		}
	default: // up
		if *steps > 0 {
			log.Printf("Migrating up %d steps", *steps)
			err = m.Steps(*steps)
		} else {
			log.Println("Migrating up all")
			err = m.Up()
		}
	}

	if err != nil {
		if err == migrate.ErrNoChange {
			log.Println("No migrations to apply")
			os.Exit(0)
		}
		log.Fatalf("Migration failed: %v", err)
	}

	currentVersion, dirty, err := m.Version()
	if err != nil {
		log.Printf("Warning: Could not get current version: %v", err)
	} else {
		log.Printf("Migration completed successfully. Current version: %d, Dirty: %t", currentVersion, dirty)
	}
}
