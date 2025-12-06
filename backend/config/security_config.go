package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// SecurityConfig holds all security-related configuration
type SecurityConfig struct {
	// API Key Configuration
	APIKey APIKeyConfig `json:"api_key"`

	// Rate Limiting Configuration
	RateLimit RateLimitConfig `json:"rate_limit"`

	// Authentication Configuration
	Auth AuthConfig `json:"auth"`

	// Encryption Configuration
	Encryption EncryptionConfig `json:"encryption"`

	// Logging Configuration
	Logging LoggingConfig `json:"logging"`

	// CORS Configuration
	CORS CORSConfig `json:"cors"`

	// Security Headers Configuration
	Headers SecurityHeadersConfig `json:"headers"`

	// Input Validation Configuration
	Validation ValidationConfig `json:"validation"`

	// Monitoring Configuration
	Monitoring MonitoringConfig `json:"monitoring"`

	// Environment Configuration
	Environment EnvironmentConfig `json:"environment"`
}

// APIKeyConfig holds API key related settings
type APIKeyConfig struct {
	Enabled           bool          `json:"enabled"`
	KeyLength         int           `json:"key_length"`
	Prefix            string        `json:"prefix"`
	DefaultExpiration time.Duration `json:"default_expiration"`
	MaxExpiration     time.Duration `json:"max_expiration"`
	HashAlgorithm     string        `json:"hash_algorithm"`
	SaltLength        int           `json:"salt_length"`
	MinEntropyBits    int           `json:"min_entropy_bits"`
	RotationInterval  time.Duration `json:"rotation_interval"`
	MaxKeysPerUser    int           `json:"max_keys_per_user"`
}

// RateLimitConfig holds rate limiting settings
type RateLimitConfig struct {
	Enabled         bool                 `json:"enabled"`
	GlobalLimit     RateLimit            `json:"global_limit"`
	PerAPIKeyLimit  RateLimit            `json:"per_api_key_limit"`
	PerIPLimit      RateLimit            `json:"per_ip_limit"`
	EndpointLimits  map[string]RateLimit `json:"endpoint_limits"`
	BurstMultiplier float64              `json:"burst_multiplier"`
	CleanupInterval time.Duration        `json:"cleanup_interval"`
	BlockDuration   time.Duration        `json:"block_duration"`
	WhitelistedIPs  []string             `json:"whitelisted_ips"`
	BlacklistedIPs  []string             `json:"blacklisted_ips"`
}

// RateLimit defines rate limiting parameters
type RateLimit struct {
	Requests int           `json:"requests"`
	Window   time.Duration `json:"window"`
	Burst    int           `json:"burst"`
}

// AuthConfig holds authentication settings
type AuthConfig struct {
	JWTSecret              string        `json:"jwt_secret"`
	JWTExpiration          time.Duration `json:"jwt_expiration"`
	RefreshTokenExpiration time.Duration `json:"refresh_token_expiration"`
	PasswordMinLength      int           `json:"password_min_length"`
	PasswordRequireUpper   bool          `json:"password_require_upper"`
	PasswordRequireLower   bool          `json:"password_require_lower"`
	PasswordRequireDigit   bool          `json:"password_require_digit"`
	PasswordRequireSymbol  bool          `json:"password_require_symbol"`
	MaxLoginAttempts       int           `json:"max_login_attempts"`
	LockoutDuration        time.Duration `json:"lockout_duration"`
	SessionTimeout         time.Duration `json:"session_timeout"`
	TwoFactorEnabled       bool          `json:"two_factor_enabled"`
}

// EncryptionConfig holds encryption settings
type EncryptionConfig struct {
	Algorithm       string   `json:"algorithm"`
	KeySize         int      `json:"key_size"`
	EncryptionKey   string   `json:"encryption_key"`
	HashAlgorithm   string   `json:"hash_algorithm"`
	SaltLength      int      `json:"salt_length"`
	Iterations      int      `json:"iterations"`
	TLSMinVersion   string   `json:"tls_min_version"`
	TLSCipherSuites []string `json:"tls_cipher_suites"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level            string        `json:"level"`
	Format           string        `json:"format"`
	Output           string        `json:"output"`
	SecurityLogPath  string        `json:"security_log_path"`
	AuditLogPath     string        `json:"audit_log_path"`
	MaxFileSize      int64         `json:"max_file_size"`
	MaxBackups       int           `json:"max_backups"`
	MaxAge           time.Duration `json:"max_age"`
	Compress         bool          `json:"compress"`
	LogSensitiveData bool          `json:"log_sensitive_data"`
	LogStackTrace    bool          `json:"log_stack_trace"`
}

// CORSConfig holds CORS settings
type CORSConfig struct {
	Enabled          bool     `json:"enabled"`
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	ExposedHeaders   []string `json:"exposed_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"`
}

// SecurityHeadersConfig holds security headers settings
type SecurityHeadersConfig struct {
	Enabled                 bool   `json:"enabled"`
	ContentTypeOptions      string `json:"content_type_options"`
	FrameOptions            string `json:"frame_options"`
	XSSProtection           string `json:"xss_protection"`
	ReferrerPolicy          string `json:"referrer_policy"`
	ContentSecurityPolicy   string `json:"content_security_policy"`
	StrictTransportSecurity string `json:"strict_transport_security"`
	PermissionsPolicy       string `json:"permissions_policy"`
	CacheControl            string `json:"cache_control"`
	ServerHeader            string `json:"server_header"`
}

// ValidationConfig holds input validation settings
type ValidationConfig struct {
	Enabled               bool     `json:"enabled"`
	MaxInputLength        int      `json:"max_input_length"`
	MaxRequestSize        int64    `json:"max_request_size"`
	AllowedContentTypes   []string `json:"allowed_content_types"`
	BlockSQLInjection     bool     `json:"block_sql_injection"`
	BlockXSS              bool     `json:"block_xss"`
	BlockCommandInjection bool     `json:"block_command_injection"`
	BlockPathTraversal    bool     `json:"block_path_traversal"`
	SanitizeInput         bool     `json:"sanitize_input"`
	StrictValidation      bool     `json:"strict_validation"`
}

// MonitoringConfig holds monitoring and alerting settings
type MonitoringConfig struct {
	Enabled               bool                  `json:"enabled"`
	MetricsEnabled        bool                  `json:"metrics_enabled"`
	HealthCheckEnabled    bool                  `json:"health_check_enabled"`
	SecurityAlertsEnabled bool                  `json:"security_alerts_enabled"`
	AlertThresholds       AlertThresholds       `json:"alert_thresholds"`
	NotificationChannels  []NotificationChannel `json:"notification_channels"`
	MetricsRetention      time.Duration         `json:"metrics_retention"`
	AuditRetention        time.Duration         `json:"audit_retention"`
}

// AlertThresholds defines thresholds for security alerts
type AlertThresholds struct {
	FailedAuthAttempts  int           `json:"failed_auth_attempts"`
	RateLimitViolations int           `json:"rate_limit_violations"`
	SecurityViolations  int           `json:"security_violations"`
	ErrorRate           float64       `json:"error_rate"`
	ResponseTime        time.Duration `json:"response_time"`
	TimeWindow          time.Duration `json:"time_window"`
}

// NotificationChannel defines how alerts are sent
type NotificationChannel struct {
	Type     string            `json:"type"`     // email, webhook, slack, etc.
	Target   string            `json:"target"`   // email address, webhook URL, etc.
	Severity []string          `json:"severity"` // low, medium, high, critical
	Enabled  bool              `json:"enabled"`
	Config   map[string]string `json:"config"` // additional configuration
}

// EnvironmentConfig holds environment-specific settings
type EnvironmentConfig struct {
	Environment     string `json:"environment"` // development, staging, production
	DebugMode       bool   `json:"debug_mode"`
	TestMode        bool   `json:"test_mode"`
	SanitizeErrors  bool   `json:"sanitize_errors"`
	DetailedErrors  bool   `json:"detailed_errors"`
	ProfilerEnabled bool   `json:"profiler_enabled"`
}

// LoadSecurityConfig loads security configuration from environment variables
func LoadSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		APIKey:      loadAPIKeyConfig(),
		RateLimit:   loadRateLimitConfig(),
		Auth:        loadAuthConfig(),
		Encryption:  loadEncryptionConfig(),
		Logging:     loadLoggingConfig(),
		CORS:        loadCORSConfig(),
		Headers:     loadSecurityHeadersConfig(),
		Validation:  loadValidationConfig(),
		Monitoring:  loadMonitoringConfig(),
		Environment: loadEnvironmentConfig(),
	}
}

func loadAPIKeyConfig() APIKeyConfig {
	return APIKeyConfig{
		Enabled:           getBoolEnv("API_KEY_ENABLED", true),
		KeyLength:         getIntEnv("API_KEY_LENGTH", 32),
		Prefix:            getStringEnv("API_KEY_PREFIX", "np_"),
		DefaultExpiration: getDurationEnv("API_KEY_DEFAULT_EXPIRATION", 365*24*time.Hour),
		MaxExpiration:     getDurationEnv("API_KEY_MAX_EXPIRATION", 2*365*24*time.Hour),
		HashAlgorithm:     getStringEnv("API_KEY_HASH_ALGORITHM", "bcrypt"),
		SaltLength:        getIntEnv("API_KEY_SALT_LENGTH", 16),
		MinEntropyBits:    getIntEnv("API_KEY_MIN_ENTROPY_BITS", 128),
		RotationInterval:  getDurationEnv("API_KEY_ROTATION_INTERVAL", 90*24*time.Hour),
		MaxKeysPerUser:    getIntEnv("API_KEY_MAX_PER_USER", 5),
	}
}

func loadRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled: getBoolEnv("RATE_LIMIT_ENABLED", true),
		GlobalLimit: RateLimit{
			Requests: getIntEnv("RATE_LIMIT_GLOBAL_REQUESTS", 10000),
			Window:   getDurationEnv("RATE_LIMIT_GLOBAL_WINDOW", time.Hour),
			Burst:    getIntEnv("RATE_LIMIT_GLOBAL_BURST", 100),
		},
		PerAPIKeyLimit: RateLimit{
			Requests: getIntEnv("RATE_LIMIT_API_KEY_REQUESTS", 1000),
			Window:   getDurationEnv("RATE_LIMIT_API_KEY_WINDOW", time.Hour),
			Burst:    getIntEnv("RATE_LIMIT_API_KEY_BURST", 50),
		},
		PerIPLimit: RateLimit{
			Requests: getIntEnv("RATE_LIMIT_IP_REQUESTS", 100),
			Window:   getDurationEnv("RATE_LIMIT_IP_WINDOW", time.Hour),
			Burst:    getIntEnv("RATE_LIMIT_IP_BURST", 10),
		},
		BurstMultiplier: getFloat64Env("RATE_LIMIT_BURST_MULTIPLIER", 2.0),
		CleanupInterval: getDurationEnv("RATE_LIMIT_CLEANUP_INTERVAL", 5*time.Minute),
		BlockDuration:   getDurationEnv("RATE_LIMIT_BLOCK_DURATION", time.Hour),
		WhitelistedIPs:  getStringSliceEnv("RATE_LIMIT_WHITELISTED_IPS", []string{}),
		BlacklistedIPs:  getStringSliceEnv("RATE_LIMIT_BLACKLISTED_IPS", []string{}),
	}
}

func loadAuthConfig() AuthConfig {
	return AuthConfig{
		JWTSecret:              getStringEnv("JWT_SECRET", ""),
		JWTExpiration:          getDurationEnv("JWT_EXPIRATION", 24*time.Hour),
		RefreshTokenExpiration: getDurationEnv("REFRESH_TOKEN_EXPIRATION", 7*24*time.Hour),
		PasswordMinLength:      getIntEnv("PASSWORD_MIN_LENGTH", 8),
		PasswordRequireUpper:   getBoolEnv("PASSWORD_REQUIRE_UPPER", true),
		PasswordRequireLower:   getBoolEnv("PASSWORD_REQUIRE_LOWER", true),
		PasswordRequireDigit:   getBoolEnv("PASSWORD_REQUIRE_DIGIT", true),
		PasswordRequireSymbol:  getBoolEnv("PASSWORD_REQUIRE_SYMBOL", true),
		MaxLoginAttempts:       getIntEnv("MAX_LOGIN_ATTEMPTS", 5),
		LockoutDuration:        getDurationEnv("LOCKOUT_DURATION", 15*time.Minute),
		SessionTimeout:         getDurationEnv("SESSION_TIMEOUT", 30*time.Minute),
		TwoFactorEnabled:       getBoolEnv("TWO_FACTOR_ENABLED", false),
	}
}

func loadEncryptionConfig() EncryptionConfig {
	return EncryptionConfig{
		Algorithm:     getStringEnv("ENCRYPTION_ALGORITHM", "AES-256-GCM"),
		KeySize:       getIntEnv("ENCRYPTION_KEY_SIZE", 256),
		EncryptionKey: getStringEnv("ENCRYPTION_KEY", ""),
		HashAlgorithm: getStringEnv("HASH_ALGORITHM", "SHA-256"),
		SaltLength:    getIntEnv("SALT_LENGTH", 32),
		Iterations:    getIntEnv("HASH_ITERATIONS", 100000),
		TLSMinVersion: getStringEnv("TLS_MIN_VERSION", "1.2"),
		TLSCipherSuites: getStringSliceEnv("TLS_CIPHER_SUITES", []string{
			"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
			"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		}),
	}
}

func loadLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Level:            getStringEnv("LOG_LEVEL", "info"),
		Format:           getStringEnv("LOG_FORMAT", "json"),
		Output:           getStringEnv("LOG_OUTPUT", "stdout"),
		SecurityLogPath:  getStringEnv("SECURITY_LOG_PATH", "./logs/security.log"),
		AuditLogPath:     getStringEnv("AUDIT_LOG_PATH", "./logs/audit.log"),
		MaxFileSize:      getInt64Env("LOG_MAX_FILE_SIZE", 100*1024*1024), // 100MB
		MaxBackups:       getIntEnv("LOG_MAX_BACKUPS", 10),
		MaxAge:           getDurationEnv("LOG_MAX_AGE", 30*24*time.Hour),
		Compress:         getBoolEnv("LOG_COMPRESS", true),
		LogSensitiveData: getBoolEnv("LOG_SENSITIVE_DATA", false),
		LogStackTrace:    getBoolEnv("LOG_STACK_TRACE", true),
	}
}

func loadCORSConfig() CORSConfig {
	return CORSConfig{
		Enabled:          getBoolEnv("CORS_ENABLED", true),
		AllowedOrigins:   getStringSliceEnv("CORS_ALLOWED_ORIGINS", []string{"*"}),
		AllowedMethods:   getStringSliceEnv("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		AllowedHeaders:   getStringSliceEnv("CORS_ALLOWED_HEADERS", []string{"*"}),
		ExposedHeaders:   getStringSliceEnv("CORS_EXPOSED_HEADERS", []string{}),
		AllowCredentials: getBoolEnv("CORS_ALLOW_CREDENTIALS", false),
		MaxAge:           getIntEnv("CORS_MAX_AGE", 86400),
	}
}

func loadSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		Enabled:                 getBoolEnv("SECURITY_HEADERS_ENABLED", true),
		ContentTypeOptions:      getStringEnv("HEADER_CONTENT_TYPE_OPTIONS", "nosniff"),
		FrameOptions:            getStringEnv("HEADER_FRAME_OPTIONS", "DENY"),
		XSSProtection:           getStringEnv("HEADER_XSS_PROTECTION", "1; mode=block"),
		ReferrerPolicy:          getStringEnv("HEADER_REFERRER_POLICY", "strict-origin-when-cross-origin"),
		ContentSecurityPolicy:   getStringEnv("HEADER_CSP", "default-src 'self'"),
		StrictTransportSecurity: getStringEnv("HEADER_HSTS", "max-age=31536000; includeSubDomains"),
		PermissionsPolicy:       getStringEnv("HEADER_PERMISSIONS_POLICY", "geolocation=(), microphone=(), camera=()"),
		CacheControl:            getStringEnv("HEADER_CACHE_CONTROL", "no-cache, no-store, must-revalidate"),
		ServerHeader:            getStringEnv("HEADER_SERVER", "Nutrition-Platform/1.0"),
	}
}

func loadValidationConfig() ValidationConfig {
	return ValidationConfig{
		Enabled:               getBoolEnv("VALIDATION_ENABLED", true),
		MaxInputLength:        getIntEnv("VALIDATION_MAX_INPUT_LENGTH", 10000),
		MaxRequestSize:        getInt64Env("VALIDATION_MAX_REQUEST_SIZE", 10*1024*1024), // 10MB
		AllowedContentTypes:   getStringSliceEnv("VALIDATION_ALLOWED_CONTENT_TYPES", []string{"application/json", "multipart/form-data"}),
		BlockSQLInjection:     getBoolEnv("VALIDATION_BLOCK_SQL_INJECTION", true),
		BlockXSS:              getBoolEnv("VALIDATION_BLOCK_XSS", true),
		BlockCommandInjection: getBoolEnv("VALIDATION_BLOCK_COMMAND_INJECTION", true),
		BlockPathTraversal:    getBoolEnv("VALIDATION_BLOCK_PATH_TRAVERSAL", true),
		SanitizeInput:         getBoolEnv("VALIDATION_SANITIZE_INPUT", true),
		StrictValidation:      getBoolEnv("VALIDATION_STRICT", false),
	}
}

func loadMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		Enabled:               getBoolEnv("MONITORING_ENABLED", true),
		MetricsEnabled:        getBoolEnv("METRICS_ENABLED", true),
		HealthCheckEnabled:    getBoolEnv("HEALTH_CHECK_ENABLED", true),
		SecurityAlertsEnabled: getBoolEnv("SECURITY_ALERTS_ENABLED", true),
		AlertThresholds: AlertThresholds{
			FailedAuthAttempts:  getIntEnv("ALERT_FAILED_AUTH_THRESHOLD", 10),
			RateLimitViolations: getIntEnv("ALERT_RATE_LIMIT_THRESHOLD", 50),
			SecurityViolations:  getIntEnv("ALERT_SECURITY_THRESHOLD", 5),
			ErrorRate:           getFloat64Env("ALERT_ERROR_RATE_THRESHOLD", 0.05),
			ResponseTime:        getDurationEnv("ALERT_RESPONSE_TIME_THRESHOLD", 5*time.Second),
			TimeWindow:          getDurationEnv("ALERT_TIME_WINDOW", 5*time.Minute),
		},
		MetricsRetention: getDurationEnv("METRICS_RETENTION", 30*24*time.Hour),
		AuditRetention:   getDurationEnv("AUDIT_RETENTION", 90*24*time.Hour),
	}
}

func loadEnvironmentConfig() EnvironmentConfig {
	return EnvironmentConfig{
		Environment:     getStringEnv("ENVIRONMENT", "development"),
		DebugMode:       getBoolEnv("DEBUG_MODE", false),
		TestMode:        getBoolEnv("TEST_MODE", false),
		SanitizeErrors:  getBoolEnv("SANITIZE_ERRORS", true),
		DetailedErrors:  getBoolEnv("DETAILED_ERRORS", false),
		ProfilerEnabled: getBoolEnv("PROFILER_ENABLED", false),
	}
}

// Helper functions for environment variable parsing

func getStringEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if int64Value, err := strconv.ParseInt(value, 10, 64); err == nil {
			return int64Value
		}
	}
	return defaultValue
}

func getFloat64Env(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getStringSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// IsProduction returns true if running in production environment
func (c *SecurityConfig) IsProduction() bool {
	return c.Environment.Environment == "production"
}

// IsDevelopment returns true if running in development environment
func (c *SecurityConfig) IsDevelopment() bool {
	return c.Environment.Environment == "development"
}

// IsTestMode returns true if running in test mode
func (c *SecurityConfig) IsTestMode() bool {
	return c.Environment.TestMode
}

// ShouldSanitizeErrors returns true if errors should be sanitized
func (c *SecurityConfig) ShouldSanitizeErrors() bool {
	return c.Environment.SanitizeErrors && c.IsProduction()
}

// ShouldShowDetailedErrors returns true if detailed errors should be shown
func (c *SecurityConfig) ShouldShowDetailedErrors() bool {
	return c.Environment.DetailedErrors || c.IsDevelopment()
}
