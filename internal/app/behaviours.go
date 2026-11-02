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
		}
		if err = behaviour.Decode(raw, &request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		switch request.Action {
		case "idle-food":
			if request.Marker == nil || request.Enabled == nil {
				err = fmt.Errorf("idle food needs a marker and enabled flag")
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
