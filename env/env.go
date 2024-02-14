package env

import (
	"os"
	"strings"
)

// AppName env var name
const AppName = "APP_NAME"

// AppVersion env var name
const AppVersion = "APP_VERSION"

// Scope env var name
const Scope = "SCOPE"

// DebugMode mode
const DebugMode = "DEBUG_MODE"

// IsDebugActive check if it's in debug mode
func IsDebugActive() bool {
	debugValue := os.Getenv(DebugMode)
	return debugValue == "TRUE" ||
		strings.ToUpper(debugValue) == "TRUE" ||
		debugValue == "1"
}

// EnvAppName app runtime name
func EnvAppName() string {
	return os.Getenv(AppName)
}

// EnvAppVersion app runtime version, eg: git commit hash
func EnvAppVersion() string {
	return os.Getenv(AppVersion)
}

// EnvScope app runtime scope, eg: staging, local, prod
func EnvScope() string {
	return os.Getenv(Scope)
}
