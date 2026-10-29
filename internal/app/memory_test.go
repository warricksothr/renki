package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
)

func TestServeWaitsForFinalLandmarkBatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	store, err := home.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	a := New(config.Default(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, store, nil)
	c := &bridge.Conn{Hello: &protocol.Hello{}, Welcome: &protocol.Welcome{}}
	a.Session.Connected(c)
	a.Session.Message(c, &protocol.State{Phase: protocol.PhaseInWorld, World: "world", HP: 25})
	a.Session.Message(c, &protocol.Perception{Structures: []protocol.PerceivedStructure{{ID: "chest", Kind: protocol.StructureContainer}}})
	a.Landmarks.Tick()
	if len(store.Landmarks("world")) != 0 {
		t.Fatal("batch committed early")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Serve(ctx, ln); err != nil {
		t.Fatal(err)
	}
	// No polling: Serve must not return before durable persistence finishes.
	reopened, err := home.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Landmarks("world"); len(got) != 1 || got[0].Entity != "chest" {
		t.Fatal(got)
	}
}

func TestMemoryAPIsAreAuthenticatedAndReportAge(t *testing.T) {
	store := home.Memory()
	now := time.Now().Add(-time.Minute)
	if err := store.RememberContainers("world", []home.RememberedContainer{{Entity: "chest", Size: [2]int{2, 2}, At: now, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 10}}}}, time.Now()); err != nil {
		t.Fatal(err)
	}
	place := home.Landmark{Entity: "portal", Kind: protocol.StructurePortal, Tag: "mine", At: now}
	if err := store.RememberLandmarks("world", []home.Landmark{place}, time.Now()); err != nil {
		t.Fatal(err)
	}
	place.Entity = "other-world"
	if err := store.RememberLandmarks("other", []home.Landmark{place}, time.Now()); err != nil {
		t.Fatal(err)
	}
	a := New(config.Default(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, store, nil)
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome { return &protocol.Welcome{} }}, a.Session)
	p := bridgetest.Connect(t, url, nil)
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "world", HP: 25})
	until := time.Now().Add(2 * time.Second)
	for a.Session.State() == nil {
		if time.Now().After(until) {
			t.Fatal("state not received")
		}
		time.Sleep(time.Millisecond)
	}
	height, biome := 32.0, "Meadows"
	tile := protocol.TerrainTile{X: -3, Z: 0, WaterLevel: 30, Heights: make([]*float64, 25), Biomes: make([]*string, 25)}
	for i := 1; i < 25; i++ {
		tile.Heights[i], tile.Biomes[i] = &height, &biome
	}
	p.Send(&protocol.Terrain{World: "world", Tiles: []protocol.TerrainTile{tile}})
	for {
		_, _, tiles := a.Session.TerrainSnapshot()
		if len(tiles) == 1 {
			break
		}
		if time.Now().After(until) {
			t.Fatal("terrain not received")
		}
		time.Sleep(time.Millisecond)
	}
	handler := panel.Handler(panel.Options{Token: "test", Status: func() any { return nil }, ContainerMemory: a.ContainerMemory, LandmarkMemory: a.LandmarkMemory, Map: a.MapSnapshot})
	get := func(path string, auth bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		if auth {
			r.Header.Set("Authorization", "Bearer test")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	for _, path := range []string{"/api/memory/containers", "/api/memory/landmarks", "/api/map"} {
		if w := get(path, false); w.Code != http.StatusUnauthorized {
			t.Fatal(path, w.Code)
		}
	}
	w := get("/api/memory/containers", true)
	if w.Code != http.StatusOK {
		t.Fatal(w.Code)
	}
	var result ContainerMemoryStatus
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.World != "world" || len(result.Containers) != 1 || result.Containers[0].AgeS < 60 || result.Containers[0].Inventory().Count("Wood") != 10 {
		t.Fatal(result)
	}
	w = get("/api/memory/landmarks", true)
	var landmarks LandmarkMemoryStatus
	if w.Code != http.StatusOK {
		t.Fatal(w.Code)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &landmarks); err != nil {
		t.Fatal(err)
	}
	if landmarks.World != "world" || len(landmarks.Landmarks) != 1 || landmarks.Landmarks[0].Entity != "portal" || landmarks.Landmarks[0].AgeS < 60 || landmarks.Landmarks[0].Tag != "mine" {
		t.Fatal(landmarks)
	}
	w = get("/api/map", true)
	var mapped MapSnapshot
	if err := json.Unmarshal(w.Body.Bytes(), &mapped); err != nil {
		t.Fatal(err)
	}
	if mapped.World != "world" || mapped.Connection == 0 || len(mapped.Landmarks) != 1 || len(mapped.Tiles) != 1 || mapped.Tiles[0].X != -3 || mapped.Tiles[0].Heights[0] != nil || *mapped.Tiles[0].Heights[1] != 32 {
		t.Fatal(mapped)
	}
}
