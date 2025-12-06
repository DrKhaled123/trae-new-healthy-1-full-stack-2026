package middleware

import (
	"crypto/subtle"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

// NOTE: This file contains Gin-based middleware that is NOT currently used.
// The project uses Echo framework. This file is kept for reference but
// should be refactored to use Echo if needed in the future.
// For now, we comment out Gin-specific code to avoid compilation errors.

// SecurityConfig holds security configuration
type SecurityConfig struct {
	// Rate limiting
	RateLimitRequestsPerMinute int
	RateLimitBurst             int

	// JWT settings
	JWTSecret             string
	JWTAccessTokenExpiry  time.Duration
	JWTRefreshTokenExpiry time.Duration
	BCryptRounds          int

	// Security headers
	SecurityHeadersEnabled bool
	CORSEnabled            bool
	HelmetEnabled          bool
	CSPEnabled             bool
	CSPReportURI           string

	// API security
	APIKeyRequired bool
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string

	// Advanced security
	MaxRequestSize        int64
	MaxConcurrentRequests int
	RequestTimeout        time.Duration
	EnableIPWhitelist     bool
	AllowedIPs            []string
	EnableIPBlacklist     bool
	BlockedIPs            []string
}

// SecurityMiddleware provides comprehensive security features
type SecurityMiddleware struct {
	config         SecurityConfig
	ipRateLimiters map[string]*rate.Limiter
	ipMutex        sync.RWMutex
	globalLimiter  *rate.Limiter
	requestCount   int64
	requestMutex   sync.RWMutex
	blockedIPs     map[string]time.Time
	ipBlockMutex   sync.RWMutex
}

// NewSecurityMiddleware creates a new security middleware instance
func NewSecurityMiddleware(config SecurityConfig) *SecurityMiddleware {
	// Calculate global rate limiter based on system resources
	maxConcurrent := int64(config.MaxConcurrentRequests)
	if maxConcurrent == 0 {
		maxConcurrent = int64(runtime.NumCPU() * 1000)
	}

	return &SecurityMiddleware{
		config:         config,
		ipRateLimiters: make(map[string]*rate.Limiter),
		globalLimiter:  rate.NewLimiter(rate.Limit(maxConcurrent), config.MaxConcurrentRequests),
		blockedIPs:     make(map[string]time.Time),
	}
}

// SecurityHeaders adds comprehensive security headers to responses
// NOTE: This uses Gin framework - not compatible with Echo. Use security_headers.go instead.
/*
func (sm *SecurityMiddleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.config.SecurityHeadersEnabled {
			c.Next()
			return
		}

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions policy
		c.Header("Permissions-Policy",
			"geolocation=(), microphone=(), camera=(), payment=(), usb=(), "+
				"magnetometer=(), gyroscope=(), accelerometer=(), autoplay=(), "+
				"encrypted-media=(), fullscreen=(), picture-in-picture=()")

		// Strict Transport Security (HTTPS only)
		if c.Request.TLS != nil {
			maxAge := 31536000 // 1 year
			c.Header("Strict-Transport-Security", fmt.Sprintf("max-age=%d; includeSubDomains; preload", maxAge))
		}

		// Content Security Policy
		if sm.config.CSPEnabled {
			csp := sm.buildCSP(c)
			c.Header("Content-Security-Policy", csp)

			if sm.config.CSPReportURI != "" {
				c.Header("Content-Security-Policy-Report-Only", csp)
			}
		}

		// Remove server information
		c.Header("Server", "")

		// Cache control for sensitive endpoints
		if isSensitiveEndpoint(c.Request.URL.Path) {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}

		c.Next()
	}
}
*/

// RateLimiter implements advanced rate limiting with IP-based and global limits
// NOTE: This uses Gin framework - not compatible with Echo. Use enhanced_rate_limiter.go instead.
/*
func (sm *SecurityMiddleware) RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		// Check if IP is blocked
		if sm.isIPBlocked(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"status":  "error",
				"message": "IP address temporarily blocked due to suspicious activity",
				"code":    "IP_BLOCKED",
			})
			c.Abort()
			return
		}

		// Global rate limiting
		if !sm.globalLimiter.Allow() {
			sm.blockIP(clientIP, 5*time.Minute)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"status":  "error",
				"message": "Server overloaded, please try again later",
				"code":    "GLOBAL_RATE_LIMIT",
			})
			c.Abort()
			return
		}

		// IP-based rate limiting
		limiter := sm.getIPRateLimiter(clientIP)
		if !limiter.Allow() {
			// Block IP temporarily after repeated violations
			sm.incrementRequestCount(clientIP)
			if sm.shouldBlockIP(clientIP) {
				sm.blockIP(clientIP, 15*time.Minute)
			}

			c.JSON(http.StatusTooManyRequests, gin.H{
				"status":      "error",
				"message":     "Rate limit exceeded. Please try again later.",
				"code":        "RATE_LIMIT_EXCEEDED",
				"retry_after": sm.config.RateLimitRequestsPerMinute,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
*/

// IPWhitelist restricts access to whitelisted IP addresses
// NOTE: This uses Gin framework - not compatible with Echo.
/*
func (sm *SecurityMiddleware) IPWhitelist() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.config.EnableIPWhitelist {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		allowed := false

		for _, allowedIP := range sm.config.AllowedIPs {
			if clientIP == allowedIP {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "Access denied from this IP address",
				"code":    "IP_NOT_ALLOWED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
*/

// APIKeyAuth validates API keys for protected endpoints
// NOTE: This uses Gin framework - not compatible with Echo.
/*
func (sm *SecurityMiddleware) APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.config.APIKeyRequired {
			c.Next()
			return
		}

		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "API key required",
				"code":    "API_KEY_REQUIRED",
			})
			c.Abort()
			return
		}

		// Validate API key (implement your validation logic)
		if !sm.validateAPIKey(apiKey) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid API key",
				"code":    "INVALID_API_KEY",
			})
			c.Abort()
			return
		}

		c.Set("api_key", apiKey)
		c.Next()
	}
}
*/

// JWTAuth validates JWT tokens for protected endpoints
// NOTE: This uses Gin framework - not compatible with Echo. Use auth.go instead.
/*
func (sm *SecurityMiddleware) JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for public endpoints
		if isPublicEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Authorization header required",
				"code":    "AUTH_HEADER_REQUIRED",
			})
			c.Abort()
			return
		}

		// Extract Bearer token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid authorization header format",
				"code":    "INVALID_AUTH_FORMAT",
			})
			c.Abort()
			return
		}

		tokenString := tokenParts[1]
		claims := jwt.MapClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(sm.config.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid or expired token",
				"code":    "INVALID_TOKEN",
			})
			c.Abort()
			return
		}

		// Check token expiration
		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"status":  "error",
					"message": "Token has expired",
					"code":    "TOKEN_EXPIRED",
				})
				c.Abort()
				return
			}
		}

		// Set user context
		c.Set("user_id", claims["user_id"])
		c.Set("user_email", claims["email"])
		c.Set("user_role", claims["role"])

		c.Next()
	}
}
*/

// RequestSizeLimit limits the size of incoming requests
// NOTE: This uses Gin framework - not compatible with Echo.
/*
func (sm *SecurityMiddleware) RequestSizeLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if sm.config.MaxRequestSize > 0 {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, sm.config.MaxRequestSize)
		}
		c.Next()
	}
}
*/

// RequestTimeout sets a timeout for request processing
// NOTE: This uses Gin framework - not compatible with Echo.
/*
func (sm *SecurityMiddleware) RequestTimeout() gin.HandlerFunc {
	return func(c *gin.Context) {
		if sm.config.RequestTimeout > 0 {
			ctx, cancel := context.WithTimeout(c.Request.Context(), sm.config.RequestTimeout)
			defer cancel()
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	}
}
*/

// Helper methods

func (sm *SecurityMiddleware) buildCSP(c interface{}) string {
	// Build CSP based on environment and request
	// NOTE: Original Gin-specific code commented out
	/*
	if gin.Mode() == gin.DebugMode {
		return "default-src 'self' 'unsafe-inline' 'unsafe-eval' *; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: blob:; " +
			"font-src 'self' data:; " +
			"connect-src 'self' ws: wss:;"
	}

	return "default-src 'self'; " +
		"script-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data: https:; " +
		"font-src 'self' data:; " +
		"connect-src 'self' https:; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self';"
	*/
	// Default CSP for production
	return "default-src 'self'; " +
		"script-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data: https:; " +
		"font-src 'self' data:; " +
		"connect-src 'self' https:; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self';"
}

func (sm *SecurityMiddleware) getIPRateLimiter(ip string) *rate.Limiter {
	sm.ipMutex.Lock()
	defer sm.ipMutex.Unlock()

	limiter, exists := sm.ipRateLimiters[ip]
	if !exists {
		// Create rate limiter: requests per minute with burst
		ratePerSecond := rate.Limit(sm.config.RateLimitRequestsPerMinute / 60.0)
		limiter = rate.NewLimiter(ratePerSecond, sm.config.RateLimitBurst)
		sm.ipRateLimiters[ip] = limiter
	}

	return limiter
}

func (sm *SecurityMiddleware) isIPBlocked(ip string) bool {
	sm.ipBlockMutex.RLock()
	defer sm.ipBlockMutex.RUnlock()

	if blockTime, exists := sm.blockedIPs[ip]; exists {
		if time.Now().Before(blockTime) {
			return true
		}
		// Remove expired block
		delete(sm.blockedIPs, ip)
	}
	return false
}

func (sm *SecurityMiddleware) blockIP(ip string, duration time.Duration) {
	sm.ipBlockMutex.Lock()
	defer sm.ipBlockMutex.Unlock()
	sm.blockedIPs[ip] = time.Now().Add(duration)
}

func (sm *SecurityMiddleware) incrementRequestCount(ip string) {
	sm.requestMutex.Lock()
	defer sm.requestMutex.Unlock()
	sm.requestCount++
}

func (sm *SecurityMiddleware) shouldBlockIP(ip string) bool {
	// Implement logic to determine if IP should be blocked based on request patterns
	// This is a simplified version
	return sm.requestCount%100 == 0 // Block 1% of violating requests
}

func (sm *SecurityMiddleware) validateAPIKey(apiKey string) bool {
	// Implement your API key validation logic
	// This could involve database lookup, HMAC validation, etc.
	// For now, return false to require implementation
	return false
}

func isSensitiveEndpoint(path string) bool {
	sensitiveEndpoints := []string{
		"/api/v1/auth/",
		"/api/v1/users/",
		"/api/v1/admin/",
		"/api/v1/health",
	}

	for _, endpoint := range sensitiveEndpoints {
		if strings.HasPrefix(path, endpoint) {
			return true
		}
	}
	return false
}

func isPublicEndpoint(path string) bool {
	publicEndpoints := []string{
		"/health",
		"/metrics",
		"/api/v1/nutrition-data/recipes",
		"/api/v1/nutrition-data/workouts",
		"/api/v1/nutrition-data/diseases",
		"/api/v1/nutrition-data/injuries",
		"/docs",
		"/openapi.json",
	}

	for _, endpoint := range publicEndpoints {
		if strings.HasPrefix(path, endpoint) {
			return true
		}
	}
	return false
}

// GenerateJWTToken creates a new JWT token
func GenerateJWTToken(userID, email, role string, secret string, expiry time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(expiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// HashPassword securely hashes a password using bcrypt
func HashPassword(password string, rounds int) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), rounds)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// VerifyPassword verifies a password against its hash
func VerifyPassword(password, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// SecureCompare performs constant-time comparison of two strings
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// ValidatePasswordStrength checks if a password meets security requirements
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}
