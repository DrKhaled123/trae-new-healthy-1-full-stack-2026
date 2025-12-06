// config.go
package main

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port             string
	DBPath           string
	JWTSecret        string
	APIRateLimit     int
	EnableHTTPS      bool
	Domain           string
	LogLevel         string
	DBDriver         string
	MaxRequestSize   int64
	ReadTimeout      int
	WriteTimeout     int
	KeepAliveTimeout int

	// Security configuration
	EncryptionKey  string
	Salt           string
	CORSOrigins    []string
	AllowedHosts   []string
	TrustedProxies []string
	SessionSecret  string
	CookieSecret   string

	// Database security
	DBPassword       string
	DBSSLMode        string
	DBMaxConnections int

	// API security
	APIKeyLength       int
	TokenExpiry        int
	RefreshTokenExpiry int

	// Monitoring and alerts
	MonitoringEnabled bool
	SentryDSN         string
	AlertEmail        string
}

func LoadConfig() Config {
	config := Config{
		Port:             getEnv("PORT", "8080"),
		DBPath:           getEnv("DB_PATH", "./nutrition_platform.db"),
		JWTSecret:        getEnv("JWT_SECRET", ""),
		APIRateLimit:     getEnvAsInt("API_RATE_LIMIT", 100),
		EnableHTTPS:      getEnvAsBool("ENABLE_HTTPS", false),
		Domain:           getEnv("DOMAIN", "localhost"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		DBDriver:         getEnv("DB_DRIVER", "sqlite3"),
		MaxRequestSize:   getEnvAsInt64("MAX_REQUEST_SIZE", 10*1024*1024), // 10MB default
		ReadTimeout:      getEnvAsInt("READ_TIMEOUT", 30),
		WriteTimeout:     getEnvAsInt("WRITE_TIMEOUT", 30),
		KeepAliveTimeout: getEnvAsInt("KEEP_ALIVE_TIMEOUT", 300),

		// Security defaults
		EncryptionKey:  getEnv("ENCRYPTION_KEY", ""),
		Salt:           getEnv("SALT", ""),
		CORSOrigins:    getEnvAsSlice("CORS_ORIGINS", []string{"http://localhost:3000"}),
		AllowedHosts:   getEnvAsSlice("ALLOWED_HOSTS", []string{"localhost", "127.0.0.1"}),
		TrustedProxies: getEnvAsSlice("TRUSTED_PROXIES", []string{"127.0.0.1"}),
		SessionSecret:  getEnv("SESSION_SECRET", ""),
		CookieSecret:   getEnv("COOKIE_SECRET", ""),

		// Database security
		DBPassword:       getEnv("DB_PASSWORD", ""),
		DBSSLMode:        getEnv("DB_SSL_MODE", "require"),
		DBMaxConnections: getEnvAsInt("DB_MAX_CONNECTIONS", 25),

		// API security
		APIKeyLength:       getEnvAsInt("API_KEY_LENGTH", 32),
		TokenExpiry:        getEnvAsInt("TOKEN_EXPIRY", 3600),          // 1 hour
		RefreshTokenExpiry: getEnvAsInt("REFRESH_TOKEN_EXPIRY", 86400), // 24 hours

		// Monitoring
		MonitoringEnabled: getEnvAsBool("MONITORING_ENABLED", true),
		SentryDSN:         getEnv("SENTRY_DSN", ""),
		AlertEmail:        getEnv("ALERT_EMAIL", ""),
	}

	// Generate secure defaults if not provided
	config.generateSecureDefaults()

	// Validate required environment variables
	validateConfig(config)

	return config
}

func (c *Config) generateSecureDefaults() {
	// Generate JWT secret if not provided
	if c.JWTSecret == "" {
		log.Println("WARNING: JWT_SECRET not provided, generating secure random secret")
		c.JWTSecret = generateSecureSecret(32)
	}

	// Generate encryption key if not provided
	if c.EncryptionKey == "" {
		log.Println("WARNING: ENCRYPTION_KEY not provided, generating secure random key")
		c.EncryptionKey = generateSecureSecret(32)
	}

	// Generate salt if not provided
	if c.Salt == "" {
		log.Println("WARNING: SALT not provided, generating secure random salt")
		c.Salt = generateSecureSecret(16)
	}

	// Generate session secret if not provided
	if c.SessionSecret == "" {
		log.Println("WARNING: SESSION_SECRET not provided, generating secure random secret")
		c.SessionSecret = generateSecureSecret(32)
	}

	// Generate cookie secret if not provided
	if c.CookieSecret == "" {
		log.Println("WARNING: COOKIE_SECRET not provided, generating secure random secret")
		c.CookieSecret = generateSecureSecret(24)
	}
}

func generateSecureSecret(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("Failed to generate secure secret: %v", err)
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value, exists := os.LookupEnv(key); exists {
		if value == "" {
			return defaultValue
		}
		// Split comma-separated values and trim whitespace
		items := strings.Split(value, ",")
		result := make([]string, 0, len(items))
		for _, item := range items {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return defaultValue
}

func validateConfig(config Config) {
	// Validate port
	if port, err := strconv.Atoi(config.Port); err != nil || port < 1 || port > 65535 {
		log.Printf("WARNING: Invalid port number '%s', using default 8080", config.Port)
		config.Port = "8080"
	}

	// Validate rate limit
	if config.APIRateLimit < 1 {
		log.Printf("WARNING: Invalid API rate limit '%d', using default 100", config.APIRateLimit)
		config.APIRateLimit = 100
	}

	// Validate max request size
	if config.MaxRequestSize < 1024 { // Minimum 1KB
		log.Printf("WARNING: Invalid max request size '%d', using default 10MB", config.MaxRequestSize)
		config.MaxRequestSize = 10 * 1024 * 1024
	}

	// Validate API key length
	if config.APIKeyLength < 16 {
		log.Printf("WARNING: API key length too short '%d', using minimum 16", config.APIKeyLength)
		config.APIKeyLength = 16
	}

	// Validate token expiry
	if config.TokenExpiry < 300 { // Minimum 5 minutes
		log.Printf("WARNING: Token expiry too short '%d', using minimum 300 seconds", config.TokenExpiry)
		config.TokenExpiry = 300
	}

	// Validate database SSL mode
	validSSLModes := []string{"disable", "require", "verify-ca", "verify-full"}
	validSSLMode := false
	for _, mode := range validSSLModes {
		if config.DBSSLMode == mode {
			validSSLMode = true
			break
		}
	}
	if !validSSLMode {
		log.Printf("WARNING: Invalid DB SSL mode '%s', using 'require'", config.DBSSLMode)
		config.DBSSLMode = "require"
	}

	// Security warnings
	if config.EnableHTTPS && config.Domain == "localhost" {
		log.Println("WARNING: HTTPS enabled but domain is localhost - please update domain for production")
	}

	if config.JWTSecret == "" {
		log.Fatal("CRITICAL: JWT secret is required but not configured")
	}

	log.Printf("Configuration loaded successfully: Port=%s, DB=%s, RateLimit=%d, HTTPS=%t",
		config.Port, config.DBDriver, config.APIRateLimit, config.EnableHTTPS)
}
