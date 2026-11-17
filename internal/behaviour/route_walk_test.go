package behaviour

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

// walkedMemory records walks through a real store, a metre a step.
func walkedMemory(t *testing.T, walks ...[]protocol.Vec3) (RouteMemory, *home.Store) {
	t.Helper()
	s, err := home.Open(filepath.Join(t.TempDir(), "memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Add(-time.Minute)
	for _, corners := range walks {
		var steps []home.RouteStep
		for i := 1; i < len(corners); i++ {
			a, b := corners[i-1], corners[i]
			n := int(distance(a, b))
			prev := a
			for k := 1; k <= n; k++ {
				f := float64(k) / float64(n)
				next := protocol.Vec3{a[0] + f*(b[0]-a[0]), a[1] + f*(b[1]-a[1]), a[2] + f*(b[2]-a[2])}
				steps = append(steps, home.RouteStep{From: prev, To: next, At: now})
				prev = next
			}
		}
		if err = s.RememberRoute("test", steps, now); err != nil {
			t.Fatal(err)
		}
	}
	return RouteMemory{Graph: s.Routes, Failed: func(world string, from, to protocol.Vec3) (bool, error) {
		return s.MarkRouteFailed(world, from, to, time.Now())
	}}, s
}

func moveTarget(t *testing.T, o any) protocol.Vec3 {
	t.Helper()
	return *o.(*protocol.MoveToArgs).To.Pos
}

var ok = &Result{Status: protocol.StatusSucceeded}

func TestRouteWalkGoesLegByLegAndArrives(t *testing.T) {
	a, corner, b := protocol.Vec3{.5, 30, .5}, protocol.Vec3{8.5, 30, .5}, protocol.Vec3{8.5, 30, 8.5}
	mem, _ := walkedMemory(t, []protocol.Vec3{a, corner, b})
	v := searchView()
	v.State.Pos = a
	w := newRouteWalk(mem, "test", a, b, .4)
	o := w.first()
	if got := moveTarget(t, o.Args); distance(got, corner) > .1 || *o.Args.(*protocol.MoveToArgs).StopDistance != routeLegStopM {
		t.Fatalf("first leg = %v", got)
	}
	v.State.Pos = corner
	if o = w.step(v, ok); o == nil || distance(moveTarget(t, o.Args), b) > .1 || *o.Args.(*protocol.MoveToArgs).StopDistance != .4 {
		t.Fatalf("second leg = %+v", o)
	}
	v.State.Pos = b
	if o = w.step(v, ok); o != nil || !w.arrived || w.story() != "route 2 legs 16 m; arrived" {
		t.Fatalf("end = %+v %q", o, w.story())
	}
}

func TestRouteWalkMarksAFailedLinkAndGoesRoundAnotherWay(t *testing.T) {
	a, b := protocol.Vec3{.5, 30, .5}, protocol.Vec3{6.5, 30, 6.5}
	near, far := protocol.Vec3{6.5, 30, .5}, protocol.Vec3{.5, 30, 6.5}
	mem, store := walkedMemory(t, []protocol.Vec3{a, near, b}, []protocol.Vec3{a, far, b})
	v := searchView()
	v.State.Pos = a
	w := newRouteWalk(mem, "test", a, b, .4)
	first := moveTarget(t, w.first().Args)
	// The move fails where the body stands: the link out of the first cell.
	o := w.step(v, &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable})
	if o == nil {
		t.Fatalf("no replan: %q", w.story())
	}
	if second := moveTarget(t, o.Args); distance(second, first) < 1 {
		t.Fatalf("replanned the same way: %v then %v", first, second)
	}
	if stats := home.SummariseRoutes(store.Routes("test")); stats.Failed != 1 {
		t.Fatalf("failed links = %d", stats.Failed)
	}
	if !strings.Contains(w.story(), "leg 1 of 2 failed (unreachable), link marked failed") {
		t.Fatal(w.story())
	}
}

func TestRouteWalkStopsAfterItsReplans(t *testing.T) {
	a, b := protocol.Vec3{.5, 30, .5}, protocol.Vec3{.5, 30, 8.5}
	mem, _ := walkedMemory(t, []protocol.Vec3{a, b})
	// Nothing is marked, so every plan is the same and every move fails.
	mem.Failed = nil
	v := searchView()
	v.State.Pos = a
	w := newRouteWalk(mem, "test", a, b, .4)
	orders := 0
	for o := w.first(); o != nil; o = w.step(v, &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonTimeout}) {
		if orders++; orders > 10 {
			t.Fatal("walk never ends")
		}
	}
	if orders != 1+routeMaxReplans || w.arrived {
		t.Fatalf("orders = %d, %q", orders, w.story())
	}
}

func TestRouteWalkCountsAnArrivalOnTheWrongFloorAsAFailure(t *testing.T) {
	a, b := protocol.Vec3{.5, 30, .5}, protocol.Vec3{.5, 30, 8.5}
	mem, _ := walkedMemory(t, []protocol.Vec3{a, b})
	v := searchView()
	w := newRouteWalk(mem, "test", a, b, .4)
	w.first()
	v.State.Pos = protocol.Vec3{.5, 27, 8.5}
	w.replans = routeMaxReplans
	if o := w.step(v, ok); o != nil || w.arrived || !strings.Contains(w.story(), "stopped short") {
		t.Fatalf("%+v %q", o, w.story())
	}
}

func TestRouteWalkWithoutAPlanIsOneDirectMove(t *testing.T) {
	to := protocol.Vec3{9, 0, 0}
	v := searchView()
	for _, mem := range []RouteMemory{{}, {Graph: func(string) []home.RouteNode { return nil }}} {
		w := newRouteWalk(mem, "test", v.State.Pos, to, 2.5)
		o := w.first()
		if moveTarget(t, o.Args) != to || *o.Args.(*protocol.MoveToArgs).StopDistance != 2.5 {
			t.Fatalf("order = %+v", o)
		}
		if o = w.step(v, &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable}); o != nil || w.arrived {
			t.Fatalf("after a failed direct move: %+v", o)
		}
		if !strings.Contains(w.story(), "direct move failed (unreachable); did not arrive") {
			t.Fatal(w.story())
		}
	}
}

// The census walks a known way leg by leg, then asks again.
func TestCensusWalksOverRouteMemoryToAFarContainer(t *testing.T) {
	a, corner, b := protocol.Vec3{.5, 0, .5}, protocol.Vec3{8.5, 0, .5}, protocol.Vec3{8.5, 0, 8.5}
	mem, _ := walkedMemory(t, []protocol.Vec3{a, corner, b})
	v := searchView()
	v.State.Pos = a
	v.Perception.Structures = []protocol.PerceivedStructure{{ID: "c", Kind: protocol.StructureContainer, Pos: protocol.Vec3{9.5, 0, 9.5}}}
	r := newCensus(t, CensusParams{World: "test", Radius: 25})
	r.routes = mem
	if _, err := r.Next(v, nil); err != nil {
		t.Fatal(err)
	}
	_ = r.Dispatched("t1")
	o, err := r.Next(v, &Result{TaskID: "t1", Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable, Detail: `inspection:{"travel":0}`})
	if err != nil || o.Task != protocol.TaskMoveTo || distance(moveTarget(t, o.Args), corner) > .1 {
		t.Fatal(moveTarget(t, o.Args), r.walk.story(), err)
	}
	_ = r.Dispatched("t2")
	v.State.Pos = corner
	if o, err = r.Next(v, &Result{TaskID: "t2", Status: protocol.StatusSucceeded}); err != nil || o.Task != protocol.TaskMoveTo {
		t.Fatal(o, err)
	}
	_ = r.Dispatched("t3")
	v.State.Pos = b
	if o, err = r.Next(v, &Result{TaskID: "t3", Status: protocol.StatusSucceeded}); err != nil || o.Task != protocol.TaskInspectContainer {
		t.Fatal(o, err)
	}
	_ = r.Dispatched("t4")
	if _, err = r.Next(v, opened(&v, "c", "t4", 9)); err != nil {
		t.Fatal(err)
	}
	if got := r.Entries(); len(got) != 1 || got[0].Outcome != "opened" || !strings.HasPrefix(got[0].Route, "route ") || !strings.HasSuffix(got[0].Route, "arrived") {
		t.Fatalf("entries = %+v", got)
	}
	if !strings.Contains(r.Summary(), "Walks: c: route ") {
		t.Fatal(r.Summary())
	}
}
