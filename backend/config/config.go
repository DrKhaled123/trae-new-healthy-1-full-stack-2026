// Package config provides application configuration management
package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds the application configuration
type Config struct {
	Port              string
	DatabaseURL       string
	JWTSecret         string
	Environment       string
	ReadTimeout       int
	WriteTimeout      int
	KeepAliveTimeout  int
	FileStorage       FileStorageConfig
	EmailConfig       EmailConfig
	PushConfig        PushConfig
}

// FileStorageConfig holds file storage configuration
type FileStorageConfig struct {
	StorageType string
	BasePath    string
	BaseURL     string
	S3Bucket    string
	S3Region    string
	S3URL       string
}

// EmailConfig holds email service configuration
type EmailConfig struct {
	Provider   string
	SMTPHost   string
	SMTPPort   int
	SMTPUser   string
	SMTPPass   string
	FromEmail  string
	FromName   string
}

// PushConfig holds push notification configuration
type PushConfig struct {
	FCMServerKey string
	APNSKeyPath  string
	APNSKeyID    string
	APNSTeamID   string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	config := &Config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      getEnv("DATABASE_URL", "sqlite3://./nutrition_platform.db"),
		JWTSecret:        getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		Environment:      getEnv("ENVIRONMENT", "development"),
		ReadTimeout:      getEnvAsInt("READ_TIMEOUT", 30),
		WriteTimeout:     getEnvAsInt("WRITE_TIMEOUT", 30),
		KeepAliveTimeout: getEnvAsInt("KEEP_ALIVE_TIMEOUT", 60),
		FileStorage: FileStorageConfig{
			StorageType: getEnv("STORAGE_TYPE", "local"),
			BasePath:    getEnv("FILE_STORAGE_PATH", "./uploads"),
			BaseURL:     getEnv("FILE_STORAGE_URL", "http://localhost:8080/uploads"),
			S3Bucket:    getEnv("S3_BUCKET", ""),
			S3Region:    getEnv("S3_REGION", "us-east-1"),
			S3URL:       getEnv("S3_URL", ""),
		},
		EmailConfig: EmailConfig{
			Provider:  getEnv("EMAIL_PROVIDER", "smtp"),
			SMTPHost:  getEnv("SMTP_HOST", "localhost"),
			SMTPPort:  getEnvAsInt("SMTP_PORT", 587),
			SMTPUser:  getEnv("SMTP_USER", ""),
			SMTPPass:  getEnv("SMTP_PASS", ""),
			FromEmail: getEnv("FROM_EMAIL", "noreply@nutrition-platform.com"),
			FromName:  getEnv("FROM_NAME", "Nutrition Platform"),
		},
		PushConfig: PushConfig{
			FCMServerKey: getEnv("FCM_SERVER_KEY", ""),
			APNSKeyPath:  getEnv("APNS_KEY_PATH", ""),
			APNSKeyID:    getEnv("APNS_KEY_ID", ""),
			APNSTeamID:   getEnv("APNS_TEAM_ID", ""),
		},
	}

	// Validate required configuration
	if config.JWTSecret == "your-secret-key-change-in-production" && config.Environment == "production" {
		panic("JWT_SECRET must be set in production")
	}

	return config
}

// IsDevelopment checks if the environment is development
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction checks if the environment is production
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsLocalStorage checks if using local file storage
func (c *Config) IsLocalStorage() bool {
	return c.FileStorage.StorageType == "local"
}

// IsS3Storage checks if using S3 file storage
func (c *Config) IsS3Storage() bool {
	return c.FileStorage.StorageType == "s3"
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// GetDatabaseURL returns the database URL for the current environment
func (c *Config) GetDatabaseURL() string {
	return c.DatabaseURL
}