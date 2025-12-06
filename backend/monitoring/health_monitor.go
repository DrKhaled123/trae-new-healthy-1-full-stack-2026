package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RedisClient interface to avoid direct dependency
type RedisClient interface {
	Ping(ctx context.Context) error
	PoolStats() interface{}
}

// HealthMonitor manages system health checks and monitoring
type HealthMonitor struct {
	config       *MonitorConfig
	metrics      *SystemMetrics
	db           *sql.DB
	redis        RedisClient
	alerts       *AlertManager
	healthChecks map[string]HealthCheck
	mu           sync.RWMutex
	ticker       *time.Ticker
	ctx          context.Context
	cancel       context.CancelFunc
}

// MonitorConfig holds monitoring configuration
type MonitorConfig struct {
	CheckInterval      time.Duration
	HealthCheckTimeout time.Duration
	AlertThresholds    *AlertThresholds
	EnableMetrics      bool
	EnableAlerts       bool
	EnableDashboard    bool
	MetricsPort        int
	HealthEndpoint     string
	LogHealthChecks    bool
	RetryAttempts      int
	RetryDelay         time.Duration
}

// AlertThresholds defines thresholds for various alerts
type AlertThresholds struct {
	CPUUsage            float64
	MemoryUsage         float64
	DiskUsage           float64
	ResponseTime        time.Duration
	ErrorRate           float64
	DatabaseConnections int
	RedisConnections    int
	ActiveGoroutines    int
}

// SystemMetrics holds Prometheus metrics for system monitoring
type SystemMetrics struct {
	HealthStatus        *prometheus.GaugeVec
	ResponseTime        *prometheus.HistogramVec
	SystemCPU           prometheus.Gauge
	SystemMemory        prometheus.Gauge
	SystemDisk          prometheus.Gauge
	DatabaseConnections prometheus.Gauge
	RedisConnections    prometheus.Gauge
	ActiveGoroutines    prometheus.Gauge
	ErrorRate           *prometheus.CounterVec
	Uptime              prometheus.Counter
	HealthCheckDuration *prometheus.HistogramVec
	AlertsFired         *prometheus.CounterVec
}

// HealthCheck represents a health check function
type HealthCheck struct {
	Name                string
	Description         string
	Check               func(ctx context.Context) error
	Timeout             time.Duration
	Critical            bool
	LastCheck           time.Time
	LastResult          error
	ConsecutiveFailures int
}

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status     string                 `json:"status"`
	Timestamp  time.Time              `json:"timestamp"`
	Uptime     time.Duration          `json:"uptime"`
	Version    string                 `json:"version"`
	Checks     map[string]CheckResult `json:"checks"`
	SystemInfo SystemInfo             `json:"system_info"`
}

// CheckResult represents the result of a health check
type CheckResult struct {
	Status              string        `json:"status"`
	Message             string        `json:"message,omitempty"`
	Duration            time.Duration `json:"duration"`
	LastCheck           time.Time     `json:"last_check"`
	ConsecutiveFailures int           `json:"consecutive_failures"`
}

// SystemInfo holds system information
type SystemInfo struct {
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryUsage   float64 `json:"memory_usage"`
	Goroutines    int     `json:"goroutines"`
	DatabaseConns int     `json:"database_connections"`
	RedisConns    int     `json:"redis_connections"`
}

// AlertManager handles alert notifications
type AlertManager struct {
	config       *AlertConfig
	notifiers    []Notifier
	alertHistory map[string][]Alert
	mu           sync.RWMutex
}

// AlertConfig holds alert configuration
type AlertConfig struct {
	Enabled         bool
	WebhookURL      string
	EmailRecipients []string
	SlackChannel    string
	CooldownPeriod  time.Duration
}

// Alert represents an alert
type Alert struct {
	ID         string                 `json:"id"`
	Severity   string                 `json:"severity"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
}

// Notifier interface for alert notifications
type Notifier interface {
	Send(alert Alert) error
}

var startTime = time.Now()

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(config *MonitorConfig, db *sql.DB, redisClient RedisClient) *HealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	hm := &HealthMonitor{
		config:       config,
		metrics:      newSystemMetrics(),
		db:           db,
		redis:        redisClient,
		alerts:       newAlertManager(config),
		healthChecks: make(map[string]HealthCheck),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Register default health checks
	hm.registerDefaultHealthChecks()

	// Start monitoring
	hm.start()

	return hm
}

// newSystemMetrics creates Prometheus metrics for system monitoring
func newSystemMetrics() *SystemMetrics {
	m := &SystemMetrics{
		HealthStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nutrition_platform_health_status",
				Help: "Health status of various components (1=healthy, 0=unhealthy)",
			},
			[]string{"component"},
		),
		ResponseTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nutrition_platform_response_time_seconds",
				Help:    "Response time of health checks",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"endpoint"},
		),
		SystemCPU: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nutrition_platform_cpu_usage_percent",
				Help: "Current CPU usage percentage",
			},
		),
		SystemMemory: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nutrition_platform_memory_usage_bytes",
				Help: "Current memory usage in bytes",
			},
		),
		SystemDisk: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nutrition_platform_disk_usage_percent",
				Help: "Current disk usage percentage",
			},
		),
		DatabaseConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nutrition_platform_database_connections",
				Help: "Number of active database connections",
			},
		),
		RedisConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nutrition_platform_redis_connections",
				Help: "Number of active Redis connections",
			},
		),
		ActiveGoroutines: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nutrition_platform_goroutines",
				Help: "Number of active goroutines",
			},
		),
		ErrorRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nutrition_platform_errors_total",
				Help: "Total number of errors by type",
			},
			[]string{"type"},
		),
		Uptime: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "nutrition_platform_uptime_seconds",
				Help: "Application uptime in seconds",
			},
		),
		HealthCheckDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nutrition_platform_health_check_duration_seconds",
				Help:    "Duration of health checks",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"check_name"},
		),
		AlertsFired: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nutrition_platform_alerts_fired_total",
				Help: "Total number of alerts fired",
			},
			[]string{"severity", "type"},
		),
	}

	// Register metrics
	prometheus.MustRegister(m.HealthStatus)
	prometheus.MustRegister(m.ResponseTime)
	prometheus.MustRegister(m.SystemCPU)
	prometheus.MustRegister(m.SystemMemory)
	prometheus.MustRegister(m.SystemDisk)
	prometheus.MustRegister(m.DatabaseConnections)
	prometheus.MustRegister(m.RedisConnections)
	prometheus.MustRegister(m.ActiveGoroutines)
	prometheus.MustRegister(m.ErrorRate)
	prometheus.MustRegister(m.Uptime)
	prometheus.MustRegister(m.HealthCheckDuration)
	prometheus.MustRegister(m.AlertsFired)

	return m
}

// newAlertManager creates a new alert manager
func newAlertManager(config *MonitorConfig) *AlertManager {
	return &AlertManager{
		config: &AlertConfig{
			Enabled:        config.EnableAlerts,
			CooldownPeriod: 5 * time.Minute,
		},
		notifiers:    make([]Notifier, 0),
		alertHistory: make(map[string][]Alert),
	}
}

// RegisterHealthCheck registers a new health check
func (hm *HealthMonitor) RegisterHealthCheck(check HealthCheck) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.healthChecks[check.Name] = check
}

// registerDefaultHealthChecks registers default health checks
func (hm *HealthMonitor) registerDefaultHealthChecks() {
	// Database health check
	hm.RegisterHealthCheck(HealthCheck{
		Name:        "database",
		Description: "PostgreSQL database connectivity",
		Check:       hm.checkDatabase,
		Timeout:     5 * time.Second,
		Critical:    true,
	})

	// Redis health check
	hm.RegisterHealthCheck(HealthCheck{
		Name:        "redis",
		Description: "Redis cache connectivity",
		Check:       hm.checkRedis,
		Timeout:     3 * time.Second,
		Critical:    true,
	})

	// System resources health check
	hm.RegisterHealthCheck(HealthCheck{
		Name:        "system",
		Description: "System resource usage",
		Check:       hm.checkSystemResources,
		Timeout:     2 * time.Second,
		Critical:    false,
	})

	// Memory health check
	hm.RegisterHealthCheck(HealthCheck{
		Name:        "memory",
		Description: "Memory usage and garbage collection",
		Check:       hm.checkMemory,
		Timeout:     1 * time.Second,
		Critical:    false,
	})
}

// start begins the monitoring process
func (hm *HealthMonitor) start() {
	hm.ticker = time.NewTicker(hm.config.CheckInterval)

	go func() {
		for {
			select {
			case <-hm.ticker.C:
				hm.runHealthChecks()
			case <-hm.ctx.Done():
				return
			}
		}
	}()

	// Start metrics server if enabled
	if hm.config.EnableMetrics {
		go hm.startMetricsServer()
	}
}

// runHealthChecks executes all registered health checks
func (hm *HealthMonitor) runHealthChecks() {
	hm.mu.RLock()
	checks := make(map[string]HealthCheck)
	for name, check := range hm.healthChecks {
		checks[name] = check
	}
	hm.mu.RUnlock()

	for name, check := range checks {
		go hm.executeHealthCheck(name, check)
	}

	// Update system metrics
	hm.updateSystemMetrics()
}

// executeHealthCheck executes a single health check
func (hm *HealthMonitor) executeHealthCheck(name string, check HealthCheck) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(hm.ctx, check.Timeout)
	defer cancel()

	err := check.Check(ctx)
	duration := time.Since(start)

	hm.metrics.HealthCheckDuration.WithLabelValues(name).Observe(duration.Seconds())

	// Update check result
	hm.mu.Lock()
	check.LastCheck = time.Now()
	check.LastResult = err
	if err != nil {
		check.ConsecutiveFailures++
		hm.metrics.HealthStatus.WithLabelValues(name).Set(0)
		hm.metrics.ErrorRate.WithLabelValues("health_check").Inc()
	} else {
		check.ConsecutiveFailures = 0
		hm.metrics.HealthStatus.WithLabelValues(name).Set(1)
	}
	hm.healthChecks[name] = check
	hm.mu.Unlock()

	// Log if enabled
	if hm.config.LogHealthChecks {
		if err != nil {
			log.Printf("Health check '%s' failed: %v (duration: %v)", name, err, duration)
		} else {
			log.Printf("Health check '%s' passed (duration: %v)", name, duration)
		}
	}

	// Check for alerts
	if err != nil && check.Critical {
		hm.handleHealthCheckFailure(name, check, err)
	}
}

// Health check implementations

func (hm *HealthMonitor) checkDatabase(ctx context.Context) error {
	if hm.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := hm.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Check connection pool stats
	stats := hm.db.Stats()
	hm.metrics.DatabaseConnections.Set(float64(stats.OpenConnections))

	if stats.OpenConnections > hm.config.AlertThresholds.DatabaseConnections {
		return fmt.Errorf("too many database connections: %d", stats.OpenConnections)
	}

	return nil
}

func (hm *HealthMonitor) checkRedis(ctx context.Context) error {
	if hm.redis == nil {
		return fmt.Errorf("redis client is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := hm.redis.Ping(ctx); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	// Check Redis pool stats (simplified since we don't have direct access)
	// In a real implementation, you would cast to the actual Redis client type
	hm.metrics.RedisConnections.Set(1) // Simplified - just indicate connection is active

	return nil
}

func (hm *HealthMonitor) checkSystemResources(ctx context.Context) error {
	// This is a simplified implementation
	// In production, you'd use a proper system monitoring library
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check memory usage (simplified)
	memoryUsage := float64(m.Alloc) / float64(m.Sys) * 100
	if memoryUsage > hm.config.AlertThresholds.MemoryUsage {
		return fmt.Errorf("high memory usage: %.2f%%", memoryUsage)
	}

	// Check goroutine count
	goroutines := runtime.NumGoroutine()
	hm.metrics.ActiveGoroutines.Set(float64(goroutines))
	if goroutines > hm.config.AlertThresholds.ActiveGoroutines {
		return fmt.Errorf("too many goroutines: %d", goroutines)
	}

	return nil
}

func (hm *HealthMonitor) checkMemory(ctx context.Context) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	hm.metrics.SystemMemory.Set(float64(m.Alloc))

	// Check for memory leaks (simplified)
	if m.NumGC > 0 && m.PauseTotalNs > uint64(100*time.Millisecond.Nanoseconds()) {
		return fmt.Errorf("high GC pause time: %v", time.Duration(m.PauseTotalNs))
	}

	return nil
}

// updateSystemMetrics updates system-wide metrics
func (hm *HealthMonitor) updateSystemMetrics() {
	// Update uptime
	hm.metrics.Uptime.Add(hm.config.CheckInterval.Seconds())

	// Update goroutine count
	hm.metrics.ActiveGoroutines.Set(float64(runtime.NumGoroutine()))

	// Update memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	hm.metrics.SystemMemory.Set(float64(m.Alloc))
}

// GetHealthStatus returns the current health status
func (hm *HealthMonitor) GetHealthStatus() HealthStatus {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	checks := make(map[string]CheckResult)
	overallStatus := "healthy"

	for name, check := range hm.healthChecks {
		status := "healthy"
		message := ""
		if check.LastResult != nil {
			status = "unhealthy"
			message = check.LastResult.Error()
			if check.Critical {
				overallStatus = "unhealthy"
			}
		}

		checks[name] = CheckResult{
			Status:              status,
			Message:             message,
			LastCheck:           check.LastCheck,
			ConsecutiveFailures: check.ConsecutiveFailures,
		}
	}

	// Get system info
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	systemInfo := SystemInfo{
		MemoryUsage: float64(m.Alloc),
		Goroutines:  runtime.NumGoroutine(),
	}

	if hm.db != nil {
		stats := hm.db.Stats()
		systemInfo.DatabaseConns = stats.OpenConnections
	}

	if hm.redis != nil {
		// Simplified Redis connection count
		systemInfo.RedisConns = 1
	}

	return HealthStatus{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Uptime:     time.Since(startTime),
		Version:    "1.0.0", // This should come from build info
		Checks:     checks,
		SystemInfo: systemInfo,
	}
}

// handleHealthCheckFailure handles a health check failure
func (hm *HealthMonitor) handleHealthCheckFailure(name string, check HealthCheck, err error) {
	if !hm.config.EnableAlerts {
		return
	}

	// Create alert
	alert := Alert{
		ID:        fmt.Sprintf("%s-%d", name, time.Now().Unix()),
		Severity:  "critical",
		Title:     fmt.Sprintf("Health Check Failed: %s", name),
		Message:   fmt.Sprintf("Health check '%s' failed: %v", name, err),
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"check_name":           name,
			"consecutive_failures": check.ConsecutiveFailures,
			"error":                err.Error(),
		},
	}

	hm.alerts.FireAlert(alert)
	hm.metrics.AlertsFired.WithLabelValues(alert.Severity, "health_check").Inc()
}

// startMetricsServer starts the Prometheus metrics server
func (hm *HealthMonitor) startMetricsServer() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", hm.healthHandler)
	mux.HandleFunc("/health/ready", hm.readinessHandler)
	mux.HandleFunc("/health/live", hm.livenessHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", hm.config.MetricsPort),
		Handler: mux,
	}

	log.Printf("Starting metrics server on port %d", hm.config.MetricsPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("Metrics server error: %v", err)
	}
}

// HTTP handlers

func (hm *HealthMonitor) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := hm.GetHealthStatus()
	w.Header().Set("Content-Type", "application/json")

	if status.Status != "healthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(status)
}

func (hm *HealthMonitor) readinessHandler(w http.ResponseWriter, r *http.Request) {
	status := hm.GetHealthStatus()
	w.Header().Set("Content-Type", "application/json")

	// Check critical components
	ready := true
	for name, check := range status.Checks {
		if hm.healthChecks[name].Critical && check.Status != "healthy" {
			ready = false
			break
		}
	}

	if !ready {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	response := map[string]interface{}{
		"ready":     ready,
		"timestamp": time.Now(),
		"checks":    status.Checks,
	}

	json.NewEncoder(w).Encode(response)
}

func (hm *HealthMonitor) livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"alive":     true,
		"timestamp": time.Now(),
		"uptime":    time.Since(startTime).String(),
	}

	json.NewEncoder(w).Encode(response)
}

// Alert management methods

func (am *AlertManager) FireAlert(alert Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Check cooldown period
	if am.isInCooldown(alert) {
		return
	}

	// Store alert
	am.alertHistory[alert.ID] = append(am.alertHistory[alert.ID], alert)

	// Send notifications
	for _, notifier := range am.notifiers {
		go func(n Notifier) {
			if err := n.Send(alert); err != nil {
				log.Printf("Failed to send alert notification: %v", err)
			}
		}(notifier)
	}

	log.Printf("Alert fired: %s - %s", alert.Title, alert.Message)
}

func (am *AlertManager) isInCooldown(alert Alert) bool {
	// Simple cooldown implementation
	// In production, you'd want more sophisticated logic
	return false
}

func (am *AlertManager) AddNotifier(notifier Notifier) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.notifiers = append(am.notifiers, notifier)
}

// Stop stops the health monitor
func (hm *HealthMonitor) Stop() {
	if hm.ticker != nil {
		hm.ticker.Stop()
	}
	if hm.cancel != nil {
		hm.cancel()
	}
}

// SetupHealthRoutes sets up health check routes for Echo
func (hm *HealthMonitor) SetupHealthRoutes(e *echo.Echo) {
	e.GET(hm.config.HealthEndpoint, func(c echo.Context) error {
		status := hm.GetHealthStatus()
		if status.Status != "healthy" {
			return c.JSON(http.StatusServiceUnavailable, status)
		}
		return c.JSON(http.StatusOK, status)
	})

	e.GET(hm.config.HealthEndpoint+"/ready", func(c echo.Context) error {
		status := hm.GetHealthStatus()

		// Check critical components
		ready := true
		for name, check := range status.Checks {
			if hm.healthChecks[name].Critical && check.Status != "healthy" {
				ready = false
				break
			}
		}

		response := map[string]interface{}{
			"ready":     ready,
			"timestamp": time.Now(),
			"checks":    status.Checks,
		}

		if !ready {
			return c.JSON(http.StatusServiceUnavailable, response)
		}
		return c.JSON(http.StatusOK, response)
	})

	e.GET(hm.config.HealthEndpoint+"/live", func(c echo.Context) error {
		response := map[string]interface{}{
			"alive":     true,
			"timestamp": time.Now(),
			"uptime":    time.Since(startTime).String(),
		}
		return c.JSON(http.StatusOK, response)
	})
}
