package food

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func policyScene() (*protocol.State, Policy) {
	return &protocol.State{Phase: protocol.PhaseInWorld, HP: 100, HPMax: 100, Food: &protocol.FoodStatus{
		Active:  []protocol.ActiveFood{{Prefab: "Honey", RemainingS: 600}},
		Carried: []protocol.CarriedFood{{Prefab: "DeerStew", Health: 45, Regen: 2, CanEat: true}, {Prefab: "CookedMeat", Health: 30, Regen: 2, CanEat: true}, {Prefab: "LoxPie", Health: 75, Regen: 4, CanEat: true}, {Prefab: "ModMeal", Health: 90, Regen: 5, CanEat: true}},
	}}, Policy{ActivityAware: true, IdleMaxTier: 2}
}

func TestMealModesAndUnknownTierReservation(t *testing.T) {
	s, p := policyScene()
	for _, tc := range []struct {
		mode, work, item string
		hp               float64
		empty            bool
	}{
		{"idle", "", "CookedMeat", 100, false}, {"work", "escort", "ModMeal", 100, false},
		{"emergency", "", "ModMeal", 50, false}, {"emergency", "", "ModMeal", 100, true},
	} {
		s.HP = tc.hp
		old := s.Food.Active
		if tc.empty {
			s.Food.Active = nil
		}
		d := Decide(s, p, tc.work)
		s.Food.Active = old
		if d.Mode != tc.mode || d.Item != tc.item {
			t.Fatal(tc, d)
		}
	}
	s.HP = 100
	p.Tiers = map[string]int{"ModMeal": 1}
	if d := Decide(s, p, ""); d.Item != "ModMeal" {
		t.Fatal(d)
	}
	p.ActivityAware = false
	if d := Decide(s, p, ""); d.Mode != "legacy" || d.Item != Choose(s.Food) {
		t.Fatal(d)
	}
}

func TestIdleFillSlotsThenDelayRefreshAndRespectNativeEligibility(t *testing.T) {
	s, p := policyScene()
	s.Food.Carried = []protocol.CarriedFood{{Prefab: "Honey", Health: 8, CanEat: true}, {Prefab: "DeerStew", Health: 45, CanEat: true}, {Prefab: "LoxPie", Health: 75, CanEat: true}}
	if d := Decide(s, p, ""); d.Item != "DeerStew" {
		t.Fatal(d)
	}
	s.Food.Active = append(s.Food.Active, protocol.ActiveFood{Prefab: "DeerStew", RemainingS: 600}, protocol.ActiveFood{Prefab: "BoarJerky", RemainingS: 600})
	if d := Decide(s, p, ""); d.Item != "" {
		t.Fatal(d)
	}
	s.Food.Active[0].RemainingS = 90
	if d := Decide(s, p, ""); d.Item != "Honey" {
		t.Fatal(d)
	}
	s.Food.Carried[0].CanEat = false
	if d := Decide(s, p, ""); d.Item != "" {
		t.Fatal(d)
	}
	before := *s.Food
	_ = Decide(s, p, "")
	if !reflect.DeepEqual(before, *s.Food) {
		t.Fatal("selector mutated observation")
	}
}

func TestKeeperDoesNotTreatUnseenPausedStateAsFresh(t *testing.T) {
	f, k, now := scene()
	k.SetReserved(func() bool { return true })
	k.Tick(context.Background())
	*now = now.Add(time.Minute)
	k.SetReserved(nil)
	k.Tick(context.Background())
	if len(f.orders) != 0 {
		t.Fatal("stale state fed after reservation", f.orders)
	}
	fresh(f)
	k.Tick(context.Background())
	if len(f.orders) != 1 {
		t.Fatal(f.orders)
	}
}

func TestTaskClassificationAndFinalTaskDoNotInventWork(t *testing.T) {
	s, p := policyScene()
	s.Task = &protocol.TaskRef{Task: protocol.TaskEscort, Status: protocol.StatusRunning}
	if d := Decide(s, p, ""); d.Mode != "work" {
		t.Fatal(d)
	}
	s.Task.Status = protocol.StatusSucceeded
	if d := Decide(s, p, ""); d.Mode != "idle" {
		t.Fatal(d)
	}
	if d := Decide(s, p, "behavior:gather"); d.Mode != "work" {
		t.Fatal(d)
	}
}

func TestWorkValuesStaminaAndIdleChoresRetainIdleContext(t *testing.T) {
	s, p := policyScene()
	s.Food.Carried = []protocol.CarriedFood{{Prefab: "DeerStew", Health: 45, Stamina: 15, CanEat: true}, {Prefab: "Salad", Health: 25, Stamina: 80, CanEat: true}}
	if d := Decide(s, p, "escort"); d.Item != "Salad" {
		t.Fatal(d)
	}
	s.Task = &protocol.TaskRef{Task: protocol.TaskMoveTo, Status: protocol.StatusRunning}
	if d := Decide(s, p, "idle:marker_food"); d.Mode != "idle" || d.Item != "DeerStew" {
		t.Fatal(d)
	}
	s.HP = 50
	if d := Decide(s, p, "escort"); d.Mode != "emergency" || d.Item != "DeerStew" {
		t.Fatal(d)
	}
}
