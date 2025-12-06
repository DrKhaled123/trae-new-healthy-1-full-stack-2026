package optimization

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// CacheManager provides intelligent caching for database queries and API responses
type CacheManager struct {
	redis   *redis.Client
	local   *LocalCache
	config  *CacheConfig
	metrics *CacheMetrics
}

// CacheConfig holds configuration for the cache manager
type CacheConfig struct {
	RedisAddr        string
	RedisPassword    string
	RedisDB          int
	DefaultTTL       time.Duration
	LocalCacheSize   int
	EnableMetrics    bool
	CompressionLevel int
	MaxValueSize     int64
}

// CacheMetrics holds Prometheus metrics for cache performance
type CacheMetrics struct {
	CacheHits     *prometheus.CounterVec
	CacheMisses   *prometheus.CounterVec
	CacheErrors   *prometheus.CounterVec
	CacheLatency  *prometheus.HistogramVec
	CacheSize     *prometheus.GaugeVec
	EvictionCount *prometheus.CounterVec
}

// LocalCache provides in-memory caching with LRU eviction
type LocalCache struct {
	data    map[string]*CacheItem
	usage   map[string]time.Time
	mu      sync.RWMutex
	maxSize int
	ttl     time.Duration
}

// CacheItem represents a cached item
type CacheItem struct {
	Value     interface{}
	ExpiresAt time.Time
	Size      int64
	Hits      int64
	CreatedAt time.Time
}

// CacheKey represents a structured cache key
type CacheKey struct {
	Namespace string
	Key       string
	Version   string
	TTL       time.Duration
}

// NewCacheManager creates a new cache manager instance
func NewCacheManager(config *CacheConfig) (*CacheManager, error) {
	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Redis connection failed, using local cache only: %v", err)
		rdb = nil
	}

	// Initialize local cache
	localCache := &LocalCache{
		data:    make(map[string]*CacheItem),
		usage:   make(map[string]time.Time),
		maxSize: config.LocalCacheSize,
		ttl:     config.DefaultTTL,
	}

	// Initialize metrics
	metrics := &CacheMetrics{
		CacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_type", "namespace"},
		),
		CacheMisses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_type", "namespace"},
		),
		CacheErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_errors_total",
				Help: "Total number of cache errors",
			},
			[]string{"cache_type", "error_type"},
		),
		CacheLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "cache_operation_duration_seconds",
				Help:    "Cache operation duration in seconds",
				Buckets: []float64{0.001, 0.01, 0.1, 1},
			},
			[]string{"cache_type", "operation"},
		),
		CacheSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "cache_size_items",
				Help: "Number of items in cache",
			},
			[]string{"cache_type"},
		),
		EvictionCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_evictions_total",
				Help: "Total number of cache evictions",
			},
			[]string{"cache_type", "reason"},
		),
	}

	cm := &CacheManager{
		redis:   rdb,
		local:   localCache,
		config:  config,
		metrics: metrics,
	}

	// Start background cleanup
	go cm.startCleanup()

	return cm, nil
}

// Get retrieves a value from cache
func (cm *CacheManager) Get(ctx context.Context, key CacheKey, dest interface{}) error {
	start := time.Now()
	defer func() {
		if cm.config.EnableMetrics {
			cm.metrics.CacheLatency.WithLabelValues("combined", "get").Observe(time.Since(start).Seconds())
		}
	}()

	fullKey := cm.buildKey(key)

	// Try local cache first
	if value, found := cm.local.Get(fullKey); found {
		if cm.config.EnableMetrics {
			cm.metrics.CacheHits.WithLabelValues("local", key.Namespace).Inc()
		}
		return cm.unmarshal(value, dest)
	}

	// Try Redis cache
	if cm.redis != nil {
		value, err := cm.redis.Get(ctx, fullKey).Result()
		if err == nil {
			if cm.config.EnableMetrics {
				cm.metrics.CacheHits.WithLabelValues("redis", key.Namespace).Inc()
			}

			// Store in local cache for faster access
			cm.local.Set(fullKey, value, key.TTL)

			return cm.unmarshal(value, dest)
		} else if err != redis.Nil {
			if cm.config.EnableMetrics {
				cm.metrics.CacheErrors.WithLabelValues("redis", "get_error").Inc()
			}
			log.Printf("Redis get error: %v", err)
		}
	}

	// Cache miss
	if cm.config.EnableMetrics {
		cm.metrics.CacheMisses.WithLabelValues("combined", key.Namespace).Inc()
	}

	return fmt.Errorf("cache miss")
}

// Set stores a value in cache
func (cm *CacheManager) Set(ctx context.Context, key CacheKey, value interface{}) error {
	start := time.Now()
	defer func() {
		if cm.config.EnableMetrics {
			cm.metrics.CacheLatency.WithLabelValues("combined", "set").Observe(time.Since(start).Seconds())
		}
	}()

	fullKey := cm.buildKey(key)

	// Marshal value
	data, err := cm.marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// Check size limit
	if int64(len(data)) > cm.config.MaxValueSize {
		if cm.config.EnableMetrics {
			cm.metrics.CacheErrors.WithLabelValues("combined", "value_too_large").Inc()
		}
		return fmt.Errorf("value too large: %d bytes", len(data))
	}

	// Store in local cache
	cm.local.Set(fullKey, data, key.TTL)

	// Store in Redis cache
	if cm.redis != nil {
		ttl := key.TTL
		if ttl == 0 {
			ttl = cm.config.DefaultTTL
		}

		err := cm.redis.Set(ctx, fullKey, data, ttl).Err()
		if err != nil {
			if cm.config.EnableMetrics {
				cm.metrics.CacheErrors.WithLabelValues("redis", "set_error").Inc()
			}
			log.Printf("Redis set error: %v", err)
		}
	}

	return nil
}

// Delete removes a value from cache
func (cm *CacheManager) Delete(ctx context.Context, key CacheKey) error {
	fullKey := cm.buildKey(key)

	// Delete from local cache
	cm.local.Delete(fullKey)

	// Delete from Redis cache
	if cm.redis != nil {
		err := cm.redis.Del(ctx, fullKey).Err()
		if err != nil {
			if cm.config.EnableMetrics {
				cm.metrics.CacheErrors.WithLabelValues("redis", "delete_error").Inc()
			}
			return fmt.Errorf("failed to delete from Redis: %w", err)
		}
	}

	return nil
}

// GetOrSet retrieves a value from cache or sets it using the provided function
func (cm *CacheManager) GetOrSet(ctx context.Context, key CacheKey, dest interface{}, fn func() (interface{}, error)) error {
	// Try to get from cache first
	err := cm.Get(ctx, key, dest)
	if err == nil {
		return nil // Cache hit
	}

	// Cache miss, call the function
	value, err := fn()
	if err != nil {
		return fmt.Errorf("function call failed: %w", err)
	}

	// Store in cache
	if err := cm.Set(ctx, key, value); err != nil {
		log.Printf("Failed to cache value: %v", err)
	}

	// Set the destination
	return cm.unmarshal(value, dest)
}

// InvalidatePattern invalidates all cache keys matching a pattern
func (cm *CacheManager) InvalidatePattern(ctx context.Context, pattern string) error {
	// Invalidate local cache
	cm.local.InvalidatePattern(pattern)

	// Invalidate Redis cache
	if cm.redis != nil {
		keys, err := cm.redis.Keys(ctx, pattern).Result()
		if err != nil {
			return fmt.Errorf("failed to get keys: %w", err)
		}

		if len(keys) > 0 {
			err = cm.redis.Del(ctx, keys...).Err()
			if err != nil {
				return fmt.Errorf("failed to delete keys: %w", err)
			}
		}
	}

	return nil
}

// LocalCache methods

// Get retrieves a value from local cache
func (lc *LocalCache) Get(key string) (interface{}, bool) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	item, exists := lc.data[key]
	if !exists {
		return nil, false
	}

	// Check expiration
	if time.Now().After(item.ExpiresAt) {
		delete(lc.data, key)
		delete(lc.usage, key)
		return nil, false
	}

	// Update usage
	lc.usage[key] = time.Now()
	item.Hits++

	return item.Value, true
}

// Set stores a value in local cache
func (lc *LocalCache) Set(key string, value interface{}, ttl time.Duration) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if ttl == 0 {
		ttl = lc.ttl
	}

	// Evict if necessary
	if len(lc.data) >= lc.maxSize {
		lc.evictLRU()
	}

	lc.data[key] = &CacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
		Size:      int64(len(fmt.Sprintf("%v", value))),
		CreatedAt: time.Now(),
	}
	lc.usage[key] = time.Now()
}

// Delete removes a value from local cache
func (lc *LocalCache) Delete(key string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	delete(lc.data, key)
	delete(lc.usage, key)
}

// InvalidatePattern removes all keys matching a pattern
func (lc *LocalCache) InvalidatePattern(pattern string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	for key := range lc.data {
		// Simple pattern matching (supports * wildcard)
		if matchPattern(key, pattern) {
			delete(lc.data, key)
			delete(lc.usage, key)
		}
	}
}

// evictLRU removes the least recently used item
func (lc *LocalCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, lastUsed := range lc.usage {
		if oldestKey == "" || lastUsed.Before(oldestTime) {
			oldestKey = key
			oldestTime = lastUsed
		}
	}

	if oldestKey != "" {
		delete(lc.data, oldestKey)
		delete(lc.usage, oldestKey)
	}
}

// Helper methods

// buildKey constructs a full cache key
func (cm *CacheManager) buildKey(key CacheKey) string {
	if key.Version != "" {
		return fmt.Sprintf("%s:%s:v%s", key.Namespace, key.Key, key.Version)
	}
	return fmt.Sprintf("%s:%s", key.Namespace, key.Key)
}

// marshal converts a value to bytes
func (cm *CacheManager) marshal(value interface{}) (string, error) {
	if str, ok := value.(string); ok {
		return str, nil
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// unmarshal converts bytes to a value
func (cm *CacheManager) unmarshal(data interface{}, dest interface{}) error {
	if str, ok := data.(string); ok {
		return json.Unmarshal([]byte(str), dest)
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(bytes, dest)
}

// matchPattern performs simple pattern matching
func matchPattern(text, pattern string) bool {
	// Simple implementation - supports * wildcard at the end
	if pattern == "*" {
		return true
	}

	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(text) >= len(prefix) && text[:len(prefix)] == prefix
	}

	return text == pattern
}

// startCleanup starts background cleanup of expired items
func (cm *CacheManager) startCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cm.cleanupExpired()
		cm.updateMetrics()
	}
}

// cleanupExpired removes expired items from local cache
func (cm *CacheManager) cleanupExpired() {
	cm.local.mu.Lock()
	defer cm.local.mu.Unlock()

	now := time.Now()
	expired := 0

	for key, item := range cm.local.data {
		if now.After(item.ExpiresAt) {
			delete(cm.local.data, key)
			delete(cm.local.usage, key)
			expired++
		}
	}

	if expired > 0 && cm.config.EnableMetrics {
		cm.metrics.EvictionCount.WithLabelValues("local", "expired").Add(float64(expired))
	}
}

// updateMetrics updates cache size metrics
func (cm *CacheManager) updateMetrics() {
	if !cm.config.EnableMetrics {
		return
	}

	cm.local.mu.RLock()
	localSize := len(cm.local.data)
	cm.local.mu.RUnlock()

	cm.metrics.CacheSize.WithLabelValues("local").Set(float64(localSize))

	// Get Redis info if available
	if cm.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := cm.redis.Info(ctx, "keyspace").Result()
		if err == nil {
			// Parse Redis keyspace info (simplified)
			cm.metrics.CacheSize.WithLabelValues("redis").Set(0) // Would need proper parsing
		}
	}
}

// GetStats returns cache statistics
func (cm *CacheManager) GetStats() map[string]interface{} {
	cm.local.mu.RLock()
	defer cm.local.mu.RUnlock()

	stats := map[string]interface{}{
		"local_cache_size": len(cm.local.data),
		"local_cache_max":  cm.local.maxSize,
		"default_ttl":      cm.config.DefaultTTL.String(),
	}

	// Add Redis stats if available
	if cm.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := cm.redis.Info(ctx).Result()
		if err == nil {
			stats["redis_connected"] = true
		} else {
			stats["redis_connected"] = false
			stats["redis_error"] = err.Error()
		}
	} else {
		stats["redis_connected"] = false
	}

	return stats
}
