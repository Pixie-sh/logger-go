package logger

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pixie-sh/logger-go/caller"
	"github.com/pixie-sh/logger-go/env"
)

// Logger is the global instance used by the package-level helpers.
//
// Reading Logger is safe; mutating it from multiple goroutines is NOT
// race-safe via direct assignment. Use SetLogger to swap at runtime.
// The package-level helpers (Log/Info/Error/...) read through an internal
// RWMutex so a concurrent SetLogger cannot tear an interface value mid-call.
var Logger Interface

var loggerMu sync.RWMutex

// SetLogger atomically replaces the global Logger. Safe for concurrent use
// from any goroutine.
func SetLogger(l Interface) {
	loggerMu.Lock()
	Logger = l
	loggerMu.Unlock()
}

func getLogger() Interface {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return Logger
}

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
		panic(fmt.Errorf("logger init failed: %w", err))
	}
	Logger = l
}

// levelFromEnv mirrors ParseLogLevel but with the default-to-LOG behavior
// the singleton has always had. Kept as a thin wrapper so the two stay in
// lockstep: anything ParseLogLevel learns to accept, the env path picks up
// for free.
func levelFromEnv() LogLevelEnum {
	if v, err := ParseLogLevel(env.EnvLogLevel()); err == nil {
		return v
	}
	return LOG
}

// must verifies Logger is set and not a typed-nil interface value (the
// classic `var x *Impl = nil; Logger = x` footgun that bypasses a plain
// nil-check).
func must() {
	l := getLogger()
	if l == nil || isNilish(l) {
		panic(fmt.Errorf("logger is not initialized, please assign logger.Logger before use"))
	}
}

func Clone() Interface                       { must(); return getLogger().Clone() }
func WithCtx(ctx context.Context) Interface  { must(); return getLogger().WithCtx(ctx) }
func With(field string, value any) Interface { must(); return getLogger().With(field, value) }
func Level() LogLevelEnum                    { must(); return getLogger().Level() }
func SetLevel(l LogLevelEnum)                { must(); getLogger().SetLevel(l) }

// The package-level Log/Info/Error/Warn/Debug helpers add one stack frame
// over the underlying method, so a plain delegation would resolve caller
// at the singleton wrapper instead of the user. When the active Logger is
// one of our concrete types (implements internalEmit) we resolve the
// caller here and bypass the level method's own caller.Upper() call.
// Otherwise (custom Interface implementations) we delegate normally and
// accept the depth shift.

// caller.Upper() MUST be evaluated in the singleton wrapper itself, not in
// a helper — its depth-3 lookup assumes the chain user → singleton.X →
// caller.Upper. Calling it one frame deeper would still point at the
// wrapper. We pass the resolved caller and the active logger into a small
// dispatcher.

// Deprecated: use Info.
func Log(format string, args ...any)   { info(caller.Upper(), format, args) }
func Info(format string, args ...any)  { info(caller.Upper(), format, args) }
func Error(format string, args ...any) { dispatch(ERROR, caller.Upper(), format, args) }
func Warn(format string, args ...any)  { dispatch(WARN, caller.Upper(), format, args) }
func Debug(format string, args ...any) { dispatch(DEBUG, caller.Upper(), format, args) }

func info(c caller.Ptr, format string, args []any) {
	dispatch(LOG, c, format, args)
}

func Fatal(format string, args ...any) {
	c := caller.Upper()
	must()
	l := getLogger()
	if e, ok := l.(internalEmit); ok {
		defer exitFn(1)
		e.emitAt(FATAL, c, time.Now(), format, args...)
		return
	}
	// Custom Interface implementer (or test mock) — defer to its Fatal so
	// it controls whether the process exits.
	l.Fatal(format, args...)
}

func dispatch(level LogLevelEnum, c caller.Ptr, format string, args []any) {
	must()
	l := getLogger()
	if e, ok := l.(internalEmit); ok {
		e.emitAt(level, c, time.Now(), format, args...)
		return
	}
	switch level {
	case ERROR:
		l.Error(format, args...)
	case WARN:
		l.Warn(format, args...)
	case DEBUG:
		l.Debug(format, args...)
	default:
		l.Info(format, args...)
	}
}
