package logger

import (
	"context"
	"github.com/pixie-sh/logger-go/mapper"
	"io"
)

type FactoryConfiguration struct {
	Mapping map[string]FactoryCreateFn
}

var DefaultFactoryConfiguration = FactoryConfiguration{
	Mapping: map[string]FactoryCreateFn{
		StdOutLoggerDriver: createJSONLogger,
	},
}

func createJSONLogger(ctx context.Context, generic Configuration) (Interface, error) {
	var cfg JSONLoggerConfiguration
	err := mapper.ObjectToStruct(generic.Values, &cfg)
	if err != nil {
		return nil, err
	}

	return NewJsonLogger(ctx, cfg.Writer, generic.App, generic.Scope, generic.UID, generic.LogLevel)
}

// Configuration  logger generic
type Configuration struct {
	App      string
	Scope    string
	UID      string
	LogLevel LogLevelEnum
	Driver   string      `toml:"driver" ,mapstructure:"driver"`
	Values   interface{} `toml:"values" ,mapstructure:"values"`
}

// JSONLoggerConfiguration std out logger config
type JSONLoggerConfiguration struct {
	Writer io.Writer
}
