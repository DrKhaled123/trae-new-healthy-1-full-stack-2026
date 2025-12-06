package services

import (
	"database/sql"
)

// ProductService handles product-related operations
type ProductService struct {
	db *sql.DB
}

// NewProductService creates a new ProductService instance
func NewProductService(db *sql.DB) *ProductService {
	return &ProductService{
		db: db,
	}
}
