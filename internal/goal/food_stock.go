package goal

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// FoodStock is an optional household objective above the personal reserve.
// Storage IDs are explicit; observations expire and never come from map hints.
type FoodStock struct {
	ReturnAfterCooking bool            `json:"returnAfterCooking,omitempty"`
	Targets            map[string]int  `json:"targets"`
	Sources            []string        `json:"sources,omitempty"`
	Pinned             []home.Waypoint `json:"pinned,omitempty"`
	Delivering         bool            `json:"delivering,omitempty"`
	Decision           string          `json:"decision,omitempty"`
	Known              map[string]int  `json:"known,omitempty"`
	Shortage           map[string]int  `json:"shortage,omitempty"`
}

func validateStock(g Record) error {
	s := g.Food.Stock
	if s == nil {
		return nil
	}
	if len(s.Targets) == 0 || len(s.Targets) > 8 || len(s.Sources) > 4 || len(s.Pinned) > 4 || len(g.Food.Storage) == 0 || len(g.Food.Storage) > 16 || g.Food.Escort != nil {
		return errors.New("household stock needs targets, explicit storage and no escort continuation")
	}
	total := 0
	for p, n := range s.Targets {
		if !slices.Contains(meals, p) || n < 1 || n > 20 {
			return errors.New("stock targets must be supported meals with counts1–20")
		}
		total += n
	}
	if total > 40 {
		return errors.New("household episode target limit is40 meals")
	}
	seen := map[string]bool{}
	for _, id := range g.Food.Storage {
		if id == "" || seen[id] {
			return errors.New("stock storage IDs must be unique")
		}
		seen[id] = true
	}
	seen = map[string]bool{}
	for _, name := range s.Sources {
		canonical, err := home.WaypointName(name)
		if err != nil || canonical != name || seen[name] {
			return errors.New("stock sources must be unique canonical marker names")
		}
		seen[name] = true
	}
	if len(s.Pinned) > 0 && len(s.Pinned) != len(s.Sources) {
		return errors.New("incomplete pinned food sources")
	}
	for i, w := range s.Pinned {
		if w.Name != s.Sources[i] || w.Kind != "berries" || !validFoodPos(w.Pos) || distance(w.Pos, g.Center) > acquisitionRadius(g) || !finite(w.Radius) || w.Radius < 0 || w.Radius > 40 {
			return errors.New("invalid pinned household source")
		}
	}
	for _, counts := range []map[string]int{s.Known, s.Shortage} {
		if len(counts) > 8 {
			return errors.New("invalid stock summary")
		}
		for p, n := range counts {
			if s.Targets[p] == 0 || n < 0 || n > 1000000 {
				return errors.New("invalid stock summary count")
			}
		}
	}
	if len(s.Decision) > 256 {
		return errors.New("stock decision too long")
	}
	return nil
}

func stockScope(g Record, v behaviour.View) error {
	s := g.Food.Stock
	if s == nil {
		return nil
	}
	for _, p := range s.Pinned {
		if !slices.Contains(v.Waypoints, p) {
			return errors.New("household source marker changed or removed")
		}
	}
	return nil
}
func stockWithin(pos protocol.Vec3, w home.Waypoint) bool {
	return math.Abs(pos[1]-w.Pos[1]) <= 4 && pos.FlatDistance(w.Pos) <= math.Max(3, w.Radius)
}
func stockSource(g Record, pos protocol.Vec3) bool {
	s := g.Food.Stock
	return s == nil || len(s.Sources) == 0 || slices.ContainsFunc(s.Pinned, func(w home.Waypoint) bool { return stockWithin(pos, w) })
}

// Keep personal meals first, favouring food outside the requested stock mix.
func stockSurplus(g Record, v behaviour.View) map[string]int {
	out := map[string]int{}
	order := slices.Clone(meals)
	sort.SliceStable(order, func(i, j int) bool { return g.Food.Stock.Targets[order[i]] == 0 && g.Food.Stock.Targets[order[j]] != 0 })
	keep := g.Food.Reserve
	for _, p := range order {
		n := v.Inventory.Count(p)
		held := min(keep, n)
		keep -= held
		out[p] = n - held
	}
	return out
}
func stockCounts(g Record, now time.Time, run string) (map[string]int, *FoodChest, error) {
	known := map[string]int{}
	for p := range g.Food.Stock.Targets {
		known[p] = 0
	}
	for _, id := range g.Food.Storage {
		i := slices.IndexFunc(g.Food.Chests, func(c FoodChest) bool { return c.Entity == id })
		if i < 0 {
			return known, nil, fmt.Errorf("household chest %s was not freshly perceived at home", id)
		}
		c := g.Food.Chests[i]
		if c.Unavailable {
			return known, nil, fmt.Errorf("household chest %s is unavailable; stock is unknown", id)
		}
		if !c.Observed || c.Run != run || c.ObservedAt.IsZero() || now.Before(c.ObservedAt) || now.Sub(c.ObservedAt) > 3*time.Minute {
			return known, &c, nil
		}
		for p := range known {
			known[p] += c.Contents[p]
		}
	}
	return known, nil, nil
}
func stockDeficits(g Record, known map[string]int) map[string]int {
	out := map[string]int{}
	for p, n := range g.Food.Stock.Targets {
		if n > known[p] {
			out[p] = n - known[p]
		}
	}
	return out
}
func (r *runner) stockDecision(g Record, known, shortage map[string]int, why string, delivery bool) error {
	return r.t.Store.update(g.ID, func(g *Record) error {
		s := g.Food.Stock
		s.Known, s.Shortage, s.Decision, s.Delivering = known, shortage, why, delivery
		return nil
	})
}
func (r *runner) stockReturn(g Record, v behaviour.View, known, shortage map[string]int, why string) (*session.Order, error) {
	if err := r.stockDecision(g, known, shortage, why, true); err != nil {
		return nil, err
	}
	g, _ = r.t.Store.Get(g.ID)
	return r.foodDispatch(g, v, "food_return", "", "", &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: g.Home}, Dry: true, AvoidHostiles: true}, TimeoutS: 180})
}

func (r *runner) stockNext(g Record, v behaviour.View) (*session.Order, error) {
	f, s := g.Food, g.Food.Stock
	if !f.SearchPinned {
		if !foodAtHome(v.State.Pos, g.Center) {
			return r.stockReturn(g, v, nil, nil, "return home before household stock inspection")
		}
		if err := r.pinFoodSearch(g, v); err != nil {
			return r.need(err.Error())
		}
		g, _ = r.t.Store.Get(g.ID)
		f, s = g.Food, g.Food.Stock
	}

	known, unknown, err := stockCounts(g, v.Now, r.t.Store.runID)
	if err != nil {
		return r.need(err.Error())
	}
	shortage := stockDeficits(g, known)
	if unknown != nil {
		if !foodAtHome(v.State.Pos, g.Center) {
			return r.stockReturn(g, v, known, shortage, "return for a fresh household stock inspection")
		}
		// Normal access/ownership and a one-meal request earn the chest snapshot.
		// A withdrawn meal becomes pack cargo and excess is deposited normally.
		live := slices.ContainsFunc(v.Perception.Structures, func(p protocol.PerceivedStructure) bool {
			return p.ID == unknown.Entity && p.Kind == protocol.StructureContainer && distance(p.Pos, unknown.Pos) < .5
		})
		if !live {
			return r.need("household chest is no longer at its observed location")
		}
		if err := r.stockDecision(g, known, shortage, "inspect unknown household chest", true); err != nil {
			return nil, err
		}
		return r.foodDispatch(g, v, "stock_probe", meals[0], unknown.Entity, &session.Order{Task: protocol.TaskWithdraw, Args: &protocol.TransferArgs{Container: unknown.Entity, Items: []protocol.TransferItem{{Prefab: meals[0], Count: 1}}}, TimeoutS: 45})
	}
	if len(shortage) == 0 {
		if !foodAtHome(v.State.Pos, g.Center) {
			return r.stockReturn(g, v, known, shortage, "household targets met; return to home floor")
		}
		d, err := Evaluate(g.Policy, map[string]any{"objectiveSatisfied": true}, g.State, g.Last)
		if err != nil || d.Action != "done" {
			return r.need("food policy refused household readiness")
		}
		return nil, r.t.Store.update(g.ID, func(g *Record) error {
			g.Status = "succeeded"
			g.Food.Phase = "complete"
			g.Food.Stock.Known = known
			g.Food.Stock.Shortage = shortage
			g.Food.Stock.Decision = "household targets met"
			g.Reason = "verified household storage targets, personal reserve and home-floor arrival"
			return nil
		})
	}
	surplus := stockSurplus(g, v)
	atHome := foodAtHome(v.State.Pos, g.Center)
	// Deliver at home whenever cargo can close a measured shortage.
	for _, p := range meals {
		if shortage[p] == 0 || surplus[p] == 0 {
			continue
		}
		if !atHome {
			continue
		}
		for _, c := range f.Chests {
			if slices.Contains(f.Tried, "stock_deposit:"+c.Entity+":"+p) {
				continue
			}
			live := slices.ContainsFunc(v.Perception.Structures, func(x protocol.PerceivedStructure) bool { return x.ID == c.Entity && distance(x.Pos, c.Pos) < .5 })
			if !live {
				continue
			}
			if err := r.stockDecision(g, known, shortage, "deposit carried food toward household shortage", true); err != nil {
				return nil, err
			}
			return r.foodDispatch(g, v, "stock_deposit", p, c.Entity, &session.Order{Task: protocol.TaskDeposit, Args: &protocol.TransferArgs{Container: c.Entity, Items: []protocol.TransferItem{{Prefab: p, Count: protocol.ItemCount(min(shortage[p], surplus[p]))}}}, TimeoutS: 45})
		}
		return r.need("household destinations have no confirmed room for carried food")
	}
	loaded := v.Inventory.Free() < 3 || v.State.WeightMax <= 0 || v.State.Weight >= v.State.WeightMax*.7
	enough := true
	cargo := 0
	raw := false
	for p, n := range shortage {
		cargo += surplus[p]
		if surplus[p] < n {
			enough = false
		}
		for ingredient, output := range cooked {
			if output == p && v.Inventory.Count(ingredient) > 0 {
				raw = true
			}
		}
	}
	risk := home.DangerRisk(v.Danger, v.State.Pos)
	if !atHome && (s.Delivering || enough || loaded || raw || !nourished(v.State) || risk >= .5 && cargo > 0 || cargo > 0 && distance(v.State.Pos, g.Center) > 90) {
		return r.stockReturn(g, v, known, shortage, "committed delivery: cargo, capacity, food, route distance or danger pressure")
	}
	// Home cooking uses existing native ownership, fire and output receipts.
	if atHome && raw {
		if f.Batches >= 4 {
			return r.need("household cooking batch budget exhausted")
		}
		station, hot := foodStation(v)
		if station == "" {
			return r.need("household food needs an available supported cooking station")
		}
		if !hot && v.Inventory.Count("Wood") == 0 {
			if o, item, chest := foodWithdrawal(g, v, []string{"Wood"}, 1); o != nil {
				return r.foodDispatch(g, v, "wood", item, chest, o)
			}
			return r.need("household cooking needs accessible fuel")
		}
		for _, ingredient := range ingredients {
			if shortage[cooked[ingredient]] > 0 && v.Inventory.Count(ingredient) > 0 {
				return r.foodDispatch(g, v, "cook", ingredient, station, &session.Order{Task: protocol.TaskInteract, Args: &protocol.InteractArgs{Entity: station, Cook: ingredient}, TimeoutS: 180})
			}
		}
	}
	if loaded {
		return r.need("household work needs free carry capacity; unrelated cargo requires storage")
	}
	// Ingredients already in known home chests are cheaper than another trip.
	wanted := []string{}
	for _, ingredient := range ingredients {
		if shortage[cooked[ingredient]] > surplus[cooked[ingredient]] {
			wanted = append(wanted, ingredient)
		}
	}
	if atHome && len(wanted) > 0 {
		if o, item, chest := foodWithdrawal(g, v, wanted, 4); o != nil {
			return r.foodDispatch(g, v, "raw", item, chest, o)
		}
	}
	items := []string{}
	for _, p := range meals {
		if shortage[p] > surplus[p] {
			items = append(items, p)
		}
	}
	items = append(items, wanted...)
	if len(items) == 0 {
		return r.need("household cargo has no usable destination")
	}
	if err := r.stockDecision(g, known, shortage, "collect unmet food targets; prefer nearby safe sources", false); err != nil {
		return nil, err
	}
	g, _ = r.t.Store.Get(g.ID)
	return r.foodAcquire(g, v, items)
}

func (r *runner) pinStockSources(g Record, v behaviour.View) error {
	s := g.Food.Stock
	if s == nil || len(s.Sources) == 0 || len(s.Pinned) > 0 {
		return nil
	}
	pinned := []home.Waypoint{}
	for _, name := range s.Sources {
		i := slices.IndexFunc(v.Waypoints, func(w home.Waypoint) bool { return w.Name == name && w.Kind == "berries" })
		if i < 0 {
			return errors.New("household source marker is missing: " + name)
		}
		pinned = append(pinned, v.Waypoints[i])
	}
	return r.t.Store.update(g.ID, func(g *Record) error { g.Food.Stock.Pinned = pinned; return nil })
}

// Check runs during native household tasks as well as between decisions.
func (r *runner) Check(v behaviour.View) error {
	g, ok := r.t.Store.Get(r.id)
	if !ok {
		return errors.New("goal disappeared")
	}
	if g.Food == nil || g.Food.Stock == nil {
		return nil
	}
	if !v.Now.Before(g.Deadline) || !foodView(v, g) {
		return errors.New("household work needs fresh scoped observations within its deadline")
	}
	s := v.State
	if v.Home == nil || distance(*v.Home, g.Center) > .1 || distance(s.Pos, g.Center) > g.Radius || !finite(s.HP) || !finite(s.HPMax) || s.HPMax <= 0 || s.HP < math.Max(20, s.HPMax*.4) || s.Stranded() || s.Encumbered() || slices.Contains(s.Flags, protocol.FlagSwimming) || s.Location != nil && s.Location.InWater {
		return errors.New("unsafe household health, footing, home or capacity")
	}
	if g.Food.Injury != nil && (v.InjurySeq == nil || *g.Food.Injury != *v.InjurySeq) {
		return errors.New("injury during household work")
	}
	for _, c := range v.Perception.Characters {
		if c.HPFrac > 0 && (c.TargetingMe || c.Hostile && distance(c.Pos, s.Pos) < 15) {
			return errors.New("nearby threat interrupts household work")
		}
	}
	return stockScope(g, v)
}
