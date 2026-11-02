package behaviour

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/session"
)

type boundaryWait struct{ until time.Time }

func (r *boundaryWait) Validate(json.RawMessage) error        { return nil }
func (r *boundaryWait) Start(json.RawMessage) (Runner, error) { return r, nil }
func (r *boundaryWait) Next(v View, _ *Result) (*session.Order, error) {
	if r.until.IsZero() {
		r.until = v.Now.Add(time.Nanosecond)
	}
	if v.Now.Before(r.until) {
		return nil, &Delay{Until: r.until}
	}
	return nil, nil
}

func TestStoredWakeBoundaryUsesTheSameClockForRunnerAndEngine(t *testing.T) {
	f := setup(t)
	f.e.o.Templates["boundary_wait"] = &boundaryWait{}
	d := def()
	d.Template, d.Params = "boundary_wait", json.RawMessage(`{}`)
	f.put(t, "wait", d)
	f.act(t, "wait", "start")
	// Model a wake that falls between view construction and the engine's
	// later clock read, as the live food-recovery wait did.
	f.e.o.Now = func() time.Time {
		n := f.now
		f.now = f.now.Add(time.Millisecond)
		return n
	}
	f.tick()
	if f.e.active != "wait" || f.e.instances["wait"].completed != 0 {
		t.Fatal("valid wait failed at its clock boundary", f.e.List())
	}
	f.tick()
	if f.e.active != "" || f.e.instances["wait"].last != "completed" {
		t.Fatal("elapsed wait did not finish normally", f.e.List())
	}
}
