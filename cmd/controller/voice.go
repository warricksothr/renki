//go:build !novoice

package main

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/warricksothr/renki/internal/chat"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/voice"
)

// newVoice builds the terva voice from the `llm` block of the configuration.
func newVoice(cfg config.Config, log *slog.Logger) (chat.Voice, error) {
	var tc voice.Config
	if len(cfg.LLM.Terva) > 0 {
		if err := json.Unmarshal(cfg.LLM.Terva, &tc); err != nil {
			return nil, fmt.Errorf("llm.terva: %w", err)
		}
	}
	return voice.New(voice.Options{
		Terva:       tc,
		Persona:     cfg.LLM.Persona,
		DailyTokens: cfg.LLM.DailyTokens,
		Logger:      log.With("part", "voice"),
	})
}
