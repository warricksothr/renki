package goal

import (
	"encoding/json"
	"errors"
	"math"
	"slices"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func (r *runner) pinHome(g Record, v behaviour.View) error {
	if v.Home == nil || distance(*v.Home, g.Center) > g.Radius {
		return errors.New("combat return requires a known home inside goal scope")
	}
	for _, n := range *v.Home {
		if !finite(n) {
			return errors.New("combat return requires a finite home")
		}
	}
	if g.Home != nil {
		if distance(*g.Home, *v.Home) > .1 {
			return errors.New("pinned home changed; return needs a new plan")
		}
		return nil
	}
	return r.t.Store.update(r.id, func(g *Record) error { at := *v.Home; g.Home = &at; return nil })
}

// afterCombat cannot choose another fight. Proof and its original task identity
// stay intact through return failures, owner takeover and restart uncertainty.
func (r *runner) afterCombat(g Record, v behaviour.View) (*session.Order, error) {
	if !finite(v.State.HP) || !finite(v.State.HPMax) || v.State.HPMax <= 0 {
		return r.need("invalid post-combat health observation")
	}
	for _, n := range v.State.Pos {
		if !finite(n) {
			return r.need("invalid post-combat position")
		}
	}
	if g.Home == nil {
		return r.need("kill verified but no pinned return home")
	}
	if err := r.pinHome(g, v); err != nil {
		return r.need(err.Error())
	}
	if distance(v.State.Pos, g.Center) > g.Radius {
		return r.need("post-combat position outside goal scope")
	}
	if g.PostCombat == nil {
		deadline := v.Now.Add(3 * time.Minute)
		if deadline.After(g.Deadline) {
			deadline = g.Deadline
		}
		if err := r.t.Store.update(r.id, func(g *Record) error {
			g.PostCombat = &PostCombat{Phase: "recover", Began: v.Now, Deadline: deadline}
			g.Reason = "kill verified; checking recovery before returning home"
			return nil
		}); err != nil {
			return nil, err
		}
		g, _ = r.t.Store.Get(r.id)
	}
	p := g.PostCombat
	if !v.Now.Before(p.Deadline) {
		return r.need("post-combat recovery/return deadline exhausted")
	}
	if v.State.Encumbered() || slices.Contains(v.State.Flags, protocol.FlagSwimming) {
		return r.need("post-combat footing or carry weight prevents safe return")
	}
	if v.Perception == nil || !v.PerceptionAt.After(g.Proof.At) || v.PerceptionAt.After(v.Now) || v.Now.Sub(v.PerceptionAt) > 3*time.Second ||
		v.State.T < v.Perception.T || v.State.T-v.Perception.T > 3 {
		return r.returnWait(g, v, "waiting for fresh post-combat surroundings")
	}
	for _, c := range v.Perception.Characters {
		if c.ID == g.Proof.Entity || c.HPFrac <= 0 || c.Kind == protocol.CharacterPlayer {
			continue
		}
		clearance := 8.0
		if c.Hostile {
			clearance = 15
		}
		if distance(c.Pos, v.State.Pos) < clearance || c.TargetingMe {
			return r.need("another creature prevents safe post-combat recovery/return")
		}
	}
	if p.Receipt != nil {
		if p.Receipt.Status != protocol.StatusSucceeded {
			return r.need("post-combat return failed; kill proof retained")
		}
		if !v.StateAt.After(p.Receipt.At) {
			return r.returnWait(g, v, "waiting for post-return position")
		}
		flat := math.Hypot(v.State.Pos[0]-(*g.Home)[0], v.State.Pos[2]-(*g.Home)[2])
		// Match native Arrival.Level; a stricter threshold can reject a valid
		// floor arrival when the home marker was placed above standing height.
		if flat > 3 || math.Abs(v.State.Pos[1]-(*g.Home)[1]) > 1.75 {
			return r.need("native return receipt lacks confirmed home arrival")
		}
		return nil, r.t.Store.update(r.id, func(g *Record) error {
			g.Observation = r.observation
			g.PostCombat.Phase = "complete"
			g.Status = "succeeded"
			g.Reason = "observed Greyling death and confirmed return home"
			return nil
		})
	}
	if p.Phase != "recover" {
		return r.need("post-combat return has no receipt; never replayed")
	}
	if v.State.Food == nil || len(v.State.Food.Active) == 0 {
		return r.need("kill verified; active food required for recovery/return")
	}
	if !finite(v.State.HPMax) || v.State.HPMax <= 0 || !finite(v.State.Stamina) || !finite(v.State.HP) {
		return r.need("invalid post-combat readiness observation")
	}
	if v.State.HP < v.State.HPMax*.8 || v.State.Stamina < 40 {
		if v.Now.Sub(p.Began) >= time.Minute {
			return r.need("post-combat readiness did not recover within one minute")
		}
		return r.returnWait(g, v, "recover health/stamina before returning home")
	}
	if g.TaskCount >= g.MaxTasks {
		return r.need("native return task budget exhausted; kill proof retained")
	}
	args := &protocol.MoveToArgs{To: protocol.Target{Pos: g.Home}, Dry: true}
	raw, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	if err := r.t.Store.update(r.id, func(g *Record) error {
		g.Observation = r.observation
		g.TaskCount++
		g.PostCombat.Phase = "return"
		g.Reason = "kill verified; returning to pinned home"
		g.Wake = time.Time{}
		g.Pending = &Intent{Template: "return_home", Params: raw}
		return nil
	}); err != nil {
		return nil, err
	}
	return &session.Order{Task: protocol.TaskMoveTo, Args: args, TimeoutS: math.Min(120, p.Deadline.Sub(v.Now).Seconds())}, nil
}

func (r *runner) returnWait(g Record, v behaviour.View, why string) (*session.Order, error) {
	if g.WaitS+2 > 120 {
		return r.need("post-combat observation/recovery wait budget exhausted")
	}
	until := v.Now.Add(2 * time.Second)
	if until.After(g.PostCombat.Deadline) {
		until = g.PostCombat.Deadline
	}
	if err := r.t.Store.update(r.id, func(g *Record) error {
		g.Observation = r.observation
		g.WaitS += until.Sub(v.Now).Seconds()
		g.Wake = until
		g.Reason = why
		return nil
	}); err != nil {
		return nil, err
	}
	return nil, &behaviour.Delay{Until: until}
}
