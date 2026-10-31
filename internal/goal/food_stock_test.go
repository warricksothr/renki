package goal

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

func stockFixture(t *testing.T) (*Template, *runner, behaviour.View, string) {
	t.Helper()
	host, r, v, id := foodFixture(t)
	v.State.WeightMax = 300
	v.Inventory.Items = []protocol.InventoryItem{{Prefab: "Mushroom", Stack: 5}}
	if err := host.Store.update(id, func(g *Record) error {
		g.Food.Stock = &FoodStock{Targets: map[string]int{"Mushroom": 3}}
		g.Food.Storage = []string{"a"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return host, r, v, id
}
func TestHouseholdStockRequiresFreshInspectionAndMatchedDeposit(t *testing.T) {
	host, r, v, id := stockFixture(t)
	v.Containers = []*protocol.Container{{Entity: "a", Items: []protocol.InventoryItem{{Prefab: "Mushroom", Stack: 999}}}}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskWithdraw {
		t.Fatal("trusted map hint", o, err)
	}
	a := o.Args.(*protocol.TransferArgs)
	if a.Container != "a" || a.Items[0].Count != 1 {
		t.Fatal(a)
	}
	last := foodTransfer(t, r, &v, "inspect", "a", meals[0], 0, nil)
	o, err = r.Next(v, last)
	if err != nil || o == nil || o.Task != protocol.TaskDeposit {
		t.Fatal(o, err)
	}
	a = o.Args.(*protocol.TransferArgs)
	if a.Items[0].Prefab != "Mushroom" || a.Items[0].Count != 3 {
		t.Fatal("did not retain two personal meals", a)
	}
	if err = r.Dispatched("deposit"); err != nil {
		t.Fatal(err)
	}
	foodFresh(&v)
	v.Inventory.Items[0].Stack = 2
	// A receipt from another task cannot establish a successful deposit.
	v.Containers = []*protocol.Container{{Entity: "a", Task: "other", Direction: protocol.TaskDeposit, Items: []protocol.InventoryItem{{Prefab: "Mushroom", Stack: 3}}, Moved: map[string]int{"Mushroom": 3}}}
	g, _ := host.Store.Get(id)
	if err = r.foodResult(g, v, &behaviour.Result{TaskID: "deposit", Status: protocol.StatusSucceeded}); err == nil {
		t.Fatal("accepted another task's stock")
	}
	v.Containers[0].Task = "deposit"
	o, err = r.Next(v, &behaviour.Result{TaskID: "deposit", Status: protocol.StatusSucceeded})
	if err != nil || o != nil {
		t.Fatal(o, err)
	}
	g, _ = host.Store.Get(id)
	if g.Status != "succeeded" || g.Food.Stock.Known["Mushroom"] != 3 || len(g.Food.Receipts) != 1 || g.Food.Receipts[0].Action != "stock_deposit" {
		t.Fatal(g)
	}
}
func TestHouseholdUnknownBusyAndOldRunStockCannotBecomeZeroOrSuccess(t *testing.T) {
	for _, mode := range []string{"old", "busy", "missing", "expired"} {
		t.Run(mode, func(t *testing.T) {
			host, r, v, id := stockFixture(t)
			if err := host.Store.update(id, func(g *Record) error {
				g.Home = v.Home
				g.Food.SearchPinned = true
				g.Food.Phase = "ready"
				g.Food.Chests = []FoodChest{{Entity: "a", Pos: protocol.Vec3{1, 0, 0}, Observed: true, ObservedAt: v.Now, Run: host.Store.runID, Contents: map[string]int{"Mushroom": 10}}}
				c := &g.Food.Chests[0]
				switch mode {
				case "old":
					c.Run = "before-restart"
				case "busy":
					c.Unavailable = true
				case "missing":
					g.Food.Chests = nil
				case "expired":
					c.ObservedAt = v.Now.Add(-4 * time.Minute)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			o, err := r.Next(v, nil)
			g, _ := host.Store.Get(id)
			if err != nil || g.Status == "succeeded" {
				t.Fatal(o, g, err)
			}
			if mode == "old" || mode == "expired" {
				if o == nil || o.Task != protocol.TaskWithdraw {
					t.Fatal("did not re-inspect", o)
				}
			} else if g.Status != "needs_plan" {
				t.Fatal("unknown stock treated as usable", g)
			}
		})
	}
}
func TestHouseholdKeepsCollectingThenCommitsDeliveryOnCapacity(t *testing.T) {
	host, r, v, id := stockFixture(t)
	v.State.Pos = protocol.Vec3{20, 0, 0}
	v.Inventory.Items[0].Stack = 3 // personal2 + cargo1 toward target3
	v.Perception.Resources = []protocol.PerceivedResource{{ID: "berries", Kind: protocol.ResourcePickable, Pos: protocol.Vec3{21, 0, 0}, Yield: &protocol.ResourceYield{Prefab: "Mushroom", Count: 1, Weight: 1, StackMax: 20}}}
	if err := host.Store.update(id, func(g *Record) error {
		g.Home = v.Home
		g.Food.SearchPinned = true
		g.Food.Phase = "ready"
		g.Food.Chests = []FoodChest{{Entity: "a", Pos: protocol.Vec3{1, 0, 0}, Observed: true, ObservedAt: v.Now, Run: host.Store.runID}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskInteract {
		t.Fatal("returned after just one meal", o, err)
	}
	// A different fixture avoids inventing a result for the admitted interaction.
	host, r, v, id = stockFixture(t)
	v.State.Pos = protocol.Vec3{20, 0, 0}
	v.State.Weight = 220
	if err = host.Store.update(id, func(g *Record) error {
		g.Home = v.Home
		g.Food.SearchPinned = true
		g.Food.Phase = "ready"
		g.Food.Chests = []FoodChest{{Entity: "a", Pos: protocol.Vec3{1, 0, 0}, Observed: true, ObservedAt: v.Now, Run: host.Store.runID}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	o, err = r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskMoveTo {
		t.Fatal(o, err)
	}
	g, _ := host.Store.Get(id)
	if !g.Food.Stock.Delivering || !o.Args.(*protocol.MoveToArgs).AvoidHostiles {
		t.Fatal(g, o)
	}
}
func TestHouseholdSourcesArePinnedAndDangerRanked(t *testing.T) {
	host, r, v, id := stockFixture(t)
	v.Inventory.Items[0].Stack = 2
	v.Waypoints = []home.Waypoint{{Name: "near", Kind: "berries", Pos: protocol.Vec3{20, 0, 0}, Radius: 4, At: v.Now}, {Name: "far", Kind: "berries", Pos: protocol.Vec3{30, 0, 0}, Radius: 4, At: v.Now}}
	v.Danger = []home.DangerZone{{Pos: v.Waypoints[0].Pos, Radius: 10, Score: 2}}
	if err := host.Store.update(id, func(g *Record) error {
		g.Food.Stock.Sources = []string{"near", "far"}
		g.Home = v.Home
		g.Food.SearchPinned = true
		g.Food.Phase = "ready"
		g.Food.Chests = []FoodChest{{Entity: "a", Pos: protocol.Vec3{1, 0, 0}, Observed: true, ObservedAt: v.Now, Run: host.Store.runID}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskMoveTo || *o.Args.(*protocol.MoveToArgs).To.Pos != v.Waypoints[1].Pos {
		t.Fatal(o, err)
	}
	v.Waypoints[1].At = v.Now.Add(time.Second)
	if err = r.Check(v); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatal("edited source remained valid", err)
	}
}
func TestHouseholdEpisodeLifecycleKeepsUncertainTaskAndSurvivesRestart(t *testing.T) {
	host, _, v, id := stockFixture(t)
	g, _ := host.Store.Get(id)
	spec := Spec{Kind: "food_resupply", World: g.World, Center: g.Center, Radius: g.Radius, Reserve: 2, Storage: []string{"a"}, Stock: &FoodStock{Targets: map[string]int{"Mushroom": 3}}}
	raw, _ := json.Marshal(spec)
	factory := &FoodWork{Goal: host}
	run, err := factory.Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	episode := run.(*foodWorkRun)
	if _, err = episode.Next(v, nil); err != nil {
		t.Fatal(err)
	}
	if err = episode.Dispatched("uncertain"); err != nil {
		t.Fatal(err)
	}
	if err = episode.EndRun("new order"); err != nil {
		t.Fatal(err)
	}
	saved, _ := host.Store.Get(episode.run.id)
	if saved.Status != "cancelled" || saved.Pending == nil || saved.Pending.TaskID != "uncertain" {
		t.Fatal(saved)
	}
	reopened, err := Open(filepath.Join(filepath.Dir(host.Store.path), "goals.json"))
	if err != nil {
		t.Fatal(err)
	}
	saved, _ = reopened.Get(saved.ID)
	if err = (&Template{Store: reopened}).Queue(saved.ID); err == nil {
		t.Fatal("replayed uncertain transfer")
	}
	// A second equal definition receives a different durable identity.
	other, _ := factory.Start(raw)
	_, err = other.Next(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	if other.(*foodWorkRun).run.id == episode.run.id {
		t.Fatal("episode identity collision")
	}
}
func TestHouseholdDepositWaitsForFreshPackAndBlocksUnprovenFullDestination(t *testing.T) {
	host, r, v, id := stockFixture(t)
	_, _ = r.Next(v, nil)
	last := foodTransfer(t, r, &v, "probe", "a", meals[0], 0, nil)
	_, _ = r.Next(v, last)
	_ = r.Dispatched("partial")
	g, _ := host.Store.Get(id)
	v.Containers = []*protocol.Container{{Entity: "a", Task: "partial", Direction: protocol.TaskDeposit, Items: []protocol.InventoryItem{{Prefab: "Mushroom", Stack: 1}}, Moved: map[string]int{"Mushroom": 1}}}
	last = &behaviour.Result{TaskID: "partial", Status: protocol.StatusFailed, Reason: protocol.ReasonInventoryFull}
	if err := r.foodResult(g, v, last); !errors.Is(err, behaviour.ErrWait) {
		t.Fatal(err)
	}
	foodFresh(&v)
	v.Inventory.Items[0].Stack = 4
	o, err := r.Next(v, last)
	g, _ = host.Store.Get(id)
	if err != nil || o != nil || g.Status != "needs_plan" || len(g.Food.Receipts) != 1 {
		t.Fatal(o, g, err)
	}
}

func TestHouseholdCookingReturnsToWaitPointBeforeEatingOrHealing(t *testing.T) {
	host, r, v, id := stockFixture(t)
	v.State.Pos = protocol.Vec3{3, 0, 1}
	v.State.HP, v.State.HPMax = 50, 80
	v.State.Food.Carried = []protocol.CarriedFood{{Prefab: "NeckTailGrilled", CanEat: true, Health: 35, Regen: 2}}
	v.Inventory.Items = append(v.Inventory.Items, protocol.InventoryItem{Prefab: "NeckTailGrilled", Stack: 1})
	if err := host.Store.update(id, func(g *Record) error { g.Home = v.Home; g.Food.Stock.ReturnAfterCooking = true; return nil }); err != nil {
		t.Fatal(err)
	}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskMoveTo || o.Args.(*protocol.MoveToArgs).StopDistance == nil || *o.Args.(*protocol.MoveToArgs).StopDistance > .5 {
		t.Fatal("waited or ate beside the fire", o, err)
	}
	g, _ := host.Store.Get(id)
	if g.Pending.Template != "cook_return" {
		t.Fatal(g.Pending)
	}
}
func TestHouseholdNewGoalRejectsInjectedCheckpointAndInvalidLimits(t *testing.T) {
	base := Spec{Kind: "food_resupply", World: "one", Radius: 200, Storage: []string{"a"}, Stock: &FoodStock{Targets: map[string]int{"Mushroom": 4}}}
	for _, mode := range []string{"checkpoint", "missing-storage", "count", "duplicate", "escort"} {
		t.Run(mode, func(t *testing.T) {
			raw, _ := json.Marshal(base)
			var spec Spec
			_ = json.Unmarshal(raw, &spec)
			switch mode {
			case "checkpoint":
				spec.Stock.Known = map[string]int{"Mushroom": 4}
			case "missing-storage":
				spec.Storage = nil
			case "count":
				spec.Stock.Targets["Mushroom"] = 21
			case "duplicate":
				spec.Storage = []string{"a", "a"}
			case "escort":
				spec.Escort = &protocol.EscortArgs{Player: "Nimah"}
			}
			if _, err := NewRecord(spec, time.Now()); err == nil {
				t.Fatal("accepted invalid stock spec", mode)
			}
		})
	}
}
