package monitoring

import (
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsCollector holds all application metrics
type MetricsCollector struct {
	// HTTP metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec

	// Database metrics
	dbConnectionsActive prometheus.Gauge
	dbConnectionsIdle   prometheus.Gauge
	dbQueryDuration     *prometheus.HistogramVec
	dbQueriesTotal      *prometheus.CounterVec
	dbErrorsTotal       *prometheus.CounterVec

	// Cache metrics
	cacheHits   prometheus.Counter
	cacheMisses prometheus.Counter
	cacheSize   prometheus.Gauge

	// Application metrics
	activeUsers      prometheus.Gauge
	totalUsers       prometheus.Counter
	errorRate        prometheus.Gauge
	uptime           prometheus.Gauge
	requestQueueSize prometheus.Gauge

	// System metrics
	cpuUsage    prometheus.Gauge
	memoryUsage prometheus.Gauge
	goroutines  prometheus.Gauge

	// Business metrics
	nutritionPlansGenerated prometheus.Counter
	workoutsLogged          prometheus.Counter
	mealsLogged             prometheus.Counter
	usersRegistered         prometheus.Counter

	mu        sync.RWMutex
	startTime time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime: time.Now(),
	}
}

// InitializeMetrics initializes all Prometheus metrics
func (m *MetricsCollector) InitializeMetrics() {
	// HTTP metrics
	m.httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	m.httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	m.httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "Size of HTTP responses in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000},
		},
		[]string{"method", "endpoint"},
	)

	// Database metrics
	m.dbConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections",
		},
	)

	m.dbConnectionsIdle = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	m.dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Duration of database queries in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	m.dbQueriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "table", "status"},
	)

	m.dbErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_errors_total",
			Help: "Total number of database errors",
		},
		[]string{"operation", "error_type"},
	)

	// Cache metrics
	m.cacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	m.cacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	m.cacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cache_size_bytes",
			Help: "Current cache size in bytes",
		},
	)

	// Application metrics
	m.activeUsers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_users",
			Help: "Number of currently active users",
		},
	)

	m.totalUsers = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "total_users",
			Help: "Total number of registered users",
		},
	)

	m.errorRate = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "error_rate",
			Help: "Current error rate (errors per second)",
		},
	)

	m.uptime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "uptime_seconds",
			Help: "Application uptime in seconds",
		},
	)

	m.requestQueueSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "request_queue_size",
			Help: "Number of requests currently in queue",
		},
	)

	// System metrics
	m.cpuUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_usage_percent",
			Help: "Current CPU usage percentage",
		},
	)

	m.memoryUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "memory_usage_bytes",
			Help: "Current memory usage in bytes",
		},
	)

	m.goroutines = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "goroutines",
			Help: "Number of goroutines",
		},
	)

	// Business metrics
	m.nutritionPlansGenerated = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nutrition_plans_generated_total",
			Help: "Total number of nutrition plans generated",
		},
	)

	m.workoutsLogged = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "workouts_logged_total",
			Help: "Total number of workouts logged",
		},
	)

	m.mealsLogged = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "meals_logged_total",
			Help: "Total number of meals logged",
		},
	)

	m.usersRegistered = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "users_registered_total",
			Help: "Total number of users registered",
		},
	)

	// Register all metrics with Prometheus
	prometheus.MustRegister(
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.httpResponseSize,
		m.dbConnectionsActive,
		m.dbConnectionsIdle,
		m.dbQueryDuration,
		m.dbQueriesTotal,
		m.dbErrorsTotal,
		m.cacheHits,
		m.cacheMisses,
		m.cacheSize,
		m.activeUsers,
		m.totalUsers,
		m.errorRate,
		m.uptime,
		m.requestQueueSize,
		m.cpuUsage,
		m.memoryUsage,
		m.goroutines,
		m.nutritionPlansGenerated,
		m.workoutsLogged,
		m.mealsLogged,
		m.usersRegistered,
	)

	// Start system metrics collection
	go m.collectSystemMetrics()
}

// RecordHTTPRequest records HTTP request metrics
func (m *MetricsCollector) RecordHTTPRequest(method, endpoint, statusCode string, duration time.Duration, responseSize int64) {
	m.httpRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
	m.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	m.httpResponseSize.WithLabelValues(method, endpoint).Observe(float64(responseSize))
}

// RecordDatabaseQuery records database query metrics
func (m *MetricsCollector) RecordDatabaseQuery(operation, table, status string, duration time.Duration) {
	m.dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
	m.dbQueriesTotal.WithLabelValues(operation, table, status).Inc()
}

// RecordDatabaseError records database error metrics
func (m *MetricsCollector) RecordDatabaseError(operation, errorType string) {
	m.dbErrorsTotal.WithLabelValues(operation, errorType).Inc()
}

// RecordCacheHit records a cache hit
func (m *MetricsCollector) RecordCacheHit() {
	m.cacheHits.Inc()
}

// RecordCacheMiss records a cache miss
func (m *MetricsCollector) RecordCacheMiss() {
	m.cacheMisses.Inc()
}

// UpdateDatabaseConnections updates database connection metrics
func (m *MetricsCollector) UpdateDatabaseConnections(active, idle int) {
	m.dbConnectionsActive.Set(float64(active))
	m.dbConnectionsIdle.Set(float64(idle))
}

// UpdateActiveUsers updates the active users count
func (m *MetricsCollector) UpdateActiveUsers(count int) {
	m.activeUsers.Set(float64(count))
}

// IncrementTotalUsers increments the total users count
func (m *MetricsCollector) IncrementTotalUsers() {
	m.totalUsers.Inc()
	m.usersRegistered.Inc()
}

// IncrementNutritionPlans increments nutrition plans generated
func (m *MetricsCollector) IncrementNutritionPlans() {
	m.nutritionPlansGenerated.Inc()
}

// IncrementWorkoutsLogged increments workouts logged
func (m *MetricsCollector) IncrementWorkoutsLogged() {
	m.workoutsLogged.Inc()
}

// IncrementMealsLogged increments meals logged
func (m *MetricsCollector) IncrementMealsLogged() {
	m.mealsLogged.Inc()
}

// UpdateRequestQueueSize updates the request queue size
func (m *MetricsCollector) UpdateRequestQueueSize(size int) {
	m.requestQueueSize.Set(float64(size))
}

// collectSystemMetrics collects system metrics periodically
func (m *MetricsCollector) collectSystemMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Update uptime
		m.uptime.Set(time.Since(m.startTime).Seconds())

		// Update goroutines count
		m.goroutines.Set(float64(runtime.NumGoroutine()))

		// Update memory usage
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		m.memoryUsage.Set(float64(m.Alloc))

		// Note: CPU usage would require additional implementation
		// This is a placeholder that would need platform-specific implementation
	}
}

// GetMetricsHandler returns the Prometheus metrics HTTP handler
func (m *MetricsCollector) GetMetricsHandler() http.Handler {
	return promhttp.Handler()
}

// MetricsMiddleware returns Echo middleware for collecting HTTP metrics
func (m *MetricsCollector) MetricsMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer to capture status code and response size
			rw := &responseWriter{ResponseWriter: w, statusCode: 200}

			next.ServeHTTP(rw, r)

			// Record metrics
			duration := time.Since(start)
			m.RecordHTTPRequest(
				r.Method,
				r.URL.Path,
				string(rune(rw.statusCode+'0')),
				duration,
				int64(rw.size),
			)
		})
	}
}

// responseWriter is a wrapper around http.ResponseWriter to capture status code and size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}
