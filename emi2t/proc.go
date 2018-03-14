package main

import (
	"emersyx.net/emersyx_apis/emcomapi"
	"emersyx.net/emersyx_apis/emircapi"
	"emersyx.net/emersyx_apis/emtgapi"
	"emersyx.net/emersyx_log/emlog"
	"errors"
	"fmt"
	"strconv"
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
		proc.log.Debugf("event loop received new event of type %T\n", event)
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

// toTelegram validates the IRC message and forwards it to the appropriate Telegram group, based on the contents of the
// toml configuration file.
func (proc *i2tProcessor) toTelegram(msg emircapi.Message) {
	// we only care about PRIVMSG events
	if msg.Command != emircapi.PRIVMSG {
		return
	}

	links := proc.findLinks(msg.GetSourceIdentifier())
	for _, link := range links {
		// check if the message can/should be forwarded
		if msg.Parameters[0] != link.IRCChannel {
			proc.log.Errorln("received an IRC messages from an unlinked channel")
			return
		}

		// check if the destination Telegram gateway (still) exists
		gw, err := proc.router.GetGateway(link.TelegramGatewayID)
		if err != nil {
			proc.log.Errorln(err.Error())
			proc.log.Errorf(
				"the Telegram gateway ID \"%s\" is not registered with the router",
				link.TelegramGatewayID,
			)
			return
		}

		// check if the gateway is a valid Telegram gateway
		tggw, ok := gw.(emtgapi.TelegramGateway)
		if ok == false {
			proc.log.Errorf(
				"the Telegram gateway ID \"%s\" does not belong to an emtgapi.TelegramGateway instance\n",
				link.TelegramGatewayID,
			)
			return
		}

		// configure the parameters for the Telegram API sendMessage method
		params := tggw.NewTelegramParameters()
		params.ChatID(link.TelegramGroup)
		params.Text(fmt.Sprintf(
			"*%s*: %s",
			msg.Origin,
			msg.Parameters[1],
		))
		params.ParseMode("Markdown")

		// send the message
		if _, err := tggw.SendMessage(params); err != nil {
			proc.log.Errorln(err.Error())
			proc.log.Errorln("an error occured while forwarding a message from IRC to Telegram")
		}
	}
}

// toIRC validates the Telegram update and forwards the message to the appropriate IRC channel, based on the contents of
// the toml configuration file.
func (proc *i2tProcessor) toIRC(eu emtgapi.EUpdate) {
	// validate the received message
	if eu.Message == nil {
		proc.log.Errorln("received a Telegram update which does not contain a message")
		return
	}
	if eu.Message.From == nil {
		proc.log.Errorln("received a Telegram update with an anonymous message (i.e. empty field)")
		return
	}
	if eu.Message.Text == "" {
		proc.log.Errorln("received a Telegram update with an empty message")
		return
	}
	if eu.Message.Chat.Type != "group" && eu.Message.Chat.Type != "supergroup" {
		proc.log.Errorln("received a Telegram update with a message not from a group or supergroup")
		return
	}

	links := proc.findLinks(eu.GetSourceIdentifier())
	for _, link := range links {
		// check if the message can/should be forwarded
		if strconv.FormatInt(eu.Message.Chat.ID, 10) != link.TelegramGroup {
			proc.log.Errorln("received a Telegram messages from an unlinked (super)group")
			continue
		}

		// check if the destination Telegram gateway (still) exists
		gw, err := proc.router.GetGateway(link.IRCGatewayID)
		if err != nil {
			proc.log.Errorln(err.Error())
			proc.log.Errorf(
				"the IRC gateway ID \"%s\" is not registered with the router",
				link.IRCGatewayID,
			)
			continue
		}

		// check if the gateway is a valid IRC gateway
		ircgw, ok := gw.(emircapi.IRCGateway)
		if ok == false {
			proc.log.Errorf(
				"the IRC gateway ID \"%s\" does not belong to an emircapi.IRCGateway instance\n",
				link.TelegramGatewayID,
			)
			continue
		}

		// send the message
		if err := ircgw.Privmsg(link.IRCChannel, eu.Message.Text); err != nil {
			proc.log.Errorln(err.Error())
			proc.log.Errorln("an error occured while forwarding a message from Telegram to IRC")
		}
	}
}

// findLinks searches for the links specified in the toml configuration file which contains an identifier equal to the
// given argument. If such a link is found, the the bool return value is true, otherwise it is false.
func (proc *i2tProcessor) findLinks(id string) []link {
	links := make([]link, 1)
	for _, l := range proc.config.Links {
		if l.IRCGatewayID == id || l.TelegramGatewayID == id {
			links = append(links, l)
		}
	}
	return links
}

// NewProcessor creates a new i2tProcessor instance, applies the options received as argument and validates it. If no
// errors occur, then the new instance is returned.
func NewProcessor(options ...func(emcomapi.Processor) error) (emcomapi.Processor, error) {
	var err error

	proc := new(i2tProcessor)

	// generate a bare logger, to be updated via options
	proc.log, err = emlog.NewEmersyxLogger(nil, "", emlog.ELNone)
	if err != nil {
		return nil, errors.New("could not create a bare logger")
	}

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
	proc.log.Debugln("starting the emi2t event loop")
	go proc.eventLoop()

	proc.log.Debugf("emersyx i2t proccessor \"%s\" initialized\n", proc.identifier)
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
