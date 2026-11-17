package panel

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const ViewerCookie = "renki_viewer"

// Sharing keeps a separate revocable viewer credential. Empty means disabled.
// Persist before publishing changes so a failed write preserves current access.
type Sharing struct {
	mu          sync.RWMutex
	path, token string
}

func OpenSharing(path string) (*Sharing, error) {
	s := &Sharing{path: path}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, &s.token); err != nil {
		return nil, err
	}
	if _, err := hex.DecodeString(s.token); err != nil || (s.token != "" && len(s.token) != 64) {
		return nil, fmt.Errorf("invalid viewer credential file")
	}
	return s, nil
}
func (s *Sharing) valid(token string) bool {
	if s == nil || token == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token != "" && subtle.ConstantTimeCompare([]byte(token), []byte(s.token)) == 1
}
func (s *Sharing) change(action string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := ""
	switch action {
	case "rotate":
		var b [32]byte
		if _, err := rand.Read(b[:]); err != nil {
			return err
		}
		next = hex.EncodeToString(b[:])
	case "disable":
	default:
		return fmt.Errorf("unknown sharing action")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".viewer-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	b, _ := json.Marshal(next)
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(f.Name(), s.path); err != nil {
		return err
	}
	s.token = next
	return nil
}
func (p *panel) sharing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if p.o.Sharing == nil {
		http.Error(w, "sharing unavailable", 503)
		return
	}
	if r.Method == "POST" {
		if r.Header.Get(ActHeader) == "" {
			http.Error(w, "missing action header", 403)
			return
		}
		var in struct {
			Action string `json:"action"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&in) != nil || (in.Action != "rotate" && in.Action != "disable") {
			http.Error(w, "invalid sharing action", 400)
			return
		}
		if err := p.o.Sharing.change(in.Action); err != nil {
			p.o.Logger.Error("viewer credential write failed", "error", err)
			http.Error(w, "sharing update failed", 500)
			return
		}
	}
	s := p.o.Sharing
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"enabled": s.token != "", "token": s.token})
}
func viewerToken(r *http.Request) string {
	if c, err := r.Cookie(ViewerCookie); err == nil {
		return c.Value
	}
	return ""
}
func (p *panel) viewerLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Header.Get(ActHeader) == "" {
		http.Error(w, "missing action header", 403)
		return
	}
	var in struct {
		Token string `json:"token"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&in) != nil || !p.o.Sharing.valid(in.Token) {
		http.Error(w, "viewer access unavailable", 401)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: ViewerCookie, Value: in.Token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil, MaxAge: 30 * 24 * 3600})
	w.WriteHeader(http.StatusNoContent)
}
func (p *panel) operator(r *http.Request) bool {
	if c, err := r.Cookie(Cookie); err == nil && p.valid(c.Value) {
		return true
	}
	h := r.Header.Get("Authorization")
	return len(h) > 7 && h[:7] == "Bearer " && p.valid(h[7:])
}
func (p *panel) viewerAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !p.operator(r) && !p.o.Sharing.valid(viewerToken(r)) {
			http.Error(w, "viewer access unavailable", 401)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// viewerEvents never subscribes to audit records. Recheck the credential each
// tick so rotation closes existing streams as well as refusing new requests.
func (p *panel) viewerEvents(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	tick := time.NewTicker(p.o.StatusEvery)
	defer tick.Stop()
	for {
		if !p.operator(r) && !p.o.Sharing.valid(viewerToken(r)) {
			return
		}
		data, err := json.Marshal(p.o.ViewerStatus())
		if err != nil {
			return
		}
		if _, err = fmt.Fprintf(w, "event: status\ndata: %s\n\n", data); err != nil {
			return
		}
		fl.Flush()
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
		}
	}
}
