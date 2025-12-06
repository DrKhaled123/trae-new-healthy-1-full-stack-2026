package security

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"nutrition-platform/errors"
)

// SecurityLogger handles security event logging
type SecurityLogger struct {
	logFile    *os.File
	logPath    string
	mutex      sync.RWMutex
	alertRules []AlertRule
	metrics    *SecurityMetrics
}

// AlertRule defines conditions for security alerts
type AlertRule struct {
	Name       string
	EventType  string
	Severity   string
	Threshold  int
	TimeWindow time.Duration
	Action     AlertAction
	Enabled    bool
}

// AlertAction defines what to do when an alert is triggered
type AlertAction struct {
	Type        string // "email", "webhook", "log", "block_ip"
	Target      string // email address, webhook URL, etc.
	Message     string
	AutoResolve bool
}

// SecurityMetrics tracks security-related metrics
type SecurityMetrics struct {
	mutex                 sync.RWMutex
	TotalEvents           int64
	EventsByType          map[string]int64
	EventsBySeverity      map[string]int64
	FailedAuthentications int64
	RateLimitViolations   int64
	SecurityViolations    int64
	BlockedIPs            map[string]time.Time
	SuspiciousIPs         map[string]int64
	LastReset             time.Time
}

// SecurityLogEntry represents a structured security log entry
type SecurityLogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	EventType string                 `json:"event_type"`
	Severity  string                 `json:"severity"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details"`
	Context   *SecurityContext       `json:"context"`
	Metadata  map[string]string      `json:"metadata"`
}

// SecurityContext provides request context for security events
type SecurityContext struct {
	IPAddress    string `json:"ip_address"`
	UserAgent    string `json:"user_agent"`
	APIKeyID     string `json:"api_key_id,omitempty"`
	Endpoint     string `json:"endpoint"`
	Method       string `json:"method"`
	StatusCode   int    `json:"status_code"`
	ResponseTime int64  `json:"response_time_ms"`
	RequestID    string `json:"request_id"`
}

// NewSecurityLogger creates a new security logger
func NewSecurityLogger(logPath string) (*SecurityLogger, error) {
	// Ensure log directory exists
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := &SecurityLogger{
		logFile:    logFile,
		logPath:    logPath,
		alertRules: getDefaultAlertRules(),
		metrics:    newSecurityMetrics(),
	}

	return logger, nil
}

// LogSecurityEvent logs a security event
func (sl *SecurityLogger) LogSecurityEvent(event *errors.SecurityEvent) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	// Update metrics
	sl.updateMetrics(event)

	// Create log entry
	logEntry := &SecurityLogEntry{
		Timestamp: event.Timestamp,
		Level:     "SECURITY",
		EventType: event.Type,
		Severity:  event.Severity,
		Message:   event.Message,
		Details:   convertDetailsToInterface(event.Details),
		Context: &SecurityContext{
			IPAddress:    event.IPAddress,
			UserAgent:    event.UserAgent,
			APIKeyID:     event.APIKeyID,
			Endpoint:     event.Endpoint,
			Method:       event.Method,
			StatusCode:   event.StatusCode,
			ResponseTime: event.ResponseTime.Milliseconds(),
		},
		Metadata: make(map[string]string),
	}

	// Add metadata
	logEntry.Metadata["source"] = "nutrition-platform"
	logEntry.Metadata["version"] = "1.0.0"
	logEntry.Metadata["environment"] = getEnvironment()

	// Write to log file
	if err := sl.writeLogEntry(logEntry); err != nil {
		log.Printf("Failed to write security log entry: %v", err)
	}

	// Check alert rules
	sl.checkAlertRules(event)

	// Handle high-severity events immediately
	if event.Severity == "critical" || event.Severity == "high" {
		sl.handleHighSeverityEvent(event)
	}
}

// writeLogEntry writes a log entry to the file
func (sl *SecurityLogger) writeLogEntry(entry *SecurityLogEntry) error {
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	_, err = sl.logFile.WriteString(string(jsonData) + "\n")
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return sl.logFile.Sync()
}

// updateMetrics updates security metrics
func (sl *SecurityLogger) updateMetrics(event *errors.SecurityEvent) {
	sl.metrics.mutex.Lock()
	defer sl.metrics.mutex.Unlock()

	sl.metrics.TotalEvents++
	sl.metrics.EventsByType[event.Type]++
	sl.metrics.EventsBySeverity[event.Severity]++

	// Update specific metrics
	switch event.Type {
	case "authentication_failure":
		sl.metrics.FailedAuthentications++
		sl.trackSuspiciousIP(event.IPAddress)
	case "rate_limit_exceeded":
		sl.metrics.RateLimitViolations++
		sl.trackSuspiciousIP(event.IPAddress)
	case "security_violation":
		sl.metrics.SecurityViolations++
		sl.trackSuspiciousIP(event.IPAddress)
	}
}

// trackSuspiciousIP tracks suspicious IP addresses
func (sl *SecurityLogger) trackSuspiciousIP(ip string) {
	if ip == "" {
		return
	}

	sl.metrics.SuspiciousIPs[ip]++

	// Auto-block IPs with too many violations
	if sl.metrics.SuspiciousIPs[ip] >= 10 {
		sl.metrics.BlockedIPs[ip] = time.Now()
		log.Printf("Auto-blocked suspicious IP: %s", ip)
	}
}

// checkAlertRules checks if any alert rules are triggered
func (sl *SecurityLogger) checkAlertRules(event *errors.SecurityEvent) {
	for _, rule := range sl.alertRules {
		if !rule.Enabled {
			continue
		}

		if sl.shouldTriggerAlert(rule, event) {
			sl.triggerAlert(rule, event)
		}
	}
}

// shouldTriggerAlert checks if an alert rule should be triggered
func (sl *SecurityLogger) shouldTriggerAlert(rule AlertRule, event *errors.SecurityEvent) bool {
	// Check event type match
	if rule.EventType != "" && rule.EventType != event.Type {
		return false
	}

	// Check severity match
	if rule.Severity != "" && rule.Severity != event.Severity {
		return false
	}

	// Check threshold within time window
	if rule.Threshold > 0 && rule.TimeWindow > 0 {
		// This would require a more sophisticated implementation
		// For now, we'll use a simple check
		return sl.getEventCountInWindow(rule.EventType, rule.TimeWindow) >= rule.Threshold
	}

	return true
}

// getEventCountInWindow gets the count of events in a time window
func (sl *SecurityLogger) getEventCountInWindow(eventType string, window time.Duration) int {
	// Simplified implementation - in production, this would query a time-series database
	sl.metrics.mutex.RLock()
	defer sl.metrics.mutex.RUnlock()

	// For now, return the total count for the event type
	return int(sl.metrics.EventsByType[eventType])
}

// triggerAlert triggers an alert action
func (sl *SecurityLogger) triggerAlert(rule AlertRule, event *errors.SecurityEvent) {
	alertMsg := fmt.Sprintf("Security Alert: %s - %s", rule.Name, event.Message)

	switch rule.Action.Type {
	case "log":
		log.Printf("SECURITY ALERT: %s", alertMsg)
	case "email":
		// In production, integrate with email service
		log.Printf("EMAIL ALERT to %s: %s", rule.Action.Target, alertMsg)
	case "webhook":
		// In production, send webhook
		log.Printf("WEBHOOK ALERT to %s: %s", rule.Action.Target, alertMsg)
	case "block_ip":
		if event.IPAddress != "" {
			sl.metrics.mutex.Lock()
			sl.metrics.BlockedIPs[event.IPAddress] = time.Now()
			sl.metrics.mutex.Unlock()
			log.Printf("BLOCKED IP: %s due to alert: %s", event.IPAddress, rule.Name)
		}
	}
}

// handleHighSeverityEvent handles critical and high severity events
func (sl *SecurityLogger) handleHighSeverityEvent(event *errors.SecurityEvent) {
	// Log to system log
	log.Printf("HIGH SEVERITY SECURITY EVENT: %s - %s (IP: %s)",
		event.Type, event.Message, event.IPAddress)

	// In production, you might want to:
	// - Send immediate notifications
	// - Trigger incident response
	// - Auto-block suspicious IPs
	// - Escalate to security team
}

// GetMetrics returns current security metrics
func (sl *SecurityLogger) GetMetrics() *SecurityMetrics {
	sl.metrics.mutex.RLock()
	defer sl.metrics.mutex.RUnlock()

	// Create a copy to avoid race conditions
	metrics := &SecurityMetrics{
		TotalEvents:           sl.metrics.TotalEvents,
		EventsByType:          make(map[string]int64),
		EventsBySeverity:      make(map[string]int64),
		FailedAuthentications: sl.metrics.FailedAuthentications,
		RateLimitViolations:   sl.metrics.RateLimitViolations,
		SecurityViolations:    sl.metrics.SecurityViolations,
		BlockedIPs:            make(map[string]time.Time),
		SuspiciousIPs:         make(map[string]int64),
		LastReset:             sl.metrics.LastReset,
	}

	// Copy maps
	for k, v := range sl.metrics.EventsByType {
		metrics.EventsByType[k] = v
	}
	for k, v := range sl.metrics.EventsBySeverity {
		metrics.EventsBySeverity[k] = v
	}
	for k, v := range sl.metrics.BlockedIPs {
		metrics.BlockedIPs[k] = v
	}
	for k, v := range sl.metrics.SuspiciousIPs {
		metrics.SuspiciousIPs[k] = v
	}

	return metrics
}

// IsIPBlocked checks if an IP is blocked
func (sl *SecurityLogger) IsIPBlocked(ip string) bool {
	sl.metrics.mutex.RLock()
	defer sl.metrics.mutex.RUnlock()

	blockTime, exists := sl.metrics.BlockedIPs[ip]
	if !exists {
		return false
	}

	// Check if block has expired (24 hours)
	if time.Since(blockTime) > 24*time.Hour {
		delete(sl.metrics.BlockedIPs, ip)
		return false
	}

	return true
}

// ResetMetrics resets security metrics
func (sl *SecurityLogger) ResetMetrics() {
	sl.metrics.mutex.Lock()
	defer sl.metrics.mutex.Unlock()

	sl.metrics.TotalEvents = 0
	sl.metrics.EventsByType = make(map[string]int64)
	sl.metrics.EventsBySeverity = make(map[string]int64)
	sl.metrics.FailedAuthentications = 0
	sl.metrics.RateLimitViolations = 0
	sl.metrics.SecurityViolations = 0
	sl.metrics.SuspiciousIPs = make(map[string]int64)
	sl.metrics.LastReset = time.Now()
	// Keep blocked IPs
}

// Close closes the security logger
func (sl *SecurityLogger) Close() error {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	if sl.logFile != nil {
		return sl.logFile.Close()
	}
	return nil
}

// Helper functions

func newSecurityMetrics() *SecurityMetrics {
	return &SecurityMetrics{
		EventsByType:     make(map[string]int64),
		EventsBySeverity: make(map[string]int64),
		BlockedIPs:       make(map[string]time.Time),
		SuspiciousIPs:    make(map[string]int64),
		LastReset:        time.Now(),
	}
}

func convertDetailsToInterface(details map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range details {
		result[k] = v
	}
	return result
}

func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		return "development"
	}
	return env
}

func getDefaultAlertRules() []AlertRule {
	return []AlertRule{
		{
			Name:       "High Authentication Failures",
			EventType:  "authentication_failure",
			Severity:   "",
			Threshold:  5,
			TimeWindow: 5 * time.Minute,
			Action: AlertAction{
				Type:    "log",
				Message: "Multiple authentication failures detected",
			},
			Enabled: true,
		},
		{
			Name:       "Critical Security Violation",
			EventType:  "security_violation",
			Severity:   "critical",
			Threshold:  1,
			TimeWindow: time.Minute,
			Action: AlertAction{
				Type:    "block_ip",
				Message: "Critical security violation - blocking IP",
			},
			Enabled: true,
		},
		{
			Name:       "Rate Limit Abuse",
			EventType:  "rate_limit_exceeded",
			Severity:   "",
			Threshold:  10,
			TimeWindow: time.Minute,
			Action: AlertAction{
				Type:    "log",
				Message: "Potential rate limit abuse detected",
			},
			Enabled: true,
		},
	}
}
