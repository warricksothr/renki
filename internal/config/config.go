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

	// Audit is where the audit log goes (internal/audit).
	Audit Audit `json:"audit"`

	// DataDir holds what the controller remembers between runs, such as
	// the bot's homes, relative to the working directory.
	DataDir string `json:"dataDir"`

	// Home says when the bot goes home by itself.
	Home HomeRules `json:"home"`

	// Overload says what the bot does when it carries more than it can.
	Overload Overload `json:"overload"`

	// Panel is the web control panel (docs/PANEL.md).
	Panel Panel `json:"panel"`
}

// Panel configures the web control panel.
type Panel struct {
	// Listen is the panel's address, such as the VM's Tailscale address
	// and a port. Empty turns the panel off.
	Listen string `json:"listen"`
	// Token is what the login form wants. RENKI_PANEL_TOKEN overrides it.
	Token string `json:"token"`
}

// HomeRules say when the bot goes home without being told.
type HomeRules struct {
	// WhenLost: follow lost its leader.
	WhenLost bool `json:"whenLost"`
	// WhenStuck: stuck recovery gave up, on anything but the way home.
	WhenStuck bool `json:"whenStuck"`
	// AfterOutage: a body task was running when the controller or the
	// connection went away.
	AfterOutage bool `json:"afterOutage"`
	// IdleMinutes: this long with no task, away from home. 0 never.
	IdleMinutes float64 `json:"idleMinutes"`
	// AfterDeath is what the bot does once it is back after a respawn:
	// AfterDeathHome, AfterDeathResume or AfterDeathStay.
	AfterDeath string `json:"afterDeath"`
}

// Overload is what the bot does when it carries more than it can, which
// leaves it unable to run and, once its stamina is gone, to move.
type Overload struct {
	// Drop puts down the least valuable, heaviest things it carries,
	// never its gear, until it is back under its carry weight, says so,
	// and remembers where (internal/burden).
	Drop bool `json:"drop"`
}

// What the bot does after a respawn (HomeRules.AfterDeath).
const (
	// AfterDeathHome goes home.
	AfterDeathHome = "home"
	// AfterDeathResume picks a follow back up, and otherwise goes home.
	AfterDeathResume = "resume"
	// AfterDeathStay stays at the spawn point.
	AfterDeathStay = "stay"
)

// Audit configures the audit log.
type Audit struct {
	// Dir holds the daily files, relative to the working directory. Empty
	// turns the audit log off.
	Dir string `json:"dir"`
	// KeepDays is how many days of files to keep; 0 keeps them all.
	KeepDays int `json:"keepDays"`
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

// KnownTasks are the tasks this controller can send (docs/BRIDGE-PROTOCOL.md
// 7.2): M1's, and what M2 has added so far. The plugin's safety switches
// still decide which of them it offers; pickup needs AllowGather, and drop
// AllowDrop.
var KnownTasks = []string{
	protocol.TaskStop, protocol.TaskMoveTo, protocol.TaskFollow,
	protocol.TaskLookAt, protocol.TaskSay, protocol.TaskEmote,
	protocol.TaskPickup, protocol.TaskDrop,
}

// Default returns the configuration used for anything a file leaves out.
func Default() Config {
	return Config{
		Listen:       "127.0.0.1:8765",
		Rates:        protocol.Rates{StateHz: 5, PerceptionHz: 1},
		Perception:   protocol.PerceptionConfig{Radius: 40, MaxPerKind: 24},
		AllowedTasks: slices.Clone(KnownTasks),
		Audit:        Audit{Dir: "audit", KeepDays: 14},
		DataDir:      "data",
		Home:         HomeRules{WhenLost: true, WhenStuck: true, AfterOutage: true, IdleMinutes: 10, AfterDeath: AfterDeathHome},
		Overload:     Overload{Drop: true},
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
	if tok := os.Getenv("RENKI_PANEL_TOKEN"); tok != "" {
		c.Panel.Token = tok
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
	if c.Panel.Listen != "" {
		if _, _, err := net.SplitHostPort(c.Panel.Listen); err != nil {
			return fmt.Errorf("panel.listen %q: %w", c.Panel.Listen, err)
		}
		if len(c.Panel.Token) < 16 {
			return errors.New("panel.token must be at least 16 characters (or set RENKI_PANEL_TOKEN)")
		}
	}
	if c.DataDir == "" {
		return errors.New("dataDir is empty")
	}
	switch c.Home.AfterDeath {
	case AfterDeathHome, AfterDeathResume, AfterDeathStay:
	default:
		return fmt.Errorf("home.afterDeath %q: want home, resume or stay", c.Home.AfterDeath)
	}
	if c.Home.IdleMinutes < 0 {
		return errors.New("home.idleMinutes must not be negative")
	}
	if c.Audit.KeepDays < 0 {
		return errors.New("audit.keepDays must not be negative")
	}
	if c.Rates.StateHz <= 0 {
		return errors.New("rates.stateHz must be positive")
	}
	for _, t := range c.AllowedTasks {
		if !slices.Contains(KnownTasks, t) {
			return fmt.Errorf("allowedTasks: %q is not a task this controller knows", t)
		}
	}
	return nil
}

// IsOwner reports whether name is one of the owners, compared exactly as
// the game shows player names.
func (c Config) IsOwner(name string) bool { return slices.Contains(c.Owners, name) }
