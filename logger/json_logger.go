package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pixie-sh/logger-go/caller"
	"github.com/pixie-sh/logger-go/structs"
	"io"
	"reflect"
	"sync"
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

	mu                sync.RWMutex
	Ctx               context.Context
	fields            map[string]any
	expectedCtxFields []string
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

	var logEntry = make(map[string]any)
	var jsonLog []byte
	var err error
	var msg = format

	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	{
		i.mu.RLock()
		defer i.mu.RUnlock()

		for k, v := range i.fields {
			if v == nil {
				logEntry[k] = "nil"
			} else {
				switch v := v.(type) {
				case error:
					// Create a map to hold both struct values and error string
					errorInfo := make(map[string]interface{})

					// Always add the error string
					errorInfo["errorString"] = v.Error()

					// Try to unwrap the error
					var innerErr interface{} = v
					for {
						u, ok := innerErr.(interface{ Unwrap() error })
						if !ok {
							break
						}
						innerErr = u.Unwrap()
						if innerErr == nil {
							break
						}
					}

					// check if it's a fmt.Errorf type
					if reflect.TypeOf(innerErr).String() != "*errors.errorString" {
						// for other error types, try reflection
						errorValue := reflect.ValueOf(innerErr)
						if errorValue.Kind() == reflect.Ptr {
							errorValue = errorValue.Elem()
						}
						if errorValue.Kind() == reflect.Struct {
							for i := 0; i < errorValue.NumField(); i++ {
								field := errorValue.Type().Field(i)
								if field.IsExported() {
									errorInfo[field.Name] = errorValue.Field(i).Interface()
								}
							}
						}
					}

					logEntry[k] = errorInfo

				default:
					logEntry[k] = v
				}
			}
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

		jsonLog, err = json.Marshal(logEntry)
		if err != nil {
			_, _ = fmt.Fprintf(i.writer, "Error marshaling log: %v", err)
			return
		}
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
func (i *JsonLogger) With(field string, value any) Interface {
	return &innerJsonLog{
		JsonLogger:        i,
		Ctx:               context.Background(),
		expectedCtxFields: i.expectedCtxFields,
		fields:            map[string]any{field: value},
	}
}

// WithCtx adds ctx to fields
func (i *JsonLogger) WithCtx(ctx context.Context) Interface {
	return &innerJsonLog{
		JsonLogger:        i,
		Ctx:               ctx,
		expectedCtxFields: i.expectedCtxFields,
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

	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	logEntry := map[string]any{
		"caller":    call,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level.String(),
		"app":       i.App,
		"scope":     i.Scope,
		"message":   msg,
	}

	if i.UID != "" {
		logEntry["uid"] = i.UID
	}

	jsonLog, err := json.Marshal(logEntry)
	if err != nil {
		_, _ = fmt.Fprintf(i.writer, "Error marshaling log: %v", err)
		return
	}

	_, _ = fmt.Fprintln(i.writer, *structs.UnsafeString(jsonLog))
}
