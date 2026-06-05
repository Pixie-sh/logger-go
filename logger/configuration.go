package logger

import (
	"io"
	"os"
)

// FactoryConfiguration defines the required logger factory configuration.
type FactoryConfiguration struct {
	Mapping map[string]FactoryCreateFn
}

// DefaultFactoryConfiguration registers the JSON and text driver creators.
var DefaultFactoryConfiguration = FactoryConfiguration{
	Mapping: map[string]FactoryCreateFn{
		JSONLoggerDriver: createJSONLogger,
		TextLoggerDriver: createTextLogger,
	},
}

// Configuration is the generic logger configuration consumed by a Factory.
//
// Writer is not serialized — supply it programmatically before calling Create.
type Configuration struct {
	App               string       `toml:"app" json:"app" mapstructure:"app"`
	Scope             string       `toml:"scope" json:"scope" mapstructure:"scope"`
	UID               string       `toml:"uid" json:"uid" mapstructure:"uid"`
	LogLevel          LogLevelEnum `toml:"level" json:"level" mapstructure:"level"`
	Driver            string       `toml:"driver" json:"driver" mapstructure:"driver"`
	ExpectedCtxFields []string     `toml:"expectedCtxFields" json:"expectedCtxFields" mapstructure:"expectedCtxFields"`
	Writer            io.Writer    `toml:"-" json:"-" mapstructure:"-"`
}

func createJSONLogger(cfg Configuration) (Interface, error) {
	return newFromConfig(cfg, DefaultJSONParser)
}

func createTextLogger(cfg Configuration) (Interface, error) {
	return newFromConfig(cfg, DefaultTextParser)
}

func newFromConfig(cfg Configuration, parser ParserFn) (Interface, error) {
	w := cfg.Writer
	if w == nil {
		w = os.Stdout
	}
	// Defensive copy: appending TraceID directly onto cfg.ExpectedCtxFields
	// would alias into the caller's backing array when it has spare capacity.
	// Also dedupe so an explicit TraceID in config isn't walked twice.
	fields := make([]string, 0, len(cfg.ExpectedCtxFields)+1)
	seenTraceID := false
	for _, f := range cfg.ExpectedCtxFields {
		if f == TraceID {
			seenTraceID = true
		}
		fields = append(fields, f)
	}
	if !seenTraceID {
		fields = append(fields, TraceID)
	}
	return NewLogger(
		w,
		cfg.App,
		cfg.Scope,
		cfg.UID,
		cfg.LogLevel,
		fields,
		parser,
	)
}
