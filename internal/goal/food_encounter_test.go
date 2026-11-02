package goal

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func TestHouseholdTravelRecoveryRequiresMatchingCurrentNativeTask(t *testing.T) {
	for _, change := range []string{"matching", "other_task", "stationary", "finished", "old_run", "opt_out", "injury"} {
		t.Run(change, func(t *testing.T) {
			host, r, v, id := stockFixture(t)
			g, _ := host.Store.Get(id)
			pos := protocol.Vec3{30, 0, 0}
			o, err := r.foodDispatch(g, v, "search_food", "", "marker:patch", &session.Order{Task: protocol.TaskMoveTo,
				Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}, Dry: true, AvoidHostiles: true}, TimeoutS: 180})
			if err != nil || o == nil || !o.Args.(*protocol.MoveToArgs).RecoverThreats {
				t.Fatal(o, err)
			}
			if err = r.Dispatched("journey"); err != nil {
				t.Fatal(err)
			}
			v.State.Task = &protocol.TaskRef{ID: "journey", Task: protocol.TaskMoveTo, Status: protocol.StatusRunning}
			v.State.Journey = &protocol.JourneyStatus{ID: "journey", Mode: "escape", Attempts: 1}
			v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "neck", Prefab: "Neck", Hostile: true, TargetingMe: true, HPFrac: 1, Pos: protocol.Vec3{10, 0, 0}}}
			switch change {
			case "other_task":
				v.State.Task.ID = "owner-order"
			case "stationary":
				v.State.Task.Task = protocol.TaskInteract
			case "finished":
				v.State.Task.Status = protocol.StatusSucceeded
			}
			if err = host.Store.update(id, func(g *Record) error {
				n := uint64(0)
				g.Food.Injury = &n
				if change == "old_run" {
					g.Pending.Run = "prior-controller"
				}
				if change == "opt_out" {
					a := *o.Args.(*protocol.MoveToArgs)
					a.RecoverThreats = false
					g.Pending.Params, _ = json.Marshal(a)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if change == "injury" {
				n := uint64(1)
				v.InjurySeq = &n
			}
			err = r.Check(v)
			if (err == nil) != (change == "matching" || change == "finished") {
				t.Fatal(change, err)
			}
			g, _ = host.Store.Get(id)
			if r.nativeTravelRecovery(g, v) != (change == "matching" || change == "injury") {
				t.Fatal("non-running/foreign task acquired native recovery permission")
			}
			if g.Status != "running" || g.Pending.TaskID != "journey" || len(g.Food.Receipts) != 0 {
				t.Fatal("guard mutated the task or invented production", g)
			}
		})
	}
}

func TestFoodTravelRecoveryAndDefenseApplyToPersonalAndHouseholdGoals(t *testing.T) {
	for _, household := range []bool{false, true} {
		for _, combat := range []bool{false, true} {
			host, r, v, id := foodFixture(t)
			host.Allowed = func(task string) bool { return task != protocol.TaskAttack || combat }
			if household {
				if err := host.Store.update(id, func(g *Record) error {
					g.Food.Stock = &FoodStock{Targets: map[string]int{"Mushroom": 3}}
					g.Food.Storage = []string{"a"}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			}
			g, _ := host.Store.Get(id)
			o, err := r.foodDispatch(g, v, "food_return", "", "", &session.Order{Task: protocol.TaskMoveTo,
				Args: &protocol.MoveToArgs{To: protocol.Target{Pos: v.Home}, Dry: true, AvoidHostiles: true}, TimeoutS: 120})
			if err != nil || o == nil {
				t.Fatal(o, err)
			}
			a := o.Args.(*protocol.MoveToArgs)
			if !a.RecoverThreats || a.DefendThreats != combat {
				t.Fatal("personal/household continuation or attack permission lost", a)
			}
		}
	}
}

func TestFoodCompletedTransferIsAccountedBeforeThreatOrInjuryStopsWork(t *testing.T) {
	for _, incident := range []string{"threat", "injury", "mismatched_receipt"} {
		t.Run(incident, func(t *testing.T) {
			host, r, v, id := foodFixture(t)
			o, err := r.Next(v, nil)
			if err != nil || o == nil || o.Task != protocol.TaskWithdraw {
				t.Fatal(o, err)
			}
			a := o.Args.(*protocol.TransferArgs)
			last := foodTransfer(t, r, &v, "withdraw-before-encounter", a.Container, a.Items[0].Prefab, 1, nil)
			if incident == "injury" {
				n := uint64(1)
				v.InjurySeq, v.State.HP = &n, 48
			} else {
				v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "neck", HPFrac: 1, Hostile: true, Pos: protocol.Vec3{10, 0, 0}}}
			}
			if incident == "mismatched_receipt" {
				v.Containers[0].Task = "another-action"
			}
			if err = r.Check(v); err != nil {
				t.Fatal("receipt boundary was cancelled before reconciliation", err)
			}
			o, err = r.Next(v, last)
			g, _ := host.Store.Get(id)
			if err != nil || o != nil || g.Status != "needs_plan" {
				t.Fatal(o, g, err)
			}
			if incident == "mismatched_receipt" {
				if len(g.Food.Receipts) != 0 || g.Pending == nil {
					t.Fatal("uncertain transfer invented progress or became replayable", g)
				}
			} else if len(g.Food.Receipts) != 1 || g.Food.Receipts[0].TaskID != last.TaskID || g.Food.Receipts[0].Count != 1 || g.Pending != nil {
				t.Fatal("verified work was lost or left replayable", g)
			}
		})
	}
}

func TestOwnedFoodTravelHasOneFiniteDamageAllowance(t *testing.T) {
	for _, scenario := range []string{"minor", "exact_budget", "over_budget", "healed_damage", "large_hit", "missing_counter", "rolled_counter", "unknown_damage", "foreign_task", "stationary", "old_run", "water", "stale"} {
		t.Run(scenario, func(t *testing.T) {
			host, r, v, id := foodFixture(t)
			if err := host.Store.update(id, func(g *Record) error { g.Home = v.Home; n := uint64(0); g.Food.Injury = &n; return nil }); err != nil {
				t.Fatal(err)
			}
			g, _ := host.Store.Get(id)
			o, err := r.foodDispatch(g, v, "food_return", "", "", &session.Order{Task: protocol.TaskMoveTo,
				Args: &protocol.MoveToArgs{To: protocol.Target{Pos: v.Home}, Dry: true, AvoidHostiles: true}, TimeoutS: 120})
			if err != nil || o == nil {
				t.Fatal(o, err)
			}
			if err = r.Dispatched("owned-travel"); err != nil {
				t.Fatal(err)
			}
			v.State.Task = &protocol.TaskRef{ID: "owned-travel", Task: protocol.TaskMoveTo, Status: protocol.StatusRunning}
			v.State.Journey = &protocol.JourneyStatus{ID: "owned-travel", Mode: "defend", Attempts: 1}
			n := uint64(111)
			v.InjurySeq, v.State.HP = &n, 45
			v.Injury = &session.InjuryStatus{Sequence: n, Damage: 5, Complete: true}
			switch scenario {
			case "exact_budget":
				n = 900
				v.Injury.Damage = 10
			case "over_budget", "healed_damage":
				n = 1500
				v.Injury.Damage = 11
				v.State.HP = 50
			case "large_hit":
				v.State.HP = 39
			case "missing_counter":
				v.InjurySeq = nil
			case "unknown_damage":
				v.Injury.Complete = false
			case "rolled_counter":
				n = 0
				if err = host.Store.update(id, func(g *Record) error { g.Food.TravelBudget.Injury = 1; k := uint64(1); g.Food.Injury = &k; return nil }); err != nil {
					t.Fatal(err)
				}
			case "foreign_task":
				v.State.Task.ID = "human-order"
			case "stationary":
				v.State.Task.Task = protocol.TaskInteract
			case "old_run":
				if err = host.Store.update(id, func(g *Record) error { g.Pending.Run = "old-controller"; return nil }); err != nil {
					t.Fatal(err)
				}
			case "water":
				v.State.Flags = []string{protocol.FlagSwimming}
			case "stale":
				v.StateAt = v.Now.Add(-10 * time.Second)
			}
			v.Injury.Sequence = n
			err = r.Check(v)
			if (err == nil) != (scenario == "minor" || scenario == "exact_budget") {
				t.Fatal(scenario, err)
			}
			g, _ = host.Store.Get(id)
			if g.Pending.TaskID != "owned-travel" || g.TaskCount != 1 || len(g.Food.Receipts) != 0 {
				t.Fatal("encounter lost destination/budget or invented work", g)
			}
		})
	}
}

func TestFoodTravelBudgetCannotRenewAcrossAnotherTaskOrRestart(t *testing.T) {
	host, r, v, id := foodFixture(t)
	n := uint64(111)
	v.InjurySeq = &n
	v.Injury = &session.InjuryStatus{Sequence: n, Damage: 8, Complete: true}
	if err := host.Store.update(id, func(g *Record) error {
		g.Home = v.Home
		g.Food.Injury = &n
		g.Food.TravelBudget = &FoodTravelBudget{Injury: 100, Damage: 3, HP: 50}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	g, _ := host.Store.Get(id)
	o, err := r.foodDispatch(g, v, "food_return", "", "", &session.Order{Task: protocol.TaskMoveTo,
		Args: &protocol.MoveToArgs{To: protocol.Target{Pos: v.Home}, Dry: true, AvoidHostiles: true}, TimeoutS: 120})
	if err != nil || o == nil {
		t.Fatal(o, err)
	}
	if err = r.Dispatched("second-trip"); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(host.Store.path)
	if err != nil {
		t.Fatal(err)
	}
	g, _ = reopened.Get(id)
	if g.Food.TravelBudget.Injury != 100 || g.Food.TravelBudget.Damage != 3 || g.Food.TravelBudget.HP != 50 ||
		g.Pending.TaskID != "second-trip" || g.Status != "needs_plan" {
		t.Fatal("baseline reset or uncertain travel became replayable", g)
	}
	if err = (&Template{Store: reopened}).Queue(id); err == nil {
		t.Fatal("replayed uncertain travel after reconnect")
	}
}

func TestFoodTravelResultAcknowledgesOnlyBoundedDamageAndRetainsBudget(t *testing.T) {
	for _, excessive := range []bool{false, true} {
		host, r, v, id := foodFixture(t)
		if err := host.Store.update(id, func(g *Record) error { g.Home = v.Home; n := uint64(0); g.Food.Injury = &n; return nil }); err != nil {
			t.Fatal(err)
		}
		g, _ := host.Store.Get(id)
		_, err := r.foodDispatch(g, v, "food_return", "", "", &session.Order{Task: protocol.TaskMoveTo,
			Args: &protocol.MoveToArgs{To: protocol.Target{Pos: v.Home}, Dry: true, AvoidHostiles: true}, TimeoutS: 120})
		if err != nil {
			t.Fatal(err)
		}
		if err = r.Dispatched("return"); err != nil {
			t.Fatal(err)
		}
		foodFresh(&v)
		n := uint64(111)
		if excessive {
			n = 1500
		}
		v.InjurySeq, v.State.HP = &n, 45
		v.Injury = &session.InjuryStatus{Sequence: n, Damage: 5, Complete: true}
		if excessive {
			v.Injury.Damage = 11
		}
		v.Inventory.Items = []protocol.InventoryItem{{Prefab: "Mushroom", Stack: 2}}
		o, err := r.Next(v, &behaviour.Result{TaskID: "return", Status: protocol.StatusSucceeded})
		g, _ = host.Store.Get(id)
		if err != nil || o != nil || (g.Status == "succeeded") == excessive || len(g.Food.Receipts) != 0 || g.Pending != nil {
			t.Fatal(o, g, err)
		}
		if excessive && (!strings.Contains(g.Reason, "injury") || *g.Food.Injury != 0) {
			t.Fatal(g)
		}
		if !excessive && *g.Food.Injury != 111 {
			t.Fatal("verified travel damage was not acknowledged", g)
		}
		if g.Food.TravelBudget.Injury != 0 || g.Food.TravelBudget.HP != 50 {
			t.Fatal("lifetime damage allowance was renewed", g)
		}
	}
}

func TestDistantTargetingDoesNotTerminateHouseholdWork(t *testing.T) {
	host, r, v, id := stockFixture(t)
	v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "neck", Prefab: "Neck", Hostile: true, TargetingMe: true, HPFrac: 1, Pos: protocol.Vec3{22, 0, 0}}}
	if err := r.Check(v); err != nil {
		t.Fatal(err)
	}
	o, err := r.Next(v, nil)
	g, _ := host.Store.Get(id)
	if err != nil || o == nil || o.Task != protocol.TaskWithdraw || g.Status != "running" || len(g.Food.Receipts) != 0 {
		t.Fatal("distant awareness cancelled work or became a receipt", o, g, err)
	}
}

func TestUnfedBootstrapDoesNotOptIntoFedTravelRecovery(t *testing.T) {
	host, r, v, id := stockFixture(t)
	v.State.Food = nil
	g, _ := host.Store.Get(id)
	pos := protocol.Vec3{5, 0, 0}
	o, err := r.foodDispatch(g, v, "home", "", "", &session.Order{Task: protocol.TaskMoveTo,
		Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}, Dry: true, AvoidHostiles: true}, TimeoutS: 60})
	if err != nil || o == nil || o.Args.(*protocol.MoveToArgs).RecoverThreats {
		t.Fatal("unfed bootstrap acquired a food-dependent encounter policy", o, err)
	}
}
