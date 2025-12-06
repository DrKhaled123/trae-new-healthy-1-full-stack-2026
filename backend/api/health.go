package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HealthChecker provides comprehensive health checking capabilities
type HealthChecker struct {
	db          *sql.DB
	redisClient *redis.Client
	checks      map[string]HealthCheck
	mu          sync.RWMutex
	config      HealthConfig
	metrics     *HealthMetrics
	lastResults map[string]HealthResult
	fallbacks   map[string]FallbackHandler
}

// HealthConfig holds health check configuration
type HealthConfig struct {
	Timeout           time.Duration `json:"timeout"`
	Interval          time.Duration `json:"interval"`
	RetryAttempts     int           `json:"retry_attempts"`
	RetryDelay        time.Duration `json:"retry_delay"`
	CriticalThreshold float64       `json:"critical_threshold"`
	WarningThreshold  float64       `json:"warning_threshold"`
	EnableMetrics     bool          `json:"enable_metrics"`
	EnableFallbacks   bool          `json:"enable_fallbacks"`
	MaxUploadSize     int64         `json:"max_upload_size"`
	AllowedMimeTypes  []string      `json:"allowed_mime_types"`
	StoragePaths      []string      `json:"storage_paths"`
}

// HealthCheck defines a health check function
type HealthCheck func(ctx context.Context) HealthResult

// FallbackHandler defines a fallback mechanism
type FallbackHandler func(ctx context.Context, err error) error

// HealthResult represents the result of a health check
type HealthResult struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message"`
	Duration    time.Duration          `json:"duration"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Error       string                 `json:"error,omitempty"`
	RetryCount  int                    `json:"retry_count,omitempty"`
	LastSuccess time.Time              `json:"last_success,omitempty"`
}

// HealthStatus represents the status of a health check
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthResponse represents the overall health response
type HealthResponse struct {
	Status      HealthStatus            `json:"status"`
	Timestamp   time.Time               `json:"timestamp"`
	Duration    time.Duration           `json:"duration"`
	Version     string                  `json:"version"`
	Environment string                  `json:"environment"`
	Checks      map[string]HealthResult `json:"checks"`
	System      SystemInfo              `json:"system"`
	Uptime      time.Duration           `json:"uptime"`
	Metrics     map[string]interface{}  `json:"metrics,omitempty"`
	Fallbacks   map[string]FallbackInfo `json:"fallbacks,omitempty"`
}

// SystemInfo provides system-level information
type SystemInfo struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	Goroutines  int     `json:"goroutines"`
	GCStats     GCStats `json:"gc_stats"`
}

// GCStats provides garbage collection statistics
type GCStats struct {
	NumGC      uint32        `json:"num_gc"`
	PauseTotal time.Duration `json:"pause_total"`
	LastGC     time.Time     `json:"last_gc"`
	HeapSize   uint64        `json:"heap_size"`
	HeapInUse  uint64        `json:"heap_in_use"`
	StackInUse uint64        `json:"stack_in_use"`
}

// FallbackInfo provides information about fallback mechanisms
type FallbackInfo struct {
	Enabled     bool      `json:"enabled"`
	LastUsed    time.Time `json:"last_used,omitempty"`
	UsageCount  int       `json:"usage_count"`
	Description string    `json:"description"`
}

// HealthMetrics tracks health check metrics
type HealthMetrics struct {
	healthCheckDuration prometheus.HistogramVec
	healthCheckTotal    prometheus.CounterVec
	healthCheckStatus   prometheus.GaugeVec
	systemMetrics       prometheus.GaugeVec
	uploadMetrics       prometheus.CounterVec
	fallbackMetrics     prometheus.CounterVec
}

// Upload limits and validation
type UploadLimits struct {
	MaxFileSize       int64    `json:"max_file_size"`
	MaxTotalSize      int64    `json:"max_total_size"`
	MaxFiles          int      `json:"max_files"`
	AllowedMimeTypes  []string `json:"allowed_mime_types"`
	AllowedExtensions []string `json:"allowed_extensions"`
	VirusScanEnabled  bool     `json:"virus_scan_enabled"`
}

// Storage fallback configuration
type StorageFallback struct {
	PrimaryPath        string        `json:"primary_path"`
	FallbackPaths      []string      `json:"fallback_paths"`
	ReplicationEnabled bool          `json:"replication_enabled"`
	SyncInterval       time.Duration `json:"sync_interval"`
}

var (
	startTime = time.Now()
)

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *sql.DB, redisClient *redis.Client, config HealthConfig) *HealthChecker {
	hc := &HealthChecker{
		db:          db,
		redisClient: redisClient,
		checks:      make(map[string]HealthCheck),
		config:      config,
		lastResults: make(map[string]HealthResult),
		fallbacks:   make(map[string]FallbackHandler),
	}

	if config.EnableMetrics {
		hc.initializeMetrics()
	}

	// Register default health checks
	hc.registerDefaultChecks()
	hc.registerDefaultFallbacks()

	return hc
}

// initializeMetrics initializes Prometheus metrics
func (hc *HealthChecker) initializeMetrics() {
	hc.metrics = &HealthMetrics{
		healthCheckDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "health_check_duration_seconds",
				Help:    "Duration of health checks in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"check_name", "status"},
		),
		healthCheckTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "health_check_total",
				Help: "Total number of health checks performed",
			},
			[]string{"check_name", "status"},
		),
		healthCheckStatus: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "health_check_status",
				Help: "Current status of health checks (1=healthy, 0.5=degraded, 0=unhealthy)",
			},
			[]string{"check_name"},
		),
		systemMetrics: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "system_metrics",
				Help: "System metrics (CPU, memory, disk usage)",
			},
			[]string{"metric_type"},
		),
		uploadMetrics: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "upload_total",
				Help: "Total number of file uploads",
			},
			[]string{"status", "mime_type"},
		),
		fallbackMetrics: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fallback_usage_total",
				Help: "Total number of fallback mechanism usages",
			},
			[]string{"fallback_type", "reason"},
		),
	}
}

// registerDefaultChecks registers default health checks
func (hc *HealthChecker) registerDefaultChecks() {
	// Database health check
	hc.RegisterCheck("database", hc.checkDatabase)

	// Redis health check
	hc.RegisterCheck("redis", hc.checkRedis)

	// System health check
	hc.RegisterCheck("system", hc.checkSystem)

	// Storage health check
	hc.RegisterCheck("storage", hc.checkStorage)

	// Memory health check
	hc.RegisterCheck("memory", hc.checkMemory)

	// Disk space health check
	hc.RegisterCheck("disk", hc.checkDisk)
}

// registerDefaultFallbacks registers default fallback mechanisms
func (hc *HealthChecker) registerDefaultFallbacks() {
	if !hc.config.EnableFallbacks {
		return
	}

	// Database fallback
	hc.RegisterFallback("database", hc.databaseFallback)

	// Redis fallback
	hc.RegisterFallback("redis", hc.redisFallback)

	// Storage fallback
	hc.RegisterFallback("storage", hc.storageFallback)
}

// RegisterCheck registers a new health check
func (hc *HealthChecker) RegisterCheck(name string, check HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[name] = check
}

// RegisterFallback registers a fallback mechanism
func (hc *HealthChecker) RegisterFallback(name string, fallback FallbackHandler) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.fallbacks[name] = fallback
}

// CheckHealth performs all health checks
func (hc *HealthChecker) CheckHealth(ctx context.Context) HealthResponse {
	start := time.Now()

	response := HealthResponse{
		Timestamp:   start,
		Version:     "1.0.0",      // Should come from build info
		Environment: "production", // Should come from config
		Checks:      make(map[string]HealthResult),
		Uptime:      time.Since(startTime),
		System:      hc.getSystemInfo(),
	}

	// Perform all health checks concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex

	hc.mu.RLock()
	checks := make(map[string]HealthCheck)
	for name, check := range hc.checks {
		checks[name] = check
	}
	hc.mu.RUnlock()

	for name, check := range checks {
		wg.Add(1)
		go func(checkName string, checkFunc HealthCheck) {
			defer wg.Done()

			result := hc.performCheck(ctx, checkName, checkFunc)

			mu.Lock()
			response.Checks[checkName] = result
			mu.Unlock()
		}(name, check)
	}

	wg.Wait()

	// Determine overall status
	response.Status = hc.calculateOverallStatus(response.Checks)
	response.Duration = time.Since(start)

	// Add metrics if enabled
	if hc.config.EnableMetrics {
		response.Metrics = hc.getMetrics()
	}

	// Add fallback information
	if hc.config.EnableFallbacks {
		response.Fallbacks = hc.getFallbackInfo()
	}

	return response
}

// performCheck performs a single health check with retry logic
func (hc *HealthChecker) performCheck(ctx context.Context, name string, check HealthCheck) HealthResult {
	start := time.Now()
	var result HealthResult

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, hc.config.Timeout)
	defer cancel()

	// Perform check with retries
	for attempt := 0; attempt <= hc.config.RetryAttempts; attempt++ {
		result = check(checkCtx)
		result.Name = name
		result.Timestamp = time.Now()
		result.Duration = time.Since(start)
		result.RetryCount = attempt

		if result.Status == HealthStatusHealthy {
			result.LastSuccess = result.Timestamp
			break
		}

		// If not the last attempt, wait before retrying
		if attempt < hc.config.RetryAttempts {
			select {
			case <-time.After(hc.config.RetryDelay):
			case <-checkCtx.Done():
				result.Status = HealthStatusUnhealthy
				result.Error = "Check timeout"
				break
			}
		}
	}

	// Update metrics
	if hc.config.EnableMetrics && hc.metrics != nil {
		hc.metrics.healthCheckDuration.WithLabelValues(name, string(result.Status)).Observe(result.Duration.Seconds())
		hc.metrics.healthCheckTotal.WithLabelValues(name, string(result.Status)).Inc()

		statusValue := 0.0
		switch result.Status {
		case HealthStatusHealthy:
			statusValue = 1.0
		case HealthStatusDegraded:
			statusValue = 0.5
		case HealthStatusUnhealthy:
			statusValue = 0.0
		}
		hc.metrics.healthCheckStatus.WithLabelValues(name).Set(statusValue)
	}

	// Store last result
	hc.mu.Lock()
	hc.lastResults[name] = result
	hc.mu.Unlock()

	// Try fallback if check failed
	if result.Status != HealthStatusHealthy && hc.config.EnableFallbacks {
		if fallback, exists := hc.fallbacks[name]; exists {
			if err := fallback(ctx, fmt.Errorf("%s", result.Error)); err == nil {
				result.Status = HealthStatusDegraded
				result.Message = "Using fallback mechanism"

				if hc.metrics != nil {
					hc.metrics.fallbackMetrics.WithLabelValues(name, "health_check_failure").Inc()
				}
			}
		}
	}

	return result
}

// calculateOverallStatus determines the overall system status
func (hc *HealthChecker) calculateOverallStatus(checks map[string]HealthResult) HealthStatus {
	if len(checks) == 0 {
		return HealthStatusUnknown
	}

	healthyCount := 0
	degradedCount := 0
	unhealthyCount := 0

	for _, result := range checks {
		switch result.Status {
		case HealthStatusHealthy:
			healthyCount++
		case HealthStatusDegraded:
			degradedCount++
		case HealthStatusUnhealthy:
			unhealthyCount++
		}
	}

	total := len(checks)
	healthyRatio := float64(healthyCount) / float64(total)

	// If any critical checks are unhealthy, system is unhealthy
	if unhealthyCount > 0 {
		if healthyRatio < hc.config.CriticalThreshold {
			return HealthStatusUnhealthy
		}
	}

	// If some checks are degraded or unhealthy, system is degraded
	if degradedCount > 0 || unhealthyCount > 0 {
		if healthyRatio < hc.config.WarningThreshold {
			return HealthStatusDegraded
		}
	}

	return HealthStatusHealthy
}

// Health check implementations
func (hc *HealthChecker) checkDatabase(ctx context.Context) HealthResult {
	if hc.db == nil {
		return HealthResult{
			Status:  HealthStatusUnhealthy,
			Message: "Database connection not configured",
			Error:   "No database connection",
		}
	}

	start := time.Now()
	err := hc.db.PingContext(ctx)
	duration := time.Since(start)

	if err != nil {
		return HealthResult{
			Status:   HealthStatusUnhealthy,
			Message:  "Database connection failed",
			Error:    err.Error(),
			Duration: duration,
			Metadata: map[string]interface{}{
				"ping_duration_ms": duration.Milliseconds(),
			},
		}
	}

	// Check connection pool stats
	stats := hc.db.Stats()
	metadata := map[string]interface{}{
		"ping_duration_ms":    duration.Milliseconds(),
		"open_connections":    stats.OpenConnections,
		"in_use":              stats.InUse,
		"idle":                stats.Idle,
		"wait_count":          stats.WaitCount,
		"wait_duration_ms":    stats.WaitDuration.Milliseconds(),
		"max_idle_closed":     stats.MaxIdleClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
	}

	status := HealthStatusHealthy
	message := "Database connection healthy"

	// Check for potential issues
	if stats.WaitCount > 0 {
		status = HealthStatusDegraded
		message = "Database connection pool under pressure"
	}

	if duration > 100*time.Millisecond {
		status = HealthStatusDegraded
		message = "Database response time degraded"
	}

	return HealthResult{
		Status:   status,
		Message:  message,
		Duration: duration,
		Metadata: metadata,
	}
}

func (hc *HealthChecker) checkRedis(ctx context.Context) HealthResult {
	if hc.redisClient == nil {
		return HealthResult{
			Status:  HealthStatusUnhealthy,
			Message: "Redis connection not configured",
			Error:   "No Redis connection",
		}
	}

	start := time.Now()
	pong, err := hc.redisClient.Ping(ctx).Result()
	duration := time.Since(start)

	if err != nil {
		return HealthResult{
			Status:   HealthStatusUnhealthy,
			Message:  "Redis connection failed",
			Error:    err.Error(),
			Duration: duration,
		}
	}

	// Get Redis info
	info, _ := hc.redisClient.Info(ctx, "memory", "stats").Result()

	metadata := map[string]interface{}{
		"ping_response":    pong,
		"ping_duration_ms": duration.Milliseconds(),
		"info":             info,
	}

	status := HealthStatusHealthy
	message := "Redis connection healthy"

	if duration > 50*time.Millisecond {
		status = HealthStatusDegraded
		message = "Redis response time degraded"
	}

	return HealthResult{
		Status:   status,
		Message:  message,
		Duration: duration,
		Metadata: metadata,
	}
}

func (hc *HealthChecker) checkSystem(ctx context.Context) HealthResult {
	systemInfo := hc.getSystemInfo()

	status := HealthStatusHealthy
	message := "System resources healthy"

	// Check CPU usage
	if systemInfo.CPUUsage > 90 {
		status = HealthStatusUnhealthy
		message = "High CPU usage detected"
	} else if systemInfo.CPUUsage > 70 {
		status = HealthStatusDegraded
		message = "Elevated CPU usage"
	}

	// Check memory usage
	if systemInfo.MemoryUsage > 95 {
		status = HealthStatusUnhealthy
		message = "Critical memory usage"
	} else if systemInfo.MemoryUsage > 80 {
		if status == HealthStatusHealthy {
			status = HealthStatusDegraded
			message = "High memory usage"
		}
	}

	// Check goroutines
	if systemInfo.Goroutines > 10000 {
		status = HealthStatusDegraded
		message = "High number of goroutines"
	}

	return HealthResult{
		Status:  status,
		Message: message,
		Metadata: map[string]interface{}{
			"cpu_usage":    systemInfo.CPUUsage,
			"memory_usage": systemInfo.MemoryUsage,
			"disk_usage":   systemInfo.DiskUsage,
			"goroutines":   systemInfo.Goroutines,
			"gc_stats":     systemInfo.GCStats,
		},
	}
}

func (hc *HealthChecker) checkStorage(ctx context.Context) HealthResult {
	// Check storage paths availability
	for _, path := range hc.config.StoragePaths {
		if err := hc.checkStoragePath(path); err != nil {
			return HealthResult{
				Status:  HealthStatusUnhealthy,
				Message: fmt.Sprintf("Storage path %s unavailable", path),
				Error:   err.Error(),
			}
		}
	}

	return HealthResult{
		Status:  HealthStatusHealthy,
		Message: "All storage paths available",
		Metadata: map[string]interface{}{
			"storage_paths": hc.config.StoragePaths,
		},
	}
}

func (hc *HealthChecker) checkMemory(ctx context.Context) HealthResult {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Convert bytes to MB
	heapMB := float64(m.HeapInuse) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024

	status := HealthStatusHealthy
	message := "Memory usage normal"

	if heapMB > 500 { // 500MB threshold
		status = HealthStatusDegraded
		message = "High heap memory usage"
	}

	if sysMB > 1000 { // 1GB threshold
		status = HealthStatusUnhealthy
		message = "Critical system memory usage"
	}

	return HealthResult{
		Status:  status,
		Message: message,
		Metadata: map[string]interface{}{
			"heap_mb":     heapMB,
			"sys_mb":      sysMB,
			"gc_cycles":   m.NumGC,
			"last_gc":     time.Unix(0, int64(m.LastGC)),
			"pause_total": time.Duration(m.PauseTotalNs),
		},
	}
}

func (hc *HealthChecker) checkDisk(ctx context.Context) HealthResult {
	// This is a simplified disk check
	// In production, you'd want to check actual disk usage
	diskUsage := 45.0 // Placeholder percentage

	status := HealthStatusHealthy
	message := "Disk usage normal"

	if diskUsage > 90 {
		status = HealthStatusUnhealthy
		message = "Critical disk usage"
	} else if diskUsage > 80 {
		status = HealthStatusDegraded
		message = "High disk usage"
	}

	return HealthResult{
		Status:  status,
		Message: message,
		Metadata: map[string]interface{}{
			"disk_usage_percent": diskUsage,
		},
	}
}

// Fallback implementations
func (hc *HealthChecker) databaseFallback(ctx context.Context, err error) error {
	// Implement database fallback logic (e.g., switch to read-only mode)
	return fmt.Errorf("database fallback not implemented")
}

func (hc *HealthChecker) redisFallback(ctx context.Context, err error) error {
	// Implement Redis fallback logic (e.g., use in-memory cache)
	return fmt.Errorf("redis fallback not implemented")
}

func (hc *HealthChecker) storageFallback(ctx context.Context, err error) error {
	// Implement storage fallback logic (e.g., switch to alternative storage)
	return fmt.Errorf("storage fallback not implemented")
}

// Utility functions
func (hc *HealthChecker) getSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemInfo{
		CPUUsage:    hc.getCPUUsage(),
		MemoryUsage: hc.getMemoryUsage(),
		DiskUsage:   hc.getDiskUsage(),
		Goroutines:  runtime.NumGoroutine(),
		GCStats: GCStats{
			NumGC:      m.NumGC,
			PauseTotal: time.Duration(m.PauseTotalNs),
			LastGC:     time.Unix(0, int64(m.LastGC)),
			HeapSize:   m.HeapSys,
			HeapInUse:  m.HeapInuse,
			StackInUse: m.StackInuse,
		},
	}
}

func (hc *HealthChecker) getCPUUsage() float64 {
	// Simplified CPU usage calculation
	// In production, you'd want to use a proper CPU monitoring library
	return 25.0 // Placeholder percentage
}

func (hc *HealthChecker) getMemoryUsage() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Calculate memory usage percentage
	usedMB := float64(m.HeapInuse) / 1024 / 1024
	totalMB := float64(m.Sys) / 1024 / 1024

	if totalMB == 0 {
		return 0
	}

	return (usedMB / totalMB) * 100
}

func (hc *HealthChecker) getDiskUsage() float64 {
	// Simplified disk usage calculation
	// In production, you'd want to check actual filesystem usage
	return 45.0 // Placeholder percentage
}

func (hc *HealthChecker) checkStoragePath(path string) error {
	// Check if storage path is accessible
	// This is a simplified implementation
	return nil
}

func (hc *HealthChecker) getMetrics() map[string]interface{} {
	return map[string]interface{}{
		"uptime_seconds": time.Since(startTime).Seconds(),
		"goroutines":     runtime.NumGoroutine(),
		"memory_usage":   hc.getMemoryUsage(),
		"cpu_usage":      hc.getCPUUsage(),
	}
}

func (hc *HealthChecker) getFallbackInfo() map[string]FallbackInfo {
	fallbackInfo := make(map[string]FallbackInfo)

	for name := range hc.fallbacks {
		fallbackInfo[name] = FallbackInfo{
			Enabled:     true,
			UsageCount:  0, // Would track actual usage
			Description: fmt.Sprintf("Fallback mechanism for %s", name),
		}
	}

	return fallbackInfo
}

// HTTP handlers
func (hc *HealthChecker) HealthHandler(c echo.Context) error {
	ctx := c.Request().Context()
	health := hc.CheckHealth(ctx)

	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	switch health.Status {
	case HealthStatusDegraded:
		statusCode = http.StatusOK // Still return 200 for degraded
	case HealthStatusUnhealthy:
		statusCode = http.StatusServiceUnavailable
	case HealthStatusUnknown:
		statusCode = http.StatusInternalServerError
	}

	return c.JSON(statusCode, health)
}

func (hc *HealthChecker) ReadinessHandler(c echo.Context) error {
	ctx := c.Request().Context()
	health := hc.CheckHealth(ctx)

	// Readiness check is more strict - only healthy is acceptable
	if health.Status != HealthStatusHealthy {
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status": "not ready",
			"reason": "One or more health checks failed",
			"checks": health.Checks,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "ready",
		"uptime": health.Uptime.Seconds(),
	})
}

func (hc *HealthChecker) LivenessHandler(c echo.Context) error {
	// Liveness check is basic - just check if the service is running
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "alive",
		"uptime":    time.Since(startTime).Seconds(),
		"timestamp": time.Now().Unix(),
	})
}

// Upload validation functions
func (hc *HealthChecker) ValidateUpload(fileSize int64, mimeType string) error {
	// Check file size
	if fileSize > hc.config.MaxUploadSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", fileSize, hc.config.MaxUploadSize)
	}

	// Check MIME type
	allowed := false
	for _, allowedType := range hc.config.AllowedMimeTypes {
		if mimeType == allowedType {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("MIME type %s is not allowed", mimeType)
	}

	return nil
}

// GetUploadLimits returns current upload limits
func (hc *HealthChecker) GetUploadLimits() UploadLimits {
	return UploadLimits{
		MaxFileSize:       hc.config.MaxUploadSize,
		MaxTotalSize:      hc.config.MaxUploadSize * 10, // 10x single file limit
		MaxFiles:          10,
		AllowedMimeTypes:  hc.config.AllowedMimeTypes,
		AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".doc", ".docx"},
		VirusScanEnabled:  false, // Would be configurable
	}
}
