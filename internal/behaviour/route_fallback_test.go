package behaviour

import (
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
