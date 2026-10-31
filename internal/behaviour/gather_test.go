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

func gathering(t *testing.T) (*gatherRun, View) {
	t.Helper()
	p := GatherParams{Site: Place{World: "test", Radius: 30}, To: []string{"a"}, Return: true}
	raw, _ := json.Marshal(p)
	runner, err := (Gather{}).Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	v := haulView()
	v.State.HPMax, v.State.Stamina = 25, 60
	v.State.Food = &protocol.FoodStatus{Active: []protocol.ActiveFood{{Prefab: "CookedDeerMeat"}}}
	v.Inventory = pack(10, protocol.InventoryItem{Prefab: "AxeFlint", Stack: 1, Durability: 90, DurabilityMax: 100, Type: "OneHandedWeapon"}, material("Wood", 12), material("BeechSeeds", 15))
	v.Inventory.Size = [2]int{8, 4}
	v.Perception.Resources = []protocol.PerceivedResource{gatherResource("tree", protocol.ResourceTree, 5), gatherResource("old", protocol.ResourceLog, 7)}
	v.Containers = []*protocol.Container{{Entity: "a"}}
	return runner.(*gatherRun), v
}

func gatherResource(id, kind string, x float64) protocol.PerceivedResource {
	tier := 0
	prefab := "beech_log"
	if kind == protocol.ResourceTree {
		prefab = "Beech1"
	}
	return protocol.PerceivedResource{ID: id, Kind: kind, Prefab: prefab, Pos: protocol.Vec3{x, 0, 0}, MinTool: &tier}
}

func gatherFinishedTarget(t *testing.T, r *gatherRun, v *View, wood int, resources ...protocol.PerceivedResource) (*session.Order, error) {
	t.Helper()
	id := r.pending.ID
	v.Perception.Resources = resources
	v.Perception.Seq++
	v.Inventory.Seq++
	v.Inventory.Items[1].Stack += wood
	v.Now = v.Now.Add(time.Second)
	if _, err := r.Next(*v, success(id)); !errors.Is(err, ErrWait) {
		t.Fatalf("expected settling wait: %v", err)
	}
	v.Now = v.Now.Add(2 * time.Second)
	return r.Next(*v, success(id))
}

func TestGatherChainsOnlyNewLogsAndStowsOnlyGainBeforeReturning(t *testing.T) {
	r, v := gathering(t)
	old := v.Perception.Resources[1]
	order := nextTask(t, r, v, nil, protocol.TaskHarvest)
	if args := order.Args.(*protocol.HarvestArgs); args.Entity != "tree" || args.Collect == nil || !*args.Collect {
		t.Fatal(args)
	}
	log := gatherResource("log", protocol.ResourceLog, 6)
	order, err := gatherFinishedTarget(t, r, &v, 0, old, log)
	if err != nil || order == nil || order.Args.(*protocol.HarvestArgs).Entity != "log" {
		t.Fatal(order, err)
	}
	half1, half2 := gatherResource("half1", protocol.ResourceLog, 7), gatherResource("half2", protocol.ResourceLog, 8)
	order, err = gatherFinishedTarget(t, r, &v, 0, old, half1, half2)
	if err != nil || order == nil || order.Args.(*protocol.HarvestArgs).Entity != "half1" {
		t.Fatal(order, err)
	}
	order, err = gatherFinishedTarget(t, r, &v, 10, old, half2)
	if err != nil || order == nil || order.Args.(*protocol.HarvestArgs).Entity != "half2" {
		t.Fatal(order, err)
	}
	// Stow must still run when the completed cut leaves less than the reserve
	// needed to start another harvest. Preserve the original twelve wood.
	v.State.Weight = 250
	order, err = gatherFinishedTarget(t, r, &v, 10, old)
	if err != nil || order == nil || order.Task != protocol.TaskDeposit {
		t.Fatal(order, err)
	}
	args := order.Args.(*protocol.TransferArgs)
	if args.Container != "a" || len(args.Items) != 1 || args.Items[0].Prefab != "Wood" || args.Items[0].Count != 20 {
		t.Fatal(args)
	}
	v.Inventory.Seq++
	v.Inventory.Items[1].Stack = 12
	v.Containers = []*protocol.Container{{Entity: "a", Task: "deposit", Direction: protocol.TaskDeposit, Moved: map[string]int{"Wood": 20}}}
	nextTask(t, r, v, success("deposit"), protocol.TaskMoveTo)
	if order, err = r.Next(v, success("return")); err != nil || order != nil {
		t.Fatal(order, err)
	}
	if r.targets != 4 || r.logs != 3 || v.Inventory.Count("BeechSeeds") != 15 {
		t.Fatal(r.targets, r.logs, v.Inventory)
	}
}

func TestGatherNeedsNewInventoryAndPerceptionAfterMatchingNativeSuccess(t *testing.T) {
	r, v := gathering(t)
	nextTask(t, r, v, nil, protocol.TaskHarvest)
	if _, err := r.Next(v, success("tree")); !errors.Is(err, ErrWait) {
		t.Fatal(err)
	}
	v.Inventory.Seq++
	if _, err := r.Next(v, success("tree")); !errors.Is(err, ErrWait) {
		t.Fatal(err)
	}
	v.Perception.Seq++
	if _, err := r.Next(v, success("tree")); !errors.Is(err, ErrWait) {
		t.Fatal("still observed resource must not be treated as broken", err)
	}
	if r.Recover(Result{Status: protocol.StatusFailed, Reason: protocol.ReasonUnreachable}) || r.Recover(Result{Status: protocol.StatusCancelled}) {
		t.Fatal("harvest failure must not advance the chain")
	}
}

func TestGatherStopsOnHealthLossBetweenNativeTasksButAllowsFoodDecay(t *testing.T) {
	r, v := gathering(t)
	nextTask(t, r, v, nil, protocol.TaskHarvest)
	v.State.HP, v.State.HPMax = 24.9, 24.9
	if _, err := r.Next(v, success("tree")); !errors.Is(err, ErrWait) {
		t.Fatal("ordinary food maximum decay must not be damage", err)
	}
	v.State.HP = 24.7
	if order, err := r.Next(v, success("tree")); order != nil || err == nil || !strings.Contains(err.Error(), "health loss") {
		t.Fatal(order, err)
	}
}

func TestGatherDistinguishesFoodRefreshFromDamageEvenWhenHealthRegenerates(t *testing.T) {
	for _, mode := range []string{"refresh", "hit", "regenerated hit", "no marker", "lost marker", "unexplained loss"} {
		t.Run(mode, func(t *testing.T) {
			r, v := gathering(t)
			seq := uint64(0)
			if mode != "no marker" {
				v.InjurySeq = &seq
			}
			v.State.HP, v.State.HPMax = 60.7, 60.7
			v.State.Food.Active[0].RemainingS = 603
			nextTask(t, r, v, nil, protocol.TaskHarvest)
			v.State.HP, v.State.HPMax = 60, 66.1
			v.State.Food.Active[0].RemainingS = 1197
			if mode == "hit" || mode == "regenerated hit" {
				seq = 100
			}
			if mode == "regenerated hit" {
				v.State.HP = 66.1
			}
			if mode == "lost marker" {
				v.InjurySeq = nil
			}
			if mode == "unexplained loss" {
				v.State.Food.Active[0].RemainingS = 600
			}
			_, err := r.Next(v, success("tree"))
			if mode == "refresh" {
				if !errors.Is(err, ErrWait) {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "health loss") {
				t.Fatal(mode, err)
			}
		})
	}
}

func TestGatherRefusesUnsafeDescendantsWithoutCuttingAnotherTree(t *testing.T) {
	for _, scenario := range []string{"character", "structure", "too far", "no log", "no wood", "old only"} {
		t.Run(scenario, func(t *testing.T) {
			r, v := gathering(t)
			nextTask(t, r, v, nil, protocol.TaskHarvest)
			log := gatherResource("new", protocol.ResourceLog, 6)
			resources := []protocol.PerceivedResource{log, gatherResource("another tree", protocol.ResourceTree, 9)}
			switch scenario {
			case "character":
				v.Perception.Characters = []protocol.PerceivedCharacter{{Kind: protocol.CharacterPlayer, HPFrac: 1, Pos: log.Pos}}
			case "structure":
				v.Perception.Structures = []protocol.PerceivedStructure{{Pos: log.Pos}}
			case "too far":
				resources[0].Pos[0] = 24 // Within 20 m of tree but over 18 m from bot.
			case "no log", "no wood":
				resources = nil
				if scenario == "no wood" {
					r.logs = 3
				}
			case "old only":
				resources = []protocol.PerceivedResource{v.Perception.Resources[1]}
			}
			order, err := gatherFinishedTarget(t, r, &v, 0, resources...)
			if order != nil || err == nil {
				t.Fatal("must stop an incomplete tree/log chain", order, err)
			}
		})
	}
}

func TestGatherHealthToolsCapacityAndScopeGuards(t *testing.T) {
	for _, scenario := range []string{"health", "food", "axe", "weight", "pack weight", "unknown weight", "slots", "stamina", "world", "stale perception", "time", "near character before travel"} {
		t.Run(scenario, func(t *testing.T) {
			r, v := gathering(t)
			switch scenario {
			case "health":
				v.State.HP = 19
			case "food":
				v.State.Food = nil
			case "axe":
				v.Inventory.Items[0].Durability = 0
			case "weight":
				v.State.Weight = 250
			case "pack weight":
				v.Inventory.Items[1].Stack = 150 // State weight has not caught up.
			case "unknown weight":
				v.Inventory.Items[1].Weight = math.NaN()
			case "slots":
				v.Inventory.Size = [2]int{3, 1}
			case "stamina":
				v.State.Stamina = 39
			case "world":
				v.State.World = "other"
			case "stale perception":
				v.State.T += 6
			case "time":
				r.began = v.Now.Add(-8 * time.Minute)
			case "near character before travel":
				v.State.Pos[0] = 50
				v.Perception.Characters = []protocol.PerceivedCharacter{{HPFrac: 1, Pos: v.State.Pos}}
			}
			if order, err := r.Next(v, nil); order != nil || err == nil {
				t.Fatal(order, err)
			}
		})
	}
}

func TestGatherSiteTravelBoundsAndValidation(t *testing.T) {
	r, v := gathering(t)
	v.State.Pos[0] = 50
	nextTask(t, r, v, nil, protocol.TaskMoveTo)
	r, v = gathering(t)
	r.targets = 5
	r.treeOpen, r.trees = true, 1
	r.center = protocol.Vec3{5, 0, 0}
	v.Perception.Resources = []protocol.PerceivedResource{gatherResource("new", protocol.ResourceLog, 6)}
	if order, err := r.Next(v, nil); order != nil || err == nil || !strings.Contains(err.Error(), "five-resource") {
		t.Fatal(order, err)
	}
	for _, raw := range []string{`{}`, `{"site":{"world":"test","radius":41},"to":["a"]}`, `{"site":{"world":"test","radius":20},"to":[]}`, `{"site":{"world":"test","radius":20},"to":["a"],"maxTrees":100}`} {
		if err := (Gather{}).Validate(json.RawMessage(raw)); err == nil {
			t.Fatal(raw)
		}
	}
}

func TestGatherApproachesWorkPointFromInsideABroadZone(t *testing.T) {
	r, v := gathering(t)
	v.State.Pos[0] = 25 // Inside the 30 m zone, beyond reach of its central tree.
	order := nextTask(t, r, v, nil, protocol.TaskMoveTo)
	if got := order.Args.(*protocol.MoveToArgs).To.Pos; got == nil || *got != r.p.Site.Pos {
		t.Fatal("did not approach work point", order)
	}
	v.State.Pos = r.p.Site.Pos
	nextTask(t, r, v, &Result{Status: protocol.StatusSucceeded}, protocol.TaskHarvest)

	r, v = gathering(t)
	r.p.Site.Radius = 5
	v.State.Pos[0] = 6 // A small zone keeps its tighter arrival boundary.
	nextTask(t, r, v, nil, protocol.TaskMoveTo)
}

func TestGatherEngineDoesNotContinueAfterOwnerTakesTheBody(t *testing.T) {
	f := setup(t, protocol.TaskHarvest)
	_, v := gathering(t)
	f.p.Send(v.State)
	f.p.Send(v.Inventory)
	f.p.Send(v.Perception)
	wait(t, func() bool { return f.s.Inventory() != nil && f.s.Perception() != nil && f.s.State().Food != nil })
	d := def()
	d.Template = "gather"
	d.Params = json.RawMessage(`{"site":{"world":"test","pos":[0,0,0],"radius":30},"to":["a"]}`)
	f.put(t, "gather", d)
	f.act(t, "gather", "start")
	f.tick()
	first := f.p.NextTask()
	if first.Task != protocol.TaskHarvest {
		t.Fatal(first)
	}
	pos := protocol.Vec3{50, 0, 0}
	if _, err := f.s.Dispatch(context.Background(), session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}}, TimeoutS: 10}); err != nil {
		t.Fatal(err)
	}
	f.p.NextTask()
	f.p.Report(first.ID, protocol.StatusSucceeded, "")
	f.tick()
	f.p.Quiet(30 * time.Millisecond)
	if status := f.e.List()[0]; status.Running || status.Pending || status.Last != "new order" {
		t.Fatal(status)
	}
}
