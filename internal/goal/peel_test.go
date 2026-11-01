package goal

import (
	"encoding/json"
	"testing"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestPeelGoalDispatchesNativeEpisodeWithoutDirectAttackBypass(t *testing.T) {
	host, r, v, id := fixture(t)
	if err := host.Store.update(id, func(g *Record) error { g.Peel = true; return nil }); err != nil {
		t.Fatal(err)
	}
	v.Perception.Characters[0].TargetingMe = true
	v.Perception.Characters[0].Pos = protocol.Vec3{3, 0, 0}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskAttack {
		t.Fatal(o, err)
	}
	a := o.Args.(*protocol.AttackArgs)
	if !a.Peel || !a.Kite || a.Entity != "grey" {
		t.Fatal(a)
	}
	if err := r.Dispatched("peel-task"); err != nil {
		t.Fatal(err)
	}
	// Uncertain peel intent survives restart protection just like any attack.
	g, _ := host.Store.Get(id)
	var p map[string]any
	_ = json.Unmarshal(g.Pending.Params, &p)
	if p["peel"] != true || p["direct"] == true {
		t.Fatal(p)
	}
	if err := host.Queue(id); err == nil {
		t.Fatal("replayed an uncertain peel")
	}
}

func TestPeelDiscoveryPreservesModeAcrossRegionalSearch(t *testing.T) {
	g := Record{World: "one", Radius: 200, Peel: true}
	_, raw, _ := nextArea(g, protocol.Vec3{})
	var params map[string]any
	if err := json.Unmarshal(raw, &params); err != nil || params["peel"] != true {
		t.Fatal(string(raw), err)
	}
}
