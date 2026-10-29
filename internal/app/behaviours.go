package app

import (
	"context"
	"encoding/json"
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
			Name       string          `json:"name"`
			Action     string          `json:"action"`
			Definition json.RawMessage `json:"definition,omitempty"`
			Target     string          `json:"target,omitempty"`
		}
		if err = behaviour.Decode(raw, &request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		switch request.Action {
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
