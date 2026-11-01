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
	"github.com/warricksothr/renki/internal/food"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

var meals = []string{"CookedDeerMeat", "NeckTailGrilled", "CookedMeat", "Honey", "Raspberry", "Blueberries", "Mushroom", "MushroomYellow"}
var ingredients = []string{"RawMeat", "DeerMeat", "NeckTail"}
var cooked = map[string]string{"RawMeat": "CookedMeat", "DeerMeat": "CookedDeerMeat", "NeckTail": "NeckTailGrilled"}

// FoodProgress is a typed host adapter, not a checkpoint of an arbitrary runner.
type FoodProgress struct {
	Stock            *FoodStock           `json:"stock,omitempty"`
	Acquisition      *FoodAcquisition     `json:"acquisition,omitempty"`
	CandidateCount   int                  `json:"candidateCount,omitempty"`
	AttemptCount     int                  `json:"attemptCount,omitempty"`
	ReceiptCount     int                  `json:"receiptCount,omitempty"`
	Continuation     *Intent              `json:"continuation,omitempty"`
	Phase            string               `json:"phase"`
	Reserve          int                  `json:"reserve"`
	Storage          []string             `json:"storage,omitempty"`
	Escort           *protocol.EscortArgs `json:"escort,omitempty"`
	Chests           []FoodChest          `json:"chests,omitempty"`
	SearchPinned     bool                 `json:"searchPinned"`
	Tried            []string             `json:"tried,omitempty"`
	Receipts         []FoodReceipt        `json:"receipts,omitempty"`
	BeforeSeq        uint64               `json:"beforeSeq,omitempty"`
	BeforePackSeq    uint64               `json:"beforePackSeq,omitempty"`
	BeforePerception uint64               `json:"beforePerception,omitempty"`
	BeforeCount      int                  `json:"beforeCount,omitempty"`
	BeforeRemaining  float64              `json:"beforeRemaining,omitempty"`
	Item             string               `json:"item,omitempty"`
	Injury           *uint64              `json:"injury,omitempty"`
	Batches          int                  `json:"batches"`
}
type FoodChest struct {
	ObservedAt  time.Time      `json:"observedAt,omitempty"`
	Run         string         `json:"run,omitempty"`
	Entity      string         `json:"entity"`
	Pos         protocol.Vec3  `json:"pos"`
	Contents    map[string]int `json:"contents,omitempty"`
	Observed    bool           `json:"observed"`
	Unavailable bool           `json:"unavailable"`
}
type FoodReceipt struct {
	Run      string `json:"run,omitempty"`
	TaskID   string `json:"taskId"`
	Action   string `json:"action"`
	Entity   string `json:"entity,omitempty"`
	Item     string `json:"item,omitempty"`
	Count    int    `json:"count"`
	Evidence string `json:"evidence"`
}

func validateFood(g Record) error {
	f := g.Food
	if f == nil || f.Reserve < 1 || f.Reserve > 3 || f.Batches < 0 || f.Batches > 4 || len(f.Chests) > 64 || len(f.Storage) > 64 || len(f.Tried) > 256 || len(f.Receipts) > 256 || g.ReturnHome || g.PostCombat != nil || len(g.Searches) > 0 {
		return errors.New("invalid food checkpoint or limits")
	}
	if !slices.Contains([]string{"home", "ready", "raw", "wood", "cook", "recover", "complete"}, f.Phase) {
		return errors.New("invalid food phase")
	}
	if err := validateStock(g); err != nil {
		return err
	}
	if err := validateAcquisition(g); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, c := range f.Chests {
		if c.Entity == "" || seen[c.Entity] || distance(c.Pos, g.Center) > 32 || len(c.Contents) > 12 {
			return errors.New("invalid food chest ledger")
		}
		for _, n := range c.Pos {
			if !finite(n) {
				return errors.New("invalid food chest position")
			}
		}
		seen[c.Entity] = true
	}
	if f.Escort != nil {
		t, err := protocol.NewTask("check", protocol.TaskEscort, f.Escort, 0)
		if err != nil {
			return err
		}
		if _, err = t.DecodeArgs(); err != nil {
			return err
		}
	}
	return nil
}

func mealCount(pack *protocol.Inventory) int {
	n := 0
	if pack != nil {
		for _, p := range meals {
			n += pack.Count(p)
		}
	}
	return n
}
func remaining(s *protocol.State, item string) float64 {
	n := 0.0
	if s.Food != nil {
		for _, f := range s.Food.Active {
			if f.Prefab == item {
				n = math.Max(n, f.RemainingS)
			}
		}
	}
	return n
}
func nourished(s *protocol.State) bool {
	if s.Food != nil {
		for _, f := range s.Food.Active {
			if f.RemainingS >= 300 {
				return true
			}
		}
	}
	return false
}
func (r *runner) foodWait(g Record, v behaviour.View, why string) (*session.Order, error) {
	if g.WaitS+2 > 240 {
		return r.need("food observation/recovery wait budget exhausted")
	}
	until := v.Now.Add(2 * time.Second)
	err := r.t.Store.update(g.ID, func(g *Record) error {
		g.WaitS += 2
		g.Reason = why
		g.Wake = until
		g.Observation = r.observation
		return nil
	})
	if err != nil {
		return nil, err
	}
	return nil, &behaviour.Delay{Until: until}
}
func foodView(v behaviour.View, g Record) bool {
	return v.State != nil && v.State.World == g.World && v.State.Phase == protocol.PhaseInWorld &&
		!v.StateAt.IsZero() && v.Now.Sub(v.StateAt) >= 0 && v.Now.Sub(v.StateAt) <= 3*time.Second &&
		v.Inventory != nil && v.Perception != nil &&
		!v.PerceptionAt.IsZero() && v.Now.Sub(v.PerceptionAt) >= 0 && v.Now.Sub(v.PerceptionAt) <= 3*time.Second && math.Abs(v.State.T-v.Perception.T) <= 3
}

func (r *runner) foodNext(g Record, v behaviour.View, last *behaviour.Result) (*session.Order, error) {
	if terminal(g.Status) {
		return nil, nil
	}
	if !v.Now.Before(g.Deadline) {
		return r.need("food goal deadline exhausted")
	}
	if !foodView(v, g) {
		return r.foodWait(g, v, "waiting for scoped state, pack and surroundings")
	}
	if err := r.pinStockSources(g, v); err != nil {
		return r.need(err.Error())
	}
	g, _ = r.t.Store.Get(g.ID)
	s, f := v.State, g.Food
	if err := stockScope(g, v); err != nil {
		return r.need(err.Error())
	}
	if !finite(s.HP) || !finite(s.HPMax) || s.HPMax <= 0 || s.HP < math.Max(20, s.HPMax*.4) || s.Stranded() || s.Encumbered() || slices.Contains(s.Flags, protocol.FlagSwimming) || s.Location != nil && s.Location.InWater {
		return r.need("unsafe food resupply health, footing or capacity")
	}
	if f.Injury != nil && (v.InjurySeq == nil || *f.Injury != *v.InjurySeq) {
		return r.need("injury during food resupply")
	}
	for _, c := range v.Perception.Characters {
		if c.HPFrac > 0 && (c.Hostile || c.TargetingMe) && distance(c.Pos, s.Pos) < 15 {
			return r.need("nearby threat prevents food resupply")
		}
	}
	if v.Home == nil || distance(*v.Home, g.Center) > .1 || distance(s.Pos, g.Center) > g.Radius {
		return r.need("food resupply needs its original home and travel scope")
	}
	if g.Home == nil {
		if err := r.t.Store.update(g.ID, func(g *Record) error {
			p := *v.Home
			g.Home = &p
			if v.InjurySeq != nil {
				n := *v.InjurySeq
				g.Food.Injury = &n
			}
			return nil
		}); err != nil {
			return nil, err
		}
		g, _ = r.t.Store.Get(g.ID)
		f = g.Food
	}
	if g.Pending != nil {
		if last == nil || last.TaskID == "" || last.TaskID != g.Pending.TaskID {
			return r.need("uncertain food action; native receipt missing or mismatched")
		}
		if v.State.Seq <= f.BeforeSeq {
			return r.foodWait(g, v, "waiting for post-action state")
		}
		if err := r.foodResult(g, v, last); err != nil {
			if errors.Is(err, behaviour.ErrWait) {
				return r.foodWait(g, v, "waiting for matching food inventory receipt")
			}
			return r.need(err.Error())
		}
		g, _ = r.t.Store.Get(g.ID)
		f = g.Food
	}
	if f.Stock != nil && f.Stock.ReturnAfterCooking {
		if !foodAtHome(s.Pos, g.Center) || s.Pos.FlatDistance(g.Center) > .75 {
			within := .5
			return r.foodDispatch(g, v, "cook_return", "", "", &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: g.Home}, StopDistance: &within, Dry: true, AvoidHostiles: true}, TimeoutS: 90})
		}
		if err := r.t.Store.update(g.ID, func(g *Record) error { g.Food.Stock.ReturnAfterCooking = false; return nil }); err != nil {
			return nil, err
		}
		g, _ = r.t.Store.Get(g.ID)
		f = g.Food
	}
	if g.Wake.After(v.Now) {
		return nil, &behaviour.Delay{Until: g.Wake}
	}
	if a := f.Acquisition; a != nil {
		if a.Product != nil {
			return r.foodProductNext(g, v)
		}
	}
	// Stock observations mark admission to household work after the initial
	// personal bootstrap. End that travel before resupply/recovery can admit
	// more remote work. Account for an in-flight product first, then retain
	// delivery even if food or capacity improves on the way home.
	if stock := f.Stock; stock != nil && !foodAtHome(s.Pos, g.Center) &&
		(stock.Delivering || stock.Known != nil && (stockLoaded(v) || !nourished(s) || mealCount(v.Inventory) < f.Reserve || s.HP < s.HPMax*.8)) {
		return r.stockReturn(g, v, stock.Known, stock.Shortage, "committed delivery: personal reserve, nourishment, recovery or capacity pressure")
	}
	itemToEat := food.Choose(s.Food)
	fillSlot := itemToEat != "" && s.Food != nil && len(s.Food.Active) < 3 && remaining(s, itemToEat) == 0
	// A reserve is measured after filling eligible food slots, so the normal
	// keeper cannot immediately eat the entire newly prepared reserve.
	if !fillSlot && mealCount(v.Inventory) >= f.Reserve && nourished(s) && s.HP >= s.HPMax*.8 {
		if f.Stock != nil {
			return r.stockNext(g, v)
		}
		if !foodAtHome(s.Pos, g.Center) {
			return r.foodDispatch(g, v, "food_return", "", "", &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: g.Home}, Dry: true, AvoidHostiles: true}, TimeoutS: 120})
		}
		d, err := Evaluate(g.Policy, map[string]any{"objectiveSatisfied": true}, g.State, g.Last)
		if err != nil || d.Action != "done" {
			return r.need("food policy refused host readiness")
		}
		return nil, r.t.Store.update(g.ID, func(g *Record) error {
			g.Status = "succeeded"
			g.Food.Phase = "complete"
			g.Reason = "verified carried food reserve, active meal and recovered health"
			g.Observation = r.observation
			return nil
		})
	}
	// Prefer eating existing carried food; ordinary keeper actions remain valid.
	if item := itemToEat; item != "" && (fillSlot || !nourished(s) || s.HP < s.HPMax*.8) {
		return r.foodDispatch(g, v, "eat", item, "", &session.Order{Task: protocol.TaskUseItem, Args: &protocol.UseItemArgs{Item: item}, TimeoutS: 5})
	}
	if s.HP < s.HPMax*.8 {
		if s.Food == nil || len(s.Food.Active) == 0 {
			return r.need("active food required to recover before travel")
		}
		return r.foodWait(g, v, "recovering health before food work")
	}
	if a := f.Acquisition; a != nil && a.Returning {
		return r.foodDispatch(g, v, "food_return", "", "", &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: g.Home}, Dry: true, AvoidHostiles: true}, TimeoutS: 120})
	}
	if f.Phase == "home" {
		if !foodAtHome(s.Pos, g.Center) {
			return r.foodDispatch(g, v, "home", "", "", &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: g.Home}, Dry: true}, TimeoutS: 180})
		}
		if err := r.pinFoodSearch(g, v); err != nil {
			return r.need(err.Error())
		}
		g, _ = r.t.Store.Get(g.ID)
		f = g.Food
	}
	// Ready-food search always exhausts its eligible unique boxes before cooking.
	if f.Phase == "ready" {
		if mealCount(v.Inventory) < f.Reserve+boolInt(!nourished(s)) {
			if o, item, chest := foodWithdrawal(g, v, meals, f.Reserve+1); o != nil {
				return r.foodDispatch(g, v, "ready", item, chest, o)
			}
		}
		if mealCount(v.Inventory) >= f.Reserve {
			return r.foodWait(g, v, "waiting for food eating and recovery")
		}
		if err := r.foodPhase(g.ID, "raw"); err != nil {
			return nil, err
		}
		g, _ = r.t.Store.Get(g.ID)
		f = g.Food
	}
	if f.Phase == "raw" {
		for _, p := range ingredients {
			if v.Inventory.Count(p) > 0 {
				if err := r.foodPhase(g.ID, "wood"); err != nil {
					return nil, err
				}
				g, _ = r.t.Store.Get(g.ID)
				f = g.Food
				break
			}
		}
		if f.Phase == "raw" {
			if o, item, chest := foodWithdrawal(g, v, ingredients, max(1, f.Reserve+1-mealCount(v.Inventory))); o != nil {
				return r.foodDispatch(g, v, "raw", item, chest, o)
			}
			return r.foodAcquire(g, v, append(slices.Clone(meals), ingredients...))
		}
	}
	if f.Phase == "wood" || f.Phase == "cook" {
		if !foodAtHome(s.Pos, g.Center) {
			return r.foodDispatch(g, v, "food_return", "", "", &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: g.Home}, Dry: true, AvoidHostiles: true}, TimeoutS: 120})
		}
		if f.Batches >= 4 {
			return r.need("bounded cooking batch limit reached")
		}
		station, hot := foodStation(v)
		if station == "" {
			return r.need("no fresh empty supported station within 12 m")
		}
		if !hot && v.Inventory.Count("Wood") == 0 {
			if o, item, chest := foodWithdrawal(g, v, []string{"Wood"}, 1); o != nil {
				return r.foodDispatch(g, v, "wood", item, chest, o)
			}
			return r.foodAcquire(g, v, []string{"Wood"})
		}
		if s.Food == nil || len(s.Food.Active) == 0 {
			return r.need("native cooking requires active food; no safe bootstrap meal available")
		}
		for _, p := range ingredients {
			if v.Inventory.Count(p) > 0 {
				return r.foodDispatch(g, v, "cook", p, station, &session.Order{Task: protocol.TaskInteract, Args: &protocol.InteractArgs{Entity: station, Cook: p}, TimeoutS: 180})
			}
		}
		return r.need("ingredient disappeared before cooking; not replaced blindly")
	}
	return r.need("food adapter has no next action")
}
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
func (r *runner) foodPhase(id, phase string) error {
	return r.t.Store.update(id, func(g *Record) error { g.Food.Phase = phase; return nil })
}

func (r *runner) pinFoodSearch(g Record, v behaviour.View) error {
	f := g.Food
	if f.SearchPinned {
		return r.foodPhase(g.ID, "ready")
	}
	cs := []FoodChest{}
	seen := map[string]bool{}
	for _, c := range v.Perception.Structures {
		if c.ID == "" || c.Kind != protocol.StructureContainer || distance(c.Pos, g.Center) > 32 || seen[c.ID] || len(f.Storage) > 0 && !slices.Contains(f.Storage, c.ID) {
			continue
		}
		seen[c.ID] = true
		chest := FoodChest{Entity: c.ID, Pos: c.Pos, Contents: map[string]int{}}
		for _, hint := range v.Containers {
			if hint.Entity == c.ID {
				for _, p := range append(append(slices.Clone(meals), ingredients...), "Wood") {
					if n := hint.Inventory().Count(p); n > 0 {
						chest.Contents[p] = n
					}
				}
			}
		}
		cs = append(cs, chest)
	}
	if len(cs) > 64 {
		return errors.New("more than 64 eligible chests; narrow the storage area explicitly")
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].Entity < cs[j].Entity })
	return r.t.Store.update(g.ID, func(g *Record) error {
		g.Food.Chests = cs
		g.Food.SearchPinned = true
		g.Food.Phase = "ready"
		return nil
	})
}

func foodWithdrawal(g Record, v behaviour.View, prefabs []string, amount int) (*session.Order, string, string) {
	type option struct {
		chest, item string
		rank        int
		cost        float64
		amount      int
	}
	var options []option
	for _, c := range g.Food.Chests {
		if c.Unavailable {
			continue
		}
		live := slices.ContainsFunc(v.Perception.Structures, func(p protocol.PerceivedStructure) bool {
			return p.ID == c.Entity && p.Kind == protocol.StructureContainer && distance(p.Pos, c.Pos) < .5
		})
		if !live {
			continue
		}
		for rank, p := range prefabs {
			key := g.Food.Phase + ":" + c.Entity + ":" + p
			if slices.Contains(g.Food.Tried, key) {
				continue
			}
			n := c.Contents[p]
			if n <= 0 {
				if c.Observed || rank != 0 {
					continue
				}
				rank = len(prefabs)
				n = 1
			}
			options = append(options, option{c.Entity, p, rank, distance(v.State.Pos, c.Pos), min(n, amount)})
		}
	}
	sort.Slice(options, func(i, j int) bool {
		a, b := options[i], options[j]
		if a.rank != b.rank {
			return a.rank < b.rank
		}
		if a.cost != b.cost {
			return a.cost < b.cost
		}
		return a.chest < b.chest
	})
	if len(options) == 0 {
		return nil, "", ""
	}
	c := options[0]
	return &session.Order{Task: protocol.TaskWithdraw, Args: &protocol.TransferArgs{Container: c.chest, Items: []protocol.TransferItem{{Prefab: c.item, Count: protocol.ItemCount(c.amount)}}}, TimeoutS: 45}, c.item, c.chest
}

func foodStation(v behaviour.View) (string, bool) {
	id := ""
	best := math.Inf(1)
	hot := false
	for _, s := range v.Perception.Structures {
		c := s.Cooking
		if s.ID == "" || c == nil || !c.Supported || len(c.Slots) == 0 || slices.ContainsFunc(c.Slots, func(s protocol.CookingSlotStatus) bool { return s.State != "empty" || s.Item != "" }) {
			continue
		}
		d := distance(s.Pos, v.State.Pos)
		if d <= 12 && (c.Heated && !hot || c.Heated == hot && (d < best || d == best && s.ID < id)) {
			id, best, hot = s.ID, d, c.Heated
		}
	}
	return id, hot
}

func (r *runner) foodDispatch(g Record, v behaviour.View, action, item, entity string, o *session.Order, prepare ...func(*Record)) (*session.Order, error) {
	if args, ok := o.Args.(*protocol.MoveToArgs); ok && g.Food.Stock != nil {
		args.RecoverThreats = args.Dry && args.AvoidHostiles && v.State != nil && v.State.Food != nil && len(v.State.Food.Active) > 0
	}
	if g.TaskCount >= g.MaxTasks {
		return r.need("food native-task budget exhausted")
	}
	raw, err := json.Marshal(o.Args)
	if err != nil {
		return nil, err
	}
	var p any
	_ = json.Unmarshal(raw, &p)
	d, err := Evaluate(g.Policy, map[string]any{"objectiveSatisfied": false, "nextAction": map[string]any{"template": action, "params": p}}, g.State, g.Last)
	if err != nil || d.Action != "run" || d.Template != action || !sameJSON(raw, d.Params) {
		return r.need("food policy attempted to change host action")
	}
	err = r.t.Store.update(g.ID, func(g *Record) error {
		g.TaskCount++
		g.Attempts++
		g.Observation = r.observation
		g.Wake = time.Time{}
		g.Reason = "food " + action + ": " + entity + " " + item
		g.Food.BeforeSeq = v.State.Seq
		g.Food.BeforePackSeq = v.Inventory.Seq
		g.Food.BeforePerception = v.Perception.Seq
		g.Food.BeforeCount = v.Inventory.Count(item)
		g.Food.BeforeRemaining = remaining(v.State, item)
		g.Food.Item = item
		g.Pending = &Intent{Run: r.t.Store.runID, At: v.Now, Template: action, Params: raw, Entity: entity, Fingerprint: fmt.Sprintf("%s:%s:%s:%d", action, entity, item, v.Inventory.Seq)}
		if slices.Contains([]string{"ready", "raw", "wood"}, action) {
			g.Food.Tried = append(g.Food.Tried, g.Food.Phase+":"+entity+":"+item)
		}
		for _, p := range prepare {
			p(g)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	o.TimeoutS = math.Min(o.TimeoutS, g.Deadline.Sub(v.Now).Seconds())
	return o, nil
}

func (r *runner) foodResult(g Record, v behaviour.View, last *behaviour.Result) error {
	p, f := g.Pending, g.Food
	out := Outcome{Template: p.Template, Status: last.Status, Reason: last.Reason, Detail: truncate(last.Detail, 256), TaskID: last.TaskID, Entity: p.Entity, At: v.Now}
	out.Run = p.Run
	var receipt *FoodReceipt
	var observed *protocol.Container
	success := last.Status == protocol.StatusSucceeded
	switch p.Template {
	case "home":
		if !success || !foodAtHome(v.State.Pos, g.Center) {
			return errors.New("food return lacks confirmed home arrival")
		}
	case "cook_return":
		if !success || !foodAtHome(v.State.Pos, g.Center) || v.State.Pos.FlatDistance(g.Center) > .75 {
			return errors.New("post-cooking wait point lacks confirmed precise home-floor arrival")
		}
	case "food_return":
		if !success {
			return fmt.Errorf("food return failed (%s): %s", last.Reason, last.Detail)
		}
		if !foodAtHome(v.State.Pos, g.Center) {
			return errors.New("food acquisition lacks confirmed home floor arrival")
		}
	case "forage", "pickup_food", "search_food", "hunt_food":
		return r.foodAcquisitionResult(g, v, last, out)
	case "eat":
		if !success {
			return errors.New("normal food consumption failed")
		}
		if remaining(v.State, f.Item) <= f.BeforeRemaining+1 {
			return behaviour.ErrWait
		}
		receipt = &FoodReceipt{TaskID: last.TaskID, Action: "eat", Item: f.Item, Count: 1, Evidence: "active_food_refresh"}
	case "ready", "raw", "wood", "stock_probe", "stock_deposit":
		direction := protocol.TaskWithdraw
		if p.Template == "stock_deposit" {
			direction = protocol.TaskDeposit
		}
		for _, c := range v.Containers {
			if c.Task == last.TaskID && c.Entity == p.Entity && c.Direction == direction {
				observed = c
				break
			}
		}
		if observed != nil {
			moved := observed.Moved[f.Item]
			limit := 4
			if p.Template == "stock_deposit" {
				var a protocol.TransferArgs
				if json.Unmarshal(p.Params, &a) != nil || len(a.Items) != 1 {
					return errors.New("invalid stock deposit intent")
				}
				limit = int(a.Items[0].Count)
			}
			if p.Template == "stock_probe" {
				limit = 1
			}
			if moved < 0 || moved > limit {
				return errors.New("invalid food transfer count")
			}
			if moved > 0 {
				consumed := 0
				if slices.Contains(meals, f.Item) && remaining(v.State, f.Item) > f.BeforeRemaining+1 {
					consumed = 1
				}
				validPack := v.Inventory.Count(f.Item) >= f.BeforeCount+moved-consumed
				if p.Template == "stock_deposit" {
					validPack = v.Inventory.Count(f.Item) == f.BeforeCount-moved
				}
				if v.Inventory.Seq <= f.BeforePackSeq || !validPack {
					return behaviour.ErrWait
				}
				receipt = &FoodReceipt{TaskID: last.TaskID, Action: p.Template, Entity: p.Entity, Item: f.Item, Count: moved, Evidence: "matched_transfer_inventory"}
			}
		} else if success {
			return errors.New("food transfer succeeded without its scoped receipt")
		}
		missingStock := observed != nil && last.Reason == protocol.ReasonInvalidTarget
		if !success && !missingStock && !slices.Contains([]string{protocol.ReasonMissingMaterials, protocol.ReasonRefused, protocol.ReasonTargetLost, protocol.ReasonUnreachable, protocol.ReasonInventoryFull}, last.Reason) {
			return errors.New("food transfer execution fault: " + last.Reason + " " + last.Detail)
		}
	case "cook":
		if !success {
			return errors.New("native cooking failed; partial batch is not retried: " + last.Reason + " " + last.Detail)
		}
		count, evidence := cookingDelivery(last.Cooking, last.TaskID, cooked[f.Item], p.Entity)
		if count < 1 {
			return errors.New("cooking success lacks full matched native delivery receipts")
		}
		receipt = &FoodReceipt{TaskID: last.TaskID, Action: "cook", Entity: p.Entity, Item: cooked[f.Item], Count: count, Evidence: evidence}
	default:
		return errors.New("unknown pending food action")
	}
	return r.t.Store.update(g.ID, func(g *Record) error {
		g.Pending = nil
		g.Last = &out
		g.History = append(g.History, out)
		if len(g.History) > 16 {
			g.History = g.History[len(g.History)-16:]
		}
		if receipt != nil {
			receipt.Run = p.Run
			g.Food.Receipts = append(g.Food.Receipts, *receipt)
		}
		if slices.Contains([]string{"ready", "raw", "wood", "stock_probe", "stock_deposit"}, p.Template) {
			for i := range g.Food.Chests {
				c := &g.Food.Chests[i]
				if c.Entity != p.Entity {
					continue
				}
				if observed == nil {
					c.Unavailable = true
					continue
				}
				c.Observed = true
				c.ObservedAt, c.Run = v.Now, r.t.Store.runID
				c.Contents = map[string]int{}
				for _, item := range observed.Items {
					if slices.Contains(meals, item.Prefab) || slices.Contains(ingredients, item.Prefab) || item.Prefab == "Wood" {
						c.Contents[item.Prefab] += item.Stack
					}
				}
			}
		}
		if p.Template == "stock_deposit" && !success {
			g.Food.Tried = append(g.Food.Tried, "stock_deposit:"+p.Entity+":"+f.Item)
		}
		if p.Template == "cook_return" && g.Food.Stock != nil {
			g.Food.Stock.ReturnAfterCooking = false
		}
		if p.Template == "cook" {
			if g.Food.Stock != nil {
				g.Food.Stock.ReturnAfterCooking = true
				g.Food.Stock.Delivering = true
			}
			g.Food.Batches++
			g.Food.Phase = "raw"
		}
		if p.Template == "food_return" && g.Food.Acquisition != nil {
			g.Food.Acquisition.Returning = false
		}
		return nil
	})
}

// Continuation runs only after the host earned readiness and under its old lease.
func (r *runner) Continuation(v behaviour.View) (*session.Order, error) {
	g, ok := r.t.Store.Get(r.id)
	if !ok || g.Kind != "food_resupply" || g.Status != "succeeded" || g.Food.Escort == nil || !slices.Contains(v.Owners, g.Food.Escort.Player) {
		return nil, nil
	}
	if g.Food.Continuation != nil {
		return nil, errors.New("uncertain escort continuation is never repeated")
	}
	if g.TaskCount >= g.MaxTasks {
		return nil, errors.New("escort continuation task budget exhausted")
	}
	a := *g.Food.Escort
	a.Protect = slices.Clone(v.Owners)
	raw, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	if err = r.t.Store.update(g.ID, func(g *Record) error {
		g.TaskCount++
		g.Food.Continuation = &Intent{Run: r.t.Store.runID, At: v.Now, Template: "escort", Params: raw, Entity: a.Player}
		return nil
	}); err != nil {
		return nil, err
	}
	return &session.Order{Task: protocol.TaskEscort, Args: &a}, nil
}

func (r *runner) Continued(id string) error {
	return r.t.Store.update(r.id, func(g *Record) error {
		if g.Food == nil || g.Food.Continuation == nil {
			return errors.New("missing escort continuation intent")
		}
		g.Food.Continuation.TaskID = id
		return nil
	})
}
