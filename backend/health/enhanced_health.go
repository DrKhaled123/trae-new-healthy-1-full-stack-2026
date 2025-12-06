package health

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// HealthStatus represents the health status
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusDegraded  HealthStatus = "degraded"
)

// HealthCheck defines interface for health checks
type HealthCheck interface {
	Name() string
	Check() HealthResult
}

// HealthResult represents the result of a health check
type HealthResult struct {
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Duration  time.Duration          `json:"duration_ms"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Critical  bool                   `json:"critical"`
}

// OverallHealth represents the overall health of the system
type OverallHealth struct {
	Status    HealthStatus            `json:"status"`
	Timestamp time.Time               `json:"timestamp"`
	Uptime    time.Duration           `json:"uptime"`
	Version   string                  `json:"version"`
	Checks    map[string]HealthResult `json:"checks"`
	Summary   HealthSummary           `json:"summary"`
	System    SystemInfo              `json:"system"`
	Metadata  map[string]interface{}  `json:"metadata,omitempty"`
}

// HealthSummary provides a summary of health checks
type HealthSummary struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Unhealthy int `json:"unhealthy"`
	Degraded  int `json:"degraded"`
	Critical  int `json:"critical"`
}

// SystemInfo provides system information
type SystemInfo struct {
	OS            string  `json:"os"`
	Arch          string  `json:"architecture"`
	GoVersion     string  `json:"go_version"`
	NumGoroutines int     `json:"num_goroutines"`
	MemoryUsage   MemInfo `json:"memory_usage"`
	CPUUsage      float64 `json:"cpu_usage_percent"`
}

// MemInfo provides memory information
type MemInfo struct {
	Alloc      uint64 `json:"alloc_mb"`
	TotalAlloc uint64 `json:"total_alloc_mb"`
	Sys        uint64 `json:"sys_mb"`
	NumGC      uint32 `json:"num_gc"`
}

// EnhancedHealthChecker manages health checks
type EnhancedHealthChecker struct {
	checks       []HealthCheck
	startTime    time.Time
	mu           sync.RWMutex
	criticalTime time.Duration
	cache        map[string]cachedHealth
	cacheTTL     time.Duration
}

type cachedHealth struct {
	result    HealthResult
	timestamp time.Time
}

// DatabaseCheck checks database connectivity
type DatabaseCheck struct {
	DB     *sql.DB
	GORMDB *gorm.DB
	name   string
}

func (dc *DatabaseCheck) Name() string {
	if dc.name != "" {
		return dc.name
	}
	return "database"
}

func (dc *DatabaseCheck) Check() HealthResult {
	start := time.Now()

	result := HealthResult{
		Timestamp: start,
		Critical:  true,
	}

	// Check SQL database
	if dc.DB != nil {
		if err := dc.DB.Ping(); err != nil {
			result.Status = StatusUnhealthy
			result.Message = fmt.Sprintf("Database ping failed: %v", err)
			result.Duration = time.Since(start)
			return result
		}

		// Get database stats
		stats := dc.DB.Stats()
		result.Details = map[string]interface{}{
			"open_connections": stats.OpenConnections,
			"in_use":           stats.InUse,
			"idle":             stats.Idle,
			"max_open":         stats.MaxOpenConnections,
		}
	}

	// Check GORM database
	if dc.GORMDB != nil {
		sqlDB, err := dc.GORMDB.DB()
		if err != nil {
			result.Status = StatusUnhealthy
			result.Message = fmt.Sprintf("Failed to get underlying SQL DB: %v", err)
			result.Duration = time.Since(start)
			return result
		}

		if err := sqlDB.Ping(); err != nil {
			result.Status = StatusUnhealthy
			result.Message = fmt.Sprintf("GORM database ping failed: %v", err)
			result.Duration = time.Since(start)
			return result
		}
	}

	result.Status = StatusHealthy
	result.Message = "Database connection is healthy"
	result.Duration = time.Since(start)
	return result
}

// FileSystemCheck checks file system accessibility
type FileSystemCheck struct {
	Paths []string
	name  string
}

func (fsc *FileSystemCheck) Name() string {
	if fsc.name != "" {
		return fsc.name
	}
	return "filesystem"
}

func (fsc *FileSystemCheck) Check() HealthResult {
	start := time.Now()

	result := HealthResult{
		Status:    StatusHealthy,
		Timestamp: start,
		Critical:  true,
		Details:   make(map[string]interface{}),
	}

	issues := []string{}
	for _, path := range fsc.Paths {
		info, err := os.Stat(path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("Path %s: %v", path, err))
			result.Details[path] = map[string]interface{}{
				"accessible": false,
				"error":      err.Error(),
			}
		} else {
			result.Details[path] = map[string]interface{}{
				"accessible": true,
				"size":       info.Size(),
				"modified":   info.ModTime(),
			}
		}
	}

	if len(issues) > 0 {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Filesystem issues: %s", strings.Join(issues, "; "))
	} else {
		result.Message = "All file system paths are accessible"
	}

	result.Duration = time.Since(start)
	return result
}

// MemoryCheck checks memory usage
type MemoryCheck struct {
	ThresholdMB uint64
	name        string
}

func (mc *MemoryCheck) Name() string {
	if mc.name != "" {
		return mc.name
	}
	return "memory"
}

func (mc *MemoryCheck) Check() HealthResult {
	start := time.Now()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := bToMb(m.Alloc)
	sysMB := bToMb(m.Sys)

	result := HealthResult{
		Status:    StatusHealthy,
		Timestamp: start,
		Critical:  false,
		Details: map[string]interface{}{
			"alloc_mb":        allocMB,
			"total_alloc_mb":  bToMb(m.TotalAlloc),
			"sys_mb":          sysMB,
			"num_gc":          m.NumGC,
			"gc_cpu_fraction": m.GCCPUFraction,
			"threshold_mb":    mc.ThresholdMB,
		},
	}

	if mc.ThresholdMB > 0 && allocMB > mc.ThresholdMB {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("Memory usage %dMB exceeds threshold %dMB", allocMB, mc.ThresholdMB)
		result.Critical = true
	} else {
		result.Message = "Memory usage is normal"
	}

	result.Duration = time.Since(start)
	return result
}

// GoroutineCheck checks goroutine count
type GoroutineCheck struct {
	MaxCount int
	name     string
}

func (gc *GoroutineCheck) Name() string {
	if gc.name != "" {
		return gc.name
	}
	return "goroutines"
}

func (gc *GoroutineCheck) Check() HealthResult {
	start := time.Now()
	count := runtime.NumGoroutine()

	result := HealthResult{
		Status:    StatusHealthy,
		Timestamp: start,
		Critical:  false,
		Details: map[string]interface{}{
			"count":     count,
			"max_count": gc.MaxCount,
		},
	}

	if gc.MaxCount > 0 && count > gc.MaxCount {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("Goroutine count %d exceeds maximum %d", count, gc.MaxCount)
		result.Critical = true
	} else {
		result.Message = fmt.Sprintf("Goroutine count is normal: %d", count)
	}

	result.Duration = time.Since(start)
	return result
}

// APICheck checks external API connectivity
type APICheck struct {
	URL     string
	Timeout time.Duration
	name    string
}

func (ac *APICheck) Name() string {
	if ac.name != "" {
		return ac.name
	}
	return "api"
}

func (ac *APICheck) Check() HealthResult {
	start := time.Now()

	client := &http.Client{
		Timeout: ac.Timeout,
	}

	resp, err := client.Get(ac.URL)
	if err != nil {
		result := HealthResult{
			Status:    StatusUnhealthy,
			Message:   fmt.Sprintf("API check failed: %v", err),
			Timestamp: start,
			Critical:  false,
			Duration:  time.Since(start),
			Details: map[string]interface{}{
				"url": ac.URL,
			},
		}
		return result
	}
	defer resp.Body.Close()

	result := HealthResult{
		Status:    StatusHealthy,
		Message:   fmt.Sprintf("API %s responded with status %d", ac.URL, resp.StatusCode),
		Timestamp: start,
		Critical:  false,
		Duration:  time.Since(start),
		Details: map[string]interface{}{
			"url":           ac.URL,
			"status_code":   resp.StatusCode,
			"response_time": time.Since(start).Milliseconds(),
		},
	}

	if resp.StatusCode >= 500 {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("API %s returned server error %d", ac.URL, resp.StatusCode)
		result.Critical = true
	}

	return result
}

// NewEnhancedHealthChecker creates a new enhanced health checker
func NewEnhancedHealthChecker() *EnhancedHealthChecker {
	return &EnhancedHealthChecker{
		checks:       []HealthCheck{},
		startTime:    time.Now(),
		criticalTime: 5 * time.Second,
		cache:        make(map[string]cachedHealth),
		cacheTTL:     30 * time.Second,
	}
}

// AddCheck adds a health check
func (ehc *EnhancedHealthChecker) AddCheck(check HealthCheck) {
	ehc.mu.Lock()
	defer ehc.mu.Unlock()
	ehc.checks = append(ehc.checks, check)
}

// CheckHealth performs all health checks
func (ehc *EnhancedHealthChecker) CheckHealth(c echo.Context) error {
	health := ehc.performHealthChecks()

	// Set appropriate HTTP status
	statusCode := http.StatusOK
	switch health.Status {
	case StatusUnhealthy:
		statusCode = http.StatusServiceUnavailable
	case StatusDegraded:
		statusCode = http.StatusPartialContent
	}

	return c.JSON(statusCode, health)
}

// CheckHealthLiveness performs quick liveness check
func (ehc *EnhancedHealthChecker) CheckHealthLiveness(c echo.Context) error {
	// Quick check - just check if the service is running
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    StatusHealthy,
		"timestamp": time.Now(),
		"uptime":    time.Since(ehc.startTime).String(),
	})
}

// CheckHealthReadiness performs readiness check
func (ehc *EnhancedHealthChecker) CheckHealthReadiness(c echo.Context) error {
	// More thorough check for readiness
	health := ehc.performHealthChecks()

	if health.Status == StatusHealthy {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    StatusHealthy,
			"timestamp": time.Now(),
			"ready":     true,
		})
	}

	return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
		"status":    health.Status,
		"timestamp": time.Now(),
		"ready":     false,
		"message":   "Service is not ready",
	})
}

// performHealthChecks executes all health checks
func (ehc *EnhancedHealthChecker) performHealthChecks() OverallHealth {
	ehc.mu.RLock()
	checks := make([]HealthCheck, len(ehc.checks))
	copy(checks, ehc.checks)
	ehc.mu.RUnlock()

	results := make(map[string]HealthResult)
	summary := HealthSummary{}

	for _, check := range checks {
		result := ehc.getCachedResult(check)
		results[check.Name()] = result

		summary.Total++
		switch result.Status {
		case StatusHealthy:
			summary.Healthy++
		case StatusUnhealthy:
			summary.Unhealthy++
		case StatusDegraded:
			summary.Degraded++
		}

		if result.Critical && (result.Status == StatusUnhealthy || result.Status == StatusDegraded) {
			summary.Critical++
		}
	}

	// Determine overall status
	status := StatusHealthy
	if summary.Critical > 0 {
		status = StatusUnhealthy
	} else if summary.Unhealthy > 0 {
		status = StatusDegraded
	} else if summary.Degraded > 0 {
		status = StatusDegraded
	}

	return OverallHealth{
		Status:    status,
		Timestamp: time.Now(),
		Uptime:    time.Since(ehc.startTime),
		Version:   "1.0.0", // This should come from config/build info
		Checks:    results,
		Summary:   summary,
		System:    getSystemInfo(),
	}
}

// getCachedResult returns cached result if available and not expired
func (ehc *EnhancedHealthChecker) getCachedResult(check HealthCheck) HealthResult {
	ehc.mu.Lock()
	defer ehc.mu.Unlock()

	if cached, exists := ehc.cache[check.Name()]; exists {
		if time.Since(cached.timestamp) < ehc.cacheTTL {
			return cached.result
		}
	}

	// Perform check and cache result
	result := check.Check()
	ehc.cache[check.Name()] = cachedHealth{
		result:    result,
		timestamp: time.Now(),
	}

	return result
}

// getSystemInfo collects system information
func getSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemInfo{
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		GoVersion:     runtime.Version(),
		NumGoroutines: runtime.NumGoroutine(),
		MemoryUsage: MemInfo{
			Alloc:      bToMb(m.Alloc),
			TotalAlloc: bToMb(m.TotalAlloc),
			Sys:        bToMb(m.Sys),
			NumGC:      m.NumGC,
		},
		CPUUsage: 0.0, // Would need more complex implementation
	}
}

// bToMb converts bytes to megabytes
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

// HealthCheckHandler creates health check endpoints
func (ehc *EnhancedHealthChecker) HealthCheckHandler() echo.HandlerFunc {
	return ehc.CheckHealth
}

// LivenessHandler creates liveness endpoint
func (ehc *EnhancedHealthChecker) LivenessHandler() echo.HandlerFunc {
	return ehc.CheckHealthLiveness
}

// ReadinessHandler creates readiness endpoint
func (ehc *EnhancedHealthChecker) ReadinessHandler() echo.HandlerFunc {
	return ehc.CheckHealthReadiness
}

// HealthMetricsHandler creates metrics endpoint for health checks
func (ehc *EnhancedHealthChecker) HealthMetricsHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		health := ehc.performHealthChecks()

		metrics := map[string]interface{}{
			"health_status":    string(health.Status),
			"uptime_seconds":   health.Uptime.Seconds(),
			"checks_total":     health.Summary.Total,
			"checks_healthy":   health.Summary.Healthy,
			"checks_unhealthy": health.Summary.Unhealthy,
			"checks_degraded":  health.Summary.Degraded,
			"checks_critical":  health.Summary.Critical,
			"goroutines":       health.System.NumGoroutines,
			"memory_alloc_mb":  health.System.MemoryUsage.Alloc,
			"memory_sys_mb":    health.System.MemoryUsage.Sys,
		}

		// Return as Prometheus metrics format if requested
		if c.Request().Header.Get("Accept") == "text/plain" {
			var builder strings.Builder
			for key, value := range metrics {
				switch v := value.(type) {
				case string:
					builder.WriteString(fmt.Sprintf("# HELP %s %s\n", key, key))
					builder.WriteString(fmt.Sprintf("# TYPE %s gauge\n", key))
					builder.WriteString(fmt.Sprintf("%s %s\n\n", key, v))
				case int, int64, float64:
					builder.WriteString(fmt.Sprintf("# HELP %s %s\n", key, key))
					builder.WriteString(fmt.Sprintf("# TYPE %s gauge\n", key))
					builder.WriteString(fmt.Sprintf("%s %v\n\n", key, v))
				}
			}
			return c.String(http.StatusOK, builder.String())
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "success",
			"data":   metrics,
		})
	}
}
