package behaviour

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func woodView(t *testing.T) View {
	_, v := gathering(t)
	v.Now = time.Now()
	v.StateAt, v.PerceptionAt = v.Now, v.Now
	h := protocol.Vec3{0, 0, -50}
	v.Home = &h
	v.Containers = []*protocol.Container{{Entity: "home", Pos: h}}
	return v
}

func TestWoodPlanSelectsSafeTreeAndHomeChestHintsPreservingReturn(t *testing.T) {
	v := woodView(t)
	v.Perception.Resources = []protocol.PerceivedResource{gatherResource("unsafe", protocol.ResourceTree, 10), gatherResource("safe", protocol.ResourceTree, 50)}
	v.Perception.Structures = []protocol.PerceivedStructure{{ID: "building", Pos: protocol.Vec3{0, 0, 0}}}
	p, remembered, err := planWood(v, nil, []home.RememberedContainer{{Entity: "far", Pos: protocol.Vec3{100, 0, 100}}})
	if err != nil || remembered || p.Site.Pos[0] != 50 || !p.ReturnHome || p.Return || p.MaxTrees != 2 || len(p.To) != 1 || p.To[0] != "home" {
		t.Fatal(p, remembered, err)
	}
}

func TestWoodPlanAvoidsKnownSolidWallsWithoutAnInteractable(t *testing.T) {
	v := woodView(t)
	v.Perception.Resources = []protocol.PerceivedResource{gatherResource("near wall", protocol.ResourceTree, 10), gatherResource("clear", protocol.ResourceTree, 50)}
	v.Walls = []protocol.Wall{{A: protocol.Vec3{0, 0, -50}, B: protocol.Vec3{0, 0, 50}, H: 2}}
	p, _, err := planWood(v, nil, nil)
	if err != nil || p.Site.Pos[0] != 50 {
		t.Fatal(p, err)
	}
}

func TestWoodPlanReusesSameWorldSiteButDoesNotReuseUnsafeLocalOrForeignSites(t *testing.T) {
	v := woodView(t)
	last := GatherParams{Site: Place{World: "test", Pos: protocol.Vec3{80, 0, 0}, Radius: 20}}
	p, remembered, err := planWood(v, &last, nil)
	if err != nil || !remembered || p.Site.Pos != last.Site.Pos {
		t.Fatal(p, remembered, err)
	}
	last.Site.World = "other"
	p, remembered, err = planWood(v, &last, nil)
	if err != nil || remembered || p.Site.Pos[0] != 5 {
		t.Fatal(p, remembered, err)
	}
	last.Site.World, last.Site.Pos = "test", protocol.Vec3{50, 0, 0}
	v.Perception.Resources = append(v.Perception.Resources, gatherResource("old site tree", protocol.ResourceTree, 45))
	p, remembered, err = planWood(v, &last, nil)
	if err != nil || !remembered || p.Site.Pos[0] != 45 {
		t.Fatal(p, remembered, err)
	}
	v.Perception.Structures = []protocol.PerceivedStructure{{ID: "new building", Pos: protocol.Vec3{60, 0, 0}}}
	p, remembered, err = planWood(v, &last, nil)
	if err != nil || remembered || p.Site.Pos[0] != 5 {
		t.Fatal(p, remembered, err)
	}
}

func TestWoodPlanRefusesMissingPrerequisitesBeforeStartingWork(t *testing.T) {
	for _, name := range []string{"axe", "food", "storage", "home", "stale", "character", "tree", "stamina", "capacity"} {
		t.Run(name, func(t *testing.T) {
			v := woodView(t)
			switch name {
			case "axe":
				v.Inventory.Items = v.Inventory.Items[1:]
			case "food":
				v.State.Food = nil
			case "storage":
				v.Containers = nil
			case "home":
				v.Home = nil
			case "stale":
				v.StateAt = v.Now.Add(-4 * time.Second)
			case "character":
				v.Perception.Characters = []protocol.PerceivedCharacter{{HPFrac: 1, Pos: protocol.Vec3{5, 0, 0}}}
			case "tree":
				v.Perception.Resources = nil
			case "stamina":
				v.State.Stamina = 10
			case "capacity":
				v.State.Weight = 290
			}
			if _, _, err := planWood(v, nil, nil); err == nil {
				t.Fatal("accepted missing", name)
			}
		})
	}
}

func TestWoodPlanCanVisitARecentDistantTreeHintWithoutTreatingItAsCutPermission(t *testing.T) {
	v := woodView(t)
	v.Perception.Resources = nil
	tier := 0
	hint := home.Landmark{Entity: "beech", Kind: protocol.ResourceTree, Prefab: "Beech1", MinTool: &tier, Pos: protocol.Vec3{80, 0, 0}, At: v.Now}
	p, _, err := planWood(v, nil, nil, hint)
	if err != nil || p.Site.Pos != hint.Pos || p.MaxTrees != 2 || !p.ReturnHome {
		t.Fatal(p, err)
	}
	for _, mode := range []string{"old", "foreign range", "near absent", "future", "blocked"} {
		vv, hh := v, hint
		switch mode {
		case "old":
			hh.At = v.Now.Add(-8 * 24 * time.Hour)
		case "foreign range":
			hh.Pos[0] = 200
		case "near absent":
			hh.Pos[0] = 10
		case "future":
			hh.At = v.Now.Add(time.Second)
		case "blocked":
			vv.Perception = &protocol.Perception{Envelope: v.Perception.Envelope, Structures: []protocol.PerceivedStructure{{Pos: hint.Pos}}}
		}
		if _, _, err := planWood(vv, nil, nil, hh); err == nil {
			t.Fatal("accepted", mode)
		}
	}
}

func woodEngine(t *testing.T) *fixture {
	t.Helper()
	f := setup(t, protocol.TaskHarvest)
	v := woodView(t)
	f.e.o.Now = time.Now
	f.e.o.Home = func() (protocol.Vec3, bool) { return *v.Home, true }
	f.p.Send(v.State)
	f.p.Send(v.Inventory)
	f.p.Send(v.Perception)
	f.p.Send(v.Containers[0])
	wait(t, func() bool { return f.s.Perception() != nil && f.s.Inventory() != nil && len(f.s.Containers()) == 1 })
	return f
}

func TestWoodCommandQueuesOnceAndNewOwnerStopCancelsBeforeAnyWork(t *testing.T) {
	f := woodEngine(t)
	reply, err := f.e.StartWood(context.Background(), "Nimah")
	if err != nil || !strings.Contains(reply, "two beeches") {
		t.Fatal(reply, err)
	}
	statuses := f.e.List()
	if len(statuses) != 1 || !statuses[0].Pending || statuses[0].Definition.Trigger.Kind != "manual" || statuses[0].Definition.Repeat.Mode != "once" {
		t.Fatal(statuses)
	}
	if _, err := f.e.StartWood(context.Background(), "Nimah"); err == nil {
		t.Fatal("replaced queued work")
	}
	if _, err := f.s.Dispatch(context.Background(), session.Order{Task: protocol.TaskStop}); err != nil {
		t.Fatal(err)
	}
	f.e.Tick(context.Background())
	if f.e.List()[0].Pending || f.e.List()[0].Running {
		t.Fatal(f.e.List())
	}
	// Reopening settings/engine preserves the destination without resuming work.
	raw := f.store.Instances()[woodInstance].Value
	var d Definition
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	reopened := New(f.e.o)
	if reopened.List()[0].Pending || reopened.List()[0].Running {
		t.Fatal("resumed on restart")
	}
}

func TestNewOrderDuringWoodPlanningCannotBeReplacedByPendingGather(t *testing.T) {
	f := woodEngine(t)
	ordered := false
	f.e.o.Containers = func(string) []home.RememberedContainer {
		if !ordered {
			ordered = true
			if _, err := f.s.Dispatch(context.Background(), session.Order{Task: protocol.TaskStop}); err != nil {
				t.Fatal(err)
			}
		}
		return nil
	}
	_, err := f.e.StartWood(context.Background(), "Nimah")
	if err == nil || !strings.Contains(err.Error(), "newer order") {
		t.Fatal(err)
	}
	if f.e.List()[0].Pending || f.e.List()[0].Running {
		t.Fatal("old request survived newer stop")
	}
}

func TestWoodCommandCannotOverwriteOwnerDefinitionsOrBypassPermissions(t *testing.T) {
	for _, collision := range []bool{false, true} {
		f := setup(t)
		if collision {
			f = setup(t, protocol.TaskHarvest)
			raw, _ := json.Marshal(def())
			if err := f.e.Put(context.Background(), woodInstance, raw, "panel"); err != nil {
				t.Fatal(err)
			}
		}
		_, err := f.e.StartWood(context.Background(), "Nimah")
		if err == nil || collision && !strings.Contains(err.Error(), "rename") || !collision && !strings.Contains(err.Error(), "not allowed") {
			t.Fatal(collision, err)
		}
		if !collision && len(f.store.Instances()) != 0 {
			t.Fatal("saved disallowed work")
		}
		if collision && f.store.Instances()[woodInstance].By != "panel" {
			t.Fatal("overwrote owner definition")
		}
	}
}
