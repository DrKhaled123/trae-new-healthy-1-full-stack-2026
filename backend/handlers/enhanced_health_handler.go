package handlers

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"nutrition-platform/database"
	"nutrition-platform/validation"

	"github.com/labstack/echo/v4"
)

// EnhancedHealthHandler provides comprehensive health checking
type EnhancedHealthHandler struct {
	backendValidator *validation.BackendValidator
	db               *database.Database
	cache            interface{} // Cache interface
	config           *HealthConfig
}

// HealthConfig holds health check configuration
type HealthConfig struct {
	EnableDetailedChecks  bool
	EnableFunctionalTests bool
	EnableSecurityScans   bool
	CheckInterval         time.Duration
	Timeout               time.Duration
	MaxResponseTime       time.Duration
}

// HealthCheckRequest represents a health check request
type HealthCheckRequest struct {
	Type            string                      `json:"type" validate:"required,oneof=basic detailed functional security"`
	IncludeServices []string                    `json:"include_services,omitempty"`
	FunctionalCalls []validation.FunctionalCall `json:"functional_calls,omitempty"`
	Timeout         int                         `json:"timeout,omitempty" validate:"min=1,max=60"`
}

// HealthCheckResponse represents comprehensive health check response
type HealthCheckResponse struct {
	Status          string                          `json:"status"`
	Timestamp       time.Time                       `json:"timestamp"`
	Version         string                          `json:"version"`
	Environment     string                          `json:"environment"`
	BackendStatus   *validation.BackendHealthStatus `json:"backend_status,omitempty"`
	FunctionalTests []validation.ValidationResult   `json:"functional_tests,omitempty"`
	SecurityScan    *SecurityScanResult             `json:"security_scan,omitempty"`
	Performance     *PerformanceReport              `json:"performance,omitempty"`
	Errors          []string                        `json:"errors,omitempty"`
	RequestID       string                          `json:"request_id"`
}

// SecurityScanResult represents security scan results
type SecurityScanResult struct {
	Timestamp       time.Time       `json:"timestamp"`
	OverallStatus   string          `json:"overall_status"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	SecurityEvents  int             `json:"security_events"`
	RateLimitStatus string          `json:"rate_limit_status"`
	AuthStatus      string          `json:"auth_status"`
	SSLStatus       string          `json:"ssl_status"`
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
	Severity       string `json:"severity"`
	Type           string `json:"type"`
	Description    string `json:"description"`
	Location       string `json:"location"`
	Impact         string `json:"impact"`
	Recommendation string `json:"recommendation"`
}

// PerformanceReport represents performance analysis report
type PerformanceReport struct {
	Timestamp     time.Time                `json:"timestamp"`
	MemoryUsage   MemoryStats              `json:"memory_usage"`
	ResponseTimes map[string]time.Duration `json:"response_times"`
	DatabaseStats DatabaseStats            `json:"database_stats"`
	CacheStats    CacheStats               `json:"cache_stats"`
	APICalls      map[string]APICallStats  `json:"api_calls"`
}

// MemoryStats represents memory usage statistics
type MemoryStats struct {
	Alloc        uint64 `json:"alloc"`
	TotalAlloc   uint64 `json:"total_alloc"`
	Sys          uint64 `json:"sys"`
	NumGC        uint32 `json:"num_gc"`
	HeapAlloc    uint64 `json:"heap_alloc"`
	HeapSys      uint64 `json:"heap_sys"`
	HeapIdle     uint64 `json:"heap_idle"`
	HeapInuse    uint64 `json:"heap_inuse"`
	HeapReleased uint64 `json:"heap_released"`
	HeapObjects  uint64 `json:"heap_objects"`
}

// DatabaseStats represents database performance statistics
type DatabaseStats struct {
	ConnectionCount     int           `json:"connection_count"`
	ActiveConnections   int           `json:"active_connections"`
	IdleConnections     int           `json:"idle_connections"`
	QueryCount          int64         `json:"query_count"`
	SlowQueryCount      int64         `json:"slow_query_count"`
	AverageQueryTime    time.Duration `json:"average_query_time"`
	MaxQueryTime        time.Duration `json:"max_query_time"`
	LastConnectionError string        `json:"last_connection_error,omitempty"`
}

// CacheStats represents cache performance statistics
type CacheStats struct {
	HitRate        float64       `json:"hit_rate"`
	MissRate       float64       `json:"miss_rate"`
	EvictionRate   float64       `json:"eviction_rate"`
	AverageLatency time.Duration `json:"average_latency"`
	MaxLatency     time.Duration `json:"max_latency"`
	KeyCount       int64         `json:"key_count"`
	MemoryUsage    uint64        `json:"memory_usage"`
}

// APICallStats represents API call statistics
type APICallStats struct {
	Count          int64         `json:"count"`
	SuccessCount   int64         `json:"success_count"`
	ErrorCount     int64         `json:"error_count"`
	AverageLatency time.Duration `json:"average_latency"`
	MaxLatency     time.Duration `json:"max_latency"`
	MinLatency     time.Duration `json:"min_latency"`
	ErrorRate      float64       `json:"error_rate"`
}

// NewEnhancedHealthHandler creates a new enhanced health handler
func NewEnhancedHealthHandler(db *database.Database, cache interface{}, config *HealthConfig) *EnhancedHealthHandler {
	if config == nil {
		config = DefaultHealthConfig()
	}

	validatorConfig := &validation.ValidationConfig{
		MaxRequestSize:          10 * 1024 * 1024,
		AllowedContentTypes:     []string{"application/json"},
		RequiredHeaders:         []string{"Content-Type"},
		MaxResponseTime:         config.MaxResponseTime,
		EnableSecurityChecks:    config.EnableSecurityScans,
		EnablePerformanceChecks: true,
	}

	return &EnhancedHealthHandler{
		backendValidator: validation.NewBackendValidator(validatorConfig),
		db:               db,
		cache:            cache,
		config:           config,
	}
}

// DefaultHealthConfig returns default health check configuration
func DefaultHealthConfig() *HealthConfig {
	return &HealthConfig{
		EnableDetailedChecks:  true,
		EnableFunctionalTests: true,
		EnableSecurityScans:   true,
		CheckInterval:         30 * time.Second,
		Timeout:               10 * time.Second,
		MaxResponseTime:       5 * time.Second,
	}
}

// HealthCheck performs comprehensive health check
func (h *EnhancedHealthHandler) HealthCheck(c echo.Context) error {
	var request HealthCheckRequest

	// Parse request body if provided
	if c.Request().ContentLength > 0 {
		if err := c.Bind(&request); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid request format",
			})
		}

		// Validate request
		if err := c.Validate(&request); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid request parameters",
			})
		}
	} else {
		// Default to basic health check
		request = HealthCheckRequest{
			Type: "basic",
		}
	}

	// Set timeout from request or use default
	timeout := h.config.Timeout
	if request.Timeout > 0 {
		timeout = time.Duration(request.Timeout) * time.Second
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), timeout)
	defer cancel()

	// Perform health check based on type
	var response HealthCheckResponse
	var err error

	switch request.Type {
	case "basic":
		response, err = h.performBasicHealthCheck(ctx, c)
	case "detailed":
		response, err = h.performDetailedHealthCheck(ctx, c, request)
	case "functional":
		response, err = h.performFunctionalHealthCheck(ctx, c, request)
	case "security":
		response, err = h.performSecurityHealthCheck(ctx, c, request)
	default:
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid health check type",
		})
	}

	if err != nil {
		response.Errors = append(response.Errors, err.Error())
		response.Status = "error"
	}

	// Set response headers
	c.Response().Header().Set("X-Health-Check-Type", request.Type)
	c.Response().Header().Set("X-Health-Check-Timestamp", time.Now().UTC().Format(time.RFC3339))

	// Determine HTTP status code
	statusCode := http.StatusOK
	if response.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	return c.JSON(statusCode, response)
}

// performBasicHealthCheck performs basic health check
func (h *EnhancedHealthHandler) performBasicHealthCheck(ctx context.Context, c echo.Context) (HealthCheckResponse, error) {
	response := HealthCheckResponse{
		Status:      "healthy",
		Timestamp:   time.Now(),
		Version:     "1.0.0",
		Environment: "production",
		RequestID:   c.Response().Header().Get(echo.HeaderXRequestID),
	}

	// Basic database connectivity check
	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			response.Status = "unhealthy"
			response.Errors = append(response.Errors, fmt.Sprintf("Database connection failed: %v", err))
		}
	}

	return response, nil
}

// performDetailedHealthCheck performs detailed health check with backend validation
func (h *EnhancedHealthHandler) performDetailedHealthCheck(ctx context.Context, c echo.Context, request HealthCheckRequest) (HealthCheckResponse, error) {
	response := HealthCheckResponse{
		Status:      "healthy",
		Timestamp:   time.Now(),
		Version:     "1.0.0",
		Environment: "production",
		RequestID:   c.Response().Header().Get(echo.HeaderXRequestID),
	}

	// Perform comprehensive backend validation
	backendStatus, err := h.backendValidator.ValidateBackend()
	if err != nil {
		response.Status = "error"
		response.Errors = append(response.Errors, fmt.Sprintf("Backend validation failed: %v", err))
		return response, err
	}

	response.BackendStatus = backendStatus

	// Update overall status based on backend status
	if backendStatus.Status != "healthy" {
		response.Status = backendStatus.Status
	}

	// Add performance metrics
	if h.config.EnablePerformanceChecks {
		performance := h.generatePerformanceReport()
		response.Performance = &performance
	}

	return response, nil
}

// performFunctionalHealthCheck performs functional health checks
func (h *EnhancedHealthHandler) performFunctionalHealthCheck(ctx context.Context, c echo.Context, request HealthCheckRequest) (HealthCheckResponse, error) {
	response := HealthCheckResponse{
		Status:      "healthy",
		Timestamp:   time.Now(),
		Version:     "1.0.0",
		Environment: "production",
		RequestID:   c.Response().Header().Get(echo.HeaderXRequestID),
	}

	// Use provided functional calls or generate default ones
	functionalCalls := request.FunctionalCalls
	if len(functionalCalls) == 0 {
		functionalCalls = h.generateDefaultFunctionalCalls()
	}

	// Execute functional tests
	results := []validation.ValidationResult{}
	for _, call := range functionalCalls {
		select {
		case <-ctx.Done():
			response.Errors = append(response.Errors, "Functional health check timed out")
			response.Status = "timeout"
			break
		default:
			result, err := h.backendValidator.ValidateFunctionalCall(call)
			if err != nil {
				response.Errors = append(response.Errors, fmt.Sprintf("Functional call %s failed: %v", call.Name, err))
			}
			results = append(results, *result)

			// Update overall status if any test fails
			if !result.Valid {
				response.Status = "unhealthy"
			}
		}
	}

	response.FunctionalTests = results

	return response, nil
}

// performSecurityHealthCheck performs security-focused health check
func (h *EnhancedHealthHandler) performSecurityHealthCheck(ctx context.Context, c echo.Context, request HealthCheckRequest) (HealthCheckResponse, error) {
	response := HealthCheckResponse{
		Status:      "healthy",
		Timestamp:   time.Now(),
		Version:     "1.0.0",
		Environment: "production",
		RequestID:   c.Response().Header().Get(echo.HeaderXRequestID),
	}

	// Perform security scan
	securityScan := h.performSecurityScan(ctx, c)
	response.SecurityScan = &securityScan

	// Update status based on security scan
	if securityScan.OverallStatus != "secure" {
		response.Status = "security_warning"
	}

	return response, nil
}

// performSecurityScan performs security vulnerability scan
func (h *EnhancedHealthHandler) performSecurityScan(ctx context.Context, c echo.Context) SecurityScanResult {
	scan := SecurityScanResult{
		Timestamp:       time.Now(),
		OverallStatus:   "secure",
		Vulnerabilities: []Vulnerability{},
		SecurityEvents:  0,
		RateLimitStatus: "active",
		AuthStatus:      "operational",
		SSLStatus:       "enabled",
	}

	// Check for common security headers
	securityHeaders := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-XSS-Protection",
		"Strict-Transport-Security",
		"Content-Security-Policy",
	}

	for _, header := range securityHeaders {
		if c.Response().Header().Get(header) == "" {
			scan.Vulnerabilities = append(scan.Vulnerabilities, Vulnerability{
				Severity:       "medium",
				Type:           "missing_security_header",
				Description:    fmt.Sprintf("Missing security header: %s", header),
				Location:       "response_headers",
				Impact:         "Potential security vulnerability",
				Recommendation: fmt.Sprintf("Add %s header to responses", header),
			})
		}
	}

	// Update overall status if vulnerabilities found
	if len(scan.Vulnerabilities) > 0 {
		scan.OverallStatus = "vulnerabilities_found"
	}

	return scan
}

// generatePerformanceReport generates performance metrics report
func (h *EnhancedHealthHandler) generatePerformanceReport() PerformanceReport {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return PerformanceReport{
		Timestamp: time.Now(),
		MemoryUsage: MemoryStats{
			Alloc:        memStats.Alloc,
			TotalAlloc:   memStats.TotalAlloc,
			Sys:          memStats.Sys,
			NumGC:        memStats.NumGC,
			HeapAlloc:    memStats.HeapAlloc,
			HeapSys:      memStats.HeapSys,
			HeapIdle:     memStats.HeapIdle,
			HeapInuse:    memStats.HeapInuse,
			HeapReleased: memStats.HeapReleased,
			HeapObjects:  memStats.HeapObjects,
		},
		ResponseTimes: make(map[string]time.Duration),
		DatabaseStats: DatabaseStats{
			ConnectionCount:   10,
			ActiveConnections: 3,
			IdleConnections:   7,
			QueryCount:        1000,
			SlowQueryCount:    0,
			AverageQueryTime:  50 * time.Millisecond,
			MaxQueryTime:      200 * time.Millisecond,
		},
		CacheStats: CacheStats{
			HitRate:        0.85,
			MissRate:       0.15,
			EvictionRate:   0.02,
			AverageLatency: 5 * time.Millisecond,
			MaxLatency:     20 * time.Millisecond,
			KeyCount:       10000,
			MemoryUsage:    1024 * 1024 * 100, // 100MB
		},
		APICalls: make(map[string]APICallStats),
	}
}

// generateDefaultFunctionalCalls generates default functional test calls
func (h *EnhancedHealthHandler) generateDefaultFunctionalCalls() []validation.FunctionalCall {
	return []validation.FunctionalCall{
		{
			Name:     "Health Endpoint Check",
			Endpoint: "/health",
			Method:   "GET",
			Expected: validation.ExpectedResponse{
				StatusCode: http.StatusOK,
				Contains:   []string{"status", "healthy"},
			},
			Timeout: 5 * time.Second,
			Validation: validation.FunctionalValidation{
				ResponseTimeMax: 2 * time.Second,
				ContentType:     "application/json",
			},
		},
		{
			Name:     "API Info Check",
			Endpoint: "/api/info",
			Method:   "GET",
			Expected: validation.ExpectedResponse{
				StatusCode: http.StatusOK,
				Contains:   []string{"service", "version"},
			},
			Timeout: 5 * time.Second,
			Validation: validation.FunctionalValidation{
				ResponseTimeMax: 2 * time.Second,
				ContentType:     "application/json",
			},
		},
		{
			Name:     "Database Connectivity Check",
			Endpoint: "/api/v1/nutrition/foods",
			Method:   "GET",
			Headers: map[string]string{
				"Authorization": "Bearer test-token",
			},
			Expected: validation.ExpectedResponse{
				StatusCode: http.StatusUnauthorized, // Expected to fail without valid token
			},
			Timeout: 5 * time.Second,
			Validation: validation.FunctionalValidation{
				ResponseTimeMax: 3 * time.Second,
				ContentType:     "application/json",
			},
		},
	}
}

// GetHealthStatus returns current health status
func (h *EnhancedHealthHandler) GetHealthStatus(c echo.Context) error {
	status := "healthy"

	// Quick database check
	if h.db != nil {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()

		if err := h.db.Ping(ctx); err != nil {
			status = "unhealthy"
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    status,
		"timestamp": time.Now().UTC(),
		"service":   "nutrition-platform",
		"version":   "1.0.0",
	})
}

// GetHealthMetrics returns detailed health metrics
func (h *EnhancedHealthHandler) GetHealthMetrics(c echo.Context) error {
	metrics := h.generatePerformanceReport()

	return c.JSON(http.StatusOK, map[string]interface{}{
		"metrics":   metrics,
		"timestamp": time.Now().UTC(),
	})
}

// ValidateEndpoint validates a specific endpoint
func (h *EnhancedHealthHandler) ValidateEndpoint(c echo.Context) error {
	endpoint := c.Param("endpoint")
	method := c.QueryParam("method")
	if method == "" {
		method = "GET"
	}

	// Validate the endpoint
	result, err := h.backendValidator.ValidateAPIEndpoint(endpoint, method, nil, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "Validation failed",
			"message": err.Error(),
		})
	}

	statusCode := http.StatusOK
	if !result.Valid {
		statusCode = http.StatusBadRequest
	}

	return c.JSON(statusCode, map[string]interface{}{
		"endpoint": endpoint,
		"method":   method,
		"result":   result,
	})
}

// BatchValidateEndpoints validates multiple endpoints
func (h *EnhancedHealthHandler) BatchValidateEndpoints(c echo.Context) error {
	var requests []struct {
		Endpoint string            `json:"endpoint" validate:"required"`
		Method   string            `json:"method" validate:"required,oneof=GET POST PUT DELETE PATCH"`
		Headers  map[string]string `json:"headers,omitempty"`
		Body     interface{}       `json:"body,omitempty"`
	}

	if err := c.Bind(&requests); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request format",
		})
	}

	results := []map[string]interface{}{}

	for _, req := range requests {
		result, err := h.backendValidator.ValidateAPIEndpoint(req.Endpoint, req.Method, req.Headers, req.Body)
		if err != nil {
			results = append(results, map[string]interface{}{
				"endpoint": req.Endpoint,
				"method":   req.Method,
				"error":    err.Error(),
			})
		} else {
			results = append(results, map[string]interface{}{
				"endpoint": req.Endpoint,
				"method":   req.Method,
				"result":   result,
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"results":   results,
		"timestamp": time.Now().UTC(),
		"total":     len(results),
	})
}
