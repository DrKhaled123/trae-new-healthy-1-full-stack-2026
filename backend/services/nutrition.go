package services

import (
	"database/sql"
)

// NutritionService handles nutrition-related operations
type NutritionService struct {
	db *sql.DB
}

// NewNutritionService creates a new NutritionService instance
func NewNutritionService(db *sql.DB) *NutritionService {
	return &NutritionService{
		db: db,
	}
}
