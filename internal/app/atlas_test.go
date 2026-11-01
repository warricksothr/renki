package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/atlas"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func TestSavedAtlasOfflineQueriesAreBoundedAndAuthenticated(t *testing.T) {
	a := New(config.Default(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil)
	path := filepath.Join(t.TempDir(), "atlas.sqlite")
	store, err := atlas.Open(path, atlas.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	scope := protocol.TerrainScope{Namespace: "test", WorldUID: "9223372036854775807", Epoch: "1", Source: "bot", Stream: "run", Game: "game", Generator: 2, Mods: "hash"}
	h, tick, biome := 40.0, 1.0, "Meadows"
	tile := protocol.TerrainTile{X: -1, Z: 0, WaterLevel: 30, Heights: make([]*float64, 25), Biomes: make([]*string, 25), MeasuredAt: make([]*float64, 25)}
	for i := range tile.Heights {
		tile.Heights[i] = &h
		tile.Biomes[i] = &biome
		tile.MeasuredAt[i] = &tick
	}
	if err = store.Ingest(context.Background(), session.TerrainObservation{World: "saved", Scope: scope, CapturedAt: tick, ReceivedAt: time.Now().Add(-time.Minute), Tiles: []protocol.TerrainTile{tile}}); err != nil {
		t.Fatal(err)
	}
	store.Close()
	store, err = atlas.Open(path, atlas.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	a.SetAtlas(store, nil)
	handler := panel.Handler(panel.Options{Token: "test", Status: func() any { return nil }, Atlas: a.AtlasHandler()})
	get := func(path string, auth bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", path, nil)
		if auth {
			req.Header.Set("Authorization", "Bearer test")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	key := atlas.Key(scope)
	url := "/api/atlas/map?world=" + key + "&minX=-16&maxX=0&minZ=0&maxZ=16"
	for _, path := range []string{"/api/atlas/worlds", url} {
		if w := get(path, false); w.Code != http.StatusUnauthorized {
			t.Fatal("unprotected atlas", w.Code)
		}
	}
	var worlds []atlas.World
	w := get("/api/atlas/worlds", true)
	if err = json.Unmarshal(w.Body.Bytes(), &worlds); err != nil || len(worlds) != 1 || worlds[0].Key != key {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	w = get(url, true)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"saved":true`) || !strings.Contains(w.Body.String(), `"connection":0`) || !strings.Contains(w.Body.String(), `"observedAt"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	// No connected game is required to show the selected world's known places.
	now := time.Now()
	pos := protocol.Vec3{500, 40, 200}
	portal := home.Landmark{Entity: "portal", Kind: protocol.StructurePortal, Tag: "mine", At: now,
		Pos: protocol.Vec3{-8, 35, 8}, Portal: &protocol.PortalStatus{Connected: true, TargetID: "remote", TargetPos: &pos}}
	outside := portal
	outside.Entity, outside.Pos = "outside", protocol.Vec3{1000, 35, 1000}
	chest := home.Landmark{Entity: "chest", Kind: protocol.StructureContainer, Pos: protocol.Vec3{-4, 35, 8}, At: now}
	if err := a.Landmarks.Store.RememberLandmarks("saved", []home.Landmark{portal, outside, chest}, now); err != nil {
		t.Fatal(err)
	}
	if err := a.Memory.Store.RememberContainers("saved", []home.RememberedContainer{{Entity: "chest", Size: [2]int{2, 2}, At: now, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 10}}}}, now); err != nil {
		t.Fatal(err)
	}
	portal.Entity = "foreign"
	if err := a.Landmarks.Store.RememberLandmarks("other", []home.Landmark{portal}, now); err != nil {
		t.Fatal(err)
	}
	w = get(url, true)
	var saved struct {
		Landmarks  []home.Landmark            `json:"landmarks"`
		Containers []home.RememberedContainer `json:"containers"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Landmarks) != 2 || saved.Landmarks[1].Entity != "portal" || saved.Landmarks[1].Portal.TargetPos[0] != 500 {
		t.Fatal("saved viewport lost portal knowledge or mixed worlds", w.Body.String())
	}
	if len(saved.Containers) != 1 || saved.Containers[0].Entity != "chest" || saved.Containers[0].Inventory().Count("Wood") != 10 {
		t.Fatal("saved chest lost its shared contents", w.Body.String())
	}
	batch := make([]home.Landmark, 520)
	for i := range batch {
		batch[i] = outside
		batch[i].Entity = strings.Repeat("x", i+1)
		batch[i].Pos = protocol.Vec3{-8, 35, 8}
	}
	if err := a.Landmarks.Store.RememberLandmarks("saved", batch, now); err != nil {
		t.Fatal(err)
	}
	w = get(url, true)
	if err := json.Unmarshal(w.Body.Bytes(), &saved); err != nil || len(saved.Landmarks) != 512 {
		t.Fatal("unbounded saved landmarks", err, len(saved.Landmarks))
	}
	for _, query := range []string{"&minX=0&maxX=10000&minZ=0&maxZ=1", "&minX=NaN&maxX=1&minZ=0&maxZ=1", "&minX=0&maxX=1&minZ=0"} {
		if w := get("/api/atlas/map?world="+key+query, true); w.Code != 400 {
			t.Fatal("unbounded/invalid request accepted", w.Code)
		}
	}
	if w := get("/api/atlas/map?world="+strings.Repeat("b", 64)+"&minX=0&maxX=1&minZ=0&maxZ=1", true); w.Code != 404 {
		t.Fatal("unknown atlas", w.Code)
	}
	if w := get("/api/atlas/map?world=live", true); w.Code != 200 || !strings.Contains(w.Body.String(), `"tiles":[]`) {
		t.Fatal("offline map presented as live", w.Code, w.Body.String())
	}
}
