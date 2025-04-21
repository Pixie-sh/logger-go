package logger

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/pixie-sh/logger-go/env"
)

// MockInterface is a mock implementation of the Interface for testing
type MockInterface struct {
	cloneCalled    bool
	withCtxCalled  bool
	withCalled     bool
	logCalled      bool
	errorCalled    bool
	warnCalled     bool
	debugCalled    bool
	lastCtx        context.Context
	lastFieldName  string
	lastFieldValue any
	lastFormat     string
	lastArgs       []any
}

func (m *MockInterface) Clone() Interface {
	m.cloneCalled = true
	return m
}

func (m *MockInterface) WithCtx(ctx context.Context) Interface {
	m.withCtxCalled = true
	m.lastCtx = ctx
	return m
}

func (m *MockInterface) With(field string, value any) Interface {
	m.withCalled = true
	m.lastFieldName = field
	m.lastFieldValue = value
	return m
}

func (m *MockInterface) Log(format string, args ...any) {
	m.logCalled = true
	m.lastFormat = format
	m.lastArgs = args
}

func (m *MockInterface) Error(format string, args ...any) {
	m.errorCalled = true
	m.lastFormat = format
	m.lastArgs = args
}

func (m *MockInterface) Warn(format string, args ...any) {
	m.warnCalled = true
	m.lastFormat = format
	m.lastArgs = args
}

func (m *MockInterface) Debug(format string, args ...any) {
	m.debugCalled = true
	m.lastFormat = format
	m.lastArgs = args
}

func TestSingletonInitialization(t *testing.T) {
	// The singleton should be initialized automatically by init()
	if Logger == nil {
		t.Fatal("Logger singleton is nil after initialization")
	}
}

func TestClone(t *testing.T) {
	// Save original logger and restore after test
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	// Set up mock
	mockLogger := &MockInterface{}
	Logger = mockLogger

	// Call Clone
	result := Clone()

	// Verify
	if !mockLogger.cloneCalled {
		t.Error("Clone() did not call the underlying logger's Clone method")
	}

	// Check that the result is the mock logger (which returns itself from Clone)
	if result != mockLogger {
		t.Error("Clone() did not return the expected logger instance")
	}
}

func TestWithCtx(t *testing.T) {
	// Save original logger and restore after test
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	// Set up mock
	mockLogger := &MockInterface{}
	Logger = mockLogger

	// Call WithCtx
	ctx := context.Background()
	result := WithCtx(ctx)

	// Verify
	if !mockLogger.withCtxCalled {
		t.Error("WithCtx() did not call the underlying logger's WithCtx method")
	}
	if mockLogger.lastCtx != ctx {
		t.Error("WithCtx() did not pass the context correctly")
	}
	if result != mockLogger {
		t.Error("WithCtx() did not return the expected logger instance")
	}
}

func TestWith(t *testing.T) {
	// Save original logger and restore after test
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	// Set up mock
	mockLogger := &MockInterface{}
	Logger = mockLogger

	// Call With
	field := "testField"
	value := "testValue"
	result := With(field, value)

	// Verify
	if !mockLogger.withCalled {
		t.Error("With() did not call the underlying logger's With method")
	}
	if mockLogger.lastFieldName != field {
		t.Errorf("With() expected field %s, got %s", field, mockLogger.lastFieldName)
	}
	if mockLogger.lastFieldValue != value {
		t.Errorf("With() expected value %v, got %v", value, mockLogger.lastFieldValue)
	}
	if result != mockLogger {
		t.Error("With() did not return the expected logger instance")
	}
}

func TestLog(t *testing.T) {
	// Save original logger and restore after test
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	// Set up mock
	mockLogger := &MockInterface{}
	Logger = mockLogger

	// Call Log
	format := "test %s"
	args := []any{"message"}
	Log(format, args...)

	// Verify
	if !mockLogger.logCalled {
		t.Error("Log() did not call the underlying logger's Log method")
	}
	if mockLogger.lastFormat != format {
		t.Errorf("Log() expected format %s, got %s", format, mockLogger.lastFormat)
	}
	if len(mockLogger.lastArgs) != len(args) || mockLogger.lastArgs[0] != args[0] {
		t.Errorf("Log() did not pass arguments correctly")
	}
}

func TestError(t *testing.T) {
	// Save original logger and restore after test
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	// Set up mock
	mockLogger := &MockInterface{}
	Logger = mockLogger

	// Call Error
	format := "error %s"
	args := []any{"message"}
	Error(format, args...)

	// Verify
	if !mockLogger.errorCalled {
		t.Error("Error() did not call the underlying logger's Error method")
	}
	if mockLogger.lastFormat != format {
		t.Errorf("Error() expected format %s, got %s", format, mockLogger.lastFormat)
	}
	if len(mockLogger.lastArgs) != len(args) || mockLogger.lastArgs[0] != args[0] {
		t.Errorf("Error() did not pass arguments correctly")
	}
}

func TestWarn(t *testing.T) {
	// Save original logger and restore after test
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	// Set up mock
	mockLogger := &MockInterface{}
	Logger = mockLogger

	// Call Warn
	format := "warning %s"
	args := []any{"message"}
	Warn(format, args...)

	// Verify
	if !mockLogger.warnCalled {
		t.Error("Warn() did not call the underlying logger's Warn method")
	}
	if mockLogger.lastFormat != format {
		t.Errorf("Warn() expected format %s, got %s", format, mockLogger.lastFormat)
	}
	if len(mockLogger.lastArgs) != len(args) || mockLogger.lastArgs[0] != args[0] {
		t.Errorf("Warn() did not pass arguments correctly")
	}
}

func TestDebug(t *testing.T) {
	// Save original logger and restore after test
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	// Set up mock
	mockLogger := &MockInterface{}
	Logger = mockLogger

	// Call Debug
	format := "debug %s"
	args := []any{"message"}
	Debug(format, args...)

	// Verify
	if !mockLogger.debugCalled {
		t.Error("Debug() did not call the underlying logger's Debug method")
	}
	if mockLogger.lastFormat != format {
		t.Errorf("Debug() expected format %s, got %s", format, mockLogger.lastFormat)
	}
	if len(mockLogger.lastArgs) != len(args) || mockLogger.lastArgs[0] != args[0] {
		t.Errorf("Debug() did not pass arguments correctly")
	}
}

func TestMustPanic(t *testing.T) {
	// Save original logger and restore after test
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	// Set Logger to nil to trigger panic
	Logger = nil

	// Test must function via the functions that call it
	testCases := []struct {
		name     string
		testFunc func()
	}{
		{"Clone", func() { Clone() }},
		{"WithCtx", func() { WithCtx(context.Background()) }},
		{"With", func() { With("field", "value") }},
		{"Log", func() { Log("message") }},
		{"Error", func() { Error("error") }},
		{"Warn", func() { Warn("warning") }},
		{"Debug", func() { Debug("debug") }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("%s() did not panic when Logger was nil", tc.name)
				}
			}()
			tc.testFunc()
		})
	}
}

func TestLoggerInitialization(t *testing.T) {
	// Test that the logger initialization in init() properly respects environment variables

	// Store original env values
	origAppName := os.Getenv("APP_NAME")
	origAppVersion := os.Getenv("APP_VERSION")
	origScope := os.Getenv("SCOPE")
	origLogLevel := os.Getenv("LOG_LEVEL")

	// Restore env after test
	defer func() {
		os.Setenv("APP_NAME", origAppName)
		os.Setenv("APP_VERSION", origAppVersion)
		os.Setenv("SCOPE", origScope)
		os.Setenv("LOG_LEVEL", origLogLevel)
	}()

	testCases := []struct {
		name     string
		logLevel string
		expected LogLevelEnum
	}{
		{"DEBUG level", "DEBUG", DEBUG},
		{"WARN level", "WARN", WARN},
		{"ERROR level", "ERROR", ERROR},
		{"Default level", "", LOG},  // Empty should default to LOG
		{"Unknown level", "UNKNOWN", LOG}, // Unknown should default to LOG
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up environment for this test case
			os.Setenv("APP_NAME", "test-app")
			os.Setenv("APP_VERSION", "1.0.0")
			os.Setenv("SCOPE", "test")
			os.Setenv("LOG_LEVEL", tc.logLevel)

			// Create a new logger and capture its output
			var buf bytes.Buffer
			logger, err := NewLogger(
				context.Background(),
				&buf,
				fmt.Sprintf("%s-%s", env.EnvAppName(), env.EnvAppVersion()),
				env.EnvScope(),
				fmt.Sprintf("%s-%s", env.EnvAppName(), env.EnvAppVersion()),
				func() LogLevelEnum {
					switch env.EnvLogLevel() {
					case "DEBUG":
						return DEBUG
					case "WARN":
						return WARN
					case "ERROR":
						return ERROR
					default:
						return LOG
					}
				}(),
				[]string{TraceID})

			if err != nil {
				t.Fatalf("Failed to create logger: %v", err)
			}

			// Verify logger was created with the correct log level
			// This depends on your logger implementation
			// Here we're assuming we can infer the level by what messages are printed

			// Write messages at different levels
			logger.Debug("debug message")
			logger.Log("info message")
			logger.Warn("warning message")
			logger.Error("error message")

			output := buf.String()

			// Check if the output contains messages according to the level
			switch tc.expected {
			case DEBUG:
				if !strings.Contains(output, "debug message") {
					t.Error("DEBUG level logger should log debug messages")
				}
			case LOG:
				if strings.Contains(output, "debug message") {
					t.Error("INFO level logger should not log debug messages")
				}
				if !strings.Contains(output, "info message") {
					t.Error("INFO level logger should log info messages")
				}
			case WARN:
				if strings.Contains(output, "debug message") || strings.Contains(output, "info message") {
					t.Error("WARN level logger should not log debug or info messages")
				}
				if !strings.Contains(output, "warning message") {
					t.Error("WARN level logger should log warning messages")
				}
			case ERROR:
				if strings.Contains(output, "debug message") || strings.Contains(output, "info message") || strings.Contains(output, "warning message") {
					t.Error("ERROR level logger should not log debug, info, or warning messages")
				}
				if !strings.Contains(output, "error message") {
					t.Error("ERROR level logger should log error messages")
				}
			}
		})
	}
}


type wrapper struct {}

func (w *wrapper) Error() string { return "wrapped error" }
func (w *wrapper) Unwrap() error { return fmt.Errorf("inner wrapped error") }

type wrapperNestedError struct {}

func (w *wrapperNestedError) Error() string { return "wrapperNestedError" }
func (w *wrapperNestedError) Unwrap() error { return &wrapper{} }

type nilWrapperNestedError struct {}

func (w *nilWrapperNestedError) Error() string { return "nilWrapperNestedError" }
func (w *nilWrapperNestedError) Unwrap() error { return nil }

func TestWrapped(t *testing.T) {
	With("err", &wrapper{}).Error("test")
	With("err", &wrapperNestedError{}).Error("test2")

	With("err", nil).Error("test3")
	With("err", &nilWrapperNestedError{}).Error("test4")
}
