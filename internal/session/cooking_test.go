package session

import (
	"context"
	"testing"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestCookingEvidenceBindsDispatchedTaskAndCannotChangeAfterFinal(t *testing.T) {
	s, _, p := setup(t, []string{protocol.TaskInteract}, []string{protocol.TaskInteract})
	task, err := s.Dispatch(context.Background(), Order{Task: protocol.TaskInteract, Args: &protocol.InteractArgs{Entity: "station", Cook: "RawMeat"}})
	if err != nil {
		t.Fatal(err)
	}
	p.NextTask()
	for _, id := range []string{"other", task.ID} {
		p.Send(&protocol.Trace{Task: id, Source: "cooking", Msg: "receipt", Data: map[string]any{"id": id}})
	}
	p.Send(&protocol.Trace{Task: task.ID, Source: "cooking", Msg: "wrong data id", Data: map[string]any{"id": "other"}})
	p.Report(task.ID, "succeeded", "")
	waitFor(t, func() bool {
		select {
		case <-task.Done():
			return true
		default:
			return false
		}
	})
	if rows := task.CookingTraces(); len(rows) != 1 || rows[0].Msg != "receipt" {
		t.Fatal(rows)
	}
	p.Send(&protocol.Trace{Task: task.ID, Source: "cooking", Msg: "late", Data: map[string]any{"id": task.ID}})
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one"})
	waitFor(t, func() bool { return s.State() != nil })
	if len(task.CookingTraces()) != 1 {
		t.Fatal("late receipt changed completed task")
	}
}
