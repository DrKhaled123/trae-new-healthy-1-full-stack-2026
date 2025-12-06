package models

import (
	"time"
)

// WeightLog represents a weight tracking entry for a user
type WeightLog struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Weight    float64   `json:"weight" db:"weight"`
	Unit      string    `json:"unit" db:"unit"` // kg, lbs
	Notes     *string   `json:"notes,omitempty" db:"notes"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
