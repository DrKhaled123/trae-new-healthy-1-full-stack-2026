package optimization

import (
	"database/sql"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PerformanceMonitor tracks application performance metrics
type PerformanceMonitor struct {
	db      *sql.DB
	config  *MonitorConfig
	metrics *PerformanceMetrics
	alerts  *AlertManager
	mu      sync.RWMutex
	stats   *PerformanceStats
}

// MonitorConfig holds configuration for performance monitoring
type MonitorConfig struct {
	CollectionInterval time.Duration
	AlertThresholds    *AlertThresholds
	EnableProfiling    bool
	MaxMemoryMB        int64
	MaxGoroutines      int
	SlowQueryThreshold time.Duration
}

// AlertThresholds defines when to trigger alerts
type AlertThresholds struct {
	HighMemoryMB      int64
	HighGoroutines    int
	HighDBConnections int
	SlowResponseMs    int64
	ErrorRatePercent  float64
}

// PerformanceMetrics holds Prometheus metrics
type PerformanceMetrics struct {
	MemoryUsage     *prometheus.GaugeVec
	GoroutineCount  *prometheus.GaugeVec
	ResponseTime    *prometheus.HistogramVec
	ErrorRate       *prometheus.CounterVec
	DatabaseMetrics *prometheus.GaugeVec
	SystemLoad      *prometheus.GaugeVec
	GCMetrics       *prometheus.GaugeVec
}

// PerformanceStats holds current performance statistics
type PerformanceStats struct {
	MemoryUsageMB   int64
	GoroutineCount  int
	DBConnections   int
	ResponseTimeP95 float64
	ErrorRate       float64
	LastUpdated     time.Time
	Alerts          []Alert
}

// Alert represents a performance alert
type Alert struct {
	Type       string
	Message    string
	Severity   string
	Timestamp  time.Time
	Resolved   bool
	ResolvedAt *time.Time
}

// AlertManager manages performance alerts
type AlertManager struct {
	alerts    []Alert
	mu        sync.RWMutex
	callbacks map[string]func(Alert)
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(db *sql.DB, config *MonitorConfig) *PerformanceMonitor {
	metrics := &PerformanceMetrics{
		MemoryUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "app_memory_usage_bytes",
				Help: "Application memory usage in bytes",
			},
			[]string{"type"},
		),
		GoroutineCount: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "app_goroutines_count",
				Help: "Number of active goroutines",
			},
			[]string{},
		),
		ResponseTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "app_response_time_seconds",
				Help:    "Application response time in seconds",
				Buckets: []float64{0.001, 0.01, 0.1, 1, 5, 10},
			},
			[]string{"endpoint", "method", "status"},
		),
		ErrorRate: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "app_errors_total",
				Help: "Total number of application errors",
			},
			[]string{"type", "component"},
		),
		DatabaseMetrics: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "app_database_stats",
				Help: "Database connection statistics",
			},
			[]string{"stat_type"},
		),
		SystemLoad: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "app_system_load",
				Help: "System load metrics",
			},
			[]string{"load_type"},
		),
		GCMetrics: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "app_gc_stats",
				Help: "Garbage collection statistics",
			},
			[]string{"gc_type"},
		),
	}

	alertManager := &AlertManager{
		alerts:    make([]Alert, 0),
		callbacks: make(map[string]func(Alert)),
	}

	pm := &PerformanceMonitor{
		db:      db,
		config:  config,
		metrics: metrics,
		alerts:  alertManager,
		stats:   &PerformanceStats{},
	}

	// Start monitoring
	go pm.startMonitoring()

	return pm
}

// startMonitoring begins the monitoring loop
func (pm *PerformanceMonitor) startMonitoring() {
	ticker := time.NewTicker(pm.config.CollectionInterval)
	defer ticker.Stop()

	for range ticker.C {
		pm.collectMetrics()
		pm.checkAlerts()
	}
}

// collectMetrics gathers performance metrics
func (pm *PerformanceMonitor) collectMetrics() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Memory metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	pm.stats.MemoryUsageMB = int64(memStats.Alloc / 1024 / 1024)
	pm.metrics.MemoryUsage.WithLabelValues("heap_alloc").Set(float64(memStats.Alloc))
	pm.metrics.MemoryUsage.WithLabelValues("heap_sys").Set(float64(memStats.HeapSys))
	pm.metrics.MemoryUsage.WithLabelValues("heap_idle").Set(float64(memStats.HeapIdle))
	pm.metrics.MemoryUsage.WithLabelValues("heap_inuse").Set(float64(memStats.HeapInuse))

	// Goroutine metrics
	pm.stats.GoroutineCount = runtime.NumGoroutine()
	pm.metrics.GoroutineCount.WithLabelValues().Set(float64(pm.stats.GoroutineCount))

	// Database metrics
	if pm.db != nil {
		dbStats := pm.db.Stats()
		pm.stats.DBConnections = dbStats.OpenConnections

		pm.metrics.DatabaseMetrics.WithLabelValues("open_connections").Set(float64(dbStats.OpenConnections))
		pm.metrics.DatabaseMetrics.WithLabelValues("in_use").Set(float64(dbStats.InUse))
		pm.metrics.DatabaseMetrics.WithLabelValues("idle").Set(float64(dbStats.Idle))
		pm.metrics.DatabaseMetrics.WithLabelValues("wait_count").Set(float64(dbStats.WaitCount))
		pm.metrics.DatabaseMetrics.WithLabelValues("wait_duration_ms").Set(float64(dbStats.WaitDuration.Milliseconds()))
	}

	// GC metrics
	pm.metrics.GCMetrics.WithLabelValues("num_gc").Set(float64(memStats.NumGC))
	pm.metrics.GCMetrics.WithLabelValues("gc_cpu_fraction").Set(memStats.GCCPUFraction)
	pm.metrics.GCMetrics.WithLabelValues("pause_total_ns").Set(float64(memStats.PauseTotalNs))

	// System load (simplified)
	pm.metrics.SystemLoad.WithLabelValues("cpu_percent").Set(pm.getCPUUsage())

	pm.stats.LastUpdated = time.Now()
}

// checkAlerts evaluates alert conditions
func (pm *PerformanceMonitor) checkAlerts() {
	thresholds := pm.config.AlertThresholds

	// Memory usage alert
	if pm.stats.MemoryUsageMB > thresholds.HighMemoryMB {
		pm.alerts.TriggerAlert("high_memory", fmt.Sprintf(
			"High memory usage: %d MB (threshold: %d MB)",
			pm.stats.MemoryUsageMB, thresholds.HighMemoryMB,
		), "warning")
	}

	// Goroutine count alert
	if pm.stats.GoroutineCount > thresholds.HighGoroutines {
		pm.alerts.TriggerAlert("high_goroutines", fmt.Sprintf(
			"High goroutine count: %d (threshold: %d)",
			pm.stats.GoroutineCount, thresholds.HighGoroutines,
		), "warning")
	}

	// Database connection alert
	if pm.stats.DBConnections > thresholds.HighDBConnections {
		pm.alerts.TriggerAlert("high_db_connections", fmt.Sprintf(
			"High database connections: %d (threshold: %d)",
			pm.stats.DBConnections, thresholds.HighDBConnections,
		), "critical")
	}
}

// getCPUUsage returns a simplified CPU usage metric
func (pm *PerformanceMonitor) getCPUUsage() float64 {
	// This is a simplified implementation
	// In production, you'd want to use a proper CPU monitoring library
	return float64(runtime.NumGoroutine()) / 1000.0 // Rough approximation
}

// RecordResponseTime records an HTTP response time
func (pm *PerformanceMonitor) RecordResponseTime(endpoint, method, status string, duration time.Duration) {
	pm.metrics.ResponseTime.WithLabelValues(endpoint, method, status).Observe(duration.Seconds())
}

// RecordError records an application error
func (pm *PerformanceMonitor) RecordError(errorType, component string) {
	pm.metrics.ErrorRate.WithLabelValues(errorType, component).Inc()
}

// GetStats returns current performance statistics
func (pm *PerformanceMonitor) GetStats() *PerformanceStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Create a copy to avoid race conditions
	statsCopy := *pm.stats
	statsCopy.Alerts = make([]Alert, len(pm.alerts.alerts))
	copy(statsCopy.Alerts, pm.alerts.alerts)

	return &statsCopy
}

// GetHealthStatus returns overall health status
func (pm *PerformanceMonitor) GetHealthStatus() map[string]interface{} {
	stats := pm.GetStats()

	status := "healthy"
	issues := make([]string, 0)

	// Check various health indicators
	if stats.MemoryUsageMB > pm.config.AlertThresholds.HighMemoryMB {
		status = "degraded"
		issues = append(issues, "high memory usage")
	}

	if stats.GoroutineCount > pm.config.AlertThresholds.HighGoroutines {
		status = "degraded"
		issues = append(issues, "high goroutine count")
	}

	if stats.DBConnections > pm.config.AlertThresholds.HighDBConnections {
		status = "critical"
		issues = append(issues, "high database connections")
	}

	// Count active alerts
	activeAlerts := 0
	for _, alert := range stats.Alerts {
		if !alert.Resolved {
			activeAlerts++
		}
	}

	return map[string]interface{}{
		"status":         status,
		"memory_mb":      stats.MemoryUsageMB,
		"goroutines":     stats.GoroutineCount,
		"db_connections": stats.DBConnections,
		"active_alerts":  activeAlerts,
		"issues":         issues,
		"last_updated":   stats.LastUpdated,
	}
}

// AlertManager methods

// TriggerAlert creates a new alert
func (am *AlertManager) TriggerAlert(alertType, message, severity string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Check if alert already exists and is active
	for i, alert := range am.alerts {
		if alert.Type == alertType && !alert.Resolved {
			// Update existing alert
			am.alerts[i].Message = message
			am.alerts[i].Timestamp = time.Now()
			return
		}
	}

	// Create new alert
	alert := Alert{
		Type:      alertType,
		Message:   message,
		Severity:  severity,
		Timestamp: time.Now(),
		Resolved:  false,
	}

	am.alerts = append(am.alerts, alert)

	// Call callbacks
	for _, callback := range am.callbacks {
		go callback(alert)
	}

	log.Printf("Alert triggered: %s - %s", alertType, message)
}

// ResolveAlert marks an alert as resolved
func (am *AlertManager) ResolveAlert(alertType string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	for i, alert := range am.alerts {
		if alert.Type == alertType && !alert.Resolved {
			now := time.Now()
			am.alerts[i].Resolved = true
			am.alerts[i].ResolvedAt = &now
			log.Printf("Alert resolved: %s", alertType)
			break
		}
	}
}

// RegisterCallback registers a callback for alerts
func (am *AlertManager) RegisterCallback(name string, callback func(Alert)) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.callbacks[name] = callback
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var active []Alert
	for _, alert := range am.alerts {
		if !alert.Resolved {
			active = append(active, alert)
		}
	}

	return active
}

// ProfileMemory creates a memory profile for debugging
func (pm *PerformanceMonitor) ProfileMemory() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"alloc_mb":        memStats.Alloc / 1024 / 1024,
		"total_alloc_mb":  memStats.TotalAlloc / 1024 / 1024,
		"sys_mb":          memStats.Sys / 1024 / 1024,
		"num_gc":          memStats.NumGC,
		"gc_cpu_fraction": memStats.GCCPUFraction,
		"heap_objects":    memStats.HeapObjects,
		"stack_inuse_mb":  memStats.StackInuse / 1024 / 1024,
		"goroutines":      runtime.NumGoroutine(),
	}
}

// OptimizationSuggestions provides performance optimization suggestions
func (pm *PerformanceMonitor) OptimizationSuggestions() []string {
	stats := pm.GetStats()
	suggestions := make([]string, 0)

	if stats.MemoryUsageMB > 100 {
		suggestions = append(suggestions, "Consider implementing memory pooling or reducing object allocations")
	}

	if stats.GoroutineCount > 1000 {
		suggestions = append(suggestions, "High goroutine count detected - review goroutine lifecycle management")
	}

	if stats.DBConnections > 50 {
		suggestions = append(suggestions, "High database connection usage - consider connection pooling optimization")
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Performance metrics look healthy")
	}

	return suggestions
}
