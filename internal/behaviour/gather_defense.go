package behaviour

import (
	"errors"
	"slices"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

type gatherDefenseReceipt struct{ state, perception uint64 }

// Optional work-site defense preserves the current cut/chain and starting
// inventory. Native kiting owns the fighting ground, lure, strikes and escape.
// It never searches for an enemy or substitutes a different target.
func (r *gatherRun) defend(v View) (*session.Order, error) {
	if !r.p.Defend || !r.arrived || !freshPerception(v) {
		return nil, nil
	}
	var target *protocol.PerceivedCharacter
	for _, c := range v.Perception.Characters {
		if c.ID != "" && c.Kind == protocol.CharacterCreature && c.Prefab == "Greyling" && c.Stars == 0 &&
			c.Hostile && c.TargetingMe && c.HPFrac > 0 && distance(v.State.Pos, c.Pos) <= 20 {
			if target != nil {
				return nil, errors.New("multiple Greyling pursuers; gathering stopped")
			}
			copy := c
			target = &copy
		}
	}
	if target == nil {
		return nil, nil
	}
	if r.defenses >= 2 || r.p.workLimit()-v.Now.Sub(r.began) < 4*time.Minute {
		return nil, errors.New("gather has no bounded defense budget remaining")
	}
	s := v.State
	if !finite(s.Stamina) || s.Stamina < 40 || s.HPMax <= 0 || s.HP < s.HPMax*.8 ||
		s.Food == nil || len(s.Food.Active) == 0 || s.Encumbered() || s.Stranded() ||
		slices.Contains(s.Flags, "swimming") || s.Location != nil && s.Location.InWater || !gatherAxe(v, .25) {
		return nil, errors.New("cannot safely defend the work site; gathering stopped")
	}
	for _, c := range v.Perception.Characters {
		if c.ID == target.ID || c.HPFrac <= 0 {
			continue
		}
		if distance(c.Pos, s.Pos) < 8 || distance(c.Pos, target.Pos) < 8 ||
			c.Hostile && c.Kind == protocol.CharacterCreature && c.Prefab != "Deer" && distance(c.Pos, s.Pos) < 15 {
			return nil, errors.New("another character prevents work-site defense")
		}
	}
	r.defender = target.ID
	r.defenses++
	return &session.Order{Task: protocol.TaskAttack,
		Args: &protocol.AttackArgs{Entity: target.ID, Style: "melee", Kite: true}, TimeoutS: 180}, nil
}

func (r *gatherRun) confirmDefense(v View, result *Result) error {
	if result == nil || result.Status != protocol.StatusSucceeded || result.Entity != r.defender || result.Detail != "observed Greyling death" {
		return errors.New("work-site defense lacks exact native Greyling death proof")
	}
	if !freshPerception(v) {
		return ErrWait
	}
	if r.defenseReceipt == nil {
		r.defenseReceipt = &gatherDefenseReceipt{v.State.Seq, v.Perception.Seq}
		return ErrWait
	}
	if v.State.Seq <= r.defenseReceipt.state || v.Perception.Seq <= r.defenseReceipt.perception {
		return ErrWait
	}
	for _, c := range v.Perception.Characters {
		if c.ID == r.defender && c.HPFrac > 0 {
			return ErrWait
		}
	}
	r.defender, r.defenseReceipt = "", nil
	return nil
}
