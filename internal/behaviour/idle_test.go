package behaviour

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func idleSetup(t *testing.T) (*fixture, *home.Waypoint) {
	f := setup(t)
	f.now = time.Now().Add(time.Second)
	w := &home.Waypoint{Name: "berries", Kind: "berries", Radius: 15, At: f.now}
	f.e.o.Waypoints = func(world string) []home.Waypoint {
		if world == "test" {
			return []home.Waypoint{*w}
		}
		return nil
	}
	d := def()
	d.Trigger = Trigger{Kind: "idle", Marker: &Marker{World: "test", Name: w.Name, Kind: w.Kind}, DwellS: 10}
	d.Repeat = Repeat{Mode: "trigger", CooldownS: 60, MaxRuns: 2}
	f.put(t, "idle", d)
	return f, w
}
func TestIdleDwellResetsOnLeaveEditStaleOrWrongFloor(t *testing.T) {
	for _, mode := range []string{"leave", "edit", "stale", "floor", "busy", "world", "delete"} {
		t.Run(mode, func(t *testing.T) {
			f, w := idleSetup(t)
			i := f.e.instances["idle"]
			v := View{State: &protocol.State{World: "test", Phase: protocol.PhaseInWorld, HP: 25}, Idle: true, Waypoints: []home.Waypoint{*w}, Now: f.now, StateAt: f.now}
			step := func(seconds int) {
				v.Now = v.Now.Add(time.Duration(seconds) * time.Second)
				v.StateAt = v.Now
				f.e.idleTrigger("idle", i, v, v.Now)
			}
			step(0)
			step(9)
			switch mode {
			case "leave":
				v.State.Pos = protocol.Vec3{20, 0, 0}
			case "floor":
				v.State.Pos = protocol.Vec3{0, -3, 0}
			case "edit":
				v.Waypoints[0].At = w.At.Add(time.Second)
			case "stale":
				v.Now = v.Now.Add(5 * time.Second)
			case "busy":
				v.Idle = false
			case "world":
				v.State.World = "another"
			case "delete":
				v.Waypoints = nil
			}
			f.e.idleTrigger("idle", i, v, v.Now)
			v.State.Pos = protocol.Vec3{}
			v.State.World = "test"
			v.Idle = true
			v.Waypoints = []home.Waypoint{*w}
			step(1)
			if i.pending {
				t.Fatal("dwell carried through invalidation")
			}
			step(10)
			if !i.pending {
				t.Fatal("fresh dwell did not queue")
			}
		})
	}
}
func TestIdleCooldownBudgetAndMarkerDeleteDuringTask(t *testing.T) {
	f, w := idleSetup(t)
	f.tick()
	f.e.instances["idle"].idleSince = f.now.Add(-11 * time.Second)
	f.tick()
	req := f.p.NextTask()
	if req.Task != protocol.TaskMoveTo {
		t.Fatal(req)
	}
	f.done(t, req, protocol.StatusSucceeded, "")
	f.tick()
	f.tick()
	if f.e.List()[0].Runs != 1 {
		t.Fatal(f.e.List())
	}
	i := f.e.instances["idle"]
	i.idleSince = f.now.Add(-11 * time.Second)
	f.tick()
	if f.e.List()[0].Running {
		t.Fatal("ignored cooldown")
	}
	i.next = f.now
	f.tick()
	req = f.p.NextTask()
	w.Name = "deleted"
	f.tick()
	stop := f.p.NextTask()
	if stop.Task != protocol.TaskStop || f.e.List()[0].Running {
		t.Fatal("marker deletion did not stop active native task")
	}
	f.p.Report(stop.ID, protocol.StatusSucceeded, "")
	w.Name = "berries"
	f.e.instances["idle"].next = f.now
	f.e.instances["idle"].idleSince = f.now.Add(-11 * time.Second)
	f.tick()
	if f.e.List()[0].Running || f.e.List()[0].Runs != 2 {
		t.Fatal("ignored finite run budget", f.e.List())
	}
}
func TestIdleNewOwnerOrderClearsDwellAndCannotStopOwner(t *testing.T) {
	f, _ := idleSetup(t)
	f.tick()
	f.e.instances["idle"].idleSince = f.now.Add(-11 * time.Second)
	f.tick()
	f.p.NextTask()
	if _, err := f.s.Dispatch(context.Background(), session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &protocol.Vec3{50, 0, 0}}}, TimeoutS: 30}); err != nil {
		t.Fatal(err)
	}
	owner := f.p.NextTask()
	f.tick()
	if f.e.List()[0].Running || f.e.List()[0].Pending {
		t.Fatal("owner did not take over")
	}
	if owner.Task != protocol.TaskMoveTo {
		t.Fatal("owner order not delivered")
	}
	if f.e.lease != nil {
		t.Fatal("revoked automation retained lease")
	}
}
func TestIdleDefinitionValidation(t *testing.T) {
	f, _ := idleSetup(t)
	d := f.e.List()[0].Definition
	for _, change := range []func(*Definition){
		func(d *Definition) { d.Trigger.DwellS = 0 }, func(d *Definition) { d.Repeat.CooldownS = 0 }, func(d *Definition) { d.Repeat.MaxRuns = 0 }, func(d *Definition) { d.Trigger.Marker.World = "" }, func(d *Definition) { d.Repeat.Mode = "while" },
	} {
		raw, _ := json.Marshal(d)
		var bad Definition
		_ = json.Unmarshal(raw, &bad)
		change(&bad)
		if err := validate(bad, f.e.o.Templates); err == nil {
			t.Fatal("invalid idle definition accepted", bad)
		}
	}
}

func TestIdleSessionChangeDropsAccumulatedDwell(t *testing.T) {
	f, _ := idleSetup(t)
	f.tick()
	i := f.e.instances["idle"]
	i.idleSince = f.now.Add(-9 * time.Second)
	f.e.conn = nil // force the same reset path used for a new handshake
	f.tick()
	if i.pending || f.e.List()[0].Running || i.idleSince.Before(f.now) {
		t.Fatal("old session dwell survived", i.idleSince)
	}
}

type travellingEpisode struct{ ended string }

func (*travellingEpisode) Next(View, *Result) (*session.Order, error) { return nil, nil }
func (*travellingEpisode) Check(View) error                           { return nil }
func (*travellingEpisode) TravelFromIdleMarker() bool                 { return true }
func (*travellingEpisode) EpisodeID() string                          { return "durable-episode" }
func (r *travellingEpisode) EndRun(why string) error                  { r.ended = why; return nil }
func TestDurableIdleJourneyCanTravelButCannotOutliveEditedAdmissionMarker(t *testing.T) {
	f, w := idleSetup(t)
	i := f.e.instances["idle"]
	i.idleMarker = w
	run := &travellingEpisode{}
	i.runner = run
	v := View{Now: f.now, StateAt: f.now, State: &protocol.State{World: "test", Phase: protocol.PhaseInWorld, HP: 25, Pos: protocol.Vec3{30, 0, 0}}, Waypoints: []home.Waypoint{*w}}
	if !idleMarkerValid(i, v) {
		t.Fatal("durable episode cancelled just for travelling")
	}
	v.Waypoints[0].At = v.Now.Add(time.Second)
	if idleMarkerValid(i, v) {
		t.Fatal("edited marker still admitted travel")
	}
	f.e.ended(i, "new order")
	if run.ended != "new order" || i.episodeID != "durable-episode" {
		t.Fatal(run, i.episodeID)
	}
	i.runner = nil
	v.Waypoints[0] = *w
	if idleMarkerValid(i, v) {
		t.Fatal("ordinary idle work gained travel permission")
	}
}
