package models

import (
	"nutrition-platform/errors"
	"time"
)

// UserFoodLog represents a food logging entry by a user
type UserFoodLog struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    int       `json:"user_id" gorm:"not null;index" validate:"required,min=1"`
	FoodID    int       `json:"food_id" gorm:"not null;index" validate:"required,min=1"`
	Quantity  float64   `json:"quantity" gorm:"not null" validate:"required,min=0"` // grams
	MealType  string    `json:"meal_type" gorm:"not null" validate:"required,oneof=breakfast lunch dinner snack"`
	LoggedAt  time.Time `json:"logged_at" gorm:"not null;index" validate:"required"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName returns the table name for the UserFoodLog model
func (UserFoodLog) TableName() string {
	return "user_food_logs"
}

// Validate validates the food log model
func (log *UserFoodLog) Validate() error {
	if log.UserID <= 0 {
		return errors.ErrInvalidInputError("user_id must be greater than 0")
	}
	if log.FoodID <= 0 {
		return errors.ErrInvalidInputError("food_id must be greater than 0")
	}
	if log.Quantity < 0 {
		return errors.ErrInvalidInputError("quantity cannot be negative")
	}
	if log.Quantity > 10000 {
		return errors.ErrInvalidInputError("quantity cannot exceed 10000 grams")
	}
	validMealTypes := []string{"breakfast", "lunch", "dinner", "snack"}
	if !contains(validMealTypes, log.MealType) {
		return errors.ErrInvalidInputError("invalid meal type")
	}
	return nil
}

// contains function is defined in food.go
