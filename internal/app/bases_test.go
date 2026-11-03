package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/atlas"
	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/goal"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func TestBaseAPIConfigurationAuthCASAndNoImplicitGameActions(t *testing.T) {
	a := New(config.Default(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil)
	now := time.Now().UTC()
	for _, w := range []home.Waypoint{{Name: "return", Kind: "place", Pos: protocol.Vec3{0, 40, 0}}, {Name: "bedroom", Kind: "rest", Pos: protocol.Vec3{-25, 40, 0}}, {Name: "warehouse", Kind: "storage", Pos: protocol.Vec3{25, 40, 0}, Radius: 8}} {
		w.At = now
		if err := a.Memory.Store.EditWaypoint("one", "", &w, nil); err != nil {
			t.Fatal(err)
		}
	}
	h := panel.Handler(panel.Options{Token: "test", Status: func() any { return nil }, Bases: a.BaseHandler()})
	call := func(method, body string, auth, csrf bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/api/bases?world=one", strings.NewReader(body))
		if auth {
			req.Header.Set("Authorization", "Bearer test")
		}
		if csrf {
			req.Header.Set(panel.ActHeader, "1")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	body := `{"action":"save","world":"one","name":"camp","base":{"name":"camp","center":[0,40,0],"radius":60,"durationS":600,"travelM":800,"return":"return","sleeping":"bedroom","storage":["warehouse"]}}`
	if w := call("GET", "", false, false); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := call("POST", body, true, false); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := call("POST", body, true, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	b := a.Memory.Store.ActiveBase("one")
	if b == nil || b.Radius != 60 || b.Sleeping != "bedroom" {
		t.Fatal(b)
	}
	if w := call("POST", body, true, true); w.Code != 409 {
		t.Fatal("unversioned overwrite accepted", w.Code)
	}
	update := strings.TrimSuffix(body, "}") + `,"expectedAt":"` + b.At.Format(time.RFC3339Nano) + `"}`
	update = strings.Replace(update, `"radius":60`, `"radius":90`, 1)
	if w := call("POST", update, true, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := call("POST", `{"action":"start","world":"one"}`, true, true); w.Code != 400 {
		t.Fatal(w.Code, w.Body.String())
	}
	if len(a.Goals.List()) != 0 {
		t.Fatal("refused survey left a ready record")
	}
	if w := call("POST", strings.Replace(update, `"radius":90`, `"radius":300`, 1), true, true); w.Code < 400 {
		t.Fatal("oversized scope accepted")
	}
	if w := call("GET", "", true, false); w.Code != 200 || !json.Valid(w.Body.Bytes()) {
		t.Fatal(w.Code, w.Body.String())
	}
	if a.Memory.Store.ActiveBase("two") != nil {
		t.Fatal("base crossed worlds")
	}
}

func TestSurveyEngineStartStopAndOwnerTakeover(t *testing.T) {
	for _, mode := range []string{"queued takeover", "running stop"} {
		t.Run(mode, func(t *testing.T) {
			a := New(config.Default(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil)
			terrain, err := atlas.Open("", atlas.Defaults())
			if err != nil {
				t.Fatal(err)
			}
			defer terrain.Close()
			a.SetAtlas(terrain, nil)
			tasks := []string{"move_to", "stop"}
			url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome { return &protocol.Welcome{AllowedTasks: tasks} }}, a.Session)
			p := bridgetest.ConnectWithFeatures(t, url, "", tasks, []string{"bounded_move_v1"})
			p.Send(&protocol.State{Envelope: protocol.Envelope{T: 100}, World: "one", WorldUID: "1", Phase: protocol.PhaseInWorld, Pos: protocol.Vec3{0, 40, 0}, HP: 80, HPMax: 80, Stamina: 60, Flags: []string{protocol.FlagGrounded}, Food: &protocol.FoodStatus{Active: []protocol.ActiveFood{{Prefab: "Raspberry", RemainingS: 1000}}}})
			p.Send(&protocol.Perception{Envelope: protocol.Envelope{T: 100}})
			scope := protocol.TerrainScope{Namespace: "test", WorldUID: "1", Epoch: "1", Source: "bot", Stream: "run", Game: "test", Generator: 2, Mods: "hash"}
			height, tick, biome := 40.0, 100.0, "Meadows"
			tile := protocol.TerrainTile{WaterLevel: 30, Heights: make([]*float64, 25), Biomes: make([]*string, 25), MeasuredAt: make([]*float64, 25)}
			for i := range tile.Heights {
				tile.Heights[i] = &height
				tile.Biomes[i] = &biome
				tile.MeasuredAt[i] = &tick
			}
			p.Send(&protocol.Terrain{Envelope: protocol.Envelope{T: 100}, World: "one", Scope: &scope, CapturedAt: 100, Tiles: []protocol.TerrainTile{tile}})
			until := time.Now().Add(time.Second)
			for time.Now().Before(until) {
				_, _, s, _ := a.Session.TerrainView()
				if s != nil {
					break
				}
				time.Sleep(time.Millisecond)
			}
			w := home.Waypoint{Name: "camp return", Kind: "place", Pos: protocol.Vec3{10, 40, 0}, At: time.Now().UTC()}
			if err = a.Memory.Store.EditWaypoint("one", "", &w, nil); err != nil {
				t.Fatal(err)
			}
			b := home.Base{Name: "camp", Center: protocol.Vec3{0, 40, 0}, Radius: 60, DurationS: 600, TravelM: 800, Return: w.Name, At: w.At}
			if err = a.Memory.Store.EditBase("one", b.Name, &b, nil, false); err != nil {
				t.Fatal(err)
			}
			if _, err = a.startSurvey(context.Background(), "panel"); err != nil {
				t.Fatal(err)
			}
			g := a.Goals.List()[0]
			if mode == "queued takeover" {
				if _, err = a.Session.Dispatch(context.Background(), session.Order{Task: "move_to", Args: json.RawMessage(`{"to":[1,40,0]}`), TimeoutS: 10}); err != nil {
					t.Fatal(err)
				}
				if req := p.NextTask(); req.Task != "move_to" {
					t.Fatal(req)
				}
				tickUntilGoal(t, a, g.ID, func(g goal.Record) bool { return g.Status == "cancelled" })
			} else {
				until = time.Now().Add(2500 * time.Millisecond)
				for time.Now().Before(until) {
					a.Behaviours.Tick(context.Background())
					g, _ = a.Goals.Get(g.ID)
					if g.Pending != nil && g.Pending.TaskID != "" {
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
				if g.Pending == nil || g.Pending.TaskID == "" {
					t.Fatal("survey never dispatched", g.Status, g.Reason)
				}
				req := p.NextTask()
				var args protocol.MoveToArgs
				if err = json.Unmarshal(req.Args, &args); err != nil || req.Task != "move_to" || args.Scope == nil {
					t.Fatal(req, err)
				}
				if err = a.goalAction(context.Background(), g.ID, "stop"); err != nil {
					t.Fatal(err)
				}
				if stop := p.NextTask(); stop.Task != "stop" {
					t.Fatal(stop)
				}
				g, _ = a.Goals.Get(g.ID)
				if g.Status != "cancelled" {
					t.Fatal(g.Status)
				}
			}
			a.Behaviours.Tick(context.Background())
			p.Quiet(20 * time.Millisecond)
		})
	}
}
