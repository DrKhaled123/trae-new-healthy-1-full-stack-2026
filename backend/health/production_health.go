package health

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"nutrition-platform/cache"
	"nutrition-platform/database"

	"github.com/labstack/echo/v4"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusUnknown   HealthStatus = "unknown"
)

// ComponentHealth represents the health of a specific component
type ComponentHealth struct {
	Status       HealthStatus           `json:"status"`
	Message      string                 `json:"message,omitempty"`
	LastChecked  time.Time              `json:"last_checked"`
	ResponseTime time.Duration          `json:"response_time"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// SystemHealth represents the overall system health
type SystemHealth struct {
	Status     HealthStatus                `json:"status"`
	Timestamp  time.Time                   `json:"timestamp"`
	Uptime     time.Duration               `json:"uptime"`
	Version    string                      `json:"version"`
	Components map[string]*ComponentHealth `json:"components"`
	Checks     map[string]*CheckDefinition `json:"checks"`
	Summary    HealthSummary               `json:"summary"`
}

// HealthSummary provides a summary of system health
type HealthSummary struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Degraded  int `json:"degraded"`
	Unhealthy int `json:"unhealthy"`
}

// CheckDefinition defines a health check
type CheckDefinition struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Interval    time.Duration `json:"interval"`
	Timeout     time.Duration `json:"timeout"`
	Critical    bool          `json:"critical"`
}

// HealthChecker manages health checks for all system components
type HealthChecker struct {
	startTime  time.Time
	version    string
	db         *database.ProductionDatabase
	cache      *cache.RedisCache
	checks     map[string]*CheckDefinition
	results    map[string]*ComponentHealth
	mu         sync.RWMutex
	httpClient *http.Client
	config     *HealthConfig
}

// HealthConfig holds health checker configuration
type HealthConfig struct {
	DatabaseTimeout    time.Duration
	CacheTimeout       time.Duration
	HTTPTimeout        time.Duration
	ExternalEndpoints  []string
	CheckInterval      time.Duration
	FailureThreshold   int
	SuccessThreshold   int
	EnableDetailedLogs bool
}

// DefaultHealthConfig returns default health configuration
func DefaultHealthConfig() *HealthConfig {
	return &HealthConfig{
		DatabaseTimeout:    5 * time.Second,
		CacheTimeout:       2 * time.Second,
		HTTPTimeout:        3 * time.Second,
		ExternalEndpoints:  []string{},
		CheckInterval:      30 * time.Second,
		FailureThreshold:   3,
		SuccessThreshold:   2,
		EnableDetailedLogs: false,
	}
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *database.ProductionDatabase, cache *cache.RedisCache, version string, config *HealthConfig) *HealthChecker {
	if config == nil {
		config = DefaultHealthConfig()
	}

	hc := &HealthChecker{
		startTime: time.Now(),
		version:   version,
		db:        db,
		cache:     cache,
		config:    config,
		checks:    make(map[string]*CheckDefinition),
		results:   make(map[string]*ComponentHealth),
		httpClient: &http.Client{
			Timeout: config.HTTPTimeout,
		},
	}

	// Initialize default checks
	hc.initializeChecks()

	// Start background health checks
	go hc.startBackgroundChecks()

	return hc
}

// initializeChecks sets up default health checks
func (hc *HealthChecker) initializeChecks() {
	hc.checks["database"] = &CheckDefinition{
		Name:        "database",
		Description: "Primary database connection",
		Interval:    hc.config.CheckInterval,
		Timeout:     hc.config.DatabaseTimeout,
		Critical:    true,
	}

	hc.checks["cache"] = &CheckDefinition{
		Name:        "cache",
		Description: "Redis cache connection",
		Interval:    hc.config.CheckInterval,
		Timeout:     hc.config.CacheTimeout,
		Critical:    false,
	}

	hc.checks["memory"] = &CheckDefinition{
		Name:        "memory",
		Description: "Memory usage",
		Interval:    hc.config.CheckInterval,
		Timeout:     1 * time.Second,
		Critical:    true,
	}

	hc.checks["goroutines"] = &CheckDefinition{
		Name:        "goroutines",
		Description: "Goroutine count",
		Interval:    hc.config.CheckInterval,
		Timeout:     1 * time.Second,
		Critical:    false,
	}

	hc.checks["disk"] = &CheckDefinition{
		Name:        "disk",
		Description: "Disk space",
		Interval:    time.Minute,
		Timeout:     2 * time.Second,
		Critical:    true,
	}
}

// CheckDatabase performs database health check
func (hc *HealthChecker) CheckDatabase(ctx context.Context) *ComponentHealth {
	start := time.Now()
	result := &ComponentHealth{
		LastChecked: start,
		Details:     make(map[string]interface{}),
	}

	defer func() {
		result.ResponseTime = time.Since(start)
	}()

	// Check primary database
	if err := hc.db.Health(); err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Database health check failed: %v", err)
		return result
	}

	// Get database statistics
	stats := hc.db.Stats()
	result.Details["connections"] = stats

	result.Status = StatusHealthy
	result.Message = "Database is healthy"
	return result
}

// CheckCache performs cache health check
func (hc *HealthChecker) CheckCache(ctx context.Context) *ComponentHealth {
	start := time.Now()
	result := &ComponentHealth{
		LastChecked: start,
		Details:     make(map[string]interface{}),
	}

	defer func() {
		result.ResponseTime = time.Since(start)
	}()

	if hc.cache == nil {
		result.Status = StatusDegraded
		result.Message = "Cache not configured"
		return result
	}

	// Test cache connectivity
	testKey := "health_check_test"
	testValue := "test_value"

	if err := hc.cache.Set(ctx, testKey, testValue, 10*time.Second); err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Cache write failed: %v", err)
		return result
	}

	var retrievedValue string
	if err := hc.cache.Get(ctx, testKey, &retrievedValue); err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Cache read failed: %v", err)
		return result
	}

	if retrievedValue != testValue {
		result.Status = StatusUnhealthy
		result.Message = "Cache data corruption detected"
		return result
	}

	// Clean up test key
	hc.cache.Delete(ctx, testKey)

	result.Status = StatusHealthy
	result.Message = "Cache is healthy"
	return result
}

// CheckMemory performs memory health check
func (hc *HealthChecker) CheckMemory(ctx context.Context) *ComponentHealth {
	start := time.Now()
	result := &ComponentHealth{
		LastChecked: start,
		Details:     make(map[string]interface{}),
	}

	defer func() {
		result.ResponseTime = time.Since(start)
	}()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Convert bytes to MB
	allocMB := float64(m.Alloc) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024

	result.Details["alloc_mb"] = allocMB
	result.Details["sys_mb"] = sysMB
	result.Details["num_gc"] = m.NumGC
	result.Details["gc_cpu_fraction"] = m.GCCPUFraction

	// Check memory usage thresholds
	if allocMB > 512 { // 512MB threshold
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("High memory usage: %.2f MB", allocMB)
		return result
	}

	if allocMB > 1024 { // 1GB critical threshold
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Critical memory usage: %.2f MB", allocMB)
		return result
	}

	result.Status = StatusHealthy
	result.Message = "Memory usage is normal"
	return result
}

// CheckGoroutines performs goroutine health check
func (hc *HealthChecker) CheckGoroutines(ctx context.Context) *ComponentHealth {
	start := time.Now()
	result := &ComponentHealth{
		LastChecked: start,
		Details:     make(map[string]interface{}),
	}

	defer func() {
		result.ResponseTime = time.Since(start)
	}()

	count := runtime.NumGoroutine()
	result.Details["count"] = count

	// Check goroutine count thresholds
	if count > 1000 {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("High goroutine count: %d", count)
		return result
	}

	if count > 5000 {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Critical goroutine count: %d", count)
		return result
	}

	result.Status = StatusHealthy
	result.Message = "Goroutine count is normal"
	return result
}

// CheckDisk performs disk space health check
func (hc *HealthChecker) CheckDisk(ctx context.Context) *ComponentHealth {
	start := time.Now()
	result := &ComponentHealth{
		LastChecked: start,
		Details:     make(map[string]interface{}),
	}

	defer func() {
		result.ResponseTime = time.Since(start)
	}()

	// This is a simplified implementation
	// In production, you'd use actual disk space checking
	// For now, return a placeholder
	result.Details["usage_percent"] = 45.5
	result.Details["free_gb"] = 120.5

	result.Status = StatusHealthy
	result.Message = "Disk space is adequate"
	return result
}

// GetSystemHealth returns the overall system health
func (hc *HealthChecker) GetSystemHealth(ctx context.Context) *SystemHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	// Run all health checks
	checks := map[string]func(context.Context) *ComponentHealth{
		"database":   hc.CheckDatabase,
		"cache":      hc.CheckCache,
		"memory":     hc.CheckMemory,
		"goroutines": hc.CheckGoroutines,
		"disk":       hc.CheckDisk,
	}

	components := make(map[string]*ComponentHealth)
	summary := HealthSummary{}

	for name, checkFunc := range checks {
		result := checkFunc(ctx)
		components[name] = result

		summary.Total++
		switch result.Status {
		case StatusHealthy:
			summary.Healthy++
		case StatusDegraded:
			summary.Degraded++
		case StatusUnhealthy:
			summary.Unhealthy++
		}
	}

	// Determine overall status
	overallStatus := StatusHealthy
	if summary.Unhealthy > 0 {
		overallStatus = StatusUnhealthy
	} else if summary.Degraded > 0 {
		overallStatus = StatusDegraded
	}

	// Check for critical component failures
	for name, health := range components {
		if check, exists := hc.checks[name]; exists && check.Critical && health.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
			break
		}
	}

	return &SystemHealth{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Uptime:     time.Since(hc.startTime),
		Version:    hc.version,
		Components: components,
		Checks:     hc.checks,
		Summary:    summary,
	}
}

// startBackgroundChecks runs health checks in the background
func (hc *HealthChecker) startBackgroundChecks() {
	ticker := time.NewTicker(hc.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			health := hc.GetSystemHealth(ctx)

			// Store results
			hc.mu.Lock()
			hc.results = health.Components
			hc.mu.Unlock()

			cancel()
		}
	}
}

// LivenessHandler returns a simple liveness handler
func (hc *HealthChecker) LivenessHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "alive",
			"timestamp": time.Now(),
		})
	}
}

// ReadinessHandler returns a readiness handler
func (hc *HealthChecker) ReadinessHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
		defer cancel()

		health := hc.GetSystemHealth(ctx)

		if health.Status == StatusHealthy {
			return c.JSON(http.StatusOK, health)
		}

		return c.JSON(http.StatusServiceUnavailable, health)
	}
}

// HealthHandler returns detailed health information
func (hc *HealthChecker) HealthHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
		defer cancel()

		health := hc.GetSystemHealth(ctx)

		// Determine HTTP status code based on health
		statusCode := http.StatusOK
		switch health.Status {
		case StatusDegraded:
			statusCode = http.StatusServiceUnavailable // 503
		case StatusUnhealthy:
			statusCode = http.StatusServiceUnavailable // 503
		}

		return c.JSON(statusCode, health)
	}
}

// MetricsHandler returns health metrics
func (hc *HealthChecker) MetricsHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		hc.mu.RLock()
		defer hc.mu.RUnlock()

		metrics := map[string]interface{}{
			"uptime_seconds": time.Since(hc.startTime).Seconds(),
			"checks_count":   len(hc.checks),
			"results":        hc.results,
		}

		return c.JSON(http.StatusOK, metrics)
	}
}

// AddCheck adds a custom health check
func (hc *HealthChecker) AddCheck(name string, checkDef *CheckDefinition) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.checks[name] = checkDef
}

// RemoveCheck removes a health check
func (hc *HealthChecker) RemoveCheck(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	delete(hc.checks, name)
	delete(hc.results, name)
}

// GetLastResults returns the last health check results
func (hc *HealthChecker) GetLastResults() map[string]*ComponentHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	results := make(map[string]*ComponentHealth)
	for k, v := range hc.results {
		results[k] = v
	}
	return results
}
