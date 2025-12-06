package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// RateLimiterConfig defines the configuration for rate limiting
type RateLimiterConfig struct {
	// Skipper defines a function to skip middleware
	Skipper middleware.Skipper

	// Store defines the rate limiter store (Redis or in-memory)
	Store RateLimiterStore

	// IdentifierExtractor extracts the identifier from request (IP, user ID, etc.)
	IdentifierExtractor func(echo.Context) string

	// Max defines the maximum number of requests
	Max int

	// Window defines the time window for rate limiting
	Window time.Duration

	// Message defines the message to return when rate limit is exceeded
	Message string

	// StatusCode defines the status code to return when rate limit is exceeded
	StatusCode int

	// Headers defines headers to include rate limit information
	Headers bool
}

// RateLimiterStore defines the interface for rate limiter storage
type RateLimiterStore interface {
	Allow(ctx context.Context, identifier string, max int, window time.Duration) (bool, int, time.Time, error)
	Reset(ctx context.Context, identifier string) error
}

// RedisStore implements RateLimiterStore using Redis
type RedisStore struct {
	client *redis.Client
	prefix string
}

// NewRedisStore creates a new Redis store for rate limiting
func NewRedisStore(client *redis.Client, prefix string) *RedisStore {
	if prefix == "" {
		prefix = "ratelimit:"
	}
	return &RedisStore{
		client: client,
		prefix: prefix,
	}
}

// Allow checks if the request is allowed and updates the counter
func (r *RedisStore) Allow(ctx context.Context, identifier string, max int, window time.Duration) (bool, int, time.Time, error) {
	key := r.prefix + identifier
	
	// Use Redis pipeline for atomic operations
	pipe := r.client.TxPipeline()
	
	// Increment counter
	countCmd := pipe.Incr(ctx, key)
	
	// Set expiry only if key is new
	pipe.Expire(ctx, key, window)
	
	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, time.Time{}, err
	}
	
	count := countCmd.Val()
	
	// Calculate reset time
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return false, int(count), time.Time{}, err
	}
	
	resetTime := time.Now().Add(ttl)
	
	// Check if limit exceeded
	allowed := count <= int64(max)
	
	return allowed, int(count), resetTime, nil
}

// Reset resets the counter for an identifier
func (r *RedisStore) Reset(ctx context.Context, identifier string) error {
	key := r.prefix + identifier
	return r.client.Del(ctx, key).Err()
}

// MemoryStore implements RateLimiterStore using in-memory storage
type MemoryStore struct {
	entries map[string]*entry
	mutex   sync.RWMutex
}

type entry struct {
	count     int
	resetTime time.Time
}

// NewMemoryStore creates a new in-memory store for rate limiting
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		entries: make(map[string]*entry),
	}
	
	// Start cleanup goroutine
	go store.cleanup()
	
	return store
}

// Allow checks if the request is allowed and updates the counter
func (m *MemoryStore) Allow(ctx context.Context, identifier string, max int, window time.Duration) (bool, int, time.Time, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	now := time.Now()
	
	// Get or create entry
	e, exists := m.entries[identifier]
	if !exists || now.After(e.resetTime) {
		e = &entry{
			count:     1,
			resetTime: now.Add(window),
		}
		m.entries[identifier] = e
		return true, 1, e.resetTime, nil
	}
	
	// Increment count
	e.count++
	
	// Check if limit exceeded
	allowed := e.count <= max
	
	return allowed, e.count, e.resetTime, nil
}

// Reset resets the counter for an identifier
func (m *MemoryStore) Reset(ctx context.Context, identifier string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	delete(m.entries, identifier)
	return nil
}

// cleanup removes expired entries
func (m *MemoryStore) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		m.mutex.Lock()
		now := time.Now()
		for key, entry := range m.entries {
			if now.After(entry.resetTime) {
				delete(m.entries, key)
			}
		}
		m.mutex.Unlock()
	}
}

// DefaultRateLimiterConfig returns default configuration
var DefaultRateLimiterConfig = RateLimiterConfig{
	Skipper: middleware.DefaultSkipper,
	IdentifierExtractor: func(c echo.Context) string {
		// Try to get user ID from context first
		if userID := c.Get("user_id"); userID != nil {
			return fmt.Sprintf("user:%v", userID)
		}
		// Fallback to IP address
		return "ip:" + c.RealIP()
	},
	Max:        100,                  // 100 requests
	Window:     15 * time.Minute,     // per 15 minutes
	Message:    "Rate limit exceeded. Please try again later.",
	StatusCode: http.StatusTooManyRequests,
	Headers:    true,
}

// RateLimiterWithConfig returns a rate limiter middleware with configuration
func RateLimiterWithConfig(config RateLimiterConfig) echo.MiddlewareFunc {
	// Set defaults
	if config.Skipper == nil {
		config.Skipper = DefaultRateLimiterConfig.Skipper
	}
	if config.IdentifierExtractor == nil {
		config.IdentifierExtractor = DefaultRateLimiterConfig.IdentifierExtractor
	}
	if config.Max == 0 {
		config.Max = DefaultRateLimiterConfig.Max
	}
	if config.Window == 0 {
		config.Window = DefaultRateLimiterConfig.Window
	}
	if config.Message == "" {
		config.Message = DefaultRateLimiterConfig.Message
	}
	if config.StatusCode == 0 {
		config.StatusCode = DefaultRateLimiterConfig.StatusCode
	}
	if config.Store == nil {
		config.Store = NewMemoryStore()
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			identifier := config.IdentifierExtractor(c)
			
			allowed, count, resetTime, err := config.Store.Allow(
				c.Request().Context(),
				identifier,
				config.Max,
				config.Window,
			)
			
			if err != nil {
				// If store fails, log error but allow request
				c.Logger().Error("Rate limiter store error:", err)
				return next(c)
			}

			// Set rate limit headers
			if config.Headers {
				c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(config.Max))
				c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(config.Max-count))
				c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
			}

			if !allowed {
				return echo.NewHTTPError(config.StatusCode, config.Message)
			}

			return next(c)
		}
	}
}

// RateLimiter returns a rate limiter middleware with default configuration
func RateLimiter() echo.MiddlewareFunc {
	return RateLimiterWithConfig(DefaultRateLimiterConfig)
}

// RateLimiterWithRedis returns a rate limiter middleware using Redis store
func RateLimiterWithRedis(redisClient *redis.Client) echo.MiddlewareFunc {
	config := DefaultRateLimiterConfig
	config.Store = NewRedisStore(redisClient, "ratelimit:")
	return RateLimiterWithConfig(config)
}

// UserBasedRateLimiter returns a rate limiter specifically for authenticated users
func UserBasedRateLimiter(max int, window time.Duration) echo.MiddlewareFunc {
	config := DefaultRateLimiterConfig
	config.Max = max
	config.Window = window
	config.IdentifierExtractor = func(c echo.Context) string {
		// Prioritize user ID for authenticated requests
		if userID := c.Get("user_id"); userID != nil {
			return fmt.Sprintf("user:%v", userID)
		}
		// More restrictive for non-authenticated
		return "anonymous:" + c.RealIP()
	}
	return RateLimiterWithConfig(config)
}

// APIKeyRateLimiter returns a rate limiter for API key based requests
func APIKeyRateLimiter(max int, window time.Duration) echo.MiddlewareFunc {
	config := DefaultRateLimiterConfig
	config.Max = max
	config.Window = window
	config.IdentifierExtractor = func(c echo.Context) string {
		// Use API key if available
		if apiKey := c.Request().Header.Get("X-API-Key"); apiKey != "" {
			return "apikey:" + apiKey
		}
		// Fallback to IP
		return "ip:" + c.RealIP()
	}
	return RateLimiterWithConfig(config)
}