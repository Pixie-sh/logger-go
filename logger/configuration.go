package logger

import (
	"context"
	"github.com/pixie-sh/logger-go/mapper"
	"io"
	"os"
)

// FactoryConfiguration defines the required logger factory configuration
type FactoryConfiguration struct {
	Mapping map[string]FactoryCreateFn
}

// DefaultFactoryConfiguration default factory configuration that creates tje json logger
var DefaultFactoryConfiguration = FactoryConfiguration{
	Mapping: map[string]FactoryCreateFn{
		JSONLoggerDriver: createJSONLogger,
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

	return NewJsonLogger(ctx, cfg.Writer, generic.App, generic.Scope, generic.UID, generic.LogLevel)
}

// Configuration  logger generic config
type Configuration struct {
	App      string       `toml:"app" ,json:"app" ,mapstructure:"app"`
	Scope    string       `toml:"scope" ,json:"scope" ,mapstructure:"scope"`
	UID      string       `toml:"uid" ,json:"uid" ,mapstructure:"uid"`
	LogLevel LogLevelEnum `toml:"level" ,json:"level" ,mapstructure:"level"`
	Driver   string       `toml:"driver" ,json:"driver" ,mapstructure:"driver"`
	Values   any          `toml:"values" ,json:"values" ,mapstructure:"values"`
}

// JSONLoggerConfiguration json logger with specific
type JSONLoggerConfiguration struct {
	Writer io.Writer
}
