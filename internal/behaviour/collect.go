package behaviour

import (
	"encoding/json"
	"errors"
	"math"
	"slices"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// Collect is an explicit, bounded site visit. It does not take over follow,
// infer work sites, swing tools, or install default background loot jobs.
type Collect struct{}
type CollectParams struct {
	Site              Place          `json:"site"`
	Items             []string       `json:"items"`
	To                []string       `json:"to"`
	Keep              map[string]int `json:"keep,omitempty"`
	MaxTargets        int            `json:"maxTargets"`
	MaxWeightFraction float64        `json:"maxWeightFraction,omitempty"`
	Drops             bool           `json:"drops,omitempty"`
	Return            bool           `json:"return,omitempty"`
}

func (p CollectParams) stow() StowParams {
	return StowParams{World: p.Site.World, Items: p.Items, To: p.To, Keep: p.Keep}
}
func (Collect) Validate(raw json.RawMessage) error {
	var p CollectParams
	if err := Decode(raw, &p); err != nil {
		return err
	}
	if err := p.stow().validate(); err != nil {
		return err
	}
	if err := (Conditions{Near: &p.Site}).validate(); err != nil {
		return err
	}
	if p.Site.Radius > 40 || p.MaxTargets < 1 || p.MaxTargets > 64 {
		return errors.New("collect needs a site radius up to 40 m and 1–64 targets")
	}
	// Exact output prefab names avoid an ambiguous mapping from scenery or
	// item groups to a pickable. Stow supports groups independently.
	for _, name := range p.Items {
		if slices.Contains([]string{"*", "wood", "ore", "food"}, name) {
			return errors.New("collect needs exact item prefabs")
		}
	}
	if p.MaxWeightFraction != 0 && (!finite(p.MaxWeightFraction) || p.MaxWeightFraction < .1 || p.MaxWeightFraction > .9) {
		return errors.New("collect weight fraction must be between 0.1 and 0.9")
	}
	return nil
}
func (t Collect) Start(raw json.RawMessage) (Runner, error) {
	if err := t.Validate(raw); err != nil {
		return nil, err
	}
	var p CollectParams
	_ = Decode(raw, &p)
	if p.MaxWeightFraction == 0 {
		p.MaxWeightFraction = .8
	}
	return &collectRun{p: p, seen: map[string]bool{}}, nil
}
func freshPerception(v View) bool {
	return v.State != nil && v.Perception != nil && v.State.T-v.Perception.T >= 0 && v.State.T-v.Perception.T <= 5
}

type collectRun struct {
	p                            CollectParams
	arrived, returning, finished bool
	seen                         map[string]bool
	targets                      int
	task                         string
	beforeSeq                    uint64
	stepSeq                      uint64
	beforeCount                  int
	yield                        *protocol.ResourceYield
	targetPos                    protocol.Vec3
	picked                       bool
	pickScan                     uint64
	stow                         *stowRun
}

func (r *collectRun) Recover(result Result) bool {
	if r.stow != nil {
		return r.stow.Recover(result)
	}
	if result.Status != protocol.StatusFailed {
		return false
	}
	// Pickable.Interact can return false even when its RPC succeeded; the
	// next live perception/inventory, rather than that bool, proves collection.
	return r.task == protocol.TaskInteract && slices.Contains([]string{protocol.ReasonRefused, protocol.ReasonTargetLost}, result.Reason) ||
		r.task == protocol.TaskPickup && result.Reason == protocol.ReasonTargetLost
}
func (r *collectRun) fits(v View, y *protocol.ResourceYield) bool {
	if y == nil || !slices.Contains(r.p.Items, y.Prefab) || y.Count < 1 || y.Count > 10000 || y.StackMax < 1 || !finite(y.Weight) || y.Weight < 0 || !finite(v.State.Weight) || !finite(v.State.WeightMax) || v.State.WeightMax <= 0 {
		return false
	}
	weight := 0.0
	for _, it := range v.Inventory.Items {
		if !finite(it.Weight) || it.Weight < 0 || it.Stack < 0 {
			return false
		}
		weight += it.Weight * float64(it.Stack)
	}
	weight = math.Max(weight, v.State.Weight)
	// Reserve whole empty slots: matching-prefab stacks can have incompatible
	// quality/world metadata, so their room is only an optimistic upper bound.
	slots := int(math.Ceil(float64(y.Count) / float64(y.StackMax)))
	return finite(weight) && weight+float64(y.Count)*y.Weight <= v.State.WeightMax*r.p.MaxWeightFraction && v.Inventory.Free() >= slots
}
func (r *collectRun) Next(v View, result *Result) (*session.Order, error) {
	if v.State == nil || v.State.World != r.p.Site.World {
		return nil, errors.New("collect world changed")
	}
	if r.finished {
		return nil, nil
	}
	if v.Inventory == nil {
		return nil, ErrWait
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
	if !r.arrived {
		r.arrived = true
		if distance(v.State.Pos, r.p.Site.Pos) > r.p.Site.Radius {
			pos := r.p.Site.Pos
			return &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}}, TimeoutS: 90}, nil
		}
	}
	if r.task != "" {
		if result == nil {
			return nil, errors.New("missing collection result")
		}
		if r.task == protocol.TaskInteract {
			if result.Reason != protocol.ReasonTargetLost {
				r.picked = true
				r.pickScan = v.PerceptionSeq()
			}
		} else if result.Status == protocol.StatusSucceeded && v.Inventory.Seq <= r.stepSeq {
			return nil, ErrWait
		}
		r.task = ""
	}
	if !freshPerception(v) {
		return nil, ErrWait
	}
	// After using a pickable, wait for its products or a measured inventory
	// increase. Its animation/RPC result alone is not a collection receipt.
	if r.picked {
		for _, drop := range v.Perception.Resources {
			if r.targets < 128 && drop.Kind == protocol.ResourceDrop && !r.seen[drop.ID] && drop.Yield != nil && drop.Yield.Prefab == r.yield.Prefab && distance(drop.Pos, r.targetPos) <= 4 && distance(drop.Pos, r.p.Site.Pos) <= r.p.Site.Radius && r.fits(v, drop.Yield) {
				return r.take(v, drop), nil
			}
		}
		if v.Perception.Seq <= r.pickScan || v.Inventory.Seq <= r.beforeSeq || v.Inventory.Count(r.yield.Prefab) <= r.beforeCount {
			return nil, ErrWait
		}
		r.picked = false
	}
	if r.targets < r.p.MaxTargets {
		var best *protocol.PerceivedResource
		for _, target := range v.Perception.Resources {
			if target.ID == "" || r.seen[target.ID] || distance(target.Pos, r.p.Site.Pos) > r.p.Site.Radius || !r.fits(v, target.Yield) {
				continue
			}
			if target.Kind != protocol.ResourcePickable && !(r.p.Drops && target.Kind == protocol.ResourceDrop) {
				continue
			}
			if best == nil || distance(v.State.Pos, target.Pos) < distance(v.State.Pos, best.Pos) || distance(v.State.Pos, target.Pos) == distance(v.State.Pos, best.Pos) && target.ID < best.ID {
				copy := target
				best = &copy
			}
		}
		if best != nil {
			return r.take(v, *best), nil
		}
	}
	r.stow = &stowRun{p: r.p.stow()}
	return r.Next(v, nil)
}
func (v View) PerceptionSeq() uint64 {
	if v.Perception == nil {
		return 0
	}
	return v.Perception.Seq
}
func (r *collectRun) take(v View, target protocol.PerceivedResource) *session.Order {
	r.seen[target.ID] = true
	r.targets++
	r.stepSeq = v.Inventory.Seq
	if !r.picked {
		r.yield = target.Yield
		r.targetPos = target.Pos
		r.beforeCount = v.Inventory.Count(target.Yield.Prefab)
		r.beforeSeq = v.Inventory.Seq
	}
	r.task = protocol.TaskPickup
	var args any = &protocol.PickupArgs{Entity: target.ID}
	if target.Kind == protocol.ResourcePickable {
		r.task = protocol.TaskInteract
		args = &protocol.InteractArgs{Entity: target.ID}
	}
	return &session.Order{Task: r.task, Args: args, TimeoutS: 60}
}
