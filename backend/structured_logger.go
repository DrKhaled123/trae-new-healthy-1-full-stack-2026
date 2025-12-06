// structured_logger.go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// LogLevel represents different logging levels
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of LogLevel
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// StructuredLogger represents a structured logger
type StructuredLogger struct {
	level  LogLevel
	writer io.Writer
	config *LogConfig
	fields map[string]interface{}
}

// LogConfig holds configuration for the structured logger
type LogConfig struct {
	Level           LogLevel
	EnableJSON      bool
	EnableCaller    bool
	EnableTimestamp bool
	TimeFormat      string
	ServiceName     string
	Environment     string
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp     time.Time              `json:"timestamp"`
	Level         string                 `json:"level"`
	Service       string                 `json:"service,omitempty"`
	Environment   string                 `json:"environment,omitempty"`
	Message       string                 `json:"message"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
	Caller        *CallerInfo            `json:"caller,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
}

// CallerInfo holds caller information
type CallerInfo struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config *LogConfig) *StructuredLogger {
	if config == nil {
		config = &LogConfig{
			Level:           INFO,
			EnableJSON:      true,
			EnableCaller:    true,
			EnableTimestamp: true,
			TimeFormat:      time.RFC3339,
			ServiceName:     "nutrition-platform",
			Environment:     getEnvironment(),
		}
	}

	writer := os.Stdout
	if logFile := os.Getenv("LOG_FILE"); logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			// Log error to stderr but continue with stdout logging
			log.Printf("Failed to open log file %s: %v", logFile, err)
		} else {
			writer = file
		}
	}

	return &StructuredLogger{
		level:  config.Level,
		writer: writer,
		config: config,
		fields: make(map[string]interface{}),
	}
}

// getEnvironment gets the current environment
func getEnvironment() string {
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		return env
	}
	return "development"
}

// WithField adds a field to the logger
func (l *StructuredLogger) WithField(key string, value interface{}) *StructuredLogger {
	newLogger := &StructuredLogger{
		level:  l.level,
		writer: l.writer,
		config: l.config,
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	newLogger.fields[key] = value
	return newLogger
}

// WithFields adds multiple fields to the logger
func (l *StructuredLogger) WithFields(fields map[string]interface{}) *StructuredLogger {
	newLogger := &StructuredLogger{
		level:  l.level,
		writer: l.writer,
		config: l.config,
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// WithContext adds context from echo.Context
func (l *StructuredLogger) WithContext(c echo.Context) *StructuredLogger {
	fields := make(map[string]interface{})

	if c != nil {
		// Add request ID if available
		if requestID := c.Response().Header().Get(echo.HeaderXRequestID); requestID != "" {
			fields["request_id"] = requestID
		}

		// Add correlation ID if available
		if correlationID := c.Request().Header.Get("X-Correlation-ID"); correlationID != "" {
			fields["correlation_id"] = correlationID
		}

		// Add user ID if available (from context or header)
		if userID := c.Get("userId"); userID != nil {
			if str, ok := userID.(string); ok && str != "" {
				fields["user_id"] = maskSensitiveData(str, "user_id")
			}
		}

		// Add API key (masked)
		if apiKey := c.Request().Header.Get("X-API-Key"); apiKey != "" {
			fields["api_key_prefix"] = maskAPIKey(apiKey)
		}

		// Add request info
		fields["method"] = c.Request().Method
		fields["path"] = c.Request().URL.Path
		fields["remote_ip"] = c.RealIP()
		fields["user_agent"] = maskUserAgent(c.Request().UserAgent())
	}

	return l.WithFields(fields)
}

// log writes a log entry
func (l *StructuredLogger) log(level LogLevel, message string, additionalFields map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp:   time.Now().UTC(),
		Level:       level.String(),
		Service:     l.config.ServiceName,
		Environment: l.config.Environment,
		Message:     message,
		Fields:      make(map[string]interface{}),
	}

	// Add configured fields
	for k, v := range l.fields {
		entry.Fields[k] = v
	}

	// Add additional fields
	for k, v := range additionalFields {
		entry.Fields[k] = v
	}

	// Add caller info if enabled
	if l.config.EnableCaller {
		entry.Caller = l.getCallerInfo()
	}

	// Add correlation/user IDs from fields if present
	if correlationID, ok := entry.Fields["correlation_id"].(string); ok {
		entry.CorrelationID = correlationID
	}
	if userID, ok := entry.Fields["user_id"].(string); ok {
		entry.UserID = userID
	}
	if requestID, ok := entry.Fields["request_id"].(string); ok {
		entry.RequestID = requestID
	}

	var output string
	if l.config.EnableJSON {
		jsonData, err := json.Marshal(entry)
		if err != nil {
			// Fallback to simple format if JSON marshaling fails
			output = fmt.Sprintf("[%s] %s: %s", entry.Timestamp.Format(l.config.TimeFormat), entry.Level, entry.Message)
		} else {
			output = string(jsonData)
		}
	} else {
		output = fmt.Sprintf("[%s] %s: %s", entry.Timestamp.Format(l.config.TimeFormat), entry.Level, entry.Message)
		if len(entry.Fields) > 0 {
			output += fmt.Sprintf(" fields=%v", entry.Fields)
		}
	}

	fmt.Fprintln(l.writer, output)

	// Handle fatal level
	if level == FATAL {
		os.Exit(1)
	}
}

// getCallerInfo gets caller information
func (l *StructuredLogger) getCallerInfo() *CallerInfo {
	_, file, line, ok := runtime.Caller(3) // Skip 3 levels to get actual caller
	if !ok {
		return nil
	}

	// Get function name
	pc, _, _, ok := runtime.Caller(3)
	if !ok {
		return &CallerInfo{File: file, Line: line}
	}

	fn := runtime.FuncForPC(pc)
	function := fn.Name()

	// Clean up file path
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		file = file[idx+1:]
	}

	return &CallerInfo{
		Function: function,
		File:     file,
		Line:     line,
	}
}

// Debug logs a debug message
func (l *StructuredLogger) Debug(message string, fields ...map[string]interface{}) {
	var additionalFields map[string]interface{}
	if len(fields) > 0 {
		additionalFields = fields[0]
	}
	l.log(DEBUG, message, additionalFields)
}

// Info logs an info message
func (l *StructuredLogger) Info(message string, fields ...map[string]interface{}) {
	var additionalFields map[string]interface{}
	if len(fields) > 0 {
		additionalFields = fields[0]
	}
	l.log(INFO, message, additionalFields)
}

// Warn logs a warning message
func (l *StructuredLogger) Warn(message string, fields ...map[string]interface{}) {
	var additionalFields map[string]interface{}
	if len(fields) > 0 {
		additionalFields = fields[0]
	}
	l.log(WARN, message, additionalFields)
}

// Error logs an error message
func (l *StructuredLogger) Error(message string, err error, fields ...map[string]interface{}) {
	var additionalFields map[string]interface{}
	if len(fields) > 0 {
		additionalFields = fields[0]
	} else {
		additionalFields = make(map[string]interface{})
	}

	// Add error information if provided
	if err != nil {
		additionalFields["error"] = err.Error()
	}

	l.log(ERROR, message, additionalFields)
}

// Fatal logs a fatal message and exits
func (l *StructuredLogger) Fatal(message string, fields ...map[string]interface{}) {
	var additionalFields map[string]interface{}
	if len(fields) > 0 {
		additionalFields = fields[0]
	}
	l.log(FATAL, message, additionalFields)
}

// maskSensitiveData masks sensitive data in logs
func maskSensitiveData(data, fieldType string) string {
	if data == "" {
		return data
	}

	switch fieldType {
	case "api_key", "password", "token":
		if len(data) <= 8 {
			return strings.Repeat("*", len(data))
		}
		return data[:4] + strings.Repeat("*", len(data)-8) + data[len(data)-4:]
	case "email":
		if idx := strings.Index(data, "@"); idx > 0 {
			return data[:1] + strings.Repeat("*", idx-1) + data[idx:]
		}
	default:
		return data
	}
	return data
}

// maskAPIKey masks API key for logging
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return strings.Repeat("*", len(apiKey))
	}
	return apiKey[:8] + "..."
}

// maskUserAgent masks sensitive information in user agent
func maskUserAgent(userAgent string) string {
	// Mask any tokens or sensitive data in user agent
	// This is a simple implementation - in production you might want more sophisticated masking
	if strings.Contains(userAgent, "token=") || strings.Contains(userAgent, "key=") {
		return "Masked User Agent"
	}
	return userAgent
}

// Global logger instance
var Logger *StructuredLogger

// InitStructuredLogger initializes the global structured logger
func InitStructuredLogger() error {
	config := &LogConfig{
		Level:           parseLogLevel(os.Getenv("LOG_LEVEL")),
		EnableJSON:      os.Getenv("LOG_FORMAT") != "text",
		EnableCaller:    os.Getenv("LOG_CALLER") == "true",
		EnableTimestamp: true,
		TimeFormat:      time.RFC3339,
		ServiceName:     "nutrition-platform",
		Environment:     getEnvironment(),
	}

	Logger = NewStructuredLogger(config)

	// Log initialization
	Logger.Info("Structured logging initialized", map[string]interface{}{
		"level":       config.Level.String(),
		"format":      map[bool]string{true: "json", false: "text"}[config.EnableJSON],
		"environment": config.Environment,
	})

	return nil
}

// parseLogLevel parses log level from string
func parseLogLevel(levelStr string) LogLevel {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}

// RequestLoggerMiddleware logs HTTP requests
func StructuredRequestLoggerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			path := c.Request().URL.Path
			method := c.Request().Method

			// Process request
			err := next(c)

			// Calculate latency
			latency := time.Since(start)
			statusCode := c.Response().Status

			// Prepare log fields
			fields := map[string]interface{}{
				"method":      method,
				"path":        path,
				"status_code": statusCode,
				"latency_ms":  latency.Milliseconds(),
				"remote_ip":   c.RealIP(),
			}

			// Add user ID if available
			if userID := c.Get("userId"); userID != nil {
				if str, ok := userID.(string); ok && str != "" {
					fields["user_id"] = str
				}
			}

			// Add API key prefix if available
			if apiKey := c.Request().Header.Get("X-API-Key"); apiKey != "" {
				fields["api_key_prefix"] = maskAPIKey(apiKey)
			}

			// Log based on status code
			logger := Logger.WithContext(c)

			if statusCode >= 500 {
				logger.Error("Request failed", err, fields)
			} else if statusCode >= 400 {
				logger.Warn("Request warning", fields)
			} else if Logger.level <= INFO {
				logger.Info("Request completed", fields)
			}

			return err
		}
	}
}

// CloseStructuredLogger closes the logger
func CloseStructuredLogger() {
	if Logger != nil && Logger.writer != os.Stdout {
		if closer, ok := Logger.writer.(io.Closer); ok {
			closer.Close()
		}
	}
}
