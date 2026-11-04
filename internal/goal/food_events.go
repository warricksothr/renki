package goal

import (
	"fmt"
	"slices"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func foodEventHazard(g Record, v behaviour.View) bool {
	e := v.State.WorldEvents
	return !e.Valid() || e.LocalEvent != nil || e.RandomEvent != nil &&
		(e.RandomEvent.Contains(g.Center) || e.RandomEvent.Contains(v.State.Pos))
}

// Household work defers in observed shelter. This does not make shelter proof
// against a raid: actual nearby/pursuing enemies and injuries still refuse work.
// The caller settles outstanding native receipts before reaching this gate.
func (r *runner) foodEventWait(g Record, v behaviour.View) (*session.Order, error) {
	for _, c := range v.Perception.Characters {
		if c.HPFrac > 0 && c.TargetingMe {
			return r.need("pursuing threat prevents sheltering through a world event")
		}
	}
	if !foodAtHome(v.State.Pos, g.Center) || !slices.Contains(v.State.Flags, protocol.FlagInShelter) {
		return r.need("household work needs safe home shelter or an event escape plan")
	}
	// Eating carried food is preparation, and never requires leaving shelter.
	decision, err := r.foodDecision(g, v)
	if err != nil {
		return nil, err
	}
	item := decision.Item
	fillSlot := item != "" && v.State.Food != nil && len(v.State.Food.Active) < 3 && remaining(v.State, item) == 0
	if item != "" && (fillSlot || !nourished(v.State) || v.State.HP < v.State.HPMax*.8) {
		return r.foodDispatch(g, v, "eat", item, "", &session.Order{Task: protocol.TaskUseItem, Args: &protocol.UseItemArgs{Item: item}, TimeoutS: 5})
	}
	why := "waiting in shelter for fresh world-event knowledge"
	if v.State.WorldEvents.Valid() {
		name := ""
		if e := v.State.WorldEvents.RandomEvent; e != nil {
			name = e.Name
		}
		if e := v.State.WorldEvents.LocalEvent; e != nil {
			name = e.Name
		}
		why = fmt.Sprintf("waiting in shelter for world event %s to clear; its timer may pause", name)
	}
	return r.foodWait(g, v, why)
}
