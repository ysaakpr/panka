package logger

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "default config",
			config: &Config{
				Level:        "info",
				Format:       "console",
				EnableCaller: true,
			},
			wantErr: false,
		},
		{
			name: "json format",
			config: &Config{
				Level:        "debug",
				Format:       "json",
				EnableCaller: false,
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			config: &Config{
				Level:  "invalid",
				Format: "console",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use buffer to capture output
			buf := &bytes.Buffer{}
			if tt.config != nil {
				tt.config.Output = buf
			}

			logger, err := New(tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, logger)
			} else {
				require.NoError(t, err)
				require.NotNil(t, logger)
				
				// Test logging
				logger.Info("test message", zap.String("key", "value"))
				assert.Contains(t, buf.String(), "test message")
				
				// Cleanup
				_ = logger.Sync()
			}
		})
	}
}

func TestNewDevelopment(t *testing.T) {
	logger, err := NewDevelopment()
	require.NoError(t, err)
	require.NotNil(t, logger)
	
	// Should not panic
	logger.Debug("debug message")
	logger.Info("info message")
	
	_ = logger.Sync()
}

func TestNewProduction(t *testing.T) {
	logger, err := NewProduction()
	require.NoError(t, err)
	require.NotNil(t, logger)
	
	// Should not panic
	logger.Info("info message")
	logger.Error("error message")
	
	_ = logger.Sync()
}

func TestWithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger, err := New(&Config{
		Level:  "info",
		Format: "json",
		Output: buf,
	})
	require.NoError(t, err)

	logger.WithFields(
		zap.String("key1", "value1"),
		zap.Int("key2", 42),
	).Info("test message")

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key1")
	assert.Contains(t, output, "value1")
	assert.Contains(t, output, "key2")
	assert.Contains(t, output, "42")
	
	_ = logger.Sync()
}

func TestWithError(t *testing.T) {
	buf := &bytes.Buffer{}
	logger, err := New(&Config{
		Level:  "info",
		Format: "json",
		Output: buf,
	})
	require.NoError(t, err)

	testErr := assert.AnError
	logger.WithError(testErr).Error("operation failed")

	output := buf.String()
	assert.Contains(t, output, "operation failed")
	assert.Contains(t, output, "error")
	
	_ = logger.Sync()
}

func TestGlobalLogger(t *testing.T) {
	// Get global logger
	logger := Global()
	require.NotNil(t, logger)

	// Create new logger and set as global
	buf := &bytes.Buffer{}
	newLogger, err := New(&Config{
		Level:  "info",
		Format: "console",
		Output: buf,
	})
	require.NoError(t, err)
	
	SetGlobal(newLogger)
	
	// Test global functions
	Info("test info")
	assert.Contains(t, buf.String(), "test info")
	
	buf.Reset()
	Infof("test %s", "formatted")
	assert.Contains(t, buf.String(), "test formatted")
	
	_ = newLogger.Sync()
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		logFunc  func(*Logger)
		expected bool
	}{
		{
			name:  "debug level logs debug",
			level: "debug",
			logFunc: func(l *Logger) {
				l.Debug("debug message")
			},
			expected: true,
		},
		{
			name:  "info level skips debug",
			level: "info",
			logFunc: func(l *Logger) {
				l.Debug("debug message")
			},
			expected: false,
		},
		{
			name:  "info level logs info",
			level: "info",
			logFunc: func(l *Logger) {
				l.Info("info message")
			},
			expected: true,
		},
		{
			name:  "warn level skips info",
			level: "warn",
			logFunc: func(l *Logger) {
				l.Info("info message")
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger, err := New(&Config{
				Level:  tt.level,
				Format: "console",
				Output: buf,
			})
			require.NoError(t, err)

			tt.logFunc(logger)

			if tt.expected {
				assert.NotEmpty(t, buf.String())
			} else {
				assert.Empty(t, buf.String())
			}
			
			_ = logger.Sync()
		})
	}
}

func TestSugar(t *testing.T) {
	buf := &bytes.Buffer{}
	logger, err := New(&Config{
		Level:  "info",
		Format: "console",
		Output: buf,
	})
	require.NoError(t, err)

	sugar := logger.Sugar()
	require.NotNil(t, sugar)

	sugar.Infow("test message",
		"key1", "value1",
		"key2", 42,
	)

	output := buf.String()
	assert.Contains(t, output, "test message")
	
	_ = logger.Sync()
}

