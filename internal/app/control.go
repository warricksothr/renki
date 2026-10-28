package app

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// Status is what GET /control/status reports.
type Status struct {
	Connected bool              `json:"connected"`
	Character string            `json:"character,omitempty"`
	Phase     string            `json:"phase,omitempty"`
	Task      *protocol.TaskRef `json:"task,omitempty"`
}

// control serves /control/ for scripts on the same machine, such as the VM's
// stop script: GET /control/status and POST /control/logout. It answers only
// loopback callers, and when a token is configured it wants it as a bearer
// token too, since anything on the machine can reach loopback.
func (a *App) control(log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /control/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, a.status())
	})
	mux.HandleFunc("POST /control/logout", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		err := a.Session.Logout(ctx)
		switch {
		case errors.Is(err, session.ErrNotConnected):
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		case err != nil:
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		default:
			log.Info("logout requested", "remote", r.RemoteAddr)
			writeJSON(w, http.StatusAccepted, a.status())
		}
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !loopback(r.RemoteAddr) {
			http.Error(w, "the control API answers loopback only", http.StatusForbidden)
			return
		}
		if tok := a.Config.Token; tok != "" {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if subtle.ConstantTimeCompare([]byte(got), []byte(tok)) != 1 {
				http.Error(w, "bad token", http.StatusUnauthorized)
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}

func (a *App) status() Status {
	hello, _, ok := a.Session.Handshake()
	if !ok {
		return Status{}
	}
	st := Status{Connected: true, Character: hello.Character, Phase: hello.Phase}
	if s := a.Session.State(); s != nil {
		st.Phase, st.Task = s.Phase, s.Task
		if s.Character != "" {
			st.Character = s.Character
		}
	}
	return st
}

func loopback(remote string) bool {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
