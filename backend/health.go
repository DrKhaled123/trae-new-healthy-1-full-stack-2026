// health.go
package main

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
)

type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Database  string            `json:"database"`
	Version   string            `json:"version"`
	Uptime    time.Duration     `json:"uptime"`
	Services  map[string]string `json:"services"`
}

var startTime = time.Now()

func healthHandler(c echo.Context) error {
	// Check database connection
	dbStatus := checkDatabaseStatus()

	// Check other services (this can be expanded)
	serviceStatuses := map[string]string{
		"database": dbStatus,
		"redis":    "unknown", // Add Redis check if needed
	}

	// Determine overall health
	status := "healthy"
	if dbStatus != "ok" {
		status = "unhealthy"
	}

	health := HealthStatus{
		Status:    status,
		Timestamp: time.Now(),
		Database:  dbStatus,
		Version:   getVersion(),
		Uptime:    time.Since(startTime),
		Services:  serviceStatuses,
	}

	statusCode := http.StatusOK
	if health.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	return c.JSON(statusCode, health)
}

func checkDatabaseStatus() string {
	db, err := sql.Open("sqlite3", "./nutrition_platform.db")
	if err != nil {
		return "error: " + err.Error()
	}
	defer db.Close()

	// Ping the database to verify connection
	if err := db.Ping(); err != nil {
		return "error: " + err.Error()
	}

	// Test a simple query
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
	if err != nil {
		return "error: " + err.Error()
	}

	return "ok"
}

func getVersion() string {
	// In a real application, you might read this from a version file or build info
	if version := os.Getenv("APP_VERSION"); version != "" {
		return version
	}
	return "1.0.0"
}
