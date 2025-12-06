package services

import (
	"database/sql"
)

// WorkoutService handles workout-related operations
type WorkoutService struct {
	db *sql.DB
}

// NewWorkoutService creates a new WorkoutService instance
func NewWorkoutService(db *sql.DB) *WorkoutService {
	return &WorkoutService{
		db: db,
	}
}
