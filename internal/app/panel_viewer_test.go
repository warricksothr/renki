package app

import (
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

func TestViewerProjectionKeepsMapScopeAndOmitsPrivateState(t *testing.T) {
	cfg := config.Default()
	cfg.Owners = []string{"Private owner"}
	a := New(cfg, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil)
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome { return Welcome(cfg, a.Settings.Values()) }}, a.Session)
	p := bridgetest.Connect(t, url, []string{protocol.TaskSay})
	p.Send(&protocol.State{Envelope: protocol.Envelope{T: 100}, World: "one", WorldUID: "123", Character: "Tas", Phase: protocol.PhaseInWorld, Pos: protocol.Vec3{1, 2, 3}, HP: 25})
	waitFor(t, func() bool { return a.Session.State() != nil })
	b, err := json.Marshal(a.PanelViewerStatus())
	if err != nil {
		t.Fatal(err)
	}
	var status map[string]any
	if err = json.Unmarshal(b, &status); err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{}
	for _, key := range []string{"connected", "character", "phase", "connection", "perceptionRadius", "world", "worldUid", "yaw", "pos", "biome", "location", "flags", "time"} {
		allowed[key] = true
	}
	for key := range status {
		if !allowed[key] {
			t.Fatalf("viewer exposed unexpected status: %s", key)
		}
	}
	if status["worldUid"] != "123" || status["character"] != "Tas" || status["connection"] == float64(0) {
		t.Fatalf("viewer lost media/map scope: %s", b)
	}
	now := time.Now()
	if err := a.Landmarks.Store.RememberLandmarks("one", []home.Landmark{{Entity: "chest", Kind: protocol.StructureContainer, Pos: protocol.Vec3{1, 2, 3}, At: now}}, now); err != nil {
		t.Fatal(err)
	}
	if err := a.Memory.Store.RememberContainers("one", []home.RememberedContainer{{Entity: "chest", Size: [2]int{2, 2}, Items: []protocol.InventoryItem{{Prefab: "SecretMeal", Stack: 10}}, At: now}}, now); err != nil {
		t.Fatal(err)
	}
	operator := a.MapSnapshot().(MapSnapshot)
	viewer := a.ViewerMapSnapshot().(MapSnapshot)
	if len(operator.Containers) != 1 || len(viewer.Containers) != 0 || len(viewer.Landmarks) != 1 {
		t.Fatalf("map contents/location filtering: %+v", viewer)
	}
}
