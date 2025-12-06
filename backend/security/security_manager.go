package security

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

// SecurityManager provides comprehensive security features
type SecurityManager struct {
	redisClient  *redis.Client
	rateLimiters map[string]*rate.Limiter
	mu           sync.RWMutex
	config       SecurityConfig
	metrics      *SecurityManagerMetrics
	ipWhitelist  map[string]bool
	ipBlacklist  map[string]bool
	sanitizers   map[string]SanitizerFunc
	validators   map[string]ValidatorFunc
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	// CORS settings
	CORSAllowOrigins     []string
	CORSAllowMethods     []string
	CORSAllowHeaders     []string
	CORSExposeHeaders    []string
	CORSAllowCredentials bool
	CORSMaxAge           int

	// CSP settings
	CSPDirectives map[string]string
	CSPReportURI  string
	CSPReportOnly bool

	// Rate limiting
	DefaultRateLimit  int           // requests per minute
	BurstLimit        int           // burst capacity
	RateLimitWindow   time.Duration // time window
	DynamicAdjustment bool          // enable dynamic rate limit adjustment
	AdjustmentFactor  float64       // factor for dynamic adjustment

	// Input validation
	MaxRequestSize   int64    // maximum request body size
	MaxHeaderSize    int      // maximum header size
	AllowedFileTypes []string // allowed file upload types
	MaxFileSize      int64    // maximum file upload size

	// Security headers
	HSTSMaxAge            int  // HSTS max age in seconds
	HSTSIncludeSubdomains bool // include subdomains in HSTS
	HSTSPreload           bool // HSTS preload

	// IP filtering
	EnableIPFiltering   bool
	WhitelistEnabled    bool
	BlacklistEnabled    bool
	AutoBlockSuspicious bool
	BlockDuration       time.Duration

	// WAF rules
	WAFEnabled   bool
	WAFRules     []WAFRule
	WAFBlockMode bool // true for block, false for log only
}

// SecurityManagerMetrics tracks security manager specific metrics
type SecurityManagerMetrics struct {
	RequestsBlocked      int64
	RateLimitViolations  int64
	CSPViolations        int64
	XSSAttempts          int64
	SQLInjectionAttempts int64
	MaliciousUploads     int64
	SuspiciousIPs        int64
	WAFBlocks            int64
	mu                   sync.RWMutex
}

// WAFRule defines a Web Application Firewall rule
type WAFRule struct {
	ID          string
	Name        string
	Pattern     *regexp.Regexp
	Severity    string // "low", "medium", "high", "critical"
	Action      string // "block", "log", "challenge"
	Description string
	Enabled     bool
}

// SanitizerFunc defines a function for input sanitization
type SanitizerFunc func(input string) string

// ValidatorFunc defines a function for input validation
type ValidatorFunc func(input string) error

// SecurityViolation represents a security violation event
type SecurityViolation struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	Severity  string                 `json:"severity"`
	IP        string                 `json:"ip"`
	UserAgent string                 `json:"user_agent"`
	URL       string                 `json:"url"`
	Method    string                 `json:"method"`
	Payload   string                 `json:"payload,omitempty"`
	RuleID    string                 `json:"rule_id,omitempty"`
	Action    string                 `json:"action"`
	Metadata  map[string]interface{} `json:"metadata"`
	Blocked   bool                   `json:"blocked"`
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(redisClient *redis.Client, config SecurityConfig) *SecurityManager {
	sm := &SecurityManager{
		redisClient:  redisClient,
		rateLimiters: make(map[string]*rate.Limiter),
		config:       config,
		metrics:      &SecurityManagerMetrics{},
		ipWhitelist:  make(map[string]bool),
		ipBlacklist:  make(map[string]bool),
		sanitizers:   make(map[string]SanitizerFunc),
		validators:   make(map[string]ValidatorFunc),
	}

	// Register default sanitizers and validators
	sm.registerDefaultSanitizers()
	sm.registerDefaultValidators()
	sm.initializeWAFRules()

	return sm
}

// registerDefaultSanitizers registers default input sanitizers
func (sm *SecurityManager) registerDefaultSanitizers() {
	// HTML sanitizer
	sm.sanitizers["html"] = func(input string) string {
		// Remove script tags
		scriptRegex := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
		input = scriptRegex.ReplaceAllString(input, "")

		// Remove dangerous attributes
		attrRegex := regexp.MustCompile(`(?i)\s*(on\w+|javascript:|data:|vbscript:)[^\s>]*`)
		input = attrRegex.ReplaceAllString(input, "")

		return input
	}

	// SQL injection sanitizer
	sm.sanitizers["sql"] = func(input string) string {
		// Remove common SQL injection patterns
		sqlPatterns := []string{
			`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute)\s`,
			`(?i)(--|#|/\*|\*/|;)`,
			`(?i)(\bor\b|\band\b)\s+\d+\s*=\s*\d+`,
		}

		for _, pattern := range sqlPatterns {
			regex := regexp.MustCompile(pattern)
			input = regex.ReplaceAllString(input, "")
		}

		return input
	}

	// XSS sanitizer
	sm.sanitizers["xss"] = func(input string) string {
		// Remove script tags and event handlers
		xssPatterns := []string{
			`(?i)<script[^>]*>.*?</script>`,
			`(?i)<iframe[^>]*>.*?</iframe>`,
			`(?i)<object[^>]*>.*?</object>`,
			`(?i)<embed[^>]*>.*?</embed>`,
			`(?i)javascript:`,
			`(?i)vbscript:`,
			`(?i)on\w+\s*=`,
		}

		for _, pattern := range xssPatterns {
			regex := regexp.MustCompile(pattern)
			input = regex.ReplaceAllString(input, "")
		}

		return input
	}

	// Path traversal sanitizer
	sm.sanitizers["path"] = func(input string) string {
		// Remove path traversal patterns
		pathPatterns := []string{
			`\.\./`,
			`\.\\`,
			`%2e%2e%2f`,
			`%2e%2e%5c`,
		}

		for _, pattern := range pathPatterns {
			regex := regexp.MustCompile(pattern)
			input = regex.ReplaceAllString(input, "")
		}

		return input
	}
}

// registerDefaultValidators registers default input validators
func (sm *SecurityManager) registerDefaultValidators() {
	// Email validator
	sm.validators["email"] = func(input string) error {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(input) {
			return fmt.Errorf("invalid email format")
		}
		return nil
	}

	// URL validator
	sm.validators["url"] = func(input string) error {
		urlRegex := regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/.*)?$`)
		if !urlRegex.MatchString(input) {
			return fmt.Errorf("invalid URL format")
		}
		return nil
	}

	// Alphanumeric validator
	sm.validators["alphanumeric"] = func(input string) error {
		alphanumericRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
		if !alphanumericRegex.MatchString(input) {
			return fmt.Errorf("input must be alphanumeric")
		}
		return nil
	}

	// Length validator
	sm.validators["length"] = func(input string) error {
		if len(input) > 1000 { // Default max length
			return fmt.Errorf("input too long")
		}
		return nil
	}
}

// initializeWAFRules initializes default WAF rules
func (sm *SecurityManager) initializeWAFRules() {
	defaultRules := []WAFRule{
		{
			ID:          "sql_injection_1",
			Name:        "SQL Injection Detection",
			Pattern:     regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop)\s`),
			Severity:    "high",
			Action:      "block",
			Description: "Detects common SQL injection patterns",
			Enabled:     true,
		},
		{
			ID:          "xss_1",
			Name:        "XSS Detection",
			Pattern:     regexp.MustCompile(`(?i)<script[^>]*>|javascript:|on\w+\s*=`),
			Severity:    "high",
			Action:      "block",
			Description: "Detects cross-site scripting attempts",
			Enabled:     true,
		},
		{
			ID:          "path_traversal_1",
			Name:        "Path Traversal Detection",
			Pattern:     regexp.MustCompile(`\.\./|\.\\|%2e%2e%2f`),
			Severity:    "medium",
			Action:      "block",
			Description: "Detects path traversal attempts",
			Enabled:     true,
		},
		{
			ID:          "command_injection_1",
			Name:        "Command Injection Detection",
			Pattern:     regexp.MustCompile(`(?i)(;|\||&|\$\(|` + "`" + `|<|>)`),
			Severity:    "high",
			Action:      "block",
			Description: "Detects command injection attempts",
			Enabled:     true,
		},
		{
			ID:          "suspicious_user_agent_1",
			Name:        "Suspicious User Agent",
			Pattern:     regexp.MustCompile(`(?i)(sqlmap|nikto|nmap|masscan|zap|burp)`),
			Severity:    "medium",
			Action:      "log",
			Description: "Detects suspicious user agents",
			Enabled:     true,
		},
	}

	sm.config.WAFRules = append(sm.config.WAFRules, defaultRules...)
}

// CORSMiddleware returns CORS middleware
func (sm *SecurityManager) CORSMiddleware() echo.MiddlewareFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     sm.config.CORSAllowOrigins,
		AllowMethods:     sm.config.CORSAllowMethods,
		AllowHeaders:     sm.config.CORSAllowHeaders,
		ExposeHeaders:    sm.config.CORSExposeHeaders,
		AllowCredentials: sm.config.CORSAllowCredentials,
		MaxAge:           sm.config.CORSMaxAge,
	})
}

// SecurityHeadersMiddleware adds security headers
func (sm *SecurityManager) SecurityHeadersMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			res := c.Response()

			// HSTS
			if sm.config.HSTSMaxAge > 0 {
				hstsValue := fmt.Sprintf("max-age=%d", sm.config.HSTSMaxAge)
				if sm.config.HSTSIncludeSubdomains {
					hstsValue += "; includeSubDomains"
				}
				if sm.config.HSTSPreload {
					hstsValue += "; preload"
				}
				res.Header().Set("Strict-Transport-Security", hstsValue)
			}

			// CSP
			if len(sm.config.CSPDirectives) > 0 {
				cspValue := sm.buildCSPHeader()
				headerName := "Content-Security-Policy"
				if sm.config.CSPReportOnly {
					headerName = "Content-Security-Policy-Report-Only"
				}
				res.Header().Set(headerName, cspValue)
			}

			// Other security headers
			res.Header().Set("X-Content-Type-Options", "nosniff")
			res.Header().Set("X-Frame-Options", "DENY")
			res.Header().Set("X-XSS-Protection", "1; mode=block")
			res.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			res.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			return next(c)
		}
	}
}

// buildCSPHeader builds the CSP header value
func (sm *SecurityManager) buildCSPHeader() string {
	directives := make([]string, 0, len(sm.config.CSPDirectives))
	for directive, value := range sm.config.CSPDirectives {
		directives = append(directives, fmt.Sprintf("%s %s", directive, value))
	}

	if sm.config.CSPReportURI != "" {
		directives = append(directives, fmt.Sprintf("report-uri %s", sm.config.CSPReportURI))
	}

	return strings.Join(directives, "; ")
}

// RateLimitMiddleware returns rate limiting middleware
func (sm *SecurityManager) RateLimitMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientIP := sm.getClientIP(c)

			// Check if IP is whitelisted
			if sm.config.WhitelistEnabled && sm.isWhitelisted(clientIP) {
				return next(c)
			}

			// Check if IP is blacklisted
			if sm.config.BlacklistEnabled && sm.isBlacklisted(clientIP) {
				sm.recordSecurityViolation(c, "ip_blacklisted", "high", "IP is blacklisted", true)
				return echo.NewHTTPError(http.StatusForbidden, "Access denied")
			}

			// Apply rate limiting
			allowed, err := sm.checkRateLimit(c.Request().Context(), clientIP)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Rate limit check failed")
			}

			if !allowed {
				sm.metrics.mu.Lock()
				sm.metrics.RateLimitViolations++
				sm.metrics.mu.Unlock()

				sm.recordSecurityViolation(c, "rate_limit_exceeded", "medium", "Rate limit exceeded", true)

				// Auto-block if enabled
				if sm.config.AutoBlockSuspicious {
					sm.addToBlacklist(clientIP, sm.config.BlockDuration)
				}

				return echo.NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded")
			}

			return next(c)
		}
	}
}

// WAFMiddleware returns Web Application Firewall middleware
func (sm *SecurityManager) WAFMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !sm.config.WAFEnabled {
				return next(c)
			}

			// Check request against WAF rules
			violation := sm.checkWAFRules(c)
			if violation != nil {
				sm.metrics.mu.Lock()
				sm.metrics.WAFBlocks++
				sm.metrics.mu.Unlock()

				sm.recordSecurityViolation(c, violation.Type, violation.Severity, violation.Payload, violation.Blocked)

				if violation.Blocked && sm.config.WAFBlockMode {
					return echo.NewHTTPError(http.StatusForbidden, "Request blocked by WAF")
				}
			}

			return next(c)
		}
	}
}

// InputSanitizationMiddleware sanitizes request inputs
func (sm *SecurityManager) InputSanitizationMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Sanitize query parameters
			for _, values := range c.QueryParams() {
				for i, value := range values {
					sanitized := sm.sanitizeInput(value)
					values[i] = sanitized
				}
			}

			// Sanitize form values
			if c.Request().Method == "POST" || c.Request().Method == "PUT" {
				c.Request().ParseForm()
				for _, values := range c.Request().Form {
					for i, value := range values {
						sanitized := sm.sanitizeInput(value)
						values[i] = sanitized
					}
				}
			}

			return next(c)
		}
	}
}

// sanitizeInput applies all registered sanitizers to input
func (sm *SecurityManager) sanitizeInput(input string) string {
	result := input

	// Apply all sanitizers
	for _, sanitizer := range sm.sanitizers {
		result = sanitizer(result)
	}

	return result
}

// checkRateLimit checks if request is within rate limits
func (sm *SecurityManager) checkRateLimit(ctx context.Context, clientIP string) (bool, error) {
	// Use Redis for distributed rate limiting
	key := fmt.Sprintf("rate_limit:%s", clientIP)

	// Get current count
	current, err := sm.redisClient.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		return false, err
	}

	// Check if limit exceeded
	limit := sm.config.DefaultRateLimit
	if sm.config.DynamicAdjustment {
		limit = sm.calculateDynamicLimit(clientIP)
	}

	if current >= limit {
		return false, nil
	}

	// Increment counter
	pipe := sm.redisClient.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, sm.config.RateLimitWindow)
	_, err = pipe.Exec(ctx)

	return err == nil, err
}

// calculateDynamicLimit calculates dynamic rate limit based on behavior
func (sm *SecurityManager) calculateDynamicLimit(clientIP string) int {
	// This is a simplified implementation
	// In practice, you'd analyze historical behavior, reputation, etc.
	baseLimit := sm.config.DefaultRateLimit

	// Check if IP has violations
	violationKey := fmt.Sprintf("violations:%s", clientIP)
	violations, _ := sm.redisClient.Get(context.Background(), violationKey).Int()

	if violations > 0 {
		// Reduce limit for IPs with violations
		return int(float64(baseLimit) * (1.0 - sm.config.AdjustmentFactor))
	}

	return baseLimit
}

// checkWAFRules checks request against WAF rules
func (sm *SecurityManager) checkWAFRules(c echo.Context) *SecurityViolation {
	req := c.Request()

	// Collect request data to check
	requestData := []string{
		req.URL.Path,
		req.URL.RawQuery,
		req.Header.Get("User-Agent"),
		req.Header.Get("Referer"),
	}

	// Add request body if available
	if req.Body != nil {
		// Note: In practice, you'd need to be careful about reading the body
		// as it can only be read once. You might need to use a buffer.
	}

	for _, rule := range sm.config.WAFRules {
		if !rule.Enabled {
			continue
		}

		for _, data := range requestData {
			if rule.Pattern.MatchString(data) {
				return &SecurityViolation{
					ID:        sm.generateViolationID(),
					Timestamp: time.Now(),
					Type:      "waf_rule_violation",
					Severity:  rule.Severity,
					IP:        sm.getClientIP(c),
					UserAgent: req.Header.Get("User-Agent"),
					URL:       req.URL.String(),
					Method:    req.Method,
					Payload:   data,
					RuleID:    rule.ID,
					Action:    rule.Action,
					Blocked:   rule.Action == "block",
					Metadata: map[string]interface{}{
						"rule_name":   rule.Name,
						"description": rule.Description,
					},
				}
			}
		}
	}

	return nil
}

// getClientIP extracts client IP from request
func (sm *SecurityManager) getClientIP(c echo.Context) string {
	req := c.Request()

	// Check X-Forwarded-For header
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, _ := net.SplitHostPort(req.RemoteAddr)
	return ip
}

// isWhitelisted checks if IP is whitelisted
func (sm *SecurityManager) isWhitelisted(ip string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.ipWhitelist[ip]
}

// isBlacklisted checks if IP is blacklisted
func (sm *SecurityManager) isBlacklisted(ip string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.ipBlacklist[ip]
}

// addToBlacklist adds IP to blacklist
func (sm *SecurityManager) addToBlacklist(ip string, duration time.Duration) {
	sm.mu.Lock()
	sm.ipBlacklist[ip] = true
	sm.mu.Unlock()

	// Set expiration in Redis
	key := fmt.Sprintf("blacklist:%s", ip)
	sm.redisClient.Set(context.Background(), key, "1", duration)

	// Schedule removal
	go func() {
		time.Sleep(duration)
		sm.mu.Lock()
		delete(sm.ipBlacklist, ip)
		sm.mu.Unlock()
	}()
}

// recordSecurityViolation records a security violation
func (sm *SecurityManager) recordSecurityViolation(c echo.Context, violationType, severity, payload string, blocked bool) {
	violation := SecurityViolation{
		ID:        sm.generateViolationID(),
		Timestamp: time.Now(),
		Type:      violationType,
		Severity:  severity,
		IP:        sm.getClientIP(c),
		UserAgent: c.Request().Header.Get("User-Agent"),
		URL:       c.Request().URL.String(),
		Method:    c.Request().Method,
		Payload:   payload,
		Action:    "log",
		Blocked:   blocked,
		Metadata:  make(map[string]interface{}),
	}

	if blocked {
		violation.Action = "block"
		sm.metrics.mu.Lock()
		sm.metrics.RequestsBlocked++
		sm.metrics.mu.Unlock()
	}

	// Store violation in Redis
	key := fmt.Sprintf("violation:%s", violation.ID)
	data, _ := json.Marshal(violation)
	sm.redisClient.Set(context.Background(), key, data, 24*time.Hour)

	// Update violation count for IP
	violationCountKey := fmt.Sprintf("violations:%s", violation.IP)
	sm.redisClient.Incr(context.Background(), violationCountKey)
	sm.redisClient.Expire(context.Background(), violationCountKey, 24*time.Hour)
}

// generateViolationID generates a unique violation ID
func (sm *SecurityManager) generateViolationID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetSecurityMetrics returns current security metrics
func (sm *SecurityManager) GetSecurityMetrics() SecurityManagerMetrics {
	sm.metrics.mu.RLock()
	defer sm.metrics.mu.RUnlock()

	return SecurityManagerMetrics{
		RequestsBlocked:      sm.metrics.RequestsBlocked,
		RateLimitViolations:  sm.metrics.RateLimitViolations,
		CSPViolations:        sm.metrics.CSPViolations,
		XSSAttempts:          sm.metrics.XSSAttempts,
		SQLInjectionAttempts: sm.metrics.SQLInjectionAttempts,
		MaliciousUploads:     sm.metrics.MaliciousUploads,
		SuspiciousIPs:        sm.metrics.SuspiciousIPs,
		WAFBlocks:            sm.metrics.WAFBlocks,
	}
}

// GetSecurityViolations returns recent security violations
func (sm *SecurityManager) GetSecurityViolations(ctx context.Context, limit int) ([]SecurityViolation, error) {
	pattern := "violation:*"
	keys, err := sm.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	violations := make([]SecurityViolation, 0)
	for i, key := range keys {
		if i >= limit {
			break
		}

		data, err := sm.redisClient.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var violation SecurityViolation
		if err := json.Unmarshal([]byte(data), &violation); err == nil {
			violations = append(violations, violation)
		}
	}

	return violations, nil
}

// UpdateSecurityConfig updates security configuration
func (sm *SecurityManager) UpdateSecurityConfig(config SecurityConfig) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.config = config
}

// AddCustomSanitizer adds a custom sanitizer
func (sm *SecurityManager) AddCustomSanitizer(name string, sanitizer SanitizerFunc) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sanitizers[name] = sanitizer
}

// AddCustomValidator adds a custom validator
func (sm *SecurityManager) AddCustomValidator(name string, validator ValidatorFunc) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.validators[name] = validator
}

// ValidateInput validates input using registered validators
func (sm *SecurityManager) ValidateInput(input string, validatorNames []string) []error {
	errors := make([]error, 0)

	for _, name := range validatorNames {
		if validator, exists := sm.validators[name]; exists {
			if err := validator(input); err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}

// SanitizeInput sanitizes input using specified sanitizers
func (sm *SecurityManager) SanitizeInput(input string, sanitizerNames []string) string {
	result := input

	for _, name := range sanitizerNames {
		if sanitizer, exists := sm.sanitizers[name]; exists {
			result = sanitizer(result)
		}
	}

	return result
}
