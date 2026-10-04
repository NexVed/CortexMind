package main

import (
	"testing"

	"github.com/NexVed/Cortex/internal/config"
)

func TestServerArgumentsAreAppliedAndRestrictedToLoopback(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Port: 8090}}
	noBrowser, err := parseArgs(cfg, []string{"serve", "--http", "127.0.0.1:8100", "--no-browser"})
	if err != nil || !noBrowser || cfg.Server.Port != 8100 {
		t.Fatalf("server arguments ignored: %v", err)
	}
	for _, args := range [][]string{{"superuser", "upsert"}, {"serve", "--http", "0.0.0.0:8090"}, {"serve", "--http", "127.0.0.1:0"}} {
		if _, err := parseArgs(cfg, args); err == nil {
			t.Fatalf("unsupported arguments accepted: %v", args)
		}
	}
}
