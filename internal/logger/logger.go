package logger

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap logger with additional functionality
type Logger struct {
	*zap.Logger
	sugar    *zap.SugaredLogger
	logFile  *os.File // Keep reference to close on cleanup
}

// Config holds logger configuration
type Config struct {
	Level        string    // debug, info, warn, error
	Format       string    // json, console
	Output       io.Writer // Custom output (if set, overrides File)
	File         string    // Log file path (e.g., "panka.log")
	EnableCaller bool
	EnableFile   bool      // Whether to write to file
	EnableStdout bool      // Whether to also write to stdout
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:        "info",
		Format:       "console",
		Output:       nil,
		File:         "panka.log",
		EnableCaller: true,
		EnableFile:   false,
		EnableStdout: true,
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

	// Configure encoder for console (with colors)
	consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
	consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Configure encoder for file (no colors, cleaner format)
	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// Create encoders
	var consoleEncoder, fileEncoder zapcore.Encoder
	if cfg.Format == "json" {
		consoleEncoder = zapcore.NewJSONEncoder(consoleEncoderConfig)
		fileEncoder = zapcore.NewJSONEncoder(fileEncoderConfig)
	} else {
		consoleEncoder = zapcore.NewConsoleEncoder(consoleEncoderConfig)
		fileEncoder = zapcore.NewConsoleEncoder(fileEncoderConfig)
	}

	// Build cores based on configuration
	var cores []zapcore.Core
	var logFile *os.File

	// If custom output is provided, use it
	if cfg.Output != nil {
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(cfg.Output), level))
	} else {
		// Add stdout core if enabled
		if cfg.EnableStdout {
			cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level))
		}

		// Add file core if enabled
		if cfg.EnableFile && cfg.File != "" {
			// Resolve file path
			filePath := cfg.File
			if !filepath.IsAbs(filePath) {
				// Use current working directory
				cwd, err := os.Getwd()
				if err == nil {
					filePath = filepath.Join(cwd, filePath)
				}
			}

			// Open or create log file
			logFile, err = os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return nil, err
			}

			cores = append(cores, zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), level))
		}
	}

	// If no cores, default to stdout
	if len(cores) == 0 {
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level))
	}

	// Create tee core to write to multiple outputs
	core := zapcore.NewTee(cores...)

	// Build logger
	zapLogger := zap.New(core)

	if cfg.EnableCaller {
		zapLogger = zapLogger.WithOptions(zap.AddCaller(), zap.AddCallerSkip(1))
	}

	return &Logger{
		Logger:  zapLogger,
		sugar:   zapLogger.Sugar(),
		logFile: logFile,
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

// Close closes the logger and any open file handles
func (l *Logger) Close() error {
	_ = l.Sync()
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Desugar returns the underlying zap.Logger
func (l *Logger) Desugar() *zap.Logger {
	return l.Logger
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

