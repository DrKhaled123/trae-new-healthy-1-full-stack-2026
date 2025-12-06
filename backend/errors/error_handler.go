package errors

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sony/gobreaker"
)

// ErrorHandler provides comprehensive error handling capabilities
type ErrorHandler struct {
	config          *ErrorHandlerConfig
	circuitBreakers map[string]*gobreaker.CircuitBreaker
	metrics         *ErrorMetrics
	mu              sync.RWMutex
	notificationCh  chan *ErrorNotification
}

// ErrorHandlerConfig holds configuration for the error handler
type ErrorHandlerConfig struct {
	DebugMode             bool
	LogStackTrace         bool
	NotifyOnPanic         bool
	SanitizeErrors        bool
	MaxRetries            int
	RetryDelay            time.Duration
	CircuitBreakerEnabled bool
	HealthCheckInterval   time.Duration
	AlertThreshold        int
	NotificationEnabled   bool
}

// ErrorMetrics holds Prometheus metrics for error tracking
type ErrorMetrics struct {
	ErrorsTotal         *prometheus.CounterVec
	ErrorsByType        *prometheus.CounterVec
	RecoveryTime        *prometheus.HistogramVec
	CircuitBreakerState *prometheus.GaugeVec
	RetryAttempts       *prometheus.CounterVec
}

// ErrorNotification represents an error notification
type ErrorNotification struct {
	Error     *APIError
	Severity  string
	Timestamp time.Time
	Context   map[string]interface{}
}

// NewErrorHandler creates a new error handler instance
func NewErrorHandler(config *ErrorHandlerConfig) *ErrorHandler {
	eh := &ErrorHandler{
		config:          config,
		circuitBreakers: make(map[string]*gobreaker.CircuitBreaker),
		metrics:         newErrorMetrics(),
		notificationCh:  make(chan *ErrorNotification, 100),
	}

	// Start notification processor
	if config.NotificationEnabled {
		go eh.processNotifications()
	}

	// Start health check monitor
	if config.HealthCheckInterval > 0 {
		go eh.healthCheckMonitor()
	}

	return eh
}

// newErrorMetrics creates Prometheus metrics for error tracking
func newErrorMetrics() *ErrorMetrics {
	m := &ErrorMetrics{
		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nutrition_platform_errors_total",
				Help: "Total number of errors by type and severity",
			},
			[]string{"type", "severity", "component"},
		),
		ErrorsByType: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nutrition_platform_errors_by_type",
				Help: "Errors grouped by error code",
			},
			[]string{"error_code", "http_status"},
		),
		RecoveryTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nutrition_platform_error_recovery_seconds",
				Help:    "Time taken to recover from errors",
				Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
			[]string{"component", "error_type"},
		),
		CircuitBreakerState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nutrition_platform_circuit_breaker_state",
				Help: "Circuit breaker state (0=closed, 1=half-open, 2=open)",
			},
			[]string{"service"},
		),
		RetryAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nutrition_platform_retry_attempts_total",
				Help: "Total number of retry attempts",
			},
			[]string{"operation", "success"},
		),
	}

	// Register metrics with error handling for tests
	registerMetricSafely(m.ErrorsTotal)
	registerMetricSafely(m.ErrorsByType)
	registerMetricSafely(m.RecoveryTime)
	registerMetricSafely(m.CircuitBreakerState)
	registerMetricSafely(m.RetryAttempts)

	return m
}

// registerMetricSafely registers a metric, ignoring duplicate registration errors
func registerMetricSafely(collector prometheus.Collector) {
	defer func() {
		if r := recover(); r != nil {
			// Ignore duplicate registration panics in tests
			if err, ok := r.(error); ok && strings.Contains(err.Error(), "duplicate metrics collector registration attempted") {
				return
			}
			panic(r)
		}
	}()
	prometheus.MustRegister(collector)
}

// HandleError processes and handles different types of errors
func (eh *ErrorHandler) HandleError(err error, c echo.Context) {
	start := time.Now()
	defer func() {
		if r := recover(); r != nil {
			eh.handlePanic(r, c)
		}
	}()

	// Don't handle if response already sent
	if c.Response().Committed {
		return
	}

	var apiErr *APIError

	// Convert error to APIError
	switch e := err.(type) {
	case *APIError:
		apiErr = e
	case *echo.HTTPError:
		apiErr = eh.convertHTTPError(e)
	default:
		apiErr = eh.handleUnexpectedError(err)
	}

	// Add context information
	apiErr = apiErr.WithContext(c)

	// Record metrics
	eh.recordErrorMetrics(apiErr)

	// Log error
	eh.logError(apiErr, c)

	// Send notification if needed
	if eh.shouldNotify(apiErr) {
		eh.sendNotification(apiErr, c)
	}

	// Sanitize error for production
	if eh.config.SanitizeErrors && !eh.config.DebugMode {
		apiErr = eh.sanitizeError(apiErr)
	}

	// Send response
	response := NewErrorResponse(apiErr)
	c.JSON(apiErr.HTTPStatus(), response)

	// Record recovery time
	eh.metrics.RecoveryTime.WithLabelValues(
		"api", string(apiErr.Code),
	).Observe(time.Since(start).Seconds())
}

// WithRetry executes a function with retry logic
func (eh *ErrorHandler) WithRetry(ctx context.Context, operation string, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= eh.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(eh.config.RetryDelay * time.Duration(attempt)):
			}
		}

		err := fn()
		if err == nil {
			if attempt > 0 {
				eh.metrics.RetryAttempts.WithLabelValues(operation, "success").Inc()
			}
			return nil
		}

		lastErr = err
		eh.metrics.RetryAttempts.WithLabelValues(operation, "failure").Inc()

		// Don't retry certain errors
		if !eh.shouldRetry(err) {
			break
		}
	}

	return lastErr
}

// GetCircuitBreaker returns or creates a circuit breaker for a service
func (eh *ErrorHandler) GetCircuitBreaker(serviceName string) *gobreaker.CircuitBreaker {
	eh.mu.RLock()
	cb, exists := eh.circuitBreakers[serviceName]
	eh.mu.RUnlock()

	if exists {
		return cb
	}

	eh.mu.Lock()
	defer eh.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists := eh.circuitBreakers[serviceName]; exists {
		return cb
	}

	// Create new circuit breaker
	settings := gobreaker.Settings{
		Name:        serviceName,
		MaxRequests: 3,
		Interval:    time.Minute,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Printf("Circuit breaker %s changed from %s to %s", name, from, to)
			eh.metrics.CircuitBreakerState.WithLabelValues(name).Set(float64(to))
		},
	}

	cb = gobreaker.NewCircuitBreaker(settings)
	eh.circuitBreakers[serviceName] = cb

	return cb
}

// Helper methods

func (eh *ErrorHandler) convertHTTPError(httpErr *echo.HTTPError) *APIError {
	return &APIError{
		Code:      ErrInternalServer,
		Message:   fmt.Sprintf("%v", httpErr.Message),
		Timestamp: time.Now().UTC(),
	}
}

func (eh *ErrorHandler) handleUnexpectedError(err error) *APIError {
	if eh.config.LogStackTrace {
		log.Printf("Unexpected error: %v\nStack trace: %s", err, debug.Stack())
	}

	return &APIError{
		Code:      ErrInternalServer,
		Message:   "An unexpected error occurred",
		Details:   err.Error(),
		Timestamp: time.Now().UTC(),
	}
}

func (eh *ErrorHandler) handlePanic(r interface{}, c echo.Context) {
	log.Printf("Panic recovered: %v\nStack trace: %s", r, debug.Stack())

	apiErr := &APIError{
		Code:      ErrInternalServer,
		Message:   "Internal server error",
		Timestamp: time.Now().UTC(),
	}

	if !c.Response().Committed {
		c.JSON(500, NewErrorResponse(apiErr))
	}
}

func (eh *ErrorHandler) recordErrorMetrics(apiErr *APIError) {
	severity := eh.getErrorSeverity(apiErr)
	component := "api"

	eh.metrics.ErrorsTotal.WithLabelValues(
		string(apiErr.Code), severity, component,
	).Inc()

	eh.metrics.ErrorsByType.WithLabelValues(
		string(apiErr.Code), fmt.Sprintf("%d", apiErr.HTTPStatus()),
	).Inc()
}

func (eh *ErrorHandler) logError(apiErr *APIError, c echo.Context) {
	log.Printf("Error [%s]: %s - %s (Path: %s, Method: %s, RequestID: %s)",
		apiErr.Code, apiErr.Message, apiErr.Details,
		apiErr.Path, apiErr.Method, apiErr.RequestID)
}

func (eh *ErrorHandler) shouldNotify(apiErr *APIError) bool {
	return eh.config.NotificationEnabled && eh.getErrorSeverity(apiErr) == "critical"
}

func (eh *ErrorHandler) sendNotification(apiErr *APIError, c echo.Context) {
	notification := &ErrorNotification{
		Error:     apiErr,
		Severity:  eh.getErrorSeverity(apiErr),
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"user_agent": c.Request().UserAgent(),
			"ip":         c.RealIP(),
		},
	}

	select {
	case eh.notificationCh <- notification:
	default:
		log.Println("Notification channel full, dropping notification")
	}
}

func (eh *ErrorHandler) sanitizeError(apiErr *APIError) *APIError {
	sanitized := *apiErr
	sanitized.Details = ""
	if apiErr.Code == ErrInternalServer {
		sanitized.Message = "Internal server error"
	}
	return &sanitized
}

func (eh *ErrorHandler) shouldRetry(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		switch apiErr.Code {
		case ErrDatabaseTimeout, ErrTimeout, ErrServiceUnavailable:
			return true
		default:
			return false
		}
	}
	return true
}

func (eh *ErrorHandler) getErrorSeverity(apiErr *APIError) string {
	switch apiErr.Code {
	case ErrInternalServer, ErrDatabaseConnection, ErrServiceUnavailable:
		return "critical"
	case ErrDatabaseTimeout, ErrTimeout:
		return "warning"
	default:
		return "info"
	}
}

func (eh *ErrorHandler) processNotifications() {
	for notification := range eh.notificationCh {
		// Process notification (send to external systems, etc.)
		log.Printf("Processing notification: %s error at %s",
			notification.Severity, notification.Timestamp)
	}
}

func (eh *ErrorHandler) healthCheckMonitor() {
	ticker := time.NewTicker(eh.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		// Perform health checks and update metrics
		eh.performHealthChecks()
	}
}

func (eh *ErrorHandler) performHealthChecks() {
	// Implement health check logic
	log.Println("Performing health checks...")
}
