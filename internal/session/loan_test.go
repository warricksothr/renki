package session

import (
	"context"
	"errors"
	"testing"

	"github.com/warricksothr/renki/internal/protocol"
)

type chatOrigin struct{ player string }

// following starts an owner's follow of Nimah and has the plugin report it
// as the running task.
func following(t *testing.T) (*Session, *listener, func() *protocol.TaskRequest, *Task, func(...protocol.Message)) {
	t.Helper()
	tasks := []string{"stop", "follow", "pickup", "say"}
	s, l, p := setup(t, tasks, tasks)
	send := func(ms ...protocol.Message) {
		for _, m := range ms {
			p.Send(m)
		}
	}
	state := func(task *protocol.TaskRef) {
		p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "test", HP: 25, Task: task})
		waitFor(t, func() bool { st := s.State(); return st != nil && (task == nil || st.Task != nil) })
	}
	state(nil)
	follow, err := s.Dispatch(context.Background(), Order{Task: protocol.TaskFollow, Args: &protocol.FollowArgs{Player: "Nimah", Run: protocol.RunAuto}, TimeoutS: 0, Origin: chatOrigin{"Nimah"}})
	if err != nil {
		t.Fatal(err)
	}
	p.NextTask()
	state(&protocol.TaskRef{ID: follow.ID, Status: protocol.StatusRunning})
	return s, l, p.NextTask, follow, send
}

func TestOwnerFollowIsLentAndReturnedWithoutCountingAsAnOrder(t *testing.T) {
	s, l, next, follow, send := following(t)
	ctx := context.Background()
	got, ok := s.OwnerFollow()
	if !ok || got != (Follow{TaskID: follow.ID, Player: "Nimah"}) {
		t.Fatal(got, ok)
	}
	if _, err := s.TryAutomation(); !errors.Is(err, ErrBodyBusy) {
		t.Fatal("an ordinary handle must not take a following body", err)
	}
	hello, _, _ := s.Handshake()
	if _, err := s.TryLoanFrom(hello, "test", "t-other"); !errors.Is(err, ErrBodyBusy) {
		t.Fatal("a loan names the follow it borrows", err)
	}
	revision := s.ControlRevision()
	lease, err := s.TryLoanFrom(hello, "test", follow.ID)
	if err != nil {
		t.Fatal(err)
	}
	if lent, ok := lease.Lent(); !ok || lent.Player != "Nimah" {
		t.Fatal(lent, ok)
	}
	pickup, err := lease.Dispatch(ctx, Order{Task: protocol.TaskPickup, Args: &protocol.PickupArgs{Entity: "wood"}, TimeoutS: 10})
	if err != nil {
		t.Fatal(err)
	}
	next()
	send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: follow.ID}, Status: protocol.StatusCancelled, Reason: protocol.ReasonSuperseded})
	waitFor(t, func() bool { return len(l.finished()) == 1 })
	if !l.finished()[0].Lent() {
		t.Fatal("the replaced follow must say it was lent")
	}
	send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: pickup.ID}, Status: protocol.StatusSucceeded})
	waitFor(t, func() bool { return len(l.finished()) == 2 })
	back, err := lease.Return(ctx)
	if err != nil || back == nil {
		t.Fatal(back, err)
	}
	req := next()
	args, _ := req.DecodeArgs()
	if req.Task != protocol.TaskFollow || args.(*protocol.FollowArgs).Player != "Nimah" || args.(*protocol.FollowArgs).Run != protocol.RunAuto {
		t.Fatal(req)
	}
	if back.Origin != (chatOrigin{"Nimah"}) || back.Lent() {
		t.Fatal("the follow returns with its origin", back.Origin)
	}
	if s.ControlRevision() != revision {
		t.Fatal("a loan and its return are not owner orders")
	}
	if lease.Valid() {
		t.Fatal("the handle ends with the return")
	}
	if _, err := lease.Return(ctx); !errors.Is(err, ErrNoLoan) {
		t.Fatal("a follow returns once", err)
	}
}

func TestAnOwnerOrderDuringALoanRevokesItAndNothingReturns(t *testing.T) {
	s, _, next, follow, _ := following(t)
	ctx := context.Background()
	hello, _, _ := s.Handshake()
	lease, err := s.TryLoanFrom(hello, "test", follow.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = lease.Dispatch(ctx, Order{Task: protocol.TaskPickup, Args: &protocol.PickupArgs{Entity: "wood"}, TimeoutS: 10}); err != nil {
		t.Fatal(err)
	}
	next()
	if _, err = s.Dispatch(ctx, Order{Task: protocol.TaskStop}); err != nil {
		t.Fatal(err)
	}
	next()
	if _, err := lease.Return(ctx); !errors.Is(err, ErrBodyBusy) {
		t.Fatal("a revoked loan must not send the follow again", err)
	}
	if _, ok := lease.DiedOnLoan(); ok {
		t.Fatal("an owner order is no death")
	}
}

func TestADeathDuringALoanHandsBackTheFollowForAResume(t *testing.T) {
	for _, c := range []struct {
		name     string
		stepSent bool
	}{{"after the step replaced the follow", true}, {"before any step", false}} {
		t.Run(c.name, func(t *testing.T) {
			s, _, next, follow, send := following(t)
			ctx := context.Background()
			hello, _, _ := s.Handshake()
			lease, err := s.TryLoanFrom(hello, "test", follow.ID)
			if err != nil {
				t.Fatal(err)
			}
			if c.stepSent {
				if _, err = lease.Dispatch(ctx, Order{Task: protocol.TaskPickup, Args: &protocol.PickupArgs{Entity: "wood"}, TimeoutS: 10}); err != nil {
					t.Fatal(err)
				}
				next()
			}
			send(&protocol.State{Envelope: protocol.Envelope{Seq: 99}, Phase: protocol.PhaseDead, World: "test", HP: 0})
			waitFor(t, func() bool { return !lease.Valid() })
			got, ok := lease.DiedOnLoan()
			// Before any step the follow itself ends with the death, and
			// its own listener leaves the resume.
			if ok != c.stepSent || ok && got != follow {
				t.Fatal(got, ok)
			}
		})
	}
}

func TestALoanThatSentNothingLeavesTheFollowRunning(t *testing.T) {
	s, _, _, follow, _ := following(t)
	hello, _, _ := s.Handshake()
	lease, err := s.TryLoanFrom(hello, "test", follow.ID)
	if err != nil {
		t.Fatal(err)
	}
	if back, err := lease.Return(context.Background()); back != nil || err != nil {
		t.Fatal(back, err)
	}
	lease.Release()
	if follow.Lent() {
		t.Fatal("an unused loan ends without touching the follow")
	}
	if got, ok := s.OwnerFollow(); !ok || got.TaskID != follow.ID {
		t.Fatal(got, ok)
	}
}

func TestAutomationsOwnFollowIsNotAnOwnerFollow(t *testing.T) {
	tasks := []string{"stop", "follow"}
	s, _, p := setup(t, tasks, tasks)
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "test", HP: 25})
	waitFor(t, func() bool { return s.State() != nil })
	lease, err := s.TryAutomation()
	if err != nil {
		t.Fatal(err)
	}
	f, err := lease.Dispatch(context.Background(), Order{Task: protocol.TaskFollow, Args: &protocol.FollowArgs{Player: "Nimah"}})
	if err != nil {
		t.Fatal(err)
	}
	p.NextTask()
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "test", HP: 25, Task: &protocol.TaskRef{ID: f.ID, Status: protocol.StatusRunning}})
	waitFor(t, func() bool { return s.State().Task != nil })
	if _, ok := s.OwnerFollow(); ok {
		t.Fatal("only an owner's follow can be lent")
	}
}

func TestAFollowThatEndedBeforeTheStepIsNotLentOrReturned(t *testing.T) {
	s, l, next, follow, send := following(t)
	ctx := context.Background()
	hello, _, _ := s.Handshake()
	lease, err := s.TryLoanFrom(hello, "test", follow.ID)
	if err != nil {
		t.Fatal(err)
	}
	send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: follow.ID}, Status: protocol.StatusFailed, Reason: protocol.ReasonTargetLost})
	waitFor(t, func() bool { return len(l.finished()) == 1 })
	if _, err = lease.Dispatch(ctx, Order{Task: protocol.TaskPickup, Args: &protocol.PickupArgs{Entity: "wood"}, TimeoutS: 10}); err != nil {
		t.Fatal(err)
	}
	next()
	if follow.Lent() {
		t.Fatal("a follow that already ended was marked lent")
	}
	if back, err := lease.Return(ctx); back != nil || err != nil {
		t.Fatal("an ended follow came back", back, err)
	}
}

func TestALentFollowThatEndedOnItsOwnIsNotReturned(t *testing.T) {
	s, l, next, follow, send := following(t)
	ctx := context.Background()
	hello, _, _ := s.Handshake()
	lease, err := s.TryLoanFrom(hello, "test", follow.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = lease.Dispatch(ctx, Order{Task: protocol.TaskPickup, Args: &protocol.PickupArgs{Entity: "wood"}, TimeoutS: 10}); err != nil {
		t.Fatal(err)
	}
	next()
	// The plugin lost the leader before the pickup reached it.
	send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: follow.ID}, Status: protocol.StatusFailed, Reason: protocol.ReasonTargetLost})
	waitFor(t, func() bool { return len(l.finished()) == 1 })
	if back, err := lease.Return(ctx); back != nil || err != nil {
		t.Fatal("a follow that failed on its own came back", back, err)
	}
}
