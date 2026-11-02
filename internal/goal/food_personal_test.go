package goal

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/protocol"
)

func personalFixture(t *testing.T) (*Template, *foodWorkRun, behaviour.View) {
	t.Helper()
	host, _, v, _ := foodFixture(t)
	v.Idle = true
	v.State.Time.Day = 1
	v.State.Flags = []string{protocol.FlagGrounded, protocol.FlagInShelter}
	v.State.HP, v.State.HPMax = 25, 25
	v.State.Food = &protocol.FoodStatus{}
	raw, _ := json.Marshal(Spec{Kind: "food_resupply", World: "one", Radius: 150, Reserve: 2})
	r, err := (&FoodWork{Goal: host, Personal: true}).Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	return host, r.(*foodWorkRun), v
}

func TestPersonalIdleFoodBootstrapsFromBedroomWithNoFoodAtFullBaseHealth(t *testing.T) {
	host, r, v := personalFixture(t)
	home := protocol.Vec3{-35.9, 37.66, 6.53}
	v.Home, r.spec.Center = &home, home
	v.State.Pos = protocol.Vec3{-31.25, 36.49, 15.17}
	for i := range v.Perception.Structures {
		v.Perception.Structures[i].Pos = protocol.Vec3{home[0] + float64(i), 36.09, home[2]}
	}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskMoveTo || r.EpisodeID() == "" {
		t.Fatal("empty-pack bedroom idle failed to create a real home resupply", o, err)
	}
	if err = r.Dispatched("bedroom-home"); err != nil {
		t.Fatal(err)
	}
	foodFresh(&v)
	v.State.Pos = protocol.Vec3{home[0], 36.09, home[2]}
	o, err = r.Next(v, &behaviour.Result{TaskID: "bedroom-home", Status: protocol.StatusSucceeded})
	if err != nil || o == nil || o.Task != protocol.TaskWithdraw {
		t.Fatal("confirmed home arrival did not search actual storage", o, err)
	}
	g, _ := host.Store.Get(r.EpisodeID())
	if g.Home == nil || *g.Home != home || g.Food.Reserve != 2 || g.Food.Stock != nil || g.Food.Escort != nil {
		t.Fatal(g)
	}
}

func TestPersonalFoodPressureRefillsReserveAndLeavesSatisfiedStockAlone(t *testing.T) {
	for _, kind := range []string{"empty", "reserve_low", "meal_expiring", "satisfied"} {
		t.Run(kind, func(t *testing.T) {
			host, r, v := personalFixture(t)
			v.State.Food.Active = []protocol.ActiveFood{{Prefab: "Mushroom", RemainingS: 600}}
			v.Inventory.Items = []protocol.InventoryItem{{Prefab: "MushroomYellow", Stack: 2}}
			switch kind {
			case "empty":
				v.State.Food.Active, v.Inventory.Items = nil, nil
			case "reserve_low":
				v.Inventory.Items[0].Stack = 1
			case "meal_expiring":
				v.State.Food.Active[0].RemainingS = 299
			}
			before := len(host.Store.List())
			o, err := r.Next(v, nil)
			if kind == "satisfied" {
				if err != nil || o != nil || r.EpisodeID() != "" || len(host.Store.List()) != before {
					t.Fatal("satisfied needs created another durable goal", o, err)
				}
			} else if err != nil || r.EpisodeID() == "" || len(host.Store.List()) != before+1 {
				t.Fatal("food pressure was ignored", o, err)
			}
		})
	}
}

func TestPersonalFoodAdmissionPreservesSleepCommandsAndHomeScope(t *testing.T) {
	for _, kind := range []string{"sleeping", "in_bed", "night", "unknown_clock", "busy", "stale", "stale_scan", "foreign", "home_changed", "under_floor", "outside", "wet", "airborne", "threat", "missing_food", "nonfinite"} {
		t.Run(kind, func(t *testing.T) {
			host, r, v := personalFixture(t)
			switch kind {
			case "sleeping", "in_bed":
				v.State.Flags = append(v.State.Flags, kind)
			case "night":
				v.State.Time.Night = true
			case "unknown_clock":
				v.State.Time.Day = 0
			case "busy":
				v.Idle = false
			case "stale":
				v.StateAt = v.Now.Add(-4 * time.Second)
			case "stale_scan":
				v.PerceptionAt = v.Now.Add(-4 * time.Second)
			case "foreign":
				v.State.World = "another"
			case "home_changed":
				v.Home = &protocol.Vec3{1, 0, 0}
			case "under_floor":
				v.State.Pos[1] = -2
			case "outside":
				v.State.Flags = []string{protocol.FlagGrounded}
			case "wet":
				v.State.Flags = append(v.State.Flags, protocol.FlagSwimming)
			case "airborne":
				v.State.Flags = []string{protocol.FlagInShelter}
			case "threat":
				v.Perception.Characters = []protocol.PerceivedCharacter{{HPFrac: 1, Hostile: true}}
			case "missing_food":
				v.State.Food = nil
			case "nonfinite":
				v.State.HP = math.NaN()
			}
			before := len(host.Store.List())
			if r.Check(v) == nil {
				t.Fatal("admission accepted", kind)
			}
			if o, err := r.Next(v, nil); err == nil || o != nil || r.EpisodeID() != "" || len(host.Store.List()) != before {
				t.Fatal("invalid idle context created work", o, err)
			}
		})
	}
}

func TestPersonalFoodCancellationRetainsUncertainNativeAction(t *testing.T) {
	host, r, v := personalFixture(t)
	if _, err := r.Next(v, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.Dispatched("uncertain-withdraw"); err != nil {
		t.Fatal(err)
	}
	if err := r.EndRun("new order"); err != nil {
		t.Fatal(err)
	}
	g, _ := host.Store.Get(r.EpisodeID())
	if g.Status != "cancelled" || g.Pending == nil || g.Pending.TaskID != "uncertain-withdraw" {
		t.Fatal(g)
	}
	if err := host.Queue(g.ID); err == nil {
		t.Fatal("uncertain action was replayed")
	}
}
