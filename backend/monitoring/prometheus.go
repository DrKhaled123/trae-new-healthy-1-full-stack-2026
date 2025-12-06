package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusMetrics contains all Prometheus metrics
type PrometheusMetrics struct {
	// HTTP metrics
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge
	HTTPResponseSize     *prometheus.HistogramVec

	// Database metrics
	DBConnectionsActive   prometheus.Gauge
	DBConnectionsIdle     prometheus.Gauge
	DBConnectionsMax      prometheus.Gauge
	DBQueryDuration       *prometheus.HistogramVec
	DBTransactionDuration *prometheus.HistogramVec
	DBErrorsTotal         *prometheus.CounterVec

	// Application metrics
	AppUptime      prometheus.Gauge
	AppVersion     *prometheus.GaugeVec
	AppMemoryUsage prometheus.Gauge
	AppGoroutines  prometheus.Gauge
	AppGCDuration  prometheus.Gauge

	// Business metrics
	UsersTotal          prometheus.Gauge
	ActiveUsers         prometheus.Gauge
	APICallsTotal       *prometheus.CounterVec
	NutritionPlansTotal prometheus.Gauge
	ErrorRateTotal      *prometheus.CounterVec

	// Circuit breaker metrics
	CircuitBreakerState    *prometheus.GaugeVec
	CircuitBreakerRequests *prometheus.CounterVec
	CircuitBreakerFailures *prometheus.CounterVec

	// Cache metrics
	CacheHits   *prometheus.CounterVec
	CacheMisses *prometheus.CounterVec
	CacheSize   *prometheus.GaugeVec

	// Security metrics
	SecurityViolations *prometheus.CounterVec
	RateLimitHits      *prometheus.CounterVec
	FailedLogins       *prometheus.CounterVec

	registry  *prometheus.Registry
	startTime time.Time
	mu        sync.RWMutex
}

// NewPrometheusMetrics creates a new Prometheus metrics instance
func NewPrometheusMetrics() *PrometheusMetrics {
	registry := prometheus.NewRegistry()

	m := &PrometheusMetrics{
		registry:  registry,
		startTime: time.Now(),
	}

	// Initialize HTTP metrics
	m.HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	m.HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	m.HTTPRequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
	)

	m.HTTPResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000},
		},
		[]string{"method", "endpoint"},
	)

	// Initialize Database metrics
	m.DBConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections",
		},
	)

	m.DBConnectionsIdle = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	m.DBConnectionsMax = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_max",
			Help: "Maximum number of database connections",
		},
	)

	m.DBQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		},
		[]string{"query_type", "table"},
	)

	m.DBTransactionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_transaction_duration_seconds",
			Help:    "Database transaction duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		},
		[]string{"operation"},
	)

	m.DBErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_errors_total",
			Help: "Total number of database errors",
		},
		[]string{"error_type", "table"},
	)

	// Initialize Application metrics
	m.AppUptime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "app_uptime_seconds",
			Help: "Application uptime in seconds",
		},
	)

	m.AppVersion = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_version_info",
			Help: "Application version information",
		},
		[]string{"version", "commit", "build_date"},
	)

	m.AppMemoryUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "app_memory_usage_bytes",
			Help: "Application memory usage in bytes",
		},
	)

	m.AppGoroutines = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "app_goroutines",
			Help: "Number of goroutines",
		},
	)

	m.AppGCDuration = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "app_gc_duration_seconds",
			Help: "Time spent in garbage collection",
		},
	)

	// Initialize Business metrics
	m.UsersTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "users_total",
			Help: "Total number of registered users",
		},
	)

	m.ActiveUsers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_users",
			Help: "Number of active users in the last 24 hours",
		},
	)

	m.APICallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_calls_total",
			Help: "Total number of API calls",
		},
		[]string{"endpoint", "method", "user_type"},
	)

	m.NutritionPlansTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nutrition_plans_total",
			Help: "Total number of nutrition plans created",
		},
	)

	m.ErrorRateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "error_rate_total",
			Help: "Total error rate by type",
		},
		[]string{"error_type", "severity"},
	)

	// Initialize Circuit breaker metrics
	m.CircuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"service"},
	)

	m.CircuitBreakerRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_requests_total",
			Help: "Total requests through circuit breaker",
		},
		[]string{"service", "state"},
	)

	m.CircuitBreakerFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_failures_total",
			Help: "Total failures in circuit breaker",
		},
		[]string{"service"},
	)

	// Initialize Cache metrics
	m.CacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total cache hits",
		},
		[]string{"cache_type"},
	)

	m.CacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total cache misses",
		},
		[]string{"cache_type"},
	)

	m.CacheSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_size_bytes",
			Help: "Cache size in bytes",
		},
		[]string{"cache_type"},
	)

	// Initialize Security metrics
	m.SecurityViolations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "security_violations_total",
			Help: "Total security violations",
		},
		[]string{"violation_type", "source_ip"},
	)

	m.RateLimitHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_hits_total",
			Help: "Total rate limit hits",
		},
		[]string{"endpoint", "user_id"},
	)

	m.FailedLogins = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "failed_logins_total",
			Help: "Total failed login attempts",
		},
		[]string{"source_ip", "reason"},
	)

	// Register all metrics
	m.registerMetrics()

	// Start background metrics collection
	go m.collectSystemMetrics()

	return m
}

// registerMetrics registers all metrics with Prometheus
func (m *PrometheusMetrics) registerMetrics() {
	// HTTP metrics
	m.registry.MustRegister(m.HTTPRequestsTotal)
	m.registry.MustRegister(m.HTTPRequestDuration)
	m.registry.MustRegister(m.HTTPRequestsInFlight)
	m.registry.MustRegister(m.HTTPResponseSize)

	// Database metrics
	m.registry.MustRegister(m.DBConnectionsActive)
	m.registry.MustRegister(m.DBConnectionsIdle)
	m.registry.MustRegister(m.DBConnectionsMax)
	m.registry.MustRegister(m.DBQueryDuration)
	m.registry.MustRegister(m.DBTransactionDuration)
	m.registry.MustRegister(m.DBErrorsTotal)

	// Application metrics
	m.registry.MustRegister(m.AppUptime)
	m.registry.MustRegister(m.AppVersion)
	m.registry.MustRegister(m.AppMemoryUsage)
	m.registry.MustRegister(m.AppGoroutines)
	m.registry.MustRegister(m.AppGCDuration)

	// Business metrics
	m.registry.MustRegister(m.UsersTotal)
	m.registry.MustRegister(m.ActiveUsers)
	m.registry.MustRegister(m.APICallsTotal)
	m.registry.MustRegister(m.NutritionPlansTotal)
	m.registry.MustRegister(m.ErrorRateTotal)

	// Circuit breaker metrics
	m.registry.MustRegister(m.CircuitBreakerState)
	m.registry.MustRegister(m.CircuitBreakerRequests)
	m.registry.MustRegister(m.CircuitBreakerFailures)

	// Cache metrics
	m.registry.MustRegister(m.CacheHits)
	m.registry.MustRegister(m.CacheMisses)
	m.registry.MustRegister(m.CacheSize)

	// Security metrics
	m.registry.MustRegister(m.SecurityViolations)
	m.registry.MustRegister(m.RateLimitHits)
	m.registry.MustRegister(m.FailedLogins)
}

// collectSystemMetrics collects system metrics in background
func (m *PrometheusMetrics) collectSystemMetrics() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.updateSystemMetrics()
	}
}

// updateSystemMetrics updates system-level metrics
func (m *PrometheusMetrics) updateSystemMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update uptime
	m.AppUptime.Set(time.Since(m.startTime).Seconds())

	// Update memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	m.AppMemoryUsage.Set(float64(memStats.Alloc))

	// Update goroutines
	m.AppGoroutines.Set(float64(runtime.NumGoroutine()))

	// Update GC duration
	m.AppGCDuration.Set(float64(memStats.PauseTotalNs) / 1e9)
}

// GetHandler returns the Prometheus metrics HTTP handler
func (m *PrometheusMetrics) GetHandler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// PrometheusMiddleware creates Echo middleware for Prometheus metrics
func (m *PrometheusMetrics) PrometheusMiddleware() echo.MiddlewareFunc {
	return middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string {
			return fmt.Sprintf("%d", time.Now().UnixNano())
		},
	})
}

// HTTPMiddleware creates HTTP middleware for request metrics
func (m *PrometheusMetrics) HTTPMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			m.HTTPRequestsInFlight.Inc()
			defer m.HTTPRequestsInFlight.Dec()

			err := next(c)

			duration := time.Since(start).Seconds()
			method := c.Request().Method
			path := c.Path()
			status := fmt.Sprintf("%d", c.Response().Status)

			// Record metrics
			m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
			m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
			m.HTTPResponseSize.WithLabelValues(method, path).Observe(float64(c.Response().Size))

			return err
		}
	}
}

// RecordDBMetrics records database operation metrics
func (m *PrometheusMetrics) RecordDBMetrics(operation, table string, duration time.Duration, err error) {
	m.DBQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())

	if err != nil {
		errorType := "unknown"
		if err.Error() != "" {
			errorType = "query_error"
		}
		m.DBErrorsTotal.WithLabelValues(errorType, table).Inc()
	}
}

// RecordCircuitBreakerMetrics records circuit breaker metrics
func (m *PrometheusMetrics) RecordCircuitBreakerMetrics(service string, state int, success bool) {
	m.CircuitBreakerState.WithLabelValues(service).Set(float64(state))

	stateStr := "closed"
	switch state {
	case 1:
		stateStr = "open"
	case 2:
		stateStr = "half-open"
	}

	m.CircuitBreakerRequests.WithLabelValues(service, stateStr).Inc()

	if !success {
		m.CircuitBreakerFailures.WithLabelValues(service).Inc()
	}
}

// RecordCacheMetrics records cache operation metrics
func (m *PrometheusMetrics) RecordCacheMetrics(cacheType string, hit bool, size int64) {
	if hit {
		m.CacheHits.WithLabelValues(cacheType).Inc()
	} else {
		m.CacheMisses.WithLabelValues(cacheType).Inc()
	}

	if size > 0 {
		m.CacheSize.WithLabelValues(cacheType).Set(float64(size))
	}
}

// RecordSecurityMetrics records security-related metrics
func (m *PrometheusMetrics) RecordSecurityMetrics(violationType, sourceIP string) {
	m.SecurityViolations.WithLabelValues(violationType, sourceIP).Inc()
}

// RecordRateLimitHit records rate limit violations
func (m *PrometheusMetrics) RecordRateLimitHit(endpoint, userID string) {
	m.RateLimitHits.WithLabelValues(endpoint, userID).Inc()
}

// RecordFailedLogin records failed login attempts
func (m *PrometheusMetrics) RecordFailedLogin(sourceIP, reason string) {
	m.FailedLogins.WithLabelValues(sourceIP, reason).Inc()
}

// UpdateBusinessMetrics updates business-related metrics
func (m *PrometheusMetrics) UpdateBusinessMetrics(totalUsers, activeUsers, nutritionPlans int64) {
	m.UsersTotal.Set(float64(totalUsers))
	m.ActiveUsers.Set(float64(activeUsers))
	m.NutritionPlansTotal.Set(float64(nutritionPlans))
}

// SetAppVersion sets application version information
func (m *PrometheusMetrics) SetAppVersion(version, commit, buildDate string) {
	m.AppVersion.WithLabelValues(version, commit, buildDate).Set(1)
}

// StartMetricsServer starts the Prometheus metrics server
func (m *PrometheusMetrics) StartMetricsServer(ctx context.Context, port string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", m.GetHandler())

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}
