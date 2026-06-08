// Package logger provides a structured logger with pluggable parsers,
// context-bound fields, and concurrency-safe cloning.
package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pixie-sh/logger-go/caller"
	"github.com/pixie-sh/logger-go/env"
)

// Entry is the value passed to ParserFn for each emitted log line.
type Entry struct {
	Timestamp time.Time
	Level     LogLevelEnum
	Caller    caller.Ptr
	App       string
	Scope     string
	UID       string
	Message   string
	Ctx       any
	Fields    map[string]any
}

// ParserFn turns an Entry into the bytes that get written to the writer.
// Implementations must NOT retain references to e or its maps after returning.
type ParserFn = func(e *Entry) []byte

// internalEmit is a package-private extension implemented by the concrete
// logger types. Callers that already know the correct caller frame and/or
// record time (the slog handler, the singleton wrappers) use this to bypass
// the level-method's own caller.Upper() resolution.
type internalEmit interface {
	emitAt(level LogLevelEnum, call caller.Ptr, t time.Time, format string, args ...any)
}

// writeTarget serialises concurrent writes from clones that share an io.Writer.
// log.Logger does the same.
type writeTarget struct {
	mu sync.Mutex
	w  io.Writer
}

var newlineBytes = []byte{'\n'}

// writeLine writes the payload followed by a newline. The two writes happen
// under one lock so callers see a single atomic line, and we never need to
// append('\n') into a slice that a custom ParserFn might be aliasing into a
// pool-backed buffer.
func (wt *writeTarget) writeLine(b []byte) {
	wt.mu.Lock()
	defer wt.mu.Unlock()
	_, _ = wt.w.Write(b)
	_, _ = wt.w.Write(newlineBytes)
}

// logger is the base, fieldless logger.
type logger struct {
	App   string
	Scope string
	UID   string

	level             atomic.Int32
	target            *writeTarget
	expectedCtxFields []string
	parser            ParserFn
}

// innerLogger wraps logger with attached fields + context. Intended for
// goroutine-local use after Clone()/With()/WithCtx().
type innerLogger struct {
	*logger

	mu     sync.RWMutex
	ctx    context.Context
	fields map[string]any
}

// NewLogger creates a new logger with default values.
func NewLogger(
	writer io.Writer,
	app, scope, uid string,
	logLevel LogLevelEnum,
	expectedCtxFields []string,
	parserFn ...ParserFn,
) (*logger, error) {
	if writer == nil {
		writer = os.Stdout
	}

	parser := DefaultJSONParser
	if env.EnvLogParser() == "text" {
		parser = DefaultTextParser
	}
	if len(parserFn) > 0 && parserFn[0] != nil {
		parser = parserFn[0]
	}

	l := &logger{
		App:               app,
		Scope:             scope,
		UID:               uid,
		target:            &writeTarget{w: writer},
		expectedCtxFields: expectedCtxFields,
		parser:            parser,
	}
	l.level.Store(int32(logLevel))
	return l, nil
}

// ---- logger (base) methods ----

func (i *logger) Level() LogLevelEnum     { return LogLevelEnum(i.level.Load()) }
func (i *logger) SetLevel(l LogLevelEnum) { i.level.Store(int32(l)) }

// Deprecated: use Info.
func (i *logger) Log(format string, a ...any) { i.info(caller.Upper(), format, a...) }
func (i *logger) Info(f string, a ...any)     { i.info(caller.Upper(), f, a...) }
func (i *logger) Error(f string, a ...any)    { i.emit(ERROR, caller.Upper(), f, a...) }
func (i *logger) Warn(f string, a ...any)     { i.emit(WARN, caller.Upper(), f, a...) }
func (i *logger) Debug(f string, a ...any)    { i.emit(DEBUG, caller.Upper(), f, a...) }

func (i *logger) info(call caller.Ptr, format string, args ...any) {
	i.emit(LOG, call, format, args...)
}

// exitFn is the process-terminator used by Fatal. Defaults to os.Exit and is
// overridable from tests so we can verify the defer-fires-on-panic behavior
// without actually killing the test binary.
var exitFn = os.Exit

// Fatal logs at FATAL level and terminates the process. The os.Exit(1) runs
// from a defer so that even if the parser or writer panics, the process is
// still killed — a Fatal that turns into a swallowed goroutine crash would
// be the most dangerous thing the library could do.
func (i *logger) Fatal(f string, a ...any) {
	defer exitFn(1)
	i.emit(FATAL, caller.Upper(), f, a...)
}

// With returns a new innerLogger carrying the given field.
func (i *logger) With(field string, value any) Interface {
	return &innerLogger{
		logger: i,
		ctx:    context.Background(),
		fields: map[string]any{field: value},
	}
}

// WithCtx returns a new innerLogger bound to the given ctx.
func (i *logger) WithCtx(ctx context.Context) Interface {
	return &innerLogger{
		logger: i,
		ctx:    ctx,
		fields: map[string]any{},
	}
}

// Clone returns an independent base logger that shares the underlying writer.
// Per-instance state (level) is copied; changing it on the clone does not
// affect the original.
func (i *logger) Clone() Interface {
	return i.cloneBase()
}

func (i *logger) cloneBase() *logger {
	c := &logger{
		App:               i.App,
		Scope:             i.Scope,
		UID:               i.UID,
		target:            i.target,
		expectedCtxFields: i.expectedCtxFields,
		parser:            i.parser,
	}
	c.level.Store(i.level.Load())
	return c
}

func (i *logger) emit(level LogLevelEnum, call caller.Ptr, format string, args ...any) {
	i.emitAt(level, call, time.Now(), format, args...)
}

func (i *logger) emitAt(level LogLevelEnum, call caller.Ptr, t time.Time, format string, args ...any) {
	if i.Level() < level {
		return
	}
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	e := Entry{
		Timestamp: t,
		Level:     level,
		Caller:    call,
		App:       i.App,
		Scope:     i.Scope,
		UID:       i.UID,
		Message:   msg,
	}
	blob := i.parser(&e)
	i.target.writeLine(blob)
}

// ---- innerLogger methods ----

// With adds (or overwrites) a field on this innerLogger and returns self.
// Intended for goroutine-local chains after Clone()/With()/WithCtx().
func (i *innerLogger) With(field string, value any) Interface {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.fields[field] = value
	return i
}

// WithCtx swaps the bound context and returns self.
func (i *innerLogger) WithCtx(ctx context.Context) Interface {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.ctx = ctx
	return i
}

// Clone returns an independent innerLogger. The base *logger is deep-copied
// so that SetLevel (and any future mutable per-instance state) on the clone
// does not affect the original.
func (i *innerLogger) Clone() Interface {
	i.mu.RLock()
	defer i.mu.RUnlock()
	newFields := make(map[string]any, len(i.fields))
	for k, v := range i.fields {
		newFields[k] = v
	}
	return &innerLogger{
		logger: i.cloneBase(),
		ctx:    i.ctx,
		fields: newFields,
	}
}

// Deprecated: use Info.
func (i *innerLogger) Log(f string, a ...any)   { i.info(caller.Upper(), f, a...) }
func (i *innerLogger) Info(f string, a ...any)  { i.info(caller.Upper(), f, a...) }
func (i *innerLogger) Error(f string, a ...any) { i.emit(ERROR, caller.Upper(), f, a...) }
func (i *innerLogger) Warn(f string, a ...any)  { i.emit(WARN, caller.Upper(), f, a...) }
func (i *innerLogger) Debug(f string, a ...any) { i.emit(DEBUG, caller.Upper(), f, a...) }
func (i *innerLogger) Fatal(f string, a ...any) {
	defer exitFn(1)
	i.emit(FATAL, caller.Upper(), f, a...)
}

func (i *innerLogger) emit(level LogLevelEnum, call caller.Ptr, format string, args ...any) {
	i.emitAt(level, call, time.Now(), format, args...)
}

func (i *innerLogger) info(call caller.Ptr, format string, args ...any) {
	i.emit(LOG, call, format, args...)
}

func (i *innerLogger) emitAt(level LogLevelEnum, call caller.Ptr, t time.Time, format string, args ...any) {
	if i.Level() < level {
		return
	}
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	i.mu.RLock()
	ctxLog := i.ctxLog(i.ctx)
	// Snapshot a stable view of the fields map for the parser. The lock is
	// released before the write so a slow writer can't block With() callers.
	var fields map[string]any
	if len(i.fields) > 0 {
		fields = make(map[string]any, len(i.fields))
		for k, v := range i.fields {
			fields[k] = v
		}
	}
	i.mu.RUnlock()

	e := Entry{
		Timestamp: t,
		Level:     level,
		Caller:    call,
		App:       i.App,
		Scope:     i.Scope,
		UID:       i.UID,
		Message:   msg,
		Ctx:       ctxLog,
		Fields:    fields,
	}
	blob := i.parser(&e)
	i.target.writeLine(blob)
}

// ctxLog matches the pre-rewrite semantics: when ctx is non-nil the result
// is always a (possibly empty) map, so downstream consumers see a stable
// "ctx" key in every WithCtx-derived line.
func (i *innerLogger) ctxLog(ctx context.Context) any {
	if ctx == nil {
		return nil
	}
	ctxFields := map[string]any{}
	for _, cf := range i.expectedCtxFields {
		if val := ctx.Value(cf); val != nil {
			ctxFields[cf] = val
		}
	}
	return ctxFields
}

// isNilish returns true for both untyped nil and typed-nil interface values
// (the classic `var e *MyErr; var i any = e; i == nil` // false footgun).
// Used by must() and by the parsers' error-field handling to avoid panicking
// on .Error()/.Unwrap() on a nil receiver.
func isNilish(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return rv.IsNil()
	}
	return false
}
