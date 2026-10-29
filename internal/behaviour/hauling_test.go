package behaviour

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func pack(seq uint64, items ...protocol.InventoryItem) *protocol.Inventory {
	return &protocol.Inventory{Envelope: protocol.Envelope{Seq: seq}, Size: [2]int{4, 2}, Items: items}
}
func material(name string, n int) protocol.InventoryItem {
	return protocol.InventoryItem{Prefab: name, Stack: n, StackMax: 50, Weight: 2, Type: "Material"}
}
func haulView() View {
	return View{Now: time.Unix(100, 0), State: &protocol.State{Envelope: protocol.Envelope{T: 10}, World: "test", Phase: protocol.PhaseInWorld, HP: 25, WeightMax: 300}, Inventory: pack(10, material("Stone", 10)), Perception: &protocol.Perception{Envelope: protocol.Envelope{Seq: 12, T: 10}}}
}
func success(id string) *Result { return &Result{Status: protocol.StatusSucceeded, TaskID: id} }
func stowParams() StowParams {
	return StowParams{World: "test", Items: []string{"Stone"}, To: []string{"a", "b"}, Keep: map[string]int{"Stone": 2}}
}
func nextTask(t *testing.T, r Runner, v View, result *Result, name string) *session.Order {
	t.Helper()
	o, e := r.Next(v, result)
	if e != nil || o == nil || o.Task != name {
		t.Fatalf("want %s; got %+v, %v", name, o, e)
	}
	return o
}
func receipt(v *View, id, chest string, n int) {
	v.Inventory = pack(v.Inventory.Seq+1, material("Stone", v.Inventory.Count("Stone")-n))
	v.Containers = []*protocol.Container{{Entity: chest, Task: id, Direction: protocol.TaskDeposit, Moved: map[string]int{"Stone": n}}}
}
func TestStowReplansAfterPartialTransferAndPreservesKeep(t *testing.T) {
	v := haulView()
	r := &stowRun{p: stowParams()}
	o := nextTask(t, r, v, nil, protocol.TaskDeposit)
	a := o.Args.(*protocol.TransferArgs)
	if a.Container != "a" || a.Items[0].Count != 8 {
		t.Fatal(a)
	}
	receipt(&v, "first", "a", 3)
	failed := &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonInventoryFull, TaskID: "first"}
	if !r.Recover(*failed) {
		t.Fatal("partial full result must be recoverable")
	}
	o = nextTask(t, r, v, failed, protocol.TaskDeposit)
	a = o.Args.(*protocol.TransferArgs)
	if a.Container != "b" || a.Items[0].Count != 5 {
		t.Fatal(a)
	}
	receipt(&v, "second", "b", 5)
	if o, e := r.Next(v, success("second")); o != nil || e != nil {
		t.Fatal(o, e)
	}
	if v.Inventory.Count("Stone") != 2 {
		t.Fatal(v.Inventory)
	}
}
func TestStowDoesNotTrustUnrelatedReceiptsOrStalePack(t *testing.T) {
	v := haulView()
	r := &stowRun{p: stowParams()}
	nextTask(t, r, v, nil, protocol.TaskDeposit)
	v.Containers = []*protocol.Container{{Entity: "a", Task: "other", Direction: protocol.TaskDeposit, Moved: map[string]int{"Stone": 8}}}
	if _, e := r.Next(v, success("ours")); !errors.Is(e, ErrWait) {
		t.Fatal(e)
	}
	v.Containers[0].Task = "ours"
	if _, e := r.Next(v, success("ours")); !errors.Is(e, ErrWait) {
		t.Fatal(e)
	}
	v.Inventory = pack(11, material("Stone", 2))
	if o, e := r.Next(v, success("ours")); o != nil || e != nil {
		t.Fatal(o, e)
	}
}
func TestStowProtectsGearAndKeepsExactCounts(t *testing.T) {
	v := haulView()
	axe := material("AxeStone", 1)
	axe.Type = "OneHandedWeapon"
	axe.Equipped = false
	worn := material("Armour", 1)
	worn.Equipped = true
	v.Inventory = pack(11, material("Stone", 10), axe, worn, material("Wood", 7))
	p := stowParams()
	p.Items = []string{"*"}
	p.Keep = map[string]int{"Stone": 8, "Wood": 20}
	r := &stowRun{p: p}
	o := nextTask(t, r, v, nil, protocol.TaskDeposit)
	a := o.Args.(*protocol.TransferArgs)
	if len(a.Items) != 1 || a.Items[0].Prefab != "Stone" || a.Items[0].Count != 2 {
		t.Fatal(a)
	}
}
func TestStowBusyChestFallsBackAndReportsShortfall(t *testing.T) {
	v := haulView()
	r := &stowRun{p: stowParams()}
	nextTask(t, r, v, nil, protocol.TaskDeposit)
	failed := &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonRefused, TaskID: "first"}
	o := nextTask(t, r, v, failed, protocol.TaskDeposit)
	if o.Args.(*protocol.TransferArgs).Container != "b" {
		t.Fatal(o)
	}
	receipt(&v, "second", "b", 0)
	failed = &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonInventoryFull, TaskID: "second"}
	_, err := r.Next(v, failed)
	var need *StorageNeed
	if !errors.As(err, &need) || need.Items["Stone"] != 8 || !strings.Contains(need.Error(), "a: refused; b: inventory_full") {
		t.Fatal(err)
	}
}
func TestStowUsesRememberedLocationOnceAndDoesNotRetryAnUnreachableChest(t *testing.T) {
	v := haulView()
	v.Containers = []*protocol.Container{{Entity: "a", Pos: protocol.Vec3{100, 0, 0}}}
	r := &stowRun{p: stowParams()}
	nextTask(t, r, v, nil, protocol.TaskMoveTo)
	// The state can lag the task completion; don't approach the same chest twice.
	o := nextTask(t, r, v, success("walk"), protocol.TaskDeposit)
	if o.Args.(*protocol.TransferArgs).Container != "a" {
		t.Fatal(o)
	}
	r = &stowRun{p: stowParams()}
	nextTask(t, r, v, nil, protocol.TaskMoveTo)
	failed := &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable}
	o = nextTask(t, r, v, failed, protocol.TaskDeposit)
	if o.Args.(*protocol.TransferArgs).Container != "b" {
		t.Fatal(o)
	}
	if r.Recover(Result{Status: protocol.StatusCancelled, Reason: protocol.ReasonStopped}) || r.Recover(Result{Status: protocol.StatusFailed, Reason: protocol.ReasonNotAllowed}) {
		t.Fatal("must not swallow stop/permission failures")
	}
}
func collectParams() CollectParams {
	return CollectParams{Site: Place{World: "test", Radius: 15}, Items: []string{"Flint", "Stone"}, To: []string{"a"}, MaxTargets: 4, MaxWeightFraction: .8, Return: true}
}
func target(id, kind, prefab string, x float64) protocol.PerceivedResource {
	return protocol.PerceivedResource{ID: id, Kind: kind, Prefab: "scenery", Pos: protocol.Vec3{x, 0, 0}, Yield: &protocol.ResourceYield{Prefab: prefab, Count: 1, StackMax: 50, Weight: 2}}
}
func TestCollectPickableThenSpawnedDropThenStowThenReturn(t *testing.T) {
	p := collectParams()
	raw, _ := json.Marshal(p)
	runner, err := (Collect{}).Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	r := runner.(*collectRun)
	v := haulView()
	v.Inventory = pack(10)
	v.Perception.Resources = []protocol.PerceivedResource{target("flint", protocol.ResourcePickable, "Flint", 3)}
	o := nextTask(t, r, v, nil, protocol.TaskInteract)
	if o.Args.(*protocol.InteractArgs).Entity != "flint" {
		t.Fatal(o)
	}
	// Interact's boolean can report refusal even when the game's pick RPC ran.
	failed := &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonRefused, TaskID: "pick"}
	if !r.Recover(*failed) {
		t.Fatal("must observe the pick result")
	}
	if _, e := r.Next(v, failed); !errors.Is(e, ErrWait) {
		t.Fatal(e)
	}
	v.Perception = &protocol.Perception{Envelope: protocol.Envelope{Seq: 15, T: 10}, Resources: []protocol.PerceivedResource{target("product", protocol.ResourceDrop, "Flint", 3)}}
	nextTask(t, r, v, failed, protocol.TaskPickup)
	if _, e := r.Next(v, success("take")); !errors.Is(e, ErrWait) {
		t.Fatal(e)
	}
	v.Inventory = pack(16, material("Flint", 1))
	v.Perception = &protocol.Perception{Envelope: protocol.Envelope{Seq: 17, T: 10}}
	o = nextTask(t, r, v, success("take"), protocol.TaskDeposit)
	a := o.Args.(*protocol.TransferArgs)
	if len(a.Items) != 1 || a.Items[0].Prefab != "Flint" || a.Items[0].Count != 1 {
		t.Fatal(a)
	}
	v.Inventory = pack(18)
	v.Containers = []*protocol.Container{{Entity: "a", Task: "store", Direction: protocol.TaskDeposit, Moved: map[string]int{"Flint": 1}}}
	nextTask(t, r, v, success("store"), protocol.TaskMoveTo)
	if o, e := r.Next(v, success("return")); o != nil || e != nil {
		t.Fatal(o, e)
	}
}
func TestCollectRejectsUnknownYieldOutOfSiteAndUnrequestedDrops(t *testing.T) {
	p := collectParams()
	p.Return = false
	p.Keep = map[string]int{"Stone": 10}
	r := &collectRun{p: p, seen: map[string]bool{}}
	v := haulView()
	unknown := target("unknown", protocol.ResourcePickable, "Flint", 2)
	unknown.Yield = nil
	v.Perception.Resources = []protocol.PerceivedResource{unknown, target("far", protocol.ResourcePickable, "Flint", 20), target("gift", protocol.ResourceDrop, "Flint", 1), target("wrong", protocol.ResourcePickable, "Raspberry", 1)}
	// Nothing eligible: no action, including no deposit of reserved stone.
	if o, e := r.Next(v, nil); o != nil || e != nil {
		t.Fatal(o, e)
	}
}
func TestCollectCapacityUsesLatestPackAndReservesWeightAndSlots(t *testing.T) {
	p := collectParams()
	r := &collectRun{p: p}
	v := haulView()
	y := &protocol.ResourceYield{Prefab: "Stone", Count: 5, StackMax: 50, Weight: 2}
	if !r.fits(v, y) {
		t.Fatal("expected room")
	}
	v.Inventory = pack(20, material("Stone", 119))
	if r.fits(v, y) {
		t.Fatal("used stale state weight instead of pack")
	}
	v.Inventory.Size = [2]int{1, 1}
	if r.fits(v, y) {
		t.Fatal("cannot assume partial stacks are metadata-compatible")
	}
	v = haulView()
	for _, bad := range []float64{math.NaN(), math.Inf(1), -1} {
		y.Weight = bad
		if r.fits(v, y) {
			t.Fatal("invalid weight accepted")
		}
	}
}
func TestCollectionValidationAndScope(t *testing.T) {
	p := collectParams()
	for _, edit := range []func(*CollectParams){func(p *CollectParams) { p.Site.World = "" }, func(p *CollectParams) { p.MaxTargets = 0 }, func(p *CollectParams) { p.Site.Radius = 50 }, func(p *CollectParams) { p.MaxWeightFraction = 1 }, func(p *CollectParams) { p.Items = []string{"*"} }} {
		q := p
		edit(&q)
		raw, _ := json.Marshal(q)
		if (Collect{}).Validate(raw) == nil {
			t.Fatal(q)
		}
	}
	r := &collectRun{p: p}
	v := haulView()
	v.State.World = "other"
	if _, e := r.Next(v, nil); e == nil {
		t.Fatal("wrong world accepted")
	}
	v = haulView()
	v.Perception.T = 1
	r = &collectRun{p: p, seen: map[string]bool{}}
	if _, e := r.Next(v, nil); !errors.Is(e, ErrWait) {
		t.Fatal(e)
	}
}

func TestEngineStowRecoveryAndOwnerStopBetweenTransfers(t *testing.T) {
	f := setup(t)
	f.p.Send(pack(0, material("Stone", 10)))
	wait(t, func() bool { return f.s.Inventory() != nil })
	d := def()
	d.Template = "stow"
	d.Params = json.RawMessage(`{"world":"test","items":["Stone"],"to":["a","b"],"keep":{"Stone":2}}`)
	f.put(t, "store", d)
	f.act(t, "store", "start")
	f.tick()
	first := f.p.NextTask()
	if first.Task != protocol.TaskDeposit {
		t.Fatal(first)
	}
	f.p.Send(&protocol.Container{Entity: "a", Task: first.ID, Direction: protocol.TaskDeposit, Moved: map[string]int{"Stone": 3}})
	f.p.Send(pack(0, material("Stone", 7)))
	f.done(t, first, protocol.StatusFailed, protocol.ReasonInventoryFull)
	f.tick()
	second := f.p.NextTask()
	if second.Task != protocol.TaskDeposit || !strings.Contains(string(second.Args), `"count":5`) {
		t.Fatal(second)
	}
	// Stop revokes the run before its result/next state can arrive.
	stop, err := f.s.Dispatch(context.Background(), session.Order{Task: protocol.TaskStop})
	if err != nil {
		t.Fatal(err)
	}
	f.p.NextTask()
	f.done(t, second, protocol.StatusSucceeded, "")
	f.tick()
	if f.e.List()[0].Running || f.e.List()[0].Last != "new order" {
		t.Fatal(f.e.List())
	}
	f.p.Report(stop.ID, protocol.StatusSucceeded, "")
}

func TestEngineBoundedObservationWaitCanBeStopped(t *testing.T) {
	f := setup(t)
	d := def()
	d.Template = "collect"
	d.Params = json.RawMessage(`{"site":{"world":"test","pos":[0,0,0],"radius":10},"items":["Flint"],"to":["a"],"maxTargets":1}`)
	f.put(t, "collect", d)
	f.act(t, "collect", "start")
	f.tick()
	// Missing inventory is a bounded wait, not successful completion.
	if !f.e.List()[0].Running {
		t.Fatal(f.e.List())
	}
	f.now = f.now.Add(6 * time.Second)
	f.tick()
	if f.e.List()[0].Running || !strings.Contains(f.e.List()[0].Last, "five seconds") {
		t.Fatal(f.e.List())
	}
	f.act(t, "collect", "start")
	f.tick()
	f.act(t, "collect", "stop")
	if f.e.List()[0].Running {
		t.Fatal(f.e.List())
	}
}

func TestStorageNoticeIsScopedCopiedAndDeduplicated(t *testing.T) {
	f := setup(t)
	lines := []string{}
	f.e.o.Notify = func(_ context.Context, line string) { lines = append(lines, line) }
	need := &StorageNeed{World: "test", Items: map[string]int{"Stone": 8}, Chests: []string{"a"}, Reason: "a: inventory_full"}
	f.e.instances["store"] = &instance{definition: def()}
	fail := func() {
		lease, err := f.s.TryAutomation()
		if err != nil {
			t.Fatal(err)
		}
		f.e.active = "store"
		f.e.lease = lease
		f.e.fail(context.Background(), need, haulView())
	}
	fail()
	fail()
	if len(lines) != 1 {
		t.Fatal(lines)
	}
	f.e.List()[0].StorageNeeded.Items["Stone"] = 999
	if f.e.List()[0].StorageNeeded.Items["Stone"] != 8 {
		t.Fatal("status leaked mutable internal state")
	}
	need.Items = map[string]int{"Stone": 7}
	fail()
	if len(lines) != 2 {
		t.Fatal(lines)
	}
	f.now = f.now.Add(6 * time.Minute)
	fail()
	if len(lines) != 3 {
		t.Fatal(lines)
	}
}

func TestCollectDoesNotChaseProductAlreadyAutoPicked(t *testing.T) {
	p := collectParams()
	p.MaxTargets = 1
	p.Return = false
	r := &collectRun{p: p, seen: map[string]bool{}}
	v := haulView()
	v.Inventory = pack(10)
	v.Perception.Resources = []protocol.PerceivedResource{target("flint", protocol.ResourcePickable, "Flint", 1)}
	nextTask(t, r, v, nil, protocol.TaskInteract)
	v.Inventory = pack(13, material("Flint", 1))
	v.Perception.Resources = []protocol.PerceivedResource{target("stale-product", protocol.ResourceDrop, "Flint", 1)}
	// A new inventory can arrive before the next perception removes the drop.
	o := nextTask(t, r, v, success("pick"), protocol.TaskDeposit)
	if o.Args.(*protocol.TransferArgs).Items[0].Prefab != "Flint" {
		t.Fatal(o)
	}
}

func TestCollectRecoversDisappearingSpawnedDropOnlyWithInventoryEvidence(t *testing.T) {
	p := collectParams()
	p.MaxTargets = 1
	p.Return = false
	makeRun := func() (*collectRun, View) {
		r := &collectRun{p: p, seen: map[string]bool{}}
		v := haulView()
		v.Inventory = pack(10)
		v.Perception.Resources = []protocol.PerceivedResource{target("flint", protocol.ResourcePickable, "Flint", 1)}
		nextTask(t, r, v, nil, protocol.TaskInteract)
		v.Perception.Resources = []protocol.PerceivedResource{target("product", protocol.ResourceDrop, "Flint", 1)}
		nextTask(t, r, v, success("pick"), protocol.TaskPickup)
		return r, v
	}
	rejected := &Result{Status: protocol.StatusRejected, Reason: protocol.ReasonInvalidTarget, TaskID: "take"}
	r, v := makeRun()
	if !r.Recover(*rejected) {
		t.Fatal("pending product rejection must be observed")
	}
	if _, err := r.Next(v, rejected); !errors.Is(err, ErrWait) {
		t.Fatal("must not claim collection with no inventory gain", err)
	}
	v.Inventory = pack(20, material("Flint", 1))
	nextTask(t, r, v, rejected, protocol.TaskDeposit)
	r, v = makeRun()
	v.Inventory = pack(20, material("Flint", 1))
	nextTask(t, r, v, rejected, protocol.TaskDeposit)
	r = &collectRun{task: protocol.TaskPickup}
	if r.Recover(*rejected) {
		t.Fatal("unrelated invalid targets cannot be swallowed")
	}
}

func TestStowFallsBackFromRejectedMissingChestWithoutAReceipt(t *testing.T) {
	r := &stowRun{p: stowParams()}
	v := haulView()
	nextTask(t, r, v, nil, protocol.TaskDeposit)
	rejected := &Result{Status: protocol.StatusRejected, Reason: protocol.ReasonInvalidTarget, TaskID: "first"}
	if !r.Recover(*rejected) {
		t.Fatal("host rejects a vanished chest before transfer starts")
	}
	o := nextTask(t, r, v, rejected, protocol.TaskDeposit)
	if a := o.Args.(*protocol.TransferArgs); a.Container != "b" || a.Items[0].Count != 8 {
		t.Fatal(a)
	}
	if r.Recover(Result{Status: protocol.StatusRejected, Reason: protocol.ReasonNotAllowed}) {
		t.Fatal("permission failure must stop")
	}
}
