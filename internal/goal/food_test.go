package goal

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/protocol"
)

func foodFixture(t *testing.T) (*Template, *runner, behaviour.View, string) {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "goals.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	g, err := NewRecord(Spec{Kind: "food_resupply", World: "one", Radius: 150}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Add(g); err != nil {
		t.Fatal(err)
	}
	host := &Template{Store: s}
	if err = host.Queue(g.ID); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(params{g.ID})
	run, err := host.Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	home := protocol.Vec3{}
	injury := uint64(0)
	v := behaviour.View{Now: now, StateAt: now, PerceptionAt: now, Home: &home, InjurySeq: &injury, Owners: []string{"Nimah"}, State: &protocol.State{Envelope: protocol.Envelope{Seq: 10, T: 100}, World: "one", Phase: protocol.PhaseInWorld, HP: 50, HPMax: 50, Food: &protocol.FoodStatus{Active: []protocol.ActiveFood{{Prefab: "Raspberry", RemainingS: 1000}}}}, Inventory: &protocol.Inventory{Envelope: protocol.Envelope{Seq: 9, T: 100}, Size: [2]int{8, 4}}, Perception: &protocol.Perception{Envelope: protocol.Envelope{Seq: 8, T: 100}, Structures: []protocol.PerceivedStructure{{ID: "a", Kind: protocol.StructureContainer, Pos: protocol.Vec3{1, 0, 0}}, {ID: "b", Kind: protocol.StructureContainer, Pos: protocol.Vec3{2, 0, 0}}, {ID: "station", Kind: protocol.StructureStation, Pos: protocol.Vec3{3, 0, 0}, Cooking: &protocol.CookingStatus{Supported: true, Heated: true, Slots: []protocol.CookingSlotStatus{{State: "empty"}}}}}}}
	return host, run.(*runner), v, g.ID
}
func foodFresh(v *behaviour.View) { fresh(v, 3); v.State.Seq += 10; v.Inventory.Seq += 10 }
func foodTransfer(t *testing.T, r *runner, v *behaviour.View, task, entity, item string, moved int, contents []protocol.InventoryItem) *behaviour.Result {
	t.Helper()
	if err := r.Dispatched(task); err != nil {
		t.Fatal(err)
	}
	foodFresh(v)
	v.Containers = append(v.Containers, &protocol.Container{Entity: entity, Task: task, Direction: protocol.TaskWithdraw, Items: contents, Moved: map[string]int{item: moved}})
	if moved > 0 {
		v.Inventory.Items = append(v.Inventory.Items, protocol.InventoryItem{Prefab: item, Stack: moved})
	}
	status, reason := "succeeded", ""
	if moved == 0 {
		status, reason = "failed", protocol.ReasonInvalidTarget
	}
	return &behaviour.Result{Status: status, Reason: reason, TaskID: task}
}
func deliveryRows(id, item, station string) []*protocol.Trace {
	row := func(msg string, data map[string]any) *protocol.Trace {
		data["id"] = id
		data["station"] = station
		return &protocol.Trace{Task: id, Source: "cooking", Msg: msg, Data: data}
	}
	return []*protocol.Trace{
		row("normal cooking removal request observed", map[string]any{"requestedCount": 1, "evidence": "remote_correlation"}),
		row("correlated remote station output observed", map[string]any{"entity": "drop-" + id, "output": item, "count": 1}),
		row("verified cooking drop pickup", map[string]any{"entity": "drop-" + id, "output": item, "count": 1}),
		row("verified cooked inventory receipt", map[string]any{"output": item, "count": 1, "evidence": "remote_correlation"}),
	}
}
func TestFoodBootstrapsFromObservedYellowMushroomsBeforeCooking(t *testing.T) {
	_, r, v, _ := foodFixture(t)
	v.State.HP, v.State.HPMax = 25, 25
	v.State.Food = &protocol.FoodStatus{}
	v.Inventory.Items = []protocol.InventoryItem{{Prefab: "RawMeat", Stack: 1}}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskWithdraw {
		t.Fatal(o, err)
	}
	// An ordinary failed probe reports the chest's real contents.
	last := foodTransfer(t, r, &v, "probe", "a", "CookedDeerMeat", 0, []protocol.InventoryItem{{Prefab: "MushroomYellow", Stack: 7}})
	o, err = r.Next(v, last)
	if err != nil || o == nil || o.Task != protocol.TaskWithdraw {
		t.Fatal("ignored available starter food", o, err)
	}
	args := o.Args.(*protocol.TransferArgs)
	if args.Container != "a" || args.Items[0].Prefab != "MushroomYellow" || args.Items[0].Count != 3 {
		t.Fatal(args)
	}
	last = foodTransfer(t, r, &v, "starter", "a", "MushroomYellow", 3, nil)
	v.State.Food.Carried = []protocol.CarriedFood{{Prefab: "MushroomYellow", CanEat: true, Health: 10, Regen: 1}}
	o, err = r.Next(v, last)
	if err != nil || o == nil || o.Task != protocol.TaskUseItem || o.Args.(*protocol.UseItemArgs).Item != "MushroomYellow" {
		t.Fatal("did not eat the verified starter before attempting cooking", o, err)
	}
}
func TestFoodWrongHintsSearchDistinctBoxesThenCookAReserve(t *testing.T) {
	host, r, v, id := foodFixture(t)
	v.Containers = []*protocol.Container{{Entity: "a", Items: []protocol.InventoryItem{{Prefab: "CookedDeerMeat", Stack: 10}}}}
	o, err := r.Next(v, nil)
	if err != nil || o.Task != "withdraw" {
		t.Fatal(o, err)
	}
	last := foodTransfer(t, r, &v, "ready-a", "a", "CookedDeerMeat", 0, []protocol.InventoryItem{{Prefab: "NeckTail", Stack: 3}})
	o, err = r.Next(v, last)
	if err != nil || o.Args.(*protocol.TransferArgs).Container != "b" {
		t.Fatal(o, err)
	}
	last = foodTransfer(t, r, &v, "ready-b", "b", "CookedDeerMeat", 0, nil)
	o, err = r.Next(v, last)
	if err != nil {
		t.Fatal(err)
	}
	a := o.Args.(*protocol.TransferArgs)
	if a.Container != "a" || a.Items[0].Prefab != "NeckTail" || a.Items[0].Count != 3 {
		t.Fatal(a)
	}
	last = foodTransfer(t, r, &v, "raw-a", "a", "NeckTail", 3, nil)
	o, err = r.Next(v, last)
	if err != nil || o.Task != "interact" || o.Args.(*protocol.InteractArgs).Cook != "NeckTail" {
		t.Fatal(o, err)
	}
	for _, task := range []string{"cook-1", "cook-2"} {
		if err = r.Dispatched(task); err != nil {
			t.Fatal(err)
		}
		foodFresh(&v)
		v.Inventory.Items[0].Stack--
		v.Inventory.Items = append(v.Inventory.Items, protocol.InventoryItem{Prefab: "NeckTailGrilled", Stack: 1})
		o, err = r.Next(v, &behaviour.Result{Status: "succeeded", TaskID: task, Cooking: deliveryRows(task, "NeckTailGrilled", "station")})
		if err != nil {
			t.Fatal(err)
		}
	}
	g, _ := host.Store.Get(id)
	if o != nil || g.Status != "succeeded" || g.TaskCount != 5 || g.Food.Batches != 2 || len(g.Food.Tried) != 3 || len(g.Food.Receipts) != 3 {
		t.Fatalf("%+v %+v", o, g)
	}
}
func TestFoodCannotCompleteFromSuccessTextOrUnrelatedReceipt(t *testing.T) {
	for _, wrong := range []string{"task", "container", "inventory", "cooking"} {
		t.Run(wrong, func(t *testing.T) {
			host, r, v, id := foodFixture(t)
			o, err := r.Next(v, nil)
			if err != nil || o == nil {
				t.Fatal(err)
			}
			last := foodTransfer(t, r, &v, "our-task", "a", "CookedDeerMeat", 3, nil)
			switch wrong {
			case "task":
				last.TaskID = "other"
			case "container":
				v.Containers[0].Task = "other"
			case "inventory":
				v.Inventory.Items = nil
			case "cooking":
				g, _ := host.Store.Get(id)
				_ = host.Store.update(id, func(g *Record) error {
					g.Pending.Template = "cook"
					g.Pending.Entity = "station"
					g.Food.Item = "NeckTail"
					return nil
				})
				_ = g
			}
			o, err = r.Next(v, last)
			g, _ := host.Store.Get(id)
			if o != nil || g.Status == "succeeded" {
				t.Fatalf("accepted missing evidence %+v %v", g, err)
			}
			if wrong == "inventory" {
				var delay *behaviour.Delay
				if !errors.As(err, &delay) || g.Pending == nil {
					t.Fatal(g, err)
				}
			} else if g.Status != "needs_plan" || g.Pending == nil {
				t.Fatal(g, err)
			}
		})
	}
}
func TestFoodPartialWithdrawalChecksNextChestAndNeverRepeatsUnchangedProbe(t *testing.T) {
	host, r, v, id := foodFixture(t)
	o, err := r.Next(v, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := o.Args.(*protocol.TransferArgs)
	last := foodTransfer(t, r, &v, "partial", "a", a.Items[0].Prefab, 1, nil)
	last.Status = "failed"
	last.Reason = protocol.ReasonMissingMaterials
	o, err = r.Next(v, last)
	if err != nil || o == nil || o.Args.(*protocol.TransferArgs).Container != "b" {
		t.Fatal(o, err)
	}
	last = foodTransfer(t, r, &v, "empty", "b", "CookedDeerMeat", 0, nil)
	o, err = r.Next(v, last)
	g, _ := host.Store.Get(id)
	if o != nil || err != nil || g.Status != "needs_plan" || len(g.Food.Tried) != 2 || len(g.Food.Receipts) != 1 {
		t.Fatal(o, err, g)
	}
	if err = host.Queue(id); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(params{id})
	run, err := host.Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	o, err = run.Next(v, nil)
	if o != nil || err != nil {
		t.Fatal(o, err)
	}
	g, _ = host.Store.Get(id)
	if len(g.Food.Tried) != 2 {
		t.Fatal(g)
	}
}
func TestFoodCheckpointFailureAndUncertainRestartCannotDispatchAgain(t *testing.T) {
	host, r, v, id := foodFixture(t)
	host.Store.write = func([]byte) error { return errors.New("disk full") }
	if o, err := r.Next(v, nil); o != nil || err == nil {
		t.Fatal(o, err)
	}
	g, _ := host.Store.Get(id)
	if g.Pending != nil || g.TaskCount != 0 {
		t.Fatal(g)
	}
	host.Store.write = host.Store.saveFile
	if _, err := r.Next(v, nil); err != nil {
		t.Fatal(err)
	}
	if err := r.Dispatched("uncertain"); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(host.Store.path)
	if err != nil {
		t.Fatal(err)
	}
	g, _ = reopened.Get(id)
	if g.Status != "needs_plan" || g.Pending.TaskID != "uncertain" || g.TaskCount != 1 {
		t.Fatal(g)
	}
	if g.Pending.Run == "" || g.Pending.Run == reopened.runID {
		t.Fatal("dispatch run identity was not retained across restart")
	}
	if err = (&Template{Store: reopened}).Queue(id); err == nil {
		t.Fatal("replayed uncertain transfer")
	}
}
func TestFoodBudgetAndHomeScopeStayPinned(t *testing.T) {
	for _, which := range []string{"budget", "home", "injury"} {
		t.Run(which, func(t *testing.T) {
			host, r, v, id := foodFixture(t)
			_ = host.Store.update(id, func(g *Record) error { g.Home = v.Home; n := uint64(0); g.Food.Injury = &n; return nil })
			switch which {
			case "budget":
				_ = host.Store.update(id, func(g *Record) error { g.TaskCount = g.MaxTasks; return nil })
			case "home":
				v.Home = &protocol.Vec3{1, 0, 0}
			case "injury":
				n := uint64(1)
				v.InjurySeq = &n
			}
			o, err := r.Next(v, nil)
			g, _ := host.Store.Get(id)
			if err != nil || o != nil || g.Status != "needs_plan" {
				t.Fatal(o, err, g)
			}
		})
	}
}
func TestFoodReadinessCanHandOffOnlyToCurrentCommander(t *testing.T) {
	host, r, v, id := foodFixture(t)
	_ = host.Store.update(id, func(g *Record) error {
		g.Food.Escort = &protocol.EscortArgs{Player: "Nimah", Resupply: true}
		return nil
	})
	v.Inventory.Items = []protocol.InventoryItem{{Prefab: "CookedMeat", Stack: 2}}
	if o, err := r.Next(v, nil); o != nil || err != nil {
		t.Fatal(o, err)
	}
	o, err := r.Continuation(v)
	if err != nil || o == nil || o.Args.(*protocol.EscortArgs).Player != "Nimah" {
		t.Fatal(o, err)
	}
	if err = r.Continued("escort-task"); err != nil {
		t.Fatal(err)
	}
	g, _ := host.Store.Get(id)
	if g.Food.Continuation == nil || g.Food.Continuation.TaskID != "escort-task" || g.Food.Continuation.Run == "" || g.TaskCount != 1 {
		t.Fatal(g)
	}
	v.Owners = []string{"Sothr"}
	if o, err := r.Continuation(v); o != nil || err != nil {
		t.Fatal("resumed revoked commander")
	}
	_ = host.End(json.RawMessage(`{"id":"`+id+`"}`), "new order")
	g, _ = host.Store.Get(id)
	if g.Status != "succeeded" {
		t.Fatal(g)
	}
}
func TestCookingDeliveryRequiresEveryIdentityAndRequestedCount(t *testing.T) {
	rows := deliveryRows("task", "CookedMeat", "station")
	if n, mode := cookingDelivery(rows, "task", "CookedMeat", "station"); n != 1 || mode != "remote_correlation" {
		t.Fatal(n, mode)
	}
	for i := range rows {
		copy := append([]*protocol.Trace(nil), rows[:i]...)
		copy = append(copy, rows[i+1:]...)
		if n, _ := cookingDelivery(copy, "task", "CookedMeat", "station"); n != 0 {
			t.Fatal("missing receipt accepted", i)
		}
	}
	for _, wrong := range []string{"count", "entity", "station", "id"} {
		t.Run(wrong, func(t *testing.T) {
			rows := deliveryRows("task", "CookedMeat", "station")
			switch wrong {
			case "count":
				rows[0].Data["requestedCount"] = 2
			case "entity":
				rows[2].Data["entity"] = "other"
			case "station":
				rows[1].Data["station"] = "other"
			case "id":
				rows[3].Data["id"] = "other"
			}
			if n, _ := cookingDelivery(rows, "task", "CookedMeat", "station"); n != 0 {
				t.Fatal("accepted mismatch")
			}
		})
	}
}
func TestFoodCancellationRetainsUncertaintyAndNeverOffersContinuation(t *testing.T) {
	host, r, v, id := foodFixture(t)
	_, _ = r.Next(v, nil)
	_ = r.Dispatched("task")
	raw, _ := json.Marshal(params{id})
	if err := host.End(raw, "new order"); err != nil {
		t.Fatal(err)
	}
	g, _ := host.Store.Get(id)
	o, err := r.Continuation(v)
	if g.Status != "cancelled" || g.Pending == nil || o != nil || err != nil || !strings.Contains(g.Reason, "new order") {
		t.Fatal(g)
	}
}
