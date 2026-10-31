package behaviour

import (
	"encoding/json"
	"errors"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// Gather is bounded work at one explicit beech site.
// Native harvest owns each approach, strike, escape and collection phase.
type Gather struct{}
type GatherParams struct {
	Site       Place    `json:"site"`
	To         []string `json:"to"`
	Return     bool     `json:"return,omitempty"`
	MaxTrees   int      `json:"maxTrees,omitempty"`
	ReturnHome bool     `json:"returnHome,omitempty"`
	Defend     bool     `json:"defend,omitempty"`
}

func (p GatherParams) treeLimit() int { return max(1, p.MaxTrees) }
func (p GatherParams) workLimit() time.Duration {
	return time.Duration(p.treeLimit()) * 8 * time.Minute
}

var errGatherCapacity = errors.New("gather needs four free slots and 60 spare carry weight")
var errGatherTool = errors.New("gather needs a usable carried axe")

var beechProducts = []string{"Wood", "BeechSeeds", "Resin", "Feathers"}

func (Gather) Validate(raw json.RawMessage) error {
	var p GatherParams
	if err := Decode(raw, &p); err != nil {
		return err
	}
	if err := (Conditions{Near: &p.Site}).validate(); err != nil {
		return err
	}
	if p.Site.Radius > 40 {
		return errors.New("gather needs a site radius up to 40 m")
	}
	if p.MaxTrees < 0 || p.MaxTrees > 3 || p.Return && p.ReturnHome {
		return errors.New("gather needs maxTrees between one and three and one return destination")
	}
	return (StowParams{World: p.Site.World, Items: beechProducts, To: p.To}).validate()
}
func (t Gather) Start(raw json.RawMessage) (Runner, error) {
	if err := t.Validate(raw); err != nil {
		return nil, err
	}
	var p GatherParams
	_ = Decode(raw, &p)
	return &gatherRun{p: p, baseline: map[string]bool{}, completed: map[string]bool{}}, nil
}

type gatherRun struct {
	p                                         GatherParams
	began, settleUntil                        time.Time
	restUntil                                 time.Time
	arrived, returning, finished              bool
	baseline, completed                       map[string]bool
	chain                                     map[string]bool
	keep                                      map[string]int
	center                                    protocol.Vec3
	pending                                   *protocol.PerceivedResource
	beforePack, beforeScan                    uint64
	previousHealth                            healthSample
	targets, logs                             int
	trees, chainTargets, chainLogs, chainWood int
	treeOpen                                  bool
	home                                      *protocol.Vec3
	homePhase                                 string
	homeReceipt                               *uint64
	repairHomeScan                            uint64
	stow                                      *stowRun
	prepared, repairPending                   bool
	repairBefore                              uint64
	defender                                  string
	defenses                                  int
	defenseReceipt                            *gatherDefenseReceipt
	resume                                    *protocol.PerceivedResource
	clearingWildlife                          bool
	wildlifeAttempts                          int
	wildlifeUntil                             time.Time
	wildlifeObserveUntil                      time.Time
	wildlifeScan                              uint64
	pursuerWaiting                            string
	pursuerUntil                              time.Time
}

func (r *gatherRun) Recover(result Result) bool {
	// Storage may try its next chest. Opted-in harvest refusal may defend,
	// then retry the same resource; it never counts as a completed cut.
	return r.stow != nil && r.stow.Recover(result) || r.p.Defend && r.pending != nil &&
		result.Entity == r.pending.ID && result.Status == protocol.StatusFailed && result.Reason == protocol.ReasonRefused
}

func beechLog(resource protocol.PerceivedResource) bool {
	return resource.Kind == protocol.ResourceLog && strings.HasPrefix(strings.ToLower(resource.Prefab), "beech")
}

func gatherClear(v View, resource protocol.PerceivedResource) bool {
	return gatherClearance(v, resource, false)
}

func passiveDeer(c protocol.PerceivedCharacter) bool {
	return c.Kind == protocol.CharacterCreature && c.Prefab == "Deer" && !c.TargetingMe
}

func gatherClearance(v View, resource protocol.PerceivedResource, withoutDeer bool) bool {
	radius := 8.0
	if resource.Kind == protocol.ResourceTree {
		radius = 32
	}
	for _, c := range v.Perception.Characters {
		if withoutDeer && passiveDeer(c) {
			continue
		}
		clearance := radius
		if c.Kind == protocol.CharacterCreature && c.Hostile && c.Prefab == "Greyling" {
			clearance = math.Max(clearance, 40)
		}
		if c.HPFrac > 0 && (distance(c.Pos, v.State.Pos) < 15 || distance(c.Pos, resource.Pos) < clearance) {
			return false
		}
	}
	for _, piece := range v.Perception.Structures {
		if distance(piece.Pos, resource.Pos) < radius {
			return false
		}
	}
	// Solid walls have their own layout snapshot and do not appear among
	// interactable structures. Include their full segments, with clearance
	// for wall thickness, before asking native harvest to approach a tree.
	for _, wall := range v.Walls {
		dx, dz := wall.B[0]-wall.A[0], wall.B[2]-wall.A[2]
		t := 0.0
		if length := dx*dx + dz*dz; length > 0 {
			t = math.Max(0, math.Min(1, ((resource.Pos[0]-wall.A[0])*dx+(resource.Pos[2]-wall.A[2])*dz)/length))
		}
		foot := protocol.Vec3{wall.A[0] + t*dx, wall.A[1] + t*(wall.B[1]-wall.A[1]), wall.A[2] + t*dz}
		foot[1] = math.Max(foot[1], math.Min(foot[1]+wall.H, resource.Pos[1]))
		if distance(foot, resource.Pos) < radius+1 {
			return false
		}
	}
	return true
}

func gatherReady(v View) error {
	for _, character := range v.Perception.Characters {
		if character.HPFrac > 0 && distance(character.Pos, v.State.Pos) < 15 {
			return errors.New("a character is too close to start harvesting")
		}
	}
	return gatherSupplied(v)
}

func gatherTravelReady(v View) error {
	for _, c := range v.Perception.Characters {
		if c.HPFrac > 0 && !passiveDeer(c) && (c.Hostile || c.TargetingMe) && distance(c.Pos, v.State.Pos) < 15 {
			return errors.New("a threat is too close to prepare or travel to the gathering site")
		}
	}
	return gatherSupplied(v)
}

func gatherSupplied(v View) error {
	s := v.State
	if !finite(s.HP) || !finite(s.HPMax) || s.HPMax <= 0 || s.HP < s.HPMax*.8 ||
		s.Stranded() || s.Encumbered() || slices.Contains(s.Flags, "swimming") || s.Location != nil && s.Location.InWater {
		return errors.New("gather needs healthy, dry, unburdened footing")
	}
	if s.Food == nil || len(s.Food.Active) == 0 {
		return errors.New("eat before gathering")
	}
	packWeight := 0.0
	for _, item := range v.Inventory.Items {
		if !finite(item.Weight) || item.Weight < 0 || item.Stack < 0 {
			return errors.New("gather needs known inventory weight")
		}
		packWeight += item.Weight * float64(item.Stack)
	}
	if !finite(s.Weight) || !finite(s.WeightMax) || !finite(packWeight) || s.WeightMax-math.Max(s.Weight, packWeight) < 60 || v.Inventory.Free() < 4 {
		return errGatherCapacity
	}
	if gatherAxe(v, .25) {
		return nil
	}
	return errGatherTool
}

func gatherAxe(v View, fraction float64) bool {
	for _, item := range v.Inventory.Items {
		if slices.Contains([]string{"AxeStone", "AxeFlint", "AxeBronze", "AxeIron", "AxeBlackMetal"}, item.Prefab) &&
			item.Stack > 0 && finite(item.Durability) && finite(item.DurabilityMax) && item.DurabilityMax > 0 && item.Durability >= item.DurabilityMax*fraction {
			return true
		}
	}
	return false
}

func (r *gatherRun) Next(v View, result *Result) (*session.Order, error) {
	if v.State == nil || v.State.Phase != protocol.PhaseInWorld || v.State.World != r.p.Site.World {
		return nil, errors.New("gather world is no longer current")
	}
	if r.finished {
		return nil, nil
	}
	if r.began.IsZero() {
		if r.p.ReturnHome {
			if v.Home == nil || distance(v.State.Pos, *v.Home) > 150 {
				return nil, errors.New("gather needs a known home within 150 m")
			}
			for _, n := range *v.Home {
				if !finite(n) {
					return nil, errors.New("gather home is invalid")
				}
			}
			home := *v.Home
			r.home = &home
		}
		r.began = v.Now
		r.previousHealth = sampleHealth(v)
	}
	if r.previousHealth.lost(sampleHealth(v)) {
		return nil, errors.New("observed health loss; gathering stopped")
	}
	r.previousHealth = sampleHealth(v)
	if v.Now.Sub(r.began) >= r.p.workLimit() {
		return nil, errors.New("gather reached its bounded work limit")
	}
	if r.defender != "" {
		if err := r.confirmDefense(v, result); err != nil {
			return nil, err
		}
		result = nil
	}
	if r.homePhase != "" {
		if result == nil || result.Status != protocol.StatusSucceeded {
			return nil, errors.New("gather home return lacks native success")
		}
		// Wait for a state strictly after observing the receipt. A task
		// response can arrive after the pose used by this engine tick.
		if r.homeReceipt == nil {
			seq := v.State.Seq
			r.homeReceipt = &seq
			if v.Perception != nil {
				r.repairHomeScan = v.Perception.Seq
			}
			return nil, ErrWait
		}
		if v.State.Seq <= *r.homeReceipt {
			return nil, ErrWait
		}
		if r.homePhase == "repair" && (!freshPerception(v) || v.Perception.Seq <= r.repairHomeScan) {
			return nil, ErrWait
		}
		if math.Hypot(v.State.Pos[0]-r.home[0], v.State.Pos[2]-r.home[2]) > 3 || math.Abs(v.State.Pos[1]-r.home[1]) > 1.75 {
			return nil, errors.New("gather native return was not confirmed on the home floor")
		}
		phase := r.homePhase
		r.homePhase = ""
		r.homeReceipt = nil
		if phase == "finish" {
			r.finished = true
			return nil, nil
		}
		if phase == "repair" {
			r.prepared = false
		} else {
			r.startStow()
		}
		result = nil
	}
	if r.stow != nil {
		order, err := r.stow.Next(v, result)
		if order != nil || err != nil {
			return order, err
		}
		r.stow = nil
		if r.p.ReturnHome {
			return r.goHome("finish"), nil
		}
		if r.p.Return {
			r.returning = true
			pos := r.p.Site.Pos
			return &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}, Dry: true, AvoidHostiles: true}, TimeoutS: 90}, nil
		}
		r.finished = true
		return nil, nil
	}
	if r.returning {
		r.finished = true
		return nil, nil
	}
	if v.Inventory == nil || !freshPerception(v) {
		return nil, ErrWait
	}
	if r.keep == nil {
		r.keep = map[string]int{}
		for _, prefab := range beechProducts {
			r.keep[prefab] = v.Inventory.Count(prefab)
		}
	}
	if r.repairPending {
		if result == nil || result.Status != protocol.StatusSucceeded {
			return nil, errors.New("gather station repair did not succeed")
		}
		if v.Inventory.Seq <= r.repairBefore {
			return nil, ErrWait
		}
		if !gatherAxe(v, .9) {
			return nil, errors.New("station did not restore a usable axe to 90% durability")
		}
		r.repairPending = false
	}
	if !r.prepared {
		r.prepared = true
		if r.p.ReturnHome && gatherAxe(v, 0) && !gatherAxe(v, .9) {
			if err := gatherTravelReady(v); err != nil && !errors.Is(err, errGatherTool) {
				return nil, err
			}
			var station *protocol.PerceivedStructure
			for _, s := range v.Perception.Structures {
				if s.Kind == protocol.StructureStation && s.Prefab == "piece_workbench" && s.ID != "" &&
					distance(s.Pos, *r.home) <= 10 && distance(v.State.Pos, s.Pos) <= 12 &&
					(station == nil || distance(v.State.Pos, s.Pos) < distance(v.State.Pos, station.Pos)) {
					copy := s
					station = &copy
				}
			}
			if station == nil {
				if math.Hypot(v.State.Pos[0]-r.home[0], v.State.Pos[2]-r.home[2]) > 3 || math.Abs(v.State.Pos[1]-r.home[1]) > 1.75 {
					return r.goHome("repair"), nil
				}
				return nil, errors.New("repair the axe at an accessible home workbench before gathering")
			}
			r.repairPending = true
			r.repairBefore = v.Inventory.Seq
			return &session.Order{Task: protocol.TaskInteract, Args: &protocol.InteractArgs{Entity: station.ID, Repair: true}, TimeoutS: 90}, nil
		}
	}
	if !r.arrived {
		if err := gatherTravelReady(v); err != nil {
			return nil, err
		}
		r.arrived = true
		// Being inside a broad work zone does not put its resources within
		// the native harvest's 18 m acquisition range. Approach the work
		// point before choosing a tree, even from the edge of that zone.
		if distance(v.State.Pos, r.p.Site.Pos) > math.Min(r.p.Site.Radius, 12) {
			pos := r.p.Site.Pos
			return &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}, Dry: true, AvoidHostiles: true}, TimeoutS: 90}, nil
		}
	}
	if r.pending != nil {
		if result == nil || result.Status != protocol.StatusSucceeded {
			if result != nil && r.Recover(*result) {
				order, err := r.defend(v)
				if order != nil {
					r.resume, r.pending = r.pending, nil
				}
				if order != nil || err != nil {
					return order, err
				}
			}
			return nil, errors.New("gather lacks a successful native harvest result")
		}
		if v.Inventory.Seq <= r.beforePack || v.Perception.Seq <= r.beforeScan {
			return nil, ErrWait
		}
		for _, resource := range v.Perception.Resources {
			if resource.ID == r.pending.ID {
				return nil, ErrWait
			}
		}
		// Bind newly observed descendants at a completed cut. Once bound,
		// rolling beyond the work-site circle must not erase unfinished work.
		for _, resource := range v.Perception.Resources {
			if beechLog(resource) && resource.ID != "" && !r.baseline[resource.ID] &&
				(distance(resource.Pos, r.center) <= 32 || distance(resource.Pos, r.pending.Pos) <= 24) {
				r.chain[resource.ID] = true
			}
		}
		r.completed[r.pending.ID] = true
		if beechLog(*r.pending) {
			r.logs++
		}
		r.pending = nil
		r.settleUntil = v.Now.Add(2 * time.Second)
	}
	if v.Now.Before(r.settleUntil) {
		return nil, ErrWait
	}
	if r.clearingWildlife {
		return r.clearWildlife(v, result)
	}
	if order, err := r.defend(v); order != nil || err != nil {
		return order, err
	}
	if r.resume != nil {
		for _, resource := range v.Perception.Resources {
			if resource.ID == r.resume.ID && resource.Prefab == r.resume.Prefab &&
				resource.Kind == r.resume.Kind && gatherClear(v, resource) {
				if err := gatherReady(v); err != nil {
					return nil, err
				}
				r.resume = nil
				return r.harvest(v, resource)
			}
		}
		return nil, errors.New("interrupted harvest target is no longer available and safe")
	}
	if r.treeOpen {
		unfinished := false
		for id := range r.chain {
			unfinished = unfinished || !r.completed[id]
		}
		if !unfinished {
			if r.logs-r.chainLogs < 2 || v.Inventory.Count("Wood") <= r.chainWood {
				return nil, errors.New("log splitting and collected Wood are not verified")
			}
			r.treeOpen = false
		}
	}
	if !r.treeOpen && r.trees < r.p.treeLimit() && r.p.workLimit()-v.Now.Sub(r.began) >= 4*time.Minute {
		if order, err := r.clearWildlife(v, result); order != nil || err != nil {
			return order, err
		}
	}
	if r.trees > 0 && !r.treeOpen {
		ready := gatherReady(v)
		pressure := errors.Is(ready, errGatherCapacity) || errors.Is(ready, errGatherTool)
		if r.trees >= r.p.treeLimit() || pressure || r.p.workLimit()-v.Now.Sub(r.began) < 4*time.Minute {
			return r.deliver(v)
		}
		if ready != nil {
			return nil, ready
		}
	}
	var best *protocol.PerceivedResource
	cost := func(pos protocol.Vec3) float64 {
		if !r.treeOpen {
			return targetCost(v, pos)
		}
		return distance(v.State.Pos, pos)
	}
	unfinished := false
	for id := range r.chain {
		unfinished = unfinished || !r.completed[id]
	}
	for _, resource := range v.Perception.Resources {
		if resource.ID == "" || r.completed[resource.ID] || resource.MinTool == nil || *resource.MinTool != 0 {
			continue
		}
		eligible := resource.Kind == protocol.ResourceTree && resource.Prefab == "Beech1"
		if r.treeOpen {
			eligible = beechLog(resource) && r.chain[resource.ID]
			unfinished = unfinished || eligible
		}
		if !eligible || !r.treeOpen && distance(resource.Pos, r.p.Site.Pos) > r.p.Site.Radius || r.treeOpen && distance(resource.Pos, r.center) > 60 || !gatherClear(v, resource) {
			continue
		}
		if best == nil || cost(resource.Pos) < cost(best.Pos) || cost(resource.Pos) == cost(best.Pos) && resource.ID < best.ID {
			copy := resource
			best = &copy
		}
	}
	if best == nil {
		if r.targets == 0 {
			return nil, errors.New("no nearby isolated tier-zero full beech at the site")
		}
		if unfinished {
			return nil, errors.New("new nearby logs remain unsafe or unavailable; gathering stopped")
		}
		return r.deliver(v)
	}
	if r.treeOpen && r.targets-r.chainTargets >= 9 {
		return nil, errors.New("gather reached its nine-resource chain limit; remaining logs left in place")
	}
	if err := gatherReady(v); err != nil {
		return nil, err
	}
	if distance(v.State.Pos, best.Pos) > 18 {
		pos := best.Pos
		if r.treeOpen {
			// A tilted trunk's network pivot can be high above its ground
			// contact. Travel toward its horizontal location, then let native
			// harvest find a supported physical strike point.
			pos[1] = v.State.Pos[1]
		}
		stop := 10.0
		return &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}, StopDistance: &stop, Dry: true, AvoidHostiles: true}, TimeoutS: 90}, nil
	}
	if !r.treeOpen {
		if !finite(v.State.Stamina) || v.State.Stamina < 40 {
			if r.trees > 0 && finite(v.State.Stamina) {
				if r.restUntil.IsZero() {
					r.restUntil = v.Now.Add(20 * time.Second)
				}
				if v.Now.Before(r.restUntil) {
					return nil, &Delay{Until: r.restUntil}
				}
				return r.deliver(v)
			}
			return nil, errors.New("gather needs 40 stamina before starting")
		}
		r.restUntil = time.Time{}
		r.baseline = map[string]bool{}
		r.chain = map[string]bool{}
		for _, resource := range v.Perception.Resources {
			r.baseline[resource.ID] = true
		}
		r.center = best.Pos
		r.treeOpen = true
		r.trees++
		r.chainLogs, r.chainTargets, r.chainWood = r.logs, r.targets, v.Inventory.Count("Wood")
	}
	return r.harvest(v, *best)
}

func (r *gatherRun) harvest(v View, resource protocol.PerceivedResource) (*session.Order, error) {
	if r.targets-r.chainTargets >= 9 {
		return nil, errors.New("gather reached its nine-resource chain limit")
	}
	remaining := math.Min(195, (r.p.workLimit() - v.Now.Sub(r.began)).Seconds())
	if remaining < 30 {
		return nil, errors.New("insufficient remaining gathering budget")
	}
	r.targets++
	r.pending, r.beforePack, r.beforeScan = &resource, v.Inventory.Seq, v.Perception.Seq
	collect := true
	return &session.Order{Task: protocol.TaskHarvest, Args: &protocol.HarvestArgs{Entity: resource.ID, Collect: &collect}, TimeoutS: remaining}, nil
}

func (r *gatherRun) startStow() {
	r.stow = &stowRun{p: StowParams{World: r.p.Site.World, Items: slices.Clone(beechProducts), To: r.p.To, Keep: r.keep}, dry: true}
}

func (r *gatherRun) deliver(v View) (*session.Order, error) {
	if r.p.ReturnHome {
		return r.goHome("storage"), nil
	}
	r.startStow()
	return r.Next(v, nil)
}

func (r *gatherRun) goHome(phase string) *session.Order {
	r.homePhase = phase
	r.homeReceipt = nil
	pos := *r.home
	stop := 2.0
	return &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}, StopDistance: &stop, Dry: true, AvoidHostiles: true}, TimeoutS: 120}
}
