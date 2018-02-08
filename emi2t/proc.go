package emirc2tg

import (
	"emersyx.net/emersyx_apis/emcomapi"
	"errors"
)

// i2tProcessor is the type which implements the functionality of the irc2tg processor. This struct implements the
// emcomapi.Processor interface.
type i2tProcessor struct {
	identifier string
	events     chan emcomapi.Event
	router     emcomapi.Router
	config     i2tConfig
}

// GetIdentifier returns the identifier of the processor.
func (proc *i2tProcessor) GetIdentifier() string {
	return proc.identifier
}

// GetInEventsChannel returns the channel via which the Processor object receives Event objects. The channel is
// write-only and can not be read from.
func (proc *i2tProcessor) GetInEventsChannel() chan<- emcomapi.Event {
	return proc.events
}

// GetOutEventsChannel returns nil since the processor does not generate any events.
func (proc *i2tProcessor) GetOutEventsChannel() <-chan emcomapi.Event {
	return nil
}

// NewProcessor creates a new i2tProcessor instance, applies the options received as argument and validates it. If no
// errors occur, then the new instance is returned.
func NewProcessor(options ...func(emcomapi.Processor) error) (emcomapi.Processor, error) {
	proc := new(i2tProcessor)

	// apply the configuration options received as arguments
	for _, option := range options {
		err := option(proc)
		if err != nil {
			return nil, err
		}
	}

	if len(proc.identifier) == 0 {
		return nil, errors.New("identifier option not set or is invalid")
	}

	return proc, nil
}
