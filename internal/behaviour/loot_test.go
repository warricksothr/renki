package behaviour

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestDeliberateDropRule(t *testing.T) {
	at := time.Unix(1000, 0)
	drop := protocol.Vec3{0, 0, 0}
	owner := []protocol.Vec3{{1.5, 0, 0}}
	for _, c := range []struct {
		name   string
		owners []protocol.Vec3
		breaks []Break
		want   bool
	}{
		{"at an owner's feet, nothing felled", owner, nil, true},
		{"no owner within 2 m", []protocol.Vec3{{2.5, 0, 0}}, nil, false},
		{"no owner at all", nil, nil, false},
		{"a log split nearby just now", owner, []Break{{Pos: protocol.Vec3{8, 0, 0}, At: at}}, false},
		{"a tree fell nearby 10 s ago", owner, []Break{{Pos: protocol.Vec3{0, 0, 9}, At: at.Add(-FellingWindow)}}, false},
		{"the break was 11 s ago", owner, []Break{{Pos: protocol.Vec3{8, 0, 0}, At: at.Add(-11 * time.Second)}}, true},
		{"the break was 11 m away", owner, []Break{{Pos: protocol.Vec3{11, 0, 0}, At: at}}, true},
		{"the break came after the drop", owner, []Break{{Pos: protocol.Vec3{1, 0, 0}, At: at.Add(time.Second)}}, true},
	} {
		if got := DeliberateDrop(drop, at, c.owners, c.breaks); got != c.want {
			t.Errorf("%s: got %v", c.name, got)
		}
	}
}

func owner(x float64) protocol.PerceivedCharacter {
	return protocol.PerceivedCharacter{Kind: protocol.CharacterPlayer, Name: "Nimah", HPFrac: 1, Pos: protocol.Vec3{x, 0, 0}}
}
func lootView(seq uint64, resources ...protocol.PerceivedResource) View {
	v := haulView()
	v.Idle = true
	v.Owners = []string{"Nimah"}
	v.Inventory = pack(10)
	v.Perception = &protocol.Perception{Envelope: protocol.Envelope{Seq: seq, T: 10}, Characters: []protocol.PerceivedCharacter{owner(10)}, Resources: resources}
	return v
}
func lootRunner(t *testing.T, l *Loot, p LootParams) *lootRun {
	t.Helper()
	raw, _ := json.Marshal(p)
	r, err := l.Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	return r.(*lootRun)
}

func TestLootPicksNearestDropsNearTheOwnerAndChecksReceipts(t *testing.T) {
	l := &Loot{}
	stone := target("stone", protocol.ResourceDrop, "Stone", 6)
	far := target("far", protocol.ResourceDrop, "Wood", 45) // 35 m from the owner
	v := lootView(1, target("wood", protocol.ResourceDrop, "Wood", 5), target("resin", protocol.ResourceDrop, "Resin", 20), stone, far)
	l.Observe(v)
	if !l.Ready(json.RawMessage(`{}`), v) {
		t.Fatal("eligible drops should make loot ready")
	}
	r := lootRunner(t, l, LootParams{})
	o := nextTask(t, r, v, nil, protocol.TaskPickup)
	if o.Args.(*protocol.PickupArgs).Entity != "wood" {
		t.Fatal("nearest first", o.Args)
	}
	// The task's success is not a receipt: wait for a newer, larger pack.
	if _, e := r.Next(v, success("p1")); !errors.Is(e, ErrWait) {
		t.Fatal(e)
	}
	v.Inventory = pack(11, material("Wood", 3))
	o = nextTask(t, r, v, success("p1"), protocol.TaskPickup)
	if o.Args.(*protocol.PickupArgs).Entity != "resin" {
		t.Fatal(o.Args)
	}
	// A newer pack without the item fails the run rather than counting it.
	v.Inventory = pack(12, material("Wood", 3))
	if _, e := r.Next(v, success("p2")); e == nil || !strings.Contains(e.Error(), "Resin") {
		t.Fatal(e)
	}
}

func TestLootSkipsTakenDropsAndStopsWhenNothingIsLeft(t *testing.T) {
	l := &Loot{}
	v := lootView(1, target("wood", protocol.ResourceDrop, "Wood", 5))
	l.Observe(v)
	r := lootRunner(t, l, LootParams{})
	nextTask(t, r, v, nil, protocol.TaskPickup)
	gone := Result{Status: protocol.StatusRejected, Reason: protocol.ReasonInvalidTarget}
	if !r.Recover(gone) {
		t.Fatal("a drop someone else took is not a failure")
	}
	if o, e := r.Next(v, &gone); o != nil || e != nil {
		t.Fatal(o, e)
	}
	if r.Recover(Result{Status: protocol.StatusFailed, Reason: protocol.ReasonNotAllowed}) {
		t.Fatal("permission failures end the run")
	}
}

func TestLootNeedsAnOwnerCapacityAndAnObservedDrop(t *testing.T) {
	l := &Loot{}
	v := lootView(1, target("wood", protocol.ResourceDrop, "Wood", 5))
	if l.Ready(json.RawMessage(`{}`), v) {
		t.Fatal("a drop the tracker has not seen has no known origin")
	}
	l.Observe(v)
	for name, edit := range map[string]func(*View){
		"no owner":     func(v *View) { v.Perception.Characters = nil },
		"dead owner":   func(v *View) { v.Perception.Characters[0].HPFrac = 0 },
		"not an owner": func(v *View) { v.Owners = nil },
		"busy":         func(v *View) { v.Idle = false },
		"stale":        func(v *View) { v.Perception.T = 1 },
		"full":         func(v *View) { v.State.Weight = 240 },
	} {
		w := lootView(1, target("wood", protocol.ResourceDrop, "Wood", 5))
		edit(&w)
		if l.Ready(json.RawMessage(`{}`), w) {
			t.Error(name)
		}
	}
	if !l.Ready(json.RawMessage(`{}`), v) {
		t.Fatal("expected ready")
	}
}

func TestLootLeavesADeliberateDropForFiveMinutes(t *testing.T) {
	l := &Loot{}
	v := lootView(1)
	l.Observe(v)
	// The owner puts wood down at their feet; nothing was felled near it.
	v = lootView(2, target("gift", protocol.ResourceDrop, "Wood", 11))
	l.Observe(v)
	start := v.Now
	for _, after := range []time.Duration{0, time.Minute, DeliberateIgnore - time.Second} {
		v.Now = start.Add(after)
		v.Perception.Seq++
		l.Observe(v)
		if l.Ready(json.RawMessage(`{}`), v) {
			t.Fatal("deliberate drop picked up after", after)
		}
		if o, e := lootRunner(t, l, LootParams{}).Next(v, nil); o != nil || e != nil {
			t.Fatal(o, e)
		}
	}
	v.Now = start.Add(DeliberateIgnore)
	v.Perception.Seq++
	l.Observe(v)
	o := nextTask(t, lootRunner(t, l, LootParams{}), v, nil, protocol.TaskPickup)
	if o.Args.(*protocol.PickupArgs).Entity != "gift" {
		t.Fatal(o.Args)
	}
}

func TestLootTakesDropsFromFellingAtTheOwnersFeet(t *testing.T) {
	l := &Loot{}
	log := protocol.PerceivedResource{ID: "log", Kind: protocol.ResourceLog, Pos: protocol.Vec3{14, 0, 0}}
	l.Observe(lootView(1, log))
	// The owner splits the log and its wood lands at their feet.
	v := lootView(2, target("split", protocol.ResourceDrop, "Wood", 11))
	l.Observe(v)
	if !l.Ready(json.RawMessage(`{}`), v) {
		t.Fatal("felling products are not deliberate drops")
	}
	// After the felling window, a new drop at the owner's feet is deliberate.
	v = lootView(3, target("split", protocol.ResourceDrop, "Wood", 11), target("later", protocol.ResourceDrop, "Resin", 10.5))
	v.Now = v.Now.Add(FellingWindow + time.Second)
	l.Observe(v)
	o := nextTask(t, lootRunner(t, l, LootParams{}), v, nil, protocol.TaskPickup)
	if o.Args.(*protocol.PickupArgs).Entity != "split" || l.eligible("later", v.Now) {
		t.Fatal(o.Args)
	}
}

func TestLootValidation(t *testing.T) {
	l := &Loot{}
	for _, bad := range []string{`{"radius":41}`, `{"radius":-1}`, `{"items":["wood"]}`, `{"items":[]}`, `{"maxWeightFraction":0.95}`, `{"maxTargets":65}`, `{"preemptFollow":true}`, `{"site":{}}`} {
		if l.Validate(json.RawMessage(bad)) == nil {
			t.Error(bad)
		}
	}
	d := LootDefinition()
	if err := validate(d, map[string]Template{"loot": l}); err != nil || !d.Enabled {
		t.Fatal(err, d)
	}
}

func TestBuiltInLootIsInstalledOnceOnAndCanBeSwitchedOff(t *testing.T) {
	f := setup(t)
	if err := f.e.InstallLoot("builtin"); err != nil {
		t.Fatal(err)
	}
	if list := f.e.List(); len(list) != 1 || list[0].Name != LootInstance || !list[0].Definition.Enabled || list[0].Definition.Template != "loot" {
		t.Fatal(list)
	}
	f.act(t, LootInstance, "disable")
	if err := f.e.InstallLoot("builtin"); err != nil || f.e.List()[0].Definition.Enabled {
		t.Fatal("installing again must not switch it back on", err)
	}
	reopened := New(Options{Session: f.s, Store: f.store, Now: func() time.Time { return f.now }})
	if err := reopened.InstallLoot("builtin"); err != nil || len(reopened.List()) != 1 || reopened.List()[0].Definition.Enabled {
		t.Fatal("the off switch must survive a restart", reopened.List())
	}
}

func TestEngineLootPicksUpADropNearTheOwner(t *testing.T) {
	f := setup(t)
	if err := f.e.InstallLoot("builtin"); err != nil {
		t.Fatal(err)
	}
	f.p.Send(pack(1))
	f.p.Send(&protocol.Perception{Envelope: protocol.Envelope{Seq: 1}, Characters: []protocol.PerceivedCharacter{owner(10)},
		Resources: []protocol.PerceivedResource{target("wood", protocol.ResourceDrop, "Wood", 5)}})
	// Perception is fresh only when the state is no older than it.
	f.p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "test", HP: 25, HPMax: 25, WeightMax: 300})
	wait(t, func() bool { v := f.e.view(); return v.Inventory != nil && freshPerception(v) })
	f.now = f.now.Add(3 * time.Second)
	f.tick()
	req := f.p.NextTask()
	if req.Task != protocol.TaskPickup {
		t.Fatal(req)
	}
	args, err := req.DecodeArgs()
	if err != nil || args.(*protocol.PickupArgs).Entity != "wood" {
		t.Fatal(args, err)
	}
}
