package main

import (
	"emersyx.net/emersyx_apis/emcomapi"
	"errors"
	"io"
)

// i2tOptions implements the emcomapi.ProcessorOptions interface. Each method returns a function, which applies a
// specific configuration to a Processor object.
type i2tOptions struct {
}

// Logging sets the io.Writer instance to write logging messages to and the verbosity level.
func (o i2tOptions) Logging(writer io.Writer, level uint) func(emcomapi.Processor) error {
	return func(p emcomapi.Processor) error {
		if writer == nil {
			return errors.New("writer argument cannot be nil")
		}
		cp := assertProcessor(p)
		cp.log.SetOutput(writer)
		cp.log.SetLevel(level)
		return nil
	}
}

// Identifier sets the processor identifier value.
func (o i2tOptions) Identifier(id string) func(emcomapi.Processor) error {
	return func(p emcomapi.Processor) error {
		if len(id) == 0 {
			return errors.New("identifier cannot have zero length")
		}
		cp := assertProcessor(p)
		cp.identifier = id
		cp.log.SetComponentID(id)
		return nil
	}
}

// Config sets and applies the configuration options stored in the specified file.
func (o i2tOptions) Config(cfg string) func(emcomapi.Processor) error {
	return func(p emcomapi.Processor) error {
		if len(cfg) == 0 {
			return errors.New("configuration file path cannot have zero length")
		}
		cp := assertProcessor(p)
		return loadConfig(cfg, &cp.config)
	}
}

// Router sets the emcomapi.Router object to be used by the processor when querying for gateways.
func (o i2tOptions) Router(rtr emcomapi.Router) func(emcomapi.Processor) error {
	return func(p emcomapi.Processor) error {
		if rtr == nil {
			return errors.New("router argument cannot be nil")
		}
		cp := assertProcessor(p)
		cp.router = rtr
		return nil
	}
}

// assertProcessor tries to make a type assertion on the p argument, to the *i2tProcessor type. If the type assertion
// fails, then panic() is called. A call to recover() is in the applyOptions function.
func assertProcessor(p emcomapi.Processor) *i2tProcessor {
	cp, ok := p.(*i2tProcessor)
	if ok == false {
		panic("unsupported Processor implementation")
	}
	return cp
}

// NewProcessorOptions returns a new i2tOptions instance.
func NewProcessorOptions() (emcomapi.ProcessorOptions, error) {
	return new(i2tOptions), nil
}
