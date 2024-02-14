package logger

import (
	"context"
	"fmt"
)

type FactoryCreateFn = func(ctx context.Context, configuration Configuration) (Interface, error)

type Factory struct {
	createMap map[string]FactoryCreateFn
}

func NewFactory(_ context.Context, config FactoryConfiguration) (Factory, error) {
	if config.Mapping == nil {
		return Factory{}, fmt.Errorf("unable to creater factory, configuration is missing mappings")
	}

	return Factory{
		createMap: config.Mapping,
	}, nil
}

// Create returns a new logger.Interface or error
func (f *Factory) Create(ctx context.Context, configuration Configuration) (Interface, error) {
	fn, exist := f.createMap[configuration.Driver]
	if !exist {
		return nil, fmt.Errorf("unknown logger driver %s. unable to create", configuration.Driver)
	}

	return fn(ctx, configuration)
}
