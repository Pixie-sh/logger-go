package logger

// LogLevelEnum is an enum to represent log levels.
type LogLevelEnum int

const (
	ERROR LogLevelEnum = iota
	WARN
	LOG
	DEBUG
)

// String returns the string representation of the LogLevelEnum.
func (l LogLevelEnum) String() string {
	switch l {
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

// Interface LoggerInterface represents the basic logging interface.
type Interface interface {
	With(field string, value any) Interface
	Log(format string, args ...any)
	Error(format string, args ...any)
	Warn(format string, args ...any)
	Debug(format string, args ...any)
}
