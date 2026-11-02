package memory

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func TestLandmarkSelectionKeepsPlacesWithoutTransientStateOrScenery(t *testing.T) {
	open := true
	frame := &protocol.Perception{
		Structures: []protocol.PerceivedStructure{
			{ID: "door", Kind: protocol.StructureDoor, Open: &open},
			{ID: "portal", Kind: protocol.StructurePortal, Tag: "mine"},
			{ID: "grave", Kind: protocol.StructureTombstone},
			{Kind: protocol.StructureBed},
		},
		Resources: []protocol.PerceivedResource{
			{ID: "berry", Kind: protocol.ResourcePickable, Prefab: "RaspberryBush"},
			{ID: "copper", Kind: protocol.ResourceRock, Prefab: "copper_rock"},
			{ID: "silver", Kind: protocol.ResourceRock, Prefab: "silvervein"},
			{ID: "stone", Kind: protocol.ResourceRock, Prefab: "rock4_coast"},
			{ID: "tree", Kind: protocol.ResourceTree, Prefab: "Beech1"},
			{ID: "drop", Kind: protocol.ResourceDrop, Prefab: "CopperOre", Stack: 3},
		},
	}
	got := landmarks(frame, time.Now())
	if len(got) != 6 || got[1].Tag != "mine" || got[5].Entity != "tree" {
		t.Fatal(got)
	}
}

func TestLandmarkRecorderBatchesAndPreservesWorldAndTimeAcrossFailures(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := session.New(listener{}, log)
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome { return &protocol.Welcome{} }}, s)
	p := bridgetest.Connect(t, url, nil)
	folder := filepath.Join(t.TempDir(), "data")
	store, err := home.Open(filepath.Join(folder, "memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := Landmarks{Session: s, Store: store, Logger: log}
	frame := func(id string) *protocol.Perception {
		pos := protocol.Vec3{}
		if id == "b" {
			pos[0] = 10 // Distinct places; identity reconciliation is tested separately.
		}
		return &protocol.Perception{Structures: []protocol.PerceivedStructure{{ID: id, Kind: protocol.StructureContainer, Prefab: "piece_chest_wood", Pos: pos}}}
	}
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one", HP: 25})
	p.Send(frame("a"))
	wait(t, func() bool { _, _, f, _ := s.PerceptionObservation(); return f != nil })
	_, _, _, arrival := s.PerceptionObservation()
	clock := time.Now()
	r.tick(clock)
	if len(store.Landmarks("one")) != 0 {
		t.Fatal("wrote before batch interval")
	}
	p.Send(frame("b"))
	wait(t, func() bool { return s.Perception().Structures[0].ID == "b" })
	r.tick(clock.Add(time.Second))
	// Block the first save; the batch must survive a simultaneous world change.
	if err := os.WriteFile(folder, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	r.tick(clock.Add(landmarkBatchEvery))
	if len(store.Landmarks("one")) != 0 || len(r.pending) != 2 {
		t.Fatal("failed batch was committed or lost")
	}
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "two", HP: 25})
	p.Send(frame("new-world"))
	wait(t, func() bool { _, w, f, _ := s.PerceptionObservation(); return w == "two" && f != nil })
	r.tick(clock.Add(landmarkBatchEvery + time.Second))
	if r.world != "one" {
		t.Fatal("failed old-world batch was relabelled")
	}
	if err := os.Remove(folder); err != nil {
		t.Fatal(err)
	}
	r.tick(clock.Add(landmarkBatchEvery + 2*time.Second))
	old := store.Landmarks("one")
	if len(old) != 2 || old[1].Entity != "a" || !old[1].At.Equal(arrival) {
		t.Fatal("original frame/time lost", old)
	}
	if !r.Flush() || len(store.Landmarks("two")) != 1 || store.Landmarks("two")[0].Entity != "new-world" {
		t.Fatal("new-world batch failed")
	}
	// Repeated ticks with the same frame must not refresh last-seen time.
	r.tick(clock.Add(3 * landmarkBatchEvery))
	if len(r.pending) != 0 {
		t.Fatal("old frame replayed")
	}
}

func TestPerceptionMemoryExcludesPreStateMenuAndReplacedConnections(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := session.New(listener{}, log)
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome { return &protocol.Welcome{} }}, s)
	p := bridgetest.Connect(t, url, nil)
	frame := func() {
		p.Send(&protocol.Perception{Structures: []protocol.PerceivedStructure{{ID: "chest", Kind: protocol.StructureContainer}}})
	}
	assertNoObservation := func() {
		t.Helper()
		if c, _, _, _ := s.PerceptionObservation(); c != nil {
			t.Fatal("unscoped frame accepted")
		}
	}
	frame()
	wait(t, func() bool { return s.Perception() != nil })
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one", HP: 25})
	wait(t, func() bool { return s.State() != nil })
	assertNoObservation()
	frame()
	wait(t, func() bool { c, _, _, _ := s.PerceptionObservation(); return c != nil })
	p.Send(&protocol.State{Phase: protocol.PhaseMenu, World: "one"})
	wait(t, func() bool { return s.State().Phase == protocol.PhaseMenu })
	assertNoObservation()
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one", HP: 25})
	wait(t, func() bool { return s.State().Phase == protocol.PhaseInWorld })
	assertNoObservation()
	frame()
	wait(t, func() bool { c, _, _, _ := s.PerceptionObservation(); return c != nil })
	old, _, _, _ := s.PerceptionObservation()
	p2 := bridgetest.Connect(t, url, nil)
	p2.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one", HP: 25})
	wait(t, func() bool { return s.State() != nil })
	assertNoObservation()
	p2.Send(&protocol.Perception{})
	wait(t, func() bool { c, _, _, _ := s.PerceptionObservation(); return c != nil && c != old })
}
