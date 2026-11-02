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

// FoodWork creates one durable episode under the behaviour engine's body lease.
// No nested engine, side dispatch or params-to-goal global map is involved.
type FoodWork struct {
	Goal     *Template
	Personal bool
}

func (t *FoodWork) Validate(raw json.RawMessage) error {
	var spec Spec
	if err := decode(raw, &spec); err != nil {
		return err
	}
	if t.Personal {
		if spec.Kind != "food_resupply" || spec.Stock != nil || spec.Escort != nil || spec.Peel || spec.ReturnHome || spec.Radius > 150 {
			return errors.New("personal idle work requires a bounded food_resupply without stock or escort")
		}
	} else if spec.Kind != "food_resupply" || spec.Stock == nil {
		return errors.New("household work requires food_resupply with stock targets")
	}
	// Only configuration may be supplied, never checkpoint evidence.
	if spec.Stock != nil && (len(spec.Stock.Pinned) > 0 || len(spec.Stock.HuntingPinned) > 0 || spec.Stock.Replenishing || spec.Stock.ReturnAfterCooking || spec.Stock.Delivering || spec.Stock.Decision != "" || len(spec.Stock.Known) > 0 || len(spec.Stock.Shortage) > 0) {
		return errors.New("household params cannot supply stock checkpoint evidence")
	}
	_, err := NewRecord(spec, time.Now().UTC())
	return err
}
func (t *FoodWork) Start(raw json.RawMessage) (behaviour.Runner, error) {
	if err := t.Validate(raw); err != nil {
		return nil, err
	}
	var spec Spec
	_ = decode(raw, &spec)
	return &foodWorkRun{t: t, spec: spec}, nil
}

type foodWorkRun struct {
	t      *FoodWork
	spec   Spec
	run    *runner
	params json.RawMessage
}

func (r *foodWorkRun) Next(v behaviour.View, last *behaviour.Result) (*session.Order, error) {
	if r.run == nil {
		if r.t.Personal {
			if err := r.personalAdmission(v); err != nil {
				return nil, err
			}
			reserve := r.spec.Reserve
			if reserve == 0 {
				reserve = 2
			}
			// Full base-health ratio alone says nothing about hunger. Avoid a
			// durable episode when both nourishment and the reserve are ready.
			if nourished(v.State) && mealCount(v.Inventory) >= reserve {
				return nil, nil
			}
		} else if v.State == nil || !foodAtHome(v.State.Pos, r.spec.Center) {
			return nil, errors.New("household idle episode must start on the home floor")
		}
		g, err := NewRecord(r.spec, v.Now)
		if err != nil {
			return nil, err
		}
		if err = r.t.Goal.Store.Add(g); err != nil {
			return nil, err
		}
		r.params, _ = json.Marshal(params{g.ID})
		if err = r.t.Goal.Queue(g.ID); err != nil {
			return nil, err
		}
		run, err := r.t.Goal.Start(r.params)
		if err != nil {
			return nil, err
		}
		r.run = run.(*runner)
	}
	return r.run.Next(v, last)
}
func (r *foodWorkRun) Recover(result behaviour.Result) bool {
	return r.run != nil && r.run.Recover(result)
}
func (r *foodWorkRun) Dispatched(id string) error {
	if r.run == nil {
		return errors.New("household episode not started")
	}
	return r.run.Dispatched(id)
}
func (r *foodWorkRun) Check(v behaviour.View) error {
	if r.run == nil {
		if r.t.Personal {
			return r.personalAdmission(v)
		}
		return nil
	}
	return r.run.Check(v)
}

func (r *foodWorkRun) personalAdmission(v behaviour.View) error {
	if !foodView(v, Record{World: r.spec.World}) || v.Home == nil || !validFoodPos(*v.Home) || !validFoodPos(v.State.Pos) ||
		distance(*v.Home, r.spec.Center) > .1 || v.State.Food == nil || !v.Idle {
		return errors.New("personal resupply needs fresh idle observations and its original home")
	}
	s := v.State
	if !finite(s.HP) || !finite(s.HPMax) || s.HP <= 0 || s.HPMax <= 0 || s.Time.Day < 1 || s.Time.Night ||
		slices.Contains(s.Flags, protocol.FlagInBed) || slices.Contains(s.Flags, protocol.FlagSleeping) ||
		!slices.Contains(s.Flags, protocol.FlagGrounded) || !slices.Contains(s.Flags, protocol.FlagInShelter) ||
		s.Stranded() || s.Encumbered() || slices.Contains(s.Flags, protocol.FlagSwimming) || s.Location != nil && s.Location.InWater ||
		s.Pos.FlatDistance(r.spec.Center) > 12 || math.Abs(s.Pos[1]-r.spec.Center[1]) > 1.75 {
		return errors.New("personal resupply starts awake on sheltered dry home footing")
	}
	for _, c := range v.Perception.Characters {
		if c.HPFrac > 0 && (c.TargetingMe || c.Hostile && distance(c.Pos, s.Pos) < 20) {
			return errors.New("nearby threat prevents idle personal resupply")
		}
	}
	return nil
}
func (*foodWorkRun) TravelFromIdleMarker() bool { return true }
func (r *foodWorkRun) EndRun(why string) error {
	if len(r.params) == 0 {
		return nil
	}
	return r.t.Goal.End(r.params, why)
}

func (r *foodWorkRun) EpisodeID() string {
	if r.run == nil {
		return ""
	}
	return r.run.id
}
