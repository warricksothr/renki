package behaviour

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func searchView() View {
	now := time.Now()
	return View{Now: now, StateAt: now, PerceptionAt: now, Features: []string{protocol.FeatureInspectContainer},
		State: &protocol.State{Envelope: protocol.Envelope{T: 10}, World: "test", Phase: protocol.PhaseInWorld, HP: 25, HPMax: 25, Flags: []string{protocol.FlagGrounded}},
		Perception: &protocol.Perception{Envelope: protocol.Envelope{T: 10}, Structures: []protocol.PerceivedStructure{
			{ID: "a", Kind: protocol.StructureContainer, Pos: protocol.Vec3{2, 0, 0}},
			{ID: "b", Kind: protocol.StructureContainer, Pos: protocol.Vec3{4, 0, 0}},
			{ID: "portal", Kind: protocol.StructurePortal, Pos: protocol.Vec3{1, 0, 0}},
			{ID: "far", Kind: protocol.StructureContainer, Pos: protocol.Vec3{30, 0, 0}},
		}}}
}

func newChestRun(t *testing.T, p SearchChestsParams) *chestSearchRun {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	r, err := (SearchChests{}).Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	return r.(*chestSearchRun)
}

func TestChestSearchOnlyFreshCorrelatedInspectionsEarnMatchesOrAbsence(t *testing.T) {
	for _, mode := range []string{"match", "empty", "wrong task", "wrong chest", "withdrawal", "old sequence", "old time"} {
		t.Run(mode, func(t *testing.T) {
			v := searchView()
			v.Containers = []*protocol.Container{{Envelope: protocol.Envelope{Seq: 20}, Entity: "a", Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 999}}}}
			r := newChestRun(t, SearchChestsParams{World: "test", Item: "wood", Chests: []string{"a"}})
			o, err := r.Next(v, nil)
			if err != nil || o.Task != protocol.TaskInspectContainer {
				t.Fatal(o, err)
			}
			if r.checked != 0 || len(r.found) != 0 {
				t.Fatal("remembered stock counted")
			}
			_ = r.Dispatched("inspect-1")
			c := &protocol.Container{Envelope: protocol.Envelope{Seq: 21, T: 10}, Entity: "a", Task: "inspect-1", Direction: protocol.TaskInspectContainer, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 7}, {Prefab: "FineWood", Stack: 2}}}
			switch mode {
			case "empty":
				c.Items = nil
			case "wrong task":
				c.Task = "other"
			case "wrong chest":
				c.Entity = "b"
			case "withdrawal":
				c.Direction = protocol.TaskWithdraw
			case "old sequence":
				c.Seq = 20
			case "old time":
				c.T = 1
			}
			v.Containers = []*protocol.Container{c}
			o, err = r.Next(v, &Result{TaskID: "inspect-1", Status: protocol.StatusSucceeded, Detail: `inspection:{"travel":0}`})
			if mode != "match" && mode != "empty" {
				if !errors.Is(err, ErrWait) || r.checked != 0 {
					t.Fatal(o, err, r.Summary())
				}
				return
			}
			if err != nil || o.Task != protocol.TaskSay || r.checked != 1 {
				t.Fatal(o, err, r.Summary())
			}
			if mode == "match" && (len(r.found) != 1 || r.found[0].count != 7) || mode == "empty" && len(r.found) != 0 {
				t.Fatal(r.Summary())
			}
			if !strings.Contains(o.Args.(*protocol.SayArgs).Text, "Unobserved chests") {
				t.Fatal(o)
			}
		})
	}
}

func TestChestSearchContinuesAfterRefusalAndBoundsTheCandidateList(t *testing.T) {
	v := searchView()
	r := newChestRun(t, SearchChestsParams{World: "test", Item: "Wood", Chests: []string{"missing", "far", "a", "b"}, MaxChests: 1})
	o, err := r.Next(v, nil)
	if err != nil || o.Args.(*protocol.InspectContainerArgs).Container != "a" {
		t.Fatal(o, err)
	}
	args := o.Args.(*protocol.InspectContainerArgs)
	if args.Scope.Center != (protocol.Vec3{}) || args.Scope.Radius != 25 || args.Scope.DurationS > 45 {
		t.Fatal(args)
	}
	_ = r.Dispatched("a1")
	o, err = r.Next(v, &Result{TaskID: "a1", Status: protocol.StatusFailed, Reason: protocol.ReasonNotAllowed, Detail: `inspection:{"travel":0}`})
	if err != nil || o.Task != protocol.TaskSay || r.checked != 0 || !strings.Contains(r.Summary(), "unchecked 4") || !strings.Contains(r.Summary(), "a: not_allowed") {
		t.Fatal(o, err, r.Summary())
	}
}

func TestChestSearchDiscoversNewVisibleChestsWithoutMovingItsCenter(t *testing.T) {
	v := searchView()
	r := newChestRun(t, SearchChestsParams{World: "test", Item: "Fine Wood"})
	o, err := r.Next(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	if o.Args.(*protocol.InspectContainerArgs).Container != "a" {
		t.Fatal(o)
	}
	_ = r.Dispatched("a1")
	v.State.Pos = protocol.Vec3{3, 0, 0}
	v.Perception.Structures = append(v.Perception.Structures, protocol.PerceivedStructure{ID: "c", Kind: protocol.StructureContainer, Pos: protocol.Vec3{24, 0, 0}})
	o, err = r.Next(v, &Result{TaskID: "a1", Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable, Detail: `inspection:{"travel":0}`})
	if err != nil || o.Args.(*protocol.InspectContainerArgs).Container != "b" || len(r.queue) != 1 || r.queue[0].id != "c" || r.center != (protocol.Vec3{}) {
		t.Fatal(o, err, r.queue)
	}
	if !searchItem("Fine Wood", protocol.InventoryItem{Prefab: "FineWood"}) {
		t.Fatal("item name spacing not resolved")
	}
}

func TestChestSearchGuardsInjuryThreatsFreshnessAndBudgetsDuringNativeWork(t *testing.T) {
	for _, mode := range []string{"injury", "threat", "time", "travel", "radius", "world", "state", "perception", "feature", "water"} {
		t.Run(mode, func(t *testing.T) {
			v := searchView()
			seq := uint64(0)
			v.InjurySeq = &seq
			r := newChestRun(t, SearchChestsParams{World: "test", Item: "Wood", TravelM: 10})
			if _, err := r.Next(v, nil); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "injury":
				seq++
			case "threat":
				v.Perception.Characters = []protocol.PerceivedCharacter{{Hostile: true, HPFrac: 1}}
			case "time":
				v.Now = v.Now.Add(181 * time.Second)
				v.StateAt = v.Now
				v.PerceptionAt = v.Now
			case "travel":
				v.State.Pos = protocol.Vec3{11, 0, 0}
			case "radius":
				v.State.Pos = protocol.Vec3{0, 0, 26}
			case "world":
				v.State.World = "other"
			case "state":
				v.StateAt = v.Now.Add(-4 * time.Second)
			case "perception":
				v.PerceptionAt = v.Now.Add(-4 * time.Second)
			case "feature":
				v.Features = nil
			case "water":
				v.State.Location = &protocol.Location{InWater: true}
			}
			if r.Check(v) == nil {
				t.Fatal("unsafe inspection remained admitted", mode)
			}
		})
	}
}

func TestChestSearchParametersRejectUnboundedOrAmbiguousWork(t *testing.T) {
	for _, raw := range []string{`{"world":"test","item":"Wood","radius":101}`, `{"world":"test","item":"Wood","maxChests":33}`, `{"world":"test","item":"Wood","durationS":301}`, `{"world":"test","item":"Wood","travelM":1001}`, `{"world":"test","item":"Wood","chests":["a","a"]}`, `{"world":"test","item":"Wood","take":true}`, `{"world":"test","item":""}`} {
		if (SearchChests{}).Validate(json.RawMessage(raw)) == nil {
			t.Fatal(raw)
		}
	}
}

func searchEngine(t *testing.T) *fixture {
	t.Helper()
	f := setup(t, protocol.TaskInspectContainer)
	f.e.o.Now = time.Now
	hello, _, _ := f.s.Handshake()
	hello.Features = []string{protocol.FeatureInspectContainer}
	v := searchView()
	f.p.Send(v.State)
	f.p.Send(v.Perception)
	wait(t, func() bool { return f.s.Perception() != nil && len(f.s.State().Flags) > 0 })
	return f
}

func TestChestSearchDoesNotReplaceRetreatAndNewOwnerStopCancelsQueuedWork(t *testing.T) {
	f := searchEngine(t)
	if _, err := f.e.StartChestSearch(context.Background(), "Nimah", SearchChestsParams{Item: "Wood"}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.e.StartChestSearch(context.Background(), "Nimah", SearchChestsParams{Item: "Wood"}); err == nil {
		t.Fatal("replaced queued search")
	}
	if _, err := f.s.Dispatch(context.Background(), session.Order{Task: protocol.TaskStop}); err != nil {
		t.Fatal(err)
	}
	f.tick()
	if f.e.List()[0].Pending || f.e.List()[0].Running {
		t.Fatal(f.e.List())
	}
	if _, err := f.s.Dispatch(context.Background(), session.Order{Task: protocol.TaskFlee, Args: &protocol.FleeArgs{From: protocol.Target{Pos: &protocol.Vec3{1, 0, 0}}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.e.StartChestSearch(context.Background(), "Nimah", SearchChestsParams{Item: "Wood"}); err == nil {
		t.Fatal("replaced retreat")
	}
	reopened := New(f.e.o)
	if reopened.List()[0].Pending || reopened.List()[0].Running {
		t.Fatal("restarted manual search")
	}
}

func TestInspectionTravelReceiptDebitsDetoursAndMissingAccountingStopsSearch(t *testing.T) {
	for _, mode := range []string{"measured", "missing", "rejected"} {
		t.Run(mode, func(t *testing.T) {
			v := searchView()
			r := newChestRun(t, SearchChestsParams{World: "test", Item: "wood", TravelM: 40})
			if _, err := r.Next(v, nil); err != nil {
				t.Fatal(err)
			}
			_ = r.Dispatched("a1")
			result := &Result{TaskID: "a1", Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable}
			if mode == "measured" {
				result.Detail = `inspection:{"travel":30}`
			} else if mode == "rejected" {
				result.Status = protocol.StatusRejected
			}
			o, err := r.Next(v, result)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "missing" {
				if o.Task != protocol.TaskSay || !strings.Contains(r.Summary(), "receipt missing") {
					t.Fatal(o, r.Summary())
				}
				return
			}
			if o.Task != protocol.TaskInspectContainer {
				t.Fatal(o)
			}
			want := 40.0
			if mode == "measured" {
				want = 10
			}
			if got := o.Args.(*protocol.InspectContainerArgs).Scope.TravelM; got != want {
				t.Fatal(got, want)
			}
		})
	}
}

func TestSearchEngineInspectsAndReportsWithoutAWithdrawal(t *testing.T) {
	f := searchEngine(t)
	if _, err := f.e.StartChestSearch(context.Background(), "Nimah", SearchChestsParams{Item: "Wood", Chests: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	f.tick()
	req := f.p.NextTask()
	if req.Task != protocol.TaskInspectContainer {
		t.Fatal(req)
	}
	f.p.Send(&protocol.Container{Envelope: protocol.Envelope{T: 10}, Entity: "a", Task: req.ID, Direction: protocol.TaskInspectContainer, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 7}}})
	f.p.Send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: req.ID}, Status: protocol.StatusSucceeded, Detail: `inspection:{"travel":2}`})

	// Wait on the task's final status; the snapshot precedes it on the stream.
	wait(t, func() bool {
		f.e.mu.Lock()
		defer f.e.mu.Unlock()
		status, _, _ := f.e.instances[chestSearchInstance].task.Status()
		return status == protocol.StatusSucceeded
	})
	f.tick()
	report := f.p.NextTask()
	if report.Task != protocol.TaskSay || !strings.Contains(string(report.Args), "7 at a") {
		t.Fatal(report)
	}
	f.done(t, report, protocol.StatusSucceeded, "")
	f.tick()
	rows := f.e.List()
	if rows[0].Running || rows[0].Runs != 1 {
		t.Fatal(rows)
	}
}

func TestInterruptedChestSearchRetainsPartialFindingsAndUnknownCandidates(t *testing.T) {
	f := searchEngine(t)
	if _, err := f.e.StartChestSearch(context.Background(), "Nimah", SearchChestsParams{Item: "Wood"}); err != nil {
		t.Fatal(err)
	}
	f.tick()
	req := f.p.NextTask()
	f.p.Send(&protocol.Container{Envelope: protocol.Envelope{T: 10}, Entity: "a", Task: req.ID, Direction: protocol.TaskInspectContainer, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 7}}})
	f.p.Send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: req.ID}, Status: protocol.StatusSucceeded, Detail: `inspection:{"travel":2}`})
	wait(t, func() bool {
		f.e.mu.Lock()
		defer f.e.mu.Unlock()
		status, _, _ := f.e.instances[chestSearchInstance].task.Status()
		return status == protocol.StatusSucceeded
	})
	f.tick()
	second := f.p.NextTask()
	if second.Task != protocol.TaskInspectContainer {
		t.Fatal(second)
	}
	if _, err := f.s.Dispatch(context.Background(), session.Order{Task: protocol.TaskStop}); err != nil {
		t.Fatal(err)
	}
	f.tick()
	row := f.e.List()[0]
	if row.Running || !strings.Contains(row.Last, "7 at a") || !strings.Contains(row.Last, "unchecked 1") {
		t.Fatal(row)
	}
}
