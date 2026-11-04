package goal

import (
	"testing"

	"github.com/warricksothr/renki/internal/food"
	"github.com/warricksothr/renki/internal/protocol"
)

func TestFoodGoalsShareIdlePreferenceAndPersistEscortOverride(t *testing.T) {
	for _, escort := range []bool{false, true} {
		host, r, v, id := foodFixture(t)
		host.FoodPolicy = func() food.Policy { return food.Policy{ActivityAware: true, IdleMaxTier: 2} }
		v.State.Food.Carried = []protocol.CarriedFood{{Prefab: "CookedMeat", Health: 30, CanEat: true}, {Prefab: "LoxPie", Health: 75, CanEat: true}}
		if escort {
			if err := host.Store.update(id, func(g *Record) error { g.Food.Escort = &protocol.EscortArgs{Player: "Nimah"}; return nil }); err != nil {
				t.Fatal(err)
			}
		}
		o, err := r.Next(v, nil)
		want := "CookedMeat"
		if escort {
			want = "LoxPie"
		}
		if err != nil || o == nil || o.Task != protocol.TaskUseItem || o.Args.(*protocol.UseItemArgs).Item != want {
			t.Fatal(escort, o, err)
		}
		g, _ := host.Store.Get(id)
		if g.Food.Decision == nil || g.Food.Decision.Item != want {
			t.Fatal(g.Food)
		}
	}
}
