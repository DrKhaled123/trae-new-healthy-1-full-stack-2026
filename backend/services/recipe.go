package services

import (
	"database/sql"
)

// RecipeService handles recipe-related operations
type RecipeService struct {
	db *sql.DB
}

// NewRecipeService creates a new RecipeService instance
func NewRecipeService(db *sql.DB) *RecipeService {
	return &RecipeService{
		db: db,
	}
}
