package goal

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/atlas"
	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func surveyFixture(t *testing.T) (*Template, *runner, behaviour.View, string) {
	t.Helper()
	now := time.Now().UTC()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	link := session.New(nil, log)
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome {
		return &protocol.Welcome{AllowedTasks: []string{"move_to", "stop"}}
	}}, link)
	p := bridgetest.ConnectWithFeatures(t, url, "", []string{"move_to", "stop"}, []string{"bounded_move_v1"})
	state := &protocol.State{Envelope: protocol.Envelope{T: 100}, World: "one", WorldUID: "1", Phase: protocol.PhaseInWorld, Pos: protocol.Vec3{0, 40, 0}, HP: 80, HPMax: 80, Stamina: 60, Flags: []string{protocol.FlagGrounded}, Food: &protocol.FoodStatus{Active: []protocol.ActiveFood{{Prefab: "Raspberry", RemainingS: 1000}}}}
	p.Send(state)
	p.Send(&protocol.Perception{Envelope: protocol.Envelope{T: 100}})
	scope := protocol.TerrainScope{Namespace: "test", WorldUID: "1", Epoch: "1", Source: "bot", Stream: "run", Game: "test", Generator: 2, Mods: "hash"}
	tiles := []protocol.TerrainTile{}
	h, tick, biome := 40.0, 100.0, "Meadows"
	for x := -2; x <= 2; x++ {
		for z := -2; z <= 2; z++ {
			tile := protocol.TerrainTile{X: x, Z: z, WaterLevel: 30, Heights: make([]*float64, 25), Biomes: make([]*string, 25), MeasuredAt: make([]*float64, 25)}
			for i := range tile.Heights {
				tile.Heights[i] = &h
				tile.Biomes[i] = &biome
				tile.MeasuredAt[i] = &tick
			}
			tiles = append(tiles, tile)
		}
	}
	p.Send(&protocol.Terrain{Envelope: protocol.Envelope{T: 100}, World: "one", Scope: &scope, CapturedAt: 100, Tiles: tiles})
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		_, _, s, _ := link.TerrainView()
		if s != nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	_, connection, seenScope, _ := link.TerrainView()
	if seenScope == nil {
		t.Fatal("no terrain identity")
	}
	terrain, err := atlas.Open("", atlas.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { terrain.Close() })
	if err = terrain.Ingest(context.Background(), session.TerrainObservation{World: "one", Scope: scope, CapturedAt: 100, ReceivedAt: now.Add(-10 * time.Second), Tiles: tiles}); err != nil {
		t.Fatal(err)
	}
	mem := home.Memory()
	for _, w := range []home.Waypoint{{Name: "return", Kind: "place", Pos: protocol.Vec3{0, 40, 0}}, {Name: "bedroom", Kind: "rest", Pos: protocol.Vec3{-25, 40, 0}}, {Name: "warehouse", Kind: "storage", Pos: protocol.Vec3{25, 40, 0}, Radius: 6}} {
		w.At = now
		if err = mem.EditWaypoint("one", "", &w, nil); err != nil {
			t.Fatal(err)
		}
	}
	b := home.Base{Name: "camp", Center: protocol.Vec3{0, 40, 0}, Radius: 60, DurationS: 600, TravelM: 800, Return: "return", Sleeping: "bedroom", Storage: []string{"warehouse"}, At: now}
	if err = mem.EditBase("one", b.Name, &b, nil, false); err != nil {
		t.Fatal(err)
	}
	pinned, err := home.ResolveBase(b, mem.Waypoints("one"), nil)
	if err != nil {
		t.Fatal(err)
	}
	store, err := Open(filepath.Join(t.TempDir(), "goals.json"))
	if err != nil {
		t.Fatal(err)
	}
	host := &Template{Store: store, Memory: mem, Session: link, Atlas: func() *atlas.Store { return terrain }}
	injury := link.InjurySequence()
	g, err := NewSurvey("one", atlas.Key(scope), scope.Source, connection, pinned, state.Pos, 80, injury, now)
	if err != nil {
		t.Fatal(err)
	}
	// Admission is exercised through the app integration tests. Runner tests
	// use the explicit clock below rather than setup-time wall-clock freshness.
	if err = store.Add(g); err != nil {
		t.Fatal(err)
	}
	if err = host.Queue(g.ID); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(params{g.ID})
	run, err := host.Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	copy := *state
	v := behaviour.View{Now: now, StateAt: now, PerceptionAt: now, State: &copy, Perception: &protocol.Perception{Envelope: protocol.Envelope{T: 100, Seq: 100}}, Waypoints: mem.Waypoints("one"), Features: []string{"bounded_move_v1"}, InjurySeq: injury}
	return host, run.(*runner), v, g.ID
}
func surveyFresh(v *behaviour.View) {
	v.Now = v.Now.Add(3 * time.Second)
	v.StateAt = v.Now
	v.PerceptionAt = v.Now
	v.State.T += 3
	v.Perception.T = v.State.T
	v.State.Seq += 2
	v.Perception.Seq++
}
func surveyReceipt(t *testing.T, r *runner, v *behaviour.View, o *session.Order, id string, status string) *behaviour.Result {
	t.Helper()
	args := o.Args.(*protocol.MoveToArgs)
	from := v.State.Pos
	to := *args.To.Pos
	if o.Task != protocol.TaskMoveTo || args.Scope == nil || args.RecoverThreats || !args.Dry || !args.AvoidHostiles {
		t.Fatal("unbounded or combat survey order", o)
	}
	if err := r.Dispatched(id); err != nil {
		t.Fatal(err)
	}
	travel := math.Sqrt(math.Pow(from[0]-to[0], 2) + math.Pow(from[1]-to[1], 2) + math.Pow(from[2]-to[2], 2))
	raw, _ := json.Marshal(map[string]any{"travel": travel, "final": to, "trail": []protocol.Vec3{from, to}})
	v.State.Pos = to
	v.State.Seq += 2
	return &behaviour.Result{TaskID: id, Status: status, Detail: "survey:" + string(raw), ReceiptSeq: v.State.Seq - 1}
}
func TestSurveyRechecksSeparatedPlacesBothWaysAndReturnsPartialWithGaps(t *testing.T) {
	host, r, v, id := surveyFixture(t)
	var last *behaviour.Result
	finished := false
	for step := 0; step < 100; step++ {
		surveyFresh(&v)
		o, err := r.Next(v, last)
		var delay *behaviour.Delay
		if errors.As(err, &delay) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		g, _ := host.Store.Get(id)
		if terminal(g.Status) {
			finished = true
			break
		}
		if o != nil {
			last = surveyReceipt(t, r, &v, o, time.Now().Add(time.Duration(step)).Format(time.RFC3339Nano), protocol.StatusSucceeded)
		}
	}
	g, _ := host.Store.Get(id)
	if !finished || g.Status != "partial" || !g.Survey.Returned || !surveyArrived(v.State.Pos, g.Survey.Pinned.Return.Pos) || g.Survey.Coverage.Unknown == 0 || g.Survey.Coverage.New != 0 {
		t.Fatal("movement alone claimed survey completion", g.Status, g.Reason)
	}
	bedVisits, storageVisits := 0, 0
	for _, leg := range g.Survey.Legs {
		if surveyArrived(leg.Final, g.Survey.Pinned.Sleeping.Pos) {
			bedVisits++
		}
		if surveyArrived(leg.Final, g.Survey.Pinned.Storage[0].Pos) {
			storageVisits++
		}
	}
	if bedVisits < 2 || storageVisits < 1 {
		t.Fatal("missing both-way role visits", bedVisits, storageVisits, g.Survey.Unresolved)
	}
}
func TestSurveyScopeEditsThreatAndStaleObservationsStopDuringNativeWork(t *testing.T) {
	for _, mode := range []string{"stale", "threat", "boundary", "role edit", "session", "injury"} {
		t.Run(mode, func(t *testing.T) {
			host, r, v, id := surveyFixture(t)
			surveyFresh(&v)
			o, err := r.Next(v, nil)
			if err != nil || o == nil {
				t.Fatal(o, err)
			}
			if err = r.Dispatched("leg"); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "stale":
				v.StateAt = v.Now.Add(-4 * time.Second)
			case "threat":
				v.Perception.Characters = []protocol.PerceivedCharacter{{HPFrac: 1, Hostile: true, Pos: v.State.Pos}}
			case "boundary":
				v.State.Pos[0] = 100
			case "role edit":
				w, _ := host.Memory.Waypoint("one", "warehouse")
				old := w.At
				w.Pos[0]++
				w.At = w.At.Add(time.Second)
				if err = host.Memory.EditWaypoint("one", w.Name, &w, &old); err != nil {
					t.Fatal(err)
				}
			case "session":
				if err = host.Store.update(id, func(g *Record) error { g.Survey.Connection++; return nil }); err != nil {
					t.Fatal(err)
				}
			case "injury":
				n := uint64(1)
				v.InjurySeq = &n
			}
			if err = r.Check(v); err == nil {
				t.Fatal("guard accepted", mode)
			}
		})
	}
}
func TestSurveyMissingReceiptCancelAndUncertainRestartNeverReplay(t *testing.T) {
	host, r, v, id := surveyFixture(t)
	surveyFresh(&v)
	o, err := r.Next(v, nil)
	if err != nil || o == nil {
		t.Fatal(o, err)
	}
	if err = r.Dispatched("leg"); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(host.Store.path)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := reopened.Get(id)
	if g.Status != "needs_plan" || g.Pending == nil {
		t.Fatal("uncertain restart lost intent")
	}
	if err = (&Template{Store: reopened}).Queue(id); err == nil {
		t.Fatal("uncertain leg resumed")
	}
	if _, err = r.Next(v, &behaviour.Result{TaskID: "leg", Status: protocol.StatusSucceeded}); err != nil {
		t.Fatal(err)
	}
	g, _ = host.Store.Get(id)
	if g.Status != "needs_plan" {
		t.Fatal("unmeasured move earned continuation")
	}
	if err = host.Cancel(id); err != nil {
		t.Fatal(err)
	}
	if o, err = r.Next(v, nil); err != nil || o != nil {
		t.Fatal("cancelled survey continued", o, err)
	}
}
func TestSurveyReturnsWhenWorkTravelReserveIsReached(t *testing.T) {
	host, r, v, id := surveyFixture(t)
	if err := host.Store.update(id, func(g *Record) error { g.Survey.Travel = 500; return nil }); err != nil {
		t.Fatal(err)
	}
	surveyFresh(&v)
	_, err := r.Next(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := host.Store.Get(id)
	if g.Survey.Phase != "complete" || !g.Survey.Returned || g.TaskCount != 0 {
		t.Fatal("reserve dispatched outbound work", g.Survey.Phase, g.TaskCount)
	}
}

func TestSurveyFailedArrivalFloorAndReturnNeverEarnSuccess(t *testing.T) {
	for _, mode := range []string{"wrong floor", "failed return", "food reserve"} {
		t.Run(mode, func(t *testing.T) {
			host, r, v, id := surveyFixture(t)
			if mode == "food reserve" {
				v.State.Food.Active[0].RemainingS = 90
				surveyFresh(&v)
				_, err := r.Next(v, nil)
				if err != nil {
					t.Fatal(err)
				}
				g, _ := host.Store.Get(id)
				if !g.Survey.Returned || g.TaskCount != 0 {
					t.Fatal("food reserve spent on survey", g)
				}
				return
			}
			surveyFresh(&v)
			o, err := r.Next(v, nil)
			if err != nil || o == nil {
				t.Fatal(o, err)
			}
			result := surveyReceipt(t, r, &v, o, "leg", protocol.StatusSucceeded)
			if mode == "wrong floor" {
				v.State.Pos[1] -= 4
			} else {
				if err = host.Store.update(id, func(g *Record) error { g.Survey.Phase = "return"; g.Survey.ReturnIndex = 0; return nil }); err != nil {
					t.Fatal(err)
				}
				result.Status = protocol.StatusFailed
				result.Reason = "blocked"
			}
			surveyFresh(&v)
			_, err = r.Next(v, result)
			var delay *behaviour.Delay
			if err != nil && !errors.As(err, &delay) {
				t.Fatal(err)
			}
			g, _ := host.Store.Get(id)
			if len(g.Survey.Legs) != 1 || g.Survey.Legs[0].Status == protocol.StatusSucceeded || g.Survey.Returned || g.Status == "succeeded" {
				t.Fatal("unearned route proof", g)
			}
			if mode == "failed return" && g.Status != "needs_plan" {
				t.Fatal(g.Status)
			}
		})
	}
}

func TestSurveySuccessRequiresFreshFullCoverageAndSupportedReturn(t *testing.T) {
	host, r, v, id := surveyFixture(t)
	surveyFresh(&v)
	_, _, scope, _ := host.Session.TerrainView()
	height, tick, biome := 40.0, v.State.T, "Meadows"
	tiles := []protocol.TerrainTile{}
	for x := -4; x <= 3; x++ {
		for z := -4; z <= 3; z++ {
			tile := protocol.TerrainTile{X: x, Z: z, WaterLevel: 30, Heights: make([]*float64, 25), Biomes: make([]*string, 25), MeasuredAt: make([]*float64, 25)}
			for i := range tile.Heights {
				tile.Heights[i] = &height
				tile.Biomes[i] = &biome
				tile.MeasuredAt[i] = &tick
			}
			tiles = append(tiles, tile)
		}
	}
	if err := host.Atlas().Ingest(context.Background(), session.TerrainObservation{World: "one", Scope: *scope, CapturedAt: tick, ReceivedAt: v.Now, Tiles: tiles}); err != nil {
		t.Fatal(err)
	}
	var last *behaviour.Result
	for step := 0; step < 100; step++ {
		surveyFresh(&v)
		if step%4 == 0 {
			tick = v.State.T
			if err := host.Atlas().Ingest(context.Background(), session.TerrainObservation{World: "one", Scope: *scope, CapturedAt: tick, ReceivedAt: v.Now, Tiles: tiles}); err != nil {
				t.Fatal(err)
			}
		}
		o, err := r.Next(v, last)
		var delay *behaviour.Delay
		if errors.As(err, &delay) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		g, _ := host.Store.Get(id)
		if terminal(g.Status) {
			break
		}
		if o != nil {
			last = surveyReceipt(t, r, &v, o, time.Now().Add(time.Duration(step)).Format(time.RFC3339Nano), protocol.StatusSucceeded)
		}
	}
	g, _ := host.Store.Get(id)
	if g.Status != "succeeded" || !g.Survey.Returned || g.Survey.Coverage.Refreshed != g.Survey.Coverage.Total || g.Survey.Target != len(g.Survey.Targets) {
		t.Fatal("missing earned completion", g.Status, g.Reason, g.Survey.Target, len(g.Survey.Targets), g.Survey.Unresolved)
	}
}

func TestSurveyKnownTerrainStillExploresUnassignedBasePlaces(t *testing.T) {
	host, r, v, id := surveyFixture(t)
	b := host.Memory.ActiveBase("one")
	expected := b.At
	b.Sleeping = ""
	b.Storage = []string{}
	b.Radius = 10
	b.At = b.At.Add(time.Second)
	if err := host.Memory.EditBase("one", b.Name, b, &expected, false); err != nil {
		t.Fatal(err)
	}
	pinned, err := home.ResolveBase(*host.Memory.ActiveBase("one"), host.Memory.Waypoints("one"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = host.Store.update(id, func(g *Record) error {
		g.Radius = b.Radius
		g.Survey.Pinned = pinned
		g.Survey.Targets = []home.Waypoint{pinned.Return, pinned.Return}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	surveyFresh(&v)
	_, _, scope, tiles := host.Session.TerrainView()
	tick := v.State.T
	for i := range tiles {
		for j := range tiles[i].MeasuredAt {
			tiles[i].MeasuredAt[j] = &tick
		}
	}
	if err = host.Atlas().Ingest(context.Background(), session.TerrainObservation{World: "one", Scope: *scope, CapturedAt: tick, ReceivedAt: v.Now, Tiles: tiles}); err != nil {
		t.Fatal(err)
	}
	o, err := r.Next(v, nil)
	g, _ := host.Store.Get(id)
	if err != nil || o == nil || g.Survey.Phase == "return" || g.Survey.Coverage.Refreshed != g.Survey.Coverage.Total {
		t.Fatal("known terrain suppressed place learning", o, err, g.Survey.Coverage)
	}
}

func TestSurveyBoundPlacesOverWaterStillRequireBoundedNativeRoundTrip(t *testing.T) {
	host, r, v, id := surveyFixture(t)
	surveyFresh(&v)
	_, _, scope, tiles := host.Session.TerrainView()
	tick, belowWater := v.State.T, 28.0
	for i := range tiles {
		for j := range tiles[i].Heights {
			tiles[i].Heights[j] = &belowWater
			tiles[i].MeasuredAt[j] = &tick
		}
	}
	if err := host.Atlas().Ingest(context.Background(), session.TerrainObservation{World: "one", Scope: *scope, CapturedAt: tick, ReceivedAt: v.Now, Tiles: tiles}); err != nil {
		t.Fatal(err)
	}
	var last *behaviour.Result
	visits := map[string]int{}
	for step := 0; step < 30; step++ {
		surveyFresh(&v)
		o, err := r.Next(v, last)
		var delay *behaviour.Delay
		if errors.As(err, &delay) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		g, _ := host.Store.Get(id)
		if terminal(g.Status) {
			break
		}
		if o == nil {
			continue
		}
		args := o.Args.(*protocol.MoveToArgs)
		if !args.Dry || !args.AvoidHostiles || args.Scope == nil || args.Scope.Radius != 60 || args.Scope.TravelM > 100 || args.Scope.DurationS > 40 {
			t.Fatalf("lost native movement constraints: %+v", args)
		}
		pos := *args.To.Pos
		name := ""
		for _, w := range g.Survey.Targets {
			if pos == w.Pos {
				name = w.Name
				break
			}
		}
		if name == "" {
			t.Fatalf("invented a dry intermediate floor: %v", pos)
		}
		visits[name]++
		// Dispatch alone is never an arrival. Feed a separate native receipt.
		if len(g.Survey.Legs) >= g.TaskCount {
			t.Fatal("leg verified without a receipt")
		}
		last = surveyReceipt(t, r, &v, o, time.Now().Add(time.Duration(step)).Format(time.RFC3339Nano), protocol.StatusSucceeded)
	}
	g, _ := host.Store.Get(id)
	if !g.Survey.Returned || visits["bedroom"] < 2 || visits["warehouse"] < 1 || visits["return"] < 1 || g.Status == "succeeded" {
		t.Fatalf("missing verified role routes or false full coverage: visits=%v status=%s returned=%v", visits, g.Status, g.Survey.Returned)
	}
}

func TestSurveyAcceptsFoodMaximumDecayButStillStopsHealthLossAndInjury(t *testing.T) {
	host, r, v, id := surveyFixture(t)
	surveyFresh(&v)
	g, _ := host.Store.Get(id)
	v.State.HP, v.State.HPMax = 70, 70
	if err := r.surveyCheck(g, v); err != nil {
		t.Fatalf("food maximum decay stopped survey: %v", err)
	}
	v.State.HP = 69
	if err := r.surveyCheck(g, v); err == nil {
		t.Fatal("real loss below decayed maximum was ignored")
	}
	v.State.HP = 70
	hit := *v.InjurySeq + 1
	v.InjurySeq = &hit
	if err := r.surveyCheck(g, v); err == nil {
		t.Fatal("injury was hidden by maximum decay")
	}
}
