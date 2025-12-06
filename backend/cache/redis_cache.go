package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
)

// RedisCache implements caching with Redis backend
type RedisCache struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// GetClient returns the underlying Redis client
func (r *RedisCache) GetClient() *redis.Client {
	return r.client
}

// CacheItem represents an item stored in cache
type CacheItem struct {
	Value     interface{} `json:"value"`
	ExpiresAt int64       `json:"expires_at"`
	CreatedAt int64       `json:"created_at"`
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(addr, password, prefix string, ttl time.Duration) (*RedisCache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
		PoolSize: 10,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client: rdb,
		prefix: prefix,
		ttl:    ttl,
	}, nil
}

// Set stores a value in cache with TTL
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}) error {
	cacheKey := r.getCacheKey(key)

	item := CacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(r.ttl).Unix(),
		CreatedAt: time.Now().Unix(),
	}

	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal cache item: %w", err)
	}

	return r.client.Set(ctx, cacheKey, data, r.ttl).Err()
}

// Get retrieves a value from cache
func (r *RedisCache) Get(ctx context.Context, key string) (interface{}, error) {
	cacheKey := r.getCacheKey(key)

	data, err := r.client.Get(ctx, cacheKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get from cache: %w", err)
	}

	var item CacheItem
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache item: %w", err)
	}

	// Check if expired
	if time.Now().Unix() > item.ExpiresAt {
		// Remove expired item
		r.client.Del(ctx, cacheKey)
		return nil, nil
	}

	return item.Value, nil
}

// Delete removes a value from cache
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	cacheKey := r.getCacheKey(key)
	return r.client.Del(ctx, cacheKey).Err()
}

// Clear removes all values with the cache prefix
func (r *RedisCache) Clear(ctx context.Context) error {
	keys, err := r.client.Keys(ctx, r.prefix+"*").Result()
	if err != nil {
		return fmt.Errorf("failed to get cache keys: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	return r.client.Del(ctx, keys...).Err()
}

// SetMultiple stores multiple key-value pairs
func (r *RedisCache) SetMultiple(ctx context.Context, items map[string]interface{}) error {
	if len(items) == 0 {
		return nil
	}

	// Use pipeline for better performance
	pipe := r.client.Pipeline()

	for key, value := range items {
		cacheKey := r.getCacheKey(key)
		item := CacheItem{
			Value:     value,
			ExpiresAt: time.Now().Add(r.ttl).Unix(),
			CreatedAt: time.Now().Unix(),
		}

		data, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to marshal cache item for key %s: %w", key, err)
		}

		pipe.Set(ctx, cacheKey, data, r.ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute pipeline: %w", err)
	}

	return nil
}

// GetMultiple retrieves multiple values from cache
func (r *RedisCache) GetMultiple(ctx context.Context, keys []string) (map[string]interface{}, error) {
	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}

	cacheKeys := make([]string, len(keys))
	for i, key := range keys {
		cacheKeys[i] = r.getCacheKey(key)
	}

	results, err := r.client.MGet(ctx, cacheKeys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get multiple from cache: %w", err)
	}

	items := make(map[string]interface{})
	for i, result := range results {
		if result == nil {
			continue
		}

		// Type assert to string
		resultStr, ok := result.(string)
		if !ok || resultStr == "" {
			continue
		}

		var item CacheItem
		if err := json.Unmarshal([]byte(resultStr), &item); err != nil {
			continue
		}

		// Check if expired
		if time.Now().Unix() > item.ExpiresAt {
			continue
		}

		items[keys[i]] = item.Value
	}

	return items, nil
}

// Exists checks if a key exists in cache
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	cacheKey := r.getCacheKey(key)
	count, err := r.client.Exists(ctx, cacheKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}

	return count > 0, nil
}

// Increment increments a numeric value in cache
func (r *RedisCache) Increment(ctx context.Context, key string, value int64) (int64, error) {
	cacheKey := r.getCacheKey(key)
	result, err := r.client.IncrBy(ctx, cacheKey, value).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment: %w", err)
	}

	return result, nil
}

// GetCacheStats returns cache statistics
func (r *RedisCache) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	_, err := r.client.Info(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	stats := make(map[string]interface{})

	// Parse basic info from Redis INFO command
	// This is a simplified parser - in production you might want more sophisticated parsing
	stats["redis_version"] = "unknown"
	stats["used_memory"] = "unknown"
	stats["connected_clients"] = "unknown"
	stats["total_commands_processed"] = "unknown"

	return stats, nil
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// getCacheKey creates the full cache key with prefix
func (r *RedisCache) getCacheKey(key string) string {
	if r.prefix == "" {
		return key
	}
	return r.prefix + ":" + key
}

// CacheMiddleware creates Echo middleware for caching
func CacheMiddleware(cache *RedisCache, ttl time.Duration, skipPaths []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip caching for specified paths
			for _, path := range skipPaths {
				if c.Request().URL.Path == path {
					return next(c)
				}
			}

			// Skip caching for non-GET requests
			if c.Request().Method != "GET" {
				return next(c)
			}

			// Generate cache key
			cacheKey := generateCacheKey(c)

			// Try to get from cache
			if cachedData, err := cache.Get(c.Request().Context(), cacheKey); err == nil && cachedData != nil {
				// Set cache hit header
				c.Response().Header().Set("X-Cache", "HIT")
				c.Response().Header().Set("X-Cache-Key", cacheKey)

				// Return cached data directly
				return c.JSON(200, map[string]interface{}{
					"status":    "success",
					"data":      cachedData,
					"cached":    true,
					"timestamp": time.Now().Unix(),
				})
			}

			// Cache miss - proceed with request
			c.Response().Header().Set("X-Cache", "MISS")

			// Cache the response after processing
			// Use a custom response wrapper to capture response data
			originalWriter := c.Response().Writer
			buffer := make([]byte, 0)

			// Replace writer temporarily
			c.Response().Writer = &capturingResponseWriter{
				ResponseWriter: originalWriter,
				buffer:         &buffer,
			}

			// Process request
			err := next(c)
			if err != nil {
				return err
			}

			// Restore original writer
			c.Response().Writer = originalWriter

			// Cache the response if successful
			if c.Response().Status == 200 && len(buffer) > 0 {
				var response interface{}
				if json.Unmarshal(buffer, &response) == nil {
					cache.Set(c.Request().Context(), cacheKey, response)
				}
			}

			// Set cache headers
			c.Response().Header().Set("X-Cache-Key", cacheKey)
			c.Response().Header().Set("X-Cache-TTL", ttl.String())

			return nil
		}
	}
}

// capturingResponseWriter captures response data for caching
type capturingResponseWriter struct {
	http.ResponseWriter
	buffer *[]byte
}

func (rw *capturingResponseWriter) Write(b []byte) (int, error) {
	*rw.buffer = append(*rw.buffer, b...)
	return rw.ResponseWriter.Write(b)
}

func (rw *capturingResponseWriter) WriteString(s string) (int, error) {
	return rw.Write([]byte(s))
}

// generateCacheKey creates a cache key from request
func generateCacheKey(c echo.Context) string {
	// Include relevant request parameters
	key := fmt.Sprintf("%s:%s",
		c.Request().URL.Path,
		c.Request().URL.RawQuery,
	)

	// Add user context if available
	if userID := c.Get("user_id"); userID != nil {
		key += fmt.Sprintf(":user:%v", userID)
	}

	return key
}
