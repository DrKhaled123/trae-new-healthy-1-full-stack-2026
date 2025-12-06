package middleware

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// CacheEntry represents a cached response
type CacheEntry struct {
	Data         []byte
	Headers      http.Header
	StatusCode   int
	ExpiresAt    time.Time
	ETag         string
	LastModified time.Time
}

// CacheConfig defines cache configuration
type CacheConfig struct {
	// DefaultTTL is the default time-to-live for cache entries
	DefaultTTL time.Duration
	// MaxSize is the maximum number of cache entries
	MaxSize int
	// KeyGenerator is a custom key generation function
	KeyGenerator func(echo.Context) string
	// SkipMethods are HTTP methods to skip caching
	SkipMethods []string
	// SkipPaths are URL patterns to skip caching
	SkipPaths []string
	// VaryByHeaders are headers that should be included in cache key
	VaryByHeaders []string
	// CompressResponses enables gzip compression
	CompressResponses bool
}

// MemoryCache represents an in-memory cache
type MemoryCache struct {
	entries map[string]*CacheEntry
	mutex   sync.RWMutex
	maxSize int
	size    int
}

// ResponseCache represents the cache middleware
type ResponseCache struct {
	cache  *MemoryCache
	config *CacheConfig
}

// NewCacheConfig returns default cache configuration
func NewCacheConfig() *CacheConfig {
	return &CacheConfig{
		DefaultTTL:        5 * time.Minute,
		MaxSize:           1000,
		SkipMethods:       []string{"POST", "PUT", "DELETE", "PATCH"},
		SkipPaths:         []string{"/health", "/metrics"},
		VaryByHeaders:     []string{"Authorization", "Accept-Language"},
		CompressResponses: false,
	}
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(maxSize int) *MemoryCache {
	return &MemoryCache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSize,
	}
}

// NewResponseCache creates a new cache middleware
func NewResponseCache(config *CacheConfig) *ResponseCache {
	if config == nil {
		config = NewCacheConfig()
	}

	return &ResponseCache{
		cache:  NewMemoryCache(config.MaxSize),
		config: config,
	}
}

// Middleware returns the Echo middleware function
func (rc *ResponseCache) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip caching for specified methods and paths
			if rc.shouldSkipCache(c) {
				return next(c)
			}

			// Generate cache key
			key := rc.generateCacheKey(c)

			// Try to get from cache
			if entry := rc.cache.Get(key); entry != nil && !entry.IsExpired() {
				return rc.serveFromCache(c, entry)
			}

			// Capture response
			recorder := &responseRecorder{
				ResponseWriter: c.Response().Writer,
				body:           &bytes.Buffer{},
				headers:        make(http.Header),
			}
			c.Response().Writer = recorder

			// Process request
			err := next(c)

			// Cache the response if it's cacheable
			if rc.isCacheable(c, recorder) {
				entry := &CacheEntry{
					Data:         recorder.body.Bytes(),
					Headers:      recorder.headers.Clone(),
					StatusCode:   recorder.status,
					ExpiresAt:    time.Now().Add(rc.getTTL(c)),
					ETag:         c.Response().Header().Get("ETag"),
					LastModified: time.Now(),
				}

				rc.cache.Set(key, entry)
			}

			// Restore original response writer
			c.Response().Writer = recorder.ResponseWriter

			// Set cache miss header if not cached
			c.Response().Header().Set("X-Cache", "MISS")

			return err
		}
	}
}

// shouldSkipCache determines if caching should be skipped
func (rc *ResponseCache) shouldSkipCache(c echo.Context) bool {
	// Skip specified methods
	for _, method := range rc.config.SkipMethods {
		if c.Request().Method == method {
			return true
		}
	}

	// Skip specified paths
	for _, path := range rc.config.SkipPaths {
		if strings.HasPrefix(c.Request().URL.Path, path) {
			return true
		}
	}

	// Skip if no-cache header is present
	if c.Request().Header.Get("Cache-Control") == "no-cache" {
		return true
	}

	return false
}

// generateCacheKey generates a cache key for the request
func (rc *ResponseCache) generateCacheKey(c echo.Context) string {
	if rc.config.KeyGenerator != nil {
		return rc.config.KeyGenerator(c)
	}

	// Default key generation
	var keyBuilder strings.Builder
	keyBuilder.WriteString(c.Request().Method)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(c.Request().URL.Path)

	if c.Request().URL.RawQuery != "" {
		keyBuilder.WriteString("?")
		keyBuilder.WriteString(c.Request().URL.RawQuery)
	}

	// Include vary headers in key
	for _, headerName := range rc.config.VaryByHeaders {
		if value := c.Request().Header.Get(headerName); value != "" {
			keyBuilder.WriteString(":")
			keyBuilder.WriteString(headerName)
			keyBuilder.WriteString("=")
			keyBuilder.WriteString(value)
		}
	}

	// Create hash for consistent key length
	hash := md5.Sum([]byte(keyBuilder.String()))
	return fmt.Sprintf("%x", hash)
}

// isCacheable determines if the response should be cached
func (rc *ResponseCache) isCacheable(c echo.Context, recorder *responseRecorder) bool {
	// Only cache successful responses
	if recorder.status < 200 || recorder.status >= 300 {
		return false
	}

	// Don't cache if explicitly disabled
	if c.Response().Header().Get("Cache-Control") == "no-store" {
		return false
	}

	// Don't cache large responses
	if recorder.body.Len() > 1024*1024 { // 1MB
		return false
	}

	return true
}

// getTTL returns the time-to-live for the cache entry
func (rc *ResponseCache) getTTL(c echo.Context) time.Duration {
	// Check for explicit cache control headers
	if cacheControl := c.Response().Header().Get("Cache-Control"); cacheControl != "" {
		// Parse max-age directive
		if strings.Contains(cacheControl, "max-age=") {
			parts := strings.Split(cacheControl, "=")
			if len(parts) == 2 {
				// Parse seconds to duration
				seconds := 0
				fmt.Sscanf(parts[1], "%d", &seconds)
				return time.Duration(seconds) * time.Second
			}
		}
	}

	return rc.config.DefaultTTL
}

// serveFromCache serves a cached response
func (rc *ResponseCache) serveFromCache(c echo.Context, entry *CacheEntry) error {
	// Copy headers
	for key, values := range entry.Headers {
		c.Response().Header()[key] = values
	}

	// Set cache-related headers
	c.Response().Header().Set("X-Cache", "HIT")
	c.Response().Header().Set("X-Cache-Expires", entry.ExpiresAt.Format(time.RFC1123))

	// Check ETag
	if entry.ETag != "" {
		ifNoneMatch := c.Request().Header.Get("If-None-Match")
		if ifNoneMatch == entry.ETag {
			return c.NoContent(http.StatusNotModified)
		}
		c.Response().Header().Set("ETag", entry.ETag)
	}

	// Write status code
	c.Response().WriteHeader(entry.StatusCode)

	// Write body
	_, err := c.Response().Writer.Write(entry.Data)
	return err
}

// responseRecorder captures response data
type responseRecorder struct {
	http.ResponseWriter
	status  int
	body    *bytes.Buffer
	headers http.Header
}

// WriteHeader captures status code
func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Write captures body
func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(data)
}

// Header returns the response headers
func (r *responseRecorder) Header() http.Header {
	return r.headers
}

// IsExpired checks if a cache entry has expired
func (ce *CacheEntry) IsExpired() bool {
	return time.Now().After(ce.ExpiresAt)
}

// Get retrieves a cache entry
func (mc *MemoryCache) Get(key string) *CacheEntry {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	entry, exists := mc.entries[key]
	if !exists {
		return nil
	}

	if entry.IsExpired() {
		// Remove expired entry
		delete(mc.entries, key)
		mc.size--
		return nil
	}

	return entry
}

// Set stores a cache entry
func (mc *MemoryCache) Set(key string, entry *CacheEntry) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	// Check if cache is full
	if mc.size >= mc.maxSize {
		mc.evictOldest()
	}

	// Store entry
	mc.entries[key] = entry
	mc.size++
}

// evictOldest removes the oldest cache entries
func (mc *MemoryCache) evictOldest() {
	// Remove 10% of entries to make room
	toRemove := mc.maxSize / 10
	if toRemove == 0 {
		toRemove = 1
	}

	count := 0
	for key := range mc.entries {
		delete(mc.entries, key)
		mc.size--
		count++
		if count >= toRemove {
			break
		}
	}
}

// Clear removes all cache entries
func (mc *MemoryCache) Clear() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.entries = make(map[string]*CacheEntry)
	mc.size = 0
}

// Stats returns cache statistics
func (mc *MemoryCache) Stats() map[string]interface{} {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	expired := 0
	now := time.Now()
	for _, entry := range mc.entries {
		if now.After(entry.ExpiresAt) {
			expired++
		}
	}

	return map[string]interface{}{
		"total_entries": mc.size,
		"max_size":      mc.maxSize,
		"expired":       expired,
		"hit_rate":      0.0, // Would need to track hits/misses
	}
}

// StaticFileCache creates a cache for static files
func StaticFileCache(ttl time.Duration, maxSize int) echo.MiddlewareFunc {
	cache := NewMemoryCache(maxSize)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Only cache GET requests
			if c.Request().Method != "GET" {
				return next(c)
			}

			// Generate file-based key
			key := c.Request().URL.Path

			// Try cache
			if entry := cache.Get(key); entry != nil && !entry.IsExpired() {
				c.Response().Header().Set("X-Cache", "HIT")
				c.Response().WriteHeader(entry.StatusCode)
				c.Response().Writer.Write(entry.Data)
				return nil
			}

			// Capture response
			recorder := &responseRecorder{
				ResponseWriter: c.Response().Writer,
				body:           &bytes.Buffer{},
				headers:        make(http.Header),
			}
			c.Response().Writer = recorder

			err := next(c)

			// Cache if successful
			if err == nil && recorder.status == 200 && recorder.body.Len() < 512*1024 { // 512KB
				entry := &CacheEntry{
					Data:       recorder.body.Bytes(),
					Headers:    recorder.headers.Clone(),
					StatusCode: recorder.status,
					ExpiresAt:  time.Now().Add(ttl),
				}
				cache.Set(key, entry)
			}

			// Restore writer
			c.Response().Writer = recorder.ResponseWriter
			c.Response().Header().Set("X-Cache", "MISS")

			return err
		}
	}
}

// APICache creates a cache optimized for API responses
func APICache(ttl time.Duration) echo.MiddlewareFunc {
	config := &CacheConfig{
		DefaultTTL:        ttl,
		MaxSize:           500,
		SkipMethods:       []string{"POST", "PUT", "DELETE", "PATCH"},
		SkipPaths:         []string{"/health", "/metrics", "/api/v1/auth"},
		VaryByHeaders:     []string{"Authorization"},
		CompressResponses: true,
	}

	cache := NewResponseCache(config)
	return cache.Middleware()
}

// ConditionalCache creates a cache that respects conditional requests
func ConditionalCache(ttl time.Duration) echo.MiddlewareFunc {
	config := NewCacheConfig()
	config.DefaultTTL = ttl
	config.MaxSize = 1000

	cache := NewResponseCache(config)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Handle If-Modified-Since
			if ifModifiedSince := c.Request().Header.Get("If-Modified-Since"); ifModifiedSince != "" {
				if modifiedTime, err := time.Parse(time.RFC1123, ifModifiedSince); err == nil {
					c.Response().Header().Set("Last-Modified", modifiedTime.Format(time.RFC1123))
					return c.NoContent(http.StatusNotModified)
				}
			}

			return cache.Middleware()(next)(c)
		}
	}
}

// CacheStats provides cache statistics endpoint
func CacheStats(cache *MemoryCache) echo.HandlerFunc {
	return func(c echo.Context) error {
		stats := cache.Stats()
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "success",
			"data":   stats,
		})
	}
}

// CacheClear provides cache clearing endpoint
func CacheClear(cache *MemoryCache) echo.HandlerFunc {
	return func(c echo.Context) error {
		cache.Clear()
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":  "success",
			"message": "Cache cleared successfully",
		})
	}
}
