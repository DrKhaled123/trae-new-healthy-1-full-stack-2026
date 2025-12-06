package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"nutrition-platform/validation"
)

func TestInputValidator_Validate(t *testing.T) {
	validator := validation.NewInputValidator()

	// Test struct for validation
	type TestStruct struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=6"`
		Name     string `json:"name" validate:"required,min=2"`
	}

	tests := []struct {
		name    string
		input   TestStruct
		wantErr bool
	}{
		{
			name: "valid input",
			input: TestStruct{
				Email:    "test@example.com",
				Password: "password123",
				Name:     "John Doe",
			},
			wantErr: false,
		},
		{
			name: "invalid email",
			input: TestStruct{
				Email:    "invalid-email",
				Password: "password123",
				Name:     "John Doe",
			},
			wantErr: true,
		},
		{
			name: "short password",
			input: TestStruct{
				Email:    "test@example.com",
				Password: "123",
				Name:     "John Doe",
			},
			wantErr: true,
		},
		{
			name: "missing required fields",
			input: TestStruct{
				Email:    "",
				Password: "",
				Name:     "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid email",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "invalid email",
			email:   "invalid-email",
			wantErr: true,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
		},
		{
			name:    "email with subdomain",
			email:   "test@sub.example.com",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateEmail(tt.email)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateInput_Security(t *testing.T) {
	validator := validation.NewInputValidator()
	context := &validation.ValidationContext{
		UserRole:  "user",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
		Endpoint:  "/test",
		Method:    "POST",
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "normal input",
			input:   "Hello World",
			wantErr: false,
		},
		{
			name:    "SQL injection attempt",
			input:   "'; DROP TABLE users; --",
			wantErr: true,
		},
		{
			name:    "XSS attempt",
			input:   "<script>alert('xss')</script>",
			wantErr: true,
		},
		{
			name:    "command injection",
			input:   "; rm -rf /",
			wantErr: true,
		},
		{
			name:    "path traversal",
			input:   "../../../etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateInput(tt.input, context)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
