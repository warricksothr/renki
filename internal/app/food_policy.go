package app

import (
	"time"

	"github.com/warricksothr/renki/internal/food"
	"github.com/warricksothr/renki/internal/protocol"
)

func (a *App) foodWorkload() string {
	if a.Behaviours != nil {
		if name := a.Behaviours.Workload(); name != "" {
			switch name {
			case "marker_food", "marker_rest", "personal_food", "household_food":
				return "idle:" + name // Enclosing chores override their nested steps.
			default:
				return "behavior:" + name
			}
		}
	}
	for _, g := range a.Goals.Index() {
		if g.Status == "running" && g.Kind != "food_resupply" {
			return "goal:" + g.Kind
		}
	}
	s := a.Session.InspectionSnapshot()
	if s.Task != nil && !protocol.Final(s.Task.Status) && food.WorkingTask(s.Task.Name) {
		return "task:" + s.Task.Name
	}
	return ""
}

func (a *App) foodDecision() *food.Decision {
	s, at := a.Session.StateObservation()
	_, _, connected := a.Session.Handshake()
	if !connected || s == nil || s.Phase != protocol.PhaseInWorld || time.Since(at) < 0 || time.Since(at) > 3*time.Second {
		return nil
	}
	for _, g := range a.Goals.Index() {
		if g.Status == "running" && g.Kind == "food_resupply" {
			full, ok := a.Goals.Get(g.ID)
			if ok && full.Food != nil && full.Food.Decision != nil {
				return full.Food.Decision
			}
			d := food.Decision{Version: 1, Mode: "reserved", Activity: "goal:food_resupply", Reason: "goal_owns_feeding"}
			return &d
		}
	}
	d := food.Decide(s, a.Settings.Values().Food, a.foodWorkload())
	return &d
}
