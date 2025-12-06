package models

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// APIKeyStatus represents the status of an API key
type APIKeyStatus string

const (
	APIKeyStatusActive   APIKeyStatus = "active"
	APIKeyStatusInactive APIKeyStatus = "inactive"
	APIKeyStatusRevoked  APIKeyStatus = "revoked"
	APIKeyStatusExpired  APIKeyStatus = "expired"
)

// APIKeyScope represents the scope/permissions of an API key
type APIKeyScope string

const (
	ScopeReadOnly    APIKeyScope = "read_only"
	ScopeReadWrite   APIKeyScope = "read_write"
	ScopeAdmin       APIKeyScope = "admin"
	ScopeNutrition   APIKeyScope = "nutrition"
	ScopeWorkouts    APIKeyScope = "workouts"
	ScopeMeals       APIKeyScope = "meals"
	ScopeHealth      APIKeyScope = "health"
	ScopeSupplements APIKeyScope = "supplements"
)

// APIKey represents an API key in the system
type APIKey struct {
	ID         string                 `json:"id" db:"id"`
	Name       string                 `json:"name" db:"name"`
	KeyHash    string                 `json:"-" db:"key_hash"` // Never expose the actual hash
	Prefix     string                 `json:"prefix" db:"prefix"`
	UserID     string                 `json:"user_id" db:"user_id"`
	Status     APIKeyStatus           `json:"status" db:"status"`
	Scopes     []APIKeyScope          `json:"scopes" db:"scopes"`
	RateLimit  int                    `json:"rate_limit" db:"rate_limit"` // requests per minute
	ExpiresAt  *time.Time             `json:"expires_at" db:"expires_at"`
	LastUsedAt *time.Time             `json:"last_used_at" db:"last_used_at"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at" db:"updated_at"`
	Metadata   map[string]interface{} `json:"metadata" db:"metadata"`
}

// APIKeyUsage tracks API key usage statistics
type APIKeyUsage struct {
	APIKeyID     string    `json:"api_key_id" db:"api_key_id"`
	Endpoint     string    `json:"endpoint" db:"endpoint"`
	Method       string    `json:"method" db:"method"`
	StatusCode   int       `json:"status_code" db:"status_code"`
	ResponseTime int64     `json:"response_time" db:"response_time"` // in milliseconds
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	UserAgent    string    `json:"user_agent" db:"user_agent"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
}

// GenerateAPIKey creates a new secure API key
func GenerateAPIKey(prefix string) (string, string, error) {
	if prefix == "" {
		prefix = "nk" // nutrition key
	}

	// Generate 32 random bytes
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Create the key with prefix
	key := fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(bytes))

	// Create hash for storage
	hash := sha256.Sum256([]byte(key))
	keyHash := hex.EncodeToString(hash[:])

	return key, keyHash, nil
}

// ValidateAPIKeyFormat checks if the API key format is valid
func ValidateAPIKeyFormat(key string) bool {
	parts := strings.Split(key, "_")
	if len(parts) != 2 {
		return false
	}

	prefix := parts[0]
	token := parts[1]

	// Check prefix (2-4 characters)
	if len(prefix) < 2 || len(prefix) > 4 {
		return false
	}

	// Check token (64 hex characters)
	if len(token) != 64 {
		return false
	}

	// Verify hex encoding
	_, err := hex.DecodeString(token)
	return err == nil
}

// HashAPIKey creates a hash of the API key for storage
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// GetPrefix extracts the prefix from an API key
func GetPrefix(key string) string {
	parts := strings.Split(key, "_")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

// IsExpired checks if the API key has expired
func (ak *APIKey) IsExpired() bool {
	if ak.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*ak.ExpiresAt)
}

// IsActive checks if the API key is active and not expired
func (ak *APIKey) IsActive() bool {
	return ak.Status == APIKeyStatusActive && !ak.IsExpired()
}

// HasScope checks if the API key has a specific scope
func (ak *APIKey) HasScope(scope APIKeyScope) bool {
	// Admin scope has access to everything
	for _, s := range ak.Scopes {
		if s == ScopeAdmin {
			return true
		}
		if s == scope {
			return true
		}
	}
	return false
}

// CanAccess checks if the API key can access a specific endpoint
func (ak *APIKey) CanAccess(endpoint, method string) bool {
	if !ak.IsActive() {
		return false
	}

	// Define endpoint to scope mapping
	endpointScopes := map[string]APIKeyScope{
		"/api/v1/nutrition":   ScopeNutrition,
		"/api/v1/meals":       ScopeMeals,
		"/api/v1/workouts":    ScopeWorkouts,
		"/api/v1/health":      ScopeHealth,
		"/api/v1/supplements": ScopeSupplements,
	}

	// Check if endpoint requires specific scope
	for endpointPrefix, requiredScope := range endpointScopes {
		if strings.HasPrefix(endpoint, endpointPrefix) {
			if !ak.HasScope(requiredScope) {
				return false
			}
			break
		}
	}

	// Check read/write permissions
	if method == "GET" || method == "HEAD" {
		return ak.HasScope(ScopeReadOnly) || ak.HasScope(ScopeReadWrite) || ak.HasScope(ScopeAdmin)
	}

	if method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE" {
		return ak.HasScope(ScopeReadWrite) || ak.HasScope(ScopeAdmin)
	}

	return false
}

// UpdateLastUsed updates the last used timestamp
func (ak *APIKey) UpdateLastUsed() {
	now := time.Now()
	ak.LastUsedAt = &now
	ak.UpdatedAt = now
}

// CreateAPIKeyRequest represents a request to create a new API key
type CreateAPIKeyRequest struct {
	Name      string                 `json:"name" validate:"required,min=3,max=100"`
	Scopes    []APIKeyScope          `json:"scopes" validate:"required,min=1"`
	RateLimit int                    `json:"rate_limit" validate:"min=1,max=10000"`
	ExpiresIn *int                   `json:"expires_in"` // days from now, nil for no expiration
	Metadata  map[string]interface{} `json:"metadata"`
}

// CreateAPIKeyResponse represents the response when creating an API key
type CreateAPIKeyResponse struct {
	APIKey  *APIKey `json:"api_key"`
	Key     string  `json:"key"` // Only returned once during creation
	Warning string  `json:"warning,omitempty"`
}

// APIKeyListResponse represents a list of API keys
type APIKeyListResponse struct {
	APIKeys []APIKey `json:"api_keys"`
	Total   int      `json:"total"`
	Page    int      `json:"page"`
	Limit   int      `json:"limit"`
}

// APIKeyStatsResponse represents API key usage statistics
type APIKeyStatsResponse struct {
	APIKeyID      string           `json:"api_key_id"`
	TotalRequests int64            `json:"total_requests"`
	LastUsed      *time.Time       `json:"last_used"`
	Endpoints     map[string]int64 `json:"endpoints"`
	StatusCodes   map[string]int64 `json:"status_codes"`
	DailyUsage    []DailyUsage     `json:"daily_usage"`
}

// DailyUsage represents daily usage statistics
type DailyUsage struct {
	Date     string `json:"date"`
	Requests int64  `json:"requests"`
}

// ValidateScopes validates that the provided scopes are valid
func ValidateScopes(scopes []APIKeyScope) error {
	validScopes := map[APIKeyScope]bool{
		ScopeReadOnly:    true,
		ScopeReadWrite:   true,
		ScopeAdmin:       true,
		ScopeNutrition:   true,
		ScopeWorkouts:    true,
		ScopeMeals:       true,
		ScopeHealth:      true,
		ScopeSupplements: true,
	}

	for _, scope := range scopes {
		if !validScopes[scope] {
			return fmt.Errorf("invalid scope: %s", scope)
		}
	}

	return nil
}
