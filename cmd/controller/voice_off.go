//go:build novoice

package main

import (
	"errors"
	"log/slog"

	"github.com/warricksothr/renki/internal/chat"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/settings"
)

// newVoice in a build without terva (-tags novoice): the bot answers the
// command grammar only, and a configuration that asks for more is refused
// rather than quietly ignored.
func newVoice(config.Config, func() settings.Model, *slog.Logger) (chat.Voice, error) {
	return nil, errors.New("this controller was built without the LLM voice (-tags novoice); remove the llm block or use a full build")
}
