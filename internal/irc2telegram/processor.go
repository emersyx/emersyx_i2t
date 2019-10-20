package main

import (
	"emersyx.net/common/pkg/api"
	"emersyx.net/common/pkg/api/irc"
	"emersyx.net/common/pkg/api/telegram"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"strconv"
	"strings"
)

// processor is the type which implements the functionality of the irc2tg processor. This struct implements the
// api.Peripheral interface.
type processor struct {
	api.PeripheralBase
	readyToForward bool
	events         chan api.Event
	links          []link
}

// GetIdentifier returns the identifier of the processor.
func (p *processor) GetIdentifier() string {
	return p.Identifier
}

// GetEventsInChannel returns the channel via which the Processor object receives Event objects. The channel is
// write-only and can not be read from.
func (p *processor) GetEventsInChannel() chan<- api.Event {
	return p.events
}

// eventLoop starts an infinite loop in which events received from receptors are processed. This method is executed in a
// sepparate goroutine.
func (p *processor) eventLoop() {
	for event := range p.events {
		p.Log.Debugf("event loop received new event of type %T\n", event)
		switch cevent := event.(type) {
		case api.CoreEvent:
			p.processCoreEvent(cevent)
		case irc.Message:
			if p.readyToForward {
				p.toTelegram(cevent)
			}
		case telegram.Update:
			if p.readyToForward {
				p.toIRC(cevent)
			}
		default:
			p.Log.Errorf(
				"processor %s received an unknown event type from receptor %s\n",
				p.Identifier,
				event.GetSourceIdentifier(),
			)
		}
	}
}

// toTelegram validates the IRC message and forwards it to the appropriate Telegram group, based on the contents of the
// toml configuration file.
func (p *processor) toTelegram(msg irc.Message) {
	// we only care about PRIVMSG events
	if msg.Command != irc.PRIVMSG {
		return
	}

	links := p.findLinks(msg.GetSourceIdentifier())
	for _, link := range links {
		// check if the message can/should be forwarded
		if msg.Parameters[0] != link.IRCChannel {
			continue
		}

		tggw := p.getTelegramGateway(link.TelegramGatewayID)
		if tggw == nil {
			continue
		}

		// configure the parameters for the Telegram API sendMessage method
		params := tggw.NewParameters()
		params.ChatID(link.TelegramGroup)
		params.Text(fmt.Sprintf(
			"<b>(irc) %s :</b> %s",
			msg.Origin,
			msg.Parameters[1],
		))
		params.ParseMode("HTML")

		// send the message
		if _, err := tggw.SendMessage(params); err != nil {
			p.Log.Errorln(err.Error())
			p.Log.Errorln("an error occured while forwarding a message from IRC to Telegram")
		}
	}
}

// getTelegramGateway retrieves the peripheral with the specified identifier and verifies that it is of type
// telegram.Gateway.
func (p *processor) getTelegramGateway(id string) telegram.Gateway {
	p.Log.Debugf("searching for the Telegram gateway with ID \"%s\"\n", id)
	// check if the Telegram gateway (still) exists
	gw, ok := p.Core.GetPeripheral(id)
	if ok == false {
		p.Log.Errorf("the Telegram gateway ID \"%s\" is not registered with the router\n", id)
		return nil
	}

	// check if the gateway is a valid Telegram gateway
	tggw, ok := gw.(telegram.Gateway)
	if ok == false {
		p.Log.Errorf("the Telegram gateway ID \"%s\" does not belong to an telegram.Gateway instance\n", id)
		return nil
	}

	return tggw
}

// toIRC validates the Telegram update and forwards the message to the appropriate IRC channel, based on the contents of
// the toml configuration file.
func (p *processor) toIRC(u telegram.Update) {
	// validate the received message
	if u.Message == nil {
		p.Log.Errorln("received a Telegram update which does not contain a message")
		return
	}
	if u.Message.From == nil {
		p.Log.Errorln("received a Telegram update with an anonymous message (i.e. empty field)")
		return
	}
	if u.Message.Text == "" {
		p.Log.Errorln("received a Telegram update with an empty message")
		return
	}
	if u.Message.Chat.Type != "group" && u.Message.Chat.Type != "supergroup" {
		p.Log.Errorln("received a Telegram update with a message not from a group or supergroup")
		return
	}

	p.Log.Debugf(
		"received a Telegram message from chat ID %d, username \"%s\"\n",
		u.Message.Chat.ID,
		u.Message.Chat.Username,
	)

	links := p.findLinks(u.GetSourceIdentifier())
	for _, link := range links {
		// check if the message can/should be forwarded
		if strconv.FormatInt(u.Message.Chat.ID, 10) != link.TelegramGroup &&
			"@"+u.Message.Chat.Username != link.TelegramGroup {
			continue
		}

		ircgw := p.getIRCGateway(link.IRCGatewayID)
		if ircgw == nil {
			continue
		}

		// retrieve the message of the sender
		var sender string
		if u.Message.From.Username != "" {
			sender = u.Message.From.Username
		} else {
			sender = u.Message.From.FirstName
			if u.Message.From.LastName != "" {
				sender += " " + u.Message.From.LastName
			}
		}
		sender = fmt.Sprintf("(tg) %s : ", sender)

		// send the message
		p.Log.Debugf("sending a PRIVMSG to the \"%s\" IRC channel\n", link.IRCChannel)
		text := strings.Split(u.Message.Text, "\n")
		for _, line := range text {
			if isWhitespaceString(line) {
				continue
			}
			if err := ircgw.Privmsg(link.IRCChannel, sender+line); err != nil {
				p.Log.Errorln(err.Error())
				p.Log.Errorln("an error occured while forwarding a message from Telegram to IRC")
			}
		}
	}
}

// getIRCGateway retrieves the peripheral with the specified identifier and verifies that it is of type irc.Gateway.
func (p *processor) getIRCGateway(id string) irc.Gateway {
	p.Log.Debugf("searching for the IRC gateway with ID \"%s\"\n", id)
	// check if the destination Telegram gateway (still) exists
	gw, ok := p.Core.GetPeripheral(id)
	if ok == false {
		p.Log.Errorf("the IRC gateway ID \"%s\" is not registered with the router\n", id)
		return nil
	}

	// check if the gateway is a valid IRC gateway
	ircgw, ok := gw.(irc.Gateway)
	if ok == false {
		p.Log.Errorf("the IRC gateway ID \"%s\" does not belong to an irc.Gateway instance\n", id)
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
func (p *processor) processCoreEvent(ce api.CoreEvent) {
	if ce.Type == api.CoreUpdate && ce.Status == api.PeripheralsLoaded {
		p.Log.Debugln("received update from emersyx core that all peripherals have been loaded")
		p.joinIRCChannels()
		p.readyToForward = true
	}
}

// joinIRCChannels uses the IRC gateways specified in the toml configuration file to join the required IRC channels for
// routing messages.
func (p *processor) joinIRCChannels() {
	p.Log.Debugln("joining IRC channels via the gateways")
	// each IRC gateway needs to join the specific #channel
	for _, link := range p.links {
		ircgw := p.getIRCGateway(link.IRCGatewayID)
		if ircgw == nil {
			continue
		}
		p.Log.Debugf("joining IRC channel \"%s\" on gateway \"%s\"\n", link.IRCChannel, link.IRCGatewayID)
		if err := ircgw.Join(link.IRCChannel); err != nil {
			p.Log.Debugln(err.Error())
			p.Log.Debugf(
				"could not join IRC channel \"%s\" on gateway \"%s\"\n",
				link.IRCChannel, link.IRCGatewayID,
			)
		}
	}
}

// findLinks searches for the links specified in the toml configuration file which contains an identifier equal to the
// given argument. If such a link is found, the the bool return value is true, otherwise it is false.
func (p *processor) findLinks(id string) []link {
	links := make([]link, 0)
	for _, l := range p.links {
		if l.IRCGatewayID == id || l.TelegramGatewayID == id {
			links = append(links, l)
		}
	}
	return links
}

// NewPeripheral creates a new processor instance, applies the options received as argument and validates it. If no
// errors occur, then the new instance is returned.
func NewPeripheral(opts api.PeripheralOptions) (api.Peripheral, error) {
	var err error

	// validate the core in options
	if opts.Core == nil {
		return nil, errors.New("core cannot be nil")
	}

	// create a new processor and initialize the base
	p := new(processor)
	p.InitializeBase(opts)

	// initially not ready to forward messages
	p.readyToForward = false

	// create the events channel
	p.events = make(chan api.Event)

	// apply the extended options from the config file
	c := new(config)
	if _, err = toml.DecodeFile(opts.ConfigPath, c); err != nil {
		return nil, err
	}
	if err = c.validate(); err != nil {
		return nil, err
	}
	c.apply(p)

	// start the event p loop in a new goroutine
	p.Log.Debugln("starting the emi2t event loop")
	go p.eventLoop()

	p.Log.Debugf("emersyx i2t proccessor \"%s\" initialized\n", p.Identifier)
	return p, nil
}
