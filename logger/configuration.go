package logger

import (
	"context"
	"io"
	"os"

	"github.com/pixie-sh/logger-go/mapper"
)

// FactoryConfiguration defines the required logger factory configuration
type FactoryConfiguration struct {
	Mapping map[string]FactoryCreateFn
}

// DefaultFactoryConfiguration default factory configuration that creates tje json logger
var DefaultFactoryConfiguration = FactoryConfiguration{
	Mapping: map[string]FactoryCreateFn{
		JSONLoggerDriver: createJSONLogger,
		TextLoggerDriver: createTextLogger,
	},
}

func createJSONLogger(ctx context.Context, generic Configuration) (Interface, error) {
	var cfg JSONLoggerConfiguration
	err := mapper.ObjectToStruct(generic.Values, &cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Writer == nil {
		cfg.Writer = os.Stdout //default
	}

	return NewLogger(
		ctx,
		cfg.Writer,
		generic.App,
		generic.Scope,
		generic.UID,
		generic.LogLevel,
		append(generic.ExpectedCtxFields, TraceID),
		DefaultJSONParser,
	)
}

func createTextLogger(ctx context.Context, generic Configuration) (Interface, error) {
	var cfg TextLoggerConfiguration
	err := mapper.ObjectToStruct(generic.Values, &cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Writer == nil {
		cfg.Writer = os.Stdout //default
	}

	return NewLogger(
		ctx,
		cfg.Writer,
		generic.App,
		generic.Scope,
		generic.UID,
		generic.LogLevel,
		append(generic.ExpectedCtxFields, TraceID),
		DefaultTextParser,
	)
}

// Configuration  logger generic config
type Configuration struct {
	App               string       `toml:"app" json:"app" mapstructure:"app"`
	Scope             string       `toml:"scope" json:"scope" mapstructure:"scope"`
	UID               string       `toml:"uid" json:"uid" mapstructure:"uid"`
	LogLevel          LogLevelEnum `toml:"level" json:"level" mapstructure:"level"`
	Driver            string       `toml:"driver" json:"driver" mapstructure:"driver"`
	Values            any          `toml:"values" json:"values" mapstructure:"values"`
	ExpectedCtxFields []string     `toml:"expectedCtxFields" json:"expectedCtxFields" mapstructure:"expectedCtxFields"`
}

// JSONLoggerConfiguration json logger with specific
type JSONLoggerConfiguration struct {
	Writer io.Writer
}

// TextLoggerConfiguration represents the configuration for a text-based logger.
// This includes details such as the destination writer for the log output.
type TextLoggerConfiguration struct {
	Writer io.Writer
}