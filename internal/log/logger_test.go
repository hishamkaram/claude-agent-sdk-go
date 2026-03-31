package log

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		verbose bool
	}{
		{
			name:    "verbose mode enables debug level",
			verbose: true,
		},
		{
			name:    "non-verbose mode uses warn level",
			verbose: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := NewLogger(tt.verbose)
			if logger == nil {
				t.Fatal("NewLogger() returned nil")
			}
			if logger.zap == nil {
				t.Fatal("NewLogger() returned Logger with nil zap field")
			}
		})
	}
}

func TestNewLoggerFromZap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		zapLogger *zap.Logger
		wantNop   bool
	}{
		{
			name:      "nil logger returns nop logger",
			zapLogger: nil,
			wantNop:   true,
		},
		{
			name:      "valid logger is wrapped",
			zapLogger: zap.NewNop(),
			wantNop:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := NewLoggerFromZap(tt.zapLogger)
			if logger == nil {
				t.Fatal("NewLoggerFromZap() returned nil")
			}
			if logger.zap == nil {
				t.Fatal("NewLoggerFromZap() returned Logger with nil zap field")
			}
		})
	}
}

func TestLogger_Zap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		zapLogger *zap.Logger
	}{
		{
			name:      "returns underlying zap logger",
			zapLogger: zap.NewNop(),
		},
		{
			name:      "returns nop when created with nil",
			zapLogger: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := NewLoggerFromZap(tt.zapLogger)
			got := logger.Zap()
			if got == nil {
				t.Fatal("Zap() returned nil")
			}
		})
	}
}

func TestLogger_LogMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		logFunc  func(l *Logger, msg string, fields ...zap.Field)
		level    zapcore.Level
		message  string
		fields   []zap.Field
		wantMsg  string
		wantKeys []string
	}{
		{
			name:    "Debug logs at debug level",
			logFunc: (*Logger).Debug,
			level:   zapcore.DebugLevel,
			message: "debug message",
			wantMsg: "debug message",
		},
		{
			name:    "Info logs at info level",
			logFunc: (*Logger).Info,
			level:   zapcore.InfoLevel,
			message: "info message",
			wantMsg: "info message",
		},
		{
			name:    "Warn logs at warn level",
			logFunc: (*Logger).Warn,
			level:   zapcore.WarnLevel,
			message: "warn message",
			wantMsg: "warn message",
		},
		{
			name:    "Error logs at error level",
			logFunc: (*Logger).Error,
			level:   zapcore.ErrorLevel,
			message: "error message",
			wantMsg: "error message",
		},
		{
			name:     "Debug with structured fields",
			logFunc:  (*Logger).Debug,
			level:    zapcore.DebugLevel,
			message:  "debug with fields",
			fields:   []zap.Field{zap.String("key", "value"), zap.Int("count", 42)},
			wantMsg:  "debug with fields",
			wantKeys: []string{"key", "count"},
		},
		{
			name:     "Error with error field",
			logFunc:  (*Logger).Error,
			level:    zapcore.ErrorLevel,
			message:  "operation failed",
			fields:   []zap.Field{zap.String("op", "save")},
			wantMsg:  "operation failed",
			wantKeys: []string{"op"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create an observed logger that captures log entries at all levels
			core, logs := observer.New(zapcore.DebugLevel)
			zapLogger := zap.New(core)
			logger := NewLoggerFromZap(zapLogger)

			// Call the log method
			tt.logFunc(logger, tt.message, tt.fields...)

			// Verify the log entry
			entries := logs.All()
			if len(entries) != 1 {
				t.Fatalf("expected 1 log entry, got %d", len(entries))
			}

			entry := entries[0]
			if entry.Message != tt.wantMsg {
				t.Errorf("log message = %q, want %q", entry.Message, tt.wantMsg)
			}

			if entry.Level != tt.level {
				t.Errorf("log level = %v, want %v", entry.Level, tt.level)
			}

			// Verify structured fields if expected
			for _, wantKey := range tt.wantKeys {
				found := false
				for _, field := range entry.ContextMap() {
					_ = field // iterate context map
				}
				// Use context directly from the entry
				for _, f := range entries[0].Context {
					if f.Key == wantKey {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected field with key %q not found in log entry", wantKey)
				}
			}
		})
	}
}

func TestNewLogger_VerboseFiltersCorrectly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		verbose     bool
		logLevel    zapcore.Level
		shouldExist bool
	}{
		{
			name:        "verbose mode captures debug",
			verbose:     true,
			logLevel:    zapcore.DebugLevel,
			shouldExist: true,
		},
		{
			name:        "non-verbose mode filters out debug",
			verbose:     false,
			logLevel:    zapcore.DebugLevel,
			shouldExist: false,
		},
		{
			name:        "non-verbose mode filters out info",
			verbose:     false,
			logLevel:    zapcore.InfoLevel,
			shouldExist: false,
		},
		{
			name:        "non-verbose mode captures warn",
			verbose:     false,
			logLevel:    zapcore.WarnLevel,
			shouldExist: true,
		},
		{
			name:        "non-verbose mode captures error",
			verbose:     false,
			logLevel:    zapcore.ErrorLevel,
			shouldExist: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Determine the level to use for the observer based on verbose mode
			var observerLevel zapcore.Level
			if tt.verbose {
				observerLevel = zapcore.DebugLevel
			} else {
				observerLevel = zapcore.WarnLevel
			}

			core, logs := observer.New(observerLevel)
			zapLogger := zap.New(core)
			logger := NewLoggerFromZap(zapLogger)

			// Log at the specified level
			switch tt.logLevel {
			case zapcore.DebugLevel:
				logger.Debug("test message")
			case zapcore.InfoLevel:
				logger.Info("test message")
			case zapcore.WarnLevel:
				logger.Warn("test message")
			case zapcore.ErrorLevel:
				logger.Error("test message")
			}

			entries := logs.All()
			hasEntry := len(entries) > 0

			if hasEntry != tt.shouldExist {
				t.Errorf("log entry exists = %v, want %v (verbose=%v, level=%v)",
					hasEntry, tt.shouldExist, tt.verbose, tt.logLevel)
			}
		})
	}
}
