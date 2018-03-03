package main

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

var confFile = flag.String("conffile", "", "path to sample configuration fileto be used for testing purposes")

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestParsing(t *testing.T) {
	var cfg i2tConfig
	loadConfig(*confFile, &cfg)

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
