package security

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// RateLimitConfig defines rate limiting rules per endpoint
type RateLimitConfig struct {
	RequestsPerMinute int                      `json:"requests_per_minute"`
	BurstSize         int                      `json:"burst_size"`
	EndpointLimits    map[string]EndpointLimit `json:"endpoint_limits"`
}

// EndpointLimit defines specific limits for endpoints
type EndpointLimit struct {
	RequestsPerMinute int      `json:"requests_per_minute"`
	RequiresAuth      bool     `json:"requires_auth"`
	AllowedRoles      []string `json:"allowed_roles"`
}

// RateLimiter manages request rate limiting
type RateLimiter struct {
	mu      sync.RWMutex
	clients map[string]*ClientBucket
	config  *RateLimitConfig
	cleanup time.Time
}

// ClientBucket tracks requests for a client
type ClientBucket struct {
	tokens     int
	lastRefill time.Time
	requests   []time.Time
}

// NewRateLimiter creates a new rate limiter with default configuration
func NewRateLimiter() *RateLimiter {
	config := &RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         10,
		EndpointLimits: map[string]EndpointLimit{
			"/api/v1/nutrition/generate-plan": {
				RequestsPerMinute: 10,
				RequiresAuth:      true,
				AllowedRoles:      []string{"user", "nutritionist", "doctor"},
			},
			"/api/v1/workouts/generate-plan": {
				RequestsPerMinute: 15,
				RequiresAuth:      true,
				AllowedRoles:      []string{"user", "nutritionist", "doctor"},
			},
			"/api/v1/health/generate-plan": {
				RequestsPerMinute: 5,
				RequiresAuth:      true,
				AllowedRoles:      []string{"user", "doctor"},
			},
			"/api/v1/users": {
				RequestsPerMinute: 30,
				RequiresAuth:      false,
				AllowedRoles:      []string{"admin", "user"},
			},
		},
	}

	return &RateLimiter{
		clients: make(map[string]*ClientBucket),
		config:  config,
	}
}

// RateLimitingMiddleware creates rate limiting middleware
func (rl *RateLimiter) RateLimitingMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientIP := getClientIP(c)
			endpoint := c.Request().URL.Path

			// Check if endpoint has specific limits
			endpointLimit, hasSpecificLimit := rl.config.EndpointLimits[endpoint]

			// Check authentication requirements
			if hasSpecificLimit && endpointLimit.RequiresAuth {
				userID := c.Get("user_id")
				if userID == nil {
					return c.JSON(http.StatusUnauthorized, map[string]interface{}{
						"error": "Authentication required for this endpoint",
					})
				}

				// Check role permissions
				userRole := c.Get("user_role").(string)
				roleAllowed := false
				for _, allowedRole := range endpointLimit.AllowedRoles {
					if userRole == allowedRole {
						roleAllowed = true
						break
					}
				}

				if !roleAllowed {
					return c.JSON(http.StatusForbidden, map[string]interface{}{
						"error": "Insufficient role permissions",
					})
				}

				clientIP = fmt.Sprintf("%s_%s", userID, endpoint) // Rate limit per user per endpoint
			}

			// Get rate limit for this endpoint
			limit := rl.config.RequestsPerMinute
			if hasSpecificLimit {
				limit = endpointLimit.RequestsPerMinute
			}

			// Check rate limit
			allowed, retryAfter := rl.IsAllowed(clientIP, limit)
			if !allowed {
				c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
				c.Response().Header().Set("X-RateLimit-Remaining", "0")
				c.Response().Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))
				c.Response().Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))

				return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
					"error":               "Rate limit exceeded",
					"retry_after_seconds": int(retryAfter.Seconds()),
				})
			}

			// Record request
			rl.RecordRequest(clientIP)

			// Set headers
			c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			c.Response().Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", limit-1))

			return next(c)
		}
	}
}

// IsAllowed checks if request is allowed
func (rl *RateLimiter) IsAllowed(clientID string, limit int) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.clients[clientID]

	if !exists {
		// Create new bucket
		rl.clients[clientID] = &ClientBucket{
			tokens:     limit,
			lastRefill: now,
			requests:   []time.Time{now},
		}
		return true, 0
	}

	// Clean old requests (sliding window)
	var recentRequests []time.Time
	cutoff := now.Add(-time.Minute)
	for _, reqTime := range bucket.requests {
		if reqTime.After(cutoff) {
			recentRequests = append(recentRequests, reqTime)
		}
	}
	bucket.requests = recentRequests

	// Check if under limit
	if len(bucket.requests) >= limit {
		// Calculate when next request will be allowed
		oldestRequest := bucket.requests[0]
		retryAfter := time.Minute - now.Sub(oldestRequest)
		return false, retryAfter
	}

	return true, 0
}

// RecordRequest records a new request
func (rl *RateLimiter) RecordRequest(clientID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket := rl.clients[clientID]
	bucket.requests = append(bucket.requests, time.Now())

	// Cleanup old clients periodically
	if time.Since(rl.cleanup) > time.Hour {
		rl.cleanupOldClients()
		rl.cleanup = time.Now()
	}
}

// cleanupOldClients removes old client entries
func (rl *RateLimiter) cleanupOldClients() {
	now := time.Now()
	threshold := now.Add(-time.Hour * 24) // Remove clients inactive for 24 hours

	for clientID, bucket := range rl.clients {
		if bucket.lastRefill.Before(threshold) {
			delete(rl.clients, clientID)
		}
	}
}

// getClientIP extracts client IP address
func getClientIP(c echo.Context) string {
	// Check X-Forwarded-For header (for load balancers/proxies)
	xff := c.Request().Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xri := c.Request().Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to remote address
	return c.RealIP()
}

// AdaptiveRateLimiter provides AI-powered rate limiting
type AdaptiveRateLimiter struct {
	*RateLimiter
	suspiciousClients map[string]*SuspiciousActivity
	aiThreshold       float64
}

// SuspiciousActivity tracks potentially malicious behavior
type SuspiciousActivity struct {
	ErrorRate      float64
	RequestPattern []string
	LastActivity   time.Time
	SuspicionScore float64
}

// NewAdaptiveRateLimiter creates an adaptive rate limiter
func NewAdaptiveRateLimiter() *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		RateLimiter:       NewRateLimiter(),
		suspiciousClients: make(map[string]*SuspiciousActivity),
		aiThreshold:       5.0, // Threshold for flagging suspicious activity
	}
}

// IsSuspiciousActivity detects potential bot or malicious activity
func (arl *AdaptiveRateLimiter) IsSuspiciousActivity(requestPattern map[string]interface{}) bool {
	score := 0.0

	// High error rate
	if errorRate, ok := requestPattern["error_rate"].(float64); ok && errorRate > 0.3 {
		score += 2.0
	}

	// Too many requests per minute
	if rpm, ok := requestPattern["requests_per_minute"].(float64); ok && rpm > 100 {
		score += 3.0
	}

	// Unusual user agents
	if userAgents, ok := requestPattern["user_agents"].([]string); ok && len(userAgents) > 10 {
		score += 1.0
	}

	// Sequential access patterns
	if pattern, ok := requestPattern["access_pattern"].(string); ok && pattern == "sequential" {
		score += 1.5
	}

	return score > arl.aiThreshold
}

// UpdateSuspiciousActivity updates tracking for suspicious clients
func (arl *AdaptiveRateLimiter) UpdateSuspiciousActivity(clientID string, isError bool) {
	arl.mu.Lock()
	defer arl.mu.Unlock()

	activity, exists := arl.suspiciousClients[clientID]
	if !exists {
		activity = &SuspiciousActivity{
			LastActivity: time.Now(),
		}
		arl.suspiciousClients[clientID] = activity
	}

	// Update error rate (simple moving average)
	if isError {
		activity.ErrorRate = (activity.ErrorRate * 0.9) + 0.1 // Weighted towards recent activity
	} else {
		activity.ErrorRate = activity.ErrorRate * 0.9
	}

	activity.LastActivity = time.Now()
	activity.SuspicionScore = activity.ErrorRate * 10 // Convert to score
}

// GetSuspiciousClients returns list of suspicious clients
func (arl *AdaptiveRateLimiter) GetSuspiciousClients() map[string]*SuspiciousActivity {
	arl.mu.RLock()
	defer arl.mu.RUnlock()

	result := make(map[string]*SuspiciousActivity)
	for clientID, activity := range arl.suspiciousClients {
		if activity.SuspicionScore > arl.aiThreshold {
			result[clientID] = activity
		}
	}

	return result
}
