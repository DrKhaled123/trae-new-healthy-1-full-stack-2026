package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisCache_GetSet(t *testing.T) {
	// Skip if Redis not available
	redisCache, err := NewRedisCache("localhost:6379", "", "test", 5*time.Minute)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer redisCache.Close()

	ctx := context.Background()
	key := "test-key"
	value := "test-value"

	// Set value
	err = redisCache.Set(ctx, key, value)
	require.NoError(t, err)

	// Get value
	retrieved, err := redisCache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestRedisCache_GetNonExistent(t *testing.T) {
	redisCache, err := NewRedisCache("localhost:6379", "", "test", 5*time.Minute)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer redisCache.Close()

	ctx := context.Background()
	key := "nonexistent-key"

	// Get non-existent value
	retrieved, err := redisCache.Get(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestRedisCache_Expiration(t *testing.T) {
	redisCache, err := NewRedisCache("localhost:6379", "", "test", 100*time.Millisecond)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer redisCache.Close()

	ctx := context.Background()
	key := "expire-key"
	value := "expire-value"

	// Set with short TTL
	err = redisCache.Set(ctx, key, value)
	require.NoError(t, err)

	// Should exist immediately
	retrieved, err := redisCache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	retrieved, err = redisCache.Get(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestRedisCache_Delete(t *testing.T) {
	redisCache, err := NewRedisCache("localhost:6379", "", "test", 5*time.Minute)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer redisCache.Close()

	ctx := context.Background()
	key := "delete-key"
	value := "delete-value"

	// Set value
	err = redisCache.Set(ctx, key, value)
	require.NoError(t, err)

	// Verify it exists
	retrieved, err := redisCache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Delete it
	err = redisCache.Delete(ctx, key)
	require.NoError(t, err)

	// Verify it's gone
	retrieved, err = redisCache.Get(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestRedisCache_Exists(t *testing.T) {
	redisCache, err := NewRedisCache("localhost:6379", "", "test", 5*time.Minute)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer redisCache.Close()

	ctx := context.Background()
	key := "exists-key"
	value := "exists-value"

	// Should not exist initially
	exists, err := redisCache.Exists(ctx, key)
	require.NoError(t, err)
	assert.False(t, exists)

	// Set value
	err = redisCache.Set(ctx, key, value)
	require.NoError(t, err)

	// Should exist now
	exists, err = redisCache.Exists(ctx, key)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestRedisCache_SetMultiple(t *testing.T) {
	redisCache, err := NewRedisCache("localhost:6379", "", "test", 5*time.Minute)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer redisCache.Close()

	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	// Set multiple values
	err = redisCache.SetMultiple(ctx, items)
	require.NoError(t, err)

	// Verify all values exist
	for key, expectedValue := range items {
		retrieved, err := redisCache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, expectedValue, retrieved)
	}
}

func TestRedisCache_GetMultiple(t *testing.T) {
	redisCache, err := NewRedisCache("localhost:6379", "", "test", 5*time.Minute)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer redisCache.Close()

	ctx := context.Background()

	// Set up some values
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err = redisCache.SetMultiple(ctx, items)
	require.NoError(t, err)

	// Get multiple values
	keys := []string{"key1", "key2", "key3", "nonexistent"}
	retrieved, err := redisCache.GetMultiple(ctx, keys)
	require.NoError(t, err)

	// Verify results
	assert.Equal(t, "value1", retrieved["key1"])
	assert.Equal(t, "value2", retrieved["key2"])
	assert.Equal(t, "value3", retrieved["key3"])
	assert.Nil(t, retrieved["nonexistent"])
}

func TestRedisCache_Increment(t *testing.T) {
	redisCache, err := NewRedisCache("localhost:6379", "", "test", 5*time.Minute)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer redisCache.Close()

	ctx := context.Background()
	key := "counter-key"

	// Increment from 0
	result, err := redisCache.Increment(ctx, key, 5)
	require.NoError(t, err)
	assert.Equal(t, int64(5), result)

	// Increment again
	result, err = redisCache.Increment(ctx, key, 3)
	require.NoError(t, err)
	assert.Equal(t, int64(8), result)
}

func TestRedisCache_Clear(t *testing.T) {
	redisCache, err := NewRedisCache("localhost:6379", "", "test", 5*time.Minute)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer redisCache.Close()

	ctx := context.Background()

	// Set up some values
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err = redisCache.SetMultiple(ctx, items)
	require.NoError(t, err)

	// Verify they exist
	for key := range items {
		exists, err := redisCache.Exists(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)
	}

	// Clear all with prefix
	err = redisCache.Clear(ctx)
	require.NoError(t, err)

	// Verify they're gone
	for key := range items {
		exists, err := redisCache.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists)
	}
}

func TestRedisCache_GetCacheStats(t *testing.T) {
	redisCache, err := NewRedisCache("localhost:6379", "", "test", 5*time.Minute)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer redisCache.Close()

	ctx := context.Background()

	// Get stats
	stats, err := redisCache.GetCacheStats(ctx)
	require.NoError(t, err)
	assert.NotNil(t, stats)
}

func TestRedisCache_PrefixHandling(t *testing.T) {
	// Test with different prefixes
	cache1, err := NewRedisCache("localhost:6379", "", "prefix1", 5*time.Minute)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer cache1.Close()

	cache2, err := NewRedisCache("localhost:6379", "", "prefix2", 5*time.Minute)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer cache2.Close()

	ctx := context.Background()
	key := "same-key"
	value1 := "value1"
	value2 := "value2"

	// Set same key in different caches
	err = cache1.Set(ctx, key, value1)
	require.NoError(t, err)

	err = cache2.Set(ctx, key, value2)
	require.NoError(t, err)

	// Verify they're separate
	retrieved1, err := cache1.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value1, retrieved1)

	retrieved2, err := cache2.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value2, retrieved2)
}

func TestRedisCache_ComplexTypes(t *testing.T) {
	redisCache, err := NewRedisCache("localhost:6379", "", "test", 5*time.Minute)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer redisCache.Close()

	ctx := context.Background()

	// Test complex data types
	complexValue := map[string]interface{}{
		"string":  "test string",
		"number":  42,
		"boolean": true,
		"array":   []int{1, 2, 3},
	}

	err = redisCache.Set(ctx, "complex-key", complexValue)
	require.NoError(t, err)

	retrieved, err := redisCache.Get(ctx, "complex-key")
	require.NoError(t, err)

	// Type assert to map for comparison
	retrievedMap, ok := retrieved.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, complexValue["string"], retrievedMap["string"])

	// Handle number type conversion (could be int64 or float64)
	switch v := retrievedMap["number"].(type) {
	case float64:
		assert.Equal(t, complexValue["number"], v)
	case int64:
		assert.Equal(t, complexValue["number"], float64(v))
	case int:
		assert.Equal(t, complexValue["number"], float64(v))
	default:
		t.Logf("Unexpected number type: %T (value: %v)", v, v)
		// For debugging, let's see what we actually got
		t.Logf("Expected: %v (%T), Got: %v (%T)", complexValue["number"], complexValue["number"], v, v)
		assert.Equal(t, complexValue["number"], v)
	}

	assert.Equal(t, complexValue["boolean"], retrievedMap["boolean"])
}

func TestRedisCache_ConnectionError(t *testing.T) {
	// Test with invalid Redis connection
	_, err := NewRedisCache("localhost:9999", "", "test", 5*time.Minute)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to Redis")
}

func TestCacheMiddleware_HitMiss(t *testing.T) {
	redisCache, err := NewRedisCache("localhost:6379", "", "test", 5*time.Minute)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer redisCache.Close()

	// Test cache middleware functionality
	ctx := context.Background()
	key := "middleware-test"
	value := map[string]interface{}{"data": "test"}

	// Set value
	err = redisCache.Set(ctx, key, value)
	require.NoError(t, err)

	// Get value
	retrieved, err := redisCache.Get(ctx, key)
	require.NoError(t, err)

	retrievedMap, ok := retrieved.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, value["data"], retrievedMap["data"])
}

func TestRedisCache_ConcurrentAccess(t *testing.T) {
	redisCache, err := NewRedisCache("localhost:6379", "", "test", 5*time.Minute)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer redisCache.Close()

	ctx := context.Background()

	// Test concurrent access
	done := make(chan bool, 10)

	// Start multiple goroutines setting and getting values
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := fmt.Sprintf("concurrent-key-%d", id)
			value := fmt.Sprintf("value-%d", id)

			// Set
			err := redisCache.Set(ctx, key, value)
			assert.NoError(t, err)

			// Get
			retrieved, err := redisCache.Get(ctx, key)
			assert.NoError(t, err)
			assert.Equal(t, value, retrieved)

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
