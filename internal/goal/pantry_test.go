package goal

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func pantryReady(t *testing.T, h *Template, id string, v behaviour.View, item string, count int) {
	t.Helper()
	if err := h.Store.update(id, func(g *Record) error {
		g.Home = v.Home
		n := uint64(0)
		g.Food.Injury = &n
		g.Food.SearchPinned = true
		g.Food.Phase = "ready"
		g.Food.Stock.Targets = map[string]int{item: 4}
		g.Food.Stock.Minimum = map[string]int{item: 2}
		g.Food.Chests = []FoodChest{{Entity: "a", Pos: protocol.Vec3{1, 0, 0}, Observed: true, Run: h.Store.runID, ObservedAt: v.Now, Contents: map[string]int{item: count}}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
func TestPantryMinimumTriggersRefillUntilHighTarget(t *testing.T) {
	for _, tc := range []struct {
		count           int
		refilling, done bool
	}{{2, false, true}, {3, false, true}, {1, false, false}, {2, true, false}, {4, true, true}} {
		t.Run(fmt.Sprint(tc), func(t *testing.T) {
			h, r, v, id := stockFixture(t)
			pantryReady(t, h, id, v, "Mushroom", tc.count)
			h.Store.update(id, func(g *Record) error { g.Food.Stock.Replenishing = tc.refilling; return nil })
			v.Perception.Resources = []protocol.PerceivedResource{forageResource("fresh", "Mushroom", protocol.Vec3{10, 0, 0})}
			o, err := r.Next(v, nil)
			g, _ := h.Store.Get(id)
			if err != nil || (g.Status == "succeeded") != tc.done {
				t.Fatal(o, g.Status, g.Reason, err)
			}
			if !tc.done && (o == nil || o.Task != protocol.TaskDeposit || !g.Food.Stock.Replenishing) {
				t.Fatal(o, g)
			}
		})
	}
}
func TestPantryHuntOnlyMissingIngredientAndOwnedCombat(t *testing.T) {
	for _, species := range []string{"Boar", "Neck"} {
		for _, scenario := range []string{"allowed", "opt-out", "revoked", "tamed", "starred", "multiple", "wrong-item"} {
			t.Run(species+scenario, func(t *testing.T) {
				h, r, v, id := stockFixture(t)
				item := cooked[preyIngredient(species)]
				pantryReady(t, h, id, v, item, 0)
				h.Allowed = func(string) bool { return scenario != "revoked" }
				v.State.Stamina = 60
				if err := h.Store.update(id, func(g *Record) error { g.Food.Stock.AllowHunting = scenario != "opt-out"; return nil }); err != nil {
					t.Fatal(err)
				}
				c := protocol.PerceivedCharacter{ID: "prey", Kind: protocol.CharacterCreature, Prefab: species, HPFrac: 1, Hostile: true, Pos: protocol.Vec3{20, 0, 0}}
				switch scenario {
				case "tamed":
					c.Kind = protocol.CharacterTamed
				case "starred":
					c.Stars = 1
				case "wrong-item":
					if species == "Boar" {
						c.Prefab = "Neck"
					} else {
						c.Prefab = "Boar"
					}
				}
				v.Perception.Characters = []protocol.PerceivedCharacter{c}
				if scenario == "multiple" {
					other := c
					other.ID = "extra"
					v.Perception.Characters = append(v.Perception.Characters, other)
				}
				o, err := r.Next(v, nil)
				if err != nil {
					t.Fatal(err)
				}
				if scenario != "allowed" {
					if o != nil && o.Task == protocol.TaskAttack {
						t.Fatal("ineligible hunt", o)
					}
					return
				}
				if o == nil || o.Task != protocol.TaskAttack {
					g, _ := h.Store.Get(id)
					t.Fatal(o, g.Reason)
				}
				if err = r.Dispatched("hunt"); err != nil {
					t.Fatal(err)
				}
				v.State.Task = &protocol.TaskRef{ID: "hunt", Task: protocol.TaskAttack, Status: protocol.StatusRunning}
				v.State.Combat = &protocol.CombatStatus{ID: "hunt", Entity: "prey", Mode: "strike"}
				v.Perception.Characters[0].Pos[0] = 2
				n := uint64(1)
				v.InjurySeq = &n
				v.Injury = &session.InjuryStatus{Sequence: 1, Damage: 5, Complete: true}
				v.State.HP = 45
				if err = r.Check(v); err != nil {
					t.Fatal("owned hunt cancelled", err)
				}
				v.Perception.Characters = append(v.Perception.Characters, protocol.PerceivedCharacter{ID: "extra", HPFrac: 1, Hostile: true, Pos: protocol.Vec3{3, 0, 0}})
				if err = r.Check(v); err == nil {
					t.Fatal("extra enemy exempted")
				}
				v.Perception.Characters = v.Perception.Characters[:1]
				v.State.Combat.Entity = "other"
				if err = r.Check(v); err == nil {
					t.Fatal("other combat exempted")
				}
				v.State.Combat.Entity = "prey"
				v.Injury.Damage = 11
				if err = r.Check(v); err == nil {
					t.Fatal("hunt renewed damage allowance")
				}
				v.Injury.Damage = 5
				foodFresh(&v)
				v.Perception.Seq += 10
				v.State.Task = nil
				v.State.Combat = nil
				last := &behaviour.Result{TaskID: "hunt", Entity: "prey", Status: protocol.StatusSucceeded, Detail: "observed " + species + " death"}
				_, err = r.Next(v, last)
				var delay *behaviour.Delay
				if !errors.As(err, &delay) {
					t.Fatal(err)
				}
				g, _ := h.Store.Get(id)
				if *g.Food.Injury != 1 || len(g.Food.Receipts) != 1 || g.Food.Receipts[0].Action != "hunt" {
					t.Fatalf("reason=%s injury=%v receipts=%+v budget=%+v", g.Reason, *g.Food.Injury, g.Food.Receipts, g.Food.TravelBudget)
				}
				if err = r.Check(v); err != nil {
					t.Fatal("stale perception cancelled confirmed death settlement", err)
				}
				extra := protocol.PerceivedCharacter{ID: "extra", HPFrac: 1, Hostile: true, Pos: protocol.Vec3{3, 0, 0}}
				v.Perception.Characters = append(v.Perception.Characters, extra)
				if err = r.Check(v); err == nil {
					t.Fatal("death settlement exempted a new enemy")
				}
				v.Perception.Characters = v.Perception.Characters[:1]
				later := v
				later.Now = v.Now.Add(3 * time.Second)
				later.StateAt = later.Now
				later.PerceptionAt = later.Now
				if err = r.Check(later); err == nil {
					t.Fatal("death settlement exceeded its original two seconds")
				}
				v.Perception.Characters = nil
				foodFresh(&v)
				v.Perception.Seq += 10
				p := forageResource("loot", preyIngredient(species), c.Pos)
				p.Kind = protocol.ResourceDrop
				v.Perception.Resources = []protocol.PerceivedResource{p}
				o, err = r.Next(v, nil)
				if err != nil || o == nil || o.Task != protocol.TaskPickup {
					t.Fatal(o, err)
				}
			})
		}
	}
}
func TestPantryAreaRotationAndCooldownSurviveRestart(t *testing.T) {
	h, r, v, id := stockFixture(t)
	pantryReady(t, h, id, v, "CookedMeat", 0)
	v.State.Stamina = 60
	h.Allowed = func(string) bool { return true }
	areas := []home.Waypoint{{Name: "west", Kind: "hunting", Pos: protocol.Vec3{20, 0, 0}, Radius: 12}, {Name: "east", Kind: "hunting", Pos: protocol.Vec3{-25, 0, 0}, Radius: 12}}
	v.Waypoints = areas
	if err := h.Store.update(id, func(g *Record) error {
		g.Food.Stock.AllowHunting = true
		g.Food.Stock.HuntAreas = []string{"west", "east"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Args.(*protocol.MoveToArgs).To.Pos[0] != 20 {
		t.Fatal(o, err)
	}
	if err = r.Dispatched("visit"); err != nil {
		t.Fatal(err)
	}
	v.State.Pos = areas[0].Pos
	foodFresh(&v)
	v.Perception.Seq += 10
	o, err = r.Next(v, &behaviour.Result{TaskID: "visit", Status: protocol.StatusSucceeded})
	if err != nil || o == nil || o.Args.(*protocol.MoveToArgs).To.Pos[0] != -25 {
		t.Fatal("did not visit another area", o, err)
	}
	// A long production episode can replace every early native outcome.
	// Neither that nor an uncertain dispatch may erase area feedback.
	if err := h.Store.update(id, func(g *Record) error {
		if len(g.Food.Acquisition.AreaVisits) != 2 {
			t.Fatal("area dispatches were not retained", g.Food.Acquisition.AreaVisits)
		}
		g.Pending = nil
		g.History = make([]Outcome, 16)
		for i := range g.History {
			g.History[i] = Outcome{Template: "cook", Status: protocol.StatusSucceeded, At: v.Now}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(h.Store.path)
	if err != nil {
		t.Fatal(err)
	}
	r.t.Store = reopened
	g, _ := reopened.Get(id)
	g.ID = "next-episode"
	if !r.foodAreaCooling(g, "west", v.Now) || !r.foodAreaCooling(g, "east", v.Now) || r.foodAreaCooling(g, "west", v.Now.Add(16*time.Minute)) {
		t.Fatal("cooldown lost or did not expire")
	}
}
func TestPantryRejectsInjectedCheckpointAndInvalidMinimums(t *testing.T) {
	for _, stock := range []FoodStock{
		{Targets: map[string]int{"Mushroom": 4}, Minimum: map[string]int{"Mushroom": 5}},
		{Targets: map[string]int{"Mushroom": 4}, Minimum: map[string]int{"Raspberry": 1}},
		{Targets: map[string]int{"Mushroom": 4}, HuntAreas: []string{"west"}},
		{Targets: map[string]int{"Mushroom": 4}, Replenishing: true},
		{Targets: map[string]int{"Mushroom": 4}, HuntingPinned: []home.Waypoint{{Name: "west"}}},
	} {
		raw, _ := json.Marshal(Spec{Kind: "food_resupply", World: "one", Radius: 200, Storage: []string{"a"}, Stock: &stock})
		if err := (&FoodWork{}).Validate(raw); err == nil {
			t.Fatal("accepted invalid/injected stock", string(raw))
		}
	}
}
func TestGoalRetentionPreservesUncertainRecentAndAtomicWrites(t *testing.T) {
	for _, scenario := range []string{"rotate", "uncertain", "recent", "write-fails"} {
		t.Run(scenario, func(t *testing.T) {
			s, err := Open(filepath.Join(t.TempDir(), "goals.json"))
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC()
			for i := 0; i < MaxRecords; i++ {
				g, _ := NewRecord(Spec{Kind: "food_resupply", World: "one", Radius: 150}, now.Add(-time.Hour))
				g.ID = fmt.Sprint(i)
				g.Status = "needs_plan"
				if scenario == "uncertain" {
					g.Pending = &Intent{Run: "old", TaskID: "maybe", Template: "eat", Entity: ""}
				}
				if scenario == "recent" {
					g.Created = now
					g.Deadline = now.Add(10 * time.Minute)
				}
				if err = s.Add(g); err != nil {
					t.Fatal(err)
				}
			}
			g, _ := NewRecord(Spec{Kind: "food_resupply", World: "one", Radius: 150}, now)
			if scenario == "write-fails" {
				s.write = func([]byte) error { return errors.New("disk full") }
			}
			err = s.Add(g)
			if (err == nil) != (scenario == "rotate") {
				t.Fatal(err)
			}
			if len(s.List()) != MaxRecords {
				t.Fatal("retention size")
			}
			_, exists := s.Get(g.ID)
			if exists != (scenario == "rotate") {
				t.Fatal("transactional write lost")
			}
			if scenario == "uncertain" && !strings.Contains(err.Error(), "unresolved") {
				t.Fatal(err)
			}
		})
	}
}

// Exercise the composed host loop with protocol receipts, including repeated
// cooking/deposit in one refill. A kill alone cannot satisfy household stock.
func TestPantryHuntLootReturnCookAndDepositToHighTarget(t *testing.T) {
	h, r, v, id := stockFixture(t)
	pantryReady(t, h, id, v, "CookedMeat", 0)
	h.Allowed = func(string) bool { return true }
	v.State.Stamina = 60
	v.Inventory.Items = []protocol.InventoryItem{{Prefab: "Mushroom", Stack: 2}}
	if err := h.Store.update(id, func(g *Record) error {
		g.Food.Stock.Targets["CookedMeat"] = 2
		g.Food.Stock.Minimum["CookedMeat"] = 1
		g.Food.Stock.AllowHunting = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "boar", Kind: protocol.CharacterCreature, Prefab: "Boar", HPFrac: 1, Hostile: true, Pos: protocol.Vec3{20, 0, 0}}}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskAttack {
		t.Fatal(o, err)
	}
	last := acquireResult(t, r, &v, "kill", protocol.StatusSucceeded, "")
	last.Entity = "boar"
	last.Detail = "observed Boar death"
	v.Perception.Characters = nil
	v.State.Pos = protocol.Vec3{20, 0, 0}
	o, err = r.Next(v, last)
	var delay *behaviour.Delay
	if o != nil || !errors.As(err, &delay) {
		t.Fatal(o, err)
	}
	foodFresh(&v)
	p := forageResource("meat", "RawMeat", v.State.Pos)
	p.Kind = protocol.ResourceDrop
	p.Yield.Count = 2
	v.Perception.Resources = []protocol.PerceivedResource{p}
	o, err = r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskPickup {
		t.Fatal(o, err)
	}
	last = acquireResult(t, r, &v, "loot", protocol.StatusSucceeded, "")
	v.Perception.Resources = nil
	v.Inventory.Items = append(v.Inventory.Items, protocol.InventoryItem{Prefab: "RawMeat", Stack: 2})
	o, err = r.Next(v, last)
	if err != nil || o == nil || o.Task != protocol.TaskMoveTo {
		t.Fatal(o, err)
	}
	last = acquireResult(t, r, &v, "home", protocol.StatusSucceeded, "")
	v.State.Pos = protocol.Vec3{}
	o, err = r.Next(v, last)
	for batch := 1; batch <= 2; batch++ {
		if err != nil || o == nil || o.Task != protocol.TaskInteract || o.Args.(*protocol.InteractArgs).Cook != "RawMeat" {
			t.Fatal("cook", batch, o, err)
		}
		task := fmt.Sprintf("cook%d", batch)
		last = acquireResult(t, r, &v, task, protocol.StatusSucceeded, "")
		last.Cooking = deliveryRows(task, "CookedMeat", "station")
		v.Inventory.Items = []protocol.InventoryItem{{Prefab: "Mushroom", Stack: 2}, {Prefab: "RawMeat", Stack: 2 - batch}, {Prefab: "CookedMeat", Stack: 1}}
		v.State.Pos = protocol.Vec3{3, 0, 0}
		o, err = r.Next(v, last)
		if err != nil || o == nil || o.Task != protocol.TaskMoveTo {
			t.Fatal("cooking return", o, err)
		}
		last = acquireResult(t, r, &v, fmt.Sprintf("wait%d", batch), protocol.StatusSucceeded, "")
		v.State.Pos = protocol.Vec3{}
		o, err = r.Next(v, last)
		if err != nil || o == nil || o.Task != protocol.TaskDeposit || o.Args.(*protocol.TransferArgs).Items[0].Count != 1 {
			t.Fatal("deposit", o, err)
		}
		task = fmt.Sprintf("deposit%d", batch)
		last = acquireResult(t, r, &v, task, protocol.StatusSucceeded, "")
		v.Inventory.Items = v.Inventory.Items[:2]
		v.Containers = append(v.Containers, &protocol.Container{Task: task, Entity: "a", Direction: protocol.TaskDeposit, Moved: map[string]int{"CookedMeat": 1}, Items: []protocol.InventoryItem{{Prefab: "CookedMeat", Stack: batch}}})
		o, err = r.Next(v, last)
	}
	g, _ := h.Store.Get(id)
	if err != nil || o != nil || g.Status != "succeeded" || g.Food.Stock.Known["CookedMeat"] != 2 {
		t.Fatal(o, g.Reason, err)
	}
	counts := map[string]int{}
	for _, receipt := range g.Food.Receipts {
		counts[receipt.Action] += receipt.Count
	}
	if counts["hunt"] != 1 || counts["collect"] != 2 || counts["cook"] != 2 || counts["stock_deposit"] != 2 {
		t.Fatal(counts)
	}
}
