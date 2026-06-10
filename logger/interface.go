package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// LogLevelEnum is an enum to represent log levels.
//
// Ordering: lower number = less verbose. A logger only emits records whose
// level is <= the logger's configured level.
//
//	FATAL (-1) is always emitted regardless of the configured level.
//	ERROR ( 0) "errors only"
//	WARN  ( 1) "warn + error"
//	LOG   ( 2) "info + warn + error" (alias: Info)
//	DEBUG ( 3) "everything"
type LogLevelEnum int32

const (
	FATAL LogLevelEnum = -1
	ERROR LogLevelEnum = 0
	WARN  LogLevelEnum = 1
	LOG   LogLevelEnum = 2
	DEBUG LogLevelEnum = 3
)

// String returns the string representation of the LogLevelEnum.
func (l LogLevelEnum) String() string {
	switch l {
	case FATAL:
		return "FATAL"
	case ERROR:
		return "ERROR"
	case WARN:
		return "WARN"
	case LOG:
		return "LOG"
	case DEBUG:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel returns the LogLevelEnum for a level name. Accepts case-insensitive
// "FATAL", "ERROR", "WARN"/"WARNING", "LOG"/"INFO", "DEBUG".
func ParseLogLevel(s string) (LogLevelEnum, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "FATAL":
		return FATAL, nil
	case "ERROR", "ERR":
		return ERROR, nil
	case "WARN", "WARNING":
		return WARN, nil
	case "LOG", "INFO":
		return LOG, nil
	case "DEBUG":
		return DEBUG, nil
	default:
		return 0, fmt.Errorf("unknown log level %q", s)
	}
}

// MarshalText implements encoding.TextMarshaler.
func (l LogLevelEnum) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (l *LogLevelEnum) UnmarshalText(b []byte) error {
	v, err := ParseLogLevel(string(b))
	if err != nil {
		return err
	}
	*l = v
	return nil
}

// MarshalJSON emits the level as its string name.
func (l LogLevelEnum) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

// UnmarshalJSON accepts the string name, the numeric value, or null
// (which leaves the field at its zero value).
func (l *LogLevelEnum) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil && s != "" {
		v, perr := ParseLogLevel(s)
		if perr != nil {
			return perr
		}
		*l = v
		return nil
	}
	var n int32
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*l = LogLevelEnum(n)
	return nil
}

// Interface represents the basic logging interface.
type Interface interface {
	Clone() Interface
	WithCtx(ctx context.Context) Interface
	With(field string, value any) Interface

	Level() LogLevelEnum
	SetLevel(LogLevelEnum)

	// Deprecated: use Info.
	Log(format string, args ...any)
	Info(format string, args ...any)
	Error(format string, args ...any)
	Warn(format string, args ...any)
	Debug(format string, args ...any)
	Fatal(format string, args ...any)
}
