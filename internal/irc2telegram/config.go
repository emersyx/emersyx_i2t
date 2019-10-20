package main

import (
	"errors"
)

// link is the type which holds information about IRC channels and Telegram groups between which messages are forwarded.
type link struct {
	IRCGatewayID      string `toml:"irc_gateway_id"`
	IRCChannel        string `toml:"irc_channel"`
	TelegramGatewayID string `toml:"telegram_gateway_id"`
	TelegramGroup     string `toml:"telegram_group"`
}

// config is the type which holds all configuration options stored in the toml file for processor instances.
type config struct {
	Links []link
}

// validate checks the values loaded from the toml configuration file. If any value is found to be invalid, then an
// error is returned.
func (c *config) validate() error {
	for _, link := range c.Links {
		if len(link.IRCGatewayID) == 0 {
			return errors.New("IRC gateway ID cannot have 0 length")
		}
		if len(link.IRCChannel) == 0 {
			return errors.New("IRC channel name cannot have 0 length")
		}
		if len(link.TelegramGatewayID) == 0 {
			return errors.New("Telegram gateway ID cannot have 0 length")
		}
		if len(link.TelegramGroup) == 0 {
			return errors.New("Telegram group name cannot have 0 length")
		}
	}
	return nil
}

// apply sets the values loaded from the toml configuration file into the processor object received as argument.
func (c *config) apply(proc *processor) {
	proc.links = make([]link, len(c.Links))
	copy(proc.links, c.Links)
}
