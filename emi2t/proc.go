package emi2t

import (
	"emersyx.net/emersyx_apis/emcomapi"
	"emersyx.net/emersyx_apis/emircapi"
	"emersyx.net/emersyx_apis/emtgapi"
	"emersyx.net/emersyx_log/emlog"
	"errors"
)

// i2tProcessor is the type which implements the functionality of the irc2tg processor. This struct implements the
// emcomapi.Processor interface.
type i2tProcessor struct {
	identifier string
	events     chan emcomapi.Event
	router     emcomapi.Router
	config     i2tConfig
	log        *emlog.EmersyxLogger
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

// eventLoop starts an infinite loop in which events received from receptors are processed. This method is executed in a
// sepparate goroutine.
func (proc *i2tProcessor) eventLoop() {
	for event := range proc.events {
		switch cevent := event.(type) {
		case emircapi.Message:
			proc.toTelegram(cevent)
		case emtgapi.EUpdate:
			proc.toIRC(cevent)
		default:
			proc.log.Errorf(
				"processor %s received an unknown event type from receptor %s\n",
				proc.identifier,
				event.GetSourceIdentifier(),
			)
		}
	}
}

func (proc *i2tProcessor) toTelegram(msg emircapi.Message) {
}

func (proc *i2tProcessor) toIRC(eu emtgapi.EUpdate) {
}

// NewProcessor creates a new i2tProcessor instance, applies the options received as argument and validates it. If no
// errors occur, then the new instance is returned.
func NewProcessor(options ...func(emcomapi.Processor) error) (emcomapi.Processor, error) {
	proc := new(i2tProcessor)

	// apply the configuration options received as arguments
	if err := applyOptions(proc, options...); err != nil {
		return nil, err
	}

	if len(proc.identifier) == 0 {
		return nil, errors.New("identifier option not set or is invalid")
	}

	if proc.log == nil {
		return nil, errors.New("logging not properly configured")
	}

	// start the event processing loop in a new goroutine
	go proc.eventLoop()

	return proc, nil
}

// applyOptions executes the functions provided as the options argument with proc as argument. The implementation relies
// calls recover() in order to stop panicking, which may be caused by the call to panic() within the assertProcessor
// function. assertProcessor is used by functions returned by i2tOptions.
func applyOptions(proc *i2tProcessor, options ...func(emcomapi.Processor) error) (e error) {
	defer func() {
		if r := recover(); r != nil {
			e = r.(error)
		}
	}()

	for _, option := range options {
		if e = option(proc); e != nil {
			return
		}
	}

	return
}
