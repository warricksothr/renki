package behaviour

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/piles"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
	"github.com/warricksothr/renki/internal/settings"
)

func TestViewAsksWhetherAPileIsNearAnOwner(t *testing.T) {
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	store, err := settings.Open(filepath.Join(t.TempDir(), "settings.json"), config.Default(), quiet, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := session.New(listener{}, quiet)
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome {
		return &protocol.Welcome{AllowedTasks: []string{"stop"}, Perception: protocol.PerceptionConfig{Radius: 40, MaxPerKind: 24}}
	}}, s)
	p := bridgetest.Connect(t, url, []string{"stop"})
	// Perception arrival is stamped by the wall clock, so the engine's is too.
	e := New(Options{Session: s, Store: store, Now: time.Now, Logger: quiet})
	state := func(world string, x, t float64) {
		p.Send(&protocol.State{Envelope: protocol.Envelope{T: t}, Phase: protocol.PhaseInWorld, World: world, HP: 25, HPMax: 25, Pos: protocol.Vec3{x, 0, 0}})
	}
	see := func(t float64, drops ...protocol.PerceivedResource) {
		p.Send(&protocol.Perception{Envelope: protocol.Envelope{T: t}, Resources: drops})
	}
	wood := protocol.PerceivedResource{ID: "a", Kind: protocol.ResourceDrop, Prefab: "Wood", Pos: protocol.Vec3{6, 0, 6}, Stack: 4}
	owner := protocol.Vec3{5, 0, 5}
	near := func() piles.Nearby {
		e.mu.Lock()
		defer e.mu.Unlock()
		v := e.view()
		return v.Piles.Near(v.Now, owner, 8, func(prefab string) bool { return prefab == "Wood" })
	}

	state("test", 0, 10)
	see(10, wood)
	wait(t, func() bool { return near().State == piles.Present })
	if got := e.view().Piles.Piles(); len(got) != 1 || got[0].Total != 4 || got[0].Trend != piles.Rising && got[0].Trend != piles.Steady {
		t.Fatalf("piles = %+v", got)
	}
	state("test", 0, 11)
	see(11)
	wait(t, func() bool { return near().State == piles.Absent })

	// A perception frame far behind state teaches nothing: the pile in it
	// does not come back.
	state("test", 0, 30)
	wait(t, func() bool { return e.view().State.T == 30 })
	see(13, wood)
	wait(t, func() bool { return e.view().Perception != nil && e.view().Perception.T == 13 })
	if n := near(); n.State != piles.Absent {
		t.Fatalf("a stale frame answered %+v", n)
	}

	// Walking 40 m off leaves part of the owner's circle out of sight.
	state("test", 40, 32)
	see(32)
	wait(t, func() bool { return near().State == piles.Unseen })

	// Another world starts with nothing known.
	state("another", 0, 33)
	wait(t, func() bool { return e.view().State.World == "another" })
	if got := e.view().Piles.Piles(); len(got) != 0 {
		t.Fatalf("piles crossed worlds: %+v", got)
	}
}
