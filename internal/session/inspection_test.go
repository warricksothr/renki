package session

import (
	"context"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestInspectionSnapshotTracksDispatchBeforeStateAndRevokedLease(t *testing.T) {
	s, _, p := setup(t, []string{"move_to", "stop"}, []string{"move_to", "stop"})
	p.Send(&protocol.State{World: "one", WorldUID: "uid", Phase: "in_world", HP: 40, HPMax: 40})
	waitFor(t, func() bool { return s.State() != nil })
	a, err := s.TryAutomation()
	if err != nil {
		t.Fatal(err)
	}
	old, err := a.Dispatch(context.Background(), Order{Task: "move_to", Args: protocol.MoveToArgs{To: protocol.Target{Pos: &protocol.Vec3{1, 0, 0}}}, Origin: "old"})
	if err != nil {
		t.Fatal(err)
	}
	p.NextTask()
	snap := s.InspectionSnapshot()
	if snap.Task == nil || snap.Task.ID != old.ID || !snap.Task.AutomationCurrent || snap.At.IsZero() {
		t.Fatal(snap)
	}
	direct, err := s.Dispatch(context.Background(), Order{Task: "stop", Origin: "new"})
	if err != nil {
		t.Fatal(err)
	}
	p.NextTask()
	snap = s.InspectionSnapshot()
	if snap.Task.ID != direct.ID || snap.Task.AutomationCurrent || snap.Automation || snap.ControlRevision == 0 {
		t.Fatal(snap)
	}
	// A permission refresh is observed without requiring a game state update.
	s.SetPermit(func(task string) bool { return task != "move_to" })
	snap = s.InspectionSnapshot()
	if len(snap.Allowed) != 1 || len(snap.Capabilities) != 2 {
		t.Fatal(snap)
	}
	p.Send(&protocol.State{Envelope: protocol.Envelope{Seq: 10}, World: "other", WorldUID: "other-uid", Phase: "in_world", HP: 40, HPMax: 40})
	waitFor(t, func() bool { return s.State().World == "other" })
	if s.InspectionSnapshot().Task != nil {
		t.Fatal("old world task leaked")
	}
	p.Close()
	waitFor(t, func() bool { return !s.InspectionSnapshot().Connected })
	if snap := s.InspectionSnapshot(); snap.Task != nil || snap.State != nil || !snap.At.Equal(time.Time{}) {
		t.Fatal(snap)
	}
}
