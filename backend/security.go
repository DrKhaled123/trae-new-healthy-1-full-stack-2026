// security.go
package main

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
)

func SecurityMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Set security headers
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			c.Response().Header().Set("X-XSS-Protection", "1; mode=block")
			c.Response().Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

			// Remove server info
			c.Response().Header().Del("Server")

			// Validate content type for POST/PUT requests
			if c.Request().Method == "POST" || c.Request().Method == "PUT" || c.Request().Method == "PATCH" {
				contentType := c.Request().Header.Get("Content-Type")
				if contentType == "" {
					contentType = "application/json"
				}

				validTypes := []string{
					"application/json",
					"multipart/form-data",
					"application/x-www-form-urlencoded",
				}

				isValid := false
				for _, validType := range validTypes {
					if strings.Contains(contentType, validType) {
						isValid = true
						break
					}
				}

				if !isValid {
					return c.JSON(http.StatusUnsupportedMediaType, map[string]interface{}{
						"error":   "Unsupported Content-Type",
						"message": "Content-Type must be one of: application/json, multipart/form-data, application/x-www-form-urlencoded",
					})
				}
			}

			// Rate limiting (we'll have this handled separately, but add logging here)
			userAgent := c.Request().UserAgent()

			// Check for suspicious user agents or headers
			if isSuspiciousRequest(userAgent, c.Request().Header) {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "Suspicious request detected",
				})
			}

			// Add request timing for DoS protection
			start := time.Now()
			defer func() {
				duration := time.Since(start)
				if duration > 10*time.Second {
					// Log potential slow loris attack
				}
			}()

			return next(c)
		}
	}
}

func isSuspiciousRequest(userAgent string, headers http.Header) bool {
	suspiciousAgents := []string{
		"sqlmap",
		"nmap",
		"nikto",
		"dirbuster",
		"gobuster",
		"hydra",
		"metasploit",
		"acunetix",
		"openvas",
		"wpscan",
	}

	userAgent = strings.ToLower(userAgent)
	for _, suspicious := range suspiciousAgents {
		if strings.Contains(userAgent, suspicious) {
			return true
		}
	}

	// Check for suspicious headers
	if headers.Get("X-Requested-With") == "XMLHttpRequest" &&
		strings.Contains(headers.Get("Referer"), "localhost") {
		// Allow legitimate development requests
		return false
	}

	return false
}

// AuthMiddleware validates JWT tokens
func AuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "Authorization header required",
				})
			}

			// Remove "Bearer " prefix if present
			token := authHeader
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}

			// Validate token (implement your JWT validation logic)
			userId, err := ValidateJWTToken(token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error":   "Invalid token",
					"details": err.Error(),
				})
			}

			// Store user ID in context
			c.Set("userId", userId)
			return next(c)
		}
	}
}

func ValidateJWTToken(token string) (string, error) {
	// Get JWT secret from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return "", echo.NewHTTPError(http.StatusInternalServerError, "JWT secret not configured")
	}

	// Parse and validate token
	jwtToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Return secret key for verification
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "Invalid token: "+err.Error())
	}

	// Extract claims
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok || !jwtToken.Valid {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "Invalid token claims")
	}

	// Validate expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return "", echo.NewHTTPError(http.StatusUnauthorized, "Token expired")
		}
	}

	// Extract user ID
	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "Invalid user ID in token")
	}

	// Validate required fields
	if role, ok := claims["role"].(string); ok {
		if !isValidRole(role) {
			return "", echo.NewHTTPError(http.StatusForbidden, "Invalid role in token")
		}
	}

	return userID, nil
}

func isValidRole(role string) bool {
	validRoles := []string{"admin", "user", "nutritionist", "doctor", "guest"}
	for _, validRole := range validRoles {
		if role == validRole {
			return true
		}
	}
	return false
}

// Role-Based Access Control Middleware
func RoleBasedAccessControlMiddleware(requiredRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get user role from JWT token context
			userRole, ok := c.Get("user_role").(string)
			if !ok {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "User role not found in token",
				})
			}

			// Check if user has required role
			hasAccess := false
			for _, requiredRole := range requiredRoles {
				if userRole == requiredRole {
					hasAccess = true
					break
				}
			}

			if !hasAccess {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":          "Insufficient permissions",
					"required_roles": requiredRoles,
					"user_role":      userRole,
				})
			}

			return next(c)
		}
	}
}

// API Key Verification Middleware
func APIKeyAuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := c.Request().Header.Get("X-API-Key")
			if apiKey == "" {
				// API key is optional for some endpoints
				return next(c)
			}

			// Validate API key
			valid, userID, role := validateAPIKey(apiKey)
			if !valid {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "Invalid API key",
				})
			}

			// Set user context
			c.Set("user_id", userID)
			c.Set("user_role", role)
			c.Set("auth_method", "api_key")

			return next(c)
		}
	}
}

func validateAPIKey(apiKey string) (bool, string, string) {
	// TODO: Implement API key validation against database
	// For now, return false to require JWT authentication
	return false, "", ""
}

// BasicAuthMiddleware for admin routes
func BasicAuthMiddleware(username, password string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, pass, hasAuth := c.Request().BasicAuth()

			if !hasAuth ||
				subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 ||
				subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
				c.Response().Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "Unauthorized",
				})
			}

			return next(c)
		}
	}
}

// Input validation functions
func SanitizeUsername(username string) string {
	// Allow only alphanumeric characters, underscores, and hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	return reg.ReplaceAllString(username, "")
}

func SanitizeEmail(email string) string {
	// Basic email sanitization - remove potentially dangerous characters
	reg := regexp.MustCompile(`[<>'"\\]`)
	return reg.ReplaceAllString(email, "")
}

func ValidatePasswordStrength(password string) []string {
	var errors []string

	if len(password) < 8 {
		errors = append(errors, "Password must be at least 8 characters long")
	}

	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		errors = append(errors, "Password must contain at least one uppercase letter")
	}

	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		errors = append(errors, "Password must contain at least one lowercase letter")
	}

	if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		errors = append(errors, "Password must contain at least one number")
	}

	if !regexp.MustCompile(`[^a-zA-Z0-9\s]`).MatchString(password) {
		errors = append(errors, "Password must contain at least one special character")
	}

	return errors
}

// XSS Prevention middleware
func XSSPreventionMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Sanitize query parameters
			for _, values := range c.Request().URL.Query() {
				for _, value := range values {
					if isXSSAttempt(value) {
						return c.JSON(http.StatusBadRequest, map[string]interface{}{
							"error": "Potential XSS attempt detected",
						})
					}
				}
			}

			return next(c)
		}
	}
}

func isXSSAttempt(input string) bool {
	xssPatterns := []string{
		"<script",
		"javascript:",
		"onload=",
		"onerror=",
		"<iframe",
		"data:text/html",
		"<object",
		"<embed",
		"<form",
		"eval(",
		"document.cookie",
		"<svg",
	}

	input = strings.ToLower(input)
	for _, pattern := range xssPatterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}

	return false
}

// Request validation middleware
func RequestValidationMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check request size
			if c.Request().ContentLength > 10*1024*1024 { // 10MB limit
				return c.JSON(http.StatusRequestEntityTooLarge, map[string]interface{}{
					"error": "Request too large",
				})
			}

			// Check for invalid UTF-8 in headers
			for headerName, values := range c.Request().Header {
				for _, value := range values {
					if !isValidUTF8(value) {
						return c.JSON(http.StatusBadRequest, map[string]interface{}{
							"error": "Invalid characters in request header: " + headerName,
						})
					}
				}
			}

			return next(c)
		}
	}
}

func isValidUTF8(str string) bool {
	return len(str) == len([]rune(str))
}

// SQL Injection prevention (additional layer)
func SQLInjectionPreventionMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check URL path for SQL injection patterns
			if hasSQLInjection(c.Request().URL.Path) {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Invalid request path",
				})
			}

			// Check query parameters for SQL injection
			for _, values := range c.Request().URL.Query() {
				for _, value := range values {
					if hasSQLInjection(value) {
						return c.JSON(http.StatusBadRequest, map[string]interface{}{
							"error": "Invalid query parameter",
						})
					}
				}
			}

			return next(c)
		}
	}
}

func hasSQLInjection(str string) bool {
	sqlInjectionPatterns := []string{
		"union select",
		"union all",
		"select * from",
		"drop table",
		"delete from",
		"update .* set",
		"insert into",
		"script>",
		"javascript:",
		"vbscript:",
		"onload=",
		"onerror=",
		"<iframe",
		"<object",
		"<embed",
		"eval(",
		"document\\.cookie",
	}

	str = strings.ToLower(str)
	for _, pattern := range sqlInjectionPatterns {
		if strings.Contains(str, pattern) {
			return true
		}
	}

	// Check for common SQL comment patterns
	if strings.Contains(str, "--") || strings.Contains(str, "/*") || strings.Contains(str, "*/") {
		return true
	}

	return false
}
