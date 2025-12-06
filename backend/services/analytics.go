package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// AnalyticsService handles API usage analytics and monitoring
type AnalyticsService struct {
	db          *sql.DB
	metrics     map[string]*APIMetrics
	mutex       sync.RWMutex
	flushTicker *time.Ticker
	stopChan    chan bool
}

// APIMetrics holds real-time metrics for API usage
type APIMetrics struct {
	TotalRequests   int64            `json:"total_requests"`
	SuccessRequests int64            `json:"success_requests"`
	ErrorRequests   int64            `json:"error_requests"`
	AverageLatency  float64          `json:"average_latency"`
	EndpointStats   map[string]int64 `json:"endpoint_stats"`
	StatusCodeStats map[int]int64    `json:"status_code_stats"`
	HourlyStats     map[string]int64 `json:"hourly_stats"`
	LastUpdated     time.Time        `json:"last_updated"`
	TopUserAgents   map[string]int64 `json:"top_user_agents"`
	TopIPAddresses  map[string]int64 `json:"top_ip_addresses"`
	ResponseTimes   []float64        `json:"-"` // For calculating average
}

// UsageAlert represents an alert condition
type UsageAlert struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Condition     string                 `json:"condition"`
	Threshold     float64                `json:"threshold"`
	Enabled       bool                   `json:"enabled"`
	LastTriggered *time.Time             `json:"last_triggered"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(db *sql.DB) *AnalyticsService {
	service := &AnalyticsService{
		db:       db,
		metrics:  make(map[string]*APIMetrics),
		stopChan: make(chan bool),
	}

	// Start background metrics flushing
	service.startMetricsFlushing()

	return service
}

// RecordAPIUsage records API usage metrics
func (s *AnalyticsService) RecordAPIUsage(apiKeyID, endpoint, method string, statusCode int, responseTime int64, ipAddress, userAgent string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Get or create metrics for this API key
	metrics, exists := s.metrics[apiKeyID]
	if !exists {
		metrics = &APIMetrics{
			EndpointStats:   make(map[string]int64),
			StatusCodeStats: make(map[int]int64),
			HourlyStats:     make(map[string]int64),
			TopUserAgents:   make(map[string]int64),
			TopIPAddresses:  make(map[string]int64),
			ResponseTimes:   make([]float64, 0),
		}
		s.metrics[apiKeyID] = metrics
	}

	// Update metrics
	metrics.TotalRequests++
	if statusCode >= 200 && statusCode < 400 {
		metrics.SuccessRequests++
	} else {
		metrics.ErrorRequests++
	}

	// Update endpoint stats
	endpointKey := fmt.Sprintf("%s %s", method, endpoint)
	metrics.EndpointStats[endpointKey]++

	// Update status code stats
	metrics.StatusCodeStats[statusCode]++

	// Update hourly stats
	hourKey := time.Now().Format("2006-01-02-15")
	metrics.HourlyStats[hourKey]++

	// Update top user agents (keep top 10)
	metrics.TopUserAgents[userAgent]++
	if len(metrics.TopUserAgents) > 10 {
		s.trimMap(metrics.TopUserAgents, 10)
	}

	// Update top IP addresses (keep top 10)
	metrics.TopIPAddresses[ipAddress]++
	if len(metrics.TopIPAddresses) > 10 {
		s.trimMap(metrics.TopIPAddresses, 10)
	}

	// Update response times (keep last 1000 for average calculation)
	metrics.ResponseTimes = append(metrics.ResponseTimes, float64(responseTime))
	if len(metrics.ResponseTimes) > 1000 {
		metrics.ResponseTimes = metrics.ResponseTimes[len(metrics.ResponseTimes)-1000:]
	}

	// Calculate average latency
	if len(metrics.ResponseTimes) > 0 {
		var sum float64
		for _, rt := range metrics.ResponseTimes {
			sum += rt
		}
		metrics.AverageLatency = sum / float64(len(metrics.ResponseTimes))
	}

	metrics.LastUpdated = time.Now()

	// Check for alerts
	go s.checkAlerts(apiKeyID, metrics)
}

// GetAPIMetrics retrieves metrics for a specific API key
func (s *AnalyticsService) GetAPIMetrics(apiKeyID string) (*APIMetrics, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	metrics, exists := s.metrics[apiKeyID]
	if !exists {
		return nil, fmt.Errorf("no metrics found for API key: %s", apiKeyID)
	}

	// Create a copy to avoid race conditions
	metricsCopy := *metrics
	metricsCopy.EndpointStats = make(map[string]int64)
	metricsCopy.StatusCodeStats = make(map[int]int64)
	metricsCopy.HourlyStats = make(map[string]int64)
	metricsCopy.TopUserAgents = make(map[string]int64)
	metricsCopy.TopIPAddresses = make(map[string]int64)

	for k, v := range metrics.EndpointStats {
		metricsCopy.EndpointStats[k] = v
	}
	for k, v := range metrics.StatusCodeStats {
		metricsCopy.StatusCodeStats[k] = v
	}
	for k, v := range metrics.HourlyStats {
		metricsCopy.HourlyStats[k] = v
	}
	for k, v := range metrics.TopUserAgents {
		metricsCopy.TopUserAgents[k] = v
	}
	for k, v := range metrics.TopIPAddresses {
		metricsCopy.TopIPAddresses[k] = v
	}

	return &metricsCopy, nil
}

// GetGlobalMetrics retrieves aggregated metrics across all API keys
func (s *AnalyticsService) GetGlobalMetrics() (*APIMetrics, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	globalMetrics := &APIMetrics{
		EndpointStats:   make(map[string]int64),
		StatusCodeStats: make(map[int]int64),
		HourlyStats:     make(map[string]int64),
		TopUserAgents:   make(map[string]int64),
		TopIPAddresses:  make(map[string]int64),
		ResponseTimes:   make([]float64, 0),
	}

	for _, metrics := range s.metrics {
		globalMetrics.TotalRequests += metrics.TotalRequests
		globalMetrics.SuccessRequests += metrics.SuccessRequests
		globalMetrics.ErrorRequests += metrics.ErrorRequests

		// Aggregate endpoint stats
		for endpoint, count := range metrics.EndpointStats {
			globalMetrics.EndpointStats[endpoint] += count
		}

		// Aggregate status code stats
		for code, count := range metrics.StatusCodeStats {
			globalMetrics.StatusCodeStats[code] += count
		}

		// Aggregate hourly stats
		for hour, count := range metrics.HourlyStats {
			globalMetrics.HourlyStats[hour] += count
		}

		// Aggregate user agents
		for ua, count := range metrics.TopUserAgents {
			globalMetrics.TopUserAgents[ua] += count
		}

		// Aggregate IP addresses
		for ip, count := range metrics.TopIPAddresses {
			globalMetrics.TopIPAddresses[ip] += count
		}

		// Collect response times for average calculation
		globalMetrics.ResponseTimes = append(globalMetrics.ResponseTimes, metrics.ResponseTimes...)
	}

	// Calculate global average latency
	if len(globalMetrics.ResponseTimes) > 0 {
		var sum float64
		for _, rt := range globalMetrics.ResponseTimes {
			sum += rt
		}
		globalMetrics.AverageLatency = sum / float64(len(globalMetrics.ResponseTimes))
	}

	// Trim maps to keep only top entries
	s.trimMap(globalMetrics.TopUserAgents, 10)
	s.trimMap(globalMetrics.TopIPAddresses, 10)

	globalMetrics.LastUpdated = time.Now()
	return globalMetrics, nil
}

// GetUsageReport generates a comprehensive usage report
func (s *AnalyticsService) GetUsageReport(apiKeyID string, days int) (*UsageReport, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	query := `
		SELECT 
			DATE(timestamp) as date,
			COUNT(*) as total_requests,
			COUNT(CASE WHEN status_code >= 200 AND status_code < 400 THEN 1 END) as success_requests,
			COUNT(CASE WHEN status_code >= 400 THEN 1 END) as error_requests,
			AVG(response_time) as avg_response_time,
			MIN(response_time) as min_response_time,
			MAX(response_time) as max_response_time
		FROM api_key_usage 
		WHERE api_key_id = $1 AND timestamp >= $2 AND timestamp <= $3
		GROUP BY DATE(timestamp)
		ORDER BY date
	`

	rows, err := s.db.Query(query, apiKeyID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage report: %w", err)
	}
	defer rows.Close()

	var dailyStats []DailyStats
	for rows.Next() {
		var stats DailyStats
		err := rows.Scan(
			&stats.Date,
			&stats.TotalRequests,
			&stats.SuccessRequests,
			&stats.ErrorRequests,
			&stats.AvgResponseTime,
			&stats.MinResponseTime,
			&stats.MaxResponseTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily stats: %w", err)
		}
		dailyStats = append(dailyStats, stats)
	}

	// Get current metrics
	currentMetrics, err := s.GetAPIMetrics(apiKeyID)
	if err != nil {
		currentMetrics = &APIMetrics{} // Empty metrics if not found
	}

	report := &UsageReport{
		APIKeyID:       apiKeyID,
		StartDate:      startDate,
		EndDate:        endDate,
		DailyStats:     dailyStats,
		CurrentMetrics: currentMetrics,
		GeneratedAt:    time.Now(),
	}

	return report, nil
}

// CreateAlert creates a new usage alert
func (s *AnalyticsService) CreateAlert(alert *UsageAlert) error {
	query := `
		INSERT INTO usage_alerts (id, name, condition, threshold, enabled, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	metadataJSON, err := json.Marshal(alert.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = s.db.Exec(query, alert.ID, alert.Name, alert.Condition, alert.Threshold, alert.Enabled, metadataJSON, time.Now())
	return err
}

// checkAlerts checks if any alerts should be triggered
func (s *AnalyticsService) checkAlerts(apiKeyID string, metrics *APIMetrics) {
	// This is a simplified alert checking mechanism
	// In production, you'd want more sophisticated alert conditions

	// Example: Check if error rate is too high
	if metrics.TotalRequests > 0 {
		errorRate := float64(metrics.ErrorRequests) / float64(metrics.TotalRequests) * 100
		if errorRate > 10.0 { // 10% error rate threshold
			s.triggerAlert("high_error_rate", apiKeyID, errorRate)
		}
	}

	// Example: Check if average latency is too high
	if metrics.AverageLatency > 5000 { // 5 seconds threshold
		s.triggerAlert("high_latency", apiKeyID, metrics.AverageLatency)
	}
}

// triggerAlert triggers an alert
func (s *AnalyticsService) triggerAlert(alertType, apiKeyID string, value float64) {
	log.Printf("ALERT: %s for API key %s - Value: %.2f", alertType, apiKeyID, value)

	// Here you would implement actual alerting mechanisms:
	// - Send email notifications
	// - Send Slack/Discord messages
	// - Create tickets in issue tracking systems
	// - Store alert in database for dashboard display
}

// startMetricsFlushing starts background metrics flushing to database
func (s *AnalyticsService) startMetricsFlushing() {
	s.flushTicker = time.NewTicker(5 * time.Minute) // Flush every 5 minutes

	go func() {
		for {
			select {
			case <-s.flushTicker.C:
				s.flushMetricsToDatabase()
			case <-s.stopChan:
				s.flushTicker.Stop()
				return
			}
		}
	}()
}

// flushMetricsToDatabase persists current metrics to database
func (s *AnalyticsService) flushMetricsToDatabase() {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for apiKeyID, metrics := range s.metrics {
		metricsJSON, err := json.Marshal(metrics)
		if err != nil {
			log.Printf("Failed to marshal metrics for API key %s: %v", apiKeyID, err)
			continue
		}

		query := `
			INSERT INTO api_metrics_snapshots (api_key_id, metrics_data, timestamp)
			VALUES ($1, $2, $3)
			ON CONFLICT (api_key_id, timestamp) DO UPDATE SET
			metrics_data = EXCLUDED.metrics_data
		`

		timestamp := time.Now().Truncate(5 * time.Minute) // Round to 5-minute intervals
		_, err = s.db.Exec(query, apiKeyID, metricsJSON, timestamp)
		if err != nil {
			log.Printf("Failed to flush metrics for API key %s: %v", apiKeyID, err)
		}
	}
}

// trimMap keeps only the top N entries in a map by value
func (s *AnalyticsService) trimMap(m map[string]int64, n int) {
	if len(m) <= n {
		return
	}

	// Convert to slice for sorting
	type kv struct {
		key   string
		value int64
	}

	var kvs []kv
	for k, v := range m {
		kvs = append(kvs, kv{k, v})
	}

	// Sort by value (descending)
	for i := 0; i < len(kvs)-1; i++ {
		for j := i + 1; j < len(kvs); j++ {
			if kvs[i].value < kvs[j].value {
				kvs[i], kvs[j] = kvs[j], kvs[i]
			}
		}
	}

	// Clear map and add top N entries
	for k := range m {
		delete(m, k)
	}
	for i := 0; i < n && i < len(kvs); i++ {
		m[kvs[i].key] = kvs[i].value
	}
}

// Stop stops the analytics service
func (s *AnalyticsService) Stop() {
	close(s.stopChan)
	s.flushMetricsToDatabase() // Final flush
}

// Supporting types

type UsageReport struct {
	APIKeyID       string       `json:"api_key_id"`
	StartDate      time.Time    `json:"start_date"`
	EndDate        time.Time    `json:"end_date"`
	DailyStats     []DailyStats `json:"daily_stats"`
	CurrentMetrics *APIMetrics  `json:"current_metrics"`
	GeneratedAt    time.Time    `json:"generated_at"`
}

type DailyStats struct {
	Date            string  `json:"date"`
	TotalRequests   int64   `json:"total_requests"`
	SuccessRequests int64   `json:"success_requests"`
	ErrorRequests   int64   `json:"error_requests"`
	AvgResponseTime float64 `json:"avg_response_time"`
	MinResponseTime int64   `json:"min_response_time"`
	MaxResponseTime int64   `json:"max_response_time"`
}
