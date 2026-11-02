package behaviour

import (
	"encoding/json"
	"errors"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"testing"
	"time"
)

func cargoFixture(t *testing.T) (*markerCargoRun, View) {
	t.Helper()
	raw, _ := json.Marshal(MarkerCargoParams{Marker: Marker{World: "one", Name: "work", Kind: "storage"}, Keep: map[string]int{"Coal": 2}})
	runner, err := (MarkerCargo{}).Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	v := markerView("storage")
	v.State.WorldUID = "uid"
	v.State.Flags = []string{protocol.FlagGrounded}
	v.Features = []string{protocol.FeatureMatchingDeposit}
	v.Inventory.Items = []protocol.InventoryItem{{Prefab: "Coal", Type: "Material", Stack: 7}, {Prefab: "CopperOre", Type: "Material", Stack: 3}, {Prefab: "CookedDeerMeat", Type: "Consumable", Stack: 4}, {Prefab: "AxeFlint", Type: "OneHandedWeapon", Stack: 1}, {Prefab: "Secret", Type: "unknown", Stack: 1}}
	return runner.(*markerCargoRun), v
}
func TestCargoPrefersMatchingChestAndRequiresExactReceipts(t *testing.T) {
	r, v := cargoFixture(t)
	v.Perception.Structures = []protocol.PerceivedStructure{{ID: "unknown", Kind: protocol.StructureContainer}, {ID: "coal", Kind: protocol.StructureContainer}}
	v.Containers = []*protocol.Container{{Entity: "coal", Items: []protocol.InventoryItem{{Prefab: "Coal", Stack: 10}}}}
	order, err := r.Next(v, nil)
	if err != nil || order == nil {
		t.Fatal(order, err)
	}
	a := order.Args.(*protocol.TransferArgs)
	if a.Container != "coal" || !a.MatchExisting || len(a.Items) != 1 || a.Items[0].Prefab != "Coal" || a.Items[0].Count != 5 {
		t.Fatal(a)
	}
	result := &Result{TaskID: "deposit", Status: protocol.StatusSucceeded}
	if _, err = r.Next(v, result); err != ErrWait {
		t.Fatal("unverified success", err)
	}
	v.Inventory.Seq++
	v.Inventory.Items[0].Stack = 2
	v.Containers = []*protocol.Container{{Entity: "coal", Task: "deposit", Direction: protocol.TaskDeposit, Moved: map[string]int{"Coal": 5}}}
	order, err = r.Next(v, result)
	if err != nil || order == nil || order.Args.(*protocol.TransferArgs).Items[0].Prefab != "CopperOre" {
		t.Fatal(order, err)
	}
	if got := r.surplus(v); len(got) != 1 || got["CopperOre"] != 3 {
		t.Fatal("protected cargo selected", got)
	}
}
func TestCargoOriginRanksUnknownChestButNeverGravesOrOtherFloors(t *testing.T) {
	r, v := cargoFixture(t)
	w := v.Waypoints[0]
	r.guard.marker = &w
	v.Perception.Structures = []protocol.PerceivedStructure{{ID: "near-me", Kind: protocol.StructureContainer}, {ID: "pickup-box", Kind: protocol.StructureContainer, Pos: protocol.Vec3{3, 0, 0}}, {ID: "grave", Kind: protocol.StructureTombstone}, {ID: "floor", Kind: protocol.StructureContainer, Pos: protocol.Vec3{0, 4, 0}}}
	v.Containers = []*protocol.Container{{Entity: "old-id", Items: []protocol.InventoryItem{{Prefab: "CopperOre", Stack: 99}}}, {Entity: "grave", Items: []protocol.InventoryItem{{Prefab: "CopperOre", Stack: 99}}}}
	v.Pickups = []home.Pickup{{World: "one", WorldUID: "uid", Prefab: "CopperOre", Pos: protocol.Vec3{3, 0, 0}, At: v.Now}}
	cs := r.candidates(v, "CopperOre")
	if len(cs) != 2 || cs[0].id != "pickup-box" {
		t.Fatal(cs)
	}
	v.Pickups[0].WorldUID = "foreign"
	cs = r.candidates(v, "CopperOre")
	if cs[0].id != "near-me" {
		t.Fatal("foreign origin ranked", cs)
	}
}
func TestCargoRefusedChestIsTriedOnceAndCargoIsKept(t *testing.T) {
	r, v := cargoFixture(t)
	v.Inventory.Items = v.Inventory.Items[:1]
	v.Perception.Structures = []protocol.PerceivedStructure{{ID: "a", Kind: protocol.StructureContainer}, {ID: "b", Kind: protocol.StructureContainer}}
	for _, id := range []string{"a", "b"} {
		order, err := r.Next(v, nil)
		if err != nil || order == nil || order.Args.(*protocol.TransferArgs).Container != id {
			t.Fatal(order, err)
		}
		// A native conditional refusal certifies no mutation; try the next chest.
		order, err = r.Next(v, &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonRefused})
		if id == "a" {
			if err != nil || order == nil || order.Args.(*protocol.TransferArgs).Container != "b" {
				t.Fatal(order, err)
			}
			break
		}
	}
	_, err := r.Next(v, &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonRefused})
	var need *StorageNeed
	if !errors.As(err, &need) || need.Items["Coal"] != 5 || v.Inventory.Count("Coal") != 7 || len(r.tried) != 2 {
		t.Fatal(err, r.tried)
	}
}
func TestCargoSafetyAndOldPluginRefuseBeforeDispatch(t *testing.T) {
	for _, mode := range []string{"feature", "threat", "stale", "marker", "injury", "timeout", "recovery", "below_floor", "airborne"} {
		t.Run(mode, func(t *testing.T) {
			r, v := cargoFixture(t)
			if mode == "feature" {
				v.Features = nil
			}
			if mode == "recovery" {
				v.State.Flags = []string{"recovering"}
			}
			if mode == "below_floor" {
				v.State.Pos[1] = -1.98
			}
			if mode == "airborne" {
				v.State.Flags = nil
			}
			if mode == "threat" {
				v.Perception.Characters = []protocol.PerceivedCharacter{{HPFrac: 1, Hostile: true}}
			}
			if mode == "stale" {
				v.PerceptionAt = v.Now.Add(-4 * time.Second)
			}
			if mode == "marker" {
				v.Waypoints[0].Kind = "cooking"
			}
			if mode == "injury" || mode == "timeout" {
				w := v.Waypoints[0]
				r.guard.marker = &w
				r.guard.began = v.Now
				r.guard.health = sampleHealth(v)
				if mode == "injury" {
					v.State.HP--
				} else {
					r.guard.began = v.Now.Add(-7 * time.Minute)
				}
			}
			if order, err := r.Next(v, nil); order != nil || err == nil {
				t.Fatal(order, err)
			}
		})
	}
}

func cargoJourney(t *testing.T) (*markerCargoRun, View) {
	t.Helper()
	r, v := cargoFixture(t)
	v.Inventory.Items = v.Inventory.Items[:1]
	w := home.Waypoint{Name: "workshop", Kind: "storage", Pos: protocol.Vec3{12, -2, 0}, Radius: 5, At: v.Now}
	v.Waypoints = append(v.Waypoints, w)
	v.Perception.Structures = []protocol.PerceivedStructure{{ID: "coal", Kind: protocol.StructureContainer, Pos: w.Pos}}
	v.Containers = []*protocol.Container{{Entity: "coal", Pos: w.Pos, Items: []protocol.InventoryItem{{Prefab: "Coal", Stack: 10}}}}
	return r, v
}
func TestCargoJourneyRequiresNamedNearbyStorageAndPinsBothMarkers(t *testing.T) {
	for _, mode := range []string{"valid", "missing", "far_chest", "far_marker"} {
		t.Run(mode, func(t *testing.T) {
			r, v := cargoJourney(t)
			if mode == "missing" {
				v.Waypoints = v.Waypoints[:1]
			}
			if mode == "far_chest" {
				v.Waypoints[1].Pos[0] = 41
				v.Perception.Structures[0].Pos[0] = 41
			}
			if mode == "far_marker" {
				v.Waypoints[1].Pos[0] = 41
				v.Waypoints[1].Radius = 40
			}
			order, err := r.Next(v, nil)
			if mode != "valid" {
				var need *StorageNeed
				if order != nil || !errors.As(err, &need) || r.TravelFromIdleMarker() {
					t.Fatal(order, err, r.journey)
				}
				return
			}
			if err != nil || order == nil || order.Task != protocol.TaskDeposit || !r.TravelFromIdleMarker() {
				t.Fatal(order, err)
			}
			v.State.Pos = v.Waypoints[1].Pos
			if err = r.Check(v); err != nil {
				t.Fatal("explicit storage floor refused", err)
			}
			v.Waypoints[1].At = v.Now.Add(time.Second)
			if r.Check(v) == nil {
				t.Fatal("edited destination retained journey permission")
			}
		})
	}
}
func TestCargoJourneyVerifiesDepositAndReturnsToAdmissionFloor(t *testing.T) {
	r, v := cargoJourney(t)
	order, err := r.Next(v, nil)
	if err != nil || order == nil || order.Task != protocol.TaskDeposit {
		t.Fatal(order, err)
	}
	v.State.Pos = v.Waypoints[1].Pos
	v.Inventory.Seq++
	v.Inventory.Items[0].Stack = 2
	v.Containers = []*protocol.Container{{Entity: "coal", Task: "deposit", Direction: protocol.TaskDeposit, Moved: map[string]int{"Coal": 5}}}
	order, err = r.Next(v, &Result{Status: protocol.StatusSucceeded, TaskID: "deposit"})
	if err != nil || order == nil || order.Task != protocol.TaskMoveTo {
		t.Fatal(order, err)
	}
	a := order.Args.(*protocol.MoveToArgs)
	if *a.To.Pos != v.Waypoints[0].Pos || !a.Dry || !a.AvoidHostiles {
		t.Fatal(a)
	}
	result := &Result{Status: protocol.StatusSucceeded, TaskID: "return"}
	if _, err = r.Next(v, result); err != ErrWait {
		t.Fatal("return text proved home arrival", err)
	}
	v.State.Pos = v.Waypoints[0].Pos
	if order, err = r.Next(v, result); err != nil || order != nil {
		t.Fatal(order, err)
	}
	if v.Inventory.Count("Coal") != 2 {
		t.Fatal("reserve lost")
	}
}
func TestCargoTravelRetainsThreatInjuryAndLocalBounds(t *testing.T) {
	for _, mode := range []string{"threat", "injury", "bounds", "origin_edit"} {
		t.Run(mode, func(t *testing.T) {
			r, v := cargoJourney(t)
			if _, err := r.Next(v, nil); err != nil {
				t.Fatal(err)
			}
			v.State.Pos = v.Waypoints[1].Pos
			switch mode {
			case "threat":
				v.Perception.Characters = []protocol.PerceivedCharacter{{HPFrac: 1, Hostile: true, Pos: v.State.Pos}}
			case "injury":
				v.State.HP--
			case "bounds":
				v.State.Pos[0] = 40.1
			case "origin_edit":
				v.Waypoints[0].At = v.Now.Add(time.Second)
			}
			if r.Check(v) == nil {
				t.Fatal("travel bypassed", mode)
			}
		})
	}
}

func TestCargoJourneyReturnsBeforeReportingUnmatchedCargo(t *testing.T) {
	r, v := cargoJourney(t)
	if _, err := r.Next(v, nil); err != nil {
		t.Fatal(err)
	}
	v.State.Pos = v.Waypoints[1].Pos
	order, err := r.Next(v, &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonRefused, TaskID: "deposit"})
	if err != nil || order == nil || order.Task != protocol.TaskMoveTo {
		t.Fatal(order, err)
	}
	v.State.Pos = v.Waypoints[0].Pos
	order, err = r.Next(v, &Result{Status: protocol.StatusSucceeded, TaskID: "return"})
	var need *StorageNeed
	if order != nil || !errors.As(err, &need) || need.Items["Coal"] != 5 || v.Inventory.Count("Coal") != 7 {
		t.Fatal(order, err)
	}
}

func TestCargoNoticesAHitAfterHealingPastItsStart(t *testing.T) {
	r, v := cargoFixture(t)
	v.State.HP, v.State.HPMax = 80, 100
	w := v.Waypoints[0]
	r.guard.marker, r.guard.began, r.guard.health = &w, v.Now, sampleHealth(v)
	if err := healThenHit(t, r.Check, v); err == nil {
		t.Fatal("a hit after healing past the start health was ignored")
	}
}
