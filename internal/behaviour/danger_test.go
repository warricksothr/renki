package behaviour

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

func TestGatherPrefersSaferTreeAndDecayRestoresNearestChoice(t *testing.T) {
	for _, age := range []time.Duration{0, 6 * 24 * time.Hour} {
		r, v := gathering(t)
		near := gatherResource("near", protocol.ResourceTree, 3)
		far := gatherResource("safe", protocol.ResourceTree, 16)
		v.Perception.Resources = []protocol.PerceivedResource{near, far}
		store := home.Memory()
		if err := store.RememberHarms([]home.Harm{{World: "test", At: v.Now.Add(-age), Pos: near.Pos, Event: protocol.EventDamaged, Amount: 10}}, v.Now); err != nil {
			t.Fatal(err)
		}
		v.Danger = store.DangerZones("test", v.Now)
		order := nextTask(t, r, v, nil, protocol.TaskHarvest)
		want := "safe"
		if age > 0 {
			want = "near"
		}
		if got := order.Args.(*protocol.HarvestArgs).Entity; got != want {
			t.Fatal(age, got, want)
		}
	}
}

func TestCollectPrefersSaferPickableAndEngineScopesDanger(t *testing.T) {
	raw, _ := json.Marshal(collectParams())
	r, err := (Collect{}).Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	v := haulView()
	v.Inventory = pack(10)
	v.Perception.Resources = []protocol.PerceivedResource{target("near", protocol.ResourcePickable, "Flint", 3), target("safe", protocol.ResourcePickable, "Flint", 15)}
	v.Danger = []home.DangerZone{{Pos: protocol.Vec3{3, 0, 0}, Radius: 10, Score: 1}}
	o := nextTask(t, r, v, nil, protocol.TaskInteract)
	if o.Args.(*protocol.InteractArgs).Entity != "safe" {
		t.Fatal(o)
	}
	f := setup(t)
	f.e.o.Danger = func(world string, now time.Time) []home.DangerZone {
		if world != "test" || !now.Equal(f.now) {
			t.Fatal(world, now)
		}
		return v.Danger
	}
	if len(f.e.view().Danger) != 1 {
		t.Fatal("engine omitted danger memory")
	}
}
