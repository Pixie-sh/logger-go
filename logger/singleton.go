package logger

import (
	"context"
	"fmt"
	"os"

	"github.com/pixie-sh/logger-go/env"
)

// Logger is the global instance used by the package-level helpers.
// It may be reassigned by callers that need to inject a custom logger.
var Logger Interface

func init() {
	scope := env.EnvScope()
	if len(scope) == 0 {
		scope = "-"
	}

	appVersion := fmt.Sprintf("%s-%s", env.EnvAppName(), env.EnvAppVersion())

	l, err := NewLogger(
		os.Stdout,
		appVersion,
		scope,
		appVersion,
		levelFromEnv(),
		[]string{TraceID},
	)
	if err != nil {
		// NewLogger cannot currently fail; panic if that ever changes so
		// the misconfiguration is surfaced loudly.
		panic(fmt.Errorf("logger init failed: %w", err))
	}
	Logger = l
}

func levelFromEnv() LogLevelEnum {
	switch env.EnvLogLevel() {
	case "DEBUG":
		return DEBUG
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return LOG
	}
}

func must() {
	if Logger == nil {
		panic(fmt.Errorf("logger is not initialized, please assign logger.Logger before use"))
	}
}

func Clone() Interface                       { must(); return Logger.Clone() }
func WithCtx(ctx context.Context) Interface  { must(); return Logger.WithCtx(ctx) }
func With(field string, value any) Interface { must(); return Logger.With(field, value) }
func Level() LogLevelEnum                    { must(); return Logger.Level() }
func SetLevel(l LogLevelEnum)                { must(); Logger.SetLevel(l) }
func Log(format string, args ...any)         { must(); Logger.Log(format, args...) }
func Info(format string, args ...any)        { must(); Logger.Info(format, args...) }
func Error(format string, args ...any)       { must(); Logger.Error(format, args...) }
func Warn(format string, args ...any)        { must(); Logger.Warn(format, args...) }
func Debug(format string, args ...any)       { must(); Logger.Debug(format, args...) }
func Fatal(format string, args ...any)       { must(); Logger.Fatal(format, args...) }
