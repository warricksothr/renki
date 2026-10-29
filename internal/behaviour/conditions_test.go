package behaviour

import (
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"testing"
	"time"
)

func ptr[T any](v T) *T { return &v }
func TestConditionsUseWeightFlagsOwnersAndRecentChestContents(t *testing.T) {
	home := protocol.Vec3{}
	view := View{
		State:      &protocol.State{Envelope: protocol.Envelope{T: 100}, Phase: protocol.PhaseInWorld, World: "test", HP: 25, Weight: 150, WeightMax: 300},
		Inventory:  &protocol.Inventory{Size: [2]int{2, 1}, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 10}}},
		Perception: &protocol.Perception{Envelope: protocol.Envelope{T: 99}, Characters: []protocol.PerceivedCharacter{{Kind: protocol.CharacterPlayer, Name: "Nimah", HPFrac: 1, Pos: protocol.Vec3{2, 0, 0}}}},
		Containers: []*protocol.Container{{Envelope: protocol.Envelope{T: 98}, Entity: "chest", Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 20}}}},
		Owners:     []string{"Nimah"}, Home: &home,
	}
	c := Conditions{Items: map[string]int{"Wood": 10}, MinFreeSlots: 1, MaxWeightFraction: ptr(.5), Encumbered: ptr(false), Stranded: ptr(false), Night: ptr(false), Owner: &OwnerNear{Name: "Nimah", Radius: 3}, AtHomeRadius: ptr(2.0), Near: &Place{World: "test", Radius: 3}, Stock: []Stock{{Container: "chest", Prefab: "Wood", MinCount: 20, MaxAgeS: 5}}}
	if !c.Match(view) {
		t.Fatal("valid observations refused")
	}
	for _, change := range []string{"items", "slots", "weight", "encumbered", "stranded", "night", "owner", "old owner", "world", "raised floor", "old stock", "future stock", "unknown stock"} {
		t.Run(change, func(t *testing.T) {
			v := view
			state := *view.State
			v.State = &state
			cond := c
			switch change {
			case "items":
				cond.Items = map[string]int{"Wood": 11}
			case "slots":
				cond.MinFreeSlots = 2
			case "weight":
				state.Weight = 151
			case "encumbered":
				state.Flags = []string{protocol.FlagEncumbered}
			case "stranded":
				state.Flags = []string{protocol.FlagStranded}
			case "night":
				state.Time.Night = true
			case "owner":
				v.Owners = []string{"someone else"}
			case "old owner":
				p := *view.Perception
				p.T = 90
				v.Perception = &p
			case "world":
				state.World = "other"
			case "raised floor":
				state.Pos[1] = 4
			case "old stock":
				chest := *view.Containers[0]
				chest.T = 90
				v.Containers = []*protocol.Container{&chest}
			case "future stock":
				chest := *view.Containers[0]
				chest.T = 101
				v.Containers = []*protocol.Container{&chest}
			case "unknown stock":
				v.Containers = nil
			}
			if cond.Match(v) {
				t.Fatal("condition accepted missing/stale evidence")
			}
		})
	}
}
func TestWhileAndUntilRespectTheirConditionAndMaximumRuns(t *testing.T) {
	for _, mode := range []string{"while", "until"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t)
			d := def()
			d.Repeat = Repeat{Mode: mode, MaxRuns: 2, Condition: &Conditions{Night: ptr(mode == "until")}}
			f.put(t, "repeat", d)
			f.act(t, "repeat", "start")
			f.tick()
			first := f.p.NextTask()
			f.done(t, first, protocol.StatusSucceeded, "")
			f.tick()
			f.now = f.now.Add(1000000000)
			f.tick()
			second := f.p.NextTask()
			f.done(t, second, protocol.StatusSucceeded, "")
			f.tick()
			f.now = f.now.Add(1000000000)
			f.tick()
			if list := f.e.List(); list[0].Runs != 2 || list[0].Running || list[0].Pending {
				t.Fatal(list)
			}
		})
	}
}
func TestManualStartBeforeFirstSchedulerTickIsNotLost(t *testing.T) {
	f := setup(t)
	e := New(f.e.o)
	f.e = e
	f.put(t, "start", def())
	f.act(t, "start", "start")
	f.tick()
	f.p.NextTask()
}

func TestRememberedStockUsesItsAgeAndLiveEmptyChestWins(t *testing.T) {
	f := setup(t)
	f.e.o.Containers = func(world string) []home.RememberedContainer {
		if world != "test" {
			t.Fatal(world)
		}
		return []home.RememberedContainer{{Entity: "chest", Size: [2]int{2, 2}, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 10}}, At: f.now.Add(-2 * time.Second)}}
	}
	c := Conditions{Stock: []Stock{{Container: "chest", Prefab: "Wood", MinCount: 10, MaxAgeS: 3}}}
	if !c.Match(f.e.view()) {
		t.Fatal("fresh remembered stock ignored")
	}
	c.Stock[0].MaxAgeS = 1
	if c.Match(f.e.view()) {
		t.Fatal("stale remembered stock accepted")
	}
	c.Stock[0].MaxAgeS = 3
	f.p.Send(&protocol.Container{Entity: "chest", Size: [2]int{2, 2}})
	wait(t, func() bool { return len(f.s.Containers()) == 1 })
	if c.Match(f.e.view()) {
		t.Fatal("old memory overruled live empty chest")
	}
}
