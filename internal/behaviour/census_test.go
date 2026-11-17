package behaviour

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

func newCensus(t *testing.T, p CensusParams) *censusRun {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	r, err := (Census{}).Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	return r.(*censusRun)
}

// opened answers the pending inspection with a fresh correlated receipt.
func opened(v *View, id, task string, seq uint64, items ...protocol.InventoryItem) *Result {
	v.Containers = append(v.Containers, &protocol.Container{Envelope: protocol.Envelope{Seq: seq, T: v.State.T}, Entity: id,
		Task: task, Direction: protocol.TaskInspectContainer, Items: items})
	return &Result{TaskID: task, Status: protocol.StatusSucceeded, Detail: `inspection:{"travel":1}`}
}

func TestCensusValidatesBounds(t *testing.T) {
	for _, bad := range []CensusParams{{}, {World: "w", Radius: 5}, {World: "w", Radius: 101}, {World: "w", MaxContainers: 97},
		{World: "w", DurationS: 10}, {World: "w", TravelM: 5000}, {World: "w", Center: &protocol.Vec3{1e9, 0, 0}}} {
		raw, _ := json.Marshal(bad)
		if (Census{}).Validate(raw) == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
	raw, _ := json.Marshal(CensusParams{World: "w"})
	if err := (Census{}).Validate(raw); err != nil {
		t.Fatal(err)
	}
}

func TestCensusOpensNearestFirstAndReportsFirstSightings(t *testing.T) {
	v := searchView()
	// "b" was opened before; "a" is known only from perception.
	v.Containers = []*protocol.Container{{Envelope: protocol.Envelope{Seq: 3}, Entity: "b", Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 9}}}}
	r := newCensus(t, CensusParams{World: "test", Radius: 25})
	o, err := r.Next(v, nil)
	if err != nil || o.Task != protocol.TaskInspectContainer || o.Args.(*protocol.InspectContainerArgs).Container != "a" {
		t.Fatal(o, err)
	}
	_ = r.Dispatched("t1")
	// A success without its receipt waits rather than counting old contents.
	if _, err = r.Next(v, &Result{TaskID: "t1", Status: protocol.StatusSucceeded, Detail: `inspection:{"travel":1}`}); !errors.Is(err, ErrWait) {
		t.Fatal(err)
	}
	o, err = r.Next(v, opened(&v, "a", "t1", 10, protocol.InventoryItem{Prefab: "Stone", Stack: 50}, protocol.InventoryItem{Prefab: "Flint", Stack: 3}))
	if err != nil || o.Args.(*protocol.InspectContainerArgs).Container != "b" {
		t.Fatal(o, err)
	}
	_ = r.Dispatched("t2")
	o, err = r.Next(v, opened(&v, "b", "t2", 11))
	if err != nil || o.Task != protocol.TaskSay {
		t.Fatal(o, err)
	}
	got := r.Entries()
	if len(got) != 2 || got[0] != (CensusEntry{Entity: "a", Pos: protocol.Vec3{2, 0, 0}, Outcome: "opened", Stacks: 2, First: true}) ||
		got[1].Entity != "b" || got[1].First || got[1].Stacks != 0 {
		t.Fatalf("%+v", got)
	}
	if line := o.Args.(*protocol.SayArgs).Text; !strings.Contains(line, "opened 2 containers (1 for the first time), 0 gone, 0 I could not open, 0 not checked") {
		t.Fatal(line)
	}
	if o, err = r.Next(v, nil); o != nil || err != nil {
		t.Fatal("census repeated after its report", o, err)
	}
}

func TestCensusWalksToAVerifiedStandingSpotThenAsksAgain(t *testing.T) {
	// 2026-11-17: a barrel behind a double row failed four bounded approaches
	// without moving, then opened from the first verified spot after an
	// ordinary move had gone round the row.
	v := searchView()
	r := newCensus(t, CensusParams{World: "test"})
	if _, err := r.Next(v, nil); err != nil {
		t.Fatal(err)
	}
	_ = r.Dispatched("t1")
	unreachable := &Result{TaskID: "t1", Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable,
		Detail: `inspection:{"travel":0,"detail":"no position","standing":[[90,0,0],[3,0,1],[3,0,-1]]}`}
	o, err := r.Next(v, unreachable)
	if err != nil || o.Task != protocol.TaskMoveTo {
		t.Fatal(o, err)
	}
	move := o.Args.(*protocol.MoveToArgs)
	// The spot outside the census radius is skipped; the walk is ordinary.
	if *move.To.Pos != (protocol.Vec3{3, 0, 1}) || move.Scope != nil || move.Dry {
		t.Fatalf("%+v", move)
	}
	_ = r.Dispatched("m1")
	v.State.Pos = protocol.Vec3{3, 0, 1}
	o, err = r.Next(v, &Result{TaskID: "m1", Status: protocol.StatusSucceeded})
	if err != nil || o.Task != protocol.TaskInspectContainer || o.Args.(*protocol.InspectContainerArgs).Container != "a" {
		t.Fatal(o, err)
	}
	_ = r.Dispatched("t2")
	if _, err = r.Next(v, opened(&v, "a", "t2", 10)); err != nil {
		t.Fatal(err)
	}
	if got := r.Entries(); len(got) != 1 || got[0].Outcome != "opened" || got[0].Walks != 1 {
		t.Fatalf("%+v", got)
	}
	// The walk is charged at twice its straight line, once: the displacement
	// seen afterwards is not added again. The reopened inspection adds 1 m.
	if want := censusWalkCharge*distance(protocol.Vec3{}, protocol.Vec3{3, 0, 1}) + 1; math.Abs(r.base.travel-want) > 1e-9 {
		t.Fatal(r.base.travel, want)
	}
}

func TestCensusDoesNotWalkToAFarSpotOrPastItsTravelBudget(t *testing.T) {
	for name, p := range map[string]CensusParams{
		"far":    {World: "test", Radius: 60},
		"budget": {World: "test", TravelM: 10},
	} {
		v := searchView()
		v.Perception.Structures = v.Perception.Structures[:1]
		r := newCensus(t, p)
		if _, err := r.Next(v, nil); err != nil {
			t.Fatal(name, err)
		}
		_ = r.Dispatched("t1")
		spot := `[20,0,0]`
		if name == "budget" {
			spot = `[6,0,0]`
		}
		o, err := r.Next(v, &Result{TaskID: "t1", Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable,
			Detail: `inspection:{"travel":0,"standing":[` + spot + `]}`})
		if err != nil || o.Task == protocol.TaskMoveTo {
			t.Fatal(name, o, err)
		}
		if got := r.Entries(); len(got) != 1 || got[0].Outcome != "refused" || got[0].Walks != 0 {
			t.Fatalf("%s: %+v", name, got)
		}
	}
}

func TestCensusGivesUpOnAContainerAfterTwoWalks(t *testing.T) {
	v := searchView()
	r := newCensus(t, CensusParams{World: "test"})
	_, _ = r.Next(v, nil)
	fail := func(id string) *Result {
		_ = r.Dispatched(id)
		return &Result{TaskID: id, Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable,
			Detail: `inspection:{"travel":0,"standing":[[3,0,1],[3,0,-1],[1,0,1]]}`}
	}
	for i, want := range []protocol.Vec3{{3, 0, 1}, {3, 0, -1}} {
		o, err := r.Next(v, fail("t"))
		if err != nil || o.Task != protocol.TaskMoveTo || *o.Args.(*protocol.MoveToArgs).To.Pos != want {
			t.Fatal(i, o, err)
		}
		_ = r.Dispatched("m")
		// A failed walk still earns one more look from wherever it ended,
		// whatever stopped it: the run must not end on a walk's reason.
		water := Result{TaskID: "m", Status: protocol.StatusFailed, Reason: "deep_water"}
		if !r.Recover(water) {
			t.Fatal("a failed walk would end the census")
		}
		if o, err = r.Next(v, &water); err != nil || o.Task != protocol.TaskInspectContainer {
			t.Fatal(i, o, err)
		}
	}
	o, err := r.Next(v, fail("t"))
	if err != nil || o.Task != protocol.TaskInspectContainer || o.Args.(*protocol.InspectContainerArgs).Container != "b" {
		t.Fatal("did not move on to the next container", o, err)
	}
	if got := r.Entries(); len(got) != 1 || got[0].Outcome != "refused" || got[0].Walks != 2 || got[0].Reason != protocol.ReasonUnreachable {
		t.Fatalf("%+v", got)
	}
}

func TestCensusCallsAContainerGoneOnlyFromLoadingRange(t *testing.T) {
	missing := &Result{TaskID: "t", Status: protocol.StatusRejected, Reason: protocol.ReasonInvalidTarget}
	start := func(at protocol.Vec3) (*censusRun, View) {
		v := searchView()
		v.State.Pos = at
		v.Perception.Structures = []protocol.PerceivedStructure{{ID: "a", Kind: protocol.StructureContainer, Pos: protocol.Vec3{2, 0, 0}}}
		center := protocol.Vec3{0, 0, 0}
		r := newCensus(t, CensusParams{World: "test", Center: &center, Radius: 40})
		if _, err := r.Next(v, nil); err != nil {
			t.Fatal(err)
		}
		_ = r.Dispatched("t")
		return r, v
	}
	// Close by, no container under that id means it is gone.
	r, v := start(protocol.Vec3{0, 0, 0})
	if o, err := r.Next(v, missing); err != nil || o.Task != protocol.TaskSay {
		t.Fatal(o, err)
	}
	if world, ids := r.Gone(); world != "test" || len(ids) != 1 || ids[0] != "a" {
		t.Fatal(world, ids)
	}
	// From beyond loading range the same answer proves nothing: walk toward
	// the container, stopping within reach, and ask again.
	r, v = start(protocol.Vec3{-35, 0, 0})
	o, err := r.Next(v, missing)
	if err != nil || o.Task != protocol.TaskMoveTo {
		t.Fatal(o, err)
	}
	if move := o.Args.(*protocol.MoveToArgs); *move.To.Pos != (protocol.Vec3{2, 0, 0}) || *move.StopDistance != 2.5 {
		t.Fatalf("%+v", move)
	}
	if _, ids := r.Gone(); len(ids) != 0 {
		t.Fatal("called gone from out of range", ids)
	}
	_ = r.Dispatched("m")
	v.State.Pos = protocol.Vec3{0, 0, 0}
	if o, err = r.Next(v, &Result{TaskID: "m", Status: protocol.StatusSucceeded}); err != nil || o.Task != protocol.TaskInspectContainer {
		t.Fatal(o, err)
	}
	_ = r.Dispatched("t")
	if o, err = r.Next(v, missing); err != nil || o.Task != protocol.TaskSay {
		t.Fatal(o, err)
	}
	if got := r.Entries(); len(got) != 1 || got[0].Outcome != "gone" || got[0].Walks != 1 {
		t.Fatalf("%+v", got)
	}
}

func TestCensusWalksTowardAContainerWhenNoStandingSpotWasMeasured(t *testing.T) {
	// 2026-11-17: from another building the bounded walker found no route
	// and measured nothing, six times running. An ordinary move had crossed
	// the same ground earlier that day.
	v := searchView()
	v.State.Pos = protocol.Vec3{-12, 0, 0}
	v.Perception.Structures = v.Perception.Structures[:1]
	center := protocol.Vec3{0, 0, 0}
	r := newCensus(t, CensusParams{World: "test", Center: &center})
	_, _ = r.Next(v, nil)
	_ = r.Dispatched("t")
	o, err := r.Next(v, &Result{TaskID: "t", Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable,
		Detail: `inspection:{"travel":0,"detail":"no supported, water-safe route"}`})
	if err != nil || o.Task != protocol.TaskMoveTo || *o.Args.(*protocol.MoveToArgs).To.Pos != (protocol.Vec3{2, 0, 0}) {
		t.Fatal(o, err)
	}
	// Already beside it, there is nowhere better to walk: it is refused.
	v = searchView()
	v.Perception.Structures = v.Perception.Structures[:1]
	r = newCensus(t, CensusParams{World: "test", Radius: 25})
	_, _ = r.Next(v, nil)
	_ = r.Dispatched("t")
	o, err = r.Next(v, &Result{TaskID: "t", Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable, Detail: `inspection:{"travel":0}`})
	if err != nil || o.Task != protocol.TaskSay || r.Entries()[0].Outcome != "refused" {
		t.Fatal(o, err, r.Entries())
	}
}

func TestCensusAsksForLandmarkOnlyContainersOncePerPlace(t *testing.T) {
	v := searchView()
	v.Perception.Structures = v.Perception.Structures[:1] // "a" at [2,0,0]
	marks := []home.Landmark{
		{Entity: "old-a", Kind: protocol.StructureContainer, Pos: protocol.Vec3{2, 0, .5}}, // the same box, older id
		{Entity: "stale", Kind: protocol.StructureContainer, Pos: protocol.Vec3{6, 0, 0}},  // nothing seen there now
		{Entity: "door", Kind: protocol.StructureDoor, Pos: protocol.Vec3{7, 0, 0}},        // not storage
		{Entity: "far", Kind: protocol.StructureContainer, Pos: protocol.Vec3{90, 0, 0}},   // outside the radius
	}
	raw, _ := json.Marshal(CensusParams{World: "test", Radius: 25})
	run, err := (Census{Landmarks: func(string) []home.Landmark { return marks }}).Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	r := run.(*censusRun)
	o, err := r.Next(v, nil)
	if err != nil || o.Args.(*protocol.InspectContainerArgs).Container != "a" {
		t.Fatal(o, err)
	}
	_ = r.Dispatched("t1")
	o, err = r.Next(v, opened(&v, "a", "t1", 10))
	if err != nil || o.Task != protocol.TaskInspectContainer || o.Args.(*protocol.InspectContainerArgs).Container != "stale" {
		t.Fatal(o, err)
	}
	_ = r.Dispatched("t2")
	if o, err = r.Next(v, &Result{TaskID: "t2", Status: protocol.StatusRejected, Reason: protocol.ReasonInvalidTarget}); err != nil || o.Task != protocol.TaskSay {
		t.Fatal(o, err)
	}
	if _, ids := r.Gone(); len(ids) != 1 || ids[0] != "stale" {
		t.Fatal(ids)
	}
}

func TestCensusStopsAfterRepeatedRefusalsButNotForGoneContainers(t *testing.T) {
	v := searchView()
	v.Perception.Structures = nil
	for i := 0; i < 12; i++ {
		v.Perception.Structures = append(v.Perception.Structures, protocol.PerceivedStructure{
			ID: string(rune('a' + i)), Kind: protocol.StructureContainer, Pos: protocol.Vec3{float64(i + 1), 0, 0}})
	}
	r := newCensus(t, CensusParams{World: "test"})
	o, err := r.Next(v, nil)
	for i := 0; err == nil && o.Task == protocol.TaskInspectContainer; i++ {
		_ = r.Dispatched("t")
		result := &Result{TaskID: "t", Status: protocol.StatusFailed, Reason: protocol.ReasonRefused, Detail: `inspection:{"travel":0}`}
		if i < 3 {
			result = &Result{TaskID: "t", Status: protocol.StatusRejected, Reason: protocol.ReasonInvalidTarget}
		}
		o, err = r.Next(v, result)
	}
	if err != nil || o.Task != protocol.TaskSay {
		t.Fatal(o, err)
	}
	_, _, gone, refused, _ := r.counts()
	if gone != 3 || refused != censusMaxFailures || r.unchecked() != 3 {
		t.Fatal(gone, refused, r.unchecked(), r.Summary())
	}
	if s := r.Summary(); !strings.Contains(s, "Gone, and forgotten: a, b, c.") || !strings.Contains(s, "Not opened: d: refused") {
		t.Fatal(s)
	}
}

func TestCensusStopsWhenAnInspectionLacksItsTravelReceipt(t *testing.T) {
	v := searchView()
	r := newCensus(t, CensusParams{World: "test"})
	_, _ = r.Next(v, nil)
	_ = r.Dispatched("t1")
	// A timeout writes no receipt; it refuses that container and goes on.
	o, err := r.Next(v, &Result{TaskID: "t1", Status: protocol.StatusFailed, Reason: protocol.ReasonTimeout})
	if err != nil || o.Task != protocol.TaskInspectContainer || strings.Contains(r.Summary(), "receipt missing") {
		t.Fatal(o, err, r.Summary())
	}
	if got := r.Entries(); len(got) != 1 || got[0].Outcome != "refused" || got[0].Reason != protocol.ReasonTimeout {
		t.Fatalf("entries = %+v", got)
	}
	_ = r.Dispatched("t2")
	o, err = r.Next(v, &Result{TaskID: "t2", Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable})
	if err != nil || o.Task != protocol.TaskSay || !strings.Contains(r.Summary(), "receipt missing") {
		t.Fatal(o, err, r.Summary())
	}
}

func TestCensusRefusesAnotherWorldAThreatAndAStaleView(t *testing.T) {
	for name, change := range map[string]func(*View){
		"world":  func(v *View) { v.State.World = "other" },
		"stale":  func(v *View) { v.PerceptionAt = v.Now.Add(-time.Minute) },
		"threat": func(v *View) { v.Perception.Characters = []protocol.PerceivedCharacter{{HPFrac: 1, TargetingMe: true}} },
		"plugin": func(v *View) { v.Features = nil },
	} {
		v := searchView()
		change(&v)
		if _, err := newCensus(t, CensusParams{World: "test"}).Next(v, nil); err == nil {
			t.Fatal(name, "was admitted")
		}
	}
}
