package logger

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var globalLogger *zap.Logger

// Initialize sets up the global logger with the specified log level
func Initialize(logLevel string) error {
	level, err := parseLogLevel(logLevel)
	if err != nil {
		return err
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level)
	config.Encoding = "json"
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.StacktraceKey = "stacktrace"

	logger, err := config.Build(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		return err
	}

	globalLogger = logger
	return nil
}

// parseLogLevel converts string log level to zapcore.Level
func parseLogLevel(logLevel string) (zapcore.Level, error) {
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		return zapcore.DebugLevel, nil
	case "INFO":
		return zapcore.InfoLevel, nil
	case "WARN", "WARNING":
		return zapcore.WarnLevel, nil
	case "ERROR":
		return zapcore.ErrorLevel, nil
	case "FATAL":
		return zapcore.FatalLevel, nil
	default:
		return zapcore.InfoLevel, nil
	}
}

// Get returns the global logger instance
func Get() *zap.Logger {
	if globalLogger == nil {
		// Fallback to a default logger if not initialized
		globalLogger, _ = zap.NewProduction()
	}
	return globalLogger
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// Sync flushes any buffered log entries
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// With creates a child logger with additional fields
func With(fields ...zap.Field) *zap.Logger {
	return Get().With(fields...)
}

// Named creates a named logger
func Named(name string) *zap.Logger {
	return Get().Named(name)
}
