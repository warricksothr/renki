package behaviour

import (
	"errors"
	"testing"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// Finish one full tree and all three newly observed log targets. The next
// action must come from the runner, not from this test choosing another tree.
func finishGatherTree(t *testing.T, r *gatherRun, v *View, prefix string, after ...protocol.PerceivedResource) *session.Order {
	t.Helper()
	log := gatherResource(prefix+"log", protocol.ResourceLog, 6)
	a, b := gatherResource(prefix+"a", protocol.ResourceLog, 7), gatherResource(prefix+"b", protocol.ResourceLog, 8)
	for i, step := range []struct {
		gain      int
		resources []protocol.PerceivedResource
	}{
		{0, []protocol.PerceivedResource{log}}, {0, []protocol.PerceivedResource{a, b}},
		{10, []protocol.PerceivedResource{b}}, {10, nil},
	} {
		resources := append(append([]protocol.PerceivedResource{}, after...), step.resources...)
		order, err := gatherFinishedTarget(t, r, v, step.gain, resources...)
		if err != nil || order == nil {
			t.Fatalf("tree step %d: %v %v", i, order, err)
		}
		if i == 3 {
			return order
		}
		if order.Task != protocol.TaskHarvest {
			t.Fatalf("unfinished tree advanced to %s", order.Task)
		}
	}
	panic("unreachable")
}

func TestGatherTwoTreesThenVerifiedPinnedHomeAndGainsOnlyDeposit(t *testing.T) {
	r, v := gathering(t)
	r.p.MaxTrees, r.p.Return, r.p.ReturnHome = 2, false, true
	home := protocol.Vec3{0, 1.5, -2}
	v.Home = &home
	second := gatherResource("second", protocol.ResourceTree, 9)
	v.Perception.Resources = append(v.Perception.Resources, second)
	nextTask(t, r, v, nil, protocol.TaskHarvest)
	order := finishGatherTree(t, r, &v, "one", second)
	if order.Task != protocol.TaskHarvest || order.Args.(*protocol.HarvestArgs).Entity != "second" {
		t.Fatal(order)
	}
	order = finishGatherTree(t, r, &v, "two")
	if order.Task != protocol.TaskMoveTo || (!order.Args.(*protocol.MoveToArgs).Dry || !order.Args.(*protocol.MoveToArgs).AvoidHostiles) || *order.Args.(*protocol.MoveToArgs).To.Pos != home {
		t.Fatal(order)
	}
	changed := protocol.Vec3{80, 0, 80}
	v.Home = &changed
	if _, err := r.Next(v, success("home")); !errors.Is(err, ErrWait) {
		t.Fatal(err)
	}
	// An older pose cannot confirm receipt, nor can the same sequence.
	if _, err := r.Next(v, success("home")); !errors.Is(err, ErrWait) {
		t.Fatal(err)
	}
	v.State.Seq++
	v.State.Pos = protocol.Vec3{0, 0, -2}
	order = nextTask(t, r, v, success("home"), protocol.TaskDeposit)
	args := order.Args.(*protocol.TransferArgs)
	if len(args.Items) != 1 || args.Items[0].Prefab != "Wood" || args.Items[0].Count != 40 {
		t.Fatal(args)
	}
	v.Inventory.Seq++
	v.Inventory.Items[1].Stack = 12
	v.Containers = []*protocol.Container{{Entity: "a", Task: "deposit", Direction: protocol.TaskDeposit, Moved: map[string]int{"Wood": 40}}}
	order = nextTask(t, r, v, success("deposit"), protocol.TaskMoveTo)
	if *order.Args.(*protocol.MoveToArgs).To.Pos != home {
		t.Fatal("home pin changed", order)
	}
	if _, err := r.Next(v, success("finish")); !errors.Is(err, ErrWait) {
		t.Fatal(err)
	}
	v.State.Seq++
	if order, err := r.Next(v, success("finish")); err != nil || order != nil || !r.finished {
		t.Fatal(order, err)
	}
	if r.trees != 2 || r.logs != 6 || v.Inventory.Count("BeechSeeds") != 15 {
		t.Fatal(r.trees, r.logs, v.Inventory)
	}
}

func TestGatherDeliversProductsCollectedOnTheOutboundTrip(t *testing.T) {
	r, v := gathering(t)
	v.State.Pos[0] = 50
	nextTask(t, r, v, nil, protocol.TaskMoveTo)
	v.State.Pos[0] = 0
	v.Inventory.Seq++
	v.Inventory.Items[1].Stack += 5
	nextTask(t, r, v, success("site"), protocol.TaskHarvest)
	order := finishGatherTree(t, r, &v, "one")
	if order.Task != protocol.TaskDeposit {
		t.Fatal(order)
	}
	args := order.Args.(*protocol.TransferArgs)
	if len(args.Items) != 1 || args.Items[0].Prefab != "Wood" || args.Items[0].Count != 25 || r.keep["Wood"] != 12 {
		t.Fatal(args, r.keep)
	}
}

func TestGatherCapacityPressureFinishesCargoBeforeSelectingAnotherTree(t *testing.T) {
	r, v := gathering(t)
	r.p.MaxTrees = 2
	second := gatherResource("second", protocol.ResourceTree, 9)
	v.Perception.Resources = append(v.Perception.Resources, second)
	nextTask(t, r, v, nil, protocol.TaskHarvest)
	// The reserve is consumed by the final cut, after all earlier cuts passed.
	log := gatherResource("log", protocol.ResourceLog, 6)
	a, b := gatherResource("a", protocol.ResourceLog, 7), gatherResource("b", protocol.ResourceLog, 8)
	for _, step := range []struct {
		gain      int
		resources []protocol.PerceivedResource
	}{
		{0, []protocol.PerceivedResource{second, log}}, {0, []protocol.PerceivedResource{second, a, b}}, {10, []protocol.PerceivedResource{second, b}},
	} {
		if _, err := gatherFinishedTarget(t, r, &v, step.gain, step.resources...); err != nil {
			t.Fatal(err)
		}
	}
	v.State.Weight = 250
	order, err := gatherFinishedTarget(t, r, &v, 10, second)
	if err != nil || order == nil || order.Task != protocol.TaskDeposit || r.trees != 1 {
		t.Fatal(order, err, r.trees)
	}
}

func TestGatherHomeProofCannotComeFromBelowTheFloor(t *testing.T) {
	r, v := gathering(t)
	r.p.Return = false
	r.p.ReturnHome = true
	home := protocol.Vec3{0, 5, 0}
	v.Home = &home
	nextTask(t, r, v, nil, protocol.TaskHarvest)
	finishGatherTree(t, r, &v, "one")
	if _, err := r.Next(v, success("home")); !errors.Is(err, ErrWait) {
		t.Fatal(err)
	}
	v.State.Seq++
	if order, err := r.Next(v, success("home")); order != nil || err == nil {
		t.Fatal("under-floor pose authorized storage", order, err)
	}
}

func TestGatherRepairsAtHomeThenRequiresFreshActuallyRestoredAxeBeforeLeaving(t *testing.T) {
	for _, mode := range []string{"repaired", "stale", "unchanged", "failed", "wrong station"} {
		t.Run(mode, func(t *testing.T) {
			r, v := gathering(t)
			r.p.Return = false
			r.p.ReturnHome = true
			r.p.Site.Pos = protocol.Vec3{50, 0, 0}
			home := protocol.Vec3{0, 0, 0}
			v.Home = &home
			v.Inventory.Items[0].Durability = 38
			v.Perception.Structures = []protocol.PerceivedStructure{{ID: "bench", Kind: protocol.StructureStation, Prefab: "piece_workbench"}}
			if mode == "wrong station" {
				v.Perception.Structures[0].Prefab = "piece_cookingstation"
				if order, err := r.Next(v, nil); order != nil || err == nil {
					t.Fatal(order, err)
				}
				return
			}
			order := nextTask(t, r, v, nil, protocol.TaskInteract)
			if args := order.Args.(*protocol.InteractArgs); !args.Repair || args.Entity != "bench" {
				t.Fatal(args)
			}
			result := success("repair")
			if mode == "failed" {
				result = &Result{Status: protocol.StatusFailed}
				if order, err := r.Next(v, result); order != nil || err == nil {
					t.Fatal(order, err)
				}
				return
			}
			if mode != "stale" {
				v.Inventory.Seq++
			}
			if mode != "unchanged" {
				v.Inventory.Items[0].Durability = 100
			}
			order, err := r.Next(v, result)
			switch mode {
			case "stale":
				if order != nil || !errors.Is(err, ErrWait) {
					t.Fatal(order, err)
				}
			case "unchanged":
				if order != nil || err == nil {
					t.Fatal(order, err)
				}
			case "repaired":
				if err != nil || order == nil || order.Task != protocol.TaskMoveTo || (!order.Args.(*protocol.MoveToArgs).Dry || !order.Args.(*protocol.MoveToArgs).AvoidHostiles) {
					t.Fatal(order, err)
				}
			}
		})
	}
}

func TestGatherCanRepairAndLeaveWithPlayerAtHomeButNeverSwingNearThem(t *testing.T) {
	r, v := gathering(t)
	r.p.Return, r.p.ReturnHome = false, true
	r.p.Site.Pos = protocol.Vec3{50, 0, 0}
	home := protocol.Vec3{0, 0, 0}
	v.Home = &home
	v.Inventory.Items[0].Durability = 10
	v.Perception.Structures = []protocol.PerceivedStructure{{ID: "bench", Kind: protocol.StructureStation, Prefab: "piece_workbench"}}
	v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "friend", Kind: protocol.CharacterPlayer, HPFrac: .49}}
	nextTask(t, r, v, nil, protocol.TaskInteract)
	v.Inventory.Seq++
	v.Inventory.Items[0].Durability = 100
	nextTask(t, r, v, success("repair"), protocol.TaskMoveTo)
	v.State.Pos, v.Perception.Characters[0].Pos = r.p.Site.Pos, r.p.Site.Pos
	v.Perception.Structures = nil
	v.Perception.Resources = []protocol.PerceivedResource{gatherResource("near friend", protocol.ResourceTree, 55)}
	if order, err := r.Next(v, success("site")); order != nil || err == nil {
		t.Fatal("travel permission authorized swinging near a player", order, err)
	}
}

func TestGatherCannotIgnoreNearbyThreatToRepairABrokenAxe(t *testing.T) {
	for _, targeting := range []bool{false, true} {
		r, v := gathering(t)
		r.p.Return, r.p.ReturnHome = false, true
		home := protocol.Vec3{}
		v.Home = &home
		v.Inventory.Items[0].Durability = 10
		v.Perception.Structures = []protocol.PerceivedStructure{{ID: "bench", Kind: protocol.StructureStation, Prefab: "piece_workbench"}}
		v.Perception.Characters = []protocol.PerceivedCharacter{{HPFrac: 1, Hostile: !targeting, TargetingMe: targeting}}
		if order, err := r.Next(v, nil); order != nil || err == nil {
			t.Fatal(order, err)
		}
	}
}
