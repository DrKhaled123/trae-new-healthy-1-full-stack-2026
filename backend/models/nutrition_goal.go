package models

import (
	"time"
)

// NutritionGoal represents a nutrition goal for a user
type NutritionGoal struct {
	ID            int        `json:"id" db:"id"`
	UserID        uint       `json:"user_id" db:"user_id"`
	DailyCalories *int       `json:"daily_calories,omitempty" db:"daily_calories"`
	ProteinGrams  *float64   `json:"protein_grams,omitempty" db:"protein_grams"`
	CarbsGrams    *float64   `json:"carbs_grams,omitempty" db:"carbs_grams"`
	FatGrams      *float64   `json:"fat_grams,omitempty" db:"fat_grams"`
	FiberGrams    *float64   `json:"fiber_grams,omitempty" db:"fiber_grams"`
	SugarGrams    *float64   `json:"sugar_grams,omitempty" db:"sugar_grams"`
	SodiumMg      *int       `json:"sodium_mg,omitempty" db:"sodium_mg"`
	WaterMl       *int       `json:"water_ml,omitempty" db:"water_ml"`
	IsActive      bool       `json:"is_active" db:"is_active"`
	StartDate     *time.Time `json:"start_date,omitempty" db:"start_date"`
	EndDate       *time.Time `json:"end_date,omitempty" db:"end_date"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}
