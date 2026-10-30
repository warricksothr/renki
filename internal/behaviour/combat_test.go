package behaviour

import (
	"encoding/json"
	"github.com/warricksothr/renki/internal/protocol"
	"testing"
)

func TestCombatDelegatesOneGoalWithoutPuppetingPhases(t *testing.T) {
	runner, err := (Combat{}).Start(json.RawMessage(`{"world":"one","at":[1,2,3]}`))
	if err != nil {
		t.Fatal(err)
	}
	v := View{State: &protocol.State{World: "one", Phase: protocol.PhaseInWorld, Food: &protocol.FoodStatus{Active: []protocol.ActiveFood{{Prefab: "CookedDeerMeat"}}}}}
	order, err := runner.Next(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, ok := order.Args.(*protocol.AttackArgs)
	if order.Task != protocol.TaskAttack || !ok || !a.Kite || a.Entity != "" || a.At == nil || *a.At != (protocol.Vec3{1, 2, 3}) {
		t.Fatal(order)
	}
	for _, phase := range []string{"search", "draw", "kite", "hold", "block", "retreat"} {
		v.State.Combat = &protocol.CombatStatus{Mode: phase}
		next, err := runner.Next(v, nil)
		if err != nil || next != nil {
			t.Fatalf("phase %s: %v %v", phase, next, err)
		}
	}
}
func TestCombatValidatesGoalsAndRequiresCurrentWorldAndFood(t *testing.T) {
	for _, raw := range []string{`{}`, `{"world":"one","extra":true}`, `{"world":"one","at":[1,2]}`, `{"world":"one","at":"entity"}`} {
		if err := (Combat{}).Validate(json.RawMessage(raw)); err == nil {
			t.Fatal(raw)
		}
	}
	for _, world := range []string{"one", "two"} {
		r, err := (Combat{}).Start(json.RawMessage(`{"world":"one"}`))
		if err != nil {
			t.Fatal(err)
		}
		_, err = r.Next(View{State: &protocol.State{World: world, Phase: protocol.PhaseInWorld}}, nil)
		if err == nil {
			t.Fatal("missing food or wrong world allowed")
		}
	}
}

func TestEditingCombatStopsOldGoalBeforeStartingReplacement(t *testing.T) {
	f := setup(t, "attack")
	f.p.Send(&protocol.State{World: "test", Phase: protocol.PhaseInWorld, HP: 80, HPMax: 80, Stamina: 60, Food: &protocol.FoodStatus{Active: []protocol.ActiveFood{{Prefab: "CookedDeerMeat"}}}})
	wait(t, func() bool { return f.s.State().Food != nil })
	d := Definition{Template: "combat", Params: json.RawMessage(`{"world":"test"}`), Trigger: Trigger{Kind: "manual"}, Repeat: Repeat{Mode: "once"}, Enabled: true}
	f.put(t, "fight", d)
	f.act(t, "fight", "start")
	f.tick()
	first := f.p.NextTask()
	if first.Task != protocol.TaskAttack {
		t.Fatal(first)
	}
	var initial protocol.AttackArgs
	if err := json.Unmarshal(first.Args, &initial); err != nil || !initial.Kite || initial.Entity != "" {
		t.Fatal(first)
	}
	d.Params = json.RawMessage(`{"world":"test","at":[1,2,3]}`)
	f.put(t, "fight", d)
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
	f.act(t, "fight", "start")
	f.tick()
	next := f.p.NextTask()
	var replacement protocol.AttackArgs
	if next.Task != protocol.TaskAttack {
		t.Fatal(next)
	}
	if err := json.Unmarshal(next.Args, &replacement); err != nil || replacement.At == nil || *replacement.At != (protocol.Vec3{1, 2, 3}) {
		t.Fatal(next)
	}
	f.done(t, next, protocol.StatusSucceeded, "")
	f.tick()
}
