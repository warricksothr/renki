// Package config loads the controller's settings: where it listens, the token
// the plugin must present, and what goes into `welcome`.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"slices"

	"github.com/warricksothr/renki/internal/protocol"
)

// Config is the controller's configuration file. Every field has a default,
// so an empty file is a working configuration for a controller on the VM.
type Config struct {
	// Listen is the address the bridge serves on. Anything but loopback
	// needs a Token.
	Listen string `json:"listen"`
	// Token is what the plugin must send in `hello`. RENKI_TOKEN overrides
	// it, so it can stay out of the file.
	Token string `json:"token"`

	// AgentName is the name the bot answers to in chat. Empty, the default,
	// means the bot's character name in the game.
	AgentName string `json:"agentName"`
	// Nicknames are other names the bot answers to, such as "Tas" for
	// Tasleroch. Whole words, any case.
	Nicknames    []string                  `json:"nicknames"`
	Owners       []string                  `json:"owners"`
	Rates        protocol.Rates            `json:"rates"`
	Perception   protocol.PerceptionConfig `json:"perception"`
	AllowedTasks []string                  `json:"allowedTasks"`

	// LLM configures the in-character replies. Absent means the bot answers
	// only the command grammar.
	LLM *LLM `json:"llm,omitempty"`
}

// LLM configures how the controller reaches terva.
type LLM struct {
	// Persona is the character the bot plays. The bot's name is given to
	// the model separately, so the persona need not repeat it.
	Persona string `json:"persona"`
	// CooldownS is the minimum time between replies to one player.
	CooldownS float64 `json:"cooldownS"`
	// DailyTokens caps the tokens spent per day; 0 means no cap.
	DailyTokens int `json:"dailyTokens"`
	// Terva holds settings for the terva integration, passed through as is.
	Terva json.RawMessage `json:"terva,omitempty"`
}

// M1Tasks is the M1 task catalogue (docs/BRIDGE-PROTOCOL.md 7.2).
var M1Tasks = []string{
	protocol.TaskStop, protocol.TaskMoveTo, protocol.TaskFollow,
	protocol.TaskLookAt, protocol.TaskSay, protocol.TaskEmote,
}

// Default returns the configuration used for anything a file leaves out.
func Default() Config {
	return Config{
		Listen:       "127.0.0.1:8765",
		Rates:        protocol.Rates{StateHz: 5, PerceptionHz: 1},
		Perception:   protocol.PerceptionConfig{Radius: 40, MaxPerKind: 24},
		AllowedTasks: slices.Clone(M1Tasks),
	}
}

// Load reads path over the defaults. An empty path means defaults only.
func Load(path string) (Config, error) {
	c := Default()
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return c, err
		}
		if err := json.Unmarshal(data, &c); err != nil {
			return c, fmt.Errorf("%s: %w", path, err)
		}
	}
	if tok := os.Getenv("RENKI_TOKEN"); tok != "" {
		c.Token = tok
	}
	return c, c.Validate()
}

// Validate refuses a configuration that would be unsafe or cannot work.
func (c Config) Validate() error {
	host, _, err := net.SplitHostPort(c.Listen)
	if err != nil {
		return fmt.Errorf("listen %q: %w", c.Listen, err)
	}
	if c.Token == "" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return fmt.Errorf("listen %q is not loopback, so a token is required (set token or RENKI_TOKEN)", c.Listen)
		}
	}
	if c.Rates.StateHz <= 0 {
		return errors.New("rates.stateHz must be positive")
	}
	for _, t := range c.AllowedTasks {
		if !slices.Contains(M1Tasks, t) {
			return fmt.Errorf("allowedTasks: %q is not a task this controller knows", t)
		}
	}
	return nil
}

// IsOwner reports whether name is one of the owners, compared exactly as
// the game shows player names.
func (c Config) IsOwner(name string) bool { return slices.Contains(c.Owners, name) }
