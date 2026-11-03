package behaviour

import (
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func activityFrame(seq uint64, resources ...protocol.PerceivedResource) View {
	v := lootView(seq, resources...)
	v.State.T = float64(seq)
	v.Perception.T = float64(seq)
	v.Now = time.Unix(1000+int64(seq), 0)
	v.PerceptionBounds = protocol.PerceptionConfig{Radius: 40, MaxPerKind: 64}
	return v
}
func TestActivitySequencesAndDecay(t *testing.T) {
	for _, c := range []struct{ kind, prefab, want string }{{protocol.ResourceTree, "Beech1", "felling"}, {protocol.ResourceLog, "beech_log", "felling"}, {protocol.ResourceStump, "Beech_Stub", "felling"}, {protocol.ResourceRock, "rock4_copper", "mining"}, {protocol.ResourcePickable, "Pickable_Raspberry", "picking"}} {
		t.Run(c.prefab, func(t *testing.T) {
			tracker := ActivityTracker{}
			r := protocol.PerceivedResource{ID: "target", Kind: c.kind, Prefab: c.prefab, Pos: protocol.Vec3{12, 0, 0}}
			if a := tracker.Observe(activityFrame(1, r)); a != nil {
				t.Fatal("first sighting is not activity", a)
			}
			v := activityFrame(2)
			a := tracker.Observe(v)
			if a == nil || a.Kind != c.want || a.Prefab != c.prefab || a.Player != "Nimah" || a.Target != "target" {
				t.Fatal(a)
			}
			a.Kind = "changed"
			v.Now = v.Now.Add(time.Second)
			if a = tracker.Observe(v); a == nil || a.Kind != c.want || a.AgeS != 1 {
				t.Fatal("duplicate refreshed or mutable", a)
			}
			v.Now = v.Now.Add(ActivityTTL)
			v.StateAt = v.Now
			v.PerceptionAt = v.Now
			if a = tracker.Observe(v); a != nil {
				t.Fatal("activity did not decay", a)
			}
		})
	}
}
func TestActivityMixedEvidenceAndScopeBoundaries(t *testing.T) {
	tree := protocol.PerceivedResource{ID: "tree", Kind: protocol.ResourceTree, Prefab: "Beech1", Pos: protocol.Vec3{15, 0, 0}}
	ore := protocol.PerceivedResource{ID: "ore", Kind: protocol.ResourceRock, Prefab: "rock4_copper", Pos: protocol.Vec3{11, 0, 0}}
	tr := ActivityTracker{}
	tr.Observe(activityFrame(1, tree, ore))
	if a := tr.Observe(activityFrame(2)); a == nil || a.Kind != "mining" {
		t.Fatal(a)
	}
	for _, kind := range []string{"world", "connection", "anchor", "gap", "stale", "cap", "scan_edge", "jump", "old_frame", "unauthorized"} {
		t.Run(kind, func(t *testing.T) {
			tr := ActivityTracker{}
			tr.Observe(activityFrame(1, tree))
			v := activityFrame(2)
			switch kind {
			case "world":
				v.State.World = "elsewhere"
			case "connection":
				v.Connection = 1
			case "anchor":
				v.Anchor = "Other"
				v.Owners = append(v.Owners, "Other")
				v.Perception.Characters[0].Name = "Other"
			case "gap":
				v.Now = v.Now.Add(6 * time.Second)
			case "stale":
				v.PerceptionAt = v.Now.Add(-6 * time.Second)
			case "cap":
				v.PerceptionBounds.MaxPerKind = 1
				v.Perception.Resources = []protocol.PerceivedResource{{ID: "replacement", Kind: protocol.ResourceTree}}
			case "scan_edge":
				v.PerceptionBounds.Radius = 18
			case "jump":
				v.State.Pos = protocol.Vec3{100, 0, 0}
			case "old_frame":
				v.Perception.Seq = 0
			case "unauthorized":
				v.Owners = nil
			}
			if a := tr.Observe(v); a != nil {
				t.Fatal("invented activity across "+kind, a)
			}
		})
	}
}
func TestLootDiagnosticsMatchPickupAndExpire(t *testing.T) {
	l := &Loot{}
	v := lootView(1, target("eligible", protocol.ResourceDrop, "Wood", 5), target("deliberate", protocol.ResourceDrop, "Wood", 10), target("far", protocol.ResourceDrop, "Wood", 40), target("wrong", protocol.ResourceDrop, "Stone", 4))
	l.Observe(v)
	raw := []byte(`{}`)
	s := l.Inspect(raw, v, nil)
	if !s.Fresh || s.Anchor.Name != "Nimah" || s.Reach != 8 || s.Params.Radius != 30 || s.Settle == nil {
		t.Fatal(s)
	}
	want := map[string]string{"eligible": "eligible", "deliberate": "deliberate", "far": "out_of_reach", "wrong": "item_not_selected"}
	for _, d := range s.Drops {
		if d.Decision != want[d.ID] {
			t.Fatalf("%s: %s", d.ID, d.Decision)
		}
	}
	r := lootRunner(t, l, LootParams{})
	if p := r.pick(v); p == nil || p.ID != "eligible" {
		t.Fatal(p)
	}
	v.StateAt = v.Now.Add(-6 * time.Second)
	if s = l.Inspect(raw, v, r); s.Fresh || s.Anchor != nil || len(s.Drops) > 0 {
		t.Fatal("stale overlay", s)
	}
	v.StateAt = time.Time{}
	v.Connection = 2
	s = l.Inspect(raw, v, nil)
	if s.Settle.SettledS != 0 {
		t.Fatal("settle crossed connections")
	}
}

type activityObserver struct {
	MoveTo
	activity *Activity
}

func (o *activityObserver) Observe(v View) { o.activity = v.Activity }
func TestActivityIsAvailableToTemplatesAndDisabledLootStatus(t *testing.T) {
	f := setup(t)
	f.e.o.Now = time.Now
	observer := &activityObserver{}
	f.e.o.Templates["activity_observer"] = observer
	d := LootDefinition()
	d.Enabled = false
	f.put(t, LootInstance, d)
	send := func(gameT float64, resources ...protocol.PerceivedResource) {
		f.p.Send(&protocol.State{Envelope: protocol.Envelope{T: gameT}, Phase: protocol.PhaseInWorld, World: "test", WorldUID: "uid", HP: 25, HPMax: 25, WeightMax: 300})
		f.p.Send(&protocol.Perception{Envelope: protocol.Envelope{T: gameT}, Characters: []protocol.PerceivedCharacter{owner(10)}, Resources: resources})
		wait(t, func() bool { return f.s.Perception() != nil && f.s.Perception().T == gameT && f.s.State().T == gameT })
		f.tick()
	}
	send(10, protocol.PerceivedResource{ID: "tree", Kind: protocol.ResourceTree, Prefab: "Beech1", Pos: protocol.Vec3{12, 0, 0}})
	send(11)
	if observer.activity == nil || observer.activity.Kind != "felling" {
		t.Fatal("template did not receive inferred work", observer.activity)
	}
	rows := f.e.List()
	if len(rows) != 1 || rows[0].Activity == nil || rows[0].Activity.Prefab != "Beech1" || rows[0].Loot == nil || !rows[0].Loot.Fresh || rows[0].Definition.Enabled || rows[0].Running {
		t.Fatal(rows)
	}
	rows[0].Activity.Kind = "mutated"
	if f.e.List()[0].Activity.Kind != "felling" {
		t.Fatal("caller mutated retained activity")
	}
}
