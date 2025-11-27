package logger

import (
	"context"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap logger with additional functionality
type Logger struct {
	*zap.Logger
	sugar *zap.SugaredLogger
}

// Config holds logger configuration
type Config struct {
	Level      string // debug, info, warn, error
	Format     string // json, console
	Output     io.Writer
	EnableCaller bool
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:        "info",
		Format:       "console",
		Output:       os.Stdout,
		EnableCaller: true,
	}
}

// New creates a new logger with the given configuration
func New(cfg *Config) (*Logger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Parse level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	// Configure encoder
	var encoderConfig zapcore.EncoderConfig
	if cfg.Format == "json" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Create encoder
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Create core
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(cfg.Output),
		level,
	)

	// Build logger
	zapLogger := zap.New(core)
	
	if cfg.EnableCaller {
		zapLogger = zapLogger.WithOptions(zap.AddCaller(), zap.AddCallerSkip(1))
	}

	return &Logger{
		Logger: zapLogger,
		sugar:  zapLogger.Sugar(),
	}, nil
}

// NewDevelopment creates a development logger (console format, debug level)
func NewDevelopment() (*Logger, error) {
	cfg := &Config{
		Level:        "debug",
		Format:       "console",
		Output:       os.Stdout,
		EnableCaller: true,
	}
	return New(cfg)
}

// NewProduction creates a production logger (JSON format, info level)
func NewProduction() (*Logger, error) {
	cfg := &Config{
		Level:        "info",
		Format:       "json",
		Output:       os.Stdout,
		EnableCaller: false,
	}
	return New(cfg)
}

// WithContext adds context fields to the logger
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract common context values if present
	// This can be extended to extract trace IDs, request IDs, etc.
	return l
}

// WithFields adds structured fields to the logger
func (l *Logger) WithFields(fields ...zap.Field) *Logger {
	newLogger := l.Logger.With(fields...)
	return &Logger{
		Logger: newLogger,
		sugar:  newLogger.Sugar(),
	}
}

// WithError adds error field to the logger
func (l *Logger) WithError(err error) *Logger {
	return l.WithFields(zap.Error(err))
}

// Sugar returns the sugared logger for Printf-style logging
func (l *Logger) Sugar() *zap.SugaredLogger {
	return l.sugar
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	_ = l.sugar.Sync()
	return l.Logger.Sync()
}

// Global logger instance
var global *Logger

func init() {
	// Initialize with development logger by default
	l, _ := NewDevelopment()
	global = l
}

// Global returns the global logger instance
func Global() *Logger {
	return global
}

// SetGlobal sets the global logger instance
func SetGlobal(l *Logger) {
	global = l
}

// Convenience functions for global logger

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	global.Logger.Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	global.Logger.Info(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	global.Logger.Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	global.Logger.Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	global.Logger.Fatal(msg, fields...)
}

// Debugf logs a debug message with Printf-style formatting
func Debugf(template string, args ...interface{}) {
	global.sugar.Debugf(template, args...)
}

// Infof logs an info message with Printf-style formatting
func Infof(template string, args ...interface{}) {
	global.sugar.Infof(template, args...)
}

// Warnf logs a warning message with Printf-style formatting
func Warnf(template string, args ...interface{}) {
	global.sugar.Warnf(template, args...)
}

// Errorf logs an error message with Printf-style formatting
func Errorf(template string, args ...interface{}) {
	global.sugar.Errorf(template, args...)
}

// Fatalf logs a fatal message with Printf-style formatting and exits
func Fatalf(template string, args ...interface{}) {
	global.sugar.Fatalf(template, args...)
}

