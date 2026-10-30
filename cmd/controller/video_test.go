package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/warricksothr/renki/internal/config"
)

func TestRelayCredentialIsRedactedWithoutChangingRuntimeConfig(t *testing.T) {
	cfg := config.Default()
	cfg.Panel.Video.Password = "private-relay-credential"
	b, err := json.Marshal(redacted(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "private-relay-credential") {
		t.Fatal("relay credential leaked")
	}
	if cfg.Panel.Video.Password != "private-relay-credential" {
		t.Fatal("redaction modified live config")
	}
}
