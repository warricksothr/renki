package behaviour

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// ownerOrder stands in for the chat brain's origin on an owner's follow.
type ownerOrder struct{ player string }

// passiveFixture has the built-in Loot installed and Tas following Nimah
// on an owner's order, with a felled log's wood 3 m from her.
type passiveFixture struct {
	*fixture
	t      *testing.T
	follow *session.Task
}

func followingNimah(t *testing.T) *passiveFixture {
	t.Helper()
	f := setup(t, protocol.TaskFollow)
	if err := f.e.InstallLoot("builtin"); err != nil {
		t.Fatal(err)
	}
	follow, err := f.s.Dispatch(context.Background(), session.Order{Task: protocol.TaskFollow, Args: &protocol.FollowArgs{Player: "Nimah", Run: protocol.RunAuto}, Origin: ownerOrder{"Nimah"}})
	if err != nil {
		t.Fatal(err)
	}
	if req := f.p.NextTask(); req.ID != follow.ID {
		t.Fatal(req)
	}
	pf := &passiveFixture{fixture: f, t: t, follow: follow}
	f.p.Send(pack(1))
	pf.see(10, follow.ID, target("wood", protocol.ResourceDrop, "Wood", 13))
	return pf
}

// see sends a fresh frame with Nimah at x and the body's task running,
// and waits until the engine's view holds it.
func (f *passiveFixture) see(x float64, task string, resources ...protocol.PerceivedResource) {
	seq := uint64(0)
	if p := f.s.Perception(); p != nil {
		seq = p.Seq
	}
	f.p.Send(&protocol.Perception{Characters: []protocol.PerceivedCharacter{owner(x)}, Resources: resources})
	f.p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "test", HP: 25, HPMax: 25, WeightMax: 300,
		Task: &protocol.TaskRef{ID: task, Status: protocol.StatusRunning}})
	wait(f.t, func() bool {
		v := f.e.view()
		return v.Perception != nil && v.Perception.Seq > seq && freshPerception(v) && v.Inventory != nil && v.State.Task != nil && v.State.Task.ID == task
	})
}

// settle has Nimah stand at x=10 past the settle delay, and ticks.
func (f *passiveFixture) settle() {
	f.tick()
	f.now = f.now.Add(6 * time.Second)
	f.see(10, f.follow.ID, target("wood", protocol.ResourceDrop, "Wood", 13))
	f.tick()
}

func (f *passiveFixture) pickup() *protocol.TaskRequest {
	f.t.Helper()
	req := f.p.NextTask()
	if req.Task != protocol.TaskPickup {
		f.t.Fatal("want a pickup, got", req)
	}
	return req
}

// returned checks that the next task is the same follow, sent again.
func (f *passiveFixture) returned() *protocol.TaskRequest {
	f.t.Helper()
	req := f.p.NextTask()
	args, err := req.DecodeArgs()
	if err != nil || req.Task != protocol.TaskFollow || args.(*protocol.FollowArgs).Player != "Nimah" || args.(*protocol.FollowArgs).Run != protocol.RunAuto {
		f.t.Fatal("want the follow back, got", req, err)
	}
	return req
}

func TestASettledFollowIsLentToPassiveLootAndRestartedAfter(t *testing.T) {
	f := followingNimah(t)
	revision := f.s.ControlRevision()
	f.settle()
	req := f.pickup()
	status := f.e.List()[0]
	if !status.Running || status.Passive == nil || *status.Passive != (PassiveRun{Follow: f.follow.ID, Player: "Nimah"}) {
		t.Fatal("the card must show the borrowed follow", status)
	}
	if !strings.Contains(f.log.String(), `"passive":true`) || !strings.Contains(f.log.String(), `"follow":"`+f.follow.ID+`"`) {
		t.Fatal("the audit must show the borrowed follow", f.log.String())
	}
	f.p.Report(f.follow.ID, protocol.StatusCancelled, protocol.ReasonSuperseded)
	wait(t, func() bool { return f.follow.Lent() && isDone(f.follow) })

	f.p.Send(pack(2, material("Wood", 1)))
	f.done(t, req, protocol.StatusSucceeded, "")
	f.see(10, req.ID)
	f.tick()
	back := f.returned()
	if f.s.ControlRevision() != revision {
		t.Fatal("the loan counted as an owner order")
	}
	if got := f.e.List()[0]; got.Running || got.Last != "picked up 1 near Nimah" || got.Passive == nil {
		t.Fatal(got)
	}
	// The returned follow is the owner's again, with its origin.
	f.see(10, back.ID)
	if follow, ok := f.s.OwnerFollow(); !ok || follow.TaskID != back.ID {
		t.Fatal("the returned follow is not the owner's", follow, ok)
	}
}

func TestAWalkingLeadersFollowIsNeverLent(t *testing.T) {
	f := followingNimah(t)
	for x := 10.0; x < 80; x += 7 {
		f.tick()
		f.now = f.now.Add(2 * time.Second)
		f.see(x, f.follow.ID, target("wood", protocol.ResourceDrop, "Wood", x+3))
	}
	f.tick()
	f.p.Quiet(50 * time.Millisecond)
	if f.e.List()[0].Running {
		t.Fatal("loot borrowed a walking leader's follow")
	}
}

func TestLeavingTheSettleAreaEndsThePassiveRunAndRestartsTheFollow(t *testing.T) {
	f := followingNimah(t)
	f.settle()
	req := f.pickup()
	f.p.Report(f.follow.ID, protocol.StatusCancelled, protocol.ReasonSuperseded)
	f.now = f.now.Add(time.Second)
	f.see(25, req.ID)
	f.tick()
	f.returned()
	if got := f.e.List()[0]; got.Running || got.Last != "followed player moved on" {
		t.Fatal(got)
	}
}

func TestAnOwnerOrderDuringAPassiveRunWinsAndNothingRestarts(t *testing.T) {
	f := followingNimah(t)
	f.settle()
	req := f.pickup()
	if _, err := f.s.Dispatch(context.Background(), session.Order{Task: protocol.TaskStop}); err != nil {
		t.Fatal(err)
	}
	if got := f.p.NextTask(); got.Task != protocol.TaskStop {
		t.Fatal(got)
	}
	f.done(t, req, protocol.StatusCancelled, protocol.ReasonSuperseded)
	f.tick()
	f.p.Quiet(50 * time.Millisecond)
	if got := f.e.List()[0]; got.Running || got.Last != "new order" {
		t.Fatal(got)
	}
	if !strings.Contains(f.log.String(), "follow not returned") {
		t.Fatal(f.log.String())
	}
}

func TestAnIdleInstanceNeverBorrowsAFollow(t *testing.T) {
	f := followingNimah(t)
	d := LootDefinition()
	d.Mode = ModeIdle
	f.put(t, LootInstance, d)
	f.settle()
	f.now = f.now.Add(3 * time.Second)
	f.see(10, f.follow.ID, target("wood", protocol.ResourceDrop, "Wood", 13))
	f.tick()
	// A manual start queues it, and it still waits for the follow to end.
	f.act(t, LootInstance, "start")
	f.tick()
	f.p.Quiet(50 * time.Millisecond)
	if got := f.e.List()[0]; got.Running || got.Passive != nil {
		t.Fatal("an idle instance borrowed a follow", got)
	}
}

func TestOnlyPassiveTemplatesMayBePassive(t *testing.T) {
	f := setup(t)
	d := def()
	d.Mode = ModePassive
	raw, _ := json.Marshal(d)
	if err := f.e.Put(context.Background(), "walk", raw, "owner:Nimah"); err == nil {
		t.Fatal("move_to cannot borrow a follow")
	}
	d.Mode = "busy"
	raw, _ = json.Marshal(d)
	if err := f.e.Put(context.Background(), "walk", raw, "owner:Nimah"); err == nil {
		t.Fatal("an unknown mode was saved")
	}
	if err := validate(LootDefinition(), map[string]Template{"loot": &Loot{}}); err != nil || LootDefinition().Mode != ModePassive {
		t.Fatal("the built-in loot is passive", err)
	}
}

func TestTheOldIdleBuiltInLootBecomesPassiveButAnEditedOneStays(t *testing.T) {
	for _, c := range []struct {
		name string
		edit func(*Definition)
		want string
	}{
		{"unedited", func(*Definition) {}, ModePassive},
		{"switched off", func(d *Definition) { d.Enabled = false }, ModePassive},
		{"edited", func(d *Definition) { d.Priority = 3 }, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := setup(t)
			old := idleLootDefinition()
			c.edit(&old)
			f.put(t, LootInstance, old)
			if err := f.e.InstallLoot("builtin"); err != nil {
				t.Fatal(err)
			}
			got := f.e.List()[0].Definition
			if got.Mode != c.want || got.Enabled != old.Enabled {
				t.Fatal(got)
			}
		})
	}
}

func isDone(t *session.Task) bool {
	select {
	case <-t.Done():
		return true
	default:
		return false
	}
}

func TestAFollowThatCannotBeSentBackStopsTheBorrowedStep(t *testing.T) {
	f := followingNimah(t)
	f.settle()
	req := f.pickup()
	f.s.SetPermit(func(task string) bool { return task != protocol.TaskFollow })
	f.now = f.now.Add(time.Second)
	f.see(25, req.ID)
	f.tick()
	if got := f.p.NextTask(); got.Task != protocol.TaskStop {
		t.Fatal("the borrowed step must not keep running unowned", got)
	}
	if !strings.Contains(f.log.String(), "could not return the follow") {
		t.Fatal(f.log.String())
	}
}

func TestAPassiveRunEndsEvenWhenNeitherFollowNorStopCanBeSent(t *testing.T) {
	f := followingNimah(t)
	var told []string
	f.e.o.Notify = func(_ context.Context, line string) { told = append(told, line) }
	f.settle()
	f.pickup()
	f.s.SetPermit(func(task string) bool { return task != protocol.TaskFollow && task != protocol.TaskStop })
	f.now = f.now.Add(time.Second)
	f.see(25, f.follow.ID)
	f.tick()
	f.p.Quiet(50 * time.Millisecond)
	if got := f.e.List()[0]; got.Running || f.e.loan != nil || f.e.lease != nil {
		t.Fatal("a spent loan must end its run", got)
	}
	if !strings.Contains(f.log.String(), "could not stop the passive run's step") || len(told) != 1 || told[0] != "I can't follow you again, Nimah." {
		t.Fatal(told, f.log.String())
	}
}
