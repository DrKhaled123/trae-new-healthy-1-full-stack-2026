// health_checks.go - Health checks for external dependencies and system resources
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/go-redis/redis/v8"
)

// HealthState represents the health state of a component
type HealthState int

const (
	StateHealthy HealthState = iota
	StateDegraded
	StateUnhealthy
)

// String returns the string representation of HealthState
func (s HealthState) String() string {
	switch s {
	case StateHealthy:
		return "healthy"
	case StateDegraded:
		return "degraded"
	case StateUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// HealthCheck represents a health check result
type HealthCheck struct {
	Name         string                 `json:"name"`
	Status       HealthState            `json:"status"`
	Message      string                 `json:"message"`
	ResponseTime time.Duration          `json:"response_time"`
	Timestamp    time.Time              `json:"timestamp"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// HealthChecker manages health checks for all dependencies
type HealthChecker struct {
	checks       map[string]HealthCheckFunc
	checkTimeout time.Duration
}

// HealthCheckFunc represents a function that performs a health check
type HealthCheckFunc func(ctx context.Context) *HealthCheck

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	hc := &HealthChecker{
		checks:       make(map[string]HealthCheckFunc),
		checkTimeout: 10 * time.Second, // Default timeout
	}

	hc.registerDefaultChecks()
	return hc
}

// registerDefaultChecks registers all default health checks
func (hc *HealthChecker) registerDefaultChecks() {
	// Database health check
	hc.RegisterCheck("database", hc.checkDatabase)

	// Redis health check
	hc.RegisterCheck("redis", hc.checkRedis)

	// System resources check
	hc.RegisterCheck("system", hc.checkSystemResources)

	// External API checks (if configured)
	if os.Getenv("EXTERNAL_API_URL") != "" {
		hc.RegisterCheck("external_api", hc.checkExternalAPI)
	}

	// Disk space check
	hc.RegisterCheck("disk_space", hc.checkDiskSpace)
}

// RegisterCheck registers a new health check
func (hc *HealthChecker) RegisterCheck(name string, checkFunc HealthCheckFunc) {
	hc.checks[name] = checkFunc
}

// RunAllChecks runs all registered health checks
func (hc *HealthChecker) RunAllChecks() map[string]*HealthCheck {
	results := make(map[string]*HealthCheck)

	for name, checkFunc := range hc.checks {
		ctx, cancel := context.WithTimeout(context.Background(), hc.checkTimeout)

		start := time.Now()
		result := checkFunc(ctx)
		result.ResponseTime = time.Since(start)
		result.Timestamp = time.Now()

		results[name] = result
		cancel()

		// Log unhealthy components
		if result.Status != StateHealthy {
			Logger.Warn("Health check failed", map[string]interface{}{
				"component":     name,
				"status":        result.Status.String(),
				"message":       result.Message,
				"response_time": result.ResponseTime.Milliseconds(),
			})
		}
	}

	return results
}

// RunCheck runs a specific health check
func (hc *HealthChecker) RunCheck(name string) *HealthCheck {
	checkFunc, exists := hc.checks[name]
	if !exists {
		return &HealthCheck{
			Name:    name,
			Status:  StateUnhealthy,
			Message: "Health check not found",
			Details: map[string]interface{}{"error": "check_not_registered"},
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), hc.checkTimeout)
	defer cancel()

	start := time.Now()
	result := checkFunc(ctx)
	result.ResponseTime = time.Since(start)
	result.Timestamp = time.Now()

	return result
}

// GetOverallHealth returns the overall health status
func (hc *HealthChecker) GetOverallHealth() *HealthCheck {
	results := hc.RunAllChecks()

	overallStatus := StateHealthy
	var unhealthyComponents []string
	var degradedComponents []string

	for name, result := range results {
		switch result.Status {
		case StateUnhealthy:
			overallStatus = StateUnhealthy
			unhealthyComponents = append(unhealthyComponents, name)
		case StateDegraded:
			if overallStatus == StateHealthy {
				overallStatus = StateDegraded
			}
			degradedComponents = append(degradedComponents, name)
		}
	}

	message := "All components healthy"
	if overallStatus != StateHealthy {
		message = fmt.Sprintf("Issues detected: unhealthy=%v, degraded=%v",
			unhealthyComponents, degradedComponents)
	}

	return &HealthCheck{
		Name:    "overall",
		Status:  overallStatus,
		Message: message,
		Details: map[string]interface{}{
			"total_checks":         len(results),
			"unhealthy_count":      len(unhealthyComponents),
			"degraded_count":       len(degradedComponents),
			"unhealthy_components": unhealthyComponents,
			"degraded_components":  degradedComponents,
			"check_results":        results,
		},
	}
}

// Individual health check implementations

func (hc *HealthChecker) checkDatabase(ctx context.Context) *HealthCheck {
	// This assumes you have a global database connection
	// In a real implementation, you'd inject the database connection
	if DB == nil {
		return &HealthCheck{
			Name:    "database",
			Status:  StateUnhealthy,
			Message: "Database connection not available",
		}
	}

	// Test database connectivity with a simple query
	err := DB.PingContext(ctx)
	if err != nil {
		return &HealthCheck{
			Name:    "database",
			Status:  StateUnhealthy,
			Message: fmt.Sprintf("Database ping failed: %v", err),
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	// Test a simple query to ensure database is responsive
	var result int
	err = DB.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return &HealthCheck{
			Name:    "database",
			Status:  StateDegraded,
			Message: fmt.Sprintf("Database query failed: %v", err),
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	if result != 1 {
		return &HealthCheck{
			Name:    "database",
			Status:  StateDegraded,
			Message: "Database returned unexpected result",
			Details: map[string]interface{}{
				"expected": 1,
				"actual":   result,
			},
		}
	}

	return &HealthCheck{
		Name:    "database",
		Status:  StateHealthy,
		Message: "Database connection healthy",
		Details: map[string]interface{}{
			"driver": "sqlite3", // This should be configurable
		},
	}
}

func (hc *HealthChecker) checkRedis(ctx context.Context) *HealthCheck {
	// Check if Redis is configured
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return &HealthCheck{
			Name:    "redis",
			Status:  StateHealthy,
			Message: "Redis not configured (optional dependency)",
		}
	}

	// Parse Redis URL and create client
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return &HealthCheck{
			Name:    "redis",
			Status:  StateUnhealthy,
			Message: fmt.Sprintf("Invalid Redis URL: %v", err),
		}
	}

	client := redis.NewClient(opt)
	defer client.Close()

	// Test Redis connectivity
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		return &HealthCheck{
			Name:    "redis",
			Status:  StateUnhealthy,
			Message: fmt.Sprintf("Redis ping failed: %v", err),
		}
	}

	if pong != "PONG" {
		return &HealthCheck{
			Name:    "redis",
			Status:  StateDegraded,
			Message: fmt.Sprintf("Redis returned unexpected pong: %s", pong),
		}
	}

	// Test Redis write/read
	testKey := "health_check_test"
	testValue := "ok"

	err = client.Set(ctx, testKey, testValue, 10*time.Second).Err()
	if err != nil {
		return &HealthCheck{
			Name:    "redis",
			Status:  StateDegraded,
			Message: fmt.Sprintf("Redis write failed: %v", err),
		}
	}

	result, err := client.Get(ctx, testKey).Result()
	if err != nil {
		return &HealthCheck{
			Name:    "redis",
			Status:  StateDegraded,
			Message: fmt.Sprintf("Redis read failed: %v", err),
		}
	}

	// Clean up test key
	client.Del(ctx, testKey)

	if result != testValue {
		return &HealthCheck{
			Name:    "redis",
			Status:  StateDegraded,
			Message: "Redis read returned unexpected value",
			Details: map[string]interface{}{
				"expected": testValue,
				"actual":   result,
			},
		}
	}

	return &HealthCheck{
		Name:    "redis",
		Status:  StateHealthy,
		Message: "Redis connection healthy",
	}
}

func (hc *HealthChecker) checkSystemResources(ctx context.Context) *HealthCheck {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check memory usage (warn if > 80%, critical if > 95%)
	memoryUsagePercent := float64(m.Alloc) / float64(m.Sys) * 100

	status := StateHealthy
	message := "System resources normal"

	if memoryUsagePercent > 95 {
		status = StateUnhealthy
		message = fmt.Sprintf("Critical memory usage: %.1f%%", memoryUsagePercent)
	} else if memoryUsagePercent > 80 {
		status = StateDegraded
		message = fmt.Sprintf("High memory usage: %.1f%%", memoryUsagePercent)
	}

	return &HealthCheck{
		Name:    "system",
		Status:  status,
		Message: message,
		Details: map[string]interface{}{
			"memory_alloc_mb":      m.Alloc / 1024 / 1024,
			"memory_sys_mb":        m.Sys / 1024 / 1024,
			"memory_usage_percent": memoryUsagePercent,
			"goroutines":           runtime.NumGoroutine(),
			"cpu_count":            runtime.NumCPU(),
		},
	}
}

func (hc *HealthChecker) checkExternalAPI(ctx context.Context) *HealthCheck {
	apiURL := os.Getenv("EXTERNAL_API_URL")
	if apiURL == "" {
		return &HealthCheck{
			Name:    "external_api",
			Status:  StateHealthy,
			Message: "External API not configured",
		}
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return &HealthCheck{
			Name:    "external_api",
			Status:  StateUnhealthy,
			Message: fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return &HealthCheck{
			Name:    "external_api",
			Status:  StateUnhealthy,
			Message: fmt.Sprintf("External API request failed: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HealthCheck{
			Name:    "external_api",
			Status:  StateDegraded,
			Message: fmt.Sprintf("External API returned status %d", resp.StatusCode),
			Details: map[string]interface{}{
				"status_code": resp.StatusCode,
				"url":         apiURL,
			},
		}
	}

	return &HealthCheck{
		Name:    "external_api",
		Status:  StateHealthy,
		Message: "External API healthy",
		Details: map[string]interface{}{
			"status_code": resp.StatusCode,
			"url":         apiURL,
		},
	}
}

func (hc *HealthChecker) checkDiskSpace(ctx context.Context) *HealthCheck {
	// This is a simplified disk space check
	// In production, you might want to use a more sophisticated approach

	logDir := "./logs"
	if logDirEnv := os.Getenv("LOG_DIR"); logDirEnv != "" {
		logDir = logDirEnv
	}

	// For now, just check if we can write to the log directory
	testFile := fmt.Sprintf("%s/health_check.tmp", logDir)
	err := os.WriteFile(testFile, []byte("health check"), 0644)
	if err != nil {
		return &HealthCheck{
			Name:    "disk_space",
			Status:  StateUnhealthy,
			Message: fmt.Sprintf("Cannot write to disk: %v", err),
			Details: map[string]interface{}{
				"directory": logDir,
			},
		}
	}

	// Clean up test file
	os.Remove(testFile)

	return &HealthCheck{
		Name:    "disk_space",
		Status:  StateHealthy,
		Message: "Disk space available",
		Details: map[string]interface{}{
			"directory": logDir,
		},
	}
}

// Global health checker instance
var HealthCheckerInstance *HealthChecker

// InitHealthChecks initializes the global health checker
func InitHealthChecks() {
	HealthCheckerInstance = NewHealthChecker()
	Logger.Info("Health checks initialized", map[string]interface{}{
		"check_count": len(HealthCheckerInstance.checks),
	})
}

// RunHealthChecks runs all health checks and returns results
func RunHealthChecks() map[string]*HealthCheck {
	if HealthCheckerInstance == nil {
		return map[string]*HealthCheck{
			"error": {
				Name:    "health_checker",
				Status:  StateUnhealthy,
				Message: "Health checker not initialized",
			},
		}
	}
	return HealthCheckerInstance.RunAllChecks()
}

// GetHealthStatus returns the overall health status
func GetHealthStatus() *HealthCheck {
	if HealthCheckerInstance == nil {
		return &HealthCheck{
			Name:    "health_checker",
			Status:  StateUnhealthy,
			Message: "Health checker not initialized",
		}
	}
	return HealthCheckerInstance.GetOverallHealth()
}

// Note: In a real implementation, you would need to:
// 1. Import the database/sql package and have a global db variable
// 2. Import the github.com/go-redis/redis/v8 package for Redis checks
// 3. Add proper disk space checking using syscall.Statfs on Unix systems
// 4. Add network connectivity checks for external services
// 5. Add CPU usage monitoring
// 6. Add database connection pool stats

// For now, this provides a basic framework that can be extended
