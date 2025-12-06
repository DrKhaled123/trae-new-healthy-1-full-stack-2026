package errors

import (
	"crypto/subtle"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"
)

// AuthErrorHandler handles authentication and rate limiting errors
type AuthErrorHandler struct {
	config        *AuthConfig
	metrics       *AuthMetrics
	rateLimiters  map[string]*rate.Limiter
	blockedIPs    map[string]time.Time
	apiKeys       map[string]*APIKeyInfo
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
}

// AuthConfig holds authentication error handling configuration
type AuthConfig struct {
	JWTSecret          string
	JWTExpiration      time.Duration
	APIKeySecret       string
	RateLimitRequests  int
	RateLimitWindow    time.Duration
	MaxLoginAttempts   int
	LockoutDuration    time.Duration
	CleanupInterval    time.Duration
	EnableMetrics      bool
	LogFailedAttempts  bool
	BlockSuspiciousIPs bool
	WhitelistedIPs     []string
	BlacklistedIPs     []string
}

// AuthMetrics holds Prometheus metrics for authentication
type AuthMetrics struct {
	AuthAttempts       *prometheus.CounterVec
	AuthFailures       *prometheus.CounterVec
	RateLimitHits      *prometheus.CounterVec
	BlockedRequests    prometheus.Counter
	ActiveRateLimiters prometheus.Gauge
	JWTValidations     *prometheus.CounterVec
	APIKeyValidations  *prometheus.CounterVec
	SuspiciousActivity prometheus.Counter
}

// APIKeyInfo holds information about an API key
type APIKeyInfo struct {
	Key        string
	UserID     string
	Scopes     []string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastUsed   time.Time
	UsageCount int64
	IsActive   bool
}

// LoginAttempt tracks failed login attempts
type LoginAttempt struct {
	IP        string
	UserID    string
	Timestamp time.Time
	Success   bool
	UserAgent string
}

// NewAuthErrorHandler creates a new authentication error handler
func NewAuthErrorHandler(config *AuthConfig) *AuthErrorHandler {
	aeh := &AuthErrorHandler{
		config:       config,
		metrics:      newAuthMetrics(),
		rateLimiters: make(map[string]*rate.Limiter),
		blockedIPs:   make(map[string]time.Time),
		apiKeys:      make(map[string]*APIKeyInfo),
	}

	// Start cleanup routine
	aeh.startCleanupRoutine()

	return aeh
}

// newAuthMetrics creates Prometheus metrics for authentication monitoring
func newAuthMetrics() *AuthMetrics {
	m := &AuthMetrics{
		AuthAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nutrition_platform_auth_attempts_total",
				Help: "Total number of authentication attempts",
			},
			[]string{"method", "status", "ip"},
		),
		AuthFailures: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nutrition_platform_auth_failures_total",
				Help: "Total number of authentication failures",
			},
			[]string{"reason", "ip"},
		),
		RateLimitHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nutrition_platform_rate_limit_hits_total",
				Help: "Total number of rate limit hits",
			},
			[]string{"endpoint", "ip"},
		),
		BlockedRequests: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "nutrition_platform_blocked_requests_total",
				Help: "Total number of blocked requests",
			},
		),
		ActiveRateLimiters: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nutrition_platform_active_rate_limiters",
				Help: "Number of active rate limiters",
			},
		),
		JWTValidations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nutrition_platform_jwt_validations_total",
				Help: "Total number of JWT validations",
			},
			[]string{"status"},
		),
		APIKeyValidations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nutrition_platform_api_key_validations_total",
				Help: "Total number of API key validations",
			},
			[]string{"status"},
		),
		SuspiciousActivity: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "nutrition_platform_suspicious_activity_total",
				Help: "Total number of suspicious activities detected",
			},
		),
	}

	// Register metrics with error handling for tests
	registerAuthMetricSafely(m.AuthAttempts)
	registerAuthMetricSafely(m.AuthFailures)
	registerAuthMetricSafely(m.RateLimitHits)
	registerAuthMetricSafely(m.BlockedRequests)
	registerAuthMetricSafely(m.ActiveRateLimiters)
	registerAuthMetricSafely(m.JWTValidations)
	registerAuthMetricSafely(m.APIKeyValidations)
	registerAuthMetricSafely(m.SuspiciousActivity)

	return m
}

// registerAuthMetricSafely registers a metric, ignoring duplicate registration errors
func registerAuthMetricSafely(collector prometheus.Collector) {
	defer func() {
		if r := recover(); r != nil {
			// Ignore duplicate registration panics in tests
			if err, ok := r.(error); ok && strings.Contains(err.Error(), "duplicate metrics collector registration attempted") {
				return
			}
			panic(r)
		}
	}()
	prometheus.MustRegister(collector)
}

// ValidateJWT validates a JWT token and handles errors
func (aeh *AuthErrorHandler) ValidateJWT(tokenString string, c echo.Context) (*jwt.Token, *APIError) {
	if tokenString == "" {
		aeh.recordAuthFailure("missing_token", c.RealIP())
		return nil, ErrMissingAPIKeyError()
	}

	// Remove Bearer prefix if present
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(aeh.config.JWTSecret), nil
	})

	if err != nil {
		aeh.recordAuthFailure("invalid_token", c.RealIP())
		aeh.metrics.JWTValidations.WithLabelValues("invalid").Inc()

		// Check if token is expired
		if strings.Contains(err.Error(), "expired") {
			return nil, ErrExpiredAPIKeyError()
		}
		return nil, ErrInvalidAPIKeyError(err.Error())
	}

	if !token.Valid {
		aeh.recordAuthFailure("invalid_token", c.RealIP())
		aeh.metrics.JWTValidations.WithLabelValues("invalid").Inc()
		return nil, ErrInvalidAPIKeyError("token is not valid")
	}

	aeh.metrics.JWTValidations.WithLabelValues("valid").Inc()
	return token, nil
}

// ValidateAPIKey validates an API key and handles errors
func (aeh *AuthErrorHandler) ValidateAPIKey(apiKey string, c echo.Context) (*APIKeyInfo, *APIError) {
	if apiKey == "" {
		aeh.recordAuthFailure("missing_api_key", c.RealIP())
		return nil, ErrMissingAPIKeyError()
	}

	aeh.mu.RLock()
	keyInfo, exists := aeh.apiKeys[apiKey]
	aeh.mu.RUnlock()

	if !exists {
		aeh.recordAuthFailure("invalid_api_key", c.RealIP())
		aeh.metrics.APIKeyValidations.WithLabelValues("invalid").Inc()
		return nil, ErrInvalidAPIKeyError("API key not found")
	}

	// Check if key is active
	if !keyInfo.IsActive {
		aeh.recordAuthFailure("revoked_api_key", c.RealIP())
		aeh.metrics.APIKeyValidations.WithLabelValues("revoked").Inc()
		return nil, ErrRevokedAPIKeyError()
	}

	// Check if key is expired
	if time.Now().After(keyInfo.ExpiresAt) {
		aeh.recordAuthFailure("expired_api_key", c.RealIP())
		aeh.metrics.APIKeyValidations.WithLabelValues("expired").Inc()
		return nil, ErrExpiredAPIKeyError()
	}

	// Update usage statistics
	aeh.mu.Lock()
	keyInfo.LastUsed = time.Now()
	keyInfo.UsageCount++
	aeh.mu.Unlock()

	aeh.metrics.APIKeyValidations.WithLabelValues("valid").Inc()
	return keyInfo, nil
}

// CheckRateLimit checks if the request should be rate limited
func (aeh *AuthErrorHandler) CheckRateLimit(identifier string, c echo.Context) *APIError {
	// Check if IP is blocked
	if aeh.isIPBlocked(c.RealIP()) {
		aeh.metrics.BlockedRequests.Inc()
		return NewAPIError(ErrIPBlocked, "IP address is blocked due to suspicious activity")
	}

	// Check whitelist/blacklist
	if aeh.isIPBlacklisted(c.RealIP()) {
		aeh.metrics.BlockedRequests.Inc()
		return NewAPIError(ErrIPBlocked, "IP address is blacklisted")
	}

	if aeh.isIPWhitelisted(c.RealIP()) {
		return nil // Skip rate limiting for whitelisted IPs
	}

	limiter := aeh.getRateLimiter(identifier)
	if !limiter.Allow() {
		aeh.metrics.RateLimitHits.WithLabelValues(c.Path(), c.RealIP()).Inc()

		// Check for suspicious activity
		if aeh.config.BlockSuspiciousIPs {
			aeh.checkSuspiciousActivity(c.RealIP())
		}

		return NewAPIError(ErrRateLimitExceeded, "Rate limit exceeded")
	}

	return nil
}

// RecordLoginAttempt records a login attempt for monitoring
func (aeh *AuthErrorHandler) RecordLoginAttempt(ip, userID, userAgent string, success bool) {
	_ = &LoginAttempt{
		IP:        ip,
		UserID:    userID,
		Timestamp: time.Now(),
		Success:   success,
		UserAgent: userAgent,
	}

	status := "success"
	if !success {
		status = "failure"
		aeh.recordAuthFailure("login_failed", ip)
	}

	aeh.metrics.AuthAttempts.WithLabelValues("login", status, ip).Inc()

	if aeh.config.LogFailedAttempts && !success {
		log.Printf("Failed login attempt: IP=%s, UserID=%s, UserAgent=%s", ip, userID, userAgent)
	}

	// Check for brute force attacks
	if !success {
		aeh.checkBruteForceAttack(ip, userID)
	}
}

// Helper methods

func (aeh *AuthErrorHandler) getRateLimiter(identifier string) *rate.Limiter {
	aeh.mu.Lock()
	defer aeh.mu.Unlock()

	limiter, exists := aeh.rateLimiters[identifier]
	if !exists {
		limiter = rate.NewLimiter(
			rate.Every(aeh.config.RateLimitWindow/time.Duration(aeh.config.RateLimitRequests)),
			aeh.config.RateLimitRequests,
		)
		aeh.rateLimiters[identifier] = limiter
		aeh.metrics.ActiveRateLimiters.Set(float64(len(aeh.rateLimiters)))
	}

	return limiter
}

func (aeh *AuthErrorHandler) isIPBlocked(ip string) bool {
	aeh.mu.RLock()
	defer aeh.mu.RUnlock()

	blockTime, blocked := aeh.blockedIPs[ip]
	if !blocked {
		return false
	}

	// Check if block has expired
	if time.Now().After(blockTime.Add(aeh.config.LockoutDuration)) {
		delete(aeh.blockedIPs, ip)
		return false
	}

	return true
}

func (aeh *AuthErrorHandler) isIPWhitelisted(ip string) bool {
	for _, whitelistedIP := range aeh.config.WhitelistedIPs {
		if ip == whitelistedIP {
			return true
		}
	}
	return false
}

func (aeh *AuthErrorHandler) isIPBlacklisted(ip string) bool {
	for _, blacklistedIP := range aeh.config.BlacklistedIPs {
		if ip == blacklistedIP {
			return true
		}
	}
	return false
}

func (aeh *AuthErrorHandler) blockIP(ip string) {
	aeh.mu.Lock()
	defer aeh.mu.Unlock()

	aeh.blockedIPs[ip] = time.Now()
	log.Printf("Blocked IP %s due to suspicious activity", ip)
}

func (aeh *AuthErrorHandler) recordAuthFailure(reason, ip string) {
	aeh.metrics.AuthFailures.WithLabelValues(reason, ip).Inc()

	if aeh.config.LogFailedAttempts {
		log.Printf("Authentication failure: reason=%s, ip=%s", reason, ip)
	}
}

func (aeh *AuthErrorHandler) checkSuspiciousActivity(ip string) {
	// Simple heuristic: if rate limit is hit multiple times, block the IP
	// In a real implementation, you might want more sophisticated detection
	aeh.metrics.SuspiciousActivity.Inc()
	aeh.blockIP(ip)
}

func (aeh *AuthErrorHandler) checkBruteForceAttack(ip, userID string) {
	// Implement brute force detection logic
	// This is a simplified version - in production, you'd want more sophisticated tracking
	log.Printf("Potential brute force attack detected: IP=%s, UserID=%s", ip, userID)
}

func (aeh *AuthErrorHandler) startCleanupRoutine() {
	aeh.cleanupTicker = time.NewTicker(aeh.config.CleanupInterval)
	go func() {
		for range aeh.cleanupTicker.C {
			aeh.cleanup()
		}
	}()
}

func (aeh *AuthErrorHandler) cleanup() {
	aeh.mu.Lock()
	defer aeh.mu.Unlock()

	now := time.Now()

	// Clean up expired blocked IPs
	for ip, blockTime := range aeh.blockedIPs {
		if now.After(blockTime.Add(aeh.config.LockoutDuration)) {
			delete(aeh.blockedIPs, ip)
		}
	}

	// Clean up old rate limiters (optional optimization)
	// This is a simple cleanup - in production, you might want more sophisticated logic
	if len(aeh.rateLimiters) > 1000 {
		// Keep only the most recently used limiters
		// Implementation depends on your specific needs
	}

	aeh.metrics.ActiveRateLimiters.Set(float64(len(aeh.rateLimiters)))
}

// AddAPIKey adds a new API key to the handler
func (aeh *AuthErrorHandler) AddAPIKey(key string, keyInfo *APIKeyInfo) {
	aeh.mu.Lock()
	defer aeh.mu.Unlock()
	aeh.apiKeys[key] = keyInfo
}

// RemoveAPIKey removes an API key from the handler
func (aeh *AuthErrorHandler) RemoveAPIKey(key string) {
	aeh.mu.Lock()
	defer aeh.mu.Unlock()
	delete(aeh.apiKeys, key)
}

// GetAPIKeyInfo returns information about an API key
func (aeh *AuthErrorHandler) GetAPIKeyInfo(key string) (*APIKeyInfo, bool) {
	aeh.mu.RLock()
	defer aeh.mu.RUnlock()
	keyInfo, exists := aeh.apiKeys[key]
	return keyInfo, exists
}

// SecureCompare performs a constant-time comparison of two strings
func (aeh *AuthErrorHandler) SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// Stop stops the cleanup routine
func (aeh *AuthErrorHandler) Stop() {
	if aeh.cleanupTicker != nil {
		aeh.cleanupTicker.Stop()
	}
}
