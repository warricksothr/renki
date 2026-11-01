package goal

import (
	"encoding/json"
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

// Acquisition is admitted only for new goals. Older checkpoints without this
// adapter retain their original storage-only scope on explicit resume.
type FoodAcquisition struct {
	VisitCount int             `json:"visitCount,omitempty"`
	Targets    int             `json:"targets"`
	Hunts      int             `json:"hunts"`
	Visits     []string        `json:"visits,omitempty"`
	AreaVisits []FoodAreaVisit `json:"areaVisits,omitempty"`
	Tried      []string        `json:"tried,omitempty"`
	Returning  bool            `json:"returning,omitempty"`
	Product    *FoodProduct    `json:"product,omitempty"`
}

// Keep dispatch feedback independently of the rolling native-outcome history.
// An uncertain or refused visit still spends its area cooldown.
type FoodAreaVisit struct {
	Name string    `json:"name"`
	At   time.Time `json:"at"`
}

type FoodProduct struct {
	Run             string        `json:"run,omitempty"`
	Entity          string        `json:"entity"`
	Item            string        `json:"item"`
	Pos             protocol.Vec3 `json:"pos"`
	Maximum         int           `json:"maximum"`
	BeforeCount     int           `json:"beforeCount"`
	BeforePack      uint64        `json:"beforePack"`
	BeforeRemaining float64       `json:"beforeRemaining"`
	TaskID          string        `json:"taskId,omitempty"`
	Until           time.Time     `json:"until"`
}

func foodAtHome(pos, center protocol.Vec3) bool {
	return pos.FlatDistance(center) <= 3 && math.Abs(pos[1]-center[1]) <= 1.75
}

// Reserve room for the native encounter's 52 m travel bound and rounding.
func acquisitionRadius(g Record) float64 {
	limit := 80.0
	if g.Food != nil && g.Food.Stock != nil {
		limit = 120
	}
	return math.Min(limit, g.Radius-64)
}
func validFoodPos(pos protocol.Vec3) bool {
	for _, x := range pos {
		if !finite(x) {
			return false
		}
	}
	return true
}
func validateAcquisition(g Record) error {
	a := g.Food.Acquisition
	if a == nil {
		return nil
	}
	if a.Targets < 0 || a.Targets > 12 || a.Hunts < 0 || a.Hunts > 2 || len(a.Visits) > 6 || len(a.AreaVisits) > 6 || len(a.Tried) > 24 {
		return errors.New("invalid food acquisition budgets")
	}
	for _, visit := range a.AreaVisits {
		name, err := home.WaypointName(visit.Name)
		if err != nil || name != visit.Name || visit.At.IsZero() || visit.At.Before(g.Created) || visit.At.After(g.Deadline) {
			return errors.New("invalid food area visit")
		}
	}
	if p := a.Product; p != nil {
		if p.Entity == "" || !slices.Contains(append(append(slices.Clone(meals), ingredients...), "Wood"), p.Item) || !validFoodPos(p.Pos) || distance(p.Pos, g.Center) > acquisitionRadius(g) || p.Maximum < 1 || p.Maximum > 50 || p.BeforeCount < 0 || !finite(p.BeforeRemaining) || p.BeforeRemaining < 0 || p.Until.IsZero() || p.Until.After(g.Deadline) {
			return errors.New("invalid food product checkpoint")
		}
	}
	return nil
}

func foodResourceFits(v behaviour.View, y *protocol.ResourceYield) bool {
	if y == nil || y.Count < 1 || y.Count > 50 || y.StackMax < 1 || !finite(y.Weight) || y.Weight < 0 || !finite(v.State.Weight) || !finite(v.State.WeightMax) || v.State.WeightMax <= 0 {
		return false
	}
	weight := v.State.Weight
	packWeight := 0.0
	for _, i := range v.Inventory.Items {
		if i.Stack < 0 || !finite(i.Weight) || i.Weight < 0 {
			return false
		}
		packWeight += float64(i.Stack) * i.Weight
	}
	weight = math.Max(weight, packWeight)
	return weight+float64(y.Count)*y.Weight <= v.State.WeightMax*.8 && v.Inventory.Free() >= int(math.Ceil(float64(y.Count)/float64(y.StackMax)))
}
func foodSiteSafe(v behaviour.View, pos protocol.Vec3, except string) bool {
	if !validFoodPos(pos) {
		return false
	}
	for _, c := range v.Perception.Characters {
		if c.ID == except || c.HPFrac <= 0 || c.Kind == protocol.CharacterPlayer {
			continue
		}
		clearance := 15.0
		if c.Hostile || c.TargetingMe {
			// Match the native loaded-hostile exclusion in the horizontal
			// plane. Perception can reject a known occupied source; it cannot
			// establish clearance from threats loaded only by the plugin.
			clearance = 40
			if c.Stars == 0 {
				switch c.Prefab {
				case "Neck":
					clearance = 16
				case "Boar":
					clearance = 24
				}
			}
		}
		if c.TargetingMe || !validFoodPos(c.Pos) || c.Pos.FlatDistance(pos) < clearance {
			return false
		}
	}
	return true
}
func foodAcquisitionNegative(last *behaviour.Result) bool {
	return slices.Contains([]string{protocol.StatusFailed, protocol.StatusRejected}, last.Status) && slices.Contains([]string{protocol.ReasonInvalidTarget, protocol.ReasonTargetLost, protocol.ReasonUnreachable, protocol.ReasonRefused}, last.Reason)
}

func (r *runner) foodAcquire(g Record, v behaviour.View, items []string) (*session.Order, error) {
	a := g.Food.Acquisition
	if a == nil {
		return r.need("no raw ingredient in pack or eligible observed home chests; goal has no acquisition adapter")
	}
	if a.Targets >= 12 {
		return r.need("food acquisition resource limit exhausted")
	}
	var candidates []protocol.PerceivedResource
	for _, p := range v.Perception.Resources {
		if p.ID == "" || slices.Contains(a.Tried, p.ID) || !validFoodPos(p.Pos) || distance(p.Pos, g.Center) > acquisitionRadius(g) || p.Yield == nil || !slices.Contains(items, p.Yield.Prefab) || !foodResourceFits(v, p.Yield) || !foodSiteSafe(v, p.Pos, "") || !stockResourceSource(g, p) {
			continue
		}
		if p.Kind == protocol.ResourceDrop || p.Kind == protocol.ResourcePickable {
			candidates = append(candidates, p)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		// A ready meal costs less than a raw ingredient plus cooking/fuel.
		ai, bi := slices.Index(items, a.Yield.Prefab), slices.Index(items, b.Yield.Prefab)
		if ai != bi {
			return ai < bi
		}
		da, db := foodTravelCost(v, a.Pos), foodTravelCost(v, b.Pos)
		if da != db {
			return da < db
		}
		return a.ID < b.ID
	})
	if len(candidates) > 0 {
		return r.foodTake(g, v, candidates[0], true)
	}
	if g.Food.Stock != nil && g.Food.Stock.AllowHunting {
		if o, err := r.foodHuntNext(g, v, items); o != nil || err != nil {
			return o, err
		}
	}
	if s := g.Food.Stock; s != nil && len(s.Sources) > 0 {
		if o, err := r.foodSearchAreas(g, v, s.Pinned); o != nil || err != nil {
			return o, err
		}

		if !foodAtHome(v.State.Pos, g.Center) {
			return r.stockReturn(g, v, s.Known, s.Shortage, "safe source search exhausted; deliver before stopping")
		}
		return r.need("household source markers exhausted or unsafe; wait for regrowth or revise sources")
	}
	// Site memory only offers a travel hint. A new live yield is required there.
	if len(a.Visits) < 6 && r.t.Landmarks != nil {
		sites := r.t.Landmarks(g.World)
		sort.Slice(sites, func(i, j int) bool {
			di, dj := foodTravelCost(v, sites[i].Pos), foodTravelCost(v, sites[j].Pos)
			if di != dj {
				return di < dj
			}
			return sites[i].Entity < sites[j].Entity
		})
		for _, p := range sites {
			item := map[string]string{"RaspberryBush": "Raspberry", "BlueberryBush": "Blueberries", "Pickable_Mushroom": "Mushroom", "Pickable_Branch": "Wood"}[p.Prefab]
			if p.Entity == "" || p.Kind != protocol.ResourcePickable || !slices.Contains(items, item) || slices.Contains(a.Tried, p.Entity) || p.At.IsZero() || p.At.After(v.Now) || v.Now.Sub(p.At) > 7*24*time.Hour || !validFoodPos(p.Pos) || distance(p.Pos, g.Center) > acquisitionRadius(g) || distance(p.Pos, v.State.Pos) <= 8 || !foodSiteSafe(v, p.Pos, "") || !stockSource(g, p.Pos) {
				continue
			}
			// Don't revisit a different remembered ID for the same small location.
			key := foodSiteKey(p.Pos)
			if slices.Contains(a.Visits, key) {
				continue
			}
			return r.foodDispatch(g, v, "search_food", item, p.Entity, &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &p.Pos}, Dry: true, AvoidHostiles: true}, TimeoutS: 90}, func(g *Record) { g.Food.Acquisition.Visits = append(g.Food.Acquisition.Visits, key) })
		}
	}
	if g.Food.Stock == nil {
		if o, err := r.foodHuntNext(g, v, items); o != nil || err != nil {
			return o, err
		}
	}
	if !foodAtHome(v.State.Pos, g.Center) {
		return r.foodDispatch(g, v, "food_return", "", "", &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: g.Home}, Dry: true, AvoidHostiles: true}, TimeoutS: 120})
	}
	return r.need("food sources exhausted: no safe live yield, unvisited forage site or permitted isolated prey")
}
func foodSiteKey(pos protocol.Vec3) string {
	// Consolidation at four-metre cells is only a search budget, never identity.
	return fmt.Sprintf("site:%d:%d:%d", int(math.Floor(pos[0]/4)), int(math.Floor(pos[1]/4)), int(math.Floor(pos[2]/4)))
}
func foodTravelCost(v behaviour.View, pos protocol.Vec3) float64 {
	return distance(v.State.Pos, pos) + 12*home.DangerRisk(v.Danger, pos)
}

func (r *runner) foodTake(g Record, v behaviour.View, p protocol.PerceivedResource, newProduct bool) (*session.Order, error) {
	action, task := "pickup_food", protocol.TaskPickup
	var args any = &protocol.PickupArgs{Entity: p.ID}
	if p.Kind == protocol.ResourcePickable {
		action, task = "forage", protocol.TaskInteract
		args = &protocol.InteractArgs{Entity: p.ID}
	}
	return r.foodDispatch(g, v, action, p.Yield.Prefab, p.ID, &session.Order{Task: task, Args: args, TimeoutS: 45}, func(g *Record) {
		a := g.Food.Acquisition
		a.Targets++
		a.Tried = append(a.Tried, p.ID)
		if newProduct {
			until := v.Now.Add(60 * time.Second)
			if until.After(g.Deadline) {
				until = g.Deadline
			}
			a.Product = &FoodProduct{Entity: p.ID, Item: p.Yield.Prefab, Pos: p.Pos, Maximum: p.Yield.Count, BeforeCount: v.Inventory.Count(p.Yield.Prefab), BeforePack: v.Inventory.Seq, BeforeRemaining: remaining(v.State, p.Yield.Prefab), Until: until}
		}
	})
}

// Account for a completed exact action without issuing any further work.
func (r *runner) foodProductGain(g Record, v behaviour.View) (bool, error) {
	if g.Food.Acquisition == nil || g.Food.Acquisition.Product == nil {
		return false, nil
	}
	p := g.Food.Acquisition.Product
	if p.TaskID == "" {
		return false, nil
	}
	if p.Run != r.t.Store.runID {
		return false, errors.New("food product belongs to another run; not replayed")
	}
	gain := v.Inventory.Count(p.Item) - p.BeforeCount
	if remaining(v.State, p.Item) > p.BeforeRemaining+1 {
		gain++
	}
	if v.Inventory.Seq > p.BeforePack && gain > 0 {
		if gain > p.Maximum {
			return false, errors.New("food product gain exceeds observed yield; receipt is ambiguous")
		}
		if err := r.t.Store.update(g.ID, func(g *Record) error {
			g.Food.Receipts = append(g.Food.Receipts, FoodReceipt{Run: p.Run, TaskID: p.TaskID, Action: "collect", Entity: p.Entity, Item: p.Item, Count: gain, Evidence: "observed_yield_fresh_inventory"})
			g.Food.Acquisition.Product = nil
			g.Food.Acquisition.Returning = g.Food.Stock == nil && !foodAtHome(v.State.Pos, g.Center)
			return nil
		}); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (r *runner) foodProductNext(g Record, v behaviour.View) (*session.Order, error) {
	a, p := g.Food.Acquisition, g.Food.Acquisition.Product
	if p.TaskID == "" {
		return r.need("food product missing scoped native result; not replayed")
	}
	if v.Now.Before(p.Until) {
		for _, drop := range v.Perception.Resources {
			if a.Targets >= 12 {
				break
			}
			if drop.Kind != protocol.ResourceDrop || drop.ID == "" || slices.Contains(a.Tried, drop.ID) || drop.Yield == nil || drop.Yield.Prefab != p.Item || drop.Yield.Count > p.Maximum || distance(drop.Pos, p.Pos) > 4 || distance(drop.Pos, g.Center) > acquisitionRadius(g) || !foodResourceFits(v, drop.Yield) || !foodSiteSafe(v, drop.Pos, "") {
				continue
			}
			return r.foodTake(g, v, drop, false)
		}
		return r.foodWait(g, v, "waiting for observed food products or fresh pack gain")
	}
	// No gain is an expected negative. The source stays spent; never re-interact.
	if err := r.t.Store.update(g.ID, func(g *Record) error {
		g.Food.Acquisition.Product = nil
		g.Reason = "food source produced no verified carried gain"
		return nil
	}); err != nil {
		return nil, err
	}
	g, _ = r.t.Store.Get(g.ID)
	return r.foodNext(g, v, nil)
}

func (r *runner) foodAcquisitionResult(g Record, v behaviour.View, last *behaviour.Result, out Outcome) error {
	a, p := g.Food.Acquisition, g.Pending
	if a == nil {
		return errors.New("missing food acquisition checkpoint")
	}
	if v.Perception.Seq <= g.Food.BeforePerception {
		return behaviour.ErrWait
	}
	success := last.Status == protocol.StatusSucceeded
	if !success && !foodAcquisitionNegative(last) {
		return errors.New("food acquisition execution fault: " + last.Reason + " " + last.Detail)
	}
	if p.Template == "hunt_food" {
		if success && (last.Entity != p.Entity || last.Detail != "observed "+g.Food.Item+" death") {
			return errors.New("food hunt lacks exact native observed target death; not retried")
		}
		if !success && !slices.Contains([]string{protocol.ReasonTargetLost, protocol.ReasonUnreachable, protocol.ReasonInvalidTarget}, last.Reason) {
			return errors.New("native food hunt stopped: " + last.Reason + " " + last.Detail)
		}
	}
	if p.Template == "search_food" && success {
		var args protocol.MoveToArgs
		if json.Unmarshal(p.Params, &args) != nil || args.To.Pos == nil || !foodAtHome(v.State.Pos, *args.To.Pos) {
			return errors.New("forage visit lacks confirmed arrival")
		}
	}
	return r.t.Store.update(g.ID, func(g *Record) error {
		a := g.Food.Acquisition
		g.Pending = nil
		g.Last = &out
		g.History = append(g.History, out)
		if len(g.History) > 16 {
			g.History = g.History[len(g.History)-16:]
		}
		if p.Template == "forage" || p.Template == "pickup_food" {
			if a.Product == nil {
				return errors.New("missing admitted food product")
			}
			a.Product.TaskID = last.TaskID
			a.Product.Run = p.Run
			// A successful interact can spawn products via RPC after the final result.
			a.Product.Until = v.Now.Add(8 * time.Second)
			if a.Product.Until.After(g.Deadline) {
				a.Product.Until = g.Deadline
			}
		}
		if p.Template == "hunt_food" && success {
			g.Food.Receipts = append(g.Food.Receipts, FoodReceipt{Run: p.Run, TaskID: last.TaskID, Action: "hunt", Entity: p.Entity, Item: g.Food.Item, Count: 1, Evidence: "native_observed_death"})
			// Death is not an ingredient receipt. A later live drop needs its own
			// measured pickup; allow RPC loot a short, charged observation interval.
			g.Wake = v.Now.Add(2 * time.Second)
			g.WaitS += 2
			if g.WaitS > 240 {
				return errors.New("food hunt loot wait budget exhausted")
			}
		}
		return nil
	})
}
