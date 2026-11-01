package app

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/goal"
	"github.com/warricksothr/renki/internal/inspection"
)

// InspectionHandler shares panel authentication. It never acquires a body lease,
// mutates a checkpoint, or treats reading a selected goal as authorization.
func (a *App) InspectionHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "read-only inspection", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("goal")
		if len(id) > 96 {
			http.Error(w, "invalid goal ID", 400)
			return
		}
		behaviours := a.Behaviours.List()
		snapshot := a.Session.InspectionSnapshot()
		choices := []inspection.Choice{}
		records := a.Goals.Index()
		for _, g := range records {
			choices = append(choices, inspection.Choice{ID: g.ID, Kind: g.Kind, Status: g.Status})
		}
		// Select an actual lease-backed task's episode before falling back to the
		// latest saved record. The projection independently checks correlation.
		if id == "" && snapshot.Task != nil && snapshot.Task.AutomationCurrent {
			if origin, ok := snapshot.Task.Origin.(behaviour.Origin); ok {
				for _, b := range behaviours {
					if b.Name != origin.Instance || !b.Running {
						continue
					}
					if b.EpisodeID != "" {
						id = b.EpisodeID
					} else if b.Definition.Template == "goal" {
						var p struct {
							ID string `json:"id"`
						}
						_ = json.Unmarshal(b.Definition.Params, &p)
						id = p.ID
					}
				}
			}
		}
		if id == "" && len(records) > 0 {
			id = records[0].ID
		}
		var selected *goal.Record
		if id != "" {
			g, ok := a.Goals.Get(id)
			if !ok {
				http.Error(w, "unknown goal", 404)
				return
			}
			selected = &g
		}
		out := inspection.Project(inspection.Input{Now: time.Now().UTC(), Run: a.Goals.RunID(), Session: snapshot, Goal: selected, Choices: choices, Behaviours: behaviours})
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, out)
	})
}
