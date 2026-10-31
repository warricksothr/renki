package behaviour

import (
	"encoding/json"
	"errors"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// MarkerFood does small local jobs. It never hunts, withdraws supplies or
// travels between sites; those decisions belong to explicit production goals.
type MarkerFood struct{}
type MarkerFoodParams struct {
	Marker Marker `json:"marker"`
}

var wildMeals = []string{"Raspberry", "Blueberries", "Mushroom"}
var readyMeals = []string{"CookedMeat", "CookedDeerMeat", "NeckTailGrilled", "Honey", "Raspberry", "Blueberries", "Mushroom", "MushroomYellow"}
var rawMeals = []string{"RawMeat", "DeerMeat", "NeckTail"}
var cookedMeals = map[string]string{"RawMeat": "CookedMeat", "DeerMeat": "CookedDeerMeat", "NeckTail": "NeckTailGrilled"}

func (MarkerFood) Validate(raw json.RawMessage) error {
	var p MarkerFoodParams
	if err := Decode(raw, &p); err != nil {
		return err
	}
	if err := p.Marker.validate(); err != nil {
		return err
	}
	if !slices.Contains([]string{"berries", "cooking", "storage"}, p.Marker.Kind) {
		return errors.New("food work supports berries, cooking and storage markers")
	}
	return nil
}
func (t MarkerFood) Start(raw json.RawMessage) (Runner, error) {
	if err := t.Validate(raw); err != nil {
		return nil, err
	}
	var p MarkerFoodParams
	_ = Decode(raw, &p)
	return &markerFoodRun{p: p}, nil
}

// MarkerFoodDefinition gives the panel and tooling the same conservative preset.
func MarkerFoodDefinition(marker Marker) Definition {
	raw, _ := json.Marshal(MarkerFoodParams{Marker: marker})
	return Definition{Template: "marker_food", Params: raw, Trigger: Trigger{Kind: "idle", Marker: &marker, DwellS: 30}, Repeat: Repeat{Mode: "trigger", CooldownS: 300, MaxRuns: 12}, Enabled: true, Priority: -10}
}

type markerFoodRun struct {
	p                       MarkerFoodParams
	marker                  *home.Waypoint
	began                   time.Time
	health                  healthSample
	collect                 *collectRun
	stow                    *stowRun
	returning               bool
	cooking                 string
	station                 string
	packSeq, stateSeq       uint64
	beforeRaw, beforeCooked int
	beforeFood              float64
	batches                 int
}

func (r *markerFoodRun) Check(v View) error {
	// State and perception stream independently. A new scan can be a few
	// milliseconds ahead of the latest state without being future-dated.
	if !idleFresh(v) || v.Perception == nil || math.Abs(v.State.T-v.Perception.T) > 3 || v.PerceptionAt.IsZero() || v.Now.Sub(v.PerceptionAt) < 0 || v.Now.Sub(v.PerceptionAt) > 3*time.Second || v.Inventory == nil {
		return errors.New("food work requires fresh state, inventory and surroundings")
	}
	w, ok := r.p.Marker.resolve(v)
	if !ok || !withinMarker(v.State.Pos, w) || r.marker != nil && w != *r.marker {
		return errors.New("food marker changed or left")
	}
	s := v.State
	fraction := .8
	if r.marker != nil {
		fraction = .4
	} // A new meal can raise max HP before healing; actual injury is checked below.
	if !finite(s.HP) || !finite(s.HPMax) || s.HPMax <= 0 || s.HP < math.Max(25, fraction*s.HPMax) || s.Stranded() || s.Encumbered() || slices.Contains(s.Flags, protocol.FlagSwimming) || s.Location != nil && s.Location.InWater {
		return errors.New("food work needs healthy dry footing and carry capacity")
	}
	fed := false
	if s.Food != nil {
		for _, f := range s.Food.Active {
			if f.RemainingS >= 300 {
				fed = true
			}
		}
	}
	if !fed {
		return errors.New("food work needs an active meal; resupply first")
	}
	for _, c := range v.Perception.Characters {
		if c.HPFrac > 0 && (c.TargetingMe || c.Hostile && distance(c.Pos, s.Pos) < 20) {
			return errors.New("nearby threat interrupts food work")
		}
	}
	if r.marker != nil && (r.health.lost(sampleHealth(v)) || v.Now.Sub(r.began) > 6*time.Minute) {
		return errors.New("food work injured or time budget exhausted")
	}
	return nil
}
func (r *markerFoodRun) Recover(result Result) bool {
	return r.collect != nil && r.collect.Recover(result) || r.stow != nil && r.stow.Recover(result)
}
func (r *markerFoodRun) Next(v View, result *Result) (*session.Order, error) {
	if err := r.Check(v); err != nil {
		return nil, err
	}
	if r.marker == nil {
		w, _ := r.p.Marker.resolve(v)
		r.marker, r.began, r.health = &w, v.Now, sampleHealth(v)
	}
	switch r.p.Marker.Kind {
	case "berries":
		return r.forage(v, result)
	case "storage":
		return r.store(v, result)
	default:
		return r.cook(v, result)
	}
}
func (r *markerFoodRun) forage(v View, result *Result) (*session.Order, error) {
	if r.returning {
		return nil, nil
	}
	if r.collect == nil {
		r.collect = &collectRun{p: CollectParams{Site: Place{World: r.p.Marker.World, Pos: r.marker.Pos, Radius: markerRadius(*r.marker)}, Items: wildMeals, MaxTargets: 4, MaxWeightFraction: .75, Drops: true, CarryOnly: true}, seen: map[string]bool{}}
	}
	// Read memory as a hint only: collect only currently perceived safe food.
	copy := *v.Perception
	copy.Resources = nil
	for _, target := range v.Perception.Resources {
		if target.Yield == nil || !withinMarker(target.Pos, *r.marker) || v.Inventory.Count(target.Yield.Prefab)+target.Yield.Count > 20 {
			continue
		}
		safe := true
		for _, c := range v.Perception.Characters {
			if c.Kind == protocol.CharacterCreature && c.HPFrac > 0 && distance(c.Pos, target.Pos) < 15 {
				safe = false
			}
		}
		if safe {
			copy.Resources = append(copy.Resources, target)
		}
	}
	v.Perception = &copy
	order, err := r.collect.Next(v, result)
	if order != nil || err != nil {
		return order, err
	}
	if r.collect.targets == 0 {
		return nil, errors.New("no eligible visible food below the 20-item stock limit")
	}
	if distance(v.State.Pos, r.marker.Pos) > 2 {
		r.returning = true
		pos := r.marker.Pos
		return &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}, Dry: true, AvoidHostiles: true}, TimeoutS: 60}, nil
	}
	return nil, nil
}
func (r *markerFoodRun) store(v View, result *Result) (*session.Order, error) {
	if r.stow == nil {
		ids := []string{}
		// Restrict storage to physically observed nearby chests. Remembered
		// contents alone cannot turn an unrelated marker into a destination.
		for _, chest := range v.Perception.Structures {
			if chest.Kind == protocol.StructureContainer && chest.ID != "" && withinMarker(chest.Pos, *r.marker) && distance(chest.Pos, v.State.Pos) <= 4 {
				ids = append(ids, chest.ID)
			}
		}
		sort.Strings(ids)
		ids = slices.Compact(ids)
		if len(ids) == 0 {
			return nil, errors.New("no currently visible chest within reach of storage marker")
		}
		if len(ids) > 16 {
			ids = ids[:16]
		}
		keep := map[string]int{}
		for _, p := range readyMeals {
			keep[p] = 2
		}
		p := StowParams{World: r.p.Marker.World, Items: readyMeals, To: ids, Keep: keep, Overflow: "leave"}
		if len(stowCounts(v.Inventory, p)) == 0 {
			return nil, errors.New("no surplus food above the two-per-kind personal reserve")
		}
		r.stow = &stowRun{p: p, dry: true}
	}
	return r.stow.Next(v, result)
}
func (r *markerFoodRun) cook(v View, result *Result) (*session.Order, error) {
	if r.cooking != "" {
		if result == nil || result.Status != protocol.StatusSucceeded || result.TaskID == "" {
			return nil, errors.New("missing successful cooking result; partial batch will not be retried")
		}
		count, _ := CookingDelivery(result.Cooking, result.TaskID, cookedMeals[r.cooking], r.station)
		if count != 1 {
			return nil, errors.New("cooking lacks matched removal, output, pickup and inventory receipts")
		}
		if v.Inventory.Seq <= r.packSeq || v.State.Seq <= r.stateSeq {
			return nil, ErrWait
		}
		gain := v.Inventory.Count(cookedMeals[r.cooking]) - r.beforeCooked
		if activeFoodRemaining(v, cookedMeals[r.cooking]) > r.beforeFood+5 {
			gain++
		}
		if gain < count || r.beforeRaw-v.Inventory.Count(r.cooking) < 1 {
			return nil, errors.New("cooking inventory delta did not match delivery")
		}
		r.batches++
		r.cooking = ""
	}
	if r.batches >= 3 {
		return nil, nil
	}
	raw := ""
	for _, p := range rawMeals {
		if v.Inventory.Count(p) > 0 && v.Inventory.Count(cookedMeals[p]) < 20 {
			raw = p
			break
		}
	}
	if raw == "" {
		if r.batches == 0 {
			return nil, errors.New("no carried raw food below the 20-item cooked stock limit")
		}
		return nil, nil
	}
	id := ""
	best := math.Inf(1)
	for _, station := range v.Perception.Structures {
		c := station.Cooking
		if station.ID == "" || c == nil || !c.Supported || !c.Heated || !withinMarker(station.Pos, *r.marker) || distance(station.Pos, v.State.Pos) > 4 || len(c.Slots) == 0 || slices.ContainsFunc(c.Slots, func(s protocol.CookingSlotStatus) bool { return s.State != "empty" || s.Item != "" }) {
			continue
		}
		d := distance(station.Pos, v.State.Pos)
		if d < best || d == best && station.ID < id {
			id, best = station.ID, d
		}
	}
	if id == "" {
		return nil, errors.New("no heated empty supported cooker within reach of marker")
	}
	r.cooking, r.station, r.packSeq, r.stateSeq = raw, id, v.Inventory.Seq, v.State.Seq
	r.beforeRaw, r.beforeCooked = v.Inventory.Count(raw), v.Inventory.Count(cookedMeals[raw])
	r.beforeFood = activeFoodRemaining(v, cookedMeals[raw])
	return &session.Order{Task: protocol.TaskInteract, Args: &protocol.InteractArgs{Entity: id, Cook: raw}, TimeoutS: 180}, nil
}
