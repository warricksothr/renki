package behaviour

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

type handoffTemplate struct{ run *handoffRun }

func (h handoffTemplate) Validate(json.RawMessage) error        { return nil }
func (h handoffTemplate) Start(json.RawMessage) (Runner, error) { return h.run, nil }

type handoffRun struct{ takeover func() }

func (h *handoffRun) Next(View, *Result) (*session.Order, error) { return nil, nil }
func (h *handoffRun) Continuation(View) (*session.Order, error) {
	if h.takeover != nil {
		h.takeover()
	}
	return &session.Order{Task: protocol.TaskEscort, Args: &protocol.EscortArgs{Player: "Nimah"}}, nil
}
func TestCompletionHandoffUsesTheOriginalLeaseAndCurrentPermission(t *testing.T) {
	for _, mode := range []string{"resume", "owner takeover", "permission revoked"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t, protocol.TaskEscort)
			run := &handoffRun{}
			f.e.o.Templates["handoff"] = handoffTemplate{run}
			d := def()
			d.Template = "handoff"
			d.Params = json.RawMessage(`{}`)
			f.put(t, "food", d)
			f.act(t, "food", "start")
			if mode == "owner takeover" {
				run.takeover = func() {
					_, err := f.s.Dispatch(context.Background(), session.Order{Task: protocol.TaskStop})
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			if mode == "permission revoked" {
				f.s.SetPermit(func(task string) bool { return task != protocol.TaskEscort })
			}
			f.tick()
			if mode != "permission revoked" {
				task := f.p.NextTask()
				want := protocol.TaskEscort
				if mode == "owner takeover" {
					want = protocol.TaskStop
				}
				if task.Task != want {
					t.Fatal(task)
				}
			}
			f.p.Quiet(20 * time.Millisecond)
			if f.e.active != "" {
				t.Fatal("completed handoff still owns the scheduler")
			}
		})
	}
}
