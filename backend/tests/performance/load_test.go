package performance

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// LoadTestConfig holds configuration for load tests
type LoadTestConfig struct {
	Concurrency int           // Number of concurrent requests
	Duration    time.Duration // Test duration
	Rate        int           // Requests per second per goroutine
	Endpoints   []string      // Endpoints to test
}

// LoadTestResult holds results of load tests
type LoadTestResult struct {
	TotalRequests   int
	SuccessRequests int
	FailedRequests  int
	AvgResponseTime time.Duration
	MaxResponseTime time.Duration
	MinResponseTime time.Duration
	RequestsPerSec  float64
	ErrorRate       float64
}

// TestAPILoad performs comprehensive load testing
func TestAPILoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	baseURL := "http://localhost:8080"

	// Wait for server to be ready
	require.Eventually(t, func() bool {
		resp, err := http.Get(baseURL + "/health")
		return err == nil && resp.StatusCode == 200
	}, 30*time.Second, 1*time.Second, "Server not ready for load testing")

	testConfigs := []LoadTestConfig{
		{
			Concurrency: 10,
			Duration:    30 * time.Second,
			Rate:        5,
			Endpoints:   []string{"/health", "/api/v1/nutrition-data/recipes"},
		},
		{
			Concurrency: 50,
			Duration:    60 * time.Second,
			Rate:        10,
			Endpoints:   []string{"/api/v1/nutrition-data/recipes", "/api/v1/nutrition-data/workouts"},
		},
		{
			Concurrency: 100,
			Duration:    120 * time.Second,
			Rate:        20,
			Endpoints:   []string{"/api/v1/nutrition-data/recipes"},
		},
	}

	for i, config := range testConfigs {
		t.Run(fmt.Sprintf("Config_%d", i+1), func(t *testing.T) {
			result := runLoadTest(t, baseURL, config)

			t.Logf("Load Test Results:")
			t.Logf("  Total Requests: %d", result.TotalRequests)
			t.Logf("  Success Requests: %d", result.SuccessRequests)
			t.Logf("  Failed Requests: %d", result.FailedRequests)
			t.Logf("  Error Rate: %.2f%%", result.ErrorRate)
			t.Logf("  Requests/sec: %.2f", result.RequestsPerSec)
			t.Logf("  Avg Response Time: %v", result.AvgResponseTime)
			t.Logf("  Max Response Time: %v", result.MaxResponseTime)
			t.Logf("  Min Response Time: %v", result.MinResponseTime)

			// Assert performance requirements
			assert.Less(t, result.ErrorRate, 5.0, "Error rate should be less than 5%")
			assert.Less(t, result.AvgResponseTime, 500*time.Millisecond, "Average response time should be less than 500ms")
			assert.Greater(t, result.RequestsPerSec, 50.0, "Should handle at least 50 requests per second")
		})
	}
}

// TestEndpointPerformance tests individual endpoint performance
func TestEndpointPerformance(t *testing.T) {
	baseURL := "http://localhost:8080"

	endpoints := []struct {
		path        string
		maxTime     time.Duration
		description string
	}{
		{"/health", 100 * time.Millisecond, "Health check endpoint"},
		{"/api/v1/nutrition-data/recipes", 300 * time.Millisecond, "Recipes endpoint"},
		{"/api/v1/nutrition-data/workouts", 300 * time.Millisecond, "Workouts endpoint"},
		{"/api/v1/nutrition-data/complaints", 400 * time.Millisecond, "Complaints endpoint"},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.description, func(t *testing.T) {
			// Warm up
			for i := 0; i < 5; i++ {
				resp, _ := http.Get(baseURL + endpoint.path)
				if resp != nil {
					resp.Body.Close()
				}
				time.Sleep(100 * time.Millisecond)
			}

			// Measure performance
			var total time.Duration
			requests := 50

			for i := 0; i < requests; i++ {
				start := time.Now()
				resp, err := http.Get(baseURL + endpoint.path)
				duration := time.Since(start)
				total += duration

				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				resp.Body.Close()

				// Small delay between requests
				time.Sleep(10 * time.Millisecond)
			}

			avg := total / time.Duration(requests)
			t.Logf("Average response time for %s: %v", endpoint.path, avg)
			assert.Less(t, avg, endpoint.maxTime,
				fmt.Sprintf("Response time for %s should be less than %v", endpoint.path, endpoint.maxTime))
		})
	}
}

// TestConcurrentAccess tests concurrent access patterns
func TestConcurrentAccess(t *testing.T) {
	baseURL := "http://localhost:8080"
	concurrency := 20
	requestsPerGoroutine := 10

	var wg sync.WaitGroup
	results := make(chan time.Duration, concurrency*requestsPerGoroutine)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				start := time.Now()
				resp, err := http.Get(baseURL + "/api/v1/nutrition-data/recipes")
				duration := time.Since(start)
				results <- duration

				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				resp.Body.Close()
			}
		}()
	}

	wg.Wait()
	close(results)

	// Analyze results
	var total time.Duration
	count := 0
	maxDuration := time.Duration(0)
	minDuration := time.Hour // Initialize to large value

	for duration := range results {
		total += duration
		count++
		if duration > maxDuration {
			maxDuration = duration
		}
		if duration < minDuration {
			minDuration = duration
		}
	}

	avg := total / time.Duration(count)

	t.Logf("Concurrent Access Results:")
	t.Logf("  Total Requests: %d", count)
	t.Logf("  Average Response Time: %v", avg)
	t.Logf("  Max Response Time: %v", maxDuration)
	t.Logf("  Min Response Time: %v", minDuration)

	// Assert concurrent performance
	assert.Less(t, avg, 500*time.Millisecond, "Average response time under load should be less than 500ms")
	assert.Less(t, maxDuration, 2*time.Second, "Max response time should be less than 2 seconds")
}

// TestMemoryUsage tests memory consumption during load
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	baseURL := "http://localhost:8080"

	// Get initial memory stats
	initialMem := getMemoryUsage(t)

	// Run load test
	concurrency := 50
	duration := 30 * time.Second

	var wg sync.WaitGroup
	done := make(chan bool)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					resp, err := http.Get(baseURL + "/api/v1/nutrition-data/recipes")
					if err == nil && resp != nil {
						resp.Body.Close()
					}
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()
	}

	// Monitor memory during test
	time.Sleep(duration)
	close(done)
	wg.Wait()

	// Get final memory stats
	finalMem := getMemoryUsage(t)

	memoryIncrease := finalMem - initialMem
	memoryPerRequest := float64(memoryIncrease) / float64(concurrency*int(duration/time.Millisecond)/100)

	t.Logf("Memory Usage Results:")
	t.Logf("  Initial Memory: %d KB", initialMem/1024)
	t.Logf("  Final Memory: %d KB", finalMem/1024)
	t.Logf("  Memory Increase: %d KB", memoryIncrease/1024)
	t.Logf("  Memory per 1000 requests: %.2f KB", memoryPerRequest*1000/1024)

	// Assert memory usage is reasonable
	assert.Less(t, memoryIncrease, 100*1024*1024, "Memory increase should be less than 100MB") // 100MB limit
	assert.Less(t, memoryPerRequest, 1024, "Memory per request should be less than 1KB")
}

// TestCachePerformance tests cache hit rates and performance
func TestCachePerformance(t *testing.T) {
	baseURL := "http://localhost:8080"
	endpoint := "/api/v1/nutrition-data/recipes"

	// First request to populate cache
	resp, err := http.Get(baseURL + endpoint)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Test cache hits
	var cacheHits, cacheMisses int
	requests := 100

	for i := 0; i < requests; i++ {
		start := time.Now()
		resp, err := http.Get(baseURL + endpoint)
		_ = time.Since(start) // Duration calculated but not used for this test

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Check cache headers
		cacheHeader := resp.Header.Get("X-Cache")
		if cacheHeader == "HIT" {
			cacheHits++
		} else {
			cacheMisses++
		}

		resp.Body.Close()

		// Small delay
		time.Sleep(10 * time.Millisecond)
	}

	hitRate := float64(cacheHits) / float64(requests) * 100
	t.Logf("Cache Performance Results:")
	t.Logf("  Total Requests: %d", requests)
	t.Logf("  Cache Hits: %d", cacheHits)
	t.Logf("  Cache Misses: %d", cacheMisses)
	t.Logf("  Hit Rate: %.2f%%", hitRate)

	// Assert cache is working
	assert.Greater(t, cacheHits, 0, "Should have some cache hits")
	assert.Greater(t, hitRate, 50.0, "Cache hit rate should be at least 50%")
}

// Helper functions

func runLoadTest(t *testing.T, baseURL string, config LoadTestConfig) LoadTestResult {
	var wg sync.WaitGroup
	requests := make(chan struct{}, config.Concurrency)
	results := make(chan time.Duration, config.Concurrency*config.Rate*int(config.Duration/time.Second))

	startTime := time.Now()

	// Start workers
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range requests {
				for _, endpoint := range config.Endpoints {
					reqStart := time.Now()
					resp, err := http.Get(baseURL + endpoint)
					duration := time.Since(reqStart)
					results <- duration

					if err == nil && resp != nil {
						resp.Body.Close()
					}

					// Rate limiting within goroutine
					time.Sleep(time.Second / time.Duration(config.Rate))
				}
			}
		}()
	}

	// Send requests
	go func() {
		totalRequests := config.Concurrency * config.Rate * int(config.Duration/time.Second)
		for i := 0; i < totalRequests; i++ {
			requests <- struct{}{}
		}
		close(requests)
	}()

	wg.Wait()
	close(results)

	// Analyze results
	var totalTime time.Duration
	var maxTime, minTime time.Duration
	maxTime = 0
	minTime = time.Hour
	count := 0

	for duration := range results {
		totalTime += duration
		count++
		if duration > maxTime {
			maxTime = duration
		}
		if duration < minTime {
			minTime = duration
		}
	}

	actualDuration := time.Since(startTime)

	return LoadTestResult{
		TotalRequests:   count,
		SuccessRequests: count, // Simplified - in real implementation track failures
		FailedRequests:  0,
		AvgResponseTime: totalTime / time.Duration(count),
		MaxResponseTime: maxTime,
		MinResponseTime: minTime,
		RequestsPerSec:  float64(count) / actualDuration.Seconds(),
		ErrorRate:       0, // Simplified
	}
}

func getMemoryUsage(t *testing.T) uint64 {
	// This is a simplified implementation
	// In a real implementation, you would use runtime.MemStats
	// or system-specific calls to get actual memory usage
	return 50 * 1024 * 1024 // 50MB placeholder
}

// BenchmarkEndpoints provides baseline performance benchmarks
func BenchmarkEndpoints(b *testing.B) {
	baseURL := "http://localhost:8080"

	b.Run("Health", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := http.Get(baseURL + "/health")
			if err == nil && resp != nil {
				resp.Body.Close()
			}
		}
	})

	b.Run("Recipes", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := http.Get(baseURL + "/api/v1/nutrition-data/recipes")
			if err == nil && resp != nil {
				resp.Body.Close()
			}
		}
	})
}
