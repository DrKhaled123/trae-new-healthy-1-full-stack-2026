// ratelimit.go
package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type RateLimiter struct {
	clients map[string]*ClientInfo
	mutex   sync.RWMutex
	limit   int
}

type ClientInfo struct {
	Requests       int
	LastAccess     time.Time
	QuotaUsed      int
	QuotaLimit     int
	DailyResetTime time.Time
}

var rateLimiter *RateLimiter
var rateLimiterOnce sync.Once

func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*ClientInfo),
		limit:   limit,
	}
}

func GetRateLimiter() *RateLimiter {
	rateLimiterOnce.Do(func() {
		config := LoadConfig()
		rateLimiter = NewRateLimiter(config.APIRateLimit)
	})
	return rateLimiter
}

func (rl *RateLimiter) GetRateLimitMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientIP := c.RealIP()
			apiKey := c.Request().Header.Get("X-API-Key")

			// Use API key if available, otherwise use IP
			identifier := apiKey
			if identifier == "" {
				identifier = clientIP
			}

			allowed, remaining, resetTime, err := rl.Allow(identifier)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Rate limit check failed",
				})
			}

			if !allowed {
				c.Response().Header().Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))
				return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
					"error":             "Rate limit exceeded",
					"reset_time":        resetTime.Format(time.RFC3339),
					"retry_after":       int(time.Until(resetTime).Seconds()),
					"rate_limit_window": "60", // seconds
				})
			}

			// Add rate limit headers to response
			c.Response().Header().Set("X-RateLimit-Limit", "100")
			c.Response().Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			c.Response().Header().Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))

			return next(c)
		}
	}
}

func (rl *RateLimiter) Allow(identifier string) (allowed bool, remaining int, resetTime time.Time, err error) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	client, exists := rl.clients[identifier]
	if !exists {
		// New client, initialize with default quota
		client = &ClientInfo{
			QuotaLimit:     rl.limit,
			DailyResetTime: now.Add(24 * time.Hour),
		}
		rl.clients[identifier] = client
	}

	// Check if we need to reset daily quota
	if now.After(client.DailyResetTime) {
		client.QuotaUsed = 0
		client.DailyResetTime = now.Add(24 * time.Hour)
	}

	// Reset counter if more than a minute has passed
	if now.Sub(client.LastAccess) > time.Minute {
		client.Requests = 0
		client.LastAccess = now
		resetTime = now.Add(time.Minute)
	} else {
		resetTime = client.LastAccess.Add(time.Minute)
	}

	// Check quota limit
	if client.QuotaUsed >= client.QuotaLimit {
		resetTime = client.DailyResetTime
		return false, 0, resetTime, nil
	}

	// Check rate limit (requests per minute)
	if client.Requests >= rl.limit {
		return false, 0, resetTime, nil
	}

	// Increment counters
	client.Requests++
	client.QuotaUsed++
	client.LastAccess = now

	remaining = rl.limit - client.Requests
	return true, remaining, resetTime, nil
}

func (rl *RateLimiter) ResetDailyQuotas() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	for _, client := range rl.clients {
		if now.After(client.DailyResetTime) {
			client.QuotaUsed = 0
			client.DailyResetTime = now.Add(24 * time.Hour)
		}
	}
}

func (rl *RateLimiter) GetQuotaInfo(identifier string) (used, limit int, resetTime time.Time) {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	client, exists := rl.clients[identifier]
	if !exists {
		// Return default values for new clients
		limit = rl.limit
		resetTime = time.Now().Add(24 * time.Hour)
		return
	}

	used = client.QuotaUsed
	limit = client.QuotaLimit
	resetTime = client.DailyResetTime
	return
}

// Quota management functions
func (rl *RateLimiter) SetQuota(identifier string, limit int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	client, exists := rl.clients[identifier]
	if !exists {
		client = &ClientInfo{
			QuotaLimit:     limit,
			DailyResetTime: time.Now().Add(24 * time.Hour),
		}
		rl.clients[identifier] = client
	} else {
		client.QuotaLimit = limit
	}
}

func (rl *RateLimiter) ResetQuota(identifier string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	client, exists := rl.clients[identifier]
	if exists {
		client.QuotaUsed = 0
		client.DailyResetTime = time.Now().Add(24 * time.Hour)
	}
}

// Admin functions for quota management
func GetQuotaHandler(c echo.Context) error {
	apiKey := c.Request().Header.Get("X-API-Key")
	if apiKey == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "API key required",
		})
	}

	rl := GetRateLimiter()
	used, limit, resetTime := rl.GetQuotaInfo(apiKey)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"used":       used,
		"limit":      limit,
		"remaining":  limit - used,
		"reset_time": resetTime.Format(time.RFC3339),
	})
}

func ResetQuotaHandler(c echo.Context) error {
	// This should be protected by admin authentication
	apiKey := c.Request().Header.Get("X-API-Key")
	if apiKey == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "API key required",
		})
	}

	rl := GetRateLimiter()
	rl.ResetQuota(apiKey)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Quota reset successfully",
	})
}
