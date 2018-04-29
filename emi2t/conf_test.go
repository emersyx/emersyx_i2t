package main

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
	"testing"
)

var conffile = flag.String("conffile", "", "path to sample configuration fileto be used for testing purposes")

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestParsing(t *testing.T) {
	cfg := new(i2tConfig)
	if _, err := toml.DecodeFile(*conffile, cfg); err != nil {
		t.Log(err.Error())
		t.Log(fmt.Sprintf("could not decode the configuration file"))
		t.Fail()
	}

	if len(cfg.Links) != 1 {
		t.Log(fmt.Sprintf("expected 1 link in the config, got %d instead", len(cfg.Links)))
		t.Fail()
	}

	link := cfg.Links[0]
	if link.IRCGatewayID != "example_irc_id" {
		t.Log(fmt.Sprintf("invalid IRC gateway ID, got \"%s\"", link.IRCGatewayID))
		t.Fail()
	}
	if link.IRCChannel != "#emersyx" {
		t.Log(fmt.Sprintf("invalid IRC channel, got \"%s\"", link.IRCChannel))
		t.Fail()
	}
	if link.TelegramGatewayID != "example_telegram_id" {
		t.Log(fmt.Sprintf("invalid Telegram gateway ID, got \"%s\"", link.TelegramGatewayID))
		t.Fail()
	}
	if link.TelegramGroup != "-1001140292730" {
		t.Log(fmt.Sprintf("invalid Telegram group, got \"%s\"", link.TelegramGroup))
		t.Fail()
	}
}
