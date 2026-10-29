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

type listener struct{}

func (listener) Chat(*protocol.Chat)    {}
func (listener) Event(*protocol.Event)  {}
func (listener) TaskDone(*session.Task) {}
func wait(t *testing.T, f func() bool) {
	t.Helper()
	until := time.Now().Add(2 * time.Second)
	for !f() {
		if time.Now().After(until) {
			t.Fatal("timed out")
		}
		time.Sleep(time.Millisecond)
	}
}
func TestRecorderScopesFramesAndPreservesArrivalTimeAcrossRetries(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := session.New(listener{}, log)
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome { return &protocol.Welcome{} }}, s)
	p := bridgetest.Connect(t, url, nil)
	folder := filepath.Join(t.TempDir(), "data")
	path := filepath.Join(folder, "memory.json")
	store, err := home.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	r := Containers{Session: s, Store: store, Logger: log}
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one", HP: 25})
	p.Send(&protocol.Perception{Structures: []protocol.PerceivedStructure{{ID: "chest", Kind: protocol.StructureContainer, Prefab: "piece_chest_wood"}}})
	p.Send(&protocol.Container{Entity: "chest", Size: [2]int{2, 2}, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 5}}})
	wait(t, func() bool { return len(s.Containers()) == 1 })
	r.Tick()
	first := store.RememberedContainers("one")[0]
	if first.Prefab != "piece_chest_wood" || first.Inventory().Count("Wood") != 5 {
		t.Fatal(first)
	}
	// No newly received frame means no save and no refreshed observation time.
	r.Tick()
	if !store.RememberedContainers("one")[0].At.Equal(first.At) {
		t.Fatal("old frame marked fresh")
	}
	backup := folder + "-saved"
	if err = os.Rename(folder, backup); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(folder, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	p.Send(&protocol.Container{Entity: "chest", Size: [2]int{2, 2}})
	wait(t, func() bool { return len(s.Containers()[0].Items) == 0 })
	_, _, _, observed := s.ContainerObservations()
	arrival := observed[0].At
	r.Tick()
	if store.RememberedContainers("one")[0].Inventory().Count("Wood") != 5 {
		t.Fatal("failed save changed memory")
	}
	if err = os.Remove(folder); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(backup, folder); err != nil {
		t.Fatal(err)
	}
	r.Tick()
	current := store.RememberedContainers("one")[0]
	if len(current.Items) != 0 || !current.At.Equal(arrival) {
		t.Fatal("retry changed arrival time", current)
	}
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "two", HP: 25})
	wait(t, func() bool { return s.State().World == "two" })
	r.Tick()
	if len(store.RememberedContainers("two")) != 0 {
		t.Fatal("old-world frames leaked")
	}
	p.Send(&protocol.Container{Entity: "chest", Size: [2]int{1, 1}, Items: []protocol.InventoryItem{{Prefab: "Stone", Stack: 8}}})
	wait(t, func() bool { return len(s.Containers()) == 1 })
	r.Tick()
	if store.RememberedContainers("two")[0].Inventory().Count("Stone") != 8 || len(store.RememberedContainers("one")[0].Items) != 0 {
		t.Fatal("world isolation failed")
	}
	reloaded, err := home.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.RememberedContainers("two")[0].Inventory().Count("Stone") != 8 {
		t.Fatal("not persisted")
	}
}
