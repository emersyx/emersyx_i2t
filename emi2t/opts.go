package emirc2tg

import (
	"emersyx.net/emersyx_apis/emcomapi"
	"errors"
)

// i2tOptions implements the emcomapi.ProcessorOptions interface. Each method returns a function, which applies a
// specific configuration to a Processor object.
type i2tOptions struct {
}

// Identifier sets the processor identifier value.
func (o i2tOptions) Identifier(id string) func(emcomapi.Processor) error {
	return func(p emcomapi.Processor) error {
		if len(id) == 0 {
			return errors.New("identifier cannot have zero length")
		}
		cp, ok := p.(*i2tProcessor)
		if ok == false {
			return errors.New("unsupported Processor implementation")
		}
		cp.identifier = id
		return nil
	}
}

// Config sets and applies the configuration options stored in the specified file.
func (o i2tOptions) Config(cfg string) func(emcomapi.Processor) error {
	return func(p emcomapi.Processor) error {
		if len(cfg) == 0 {
			return errors.New("configuration file path cannot have zero length")
		}
		cp, ok := p.(*i2tProcessor)
		if ok == false {
			return errors.New("unsupported Processor implementation")
		}
		return loadConfig(cfg, &cp.config)
	}
}

// Router sets the emcomapi.Router object to be used by the processor when querying for gateways.
func (o i2tOptions) Router(rtr emcomapi.Router) func(emcomapi.Processor) error {
	return func(p emcomapi.Processor) error {
		if rtr == nil {
			return errors.New("router argument cannot be nil")
		}
		cp, ok := p.(*i2tProcessor)
		if ok == false {
			return errors.New("unsupported Processor implementation")
		}
		cp.router = rtr
		return nil
	}
}

// NewProcessorOptions returns a new i2tOptions instance.
func NewProcessorOptions() (emcomapi.ProcessorOptions, error) {
	return new(i2tOptions), nil
}
