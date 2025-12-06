package models

import (
	"nutrition-platform/errors"
	"time"
)

// User represents a user in the system
type User struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Username  string    `json:"username" gorm:"uniqueIndex;not null" validate:"required,min=3,max=50"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null" validate:"required,email"`
	Age       int       `json:"age" gorm:"not null" validate:"min=1,max=120"`
	Gender    string    `json:"gender" gorm:"not null" validate:"required,oneof=male female other"`
	Height    float64   `json:"height" gorm:"not null" validate:"min=50,max=300"` // in cm
	Weight    float64   `json:"weight" gorm:"not null" validate:"min=1,max=500"`  // in kg
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName returns the table name for the User model
func (User) TableName() string {
	return "users"
}

// Validate validates the user model
func (u *User) Validate() error {
	// Basic validation - in a real app, you'd use a validation library
	if u.Username == "" {
		return errors.ErrInvalidInputError("username is required")
	}
	if len(u.Username) < 3 || len(u.Username) > 50 {
		return errors.ErrInvalidInputError("username must be between 3 and 50 characters")
	}
	if u.Email == "" {
		return errors.ErrInvalidInputError("email is required")
	}
	if u.Age < 1 || u.Age > 120 {
		return errors.ErrInvalidInputError("age must be between 1 and 120")
	}
	if u.Gender == "" {
		return errors.ErrInvalidInputError("gender is required")
	}
	if u.Height < 50 || u.Height > 300 {
		return errors.ErrInvalidInputError("height must be between 50 and 300 cm")
	}
	if u.Weight < 1 || u.Weight > 500 {
		return errors.ErrInvalidInputError("weight must be between 1 and 500 kg")
	}
	return nil
}
