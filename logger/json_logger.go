package logger

import (
	"context"
	"fmt"
	"github.com/goccy/go-json"
	"io"
	"sync"

	"github.com/pixie-sh/logger-go/caller"
)

type ParserFn = func(
	level LogLevelEnum,
	app string,
	scope string,
	expandedMsg string,
	logUID string,
	ctxLog any,
	fields map[string]any,
) map[string]any

// JsonLogger represents a logger that outputs JSON logs.
type JsonLogger struct {
	App      string
	Scope    string
	UID      string
	LogLevel LogLevelEnum

	writer            io.Writer
	expectedCtxFields []string
	parser            ParserFn
}

// innerJsonLog represents a logger with additional fields.
type innerJsonLog struct {
	*JsonLogger

	mu                sync.RWMutex
	Ctx               context.Context
	expectedCtxFields []string

	fields map[string]any
	parser ParserFn
}

func (i *innerJsonLog) With(field string, value any) Interface {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.fields[field] = value
	return i
}

// WithCtx adds ctx to fields
func (i *innerJsonLog) WithCtx(ctx context.Context) Interface {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.Ctx = ctx
	return i
}

func (i *innerJsonLog) Clone() Interface {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// Create a new map and copy all fields
	newFields := make(map[string]any, len(i.fields))
	for k, v := range i.fields {
		newFields[k] = v
	}

	// Create a new innerJsonLog with copied fields
	return &innerJsonLog{
		JsonLogger:        i.JsonLogger,
		Ctx:               i.Ctx,
		fields:            newFields,
		parser:            i.parser,
		expectedCtxFields: i.expectedCtxFields,
	}
}

// Log logs a message at LOG level.
func (i *innerJsonLog) Log(format string, args ...any) {
	i.With("caller", caller.Upper())
	i.log(LOG, format, args...)
}

// Error logs a message at ERROR level.
func (i *innerJsonLog) Error(format string, args ...any) {
	i.With("caller", caller.Upper())
	i.log(ERROR, format, args...)
}

// Warn logs a message at WARN level.
func (i *innerJsonLog) Warn(format string, args ...any) {
	i.With("caller", caller.Upper())
	i.log(WARN, format, args...)
}

// Debug logs a message at DEBUG level.
func (i *innerJsonLog) Debug(format string, args ...any) {
	i.With("caller", caller.Upper())
	i.log(DEBUG, format, args...)
}

// log is an internal method to log messages with structured logging.
func (i *innerJsonLog) log(level LogLevelEnum, format string, args ...any) {
	if i.LogLevel < level {
		return
	}

	var msg = format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	var parser = DefaultJSONParser
	if i.parser != nil {
		parser = i.parser
	}

	i.mu.RLock()
	defer i.mu.RUnlock()
	logEntry := parser(level, i.App, i.Scope, msg, i.UID, i.ctxLog(i.Ctx), i.fields)
	jsonLog, err := json.Marshal(logEntry)
	if err != nil {
		_, _ = fmt.Fprintf(i.writer, "error marshaling log: %v; %+v", err, logEntry)
		return
	}

	_, _ = fmt.Fprintln(i.writer, string(jsonLog))
}

func (i *innerJsonLog) ctxLog(ctx context.Context) any {
	if ctx == nil {
		return nil
	}

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
	expectedCtxFields []string, parserFn ...ParserFn) (*JsonLogger, error) {
	parser := DefaultJSONParser
	if len(parserFn) > 0 && parserFn[0] != nil {
		parser = parserFn[0]
	}

	return &JsonLogger{
		App:               app,
		Scope:             scope,
		UID:               uid,
		LogLevel:          logLevel,
		writer:            writer,
		parser:            parser,
		expectedCtxFields: expectedCtxFields,
	}, nil
}

// With adds a field to the logger.
func (i *JsonLogger) With(field string, value any) Interface {
	return &innerJsonLog{
		JsonLogger:        i,
		Ctx:               context.Background(),
		expectedCtxFields: i.expectedCtxFields,
		parser:            i.parser,
		fields:            map[string]any{field: value},
	}
}

// WithCtx adds ctx to fields
func (i *JsonLogger) WithCtx(ctx context.Context) Interface {
	return &innerJsonLog{
		JsonLogger:        i,
		Ctx:               ctx,
		expectedCtxFields: i.expectedCtxFields,
		parser:            i.parser,
		fields:            map[string]any{},
	}
}

func (i *JsonLogger) Clone() Interface {
	return &JsonLogger{
		App:               i.App,
		Scope:             i.Scope,
		UID:               i.UID,
		LogLevel:          i.LogLevel,
		writer:            i.writer,
		expectedCtxFields: i.expectedCtxFields,
		parser:            i.parser,
	}
}

// Log logs a message at LOG level.
func (i *JsonLogger) Log(format string, args ...any) {
	i.log(LOG, caller.Upper(), format, args...)
}

// Error logs a message at ERROR level.
func (i *JsonLogger) Error(format string, args ...any) {
	i.log(ERROR, caller.Upper(), format, args...)
}

// Warn logs a message at WARN level.
func (i *JsonLogger) Warn(format string, args ...any) {
	i.log(WARN, caller.Upper(), format, args...)
}

// Debug logs a message at DEBUG level.
func (i *JsonLogger) Debug(format string, args ...any) {
	i.log(DEBUG, caller.Upper(), format, args...)
}

// log is an internal method to log messages with structured logging.
func (i *JsonLogger) log(level LogLevelEnum, call caller.Ptr, format string, args ...any) {
	if i.LogLevel < level {
		return
	}

	var msg = format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	var parser = DefaultJSONParser
	if i.parser != nil {
		parser = i.parser
	}

	logEntry := parser(level, i.App, i.Scope, msg, i.UID, nil, nil)
	jsonLog, err := json.Marshal(logEntry)
	if err != nil {
		_, _ = fmt.Fprintf(i.writer, "error marshaling log: %v; %+v", err, logEntry)
		return
	}

	_, _ = fmt.Fprintln(i.writer, string(jsonLog))
}
