package behaviour

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

func markerView(kind string) View {
	now := time.Now()
	return View{Now: now, StateAt: now, PerceptionAt: now, State: &protocol.State{Envelope: protocol.Envelope{Seq: 10, T: 20}, World: "one", Phase: protocol.PhaseInWorld, HP: 70, HPMax: 70, WeightMax: 300, Food: &protocol.FoodStatus{Active: []protocol.ActiveFood{{Prefab: "Mushroom", RemainingS: 900}}}}, Inventory: &protocol.Inventory{Envelope: protocol.Envelope{Seq: 10}, Size: [2]int{8, 4}}, Perception: &protocol.Perception{Envelope: protocol.Envelope{Seq: 10, T: 20}}, Waypoints: []home.Waypoint{{Name: "work", Kind: kind, Radius: 10, At: now}}}
}
func foodRunner(t *testing.T, kind string) *markerFoodRun {
	t.Helper()
	raw, _ := json.Marshal(MarkerFoodParams{Marker: Marker{World: "one", Name: "work", Kind: kind}})
	r, err := (MarkerFood{}).Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	return r.(*markerFoodRun)
}
func TestMarkerFoodEmptySitesBackOffWithUsefulReason(t *testing.T) {
	for _, kind := range []string{"berries", "cooking", "storage"} {
		t.Run(kind, func(t *testing.T) {
			r := foodRunner(t, kind)
			order, err := r.Next(markerView(kind), nil)
			if order != nil || err == nil {
				t.Fatal("empty site fabricated work", order, err)
			}
		})
	}
}

func TestMarkerFoodGuardAcceptsIndependentFreshStreamTiming(t *testing.T) {
	r := foodRunner(t, "cooking")
	v := markerView("cooking")
	v.Perception.T = v.State.T + .06 // observed live state/perception skew
	if err := r.Check(v); err != nil {
		t.Fatal("fresh independent scan cancelled cooking", err)
	}
	v.Perception.T = v.State.T + 4
	if err := r.Check(v); err == nil {
		t.Fatal("out-of-scope scan accepted")
	}
}
func TestMarkerFoodCollectsOnlyFreshLocalSafeEdibleYields(t *testing.T) {
	r := foodRunner(t, "berries")
	v := markerView("berries")
	v.Perception.Resources = []protocol.PerceivedResource{
		{ID: "far", Kind: protocol.ResourcePickable, Pos: protocol.Vec3{15, 0, 0}, Yield: &protocol.ResourceYield{Prefab: "Mushroom", Count: 1, StackMax: 50, Weight: .1}},
		{ID: "wrong-floor", Kind: protocol.ResourcePickable, Pos: protocol.Vec3{0, 3, 0}, Yield: &protocol.ResourceYield{Prefab: "Mushroom", Count: 1, StackMax: 50, Weight: .1}},
		{ID: "good", Kind: protocol.ResourcePickable, Pos: protocol.Vec3{2, 0, 0}, Yield: &protocol.ResourceYield{Prefab: "Mushroom", Count: 1, StackMax: 50, Weight: .1}},
	}
	order, err := r.Next(v, nil)
	if err != nil || order == nil || order.Args.(*protocol.InteractArgs).Entity != "good" {
		t.Fatal(order, err)
	}
	v.Perception.Resources = nil
	v.Inventory.Seq++
	v.Inventory.Items = []protocol.InventoryItem{{Prefab: "Mushroom", Stack: 1}}
	order, err = r.Next(v, &Result{Status: protocol.StatusSucceeded})
	if err != nil || order != nil {
		t.Fatal("verified collection did not finish", order, err)
	}
}
func TestMarkerFoodStockLimitAndInjuryAndThreatGuards(t *testing.T) {
	for _, mode := range []string{"stock", "injury", "threat", "marker", "stale"} {
		t.Run(mode, func(t *testing.T) {
			r := foodRunner(t, "berries")
			v := markerView("berries")
			if mode == "injury" || mode == "marker" {
				w := v.Waypoints[0]
				r.marker = &w
				r.began = v.Now
				r.health = sampleHealth(v)
			}
			switch mode {
			case "stock":
				v.Inventory.Items = []protocol.InventoryItem{{Prefab: "Mushroom", Stack: 20}}
				v.Perception.Resources = []protocol.PerceivedResource{{ID: "full", Kind: protocol.ResourcePickable, Yield: &protocol.ResourceYield{Prefab: "Mushroom", Count: 1, StackMax: 50, Weight: .1}}}
			case "injury":
				v.State.HP = 69
			case "threat":
				v.Perception.Characters = []protocol.PerceivedCharacter{{HPFrac: 1, Hostile: true}}
			case "marker":
				v.Waypoints[0].At = v.Now.Add(time.Second)
			case "stale":
				v.PerceptionAt = v.Now.Add(-4 * time.Second)
			}
			if order, err := r.Next(v, nil); order != nil || err == nil {
				t.Fatal("unsafe work allowed", order, err)
			}
		})
	}
}
func TestMarkerFoodStoreKeepsPersonalMealsAndRequiresTransferReceipt(t *testing.T) {
	for _, prefab := range []string{"Mushroom", "MushroomYellow"} {
		t.Run(prefab, func(t *testing.T) {
			r := foodRunner(t, "storage")
			v := markerView("storage")
			v.Inventory.Items = []protocol.InventoryItem{{Prefab: prefab, Stack: 5}}
			v.Perception.Structures = []protocol.PerceivedStructure{{ID: "box", Kind: protocol.StructureContainer}}
			order, err := r.Next(v, nil)
			if err != nil || order == nil {
				t.Fatal(order, err)
			}
			args := order.Args.(*protocol.TransferArgs)
			if args.Container != "box" || int(args.Items[0].Count) != 3 {
				t.Fatal(args)
			}
			if _, err = r.Next(v, &Result{TaskID: "deposit", Status: protocol.StatusSucceeded}); err != ErrWait {
				t.Fatal("accepted success without receipt", err)
			}
			v.Inventory.Seq++
			v.Inventory.Items[0].Stack = 2
			v.Containers = []*protocol.Container{{Entity: "box", Task: "deposit", Direction: protocol.TaskDeposit, Moved: map[string]int{prefab: 3}}}
			if order, err = r.Next(v, &Result{TaskID: "deposit", Status: protocol.StatusSucceeded}); order != nil || err != nil {
				t.Fatal(order, err)
			}
		})
	}
}
func cookingRows() []*protocol.Trace {
	rows := []*protocol.Trace{}
	for _, row := range []struct {
		msg  string
		data map[string]any
	}{
		{"normal cooking removal request observed", map[string]any{"requestedCount": 1, "evidence": "direct_spawn"}},
		{"normal station output spawned", map[string]any{"entity": "drop", "count": 1, "output": "CookedMeat"}},
		{"verified cooking drop pickup", map[string]any{"entity": "drop", "count": 1, "output": "CookedMeat"}},
		{"verified cooked inventory receipt", map[string]any{"count": 1, "output": "CookedMeat", "evidence": "direct_spawn"}},
	} {
		row.data["id"] = "cook"
		row.data["station"] = "station"
		rows = append(rows, &protocol.Trace{Task: "cook", Source: "cooking", Msg: row.msg, Data: row.data})
	}
	return rows
}
func TestMarkerFoodCookingRequiresHeatedCookerAndMatchedDelivery(t *testing.T) {
	for _, mode := range []string{"cold", "missing-receipt", "wrong-station", "success"} {
		t.Run(mode, func(t *testing.T) {
			r := foodRunner(t, "cooking")
			v := markerView("cooking")
			v.Inventory.Items = []protocol.InventoryItem{{Prefab: "RawMeat", Stack: 1}}
			v.Perception.Structures = []protocol.PerceivedStructure{{ID: "station", Cooking: &protocol.CookingStatus{Supported: true, Heated: mode != "cold", Slots: []protocol.CookingSlotStatus{{State: "empty"}}}}}
			order, err := r.Next(v, nil)
			if mode == "cold" {
				if err == nil || order != nil {
					t.Fatal("cold cooker accepted")
				}
				return
			}
			if err != nil || order == nil || order.Args.(*protocol.InteractArgs).Cook != "RawMeat" {
				t.Fatal(order, err)
			}
			rows := cookingRows()
			if mode == "missing-receipt" {
				rows = rows[:3]
			}
			if mode == "wrong-station" {
				rows[3].Data["station"] = "other"
			}
			v.Inventory.Seq++
			v.State.Seq++
			v.Inventory.Items = []protocol.InventoryItem{{Prefab: "CookedMeat", Stack: 1}}
			order, err = r.Next(v, &Result{TaskID: "cook", Status: protocol.StatusSucceeded, Cooking: rows})
			if mode == "success" {
				if err != nil || order != nil {
					t.Fatal(order, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "receipts") {
				t.Fatal("unverified cook accepted", err)
			}
		})
	}
}

func TestMarkerFoodLeavesTheCookingPickupPositionBeforeCompleting(t *testing.T) {
	for _, status := range []string{protocol.StatusSucceeded, protocol.StatusFailed} {
		t.Run(status, func(t *testing.T) {
			r := foodRunner(t, "cooking")
			r.batches = 1 // a batch with verified delivery has just finished
			v := markerView("cooking")
			v.State.Pos = protocol.Vec3{2, 0, 0}
			order, err := r.Next(v, nil)
			if err != nil || order == nil || order.Task != protocol.TaskMoveTo {
				t.Fatal("left Tas at the cooker pickup position", order, err)
			}
			args := order.Args.(*protocol.MoveToArgs)
			if args.To.Pos == nil || *args.To.Pos != v.Waypoints[0].Pos || !args.Dry || !args.AvoidHostiles || args.StopDistance == nil || *args.StopDistance != 1 {
				t.Fatal(args)
			}
			v.State.Pos = v.Waypoints[0].Pos
			order, err = r.Next(v, &Result{Status: status})
			if order != nil || (err == nil) != (status == protocol.StatusSucceeded) {
				t.Fatal("return outcome was not respected", order, err)
			}
		})
	}
}

func TestPreparingFoodReservesKeeperForActiveKitchenOnly(t *testing.T) {
	e := &Engine{instances: map[string]*instance{}}
	if e.PreparingFood() {
		t.Fatal("idle keeper reserved")
	}
	for _, kind := range []string{"cooking", "berries", "storage"} {
		d := MarkerFoodDefinition(Marker{World: "one", Name: "work", Kind: kind})
		e.instances["work"] = &instance{definition: d, pending: true}
		if e.PreparingFood() {
			t.Fatal("queued work reserved keeper")
		}
		e.active = "work"
		if e.PreparingFood() != (kind == "cooking") {
			t.Fatal("wrong active food reservation", kind)
		}
		e.active = ""
	}
}

func TestMarkerFoodNewMealDoesNotCountAsAnInjury(t *testing.T) {
	r := foodRunner(t, "berries")
	v := markerView("berries")
	seq := uint64(10)
	v.InjurySeq = &seq
	w := v.Waypoints[0]
	r.marker = &w
	r.began = v.Now
	r.health = sampleHealth(v)
	v.State.HPMax = 100
	v.State.Food.Active = append(v.State.Food.Active, protocol.ActiveFood{Prefab: "CookedMeat", RemainingS: 1200})
	if err := r.Check(v); err != nil {
		t.Fatal("new food maximum cancelled safe work", err)
	}
	seq++
	if err := r.Check(v); err == nil {
		t.Fatal("real injury ignored after meal activation")
	}
}
