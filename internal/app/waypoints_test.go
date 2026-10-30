package app

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
)

func TestWaypointAPIAuthenticationEditingConflictAndOfflineMovement(t *testing.T) {
	a := New(config.Default(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil)
	handler := panel.Handler(panel.Options{Token: "test", Status: func() any { return nil }, Waypoints: a.WaypointHandler()})
	call := func(method, path string, body any, token, act bool) *httptest.ResponseRecorder {
		raw, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(raw))
		if token {
			r.Header.Set("Authorization", "Bearer test")
		}
		if act {
			r.Header.Set(panel.ActHeader, "1")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := call("GET", "/api/waypoints", nil, false, false); w.Code != 401 {
		t.Fatal(w.Code)
	}
	create := WaypointCommand{Action: "set", World: "one", Waypoint: &home.Waypoint{Name: "storage", Kind: "storage", Pos: protocol.Vec3{1, 36, 3}, Radius: 8}}
	if w := call("POST", "/api/waypoints", create, true, false); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := call("POST", "/api/waypoints", create, true, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	old, _ := a.Memory.Store.Waypoint("one", "storage")
	next := old
	next.Name = "east store"
	next.Radius = 12
	change := WaypointCommand{Action: "set", World: "one", Name: "storage", Waypoint: &next, ExpectedAt: &old.At}
	if w := call("POST", "/api/waypoints", change, true, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	current, _ := a.Memory.Store.Waypoint("one", "east store")
	if current.Source != "owner" || current.SetBy != "panel" || current.Radius != 12 {
		t.Fatal(current)
	}
	stale := WaypointCommand{Action: "delete", World: "one", Name: current.Name, ExpectedAt: &old.At}
	if w := call("POST", "/api/waypoints", stale, true, true); w.Code != 409 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := call("POST", "/api/waypoints", WaypointCommand{Action: "go", World: "one", Name: current.Name}, true, true); w.Code != 400 {
		t.Fatal(w.Code)
	}
	var got struct {
		Waypoints []home.Waypoint `json:"waypoints"`
	}
	w := call("GET", "/api/waypoints?world=two", nil, true, false)
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || len(got.Waypoints) != 0 {
		t.Fatal(got, err)
	}
	if w := call("POST", "/api/waypoints", WaypointCommand{Action: "delete", World: "one", Name: current.Name, ExpectedAt: &current.At}, true, true); w.Code != 200 {
		t.Fatal(w.Code)
	}
}

func TestWaypointContextUsesCurrentMovesAndWorldScopedRememberedContents(t *testing.T) {
	a := New(config.Default(), nil, nil, nil, nil, nil)
	now := time.Now().UTC()
	memory := []home.Landmark{{Entity: "moved", Kind: "container", Pos: protocol.Vec3{2, 36, 0}, At: now}, {Entity: "old-position", Kind: "container", Pos: protocol.Vec3{2, 36, 0}, At: now.Add(-time.Hour)}}
	if err := a.Memory.Store.RememberLandmarks("one", memory, now); err != nil {
		t.Fatal(err)
	}
	if err := a.Memory.Store.RememberContainers("one", []home.RememberedContainer{
		{Entity: "chest", Kind: "container", Pos: protocol.Vec3{1, 36, 0}, Size: [2]int{5, 2}, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 12}}, At: now},
		{Entity: "moved", Kind: "container", Pos: protocol.Vec3{2, 36, 0}, Size: [2]int{5, 2}, At: now},
		{Entity: "old-position", Kind: "container", Pos: protocol.Vec3{50, 36, 0}, Size: [2]int{5, 2}, At: now},
	}, now); err != nil {
		t.Fatal(err)
	}
	c := &bridge.Conn{Hello: &protocol.Hello{}, Welcome: &protocol.Welcome{}}
	a.Session.Connected(c)
	a.Session.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 1}, World: "one", Phase: protocol.PhaseInWorld})
	a.Session.Message(c, &protocol.Perception{Envelope: protocol.Envelope{Seq: 2}, Structures: []protocol.PerceivedStructure{{ID: "moved", Kind: "station", Pos: protocol.Vec3{50, 36, 0}}}, Resources: []protocol.PerceivedResource{{ID: "tree", Kind: "tree", Pos: protocol.Vec3{3, 32, 0}}}})
	out := a.nearWaypoint("one", home.Waypoint{Radius: 10})
	if len(out) != 2 || out[0].Entity != "chest" || out[0].Source != "remembered" || out[0].Contents == nil || out[0].Contents.Items[0].Stack != 12 || out[1].Source != "current" {
		t.Fatal(out)
	}
	if out := a.nearWaypoint("two", home.Waypoint{Radius: 10}); len(out) != 0 {
		t.Fatal(out)
	}
}

func TestStandardPlacesOnlyUseObservedPersistentResourceKinds(t *testing.T) {
	seen := &protocol.Perception{Structures: []protocol.PerceivedStructure{{Kind: "container", Prefab: "piece_chest_wood", Pos: protocol.Vec3{5, 36, 0}}, {Kind: "container", Pos: protocol.Vec3{2, 36, 0}}, {Kind: "station", Prefab: "piece_cookingstation", Pos: protocol.Vec3{3, 36, 0}}}, Resources: []protocol.PerceivedResource{
		{Kind: "stump", Prefab: "Beech_Stub"}, {Kind: "drop", Prefab: "Flint"}, {Kind: "tree", Prefab: "Beech1", Pos: protocol.Vec3{10, 32, 0}},
		{Kind: "pickable", Prefab: "Pickable_Flint", Pos: protocol.Vec3{15, 31, 0}}, {Kind: "pickable", Prefab: "RaspberryBush", Pos: protocol.Vec3{20, 34, 0}},
	}}
	out := standardWaypoints(seen, protocol.Vec3{}, time.Now())
	kinds := map[string]home.Waypoint{}
	for _, w := range out {
		kinds[w.Kind] = w
	}
	if len(kinds) != 5 || kinds["storage"].Pos != (protocol.Vec3{2, 36, 0}) || kinds["flint"].Pos[0] != 15 || kinds["wood"].Pos[0] != 10 || kinds["cooking"].Pos[1] != 36 {
		t.Fatal(out)
	}
}
