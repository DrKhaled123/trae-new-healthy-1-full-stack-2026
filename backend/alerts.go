// alerts.go - Alerting system for critical errors and performance issues
package main

import (
	"fmt"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AlertLevel represents different alert severity levels
type AlertLevel int

const (
	AlertInfo AlertLevel = iota
	AlertWarning
	AlertError
	AlertCritical
)

// String returns the string representation of AlertLevel
func (l AlertLevel) String() string {
	switch l {
	case AlertInfo:
		return "INFO"
	case AlertWarning:
		return "WARNING"
	case AlertError:
		return "ERROR"
	case AlertCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Alert represents an alert configuration
type Alert struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Level       AlertLevel        `json:"level"`
	Condition   string            `json:"condition"`
	Threshold   float64           `json:"threshold"`
	TimeWindow  time.Duration     `json:"time_window"`
	Cooldown    time.Duration     `json:"cooldown"`
	Channels    []string          `json:"channels"`
	Enabled     bool              `json:"enabled"`
	LastFired   *time.Time        `json:"last_fired,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// AlertManager manages alerts and notifications
type AlertManager struct {
	alerts       map[string]*Alert
	alertHistory []AlertEvent
	mu           sync.RWMutex
	// Alert channels
	emailConfig  *EmailConfig
	slackWebhook string
	webhookURL   string
}

// AlertEvent represents a fired alert event
type AlertEvent struct {
	ID        string            `json:"id"`
	AlertID   string            `json:"alert_id"`
	Level     AlertLevel        `json:"level"`
	Message   string            `json:"message"`
	Value     float64           `json:"value"`
	Threshold float64           `json:"threshold"`
	Timestamp time.Time         `json:"timestamp"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// EmailConfig holds email configuration
type EmailConfig struct {
	SMTPHost    string
	SMTPPort    int
	Username    string
	Password    string
	FromAddress string
	ToAddresses []string
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	am := &AlertManager{
		alerts:       make(map[string]*Alert),
		alertHistory: make([]AlertEvent, 0),
		emailConfig:  loadEmailConfig(),
		slackWebhook: getEnvOrDefault("SLACK_WEBHOOK", ""),
		webhookURL:   getEnvOrDefault("ALERT_WEBHOOK_URL", ""),
	}

	am.loadDefaultAlerts()
	return am
}

// loadDefaultAlerts loads the default alert configurations
func (am *AlertManager) loadDefaultAlerts() {
	defaultAlerts := []*Alert{
		{
			ID:          "high_error_rate",
			Name:        "High Error Rate",
			Description: "Error rate exceeds threshold",
			Level:       AlertCritical,
			Condition:   "error_rate > threshold",
			Threshold:   5.0, // 5%
			TimeWindow:  5 * time.Minute,
			Cooldown:    10 * time.Minute,
			Channels:    []string{"email", "log"},
			Enabled:     true,
			Labels:      map[string]string{"component": "api"},
		},
		{
			ID:          "circuit_breaker_open",
			Name:        "Circuit Breaker Open",
			Description: "Circuit breaker is in open state",
			Level:       AlertError,
			Condition:   "circuit_breaker_state == 2",
			Threshold:   2.0,
			TimeWindow:  1 * time.Minute,
			Cooldown:    5 * time.Minute,
			Channels:    []string{"email", "slack"},
			Enabled:     true,
			Labels:      map[string]string{"component": "resilience"},
		},
		{
			ID:          "high_response_time",
			Name:        "High Response Time",
			Description: "Average response time exceeds threshold",
			Level:       AlertWarning,
			Condition:   "avg_response_time > threshold",
			Threshold:   2000.0, // 2 seconds
			TimeWindow:  2 * time.Minute,
			Cooldown:    15 * time.Minute,
			Channels:    []string{"log"},
			Enabled:     true,
			Labels:      map[string]string{"component": "performance"},
		},
		{
			ID:          "database_connection_errors",
			Name:        "Database Connection Errors",
			Description: "Database connection failures detected",
			Level:       AlertCritical,
			Condition:   "db_connection_errors > threshold",
			Threshold:   3.0,
			TimeWindow:  5 * time.Minute,
			Cooldown:    10 * time.Minute,
			Channels:    []string{"email", "slack"},
			Enabled:     true,
			Labels:      map[string]string{"component": "database"},
		},
		{
			ID:          "memory_usage_high",
			Name:        "High Memory Usage",
			Description: "Memory usage exceeds safe threshold",
			Level:       AlertWarning,
			Condition:   "memory_usage_percent > threshold",
			Threshold:   85.0, // 85%
			TimeWindow:  1 * time.Minute,
			Cooldown:    30 * time.Minute,
			Channels:    []string{"email"},
			Enabled:     true,
			Labels:      map[string]string{"component": "system"},
		},
	}

	for _, alert := range defaultAlerts {
		am.AddAlert(alert)
	}
}

// AddAlert adds a new alert to the manager
func (am *AlertManager) AddAlert(alert *Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.alerts[alert.ID] = alert
}

// CheckAlert evaluates an alert condition
func (am *AlertManager) CheckAlert(alertID string, currentValue float64, labels map[string]string) {
	am.mu.RLock()
	alert, exists := am.alerts[alertID]
	am.mu.RUnlock()

	if !exists || !alert.Enabled {
		return
	}

	// Check cooldown period
	if alert.LastFired != nil && time.Since(*alert.LastFired) < alert.Cooldown {
		return
	}

	// Evaluate condition
	shouldFire := am.evaluateCondition(alert.Condition, currentValue, alert.Threshold)

	if shouldFire {
		event := AlertEvent{
			ID:        generateAlertEventID(),
			AlertID:   alertID,
			Level:     alert.Level,
			Message:   fmt.Sprintf("%s: %s (%.2f > %.2f)", alert.Name, alert.Description, currentValue, alert.Threshold),
			Value:     currentValue,
			Threshold: alert.Threshold,
			Timestamp: time.Now(),
			Labels:    labels,
		}

		am.fireAlert(alert, &event)
	}
}

// evaluateCondition evaluates the alert condition
func (am *AlertManager) evaluateCondition(condition string, value, threshold float64) bool {
	switch condition {
	case "error_rate > threshold":
		return value > threshold
	case "circuit_breaker_state == 2":
		return value == threshold // 2 = open state
	case "avg_response_time > threshold":
		return value > threshold
	case "db_connection_errors > threshold":
		return value > threshold
	case "memory_usage_percent > threshold":
		return value > threshold
	default:
		return false
	}
}

// fireAlert fires an alert through configured channels
func (am *AlertManager) fireAlert(alert *Alert, event *AlertEvent) {
	// Update last fired time
	now := time.Now()
	alert.LastFired = &now

	// Add to history
	am.mu.Lock()
	am.alertHistory = append(am.alertHistory, *event)
	// Keep only last 1000 events
	if len(am.alertHistory) > 1000 {
		am.alertHistory = am.alertHistory[len(am.alertHistory)-1000:]
	}
	am.mu.Unlock()

	// Log the alert
	Logger.Warn("Alert fired", map[string]interface{}{
		"alert_id":   alert.ID,
		"alert_name": alert.Name,
		"level":      alert.Level.String(),
		"value":      event.Value,
		"threshold":  event.Threshold,
		"message":    event.Message,
	})

	// Send through channels
	for _, channel := range alert.Channels {
		switch channel {
		case "email":
			am.sendEmailAlert(alert, event)
		case "slack":
			am.sendSlackAlert(alert, event)
		case "webhook":
			am.sendWebhookAlert(alert, event)
		case "log":
			// Already logged above
		}
	}
}

// sendEmailAlert sends an email alert
func (am *AlertManager) sendEmailAlert(alert *Alert, event *AlertEvent) {
	if am.emailConfig == nil {
		Logger.Warn("Email alert not sent - email config not available")
		return
	}

	subject := fmt.Sprintf("[%s] %s", alert.Level.String(), alert.Name)
	body := am.formatAlertMessage(alert, event)

	err := am.sendEmail(subject, body)
	if err != nil {
		Logger.Error("Failed to send email alert", err, map[string]interface{}{
			"alert_id": alert.ID,
			"subject":  subject,
		})
	}
}

// sendSlackAlert sends a Slack alert
func (am *AlertManager) sendSlackAlert(alert *Alert, event *AlertEvent) {
	if am.slackWebhook == "" {
		Logger.Warn("Slack alert not sent - webhook not configured")
		return
	}

	payload := map[string]interface{}{
		"text": fmt.Sprintf("*%s Alert*: %s", alert.Level.String(), alert.Name),
		"attachments": []map[string]interface{}{
			{
				"color": am.getSlackColor(alert.Level),
				"fields": []map[string]interface{}{
					{
						"title": "Description",
						"value": alert.Description,
						"short": false,
					},
					{
						"title": "Value",
						"value": fmt.Sprintf("%.2f", event.Value),
						"short": true,
					},
					{
						"title": "Threshold",
						"value": fmt.Sprintf("%.2f", event.Threshold),
						"short": true,
					},
					{
						"title": "Time",
						"value": event.Timestamp.Format(time.RFC3339),
						"short": true,
					},
				},
			},
		},
	}

	err := am.sendSlackMessage(payload)
	if err != nil {
		Logger.Error("Failed to send Slack alert", err, map[string]interface{}{
			"alert_id": alert.ID,
		})
	}
}

// sendWebhookAlert sends a webhook alert
func (am *AlertManager) sendWebhookAlert(alert *Alert, event *AlertEvent) {
	if am.webhookURL == "" {
		Logger.Warn("Webhook alert not sent - URL not configured")
		return
	}

	payload := map[string]interface{}{
		"alert_id":   alert.ID,
		"alert_name": alert.Name,
		"level":      alert.Level.String(),
		"message":    event.Message,
		"value":      event.Value,
		"threshold":  event.Threshold,
		"timestamp":  event.Timestamp,
		"labels":     event.Labels,
	}

	err := am.sendWebhookMessage(payload)
	if err != nil {
		Logger.Error("Failed to send webhook alert", err, map[string]interface{}{
			"alert_id": alert.ID,
		})
	}
}

// Helper functions

func (am *AlertManager) formatAlertMessage(alert *Alert, event *AlertEvent) string {
	return fmt.Sprintf(`Alert: %s

Description: %s
Level: %s
Value: %.2f
Threshold: %.2f
Time: %s

This is an automated alert from the Nutrition Platform monitoring system.
`, alert.Name, alert.Description, alert.Level.String(), event.Value, event.Threshold, event.Timestamp.Format(time.RFC3339))
}

func (am *AlertManager) getSlackColor(level AlertLevel) string {
	switch level {
	case AlertCritical:
		return "danger"
	case AlertError:
		return "danger"
	case AlertWarning:
		return "#ffa500"
	case AlertInfo:
		return "good"
	default:
		return "#808080"
	}
}

func (am *AlertManager) sendEmail(subject, body string) error {
	if am.emailConfig == nil {
		return fmt.Errorf("email config not available")
	}

	auth := smtp.PlainAuth("", am.emailConfig.Username, am.emailConfig.Password, am.emailConfig.SMTPHost)

	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s",
		strings.Join(am.emailConfig.ToAddresses, ","), subject, body))

	addr := fmt.Sprintf("%s:%d", am.emailConfig.SMTPHost, am.emailConfig.SMTPPort)
	err := smtp.SendMail(addr, auth, am.emailConfig.FromAddress, am.emailConfig.ToAddresses, msg)

	return err
}

func (am *AlertManager) sendSlackMessage(payload map[string]interface{}) error {
	// Implement Slack webhook sending
	// This would use HTTP POST to the webhook URL with JSON payload
	return fmt.Errorf("Slack integration not implemented - configure webhook")
}

func (am *AlertManager) sendWebhookMessage(payload map[string]interface{}) error {
	// Implement generic webhook sending
	// This would use HTTP POST to the configured URL with JSON payload
	return fmt.Errorf("Webhook integration not implemented - configure URL")
}

func generateAlertEventID() string {
	return fmt.Sprintf("alert-%d-%s", time.Now().UnixNano(), randomString(6))
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

func loadEmailConfig() *EmailConfig {
	host := getEnvOrDefault("SMTP_HOST", "")
	if host == "" {
		return nil
	}

	port, _ := strconv.Atoi(getEnvOrDefault("SMTP_PORT", "587"))
	username := getEnvOrDefault("SMTP_USERNAME", "")
	password := getEnvOrDefault("SMTP_PASSWORD", "")
	from := getEnvOrDefault("ALERT_FROM_EMAIL", "")
	to := strings.Split(getEnvOrDefault("ALERT_TO_EMAILS", ""), ",")

	return &EmailConfig{
		SMTPHost:    host,
		SMTPPort:    port,
		Username:    username,
		Password:    password,
		FromAddress: from,
		ToAddresses: to,
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Global alert manager instance
var AlertManagerInstance *AlertManager

// InitAlerts initializes the global alert manager
func InitAlerts() {
	AlertManagerInstance = NewAlertManager()
	Logger.Info("Alert system initialized", map[string]interface{}{
		"alert_count": len(AlertManagerInstance.alerts),
	})
}

// CheckErrorRate checks error rate and triggers alerts if needed
func CheckErrorRate(errorCount, totalRequests int) {
	if totalRequests == 0 {
		return
	}

	errorRate := float64(errorCount) / float64(totalRequests) * 100
	AlertManagerInstance.CheckAlert("high_error_rate", errorRate, map[string]string{
		"error_count":    strconv.Itoa(errorCount),
		"total_requests": strconv.Itoa(totalRequests),
	})
}

// CheckCircuitBreakerState checks circuit breaker state
func CheckCircuitBreakerState(serviceName string, state int) {
	AlertManagerInstance.CheckAlert("circuit_breaker_open", float64(state), map[string]string{
		"service": serviceName,
		"state":   strconv.Itoa(state),
	})
}

// CheckResponseTime checks average response time
func CheckResponseTime(avgResponseTime float64) {
	AlertManagerInstance.CheckAlert("high_response_time", avgResponseTime, nil)
}

// GetAlertHistory returns recent alert history
func GetAlertHistory(limit int) []AlertEvent {
	if limit <= 0 {
		limit = 50
	}

	AlertManagerInstance.mu.RLock()
	defer AlertManagerInstance.mu.RUnlock()

	history := AlertManagerInstance.alertHistory
	if len(history) <= limit {
		return history
	}

	return history[len(history)-limit:]
}
