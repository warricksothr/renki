package app

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/warricksothr/renki/internal/access"
	"github.com/warricksothr/renki/internal/audit"
	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/chat"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/goal"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
	"github.com/warricksothr/renki/internal/settings"
	"github.com/warricksothr/renki/internal/video"
)

// PanelStatus is the panel's snapshot of the bot (GET /api/status).
type PanelStatus struct {
	SharePosition *bool                  `json:"sharePosition,omitempty"`
	Presence      config.Presence        `json:"presence"`
	Players       access.Rules           `json:"players"`
	Goals         []goal.Record          `json:"goals"`
	Version       string                 `json:"version"`
	Connected     bool                   `json:"connected"`
	Character     string                 `json:"character,omitempty"`
	World         string                 `json:"world,omitempty"`
	WorldUID      string                 `json:"worldUid,omitempty"`
	Phase         string                 `json:"phase,omitempty"`
	Task          *protocol.TaskRef      `json:"task,omitempty"`
	HP            float64                `json:"hp,omitempty"`
	HPMax         float64                `json:"hpMax,omitempty"`
	InjurySeq     *uint64                `json:"injurySeq,omitempty"`
	Food          *protocol.FoodStatus   `json:"food,omitempty"`
	Combat        *protocol.CombatStatus `json:"combat,omitempty"`
	Stamina       float64                `json:"stamina,omitempty"`
	StaminaMax    float64                `json:"staminaMax,omitempty"`
	PVP           *bool                  `json:"pvp,omitempty"`
	Pos           *protocol.Vec3         `json:"pos,omitempty"`
	Biome         string                 `json:"biome,omitempty"`
	Location      *protocol.Location     `json:"location,omitempty"`
	Flags         []string               `json:"flags,omitempty"`
	Yaw           float64                `json:"yaw"`
	Connection    uint64                 `json:"connection"`
	Home          *PanelHome             `json:"home,omitempty"`
	// PerceptionRadius is how far perception reaches, for the map's first view.
	PerceptionRadius float64 `json:"perceptionRadius"`
	// LeftBehind is what the bot had to leave on the ground in this world.
	LeftBehind    []home.Cache            `json:"leftBehind,omitempty"`
	StorageNeeded []behaviour.StorageNeed `json:"storageNeeded,omitempty"`

	AgentName string           `json:"agentName"`
	Nicknames []string         `json:"nicknames"`
	Owners    []string         `json:"owners"`
	Rules     config.HomeRules `json:"homeRules"`
	// Model is the voice's model now; Voice is false when the bot answers
	// the command grammar only.
	Voice bool           `json:"voice"`
	Model settings.Model `json:"model"`
}

// PanelHome is the home in the bot's current world.
type PanelHome struct {
	World    string        `json:"world"`
	Pos      protocol.Vec3 `json:"pos"`
	SetBy    string        `json:"setBy"`
	At       time.Time     `json:"at"`
	Distance *float64      `json:"distance,omitempty"`
}

// PanelStatus takes the snapshot.
func (a *App) PanelStatus(version string) PanelStatus {
	st := a.status()
	ps := PanelStatus{Version: version, Connected: st.Connected, Character: st.Character, Phase: st.Phase, Task: st.Task,
		AgentName: a.Config.AgentName, Nicknames: a.Settings.Values().Nicknames,
		Owners: a.Settings.Values().Owners, Rules: a.Home.Rules(),
		PerceptionRadius: a.Config.Perception.Radius,
		Voice:            a.Voice != nil && a.Settings.Values().Voice, Model: a.Settings.Values().Model}
	if s := a.Session.State(); s != nil && st.Connected {
		ps.World, ps.HP, ps.HPMax, ps.Biome = s.World, s.HP, s.HPMax, s.Biome
		ps.WorldUID = s.WorldUID
		ps.InjurySeq = a.Session.InjurySequence()
		ps.Location, ps.Flags, ps.Yaw = s.Location, s.Flags, s.Yaw
		ps.Food = s.Food
		ps.Combat, ps.Stamina, ps.StaminaMax = s.Combat, s.Stamina, s.StaminaMax
		ps.PVP = s.PVP
		ps.SharePosition = s.SharePosition
		ps.Connection = a.Session.ConnectionID()
		if s.Phase == protocol.PhaseInWorld {
			pos := s.Pos
			ps.Pos = &pos
		}
	}
	ps.Goals = a.Goals.Summaries()
	ps.Presence, ps.Players = a.Settings.Values().Presence, a.Settings.Values().Players
	ps.LeftBehind = a.Home.LeftBehind()
	if ps.Connected && ps.Phase == protocol.PhaseInWorld {
		for _, instance := range a.Behaviours.List() {
			if need := instance.StorageNeeded; need != nil && need.World == ps.World {
				ps.StorageNeeded = append(ps.StorageNeeded, *need)
			}
		}
	}
	if world, h, ok := a.Home.Current(); ok {
		ps.Home = &PanelHome{World: world, Pos: h.Pos, SetBy: h.SetBy, At: h.At}
		if d, ok := a.Home.HomeDistance(); ok {
			ps.Home.Distance = &d
		}
	}
	return ps
}

// panelSay passes the panel's words to the brain while the plugin is
// connected; the state the brain checks outlives a disconnect.
func (a *App) panelSay(as, text string) error {
	if _, _, ok := a.Session.Handshake(); !ok {
		return chat.ErrNoBot
	}
	return a.Brain.FromPanel(as, text)
}

// PanelTask follows the same direct-order semantics as the panel buttons.
func (a *App) PanelTask(ctx context.Context, command panel.TaskCommand) (string, error) {
	if err := command.Validate(); err != nil {
		return "", err
	}
	a.Home.Touch()
	a.Home.Ordered()
	task, err := a.Session.Dispatch(ctx, session.Order{Task: command.Task, Args: command.Args, TimeoutS: command.TimeoutS, Origin: "panel"})
	if err != nil {
		return "", err
	}
	a.Home.Told()
	return task.ID, nil
}

// models is the voice's model list, or nil when the voice cannot list them.
func (a *App) models() func(context.Context) ([]string, error) {
	if l, ok := a.Voice.(interface {
		Models(context.Context) ([]string, error)
	}); ok {
		return l.Models
	}
	return nil
}

// PanelAct does what the panel's buttons ask: stop, go home or log out.
// Errors are worded for the page.
func (a *App) PanelAct(ctx context.Context, action string) (string, error) {
	var err error
	if action == "stop" || action == "home" {
		// An order, as from chat: it resets the idle clock and wins over
		// whatever a death left pending (internal/home).
		a.Home.Touch()
		a.Home.Ordered()
	}
	switch action {
	case "stop":
		if _, err = a.Session.Dispatch(ctx, session.Order{Task: protocol.TaskStop}); err == nil {
			a.Home.Told()
			return "Stopped.", nil
		}
	case "home":
		if _, err = a.Home.GoHome(ctx, nil, "the panel"); err == nil {
			a.Home.Told()
			return "Heading home.", nil
		}
	case "logout":
		if err = a.Session.Logout(ctx); err == nil {
			return "Logging out.", nil
		}
	default:
		return "", panel.ErrUnknownAction
	}
	switch {
	case errors.Is(err, session.ErrNotConnected):
		return "", errors.New("the bot is not connected")
	case errors.Is(err, session.ErrNotAllowed):
		return "", errors.New("that task is not in allowedTasks")
	case errors.Is(err, home.ErrNoHome):
		return "", errors.New(`no home in this world yet; say "set home" in the game`)
	case errors.Is(err, home.ErrNoWorld):
		return "", errors.New("the bot is not in a world")
	case errors.Is(err, home.ErrStranded):
		return "", errors.New("the bot carries too much to move")
	}
	return "", err
}

// panelRetry is how long ServePanel waits before trying to listen again.
const panelRetry = 30 * time.Second

// ServePanel serves the control panel on cfg.Panel.Listen until ctx ends.
// It never stops the controller: while it cannot listen, as when the
// Tailscale address is not up yet, it logs that and tries again.
func (a *App) ServePanel(ctx context.Context, trail *audit.Log, version string, log *slog.Logger) {
	addr := a.Config.Panel.Listen
	if addr == "" {
		return
	}
	var ln net.Listener
	for {
		var err error
		if ln, err = net.Listen("tcp", addr); err == nil {
			break
		}
		log.Error("panel cannot listen yet; trying again", "addr", addr, "err", err, "in", panelRetry)
		select {
		case <-ctx.Done():
			return
		case <-time.After(panelRetry):
		}
	}
	srv := &http.Server{
		Handler: panel.Handler(panel.Options{
			Token:           a.Config.Panel.Token,
			Log:             trail,
			Status:          func() any { return a.PanelStatus(version) },
			Act:             a.PanelAct,
			Settings:        a.Settings,
			Models:          a.models(),
			AuditDir:        a.Config.Audit.Dir,
			Say:             a.panelSay,
			Dispatch:        a.PanelTask,
			Behaviours:      a.BehaviourHandler(),
			Goals:           a.GoalHandler(),
			ContainerMemory: a.ContainerMemory,
			LandmarkMemory:  a.LandmarkMemory,
			PlayerMemory:    a.PlayerMemory,
			Map:             a.MapSnapshot,
			Camera:          a.CameraHandler(),
			Video: video.New(a.Config.Panel.Video, func() video.Scope {
				s := a.PanelStatus(version)
				return video.Scope{Online: s.Connected && s.Phase == "in_world", Connection: s.Connection, World: s.World, WorldUID: s.WorldUID}
			}),
			Atlas:     a.AtlasHandler(),
			Inventory: func() any { return a.Session.PackSnapshot() },
			Waypoints: a.WaypointHandler(),
			Logger:    log,
		}),
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	log.Info("panel listening", "addr", ln.Addr().String())
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("panel stopped", "err", err)
	}
}
