//go:build !novoice

package main

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/warricksothr/renki/internal/chat"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/settings"
	"github.com/warricksothr/renki/internal/voice"
)

// newVoice builds the terva voice from the `llm` block of the configuration,
// with the model read from the settings store before each reply.
func newVoice(cfg config.Config, model func() settings.Model, log *slog.Logger) (chat.Voice, error) {
	var tc voice.Config
	if len(cfg.LLM.Terva) > 0 {
		if err := json.Unmarshal(cfg.LLM.Terva, &tc); err != nil {
			return nil, fmt.Errorf("llm.terva: %w", err)
		}
	}
	return voice.New(voice.Options{
		Terva:       tc,
		Model:       model,
		Persona:     cfg.LLM.Persona,
		DailyTokens: cfg.LLM.DailyTokens,
		Logger:      log.With("part", "voice"),
	})
}
