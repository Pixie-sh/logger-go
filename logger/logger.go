package logger

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/goccy/go-json"
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

// logger represents a logger that outputs JSON logs.
type logger struct {
	App      string
	Scope    string
	UID      string
	LogLevel LogLevelEnum

	writer            io.Writer
	expectedCtxFields []string
	parser            ParserFn
}

// innerLogger represents a logger with additional fields.
type innerLogger struct {
	*logger

	mu                sync.RWMutex
	Ctx               context.Context
	expectedCtxFields []string

	fields map[string]any
	parser ParserFn
}

func (i *innerLogger) With(field string, value any) Interface {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.fields[field] = value
	return i
}

// WithCtx adds ctx to fields
func (i *innerLogger) WithCtx(ctx context.Context) Interface {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.Ctx = ctx
	return i
}

func (i *innerLogger) Clone() Interface {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// Create a new map and copy all fields
	newFields := make(map[string]any, len(i.fields))
	for k, v := range i.fields {
		newFields[k] = v
	}

	// Create a new innerLogger with copied fields
	return &innerLogger{
		logger:            i.logger,
		Ctx:               i.Ctx,
		fields:            newFields,
		parser:            i.parser,
		expectedCtxFields: i.expectedCtxFields,
	}
}

// Log logs a message at LOG level.
func (i *innerLogger) Log(format string, args ...any) {
	i.With("caller", caller.Upper())
	i.log(LOG, format, args...)
}

// Error logs a message at ERROR level.
func (i *innerLogger) Error(format string, args ...any) {
	i.With("caller", caller.Upper())
	i.log(ERROR, format, args...)
}

// Warn logs a message at WARN level.
func (i *innerLogger) Warn(format string, args ...any) {
	i.With("caller", caller.Upper())
	i.log(WARN, format, args...)
}

// Debug logs a message at DEBUG level.
func (i *innerLogger) Debug(format string, args ...any) {
	i.With("caller", caller.Upper())
	i.log(DEBUG, format, args...)
}

// log is an internal method to log messages with structured logging.
func (i *innerLogger) log(level LogLevelEnum, format string, args ...any) {
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

func (i *innerLogger) ctxLog(ctx context.Context) any {
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

// NewLogger creates a new logger with default values.
func NewLogger(
	_ context.Context,
	writer io.Writer,
	app, scope, uid string,
	logLevel LogLevelEnum,
	expectedCtxFields []string, parserFn ...ParserFn) (*logger, error) {
	parser := DefaultJSONParser
	if len(parserFn) > 0 && parserFn[0] != nil {
		parser = parserFn[0]
	}

	return &logger{
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
func (i *logger) With(field string, value any) Interface {
	return &innerLogger{
		logger:            i,
		Ctx:               context.Background(),
		expectedCtxFields: i.expectedCtxFields,
		parser:            i.parser,
		fields:            map[string]any{field: value},
	}
}

// WithCtx adds ctx to fields
func (i *logger) WithCtx(ctx context.Context) Interface {
	return &innerLogger{
		logger:            i,
		Ctx:               ctx,
		expectedCtxFields: i.expectedCtxFields,
		parser:            i.parser,
		fields:            map[string]any{},
	}
}

func (i *logger) Clone() Interface {
	return &logger{
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
func (i *logger) Log(format string, args ...any) {
	i.log(LOG, caller.Upper(), format, args...)
}

// Error logs a message at ERROR level.
func (i *logger) Error(format string, args ...any) {
	i.log(ERROR, caller.Upper(), format, args...)
}

// Warn logs a message at WARN level.
func (i *logger) Warn(format string, args ...any) {
	i.log(WARN, caller.Upper(), format, args...)
}

// Debug logs a message at DEBUG level.
func (i *logger) Debug(format string, args ...any) {
	i.log(DEBUG, caller.Upper(), format, args...)
}

// log is an internal method to log messages with structured logging.
func (i *logger) log(level LogLevelEnum, call caller.Ptr, format string, args ...any) {
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
