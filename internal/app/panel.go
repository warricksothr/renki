package app

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/warricksothr/renki/internal/audit"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
)

// PanelStatus is the panel's snapshot of the bot (GET /api/status).
type PanelStatus struct {
	Version   string            `json:"version"`
	Connected bool              `json:"connected"`
	Character string            `json:"character,omitempty"`
	World     string            `json:"world,omitempty"`
	Phase     string            `json:"phase,omitempty"`
	Task      *protocol.TaskRef `json:"task,omitempty"`
	HP        float64           `json:"hp,omitempty"`
	HPMax     float64           `json:"hpMax,omitempty"`
	Pos       *protocol.Vec3    `json:"pos,omitempty"`
	Biome     string            `json:"biome,omitempty"`
	Home      *PanelHome        `json:"home,omitempty"`
	// PerceptionRadius is how far perception reaches, for the map's first view.
	PerceptionRadius float64 `json:"perceptionRadius"`
	// LeftBehind is what the bot had to leave on the ground in this world.
	LeftBehind []home.Cache `json:"leftBehind,omitempty"`

	AgentName string           `json:"agentName"`
	Nicknames []string         `json:"nicknames"`
	Owners    []string         `json:"owners"`
	Rules     config.HomeRules `json:"homeRules"`
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
		PerceptionRadius: a.Config.Perception.Radius}
	if s := a.Session.State(); s != nil && st.Connected {
		ps.World, ps.HP, ps.HPMax, ps.Biome = s.World, s.HP, s.HPMax, s.Biome
		if s.Phase == protocol.PhaseInWorld {
			pos := s.Pos
			ps.Pos = &pos
		}
	}
	ps.LeftBehind = a.Home.LeftBehind()
	if world, h, ok := a.Home.Current(); ok {
		ps.Home = &PanelHome{World: world, Pos: h.Pos, SetBy: h.SetBy, At: h.At}
		if d, ok := a.Home.HomeDistance(); ok {
			ps.Home.Distance = &d
		}
	}
	return ps
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
			Token:  a.Config.Panel.Token,
			Log:    trail,
			Status: func() any { return a.PanelStatus(version) },
			Logger: log,
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
