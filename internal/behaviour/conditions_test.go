package behaviour

import (
	"bytes"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"testing"
	"time"
)

func ptr[T any](v T) *T { return &v }
func TestConditionsUseWeightFlagsOwnersAndRecentChestContents(t *testing.T) {
	home := protocol.Vec3{}
	view := View{
		State:      &protocol.State{Envelope: protocol.Envelope{T: 100}, Phase: protocol.PhaseInWorld, World: "test", HP: 25, Weight: 150, WeightMax: 300},
		Inventory:  &protocol.Inventory{Size: [2]int{2, 1}, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 10}}},
		Perception: &protocol.Perception{Envelope: protocol.Envelope{T: 99}, Characters: []protocol.PerceivedCharacter{{Kind: protocol.CharacterPlayer, Name: "Nimah", HPFrac: 1, Pos: protocol.Vec3{2, 0, 0}}}},
		Containers: []*protocol.Container{{Envelope: protocol.Envelope{T: 98}, Entity: "chest", Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 20}}}},
		Owners:     []string{"Nimah"}, Home: &home,
	}
	c := Conditions{Items: map[string]int{"Wood": 10}, MinFreeSlots: 1, MaxWeightFraction: ptr(.5), Encumbered: ptr(false), Stranded: ptr(false), Night: ptr(false), Owner: &OwnerNear{Name: "Nimah", Radius: 3}, AtHomeRadius: ptr(2.0), Near: &Place{World: "test", Radius: 3}, Stock: []Stock{{Container: "chest", Prefab: "Wood", MinCount: 20, MaxAgeS: 5}}}
	if !c.Match(view) {
		t.Fatal("valid observations refused")
	}
	// The plugin sends a scan between state frames, so the latest scan is
	// often stamped a little after the latest state; the owner is still seen.
	ahead := view
	scan := *view.Perception
	scan.T = 100.15
	ahead.Perception = &scan
	if !c.Match(ahead) {
		t.Fatal("a scan sent just after the latest state refused")
	}
	for _, change := range []string{"items", "slots", "weight", "encumbered", "stranded", "night", "owner", "old owner", "owner seen after state stopped", "world", "raised floor", "old stock", "future stock", "unknown stock"} {
		t.Run(change, func(t *testing.T) {
			v := view
			state := *view.State
			v.State = &state
			cond := c
			switch change {
			case "items":
				cond.Items = map[string]int{"Wood": 11}
			case "slots":
				cond.MinFreeSlots = 2
			case "weight":
				state.Weight = 151
			case "encumbered":
				state.Flags = []string{protocol.FlagEncumbered}
			case "stranded":
				state.Flags = []string{protocol.FlagStranded}
			case "night":
				state.Time.Night = true
			case "owner":
				v.Owners = []string{"someone else"}
			case "old owner":
				p := *view.Perception
				p.T = 90
				v.Perception = &p
			case "owner seen after state stopped":
				p := *view.Perception
				p.T = 101.5
				v.Perception = &p
			case "world":
				state.World = "other"
			case "raised floor":
				state.Pos[1] = 4
			case "old stock":
				chest := *view.Containers[0]
				chest.T = 90
				v.Containers = []*protocol.Container{&chest}
			case "future stock":
				chest := *view.Containers[0]
				chest.T = 101
				v.Containers = []*protocol.Container{&chest}
			case "unknown stock":
				v.Containers = nil
			}
			if cond.Match(v) {
				t.Fatal("condition accepted missing/stale evidence")
			}
		})
	}
}
func TestWhileAndUntilRespectTheirConditionAndMaximumRuns(t *testing.T) {
	for _, mode := range []string{"while", "until"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t)
			d := def()
			d.Repeat = Repeat{Mode: mode, MaxRuns: 2, Condition: &Conditions{Night: ptr(mode == "until")}}
			f.put(t, "repeat", d)
			f.act(t, "repeat", "start")
			f.tick()
			first := f.p.NextTask()
			f.done(t, first, protocol.StatusSucceeded, "")
			f.tick()
			f.now = f.now.Add(1000000000)
			f.tick()
			second := f.p.NextTask()
			f.done(t, second, protocol.StatusSucceeded, "")
			f.tick()
			f.now = f.now.Add(1000000000)
			f.tick()
			if list := f.e.List(); list[0].Runs != 2 || list[0].Running || list[0].Pending {
				t.Fatal(list)
			}
		})
	}
}

// The repeat condition decides whether to run again. Reaching the goal
// mid-run (the 20th flint picked up before the stow) must not stop the run.
func TestUntilConditionReachedMidRunLetsTheRunFinish(t *testing.T) {
	f := setup(t)
	f.p.Send(&protocol.Inventory{Size: [2]int{8, 4}, Items: []protocol.InventoryItem{{Prefab: "Flint", Stack: 5}}})
	wait(t, func() bool { return f.s.Inventory() != nil })
	d := def()
	d.Template = "two_steps"
	d.Repeat = Repeat{Mode: "until", Condition: &Conditions{Items: map[string]int{"Flint": 20}}}
	f.put(t, "collect", d)
	f.act(t, "collect", "start")
	f.tick()
	first := f.p.NextTask()
	f.p.Send(&protocol.Inventory{Size: [2]int{8, 4}, Items: []protocol.InventoryItem{{Prefab: "Flint", Stack: 20}}})
	wait(t, func() bool { return f.s.Inventory().Items[0].Stack == 20 })
	f.tick()
	f.done(t, first, protocol.StatusSucceeded, "")
	f.tick()
	second := f.p.NextTask()
	if second.Task == protocol.TaskStop {
		t.Fatal("until condition stopped the run in progress")
	}
	f.done(t, second, protocol.StatusSucceeded, "")
	f.tick()
	f.now = f.now.Add(time.Hour)
	f.tick()
	f.p.Quiet(20 * time.Millisecond)
	if list := f.e.List(); list[0].Runs != 1 || list[0].Running || list[0].Pending || list[0].Last != "completed" {
		t.Fatal(list)
	}
}

// A run preempted after its until goal is reached is still a run in
// progress: it restarts (and so stows) rather than waiting forever as
// pending, even when the owner presses Start again while it waits.
func TestPreemptedRunRestartsAfterItsUntilGoalIsReached(t *testing.T) {
	for _, pressStart := range []bool{false, true} {
		t.Run(map[bool]string{false: "on its own", true: "after Start"}[pressStart], func(t *testing.T) {
			preemptAfterUntilGoal(t, pressStart)
		})
	}
}
func preemptAfterUntilGoal(t *testing.T, pressStart bool) {
	t.Helper()
	f := setup(t)
	f.p.Send(&protocol.Inventory{Size: [2]int{8, 4}, Items: []protocol.InventoryItem{{Prefab: "Flint", Stack: 20}}})
	wait(t, func() bool { return f.s.Inventory() != nil })
	low := def()
	low.Template = "two_steps"
	low.Repeat = Repeat{Mode: "until", Condition: &Conditions{Items: map[string]int{"Flint": 30}}}
	f.put(t, "collect", low)
	high := def()
	high.Priority = 10
	f.put(t, "high", high)
	f.act(t, "collect", "start")
	f.tick()
	first := f.p.NextTask()
	// The goal is reached while the run is in progress.
	f.p.Send(&protocol.Inventory{Size: [2]int{8, 4}, Items: []protocol.InventoryItem{{Prefab: "Flint", Stack: 30}}})
	wait(t, func() bool { return f.s.Inventory().Items[0].Stack == 30 })
	f.act(t, "high", "start")
	f.tick()
	stop := f.p.NextTask()
	if stop.Task != protocol.TaskStop {
		t.Fatal(stop)
	}
	f.p.Report(first.ID, protocol.StatusCancelled, protocol.ReasonStopped)
	f.p.Report(stop.ID, protocol.StatusSucceeded, "")
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
	if pressStart {
		f.act(t, "collect", "start")
	}
	f.done(t, next, protocol.StatusSucceeded, "")
	f.tick()
	f.tick()
	restarted := f.p.NextTask()
	if restarted.Task != protocol.TaskMoveTo || !bytes.Contains(restarted.Args, []byte(`[1,0,0]`)) {
		t.Fatal("preempted run did not restart", restarted)
	}
}
func TestManualStartBeforeFirstSchedulerTickIsNotLost(t *testing.T) {
	f := setup(t)
	e := New(f.e.o)
	f.e = e
	f.put(t, "start", def())
	f.act(t, "start", "start")
	f.tick()
	f.p.NextTask()
}

func TestRememberedStockUsesItsAgeAndLiveEmptyChestWins(t *testing.T) {
	f := setup(t)
	f.e.o.Containers = func(world string) []home.RememberedContainer {
		if world != "test" {
			t.Fatal(world)
		}
		return []home.RememberedContainer{{Entity: "chest", Size: [2]int{2, 2}, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 10}}, At: f.now.Add(-2 * time.Second)}}
	}
	c := Conditions{Stock: []Stock{{Container: "chest", Prefab: "Wood", MinCount: 10, MaxAgeS: 3}}}
	if !c.Match(f.e.view()) {
		t.Fatal("fresh remembered stock ignored")
	}
	c.Stock[0].MaxAgeS = 1
	if c.Match(f.e.view()) {
		t.Fatal("stale remembered stock accepted")
	}
	c.Stock[0].MaxAgeS = 3
	f.p.Send(&protocol.Container{Entity: "chest", Size: [2]int{2, 2}})
	wait(t, func() bool { return len(f.s.Containers()) == 1 })
	if c.Match(f.e.view()) {
		t.Fatal("old memory overruled live empty chest")
	}
}

func TestContentsInheritedFromAnotherObjectCannotSatisfyStock(t *testing.T) {
	f := setup(t)
	f.e.o.Containers = func(string) []home.RememberedContainer {
		return []home.RememberedContainer{{Entity: "current", ObservedEntity: "old", At: f.now,
			Size: [2]int{2, 2}, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 20}}}}
	}
	c := Conditions{Stock: []Stock{{Container: "current", Prefab: "Wood", MinCount: 10, MaxAgeS: 5}}}
	if c.Match(f.e.view()) {
		t.Fatal("unverified old contents satisfied current stock")
	}
}
