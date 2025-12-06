package services

import (
	"database/sql"
)

// HealthService handles health-related operations
type HealthService struct {
	db *sql.DB
}

// NewHealthService creates a new HealthService instance
func NewHealthService(db *sql.DB) *HealthService {
	return &HealthService{
		db: db,
	}
}
