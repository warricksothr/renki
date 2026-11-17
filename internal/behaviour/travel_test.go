package behaviour

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func newTravel(t *testing.T, nav Navigator, p TravelParams) *travelRun {
	t.Helper()
	raw, _ := json.Marshal(p)
	r, err := (Travel{Routes: nav}).Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	return r.(*travelRun)
}

func TestTravelValidatesItsDestination(t *testing.T) {
	for _, bad := range []TravelParams{{}, {World: "w", StopM: 50}, {World: "w", To: protocol.Vec3{1e9, 0, 0}}, {World: "w", Place: strings.Repeat("x", 65)}} {
		raw, _ := json.Marshal(bad)
		if (Travel{}).Validate(raw) == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
}

func TestTravelWalksItsLegsAndEndsQuietlyOnArrival(t *testing.T) {
	a, corner, b := protocol.Vec3{.5, 30, .5}, protocol.Vec3{8.5, 30, .5}, protocol.Vec3{8.5, 30, 8.5}
	mem, _ := walkedMemory(t, []protocol.Vec3{a, corner, b})
	v := searchView()
	v.State.Pos = a
	r := newTravel(t, mem, TravelParams{World: "test", To: b, Place: "pantry", Run: true})
	o, err := r.Next(v, nil)
	if err != nil || distance(moveTarget(t, o.Args), corner) > .1 || !*o.Args.(*protocol.MoveToArgs).Run {
		t.Fatal(o, err)
	}
	_ = r.Dispatched("t1")
	v.State.Pos = corner
	if o, err = r.Next(v, &Result{TaskID: "t1", Status: protocol.StatusSucceeded}); err != nil || distance(moveTarget(t, o.Args), b) > .1 {
		t.Fatal(o, err)
	}
	_ = r.Dispatched("t2")
	v.State.Pos = b
	if o, err = r.Next(v, &Result{TaskID: "t2", Status: protocol.StatusSucceeded}); err != nil || o != nil {
		t.Fatal(o, err)
	}
	if got := r.Summary(); got != "Travel to pantry: route 2 legs 16 m; arrived." {
		t.Fatal(got)
	}
}

func TestTravelSaysWhyItDidNotArriveAndStopsAtItsDeadline(t *testing.T) {
	a, b := protocol.Vec3{.5, 30, .5}, protocol.Vec3{.5, 30, 8.5}
	mem, _ := walkedMemory(t, []protocol.Vec3{a, b})
	v := searchView()
	v.State.Pos = a
	r := newTravel(t, mem, TravelParams{World: "test", To: b, Place: "pantry"})
	if _, err := r.Next(v, nil); err != nil {
		t.Fatal(err)
	}
	_ = r.Dispatched("t1")
	if !r.Recover(Result{}) {
		t.Fatal("a failed leg would end the trip")
	}
	// The leg failed and the time is up: no second plan is walked.
	v.Now = v.Now.Add(2 * time.Minute)
	v.StateAt = v.Now
	o, err := r.Next(v, &Result{TaskID: "t1", Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable})
	if err != nil || o.Task != protocol.TaskSay || !strings.Contains(o.Args.(*protocol.SayArgs).Text, "I couldn't get to pantry") || !strings.Contains(r.Summary(), "out of time; did not arrive") {
		t.Fatal(o, err, r.Summary())
	}
	if o, err = r.Next(v, nil); o != nil || err != nil {
		t.Fatal(o, err)
	}
	if _, err = newTravel(t, mem, TravelParams{World: "elsewhere", To: b}).Next(v, nil); err == nil {
		t.Fatal("travelled in another world")
	}
}

func TestStartTravelNeedsAKnownRouteOfMoreThanOneLeg(t *testing.T) {
	f := setup(t)
	if _, err := f.e.StartTravel(context.Background(), "Nimah", TravelParams{To: protocol.Vec3{8.5, 0, 8.5}}); !errors.Is(err, ErrNoKnownRoute) {
		t.Fatalf("without route memory: %v", err)
	}
	a, corner, b := protocol.Vec3{.5, 0, .5}, protocol.Vec3{8.5, 0, .5}, protocol.Vec3{8.5, 0, 8.5}
	mem, _ := walkedMemory(t, []protocol.Vec3{a, corner, b})
	o := f.e.o
	o.Routes, o.Now = mem, time.Now
	f.e = New(o)
	f.p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "test", HP: 25, HPMax: 25, WeightMax: 300, Pos: a, Flags: []string{protocol.FlagGrounded}})
	wait(t, func() bool { return len(f.s.State().Flags) > 0 })
	// One straight leg is no better than the direct move.
	if _, err := f.e.StartTravel(context.Background(), "Nimah", TravelParams{To: corner}); !errors.Is(err, ErrNoKnownRoute) {
		t.Fatalf("one leg: %v", err)
	}
	if _, err := f.e.StartTravel(context.Background(), "Nimah", TravelParams{To: protocol.Vec3{500, 0, 500}}); !errors.Is(err, ErrNoKnownRoute) {
		t.Fatalf("unknown ground: %v", err)
	}
	story, err := f.e.StartTravel(context.Background(), "Nimah", TravelParams{To: b, Place: "pantry"})
	if err != nil || story != "2 legs, 16 m" {
		t.Fatal(story, err)
	}
	if _, err = f.e.StartTravel(context.Background(), "Nimah", TravelParams{To: b}); err == nil || errors.Is(err, ErrNoKnownRoute) {
		t.Fatalf("started a second trip over the first: %v", err)
	}
}
