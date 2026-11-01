package goal

import (
	"encoding/json"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func preyIngredient(prefab string) string {
	return map[string]string{"Boar": "RawMeat", "Neck": "NeckTail"}[prefab]
}

func stockHuntSource(g Record, pos protocol.Vec3) bool {
	s := g.Food.Stock
	return s == nil || s.AllowHunting && (len(s.HuntAreas) == 0 || slices.ContainsFunc(s.HuntingPinned, func(w home.Waypoint) bool { return stockWithin(pos, w) }))
}

func stockResourceSource(g Record, p protocol.PerceivedResource) bool {
	return stockSource(g, p.Pos) || p.Yield != nil && slices.Contains(ingredients, p.Yield.Prefab) && stockHuntSource(g, p.Pos)
}

// Saved outcomes rank another area; none proves current wildlife or clearance.
func (r *runner) foodAreaCooling(g Record, name string, now time.Time) bool {
	for _, old := range r.t.Store.List() {
		if old.ID == g.ID || old.World != g.World || old.Food == nil {
			continue
		}
		if a := old.Food.Acquisition; a != nil {
			for _, visit := range a.AreaVisits {
				if visit.Name == name && !now.Before(visit.At) && now.Sub(visit.At) < 15*time.Minute {
					return true
				}
			}
		}
		for _, result := range old.History {
			if result.Template == "search_food" && result.Entity == "marker:"+name && !result.At.IsZero() && !now.Before(result.At) && now.Sub(result.At) < 15*time.Minute {
				return true
			}
		}
		if p := old.Pending; p != nil && p.Template == "search_food" && p.Entity == "marker:"+name && !now.Before(p.At) && now.Sub(p.At) < 15*time.Minute {
			return true
		}
	}
	return false
}

func (r *runner) foodPrey(g Record, v behaviour.View, items []string) []protocol.PerceivedCharacter {
	a := g.Food.Acquisition
	if a == nil || g.Food.TravelBudget != nil && !foodTravelDamage(g, v) || v.InjurySeq == nil || v.Injury == nil || !v.Injury.Complete || v.Injury.Sequence != *v.InjurySeq || !finite(v.Injury.Damage) || v.Injury.Damage < 0 || a.Hunts >= 2 || r.t.Allowed == nil || !r.t.Allowed(protocol.TaskAttack) || !nourished(v.State) || v.State.HP < math.Max(30, v.State.HPMax*.8) || v.State.Stamina < 40 || v.Inventory.Free() < 3 {
		return nil
	}
	if !slices.Contains(items, "RawMeat") && !slices.Contains(items, "NeckTail") {
		return nil
	}
	if g.Food.Stock != nil && !g.Food.Stock.AllowHunting {
		return nil
	}
	var prey []protocol.PerceivedCharacter
	for _, c := range v.Perception.Characters {
		if c.ID == "" || c.Kind != protocol.CharacterCreature || !c.Hostile || c.Stars != 0 || c.HPFrac <= 0 || !finite(c.HPFrac) ||
			!slices.Contains(items, preyIngredient(c.Prefab)) || slices.Contains(a.Tried, c.ID) || distance(c.Pos, v.State.Pos) > 30 ||
			distance(c.Pos, g.Center) > acquisitionRadius(g) || !stockHuntSource(g, c.Pos) || !foodSiteSafe(v, c.Pos, c.ID) {
			continue
		}
		prey = append(prey, c)
	}
	sort.Slice(prey, func(i, j int) bool {
		di, dj := distance(v.State.Pos, prey[i].Pos), distance(v.State.Pos, prey[j].Pos)
		if di != dj {
			return di < dj
		}
		return prey[i].ID < prey[j].ID
	})
	return prey
}

func (r *runner) foodHuntNext(g Record, v behaviour.View, items []string) (*session.Order, error) {
	a := g.Food.Acquisition
	prey := r.foodPrey(g, v, items)
	if len(prey) > 0 {
		c := prey[0]
		return r.foodDispatch(g, v, "hunt_food", c.Prefab, c.ID, &session.Order{Task: protocol.TaskAttack, Args: &protocol.AttackArgs{Entity: c.ID, Style: "melee", Kite: true}, TimeoutS: 180}, func(g *Record) {
			g.Food.Acquisition.Hunts++
			g.Food.Acquisition.Tried = append(g.Food.Acquisition.Tried, c.ID)
		})
	}
	if s := g.Food.Stock; s != nil && s.AllowHunting && a != nil && a.Hunts < 2 && len(a.Visits) < 6 && (slices.Contains(items, "RawMeat") || slices.Contains(items, "NeckTail")) && r.t.Allowed != nil && r.t.Allowed(protocol.TaskAttack) {
		return r.foodSearchAreas(g, v, s.HuntingPinned)
	}
	return nil, nil
}

// Pantry pressure is evaluated from fresh storage counts before it can
// authorize a hunt. Personal bootstrap hunts remain part of the raw phase.
func foodHuntItems(g Record, v behaviour.View) []string {
	if s := g.Food.Stock; s != nil {
		if !s.Replenishing || s.Known == nil || mealCount(v.Inventory) < g.Food.Reserve || !foodAtHome(v.State.Pos, g.Center) && s.Delivering {
			return nil
		}
		items := []string{}
		surplus := stockSurplus(g, v)
		for _, ingredient := range []string{"RawMeat", "NeckTail"} {
			if s.Targets[cooked[ingredient]] > s.Known[cooked[ingredient]]+surplus[cooked[ingredient]] && v.Inventory.Count(ingredient) == 0 {
				items = append(items, ingredient)
			}
		}
		return items
	}
	if g.Food.Phase == "raw" && mealCount(v.Inventory) < g.Food.Reserve {
		return []string{"RawMeat", "NeckTail"}
	}
	return nil
}

func (r *runner) foodHuntTask(g Record) bool {
	p := g.Pending
	if p == nil || p.Template != "hunt_food" || p.Run != r.t.Store.runID || p.TaskID == "" || p.Entity == "" || r.t.Allowed == nil || !r.t.Allowed(protocol.TaskAttack) || g.Food.Stock != nil && !g.Food.Stock.AllowHunting {
		return false
	}
	var args protocol.AttackArgs
	return json.Unmarshal(p.Params, &args) == nil && args.Entity == p.Entity && args.Style == "melee" && args.Kite && preyIngredient(g.Food.Item) != ""
}

func (r *runner) nativeFoodHunt(g Record, v behaviour.View) bool {
	if !r.foodHuntTask(g) || v.State.Task == nil || v.State.Combat == nil {
		return false
	}
	p, t, c := g.Pending, v.State.Task, v.State.Combat
	return t.ID == p.TaskID && t.Task == protocol.TaskAttack && t.Status == protocol.StatusRunning && c.ID == p.TaskID && c.Entity == p.Entity && !c.Defensive &&
		slices.Contains([]string{"approach", "closing", "strike", "wait", "block", "dodge", "retreat"}, c.Mode)
}

// The native death receipt can precede the next perception frame. Only the
// exact confirmed target may settle during the already charged two-second
// loot wait; it grants no further action or exemption for another creature.
func (r *runner) foodKillSettling(g Record, v behaviour.View, entity string) bool {
	last := g.Last
	if last == nil || last.Template != "hunt_food" || last.Status != protocol.StatusSucceeded ||
		last.Run != r.t.Store.runID || last.TaskID == "" || last.Entity != entity || entity == "" ||
		last.At.IsZero() || v.Now.Before(last.At) || !v.Now.Before(last.At.Add(2*time.Second)) || !g.Wake.After(v.Now) {
		return false
	}
	return slices.ContainsFunc(g.Food.Receipts, func(receipt FoodReceipt) bool {
		return receipt.Action == "hunt" && receipt.Run == last.Run && receipt.TaskID == last.TaskID && receipt.Entity == entity
	})
}

// Visit each area before sampling its interior. Six total visits bound the
// episode; marker radius never becomes evidence that the whole area was seen.
func (r *runner) foodSearchAreas(g Record, v behaviour.View, areas []home.Waypoint) (*session.Order, error) {
	a := g.Food.Acquisition
	if a == nil || len(a.Visits) >= 6 {
		return nil, nil
	}
	type site struct {
		marker home.Waypoint
		pos    protocol.Vec3
		pass   int
	}
	sites := []site{}
	for _, w := range areas {
		if r.foodAreaCooling(g, w.Name, v.Now) {
			continue
		}
		sites = append(sites, site{w, w.Pos, 0})
		if w.Radius >= 8 {
			for i, sign := range []float64{-1, 1} {
				p := w.Pos
				p[0] += sign * w.Radius * .6
				sites = append(sites, site{w, p, i + 1})
			}
		}
	}
	sort.SliceStable(sites, func(i, j int) bool {
		if sites[i].pass != sites[j].pass {
			return sites[i].pass < sites[j].pass
		}
		return foodTravelCost(v, sites[i].pos) < foodTravelCost(v, sites[j].pos)
	})
	for _, site := range sites {
		pos := site.pos
		key := foodSiteKey(pos)
		if slices.Contains(a.Visits, key) || distance(v.State.Pos, pos) <= 3 || distance(pos, g.Center) > acquisitionRadius(g) || !foodSiteSafe(v, pos, "") || home.DangerRisk(v.Danger, pos) >= 1 {
			continue
		}
		return r.foodDispatch(g, v, "search_food", "", "marker:"+site.marker.Name, &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}, Dry: true, AvoidHostiles: true}, TimeoutS: 180}, func(g *Record) { g.Food.Acquisition.Visits = append(g.Food.Acquisition.Visits, key) })
	}
	return nil, nil
}
