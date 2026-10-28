// Package app wires the controller together: the bridge, the session and the
// chat brain, from a configuration.
package app

import (
	"context"
	"log/slog"
	"net"
	"slices"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/chat"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// App is a configured controller.
type App struct {
	Config  config.Config
	Session *session.Session
	Brain   *chat.Brain
	Bridge  *bridge.Server
}

// New builds a controller from cfg. voice may be nil, which leaves the bot
// with the command grammar only.
func New(cfg config.Config, voice chat.Voice, log *slog.Logger) *App {
	if log == nil {
		log = slog.Default()
	}
	s := session.New(nil, log.With("part", "session"))
	brain := chat.New(chat.Options{
		AgentName: cfg.AgentName,
		Nicknames: cfg.Nicknames,
		Owners:    cfg.Owners,
		Voice:     voice,
		Cooldown:  cooldown(cfg),
		Logger:    log.With("part", "chat"),
	}, s)
	s.SetListener(brain)
	a := &App{Config: cfg, Session: s, Brain: brain}
	a.Bridge = bridge.NewServer(bridge.Options{
		Token:    cfg.Token,
		Welcome:  func(*protocol.Hello) *protocol.Welcome { return Welcome(cfg) },
		Logger:   log.With("part", "bridge"),
		Fallback: a.control(log.With("part", "control")),
	}, s)
	return a
}

// Welcome is the reply to every hello, built from configuration.
func Welcome(cfg config.Config) *protocol.Welcome {
	return &protocol.Welcome{
		AgentName:    cfg.AgentName,
		Owners:       slices.Clone(cfg.Owners),
		Rates:        cfg.Rates,
		Perception:   cfg.Perception,
		AllowedTasks: slices.Clone(cfg.AllowedTasks),
	}
}

// Serve runs the controller on ln until ctx ends.
func (a *App) Serve(ctx context.Context, ln net.Listener) error {
	return a.Bridge.Serve(ctx, ln)
}

func cooldown(cfg config.Config) time.Duration {
	if cfg.LLM == nil {
		return 0
	}
	return time.Duration(cfg.LLM.CooldownS * float64(time.Second))
}
