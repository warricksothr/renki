package behaviour

import (
	"errors"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// A short ordinary running approach can make a passive deer leave a work site.
// It grants no harvest permission: all original exclusions must be clear afterward.
func (r *gatherRun) clearWildlife(v View, result *Result) (*session.Order, error) {
	if r.clearingWildlife {
		if result == nil || result.Status != protocol.StatusSucceeded {
			return nil, errors.New("wildlife clearance approach did not complete")
		}
		r.clearingWildlife = false
		r.wildlifeScan = v.Perception.Seq
		r.wildlifeUntil = v.Now.Add(20 * time.Second)
		return nil, ErrWait
	}
	if !r.wildlifeUntil.IsZero() && v.Perception.Seq <= r.wildlifeScan {
		if !v.Now.Before(r.wildlifeUntil) {
			return nil, errors.New("no fresh work-site observation after wildlife clearance")
		}
		return nil, ErrWait
	}
	// Delivery pressures and genuine threats retain their existing handling.
	if gatherSupplied(v) != nil {
		return nil, nil
	}
	for _, c := range v.Perception.Characters {
		if c.HPFrac > 0 && !passiveDeer(c) && distance(c.Pos, v.State.Pos) < 15 {
			return nil, nil
		}
	}
	var candidate *protocol.PerceivedResource
	for _, tree := range v.Perception.Resources {
		if tree.ID == "" || r.completed[tree.ID] || tree.Kind != protocol.ResourceTree || tree.Prefab != "Beech1" ||
			tree.MinTool == nil || *tree.MinTool != 0 || distance(tree.Pos, r.p.Site.Pos) > r.p.Site.Radius {
			continue
		}
		if gatherClear(v, tree) {
			r.wildlifeUntil = time.Time{}
			return nil, nil
		}
		if gatherClearance(v, tree, true) && (candidate == nil || targetCost(v, tree.Pos) < targetCost(v, candidate.Pos) ||
			targetCost(v, tree.Pos) == targetCost(v, candidate.Pos) && tree.ID < candidate.ID) {
			copy := tree
			candidate = &copy
		}
	}
	if candidate == nil {
		return nil, nil
	}
	if !r.wildlifeUntil.IsZero() {
		if v.Now.Before(r.wildlifeUntil) {
			return nil, ErrWait
		}
		return nil, errors.New("passive wildlife remains inside the falling-tree exclusion after clearance")
	}
	if r.wildlifeAttempts >= r.p.treeLimit() || !finite(v.State.Stamina) || v.State.Stamina < 40 {
		return nil, errors.New("no wildlife clearance budget or stamina remains")
	}
	var deer *protocol.PerceivedCharacter
	for _, c := range v.Perception.Characters {
		if !passiveDeer(c) || c.ID == "" || c.HPFrac <= 0 || distance(c.Pos, v.State.Pos) > 20 ||
			distance(c.Pos, r.p.Site.Pos) > 40 || distance(c.Pos, candidate.Pos) >= 32 && distance(c.Pos, v.State.Pos) >= 15 ||
			r.home != nil && distance(c.Pos, *r.home) > 150 {
			continue
		}
		if deer == nil || distance(c.Pos, v.State.Pos) < distance(deer.Pos, v.State.Pos) ||
			distance(c.Pos, v.State.Pos) == distance(deer.Pos, v.State.Pos) && c.ID < deer.ID {
			copy := c
			deer = &copy
		}
	}
	if deer == nil {
		return nil, errors.New("passive wildlife blocks the work site outside bounded approach range")
	}
	r.clearingWildlife = true
	r.wildlifeAttempts++
	pos, stop, run := deer.Pos, 8.0, true
	return &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos},
		StopDistance: &stop, Run: &run, Dry: true, AvoidHostiles: true}, TimeoutS: 30}, nil
}
