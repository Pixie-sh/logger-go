package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// JsonLogger represents a logger that outputs JSON logs.
type JsonLogger struct {
	App      string
	Scope    string
	UID      string
	LogLevel LogLevelEnum
	writer   io.Writer
}

// innerJsonLog represents a logger with additional fields.
type innerJsonLog struct {
	*JsonLogger
	fields map[string]any
}

func (i *innerJsonLog) With(field string, value any) Interface {
	i.fields[field] = value
	return i
}

// Log logs a message at LOG level.
func (i *innerJsonLog) Log(format string, args ...any) {
	i.log(LOG, format, args...)
}

// Error logs a message at ERROR level.
func (i *innerJsonLog) Error(format string, args ...any) {
	i.log(ERROR, format, args...)
}

// Warn logs a message at WARN level.
func (i *innerJsonLog) Warn(format string, args ...any) {
	i.log(WARN, format, args...)
}

// Debug logs a message at DEBUG level.
func (i *innerJsonLog) Debug(format string, args ...any) {
	i.log(DEBUG, format, args...)
}

// log is an internal method to log messages with structured logging.
func (i *innerJsonLog) log(level LogLevelEnum, format string, args ...any) {
	if i.LogLevel < level {
		return
	}

	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	logEntry := map[string]any{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level.String(),
		"app":       i.App,
		"scope":     i.Scope,
		"message":   msg,
	}

	for k, v := range i.fields {
		logEntry[k] = v
	}

	if i.UID != "" {
		logEntry["uid"] = i.UID
	}

	jsonLog, err := json.Marshal(logEntry)
	if err != nil {
		_, _ = fmt.Fprintf(i.writer, "Error marshaling log: %v", err)
		return
	}

	_, _ = fmt.Fprintln(i.writer, string(jsonLog))
}

// NewJsonLogger creates a new JsonLogger with default values.
func NewJsonLogger(_ context.Context, writer io.Writer, app, scope, uid string, logLevel LogLevelEnum) (*JsonLogger, error) {
	return &JsonLogger{
		App:      app,
		Scope:    scope,
		UID:      uid,
		LogLevel: logLevel,
		writer:   writer,
	}, nil
}

// With adds a field to the logger.
func (l *JsonLogger) With(field string, value any) Interface {
	return &innerJsonLog{
		JsonLogger: l,
		fields:     map[string]any{field: value},
	}
}

// Log logs a message at LOG level.
func (l *JsonLogger) Log(format string, args ...any) {
	l.log(LOG, format, args...)
}

// Error logs a message at ERROR level.
func (l *JsonLogger) Error(format string, args ...any) {
	l.log(ERROR, format, args...)
}

// Warn logs a message at WARN level.
func (l *JsonLogger) Warn(format string, args ...any) {
	l.log(WARN, format, args...)
}

// Debug logs a message at DEBUG level.
func (l *JsonLogger) Debug(format string, args ...any) {
	l.log(DEBUG, format, args...)
}

// log is an internal method to log messages with structured logging.
func (l *JsonLogger) log(level LogLevelEnum, format string, args ...any) {
	if l.LogLevel < level {
		return
	}

	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	logEntry := map[string]any{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level.String(),
		"app":       l.App,
		"scope":     l.Scope,
		"message":   msg,
	}

	if l.UID != "" {
		logEntry["uid"] = l.UID
	}

	jsonLog, err := json.Marshal(logEntry)
	if err != nil {
		_, _ = fmt.Fprintf(l.writer, "Error marshaling log: %v", err)
		return
	}

	_, _ = fmt.Fprintln(l.writer, string(jsonLog))
}
