package models

import (
	"nutrition-platform/errors"
	"time"
)

// Food represents a food item in the database
// Updated to match repository expectations
type Food struct {
	ID           uint      `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Description  *string   `json:"description" db:"description"`
	Brand        *string   `json:"brand" db:"brand"`
	Barcode      *string   `json:"barcode" db:"barcode"`
	BarCode      *string   `json:"bar_code" db:"bar_code"` // Repository uses BarCode
	Category     *string   `json:"category" db:"category"`
	Calories     float64   `json:"calories" db:"calories"`
	Protein      float64   `json:"protein" db:"protein"`
	Carbs        float64   `json:"carbs" db:"carbs"`
	Fat          float64   `json:"fat" db:"fat"`
	SaturatedFat float64   `json:"saturated_fat" db:"saturated_fat"`
	Fiber        float64   `json:"fiber" db:"fiber"`
	Sugar        float64   `json:"sugar" db:"sugar"`
	Sodium       int       `json:"sodium" db:"sodium"`
	Cholesterol  float64   `json:"cholesterol" db:"cholesterol"`
	Potassium    float64   `json:"potassium" db:"potassium"`
	ServingSize  string    `json:"serving_size" db:"serving_size"`
	ServingUnit  string    `json:"serving_unit" db:"serving_unit"`
	UserID       *uint     `json:"user_id" db:"user_id"`
	SourceType   string    `json:"source_type" db:"source_type"`
	IsVerified   bool      `json:"is_verified" db:"is_verified"`
	Verified     bool      `json:"verified" db:"verified"` // Repository uses Verified
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// FoodSearchFilters represents filters for food search
type FoodSearchFilters struct {
	Brand         string  `json:"brand"`
	SourceType    string  `json:"source_type"`
	Verified      *bool   `json:"verified"`
	SortBy        string  `json:"sort_by"`
	SortDirection string  `json:"sort_direction"`
}

// CreateFoodRequest represents a request to create a food
type CreateFoodRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=200"`
	Description *string `json:"description,omitempty"`
	Brand       *string `json:"brand,omitempty"`
	Barcode     *string `json:"barcode,omitempty"`
	Category    *string `json:"category,omitempty"`
	Calories    float64 `json:"calories" validate:"required,min=0"`
	Protein     float64 `json:"protein" validate:"required,min=0"`
	Carbs       float64 `json:"carbs" validate:"required,min=0"`
	Fat         float64 `json:"fat" validate:"required,min=0"`
	Fiber       float64 `json:"fiber" validate:"min=0"`
	Sugar       float64 `json:"sugar" validate:"min=0"`
	Sodium      int     `json:"sodium" validate:"min=0"`
	ServingSize string  `json:"serving_size" validate:"required,min=1,max=50"`
	ServingUnit string  `json:"serving_unit" validate:"required,min=1,max=20"`
}

// UpdateFoodRequest represents a request to update a food
type UpdateFoodRequest struct {
	Name        *string  `json:"name,omitempty" validate:"omitempty,min=2,max=200"`
	Description *string  `json:"description,omitempty"`
	Brand       *string  `json:"brand,omitempty"`
	Barcode     *string  `json:"barcode,omitempty"`
	Category    *string  `json:"category,omitempty"`
	Calories    *float64 `json:"calories,omitempty" validate:"omitempty,min=0"`
	Protein     *float64 `json:"protein,omitempty" validate:"omitempty,min=0"`
	Carbs       *float64 `json:"carbs,omitempty" validate:"omitempty,min=0"`
	Fat         *float64 `json:"fat,omitempty" validate:"omitempty,min=0"`
	Fiber       *float64 `json:"fiber,omitempty" validate:"omitempty,min=0"`
	Sugar       *float64 `json:"sugar,omitempty" validate:"omitempty,min=0"`
	Sodium      *int     `json:"sodium,omitempty" validate:"omitempty,min=0"`
	ServingSize *string  `json:"serving_size,omitempty" validate:"omitempty,min=1,max=50"`
	ServingUnit *string  `json:"serving_unit,omitempty" validate:"omitempty,min=1,max=20"`
}

// TableName returns the table name for the Food model
func (Food) TableName() string {
	return "foods"
}

// Validate validates the food model
func (f *Food) Validate() error {
	if f.Name == "" {
		return errors.ErrInvalidInputError("name is required")
	}
	if len(f.Name) > 200 {
		return errors.ErrInvalidInputError("name must be less than 200 characters")
	}
	// Category is optional now, but if provided, validate it
	if f.Category != nil && *f.Category != "" {
		validCategories := []string{"fruits", "vegetables", "grains", "proteins", "dairy", "fats", "beverages", "other"}
		if !contains(validCategories, *f.Category) {
			return errors.ErrInvalidInputError("invalid category")
		}
	}
	if f.Calories < 0 {
		return errors.ErrInvalidInputError("calories cannot be negative")
	}
	if f.Protein < 0 || f.Carbs < 0 || f.Fat < 0 || f.Fiber < 0 || f.Sugar < 0 {
		return errors.ErrInvalidInputError("nutrient values cannot be negative")
	}
	return nil
}

// CalculateNutrition calculates nutrition info for a specific quantity
func (f *Food) CalculateNutrition(quantity float64) *NutritionInfo {
	scale := quantity / 100.0 // nutrients are per 100g

	return &NutritionInfo{
		Calories:      f.Calories * scale,
		Protein:       f.Protein * scale,
		Carbohydrates: f.Carbs * scale,
		Fat:           f.Fat * scale,
		Fiber:         f.Fiber * scale,
		Sugar:         f.Sugar * scale,
		Sodium:        float64(f.Sodium) * scale, // Convert int to float64
		// VitaminC, Calcium, Iron removed as they're not in the updated model
	}
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
