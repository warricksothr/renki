package behaviour

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
	"github.com/warricksothr/renki/internal/settings"
)

type listener struct{}

func (listener) Chat(*protocol.Chat)    {}
func (listener) Event(*protocol.Event)  {}
func (listener) TaskDone(*session.Task) {}
func wait(t *testing.T, f func() bool) {
	t.Helper()
	until := time.Now().Add(2 * time.Second)
	for !f() {
		if time.Now().After(until) {
			t.Fatal("timed out")
		}
		time.Sleep(time.Millisecond)
	}
}

type fixture struct {
	e     *Engine
	s     *session.Session
	p     *bridgetest.Plugin
	now   time.Time
	store *settings.Store
	path  string
	log   bytes.Buffer
}

func setup(t *testing.T, extraTasks ...string) *fixture {
	t.Helper()
	f := &fixture{now: time.Unix(1000, 0), path: filepath.Join(t.TempDir(), "settings.json")}
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	var err error
	f.store, err = settings.Open(f.path, config.Default(), quiet, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.s = session.New(listener{}, quiet)
	tasks := append([]string{"stop", "move_to", "flee", "deposit", "pickup", "interact", "say"}, extraTasks...)
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome {
		return &protocol.Welcome{AllowedTasks: tasks}
	}}, f.s)
	f.p = bridgetest.Connect(t, url, tasks)
	f.p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "test", HP: 25, HPMax: 25, WeightMax: 300})
	wait(t, func() bool { return f.s.State() != nil })
	f.e = New(Options{Session: f.s, Store: f.store, Now: func() time.Time { return f.now }, Owners: func() []string { return []string{"Nimah"} }, Templates: map[string]Template{"two_steps": twoSteps{}}, Logger: slog.New(slog.NewJSONHandler(&f.log, nil))})
	f.tick()
	return f
}
func (f *fixture) tick() { f.e.Tick(context.Background()) }

func TestViewIncludesCurrentLayoutAndDropsItOnWorldChange(t *testing.T) {
	f := setup(t)
	f.p.Send(&protocol.Layout{Walls: []protocol.Wall{{A: protocol.Vec3{1, 0, 0}, B: protocol.Vec3{1, 0, 4}, H: 2}}})
	wait(t, func() bool { return len(f.e.view().Walls) == 1 })
	f.p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "another", HP: 25, HPMax: 25})
	wait(t, func() bool { return f.e.view().State.World == "another" })
	if len(f.e.view().Walls) != 0 {
		t.Fatal("layout crossed worlds")
	}
}

func def() Definition {
	return Definition{Template: "move_to", Params: json.RawMessage(`{"to":[3,0,0]}`), Trigger: Trigger{Kind: "manual"}, Repeat: Repeat{Mode: "once"}, Enabled: true}
}
func (f *fixture) put(t *testing.T, name string, d Definition) {
	t.Helper()
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.e.Put(context.Background(), name, raw, "owner:Nimah"); err != nil {
		t.Fatal(err)
	}
}
func (f *fixture) act(t *testing.T, name, action string) {
	t.Helper()
	if err := f.e.Action(context.Background(), name, action, "owner:Nimah"); err != nil {
		t.Fatal(err)
	}
}
func (f *fixture) done(t *testing.T, req *protocol.TaskRequest, status, reason string) {
	t.Helper()
	f.p.Report(req.ID, status, reason)
	wait(t, func() bool {
		// Task status can be polled without another state frame.
		f.e.mu.Lock()
		defer f.e.mu.Unlock()
		for _, i := range f.e.instances {
			if i.task != nil && i.task.ID == req.ID {
				select {
				case <-i.task.Done():
					return true
				default:
					return false
				}
			}
		}
		return true
	})
}

type twoSteps struct{}

func (twoSteps) Validate(json.RawMessage) error        { return nil }
func (twoSteps) Start(json.RawMessage) (Runner, error) { return &twoRun{}, nil }

type twoRun struct{ n int }

func (r *twoRun) Next(_ View, _ *Result) (*session.Order, error) {
	r.n++
	if r.n > 2 {
		return nil, nil
	}
	pos := protocol.Vec3{float64(r.n), 0, 0}
	return &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}}, TimeoutS: 10}, nil
}

func TestValidationNeverSavesAnInvalidInstance(t *testing.T) {
	f := setup(t)
	for _, raw := range []string{
		`{"template":"missing"}`, `{"template":"move_to","params":{"to":[1,2,3],"typo":true},"trigger":{"kind":"manual"},"repeat":{"mode":"once"}}`,
		`{"template":"move_to","params":{"to":[1,2,3]},"trigger":{"kind":"timer","everyS":0},"repeat":{"mode":"once"}}`,
		`{"template":"move_to","params":{"to":[1,2,3]},"trigger":{"kind":"manual"},"repeat":{"mode":"while"}}`,
		`{"template":"move_to","params":{"to":[1,2,3]},"trigger":{"kind":"manual"},"repeat":{"mode":"once"},"unknown":1}`,
	} {
		if err := f.e.Put(context.Background(), "bad", json.RawMessage(raw), "owner:Nimah"); err == nil {
			t.Fatal("accepted", raw)
		}
	}
	if len(f.store.Instances()) != 0 {
		t.Fatal("invalid definition saved")
	}
}
func TestPriorityPreemptionRestartsLowerInstance(t *testing.T) {
	f := setup(t)
	low := def()
	low.Template = "two_steps"
	f.put(t, "low", low)
	high := def()
	high.Priority = 10
	f.put(t, "high", high)
	f.act(t, "low", "start")
	f.tick()
	first := f.p.NextTask()
	f.act(t, "high", "start")
	f.tick()
	stop := f.p.NextTask()
	if stop.Task != protocol.TaskStop {
		t.Fatal(stop)
	}
	f.p.Report(first.ID, protocol.StatusCancelled, protocol.ReasonStopped)
	f.p.Report(stop.ID, protocol.StatusSucceeded, "")
	// The pending stop and old task must be acknowledged before re-acquiring.
	wait(t, func() bool {
		lease, err := f.s.TryAutomation()
		if err != nil {
			return false
		}
		lease.Release()
		return true
	})
	f.tick()
	next := f.p.NextTask()
	if f.e.List()[0].Name != "high" || !f.e.List()[0].Running {
		t.Fatal(f.e.List())
	}
	f.done(t, next, protocol.StatusSucceeded, "")
	f.tick()
	f.tick()
	resumed := f.p.NextTask()
	if !strings.Contains(string(resumed.Args), `[1,0,0]`) {
		t.Fatal("lower instance did not restart", resumed)
	}
}
func TestOwnerOrderStopsOldSequenceAndStaleStopCannotCancelIt(t *testing.T) {
	f := setup(t)
	d := def()
	d.Template = "two_steps"
	f.put(t, "work", d)
	f.act(t, "work", "start")
	f.tick()
	first := f.p.NextTask()
	pos := protocol.Vec3{50, 0, 0}
	if _, err := f.s.Dispatch(context.Background(), session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}}, TimeoutS: 10}); err != nil {
		t.Fatal(err)
	}
	owner := f.p.NextTask()
	if owner.ID == first.ID {
		t.Fatal(owner)
	}
	f.p.Report(first.ID, protocol.StatusSucceeded, "")
	f.act(t, "work", "stop")
	f.tick()
	f.p.Quiet(30 * time.Millisecond)
	if f.e.List()[0].Running || f.e.List()[0].Pending {
		t.Fatal(f.e.List())
	}
}
func TestTimerCooldownAndRunLimit(t *testing.T) {
	f := setup(t)
	d := def()
	d.Trigger = Trigger{Kind: "timer", EveryS: 2}
	d.Repeat = Repeat{Mode: "trigger", CooldownS: 5, MaxRuns: 2}
	f.put(t, "trip", d)
	f.tick()
	f.p.Quiet(20 * time.Millisecond)
	f.now = f.now.Add(2 * time.Second)
	f.tick()
	first := f.p.NextTask()
	f.done(t, first, protocol.StatusSucceeded, "")
	f.tick()
	f.now = f.now.Add(2 * time.Second)
	f.tick()
	f.p.Quiet(20 * time.Millisecond)
	f.now = f.now.Add(3 * time.Second)
	f.tick()
	second := f.p.NextTask()
	f.done(t, second, protocol.StatusSucceeded, "")
	f.tick()
	f.now = f.now.Add(time.Hour)
	f.tick()
	f.p.Quiet(20 * time.Millisecond)
	if f.e.List()[0].Runs != 2 {
		t.Fatal(f.e.List())
	}
}
func TestEventsCoalesceAndFailureDoesNotLoop(t *testing.T) {
	f := setup(t)
	d := def()
	d.Trigger = Trigger{Kind: "event", Event: "stock_low"}
	d.Repeat = Repeat{Mode: "trigger"}
	f.put(t, "stock", d)
	for range 1000 {
		f.e.Event("stock_low")
	}
	f.tick()
	req := f.p.NextTask()
	f.done(t, req, protocol.StatusFailed, protocol.ReasonUnreachable)
	f.tick()
	f.now = f.now.Add(time.Hour)
	f.tick()
	f.p.Quiet(20 * time.Millisecond)
	if !strings.Contains(f.log.String(), `"msg":"behaviour failed"`) || !strings.Contains(f.log.String(), `"instance":"stock"`) {
		t.Fatal(f.log.String())
	}
}
func TestConditionsStopAnActiveRun(t *testing.T) {
	f := setup(t)
	d := def()
	d.When.MinFreeSlots = 1
	f.put(t, "trip", d)
	f.act(t, "trip", "start")
	f.tick()
	f.p.Quiet(20 * time.Millisecond)
	f.p.Send(&protocol.Inventory{Size: [2]int{1, 1}})
	wait(t, func() bool { return f.s.Inventory() != nil })
	f.tick()
	f.p.NextTask()
	f.p.Send(&protocol.Inventory{Size: [2]int{1, 1}, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 1}}})
	wait(t, func() bool { return f.s.Inventory().Free() == 0 })
	f.tick()
	if req := f.p.NextTask(); req.Task != protocol.TaskStop {
		t.Fatal(req)
	}
}
func TestPersistenceCloneAndNoManualReplayAfterRestart(t *testing.T) {
	f := setup(t)
	d := def()
	f.put(t, "original", d)
	f.act(t, "original", "start")
	f.tick()
	f.p.NextTask()
	if err := f.e.Clone(context.Background(), "original", "copy", "owner:Nimah"); err != nil {
		t.Fatal(err)
	}
	f.act(t, "original", "disable")
	if req := f.p.NextTask(); req.Task != protocol.TaskStop {
		t.Fatal(req)
	}
	store, err := settings.Open(f.path, config.Default(), slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	if err != nil {
		t.Fatal(err)
	}
	restored := New(Options{Session: f.s, Store: store})
	list := restored.List()
	if len(list) != 2 || list[0].Definition.Enabled || list[1].Definition.Enabled || list[1].Pending || list[1].Running {
		t.Fatal(list)
	}
	f.act(t, "copy", "enable")
	f.act(t, "copy", "delete")
	if len(f.e.List()) != 1 {
		t.Fatal(f.e.List())
	}
}

func TestOwnerStopDiscardsQueuedWorkBetweenRuns(t *testing.T) {
	f := setup(t)
	d := def()
	d.When.MinFreeSlots = 1
	f.put(t, "waiting", d)
	f.act(t, "waiting", "start")
	f.tick()
	if !f.e.List()[0].Pending {
		t.Fatal("condition should leave request pending")
	}
	stop, err := f.s.Dispatch(context.Background(), session.Order{Task: protocol.TaskStop})
	if err != nil {
		t.Fatal(err)
	}
	f.p.NextTask()
	f.p.Report(stop.ID, protocol.StatusSucceeded, "")
	f.p.Send(&protocol.Inventory{Size: [2]int{2, 1}})
	wait(t, func() bool { return f.s.Inventory() != nil })
	f.tick()
	f.tick()
	f.p.Quiet(20 * time.Millisecond)
	if f.e.List()[0].Pending || f.e.List()[0].Running {
		t.Fatal("old queued work survived owner stop")
	}
}

func TestNewManualStartAfterAnOwnerOrderReplacesTheOldRequest(t *testing.T) {
	f := setup(t)
	f.put(t, "trip", def())
	f.act(t, "trip", "start")
	f.tick()
	old := f.p.NextTask()
	stop, err := f.s.Dispatch(context.Background(), session.Order{Task: protocol.TaskStop})
	if err != nil {
		t.Fatal(err)
	}
	f.p.NextTask()
	f.p.Report(old.ID, protocol.StatusCancelled, protocol.ReasonStopped)
	f.p.Report(stop.ID, protocol.StatusSucceeded, "")
	wait(t, func() bool {
		lease, err := f.s.TryAutomation()
		if err != nil {
			return false
		}
		lease.Release()
		return true
	})
	f.act(t, "trip", "start")
	f.tick()
	if req := f.p.NextTask(); req.Task != protocol.TaskMoveTo {
		t.Fatal(req)
	}
}

func TestEveryLifecycleRecordNamesItsInstance(t *testing.T) {
	f := setup(t)
	f.put(t, "workshop trip", def())
	f.act(t, "workshop trip", "start")
	f.tick()
	req := f.p.NextTask()
	f.done(t, req, protocol.StatusSucceeded, "")
	f.tick()
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(f.log.String()), "\n") {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		if msg, ok := record["msg"].(string); ok && strings.HasPrefix(msg, "behaviour ") {
			if record["instance"] != "workshop trip" {
				t.Fatal(record)
			}
			seen[msg] = true
		}
	}
	for _, msg := range []string{"behaviour triggered", "behaviour started", "behaviour step", "behaviour finished"} {
		if !seen[msg] {
			t.Fatal("missing lifecycle record", msg)
		}
	}
}

func TestControllerDeadlineStopsTheStepAndCountsTheFailedAttempt(t *testing.T) {
	f := setup(t)
	d := def()
	d.Repeat.MaxRuns = 1
	f.put(t, "timeout", d)
	f.act(t, "timeout", "start")
	f.tick()
	f.p.NextTask()
	f.now = f.now.Add(66 * time.Second)
	f.tick()
	if req := f.p.NextTask(); req.Task != protocol.TaskStop {
		t.Fatal(req)
	}
	list := f.e.List()
	if list[0].Runs != 1 || list[0].Running || !strings.Contains(f.log.String(), `"msg":"behaviour failed"`) {
		t.Fatal(list, f.log.String())
	}
}
