package app

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/goal"
	"github.com/warricksothr/renki/internal/panel"
)

// BehaviourHandler is installed behind the panel's existing token/cookie
// authentication. Mutations use the same CSRF header as other controls.
func (a *App) BehaviourHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, a.Behaviours.List())
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get(panel.ActHeader) == "" {
			http.Error(w, "missing "+panel.ActHeader, http.StatusForbidden)
			return
		}
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 128<<10))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var request struct {
			Name       string            `json:"name"`
			Action     string            `json:"action"`
			Definition json.RawMessage   `json:"definition,omitempty"`
			Target     string            `json:"target,omitempty"`
			Marker     *behaviour.Marker `json:"marker,omitempty"`
			Enabled    *bool             `json:"enabled,omitempty"`
			Sleep      *bool             `json:"sleep,omitempty"`
		}
		if err = behaviour.Decode(raw, &request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		switch request.Action {
		case "idle-food", "idle-cargo", "idle-rest", "idle-resupply":
			if request.Marker == nil || request.Enabled == nil {
				err = fmt.Errorf("idle food needs a marker and enabled flag")
				break
			}
			if request.Action == "idle-rest" && request.Sleep == nil {
				err = fmt.Errorf("idle rest needs an explicit sleep mode")
				break
			}
			m := *request.Marker
			found := false
			for _, waypoint := range a.Memory.Store.Waypoints(m.World) {
				if waypoint.Name == m.Name && waypoint.Kind == m.Kind {
					found = true
				}
			}
			if !found {
				err = fmt.Errorf("marker changed or is missing")
				break
			}
			// The same world/name maps to one definition even when kind changes.
			name := fmt.Sprintf("idle food %x", sha256.Sum256([]byte(m.World+"\x00"+m.Name)))
			definition := behaviour.MarkerFoodDefinition(m)
			if request.Action == "idle-resupply" {
				h, ok := a.Memory.Store.Home(m.World)
				if !ok {
					err = fmt.Errorf("personal resupply needs known home in the marker world")
					break
				}
				name = fmt.Sprintf("idle resupply %x", sha256.Sum256([]byte(m.World+"\x00"+m.Name)))
				params, _ := json.Marshal(goal.Spec{Kind: "food_resupply", World: m.World, Center: h.Pos, Radius: 150, Reserve: 2})
				definition = behaviour.Definition{Template: "personal_food", Params: params,
					Trigger: behaviour.Trigger{Kind: "idle", Marker: &m, DwellS: 15},
					Repeat:  behaviour.Repeat{Mode: "trigger", CooldownS: 300, MaxRuns: 100}, Priority: 10, Enabled: true}
			}
			if request.Action == "idle-cargo" {
				name = fmt.Sprintf("idle cargo %x", sha256.Sum256([]byte(m.World+"\x00"+m.Name)))
				definition = behaviour.MarkerCargoDefinition(m)
			}
			if request.Action == "idle-rest" {
				prefix := "idle rest"
				if *request.Sleep {
					prefix = "idle sleep"
				}
				name = fmt.Sprintf("%s %x", prefix, sha256.Sum256([]byte(m.World+"\x00"+m.Name)))
				definition = behaviour.MarkerRestDefinition(m, *request.Sleep)
			}
			definition.Enabled = *request.Enabled
			var raw json.RawMessage
			raw, err = json.Marshal(definition)
			if err == nil {
				err = a.Behaviours.Put(ctx, name, raw, "panel")
			}
		case "save":
			err = a.Behaviours.Put(ctx, request.Name, request.Definition, "panel")
		case "clone":
			err = a.Behaviours.Clone(ctx, request.Name, request.Target, "panel")
		default:
			err = a.Behaviours.Action(ctx, request.Name, request.Action, "panel")
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, a.Behaviours.List())
	})
}
