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
	if order.Task != protocol.TaskMoveTo || !order.Args.(*protocol.MoveToArgs).Dry || *order.Args.(*protocol.MoveToArgs).To.Pos != home {
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
