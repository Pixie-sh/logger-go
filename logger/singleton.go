package logger

import (
	"context"
	"fmt"
	"github.com/pixie-sh/logger-go/env"
	"github.com/pixie-sh/logger-go/mapper"
	"os"
)

// Logger global instance to be used everywhere,
// until a specific instance is assigned
var Logger Interface

func init() {
	scope := env.EnvScope()
	if len(scope) == 0 {
		scope = "-"
	}

	Logger, _ = NewLogger(
		context.Background(),
		os.Stdout,
		fmt.Sprintf("%s-%s", env.EnvAppName(), env.EnvAppVersion()),
		scope,
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
		[]string{TraceID},
	)
}

func Clone() Interface {
	must(Logger)
	return Logger.Clone()
}

func must(l Interface) {
	if mapper.Nil(l) {
		panic(fmt.Errorf("logger is not initialized, please call NewLogger() first"))
	}
}

func WithCtx(ctx context.Context) Interface {
	must(Logger)
	return Logger.WithCtx(ctx)
}

func With(field string, value any) Interface {
	must(Logger)
	return Logger.With(field, value)
}

func Log(format string, args ...any) {
	must(Logger)
	Logger.Log(format, args...)
}

func Error(format string, args ...any) {
	must(Logger)
	Logger.Error(format, args...)
}

func Warn(format string, args ...any) {
	must(Logger)
	Logger.Warn(format, args...)
}

func Debug(format string, args ...any) {
	must(Logger)
	Logger.Debug(format, args...)
}
