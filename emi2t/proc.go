package main

import (
	"emersyx.net/emersyx/api"
	"emersyx.net/emersyx/api/ircapi"
	"emersyx.net/emersyx/api/tgapi"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"strconv"
	"strings"
)

// i2tProcessor is the type which implements the functionality of the irc2tg processor. This struct implements the
// api.Peripheral interface.
type i2tProcessor struct {
	identifier     string
	core           api.Core
	readyToForward bool
	events         chan api.Event
	links          []link
	log            *api.EmersyxLogger
}

// GetIdentifier returns the identifier of the processor.
func (proc *i2tProcessor) GetIdentifier() string {
	return proc.identifier
}

// GetEventsInChannel returns the channel via which the Processor object receives Event objects. The channel is
// write-only and can not be read from.
func (proc *i2tProcessor) GetEventsInChannel() chan<- api.Event {
	return proc.events
}

// eventLoop starts an infinite loop in which events received from receptors are processed. This method is executed in a
// sepparate goroutine.
func (proc *i2tProcessor) eventLoop() {
	for event := range proc.events {
		proc.log.Debugf("event loop received new event of type %T\n", event)
		switch cevent := event.(type) {
		case api.CoreEvent:
			proc.processCoreEvent(cevent)
		case ircapi.IRCMessage:
			if proc.readyToForward {
				proc.toTelegram(cevent)
			}
		case tgapi.EUpdate:
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
func (proc *i2tProcessor) toTelegram(msg ircapi.IRCMessage) {
	// we only care about PRIVMSG events
	if msg.Command != ircapi.PRIVMSG {
		return
	}

	links := proc.findLinks(msg.GetSourceIdentifier())
	for _, link := range links {
		// check if the message can/should be forwarded
		if msg.Parameters[0] != link.IRCChannel {
			continue
		}

		tggw := proc.getTelegramGateway(link.TelegramGatewayID)
		if tggw == nil {
			continue
		}

		// configure the parameters for the Telegram API sendMessage method
		params := tggw.NewTelegramParameters()
		params.ChatID(link.TelegramGroup)
		params.Text(fmt.Sprintf(
			"<b>(irc) %s :</b> %s",
			msg.Origin,
			msg.Parameters[1],
		))
		params.ParseMode("HTML")

		// send the message
		if _, err := tggw.SendMessage(params); err != nil {
			proc.log.Errorln(err.Error())
			proc.log.Errorln("an error occured while forwarding a message from IRC to Telegram")
		}
	}
}

// getTelegramGateway retrieves the peripheral with the specified identifier and verifies that it is of type
// tgapi.TelegramGateway.
func (proc *i2tProcessor) getTelegramGateway(id string) tgapi.TelegramGateway {
	proc.log.Debugf("searching for the Telegram gateway with ID \"%s\"\n", id)
	// check if the Telegram gateway (still) exists
	gw, ok := proc.core.GetPeripheral(id)
	if ok == false {
		proc.log.Errorf("the Telegram gateway ID \"%s\" is not registered with the router\n", id)
		return nil
	}

	// check if the gateway is a valid Telegram gateway
	tggw, ok := gw.(tgapi.TelegramGateway)
	if ok == false {
		proc.log.Errorf("the Telegram gateway ID \"%s\" does not belong to an tgapi.TelegramGateway instance\n", id)
		return nil
	}

	return tggw
}

// toIRC validates the Telegram update and forwards the message to the appropriate IRC channel, based on the contents of
// the toml configuration file.
func (proc *i2tProcessor) toIRC(eu tgapi.EUpdate) {
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

	proc.log.Debugf(
		"received a Telegram message from chat ID %d, username \"%s\"\n",
		eu.Message.Chat.ID,
		eu.Message.Chat.Username,
	)

	links := proc.findLinks(eu.GetSourceIdentifier())
	for _, link := range links {
		// check if the message can/should be forwarded
		if strconv.FormatInt(eu.Message.Chat.ID, 10) != link.TelegramGroup &&
			"@"+eu.Message.Chat.Username != link.TelegramGroup {
			continue
		}

		ircgw := proc.getIRCGateway(link.IRCGatewayID)
		if ircgw == nil {
			continue
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
		text := strings.Split(eu.Message.Text, "\n")
		for _, line := range text {
			if isWhitespaceString(line) {
				continue
			}
			if err := ircgw.Privmsg(link.IRCChannel, sender+line); err != nil {
				proc.log.Errorln(err.Error())
				proc.log.Errorln("an error occured while forwarding a message from Telegram to IRC")
			}
		}
	}
}

// getIRCGateway retrieves the peripheral with the specified identifier and verifies that it is of type
// ircapi.IRCGateway.
func (proc *i2tProcessor) getIRCGateway(id string) ircapi.IRCGateway {
	proc.log.Debugf("searching for the IRC gateway with ID \"%s\"\n", id)
	// check if the destination Telegram gateway (still) exists
	gw, ok := proc.core.GetPeripheral(id)
	if ok == false {
		proc.log.Errorf("the IRC gateway ID \"%s\" is not registered with the router\n", id)
		return nil
	}

	// check if the gateway is a valid IRC gateway
	ircgw, ok := gw.(ircapi.IRCGateway)
	if ok == false {
		proc.log.Errorf("the IRC gateway ID \"%s\" does not belong to an ircapi.IRCGateway instance\n", id)
		return nil
	}

	return ircgw
}

// isWhitespaceString is a utility function which verifies whether the string given as argument contains only whitespace
// characters (i.e. ' ', newlines and tabs).
func isWhitespaceString(s string) bool {
	if len(s) == 0 {
		return true
	}
	for _, c := range s {
		if c != ' ' && c != '\n' && c != '\t' {
			return false
		}
	}
	return true
}

// processCoreEvent is a function which processes events received from the emersyx core.
func (proc *i2tProcessor) processCoreEvent(ce api.CoreEvent) {
	if ce.Type == api.CoreUpdate && ce.Status == api.PeripheralsLoaded {
		proc.log.Debugln("received update from emersyx core that all peripherals have been loaded")
		proc.joinIRCChannels()
		proc.readyToForward = true
	}
}

// joinIRCChannels uses the IRC gateways specified in the toml configuration file to join the required IRC channels for
// routing messages.
func (proc *i2tProcessor) joinIRCChannels() {
	proc.log.Debugln("joining IRC channels via the gateways")
	// each IRC gateway needs to join the specific #channel
	for _, link := range proc.links {
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
	for _, l := range proc.links {
		if l.IRCGatewayID == id || l.TelegramGatewayID == id {
			links = append(links, l)
		}
	}
	return links
}

// NewPeripheral creates a new i2tProcessor instance, applies the options received as argument and validates it. If no
// errors occur, then the new instance is returned.
func NewPeripheral(opts api.PeripheralOptions) (api.Peripheral, error) {
	var err error

	// validate identifier in options
	if len(opts.Identifier) == 0 {
		return nil, errors.New("identifier cannot have 0 length")
	}

	// validate the core in options
	if opts.Core == nil {
		return nil, errors.New("core cannot be nil")
	}

	proc := new(i2tProcessor)

	// initially not ready to forward messages
	proc.readyToForward = false

	// create the events channel
	proc.events = make(chan api.Event)

	proc.identifier = opts.Identifier
	proc.core = opts.Core
	proc.log, err = api.NewEmersyxLogger(opts.LogWriter, opts.Identifier, opts.LogLevel)
	if err != nil {
		return nil, err
	}

	// apply the extended options from the config file
	config := new(i2tConfig)
	if _, err = toml.DecodeFile(opts.ConfigPath, config); err != nil {
		return nil, err
	}
	if err = config.validate(); err != nil {
		return nil, err
	}
	config.apply(proc)

	// start the event processing loop in a new goroutine
	proc.log.Debugln("starting the emi2t event loop")
	go proc.eventLoop()

	proc.log.Debugf("emersyx i2t proccessor \"%s\" initialized\n", proc.identifier)
	return proc, nil
}
