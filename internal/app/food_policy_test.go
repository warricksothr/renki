package app

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
	"github.com/warricksothr/renki/internal/settings"
)

func TestPendingControllerWorkChangesFoodPolicyBeforeNativeTaskState(t *testing.T) {
	a, p, _, _ := goalFixture(t, protocol.TaskUseItem)
	if err := a.Settings.Set("food.activityAware", json.RawMessage("true"), settings.ByPanel); err != nil {
		t.Fatal(err)
	}
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one", HP: 100, HPMax: 100, Food: &protocol.FoodStatus{Active: []protocol.ActiveFood{{Prefab: "Honey", RemainingS: 500}}, Carried: []protocol.CarriedFood{{Prefab: "CookedMeat", Health: 30, CanEat: true}, {Prefab: "LoxPie", Health: 75, CanEat: true}}}})
	waitFor(t, func() bool { return a.Session.State().HP == 100 && len(a.Session.State().Food.Carried) == 2 })
	if d := a.foodDecision(); d == nil || d.Mode != "idle" || d.Item != "CookedMeat" {
		t.Fatal(d)
	}
	task, err := a.Session.Dispatch(context.Background(), session.Order{Task: protocol.TaskMoveTo, Args: protocol.MoveToArgs{To: protocol.Target{Pos: &protocol.Vec3{1, 0, 0}}}})
	if err != nil {
		t.Fatal(err)
	}
	p.NextTask()
	rev := a.Session.ControlRevision()
	if d := a.foodDecision(); d == nil || d.Mode != "work" || d.Item != "LoxPie" {
		t.Fatal(d)
	}
	a.Food.Tick(context.Background())
	eat := p.NextTask()
	if eat.Task != protocol.TaskUseItem || a.Session.ControlRevision() != rev {
		t.Fatal(eat)
	}
	if status, _, _ := task.Status(); protocol.Final(status) {
		t.Fatal("feeding interrupted body")
	}
}
