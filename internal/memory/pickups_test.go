package memory

import (
	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
	"os"
	"path/filepath"
	"testing"
)

func TestPickupRecorderPersistsOriginalScopeAndRetriesWithoutDuplication(t *testing.T) {
	s := session.New(listener{}, nil)
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome { return &protocol.Welcome{} }}, s)
	p := bridgetest.Connect(t, url, nil)
	dir := filepath.Join(t.TempDir(), "data")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "memory.json")
	store, err := home.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	r := Containers{Session: s, Store: store}
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one", WorldUID: "uid", HP: 50})
	pos := protocol.Vec3{1, 2, 3}
	p.Send(&protocol.Event{Event: protocol.EventItemPickedUp, World: "one", WorldUID: "uid", Prefab: "Coal", Count: 4, Pos: &pos})
	wait(t, func() bool { _, got := s.PickupObservations(); return len(got) == 1 })
	_, events := s.PickupObservations()
	at := events[0].At
	// Save may fail, but must retry using the arrival scope/time, not later state.
	if err = os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(dir, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	r.Tick()
	if len(store.Pickups("one")) != 0 {
		t.Fatal("failed save published")
	}
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "two", WorldUID: "uid-two", HP: 50})
	wait(t, func() bool { return s.State().World == "two" })
	if err = os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	r.Tick()
	records := store.Pickups("one")
	if len(records) != 1 || records[0].Count != 4 || records[0].Pos != pos || !records[0].At.Equal(at) || len(store.Pickups("two")) != 0 {
		t.Fatal(records)
	}
	freshWorker := Containers{Session: s, Store: store}
	freshWorker.Tick()
	reloaded, err := home.Open(path)
	if err != nil || len(reloaded.Pickups("one")) != 1 {
		t.Fatal("replayed worker duplicated pickup", err)
	}
}
