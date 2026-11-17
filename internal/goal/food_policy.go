package goal

import (
	"reflect"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/food"
)

func (r *runner) foodDecision(g Record, v behaviour.View) (food.Decision, error) {
	p := food.Policy{}
	if r.t.FoodPolicy != nil {
		p = r.t.FoodPolicy()
	}
	work := ""
	// The durable goal owns workload, including gaps between native steps.
	if g.Food.Escort != nil {
		work = "escort_preparation"
	} else if !foodAtHome(v.State.Pos, g.Center) {
		work = "food_trip"
	} else if foodEventHazard(g, v) && v.State.WorldEvents.Valid() {
		work = "world_event"
	}
	st := *v.State
	st.Task = nil
	d := food.Decide(&st, p, work)
	if reflect.DeepEqual(g.Food.Decision, &d) {
		return d, nil
	}
	err := r.t.Store.update(g.ID, func(g *Record) error { g.Food.Decision = &d; return nil })
	if err == nil && r.t.Logger != nil {
		r.t.Logger.Info("food goal decision", "goalId", g.ID, "run", r.t.Store.RunID(), "decision", d, "worldUid", v.State.WorldUID, "connection", v.Connection, "stateSeq", v.State.Seq)
	}
	return d, err
}
