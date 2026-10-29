package burden

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func item(prefab, typ string, stack int, weight float64) protocol.InventoryItem {
	return protocol.InventoryItem{Prefab: prefab, Type: typ, Stack: stack, Weight: weight}
}

// The pack tas was stuck with in the game: 124 Wood, 16 Stone, 13 Flint,
// and its gear.
func pack() *protocol.Inventory {
	axe := item("AxeStone", "TwoHandedWeapon", 1, 3)
	amber := item("Amber", "Material", 3, 0.1)
	amber.Value = 5
	worn := item("ArmorLeatherChest", "Chest", 1, 5)
	worn.Equipped = true
	return &protocol.Inventory{Size: [2]int{8, 4}, Items: []protocol.InventoryItem{
		axe, worn, amber,
		item("Wood", "Material", 50, 2), item("Wood", "Material", 50, 2), item("Wood", "Material", 24, 2),
		item("Stone", "Material", 16, 2), item("Flint", "Material", 13, 1),
		item("LeatherScraps", "Material", 10, 0.5), item("CookedMeat", "Consumable", 5, 1),
	}}
}

func TestPlanDropsCommonHeavyThingsFirst(t *testing.T) {
	cases := []struct {
		weight float64
		want   []Drop
		short  bool
	}{
		{306, []Drop{{"Wood", 3}}, false},   // heaviest common, the most of it
		{300, nil, false},                   // at the limit is not over it
		{300.5, []Drop{{"Wood", 1}}, false}, // a part of one is still one
		{560, []Drop{{"Wood", 124}, {"Stone", 6}}, false},
		{600, []Drop{{"Wood", 124}, {"Stone", 16}, {"Flint", 13}, {"LeatherScraps", 10}, {"CookedMeat", 2}}, false},
		// Then valuables last, and never gear: not enough.
		{620, []Drop{{"Wood", 124}, {"Stone", 16}, {"Flint", 13}, {"LeatherScraps", 10}, {"CookedMeat", 5}, {"Amber", 3}}, true},
	}
	for _, c := range cases {
		got, short := Plan(pack(), c.weight, 300)
		if !reflect.DeepEqual(got, c.want) || short != c.short {
			t.Errorf("at %v: %v short %v\nwant  %v short %v", c.weight, got, short, c.want, c.short)
		}
	}
}

type fake struct {
	mu    sync.Mutex
	st    *protocol.State
	inv   *protocol.Inventory
	seen  *protocol.Perception
	sent  []string
	drops map[string]int // prefab → how many the plugin drops; absent drops all asked
}

func (f *fake) State() *protocol.State {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.st
}
func (f *fake) Inventory() *protocol.Inventory   { return f.inv }
func (f *fake) Perception() *protocol.Perception { return f.seen }

func (f *fake) Dispatch(_ context.Context, o session.Order) (*session.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch a := o.Args.(type) {
	case *protocol.DropArgs:
		f.sent = append(f.sent, fmt.Sprintf("drop %d %s", a.Count, a.Prefab))
		n := a.Count
		if m, ok := f.drops[a.Prefab]; ok {
			n = m
		}
		if n == 0 {
			return session.FinishedWith(o.Task, nil, protocol.StatusFailed, protocol.ReasonInvalidTarget, "no "+a.Prefab), nil
		}
		return session.FinishedWith(o.Task, nil, protocol.StatusSucceeded, "", fmt.Sprintf("dropped %d %s", n, a.Prefab)), nil
	case *protocol.SayArgs:
		f.sent = append(f.sent, "say: "+a.Text)
	}
	return session.Finished(o.Task, nil, protocol.StatusSucceeded, ""), nil
}

func (f *fake) said() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.sent...)
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func overloaded(weight float64) *protocol.State {
	return &protocol.State{Phase: protocol.PhaseInWorld, World: "Midgard", Pos: protocol.Vec3{10, 0, 20},
		Weight: weight, WeightMax: 300, Flags: []string{protocol.FlagEncumbered, protocol.FlagStranded}}
}

func unloader(on bool) (*Unloader, *fake, *home.Store, *clock) {
	f := &fake{inv: pack()}
	store := home.Memory()
	c := &clock{t: time.Date(2026, 10, 28, 23, 0, 0, 0, time.UTC)}
	return New(on, f, store, slog.New(slog.NewTextHandler(io.Discard, nil)), c.now), f, store, c
}

func TestOverloadedBotDropsSaysAndRemembers(t *testing.T) {
	u, f, store, c := unloader(true)
	ctx := context.Background()
	f.st = overloaded(310)
	u.Tick(ctx)
	want := []string{"drop 5 Wood", "say: I was carrying too much, so I left 5 Wood here. I'll remember where."}
	if got := f.said(); !reflect.DeepEqual(got, want) {
		t.Fatalf("sent %q\nwant %q", got, want)
	}
	left := store.LeftBehind("Midgard")
	if len(left) != 1 || left[0].Pos != (protocol.Vec3{10, 0, 20}) || left[0].Items["Wood"] != 5 {
		t.Fatalf("remembered %+v", left)
	}

	// Still over before the state catches up: it waits for that first.
	u.Tick(ctx)
	if len(f.said()) != 2 {
		t.Fatalf("dropped again before it settled: %q", f.said())
	}
	// Still over once it has: more, added to the same place.
	c.t = c.t.Add(Settle)
	f.st = overloaded(301)
	f.st.Pos = protocol.Vec3{11, 0, 21}
	u.Tick(ctx)
	if left := store.LeftBehind("Midgard"); len(left) != 1 || left[0].Items["Wood"] != 6 {
		t.Errorf("remembered %+v after a second drop beside the first", left)
	}
}

func TestOnlyWhatTheGameDroppedIsRemembered(t *testing.T) {
	u, f, store, _ := unloader(true)
	f.drops = map[string]int{"Wood": 2}
	f.st = overloaded(310)
	u.Tick(context.Background())
	if left := store.LeftBehind("Midgard"); len(left) != 1 || left[0].Items["Wood"] != 2 {
		t.Errorf("remembered %+v", left)
	}
	f.drops = map[string]int{"Wood": 0}
	f.st.Pos = protocol.Vec3{90, 0, 90}
	u.next = time.Time{}
	u.Tick(context.Background())
	if left := store.LeftBehind("Midgard"); len(left) != 1 {
		t.Errorf("a drop that failed was remembered: %+v", left)
	}
}

func TestNotAllowedToDropSaysSoOnce(t *testing.T) {
	u, f, store, c := unloader(true)
	f.drops = map[string]int{"Wood": 0}
	f.st = overloaded(310)
	u.Tick(context.Background())
	c.t = c.t.Add(Settle)
	u.Tick(context.Background())
	want := []string{"drop 5 Wood", "say: I'm carrying too much, and I can't put anything down."}
	if got := f.said(); !reflect.DeepEqual(got, want) || len(store.LeftBehind("Midgard")) != 0 {
		t.Fatalf("sent %q\nwant %q", got, want)
	}
	c.t = c.t.Add(Retry)
	u.Tick(context.Background())
	if got := f.said(); len(got) != 3 || got[2] != "drop 5 Wood" {
		t.Errorf("after Retry: %q", got)
	}
}

func TestGearIsNeverDropped(t *testing.T) {
	u, f, store, c := unloader(true)
	axe := item("AxeStone", "TwoHandedWeapon", 1, 3)
	f.inv = &protocol.Inventory{Items: []protocol.InventoryItem{axe}}
	f.st = overloaded(303)
	u.Tick(context.Background())
	c.t = c.t.Add(Settle)
	u.Tick(context.Background())
	want := []string{"say: I'm carrying too much, and all of it is gear I won't leave behind."}
	if got := f.said(); !reflect.DeepEqual(got, want) || len(store.LeftBehind("Midgard")) != 0 {
		t.Errorf("sent %q", got)
	}
}

func TestSwitchedOffLeavesTheLoad(t *testing.T) {
	u, f, _, _ := unloader(false)
	f.st = overloaded(310)
	u.Tick(context.Background())
	if len(f.said()) != 0 {
		t.Errorf("sent %q with dropping off", f.said())
	}
}

func TestWhatWasLeftIsForgottenOnceGone(t *testing.T) {
	u, f, store, c := unloader(true)
	store.UpdateLeftBehind("Midgard", func([]home.Cache) []home.Cache {
		return []home.Cache{{Pos: protocol.Vec3{0, 0, 0}, Items: map[string]int{"Wood": 5, "Stone": 2}}}
	})
	f.st = &protocol.State{Phase: protocol.PhaseInWorld, World: "Midgard", Pos: protocol.Vec3{30, 0, 0}, WeightMax: 300}
	f.seen = &protocol.Perception{}
	// Too far off to tell.
	u.Tick(context.Background())
	c.t = c.t.Add(GoneAfter)
	u.Tick(context.Background())
	if left := store.LeftBehind("Midgard"); len(left) != 1 || len(left[0].Items) != 2 {
		t.Fatalf("forgot from afar: %+v", left)
	}
	// Close by: the wood is still there, the stone is not.
	f.st.Pos = protocol.Vec3{5, 0, 0}
	f.seen = &protocol.Perception{Resources: []protocol.PerceivedResource{
		{Kind: protocol.ResourceDrop, Prefab: "Wood", Pos: protocol.Vec3{2, 0, 1}, Stack: 5}}}
	u.Tick(context.Background())
	if left := store.LeftBehind("Midgard"); len(left[0].Items) != 2 {
		t.Fatalf("forgot at the first look: %+v", left)
	}
	c.t = c.t.Add(GoneAfter)
	u.Tick(context.Background())
	left := store.LeftBehind("Midgard")
	if len(left) != 1 || left[0].Items["Wood"] != 5 || left[0].Items["Stone"] != 0 {
		t.Fatalf("after the stone was gone a while: %+v", left)
	}
	// A miss, then away and back much later: the grace starts again.
	f.seen = &protocol.Perception{}
	u.Tick(context.Background())
	f.st.Pos = protocol.Vec3{60, 0, 0}
	u.Tick(context.Background())
	c.t = c.t.Add(time.Hour)
	f.st.Pos = protocol.Vec3{5, 0, 0}
	u.Tick(context.Background())
	if left := store.LeftBehind("Midgard"); len(left) != 1 {
		t.Fatalf("forgot at the first look after coming back: %+v", left)
	}
	c.t = c.t.Add(GoneAfter)
	u.Tick(context.Background())
	if left := store.LeftBehind("Midgard"); len(left) != 0 {
		t.Errorf("still remembered once all of it was gone: %+v", left)
	}
}
