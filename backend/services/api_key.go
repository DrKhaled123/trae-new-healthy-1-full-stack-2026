package services

import (
	"database/sql"
)

// APIKeyService handles API key-related operations
type APIKeyService struct {
	db *sql.DB
}

// NewAPIKeyService creates a new APIKeyService instance
func NewAPIKeyService(db *sql.DB) *APIKeyService {
	return &APIKeyService{
		db: db,
	}
}
