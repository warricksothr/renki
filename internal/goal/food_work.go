package goal

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/session"
)

// FoodWork creates one durable episode under the behaviour engine's body lease.
// No nested engine, side dispatch or params-to-goal global map is involved.
type FoodWork struct{ Goal *Template }

func (t *FoodWork) Validate(raw json.RawMessage) error {
	var spec Spec
	if err := decode(raw, &spec); err != nil {
		return err
	}
	if spec.Kind != "food_resupply" || spec.Stock == nil {
		return errors.New("household work requires food_resupply with stock targets")
	}
	// Only configuration may be supplied, never checkpoint evidence.
	if len(spec.Stock.Pinned) > 0 || spec.Stock.ReturnAfterCooking || spec.Stock.Delivering || spec.Stock.Decision != "" || len(spec.Stock.Known) > 0 || len(spec.Stock.Shortage) > 0 {
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
		if v.State == nil || !foodAtHome(v.State.Pos, r.spec.Center) {
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
		return nil
	}
	return r.run.Check(v)
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
