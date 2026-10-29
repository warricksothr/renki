// Package app wires the controller together: the bridge, the session and the
// chat brain, from a configuration.
package app

import (
	"context"
	"log/slog"
	"net"
	"slices"
	"time"

	"github.com/warricksothr/renki/internal/audit"
	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/burden"
	"github.com/warricksothr/renki/internal/chat"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
	"github.com/warricksothr/renki/internal/settings"
)

// App is a configured controller.
type App struct {
	Config  config.Config
	Session *session.Session
	Brain   *chat.Brain
	Home    *home.Keeper
	Burden  *burden.Unloader
	Bridge  *bridge.Server
	// Settings are what can change while the bot runs: owners, nicknames,
	// allowedTasks, the voice switch, the home rules and overload.drop.
	Settings *settings.Store
}

// New builds a controller from cfg. voice may be nil, which leaves the bot
// with the command grammar only.
// audit may be nil, which records nothing, homes nil, which remembers
// homes only until the controller stops, and set nil, which keeps runtime
// settings in memory over cfg.
func New(cfg config.Config, voice chat.Voice, log *slog.Logger, audit *audit.Log, homes *home.Store, set *settings.Store) *App {
	if log == nil {
		log = slog.Default()
	}
	if homes == nil {
		homes = home.Memory()
	}
	if set == nil {
		var err error
		// cfg has passed Validate, so its own values build.
		if set, err = settings.Open("", cfg, log.With("part", "settings"), nil); err != nil {
			panic(err)
		}
	}
	s := session.New(nil, log.With("part", "session"))
	s.SetPermit(func(task string) bool { return slices.Contains(set.Values().AllowedTasks, task) })
	keeper := home.NewLive(homes, func() config.HomeRules { return set.Values().Home }, s, log.With("part", "home"), nil)
	brain := chat.New(chat.Options{
		Home:      keeper,
		AgentName: cfg.AgentName,
		Live: func() chat.Live {
			v := set.Values()
			return chat.Live{Owners: v.Owners, Nicknames: v.Nicknames, Voice: v.Voice}
		},
		Voice:    voice,
		Cooldown: cooldown(cfg),
		Logger:   log.With("part", "chat"),
	}, s)
	s.SetListener(brain)
	a := &App{Config: cfg, Session: s, Brain: brain, Home: keeper, Settings: set,
		Burden: burden.New(func() bool { return set.Values().Overload.Drop }, s, homes, log.With("part", "burden"), nil)}
	a.Bridge = bridge.NewServer(bridge.Options{
		Token:    cfg.Token,
		Welcome:  func(*protocol.Hello) *protocol.Welcome { return Welcome(cfg, set.Values()) },
		Logger:   log.With("part", "bridge"),
		Fallback: a.control(log.With("part", "control")),
		Tap:      audit.Frame,
	}, s)
	return a
}

// Welcome is the reply to every hello, built from the configuration and
// the runtime settings.
func Welcome(cfg config.Config, v settings.Values) *protocol.Welcome {
	return &protocol.Welcome{
		AgentName:    cfg.AgentName,
		Owners:       slices.Clone(v.Owners),
		Rates:        cfg.Rates,
		Perception:   cfg.Perception,
		AllowedTasks: slices.Clone(v.AllowedTasks),
	}
}

// Serve runs the controller on ln until ctx ends.
func (a *App) Serve(ctx context.Context, ln net.Listener) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go a.Home.Run(ctx)
	go a.Burden.Run(ctx)
	return a.Bridge.Serve(ctx, ln)
}

func cooldown(cfg config.Config) time.Duration {
	if cfg.LLM == nil {
		return 0
	}
	return time.Duration(cfg.LLM.CooldownS * float64(time.Second))
}
