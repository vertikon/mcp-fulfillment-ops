// Package observability provides structured logging capabilities
package observability

import (
	"context"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// LogLevel represents log level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// StructuredLogger provides structured logging with context support
type StructuredLogger struct {
	logger *zap.Logger
	mu     sync.RWMutex
	level  LogLevel
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields"`
	TraceID   string                 `json:"trace_id,omitempty"`
	SpanID    string                 `json:"span_id,omitempty"`
	Service   string                 `json:"service,omitempty"`
	Component string                 `json:"component,omitempty"`
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level       LogLevel `json:"level"`
	Format      string   `json:"format"` // json, text
	Output      string   `json:"output"` // stdout, file, syslog
	Service     string   `json:"service"`
	Component   string   `json:"component"`
	EnableTrace bool     `json:"enable_trace"`
}

// DefaultLogConfig returns default logging configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:       LogLevelInfo,
		Format:      "json",
		Output:      "stdout",
		Service:     "mcp-fulfillment-ops",
		Component:   "monitoring",
		EnableTrace: true,
	}
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config *LogConfig) (*StructuredLogger, error) {
	if config == nil {
		config = DefaultLogConfig()
	}

	// Initialize logger using pkg/logger
	if err := logger.Init(string(config.Level), false); err != nil {
		return nil, err
	}

	log := logger.Get()
	if log == nil {
		log = zap.NewNop()
	}

	return &StructuredLogger{
		logger: log,
		level:  config.Level,
	}, nil
}

// Debug logs a debug message
func (sl *StructuredLogger) Debug(ctx context.Context, message string, fields ...zap.Field) {
	if sl.shouldLog(LogLevelDebug) {
		logger := sl.getLoggerWithContext(ctx)
		logger.Debug(message, fields...)
	}
}

// Info logs an info message
func (sl *StructuredLogger) Info(ctx context.Context, message string, fields ...zap.Field) {
	if sl.shouldLog(LogLevelInfo) {
		logger := sl.getLoggerWithContext(ctx)
		logger.Info(message, fields...)
	}
}

// Warn logs a warning message
func (sl *StructuredLogger) Warn(ctx context.Context, message string, fields ...zap.Field) {
	if sl.shouldLog(LogLevelWarn) {
		logger := sl.getLoggerWithContext(ctx)
		logger.Warn(message, fields...)
	}
}

// Error logs an error message
func (sl *StructuredLogger) Error(ctx context.Context, message string, fields ...zap.Field) {
	if sl.shouldLog(LogLevelError) {
		logger := sl.getLoggerWithContext(ctx)
		logger.Error(message, fields...)
	}
}

// Fatal logs a fatal message and exits
func (sl *StructuredLogger) Fatal(ctx context.Context, message string, fields ...zap.Field) {
	if sl.shouldLog(LogLevelFatal) {
		logger := sl.getLoggerWithContext(ctx)
		logger.Fatal(message, fields...)
	}
}

// LogEntry logs a structured log entry
func (sl *StructuredLogger) LogEntry(ctx context.Context, entry LogEntry) {
	logger := sl.getLoggerWithContext(ctx)

	fields := []zap.Field{
		zap.String("level", string(entry.Level)),
		zap.Time("timestamp", entry.Timestamp),
	}

	if entry.TraceID != "" {
		fields = append(fields, zap.String("trace_id", entry.TraceID))
	}
	if entry.SpanID != "" {
		fields = append(fields, zap.String("span_id", entry.SpanID))
	}
	if entry.Service != "" {
		fields = append(fields, zap.String("service", entry.Service))
	}
	if entry.Component != "" {
		fields = append(fields, zap.String("component", entry.Component))
	}

	for k, v := range entry.Fields {
		fields = append(fields, zap.Any(k, v))
	}

	switch entry.Level {
	case LogLevelDebug:
		logger.Debug(entry.Message, fields...)
	case LogLevelInfo:
		logger.Info(entry.Message, fields...)
	case LogLevelWarn:
		logger.Warn(entry.Message, fields...)
	case LogLevelError:
		logger.Error(entry.Message, fields...)
	case LogLevelFatal:
		logger.Fatal(entry.Message, fields...)
	}
}

// SetLevel sets the log level
func (sl *StructuredLogger) SetLevel(level LogLevel) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.level = level
}

// GetLevel returns the current log level
func (sl *StructuredLogger) GetLevel() LogLevel {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.level
}

// shouldLog checks if a log level should be logged
func (sl *StructuredLogger) shouldLog(level LogLevel) bool {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	levels := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
		LogLevelFatal: 4,
	}

	currentLevel := levels[sl.level]
	requestedLevel := levels[level]

	return requestedLevel >= currentLevel
}

// getLoggerWithContext returns logger with context fields
func (sl *StructuredLogger) getLoggerWithContext(ctx context.Context) *zap.Logger {
	return logger.WithContext(ctx)
}

// Sync flushes any buffered log entries
func (sl *StructuredLogger) Sync() error {
	if sl.logger != nil {
		return sl.logger.Sync()
	}
	return nil
}
