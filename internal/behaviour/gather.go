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

// Gather is one explicit beech worksite, not an unbounded harvesting loop.
// Native harvest owns each approach, strike, escape and collection phase.
type Gather struct{}
type GatherParams struct {
	Site   Place    `json:"site"`
	To     []string `json:"to"`
	Return bool     `json:"return,omitempty"`
}

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
	p                            GatherParams
	began, settleUntil           time.Time
	arrived, returning, finished bool
	baseline, completed          map[string]bool
	keep                         map[string]int
	center                       protocol.Vec3
	pending                      *protocol.PerceivedResource
	beforePack, beforeScan       uint64
	previousHealth               float64
	targets, logs                int
	stow                         *stowRun
}

func (r *gatherRun) Recover(result Result) bool {
	// Only the existing bounded storage fallback may consume a refusal.
	// Failed harvesting never advances to a different resource.
	return r.stow != nil && r.stow.Recover(result)
}

func beechLog(resource protocol.PerceivedResource) bool {
	return resource.Kind == protocol.ResourceLog && strings.HasPrefix(strings.ToLower(resource.Prefab), "beech")
}

func gatherClear(v View, resource protocol.PerceivedResource) bool {
	radius := 8.0
	if resource.Kind == protocol.ResourceTree {
		radius = 32
	}
	for _, c := range v.Perception.Characters {
		if c.HPFrac > 0 && (distance(c.Pos, v.State.Pos) < 15 || distance(c.Pos, resource.Pos) < radius) {
			return false
		}
	}
	for _, piece := range v.Perception.Structures {
		if distance(piece.Pos, resource.Pos) < radius {
			return false
		}
	}
	return true
}

func gatherReady(v View) error {
	s := v.State
	if !finite(s.HP) || !finite(s.HPMax) || s.HPMax <= 0 || s.HP < s.HPMax*.8 ||
		s.Stranded() || s.Encumbered() || slices.Contains(s.Flags, "swimming") || s.Location != nil && s.Location.InWater {
		return errors.New("gather needs healthy, dry, unburdened footing")
	}
	if s.Food == nil || len(s.Food.Active) == 0 {
		return errors.New("eat before gathering")
	}
	for _, character := range v.Perception.Characters {
		if character.HPFrac > 0 && distance(character.Pos, s.Pos) < 15 {
			return errors.New("a character is too close to start gathering or travel to its site")
		}
	}
	packWeight := 0.0
	for _, item := range v.Inventory.Items {
		if !finite(item.Weight) || item.Weight < 0 || item.Stack < 0 {
			return errors.New("gather needs known inventory weight")
		}
		packWeight += item.Weight * float64(item.Stack)
	}
	if !finite(s.Weight) || !finite(s.WeightMax) || !finite(packWeight) || s.WeightMax-math.Max(s.Weight, packWeight) < 60 || v.Inventory.Free() < 4 {
		return errors.New("gather needs four free slots and 60 spare carry weight")
	}
	for _, item := range v.Inventory.Items {
		if slices.Contains([]string{"AxeStone", "AxeFlint", "AxeBronze", "AxeIron", "AxeBlackMetal"}, item.Prefab) &&
			item.Stack > 0 && finite(item.Durability) && finite(item.DurabilityMax) && item.DurabilityMax > 0 && item.Durability >= item.DurabilityMax*.25 {
			return nil
		}
	}
	return errors.New("gather needs a usable carried axe")
}

func (r *gatherRun) Next(v View, result *Result) (*session.Order, error) {
	if v.State == nil || v.State.Phase != protocol.PhaseInWorld || v.State.World != r.p.Site.World {
		return nil, errors.New("gather world is no longer current")
	}
	if r.finished {
		return nil, nil
	}
	if r.began.IsZero() {
		r.began = v.Now
		r.previousHealth = v.State.HP
	}
	if v.State.HP < math.Min(r.previousHealth, v.State.HPMax)-.15 {
		return nil, errors.New("observed health loss; gathering stopped")
	}
	r.previousHealth = v.State.HP
	if v.Now.Sub(r.began) >= 8*time.Minute {
		return nil, errors.New("gather reached its eight-minute work limit")
	}
	if r.stow != nil {
		order, err := r.stow.Next(v, result)
		if order != nil || err != nil {
			return order, err
		}
		r.stow = nil
		if r.p.Return {
			r.returning = true
			pos := r.p.Site.Pos
			return &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}}, TimeoutS: 90}, nil
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
	if !r.arrived {
		if err := gatherReady(v); err != nil {
			return nil, err
		}
		r.arrived = true
		if distance(v.State.Pos, r.p.Site.Pos) > r.p.Site.Radius {
			pos := r.p.Site.Pos
			return &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}}, TimeoutS: 90}, nil
		}
	}
	if r.pending != nil {
		if result == nil || result.Status != protocol.StatusSucceeded {
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
	var best *protocol.PerceivedResource
	cost := func(pos protocol.Vec3) float64 {
		if r.targets == 0 {
			return targetCost(v, pos)
		}
		return distance(v.State.Pos, pos)
	}
	unfinished := false
	for _, resource := range v.Perception.Resources {
		if resource.ID == "" || r.completed[resource.ID] || resource.MinTool == nil || *resource.MinTool != 0 {
			continue
		}
		eligible := resource.Kind == protocol.ResourceTree && resource.Prefab == "Beech1"
		if r.targets > 0 {
			eligible = beechLog(resource) && !r.baseline[resource.ID] && distance(resource.Pos, r.center) <= 20
			unfinished = unfinished || eligible
		}
		if !eligible || distance(resource.Pos, r.p.Site.Pos) > r.p.Site.Radius || distance(resource.Pos, v.State.Pos) > 18 || !gatherClear(v, resource) {
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
		if r.logs < 2 || v.Inventory.Count("Wood") <= r.keep["Wood"] {
			return nil, errors.New("log splitting and collected Wood are not verified")
		}
		r.stow = &stowRun{p: StowParams{World: r.p.Site.World, Items: slices.Clone(beechProducts), To: r.p.To, Keep: r.keep}}
		return r.Next(v, nil)
	}
	if r.targets == 5 {
		return nil, errors.New("gather reached its five-resource limit; remaining logs left in place")
	}
	if err := gatherReady(v); err != nil {
		return nil, err
	}
	if r.targets == 0 {
		if !finite(v.State.Stamina) || v.State.Stamina < 40 {
			return nil, errors.New("gather needs 40 stamina before starting")
		}
		r.keep = map[string]int{}
		for _, prefab := range beechProducts {
			r.keep[prefab] = v.Inventory.Count(prefab)
		}
		for _, resource := range v.Perception.Resources {
			r.baseline[resource.ID] = true
		}
		r.center = best.Pos
	}
	remaining := math.Min(195, (8*time.Minute - v.Now.Sub(r.began)).Seconds())
	if remaining < 30 {
		return nil, errors.New("insufficient remaining gathering budget")
	}
	r.targets++
	r.pending, r.beforePack, r.beforeScan = best, v.Inventory.Seq, v.Perception.Seq
	collect := true
	return &session.Order{Task: protocol.TaskHarvest, Args: &protocol.HarvestArgs{Entity: best.ID, Collect: &collect}, TimeoutS: remaining}, nil
}
