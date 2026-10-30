package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/goal"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
)

// GoalHandler shares panel authentication and CSRF protection. It never grants
// permissions or resumes a goal implicitly after an outage.
func (a *App) GoalHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeJSON(w, 200, a.Goals.Summaries())
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		if r.Header.Get(panel.ActHeader) == "" {
			http.Error(w, "missing "+panel.ActHeader, 403)
			return
		}
		var req struct {
			Action string     `json:"action"`
			ID     string     `json:"id,omitempty"`
			Spec   *goal.Spec `json:"spec,omitempty"`
		}
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 32<<10))
		if err == nil {
			err = behaviour.Decode(raw, &req)
		}
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		a.goalMu.Lock()
		defer a.goalMu.Unlock()
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if req.Action == "start" && req.ID == "" && req.Spec != nil {
			var record goal.Record
			record, err = goal.NewRecord(*req.Spec, time.Now().UTC())
			if err == nil {
				err = a.Goals.Add(record)
				req.ID = record.ID
			}
		}
		if err == nil {
			err = a.goalAction(ctx, req.ID, req.Action)
		}
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		record, _ := a.Goals.Get(req.ID)
		writeJSON(w, 200, record)
	})
}
func (a *App) goalAction(ctx context.Context, id, action string) error {
	record, ok := a.Goals.Get(id)
	if !ok {
		return errors.New("unknown goal")
	}
	name := "goal/" + id
	switch action {
	case "stop":
		// Persist cancellation before stopping the engine; a disk failure must not
		// prevent an actual stop. The engine independently checkpoints its end.
		saveErr := a.goalTemplate.Cancel(id)
		var stopErr error
		for _, instance := range a.Behaviours.List() {
			if instance.Name == name {
				stopErr = a.Behaviours.Action(ctx, name, "stop", "panel")
				break
			}
		}
		return errors.Join(saveErr, stopErr)
	case "start", "resume":
		_, _, connected := a.Session.Handshake()
		state, at := a.Session.StateObservation()
		if !connected || state == nil || state.Phase != protocol.PhaseInWorld || state.World != record.World || state.HP <= 0 || time.Since(at) > 3*time.Second || time.Since(at) < 0 {
			return errors.New("fresh living observation in goal world required")
		}
		if state.Task != nil && !protocol.Final(state.Task.Status) {
			return errors.New("body is busy")
		}
		if err := a.goalTemplate.Queue(id); err != nil {
			return err
		}
		params, _ := json.Marshal(map[string]string{"id": id})
		d, _ := json.Marshal(behaviour.Definition{Template: "goal", Params: params, Trigger: behaviour.Trigger{Kind: "manual"}, Repeat: behaviour.Repeat{Mode: "once"}, Enabled: true})
		err := a.Behaviours.Put(ctx, name, d, "panel")
		if err == nil {
			err = a.Behaviours.Action(ctx, name, "start", "panel")
		}
		if err != nil {
			_ = a.goalTemplate.End(params, "could not queue: "+err.Error())
		}
		return err
	default:
		return errors.New("goal action must be start, resume or stop")
	}
}
