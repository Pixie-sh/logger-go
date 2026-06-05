package logger

import "fmt"

// FactoryCreateFn builds an Interface from a Configuration.
type FactoryCreateFn = func(configuration Configuration) (Interface, error)

// Factory dispatches Configuration to a registered driver.
type Factory struct {
	createMap map[string]FactoryCreateFn
}

// NewFactory returns a new Logger factory instance.
func NewFactory(config FactoryConfiguration) (Factory, error) {
	if config.Mapping == nil {
		return Factory{}, fmt.Errorf("unable to create factory, configuration is missing mappings")
	}
	return Factory{createMap: config.Mapping}, nil
}

// Create returns a new logger.Interface or error.
func (f *Factory) Create(configuration Configuration) (Interface, error) {
	fn, exist := f.createMap[configuration.Driver]
	if !exist {
		return nil, fmt.Errorf("unknown logger driver %q. unable to create", configuration.Driver)
	}
	return fn(configuration)
}
