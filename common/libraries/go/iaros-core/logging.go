package logging

import (
	"context"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.Logger with IAROS-specific functionality
type Logger struct {
	*zap.Logger
	serviceName string
	version     string
	environment string
}

// Config holds configuration for the logger
type Config struct {
	Level       string
	ServiceName string
	Version     string
	Environment string
	OutputPath  string
	Format      string // json or console
	EnableCaller bool
	EnableStacktrace bool
}

// RequestIDKey is the context key for request ID
const RequestIDKey = "request_id"

// NewIAROSLogger creates a new IAROS logger with the specified service name
func NewIAROSLogger(serviceName string, opts ...Config) *Logger {
	config := Config{
		Level:       "info",
		ServiceName: serviceName,
		Version:     "1.0.0",
		Environment: getEnv("IAROS_ENV", "development"),
		OutputPath:  "stdout",
		Format:      "json",
		EnableCaller: true,
		EnableStacktrace: true,
	}

	// Apply optional configuration
	if len(opts) > 0 {
		if opts[0].Level != "" {
			config.Level = opts[0].Level
		}
		if opts[0].ServiceName != "" {
			config.ServiceName = opts[0].ServiceName
		}
		if opts[0].Version != "" {
			config.Version = opts[0].Version
		}
		if opts[0].Environment != "" {
			config.Environment = opts[0].Environment
		}
		if opts[0].OutputPath != "" {
			config.OutputPath = opts[0].OutputPath
		}
		if opts[0].Format != "" {
			config.Format = opts[0].Format
		}
		config.EnableCaller = opts[0].EnableCaller
		config.EnableStacktrace = opts[0].EnableStacktrace
	}

	// Parse log level
	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create encoder
	var encoder zapcore.Encoder
	if config.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create writer syncer
	var writeSyncer zapcore.WriteSyncer
	if config.OutputPath == "stdout" {
		writeSyncer = zapcore.AddSync(os.Stdout)
	} else {
		file, err := os.OpenFile(config.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			writeSyncer = zapcore.AddSync(os.Stdout)
		} else {
			writeSyncer = zapcore.AddSync(file)
		}
	}

	// Create core
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// Create logger options
	opts_zap := []zap.Option{}
	if config.EnableCaller {
		opts_zap = append(opts_zap, zap.AddCaller())
	}
	if config.EnableStacktrace {
		opts_zap = append(opts_zap, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	// Create base logger
	baseLogger := zap.New(core, opts_zap...)

	// Add service fields
	baseLogger = baseLogger.With(
		zap.String("service", config.ServiceName),
		zap.String("version", config.Version),
		zap.String("environment", config.Environment),
	)

	return &Logger{
		Logger:      baseLogger,
		serviceName: config.ServiceName,
		version:     config.Version,
		environment: config.Environment,
	}
}

// WithRequestID adds request ID to logger context
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger:      l.Logger.With(zap.String("request_id", requestID)),
		serviceName: l.serviceName,
		version:     l.version,
		environment: l.environment,
	}
}

// WithContext extracts request ID from context and adds it to logger
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return l.WithRequestID(requestID)
	}
	return l
}

// WithFields adds multiple fields to logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
	}
	
	return &Logger{
		Logger:      l.Logger.With(zapFields...),
		serviceName: l.serviceName,
		version:     l.version,
		environment: l.environment,
	}
}

// WithError adds error information to logger context
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger:      l.Logger.With(zap.Error(err)),
		serviceName: l.serviceName,
		version:     l.version,
		environment: l.environment,
	}
}

// WithUserID adds user ID to logger context
func (l *Logger) WithUserID(userID string) *Logger {
	return &Logger{
		Logger:      l.Logger.With(zap.String("user_id", userID)),
		serviceName: l.serviceName,
		version:     l.version,
		environment: l.environment,
	}
}

// WithTransaction adds transaction ID to logger context
func (l *Logger) WithTransaction(transactionID string) *Logger {
	return &Logger{
		Logger:      l.Logger.With(zap.String("transaction_id", transactionID)),
		serviceName: l.serviceName,
		version:     l.version,
		environment: l.environment,
	}
}

// HTTPRequestLogger logs HTTP request details
func (l *Logger) HTTPRequestLogger(method, path, userAgent, remoteAddr string, duration time.Duration, statusCode int) {
	l.Info("HTTP request",
		zap.String("method", method),
		zap.String("path", path),
		zap.String("user_agent", userAgent),
		zap.String("remote_addr", remoteAddr),
		zap.Duration("duration", duration),
		zap.Int("status_code", statusCode),
	)
}

// DatabaseQueryLogger logs database query details
func (l *Logger) DatabaseQueryLogger(query string, duration time.Duration, rowsAffected int64) {
	l.Debug("Database query",
		zap.String("query", query),
		zap.Duration("duration", duration),
		zap.Int64("rows_affected", rowsAffected),
	)
}

// ExternalServiceLogger logs external service call details
func (l *Logger) ExternalServiceLogger(service, method, endpoint string, duration time.Duration, statusCode int, success bool) {
	level := l.Info
	if !success {
		level = l.Error
	}

	level("External service call",
		zap.String("external_service", service),
		zap.String("method", method),
		zap.String("endpoint", endpoint),
		zap.Duration("duration", duration),
		zap.Int("status_code", statusCode),
		zap.Bool("success", success),
	)
}

// BusinessEventLogger logs business events
func (l *Logger) BusinessEventLogger(eventType, eventID string, data map[string]interface{}) {
	fields := []zap.Field{
		zap.String("event_type", eventType),
		zap.String("event_id", eventID),
		zap.Time("event_time", time.Now()),
	}

	for key, value := range data {
		fields = append(fields, zap.Any(key, value))
	}

	l.Info("Business event", fields...)
}

// SecurityEventLogger logs security-related events
func (l *Logger) SecurityEventLogger(eventType, userID, resource, action string, success bool, reason string) {
	level := l.Info
	if !success {
		level = l.Warn
	}

	level("Security event",
		zap.String("event_type", eventType),
		zap.String("user_id", userID),
		zap.String("resource", resource),
		zap.String("action", action),
		zap.Bool("success", success),
		zap.String("reason", reason),
	)
}

// AuditLogger logs audit events
func (l *Logger) AuditLogger(action, resource, userID string, before, after interface{}) {
	l.Info("Audit event",
		zap.String("action", action),
		zap.String("resource", resource),
		zap.String("user_id", userID),
		zap.Any("before", before),
		zap.Any("after", after),
		zap.Time("audit_time", time.Now()),
	)
}

// PerformanceLogger logs performance metrics
func (l *Logger) PerformanceLogger(operation string, duration time.Duration, metadata map[string]interface{}) {
	fields := []zap.Field{
		zap.String("operation", operation),
		zap.Duration("duration", duration),
		zap.Time("measured_at", time.Now()),
	}

	for key, value := range metadata {
		fields = append(fields, zap.Any(key, value))
	}

	l.Debug("Performance metric", fields...)
}

// CacheLogger logs cache operations
func (l *Logger) CacheLogger(operation, key string, hit bool, duration time.Duration) {
	l.Debug("Cache operation",
		zap.String("operation", operation),
		zap.String("key", key),
		zap.Bool("hit", hit),
		zap.Duration("duration", duration),
	)
}

// MetricsLogger logs metrics for monitoring systems
func (l *Logger) MetricsLogger(metricName string, value float64, tags map[string]string) {
	fields := []zap.Field{
		zap.String("metric_name", metricName),
		zap.Float64("value", value),
		zap.Time("timestamp", time.Now()),
	}

	for key, value := range tags {
		fields = append(fields, zap.String(key, value))
	}

	l.Info("Metric", fields...)
}

// AlertLogger logs alert events
func (l *Logger) AlertLogger(alertType, severity, message string, metadata map[string]interface{}) {
	level := l.Info
	switch strings.ToLower(severity) {
	case "critical", "error":
		level = l.Error
	case "warning", "warn":
		level = l.Warn
	}

	fields := []zap.Field{
		zap.String("alert_type", alertType),
		zap.String("severity", severity),
		zap.String("alert_message", message),
		zap.Time("alert_time", time.Now()),
	}

	for key, value := range metadata {
		fields = append(fields, zap.Any(key, value))
	}

	level("Alert triggered", fields...)
}

// Structured logging methods with common fields

// InfoWithFields logs info message with structured fields
func (l *Logger) InfoWithFields(msg string, fields map[string]interface{}) {
	l.WithFields(fields).Info(msg)
}

// WarnWithFields logs warning message with structured fields
func (l *Logger) WarnWithFields(msg string, fields map[string]interface{}) {
	l.WithFields(fields).Warn(msg)
}

// ErrorWithFields logs error message with structured fields
func (l *Logger) ErrorWithFields(msg string, fields map[string]interface{}) {
	l.WithFields(fields).Error(msg)
}

// FatalWithFields logs fatal message with structured fields
func (l *Logger) FatalWithFields(msg string, fields map[string]interface{}) {
	l.WithFields(fields).Fatal(msg)
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// Close closes the logger and flushes any buffered log entries
func (l *Logger) Close() error {
	return l.Sync()
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level string) *Logger {
	// Note: In a real implementation, you would need to recreate the logger
	// with the new level, as zap doesn't support dynamic level changes
	// This is a simplified version for demonstration
	return l
}

// GetServiceName returns the service name
func (l *Logger) GetServiceName() string {
	return l.serviceName
}

// GetVersion returns the service version
func (l *Logger) GetVersion() string {
	return l.version
}

// GetEnvironment returns the environment
func (l *Logger) GetEnvironment() string {
	return l.environment
}

// Helper functions

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Global logger instance
var globalLogger *Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(serviceName string, opts ...Config) {
	globalLogger = NewIAROSLogger(serviceName, opts...)
}

// GetGlobalLogger returns the global logger
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		globalLogger = NewIAROSLogger("iaros-service")
	}
	return globalLogger
}

// Global logging functions

// Info logs an info message using the global logger
func Info(msg string, fields ...zap.Field) {
	GetGlobalLogger().Info(msg, fields...)
}

// Warn logs a warning message using the global logger
func Warn(msg string, fields ...zap.Field) {
	GetGlobalLogger().Warn(msg, fields...)
}

// Error logs an error message using the global logger
func Error(msg string, fields ...zap.Field) {
	GetGlobalLogger().Error(msg, fields...)
}

// Fatal logs a fatal message using the global logger
func Fatal(msg string, fields ...zap.Field) {
	GetGlobalLogger().Fatal(msg, fields...)
}

// Debug logs a debug message using the global logger
func Debug(msg string, fields ...zap.Field) {
	GetGlobalLogger().Debug(msg, fields...)
} 