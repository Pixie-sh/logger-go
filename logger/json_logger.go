package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pixie-sh/logger-go/structs"
	"io"
	"time"
)

// JsonLogger represents a logger that outputs JSON logs.
type JsonLogger struct {
	App               string
	Scope             string
	UID               string
	LogLevel          LogLevelEnum
	writer            io.Writer
	expectedCtxFields []string
}

// innerJsonLog represents a logger with additional fields.
type innerJsonLog struct {
	*JsonLogger
	Ctx               context.Context
	fields            map[string]any
	expectedCtxFields []string
}

func (i *innerJsonLog) With(field string, value any) Interface {
	i.fields[field] = value
	return i
}

// WithCtx adds ctx to fields
func (i *innerJsonLog) WithCtx(ctx context.Context) Interface {
	i.Ctx = ctx
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

	logEntry := map[string]any{}
	for k, v := range i.fields {
		logEntry[k] = v
	}

	logEntry["timestamp"] = time.Now().Format(time.RFC3339)
	logEntry["level"] = level.String()
	logEntry["app"] = i.App
	logEntry["scope"] = i.Scope
	logEntry["message"] = msg

	if i.UID != "" {
		logEntry["uid"] = i.UID
	}

	if i.Ctx != nil {
		logEntry["ctx"] = i.ctxLog(i.Ctx)
	}

	jsonLog, err := json.Marshal(logEntry)
	if err != nil {
		_, _ = fmt.Fprintf(i.writer, "Error marshaling log: %v", err)
		return
	}

	_, _ = fmt.Fprintln(i.writer, string(jsonLog))
}

func (i *innerJsonLog) ctxLog(ctx context.Context) any {
	ctxFields := map[string]any{}

	for _, cf := range i.expectedCtxFields {
		val := ctx.Value(cf)
		if val != nil {
			ctxFields[cf] = val
		}
	}

	return ctxFields
}

// NewJsonLogger creates a new JsonLogger with default values.
func NewJsonLogger(
	_ context.Context,
	writer io.Writer,
	app, scope, uid string,
	logLevel LogLevelEnum,
	expectedCtxFields []string) (*JsonLogger, error) {
	return &JsonLogger{
		App:               app,
		Scope:             scope,
		UID:               uid,
		LogLevel:          logLevel,
		writer:            writer,
		expectedCtxFields: expectedCtxFields,
	}, nil
}

// With adds a field to the logger.
func (l *JsonLogger) With(field string, value any) Interface {
	return &innerJsonLog{
		JsonLogger:        l,
		Ctx:               context.Background(),
		expectedCtxFields: l.expectedCtxFields,
		fields:            map[string]any{field: value},
	}
}

// WithCtx adds ctx to fields
func (l *JsonLogger) WithCtx(ctx context.Context) Interface {
	return &innerJsonLog{
		JsonLogger:        l,
		Ctx:               ctx,
		expectedCtxFields: l.expectedCtxFields,
		fields:            map[string]any{},
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

	_, _ = fmt.Fprintln(l.writer, *structs.UnsafeString(jsonLog))
}
