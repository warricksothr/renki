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
	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/burden"
	"github.com/warricksothr/renki/internal/chat"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/food"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/memory"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/retreat"
	"github.com/warricksothr/renki/internal/session"
	"github.com/warricksothr/renki/internal/settings"
)

// App is a configured controller.
type App struct {
	Config     config.Config
	Session    *session.Session
	Brain      *chat.Brain
	Home       *home.Keeper
	Burden     *burden.Unloader
	Retreat    *retreat.Retreat
	Food       *food.Keeper
	Behaviours *behaviour.Engine
	Memory     *memory.Containers
	Landmarks  *memory.Landmarks
	Danger     *memory.Danger
	Bridge     *bridge.Server
	// Settings are what can change while the bot runs: owners, nicknames,
	// allowedTasks, the voice switch, the home rules and overload.drop.
	Settings *settings.Store
	// Voice is the LLM's voice, or nil with the command grammar only.
	Voice chat.Voice
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
	s.SetMovePlace(func(world, name string) (protocol.Vec3, error) {
		if place, ok := homes.Waypoint(world, name); ok {
			return place.Pos, nil
		}
		return protocol.Vec3{}, home.ErrNoWaypoint
	})
	brain := chat.New(chat.Options{
		Home:      keeper,
		Places:    homes,
		AgentName: cfg.AgentName,
		Live: func() chat.Live {
			v := set.Values()
			return chat.Live{Owners: v.Owners, Nicknames: v.Nicknames, Voice: v.Voice}
		},
		Voice:    voice,
		Cooldown: cooldown(cfg),
		Logger:   log.With("part", "chat"),
	}, s)
	a := &App{Config: cfg, Session: s, Brain: brain, Home: keeper, Settings: set, Voice: voice,
		Burden: burden.New(func() bool { return set.Values().Overload.Drop }, s, homes, log.With("part", "burden"), nil)}
	a.Food = food.New(s, func() []string { return set.Values().Owners }, log.With("part", "food"), nil)
	a.Retreat = retreat.New(s, func() []string { return set.Values().Owners }, func() (protocol.Vec3, bool) {
		_, h, ok := keeper.Current()
		return h.Pos, ok
	}, log.With("part", "retreat"), nil)
	a.Behaviours = behaviour.New(behaviour.Options{Session: s, Store: set,
		Owners:     func() []string { return set.Values().Owners },
		Home:       func() (protocol.Vec3, bool) { _, h, ok := keeper.Current(); return h.Pos, ok },
		Containers: homes.RememberedContainers,
		Danger:     homes.DangerZones,
		Notify: func(ctx context.Context, line string) {
			_, err := s.Dispatch(ctx, session.Order{Task: protocol.TaskSay, Args: &protocol.SayArgs{Text: line}, TimeoutS: 5})
			if err != nil {
				log.Warn("could not report storage needed", "err", err)
			}
		},
		Logger: log.With("part", "behaviour")})
	a.Memory = &memory.Containers{Session: s, Store: homes, Logger: log.With("part", "memory")}
	a.Landmarks = &memory.Landmarks{Session: s, Store: homes, Logger: log.With("part", "memory")}
	a.Danger = &memory.Danger{Store: homes, Logger: log.With("part", "memory")}
	s.SetListener(&listeners{brain: brain, retreat: a.Retreat, behaviours: a.Behaviours, danger: a.Danger})
	a.Bridge = bridge.NewServer(bridge.Options{
		Token:    cfg.Token,
		Welcome:  func(*protocol.Hello) *protocol.Welcome { return Welcome(cfg, set.Values()) },
		Logger:   log.With("part", "bridge"),
		Fallback: a.control(log.With("part", "control")),
		Tap:      audit.Frame,
	}, s)
	return a
}

// listeners fan out session events without doing blocking work on the bridge.
type listeners struct {
	brain      *chat.Brain
	retreat    *retreat.Retreat
	behaviours *behaviour.Engine
	danger     *memory.Danger
}

func (l *listeners) Chat(c *protocol.Chat)    { l.brain.Chat(c) }
func (l *listeners) TaskDone(t *session.Task) { l.brain.TaskDone(t) }
func (l *listeners) Injury(e *protocol.Event, observation session.InjuryObservation) {
	l.danger.Observe(e, observation)
}
func (l *listeners) Event(e *protocol.Event) {
	l.retreat.Event(e)
	l.behaviours.Event(e.Event)
	l.brain.Event(e)
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
	landmarksDone := make(chan struct{})
	dangerDone := make(chan struct{})
	defer func() {
		cancel()
		// Let the recorder persist its final batch before main can exit.
		<-landmarksDone
		<-dangerDone
	}()
	go a.Home.Run(ctx)
	go a.Burden.Run(ctx)
	go a.Retreat.Run(ctx)
	go a.Food.Run(ctx)
	go a.discoverWaypoints(ctx)
	go a.Behaviours.Run(ctx)
	go a.Memory.Run(ctx)
	go func() { defer close(dangerDone); a.Danger.Run(ctx) }()
	go func() {
		defer close(landmarksDone)
		a.Landmarks.Run(ctx)
	}()
	return a.Bridge.Serve(ctx, ln)
}

func cooldown(cfg config.Config) time.Duration {
	if cfg.LLM == nil {
		return 0
	}
	return time.Duration(cfg.LLM.CooldownS * float64(time.Second))
}
