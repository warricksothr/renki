package behaviour

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/route"
)

// inWorld answers for one recorded world whatever world is asked about.
type inWorld struct {
	Navigator
	world string
}

func (n inWorld) Plan(_ string, from, to protocol.Vec3) route.Plan {
	return n.Navigator.Plan(n.world, from, to)
}

var noWay = &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable}

// The native approach finds no way; route memory knows one. Stow walks it,
// then deposits in the chest it was after.
func TestStowWalksAKnownWayWhenItsApproachFails(t *testing.T) {
	a, corner, chest := protocol.Vec3{.5, 0, .5}, protocol.Vec3{8.5, 0, .5}, protocol.Vec3{8.5, 0, 8.5}
	mem, _ := walkedMemory(t, []protocol.Vec3{a, corner, chest})
	v := haulView()
	v.State.Pos, v.Routes = a, mem
	v.Containers = []*protocol.Container{{Entity: "a", Pos: chest}}
	r := &stowRun{p: stowParams()}
	nextTask(t, r, v, nil, protocol.TaskMoveTo)
	o := nextTask(t, r, v, noWay, protocol.TaskMoveTo)
	if distance(moveTarget(t, o.Args), corner) > .1 {
		t.Fatalf("first leg = %v", moveTarget(t, o.Args))
	}
	if !r.Recover(Result{Status: protocol.StatusFailed, Reason: protocol.ReasonTimeout}) {
		t.Fatal("a failed leg would end the run")
	}
	v.State.Pos = corner
	o = nextTask(t, r, v, success("leg"), protocol.TaskMoveTo)
	if *o.Args.(*protocol.MoveToArgs).StopDistance != stowRouteStopM {
		t.Fatal(o.Args)
	}
	v.State.Pos = protocol.Vec3{8.5, 0, 6.5}
	if o = nextTask(t, r, v, success("leg"), protocol.TaskDeposit); o.Args.(*protocol.TransferArgs).Container != "a" {
		t.Fatal(o)
	}
	if r.Recover(Result{Status: protocol.StatusFailed, Reason: protocol.ReasonTimeout}) {
		t.Fatal("still answering for a walk that is over")
	}
}

// A deposit that walks itself and finds no way is asked again after a walk
// over known ground, once. A second failure moves on to the next chest.
func TestStowWalksAKnownWayOnceWhenADepositCannotReachItsChest(t *testing.T) {
	a, corner, chest := protocol.Vec3{.5, 0, .5}, protocol.Vec3{8.5, 0, .5}, protocol.Vec3{8.5, 0, 8.5}
	mem, _ := walkedMemory(t, []protocol.Vec3{a, corner, chest})
	v := haulView()
	v.State.Pos, v.Routes = a, mem
	v.Containers = []*protocol.Container{{Entity: "a", Pos: chest}}
	p := stowParams()
	p.MatchExisting = true
	r := &stowRun{p: p}
	nextTask(t, r, v, nil, protocol.TaskDeposit)
	nextTask(t, r, v, noWay, protocol.TaskMoveTo)
	v.State.Pos = corner
	nextTask(t, r, v, success("leg"), protocol.TaskMoveTo)
	v.State.Pos = protocol.Vec3{8.5, 0, 6.5}
	if o := nextTask(t, r, v, success("leg"), protocol.TaskDeposit); o.Args.(*protocol.TransferArgs).Container != "a" {
		t.Fatal(o)
	}
	if o := nextTask(t, r, v, noWay, protocol.TaskDeposit); o.Args.(*protocol.TransferArgs).Container != "b" {
		t.Fatal(o)
	}
}

// A walk that does not arrive is that chest's failure, with its story.
func TestStowReportsAWalkThatDidNotArrive(t *testing.T) {
	a, corner, chest := protocol.Vec3{.5, 0, .5}, protocol.Vec3{8.5, 0, .5}, protocol.Vec3{8.5, 0, 8.5}
	mem, _ := walkedMemory(t, []protocol.Vec3{a, corner, chest})
	mem.MarkFailed = nil
	v := haulView()
	v.State.Pos, v.Routes = a, mem
	v.Containers = []*protocol.Container{{Entity: "a", Pos: chest}}
	r := &stowRun{p: stowParams()}
	nextTask(t, r, v, nil, protocol.TaskMoveTo)
	nextTask(t, r, v, noWay, protocol.TaskMoveTo)
	for i := 0; i < routeMaxReplans; i++ {
		nextTask(t, r, v, noWay, protocol.TaskMoveTo)
	}
	if o := nextTask(t, r, v, noWay, protocol.TaskDeposit); o.Args.(*protocol.TransferArgs).Container != "b" {
		t.Fatal(o)
	}
	if len(r.failures) != 1 || !strings.Contains(r.failures[0], "a: route 2 legs") || !strings.HasSuffix(r.failures[0], "did not arrive") {
		t.Fatal(r.failures)
	}
}

// The cargo return finds no way back to its marker; route memory knows one.
func TestCargoReturnWalksAKnownWayWhenItsMoveFails(t *testing.T) {
	r, v := cargoJourney(t)
	origin, store := v.Waypoints[0].Pos, v.Waypoints[1].Pos
	side := protocol.Vec3{store[0], store[1], store[2] + 6}
	far := protocol.Vec3{origin[0], origin[1], origin[2] + 6}
	mem, _ := walkedMemory(t, []protocol.Vec3{store, side, far, origin})
	// The fixture's world is "one"; the walks were recorded in "test".
	v.Routes = inWorld{mem, "test"}
	if _, err := r.Next(v, nil); err != nil {
		t.Fatal(err)
	}
	v.State.Pos = store
	v.Inventory.Seq++
	v.Inventory.Items[0].Stack = 2
	v.Containers = []*protocol.Container{{Entity: "coal", Task: "deposit", Direction: protocol.TaskDeposit, Moved: map[string]int{"Coal": 5}}}
	nextTask(t, r, v, &Result{Status: protocol.StatusSucceeded, TaskID: "deposit"}, protocol.TaskMoveTo)
	if !r.Recover(*noWay) || r.Recover(Result{Status: protocol.StatusCancelled, Reason: protocol.ReasonStopped}) {
		t.Fatal("recovers the wrong return failures")
	}
	steps := 0
	for o := nextTask(t, r, v, noWay, protocol.TaskMoveTo); ; steps++ {
		if steps > 8 {
			t.Fatal("return never ends")
		}
		v.State.Pos = moveTarget(t, o.Args)
		next, err := r.Next(v, success("leg"))
		if err != nil {
			t.Fatal(err)
		}
		if next == nil {
			break
		}
		o = next
	}
	if steps < 1 || v.State.Pos != origin {
		t.Fatalf("steps = %d, at %v", steps, v.State.Pos)
	}
}

// A blocked approach is recovered like an unreachable one: the walk if there
// is a known way, and otherwise the next chest.
func TestStowRecoversABlockedApproach(t *testing.T) {
	blocked := &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonBlocked}
	a, corner, chest := protocol.Vec3{.5, 0, .5}, protocol.Vec3{8.5, 0, .5}, protocol.Vec3{8.5, 0, 8.5}
	mem, _ := walkedMemory(t, []protocol.Vec3{a, corner, chest})
	v := haulView()
	v.State.Pos = a
	v.Containers = []*protocol.Container{{Entity: "a", Pos: chest}}
	r := &stowRun{p: stowParams()}
	nextTask(t, r, v, nil, protocol.TaskMoveTo)
	if !r.Recover(*blocked) {
		t.Fatal("a blocked approach ends the run")
	}
	if o := nextTask(t, r, v, blocked, protocol.TaskDeposit); o.Args.(*protocol.TransferArgs).Container != "b" {
		t.Fatal(o)
	}
	v.Routes = mem
	r = &stowRun{p: stowParams()}
	nextTask(t, r, v, nil, protocol.TaskMoveTo)
	nextTask(t, r, v, blocked, protocol.TaskMoveTo)
}

func TestCargoReturnKeepsTheNativeReasonWhenNoWalkFollows(t *testing.T) {
	r, v := cargoJourney(t)
	if _, err := r.Next(v, nil); err != nil {
		t.Fatal(err)
	}
	v.State.Pos = v.Waypoints[1].Pos
	v.Inventory.Seq++
	v.Inventory.Items[0].Stack = 2
	v.Containers = []*protocol.Container{{Entity: "coal", Task: "deposit", Direction: protocol.TaskDeposit, Moved: map[string]int{"Coal": 5}}}
	nextTask(t, r, v, &Result{Status: protocol.StatusSucceeded, TaskID: "deposit"}, protocol.TaskMoveTo)
	_, err := r.Next(v, &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable, Detail: "no dry route"})
	if err == nil || err.Error() != "cargo return did not succeed: unreachable no dry route" {
		t.Fatal(err)
	}
}

func TestDetourWalksAKnownWayOrKeepsTheNativeReason(t *testing.T) {
	a, corner, b := protocol.Vec3{.5, 0, .5}, protocol.Vec3{8.5, 0, .5}, protocol.Vec3{8.5, 0, 8.5}
	mem, _ := walkedMemory(t, []protocol.Vec3{a, corner, b})
	v := haulView()
	v.State.Pos = a
	var none detour
	if _, _, err := none.next(v, "test", b, 1, noWay, "return"); err == nil || err.Error() != "return did not succeed: unreachable" {
		t.Fatal(err)
	}
	if _, _, err := none.next(v, "test", b, 1, nil, "return"); err == nil {
		t.Fatal("a missing result arrived")
	}
	if _, arrived, err := none.next(v, "test", b, 1, success("move"), "return"); !arrived || err != nil {
		t.Fatal(arrived, err)
	}
	stopped := Result{Status: protocol.StatusCancelled, Reason: protocol.ReasonStopped}
	if !none.recovers(*noWay) || none.recovers(stopped) || none.recovers(Result{Status: protocol.StatusFailed, Reason: protocol.ReasonTimeout}) {
		t.Fatal("recovers the wrong native failures")
	}
	v.Routes = mem
	var d detour
	// A timeout says nothing about the way: no walk for it.
	if _, _, err := d.next(v, "test", b, 1, &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonTimeout}, "return"); err == nil || d.walk != nil {
		t.Fatal(err)
	}
	o, arrived, err := d.next(v, "test", b, 1, noWay, "return")
	if err != nil || arrived || distance(moveTarget(t, o.Args), corner) > .1 {
		t.Fatal(o, arrived, err)
	}
	if !d.recovers(Result{Status: protocol.StatusFailed, Reason: protocol.ReasonTimeout}) {
		t.Fatal("a failed leg would end the run")
	}
	// A cancelled leg is a player's order or a dropped session: not recovered.
	if d.recovers(stopped) || d.recovers(Result{Status: protocol.StatusCancelled, Reason: protocol.ReasonSuperseded}) {
		t.Fatal("walks on over a cancelled leg")
	}
	v.State.Pos = corner
	if o, _, err = d.next(v, "test", b, 1, success("leg"), "return"); err != nil || o == nil {
		t.Fatal(o, err)
	}
	v.State.Pos = b
	if o, arrived, err = d.next(v, "test", b, 1, success("leg"), "return"); err != nil || o != nil || !arrived {
		t.Fatal(o, arrived, err)
	}
	if d.recovers(*noWay) {
		t.Fatal("answers for a walk that is over")
	}
}

// The bedside move finds no way; route memory knows one.
func TestDayRestWalksAKnownWayToTheBedside(t *testing.T) {
	r, v := restFixture(t, false)
	bed := v.Perception.Structures[0].Pos
	from := v.State.Pos
	side := protocol.Vec3{from[0], from[1], from[2] + 6}
	far := protocol.Vec3{bed[0], bed[1], bed[2] + 6}
	mem, _ := walkedMemory(t, []protocol.Vec3{from, side, far, bed})
	v.Routes = inWorld{mem, "test"}
	nextTask(t, r, v, nil, protocol.TaskMoveTo)
	if !r.Recover(*noWay) || r.Recover(Result{Status: protocol.StatusCancelled, Reason: protocol.ReasonStopped}) {
		t.Fatal("recovers the wrong bedside failures")
	}
	o := nextTask(t, r, v, noWay, protocol.TaskMoveTo)
	if o.Args.(*protocol.MoveToArgs).To.Pos == nil {
		t.Fatal("not a leg over walked ground", o.Args)
	}
	// With no known way the native reason is kept.
	r, v = restFixture(t, false)
	nextTask(t, r, v, nil, protocol.TaskMoveTo)
	if _, err := r.Next(v, noWay); err == nil || !strings.Contains(err.Error(), "rest lacks native success: bedside move did not succeed: unreachable") {
		t.Fatal(err)
	}
}

// Collect's move to a far site finds no way; route memory knows one.
func TestCollectWalksAKnownWayToItsSite(t *testing.T) {
	a, corner, site := protocol.Vec3{.5, 0, .5}, protocol.Vec3{30.5, 0, .5}, protocol.Vec3{30.5, 0, 30.5}
	mem, _ := walkedMemory(t, []protocol.Vec3{a, corner, site})
	p := collectParams()
	p.Site.Pos = site
	raw, _ := json.Marshal(p)
	runner, err := (Collect{}).Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	r := runner.(*collectRun)
	v := haulView()
	v.State.Pos, v.Routes = a, mem
	v.Inventory = pack(10)
	nextTask(t, r, v, nil, protocol.TaskMoveTo)
	if !r.Recover(*noWay) {
		t.Fatal("a site with no native way ends the run")
	}
	o := nextTask(t, r, v, noWay, protocol.TaskMoveTo)
	if to := moveTarget(t, o.Args); to == site || distance(to, a) > routeMaxLegForTest {
		t.Fatalf("first leg = %v", to)
	}
}

const routeMaxLegForTest = 10.5

func TestNoWalkerUserRecoversACancelledLeg(t *testing.T) {
	a, corner, chest := protocol.Vec3{.5, 0, .5}, protocol.Vec3{8.5, 0, .5}, protocol.Vec3{8.5, 0, 8.5}
	mem, _ := walkedMemory(t, []protocol.Vec3{a, corner, chest})
	cancelled := Result{Status: protocol.StatusCancelled, Reason: protocol.ReasonSuperseded}
	timeout := Result{Status: protocol.StatusFailed, Reason: protocol.ReasonTimeout}

	v := haulView()
	v.State.Pos, v.Routes = a, mem
	v.Containers = []*protocol.Container{{Entity: "a", Pos: chest}}
	stow := &stowRun{p: stowParams()}
	nextTask(t, stow, v, nil, protocol.TaskMoveTo)
	nextTask(t, stow, v, noWay, protocol.TaskMoveTo)
	if !stow.Recover(timeout) || stow.Recover(cancelled) {
		t.Fatal("stow")
	}

	sv := searchView()
	sv.State.Pos = a
	trip := newTravel(t, mem, TravelParams{World: "test", To: chest})
	if _, err := trip.Next(sv, nil); err != nil {
		t.Fatal(err)
	}
	if !trip.Recover(timeout) || trip.Recover(cancelled) {
		t.Fatal("travel")
	}

	census := newCensus(t, CensusParams{World: "test", Radius: 25})
	census.routes = mem
	sv.Perception.Structures = []protocol.PerceivedStructure{{ID: "c", Kind: protocol.StructureContainer, Pos: protocol.Vec3{9.5, 0, 9.5}}}
	if _, err := census.Next(sv, nil); err != nil {
		t.Fatal(err)
	}
	_ = census.Dispatched("t1")
	if _, err := census.Next(sv, &Result{TaskID: "t1", Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable, Detail: `inspection:{"travel":0}`}); err != nil {
		t.Fatal(err)
	}
	if !census.Recover(timeout) || census.Recover(cancelled) {
		t.Fatal("census")
	}
}

// Gather's walk home finds no way up to the home floor; route memory knows
// the stair. The arrival is still proved on the home floor afterwards.
func TestGatherWalksAKnownWayHome(t *testing.T) {
	r, v := gathering(t)
	r.p.Return = false
	r.p.ReturnHome = true
	home := protocol.Vec3{20.5, 5, 20.5}
	v.Home = &home
	nextTask(t, r, v, nil, protocol.TaskHarvest)
	finishGatherTree(t, r, &v, "one")
	from := v.State.Pos
	foot := protocol.Vec3{from[0] + 10, from[1], from[2]}
	top := protocol.Vec3{foot[0], 5, foot[2] + 10}
	mem, _ := walkedMemory(t, []protocol.Vec3{from, foot, top, home})
	v.Routes = inWorld{mem, "test"}
	if !r.Recover(*noWay) || r.Recover(Result{Status: protocol.StatusCancelled, Reason: protocol.ReasonStopped}) {
		t.Fatal("recovers the wrong home failures")
	}
	o := nextTask(t, r, v, noWay, protocol.TaskMoveTo)
	for steps := 0; ; steps++ {
		if steps > 12 {
			t.Fatal("never home")
		}
		v.State.Pos = moveTarget(t, o.Args)
		next, err := r.Next(v, success("leg"))
		if errors.Is(err, ErrWait) {
			break
		}
		if err != nil || next == nil {
			t.Fatal(next, err)
		}
		o = next
	}
	if distance(v.State.Pos, home) > 2 {
		t.Fatalf("stopped at %v", v.State.Pos)
	}
	// With no known way the native reason is kept.
	r, v = gathering(t)
	r.p.Return, r.p.ReturnHome = false, true
	v.Home = &home
	nextTask(t, r, v, nil, protocol.TaskHarvest)
	finishGatherTree(t, r, &v, "one")
	if _, err := r.Next(v, noWay); err == nil || !strings.Contains(err.Error(), "gather home return lacks native success: home return did not succeed: unreachable") {
		t.Fatal(err)
	}
}
