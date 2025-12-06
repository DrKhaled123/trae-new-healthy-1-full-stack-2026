package tests

import (
	"fmt"
	"log"
	"nutrition-platform/validation"
)

type TestStruct struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required,min=2"`
}

func main() {
	// Test validator initialization
	validator := validation.NewInputValidator()
	if validator == nil {
		log.Fatal("Failed to create validator")
	}

	// Test valid input
	validInput := TestStruct{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "John Doe",
	}

	err := validator.Validate(validInput)
	if err != nil {
		fmt.Printf("Validation failed for valid input: %v\n", err)
	} else {
		fmt.Println("✅ Validation passed for valid input")
	}

	// Test invalid input
	invalidInput := TestStruct{
		Email:    "invalid-email",
		Password: "123",
		Name:     "John Doe",
	}

	err = validator.Validate(invalidInput)
	if err != nil {
		fmt.Printf("✅ Validation correctly failed for invalid input: %v\n", err)
	} else {
		fmt.Println("❌ Validation should have failed for invalid input")
	}

	fmt.Println("Validator test completed successfully!")
}
