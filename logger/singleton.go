package logger

import (
	"context"
	"fmt"
	"github.com/pixie-sh/logger-go/env"
	"os"
)

// Logger global instance to be used everywhere, until a specific instance is assigned
var Logger, _ = NewJsonLogger(
	context.Background(),
	os.Stdout,
	fmt.Sprintf("%s-%s", env.EnvAppName(), env.EnvAppVersion()),
	env.EnvScope(),
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
	[]string{TraceID})
