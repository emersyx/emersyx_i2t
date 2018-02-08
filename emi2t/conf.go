package emirc2tg

import (
	"github.com/BurntSushi/toml"
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

// loadConfig opens, reads and parses the toml configuration file specified as argument.
func loadConfig(path string, config *i2tConfig) error {
	_, err := toml.DecodeFile(path, config)
	if err != nil {
		return err
	}
	return nil
}
