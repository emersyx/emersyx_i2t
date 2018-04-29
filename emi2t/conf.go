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

// i2tConfig is the type which holds all configuration options stored in the toml file for i2tProcessor instances.
type i2tConfig struct {
	Links []link
}

// validate checks the values loaded from the toml configuration file. If any value is found to be invalid, then an
// error is returned.
func (cfg *i2tConfig) validate() error {
	for _, link := range cfg.Links {
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

// apply sets the values loaded from the toml configuration file into the i2tProcessor object received as argument.
func (cfg *i2tConfig) apply(proc *i2tProcessor) {
	proc.links = make([]link, len(cfg.Links))
	copy(proc.links, cfg.Links)
}
