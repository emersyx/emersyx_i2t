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
	identifier     string
	readyToForward bool
	events         chan emcomapi.Event
	router         emcomapi.Router
	config         i2tConfig
	log            *emlog.EmersyxLogger
}

// GetIdentifier returns the identifier of the processor.
func (proc *i2tProcessor) GetIdentifier() string {
	return proc.identifier
}

// GetEventsInChannel returns the channel via which the Processor object receives Event objects. The channel is
// write-only and can not be read from.
func (proc *i2tProcessor) GetEventsInChannel() chan<- emcomapi.Event {
	return proc.events
}

// eventLoop starts an infinite loop in which events received from receptors are processed. This method is executed in a
// sepparate goroutine.
func (proc *i2tProcessor) eventLoop() {
	for event := range proc.events {
		proc.log.Debugf("event loop received new event of type %T\n", event)
		switch cevent := event.(type) {
		case emcomapi.CoreEvent:
			proc.processCoreEvent(cevent)
		case emircapi.Message:
			if proc.readyToForward {
				proc.toTelegram(cevent)
			}
		case emtgapi.EUpdate:
			if proc.readyToForward {
				proc.toIRC(cevent)
			}
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

		tggw := proc.getTelegramGateway(link.TelegramGatewayID)
		if tggw == nil {
			return
		}

		// configure the parameters for the Telegram API sendMessage method
		params := tggw.NewTelegramParameters()
		params.ChatID(link.TelegramGroup)
		params.Text(fmt.Sprintf(
			"*(irc) %s :* %s",
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

func (proc *i2tProcessor) getTelegramGateway(id string) emtgapi.TelegramGateway {
	proc.log.Debugf("searching for the Telegram gateway with ID \"%s\"\n", id)
	// check if the Telegram gateway (still) exists
	gw, err := proc.router.GetGateway(id)
	if err != nil {
		proc.log.Errorln(err.Error())
		proc.log.Errorf("the Telegram gateway ID \"%s\" is not registered with the router\n", id)
		return nil
	}

	// check if the gateway is a valid Telegram gateway
	tggw, ok := gw.(emtgapi.TelegramGateway)
	if ok == false {
		proc.log.Errorf("the Telegram gateway ID \"%s\" does not belong to an emtgapi.TelegramGateway instance\n", id)
		return nil
	}

	return tggw
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
		proc.log.Debugf("received a Telegram message from chat ID %d\n", eu.Message.Chat.ID)
		if strconv.FormatInt(eu.Message.Chat.ID, 10) != link.TelegramGroup {
			proc.log.Errorf("received a Telegram messages from an unlinked (super)group with chat ID %d, expected %s",
				eu.Message.Chat.ID,
				link.TelegramGroup,
			)
			continue
		}

		ircgw := proc.getIRCGateway(link.IRCGatewayID)
		if ircgw == nil {
			return
		}

		// retrieve the message of the sender
		var sender string
		if eu.Update.Message.From.Username != "" {
			sender = eu.Update.Message.From.Username
		} else {
			sender = eu.Update.Message.From.FirstName
			if eu.Update.Message.From.LastName != "" {
				sender += " " + eu.Update.Message.From.LastName
			}
		}
		sender = fmt.Sprintf("(tg) %s : ", sender)

		// send the message
		proc.log.Debugf("sending a PRIVMSG to the \"%s\" IRC channel\n", link.IRCChannel)
		if err := ircgw.Privmsg(link.IRCChannel, sender+eu.Message.Text); err != nil {
			proc.log.Errorln(err.Error())
			proc.log.Errorln("an error occured while forwarding a message from Telegram to IRC")
		}
	}
}

func (proc *i2tProcessor) getIRCGateway(id string) emircapi.IRCGateway {
	proc.log.Debugf("searching for the IRC gateway with ID \"%s\"\n", id)
	// check if the destination Telegram gateway (still) exists
	gw, err := proc.router.GetGateway(id)
	if err != nil {
		proc.log.Errorln(err.Error())
		proc.log.Errorf("the IRC gateway ID \"%s\" is not registered with the router\n", id)
		return nil
	}

	// check if the gateway is a valid IRC gateway
	ircgw, ok := gw.(emircapi.IRCGateway)
	if ok == false {
		proc.log.Errorf("the IRC gateway ID \"%s\" does not belong to an emircapi.IRCGateway instance\n", id)
		return nil
	}

	return ircgw
}

func (proc *i2tProcessor) processCoreEvent(ce emcomapi.CoreEvent) {
	if ce.Type == emcomapi.CoreUpdate && ce.Status == emcomapi.ComponentsLoaded {
		proc.log.Debugln("received update from emersyx core that all components have been loaded")
		proc.joinIRCChannels()
		proc.readyToForward = true
	}
}

func (proc *i2tProcessor) joinIRCChannels() {
	proc.log.Debugln("joining IRC channels via the gateways")
	// each IRC gateway needs to join the specific #channel
	for _, link := range proc.config.Links {
		ircgw := proc.getIRCGateway(link.IRCGatewayID)
		if ircgw == nil {
			continue
		}
		proc.log.Debugf("joining IRC channel \"%s\" on gateway \"%s\"\n", link.IRCChannel, link.IRCGatewayID)
		if err := ircgw.Join(link.IRCChannel); err != nil {
			proc.log.Debugln(err.Error())
			proc.log.Debugf(
				"could not join IRC channel \"%s\" on gateway \"%s\"\n",
				link.IRCChannel, link.IRCGatewayID,
			)
		}
	}
}

// findLinks searches for the links specified in the toml configuration file which contains an identifier equal to the
// given argument. If such a link is found, the the bool return value is true, otherwise it is false.
func (proc *i2tProcessor) findLinks(id string) []link {
	links := make([]link, 0)
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

	// initially not ready to forward messages
	proc.readyToForward = false

	// create the events channel
	proc.events = make(chan emcomapi.Event)

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
