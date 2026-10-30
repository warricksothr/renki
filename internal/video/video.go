// Package video bridges private WHEP signaling only. Media never traverses the
// game bridge, and game logout never destroys a video session.
package video

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const maxSDP = 64 << 10

var sessionID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Config is startup-only. The upstream host and stream path are deliberately
// fixed. Password and HealthFile never enter public responses.
type Config struct {
	Port       uint16 `json:"port"`
	Password   string `json:"password"`
	HealthFile string `json:"healthFile"`
}

// Scope is independent of the lifetime of a media session.
type Scope struct {
	Online     bool
	Connection uint64
	World      string
	WorldUID   string
}

// Health is a local capture adapter's assertion, NOT WebRTC/OBS source activity.
// The adapter must fail closed on uncertain capture and write atomically.
type Health struct {
	Gameplay   bool      `json:"gameplay"`
	Connection uint64    `json:"connection"`
	WorldUID   string    `json:"worldUid"`
	ObservedAt time.Time `json:"observedAt"`
}

func trusted(h Health, s Scope, now time.Time) bool {
	age := now.Sub(h.ObservedAt)
	return s.Online && s.WorldUID != "" && h.Gameplay && h.Connection == s.Connection && h.WorldUID == s.WorldUID && age >= 0 && age <= 3*time.Second
}

// New returns a bounded signaling handler; mount only behind panel auth.
func New(c Config, scope func() Scope) http.Handler {
	return &handler{config: c, scope: scope, slots: make(chan struct{}, 4), client: &http.Client{
		Timeout:       8 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		Transport:     &http.Transport{Proxy: nil, MaxConnsPerHost: 4, MaxIdleConnsPerHost: 4, ResponseHeaderTimeout: 7 * time.Second},
	}}
}

type handler struct {
	config Config
	scope  func() Scope
	slots  chan struct{}
	client *http.Client
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == "GET" && r.URL.Path == "/api/video" {
		state := "standby"
		reason := "Video is not configured"
		s := h.scope()
		if h.config.Port != 0 {
			reason = "Waiting for verified game capture"
			if !s.Online {
				reason = "Tas is resting"
			} else if h.config.HealthFile != "" {
				f, err := os.Open(h.config.HealthFile)
				if err == nil {
					var health Health
					data, readErr := io.ReadAll(io.LimitReader(f, 4097))
					f.Close()
					if readErr == nil && len(data) <= 4096 && json.Unmarshal(data, &health) == nil && trusted(health, s, time.Now()) {
						state = "gameplay"
						reason = ""
					}
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Enabled bool   `json:"enabled"`
			Content string `json:"content"`
			Reason  string `json:"reason"`
			Scope   string `json:"scope"`
		}{h.config.Port != 0, state, reason, fmt.Sprintf("%d:%s:%s", s.Connection, s.World, s.WorldUID)})
		return
	}
	if h.config.Port == 0 {
		http.Error(w, "Video is not configured", 503)
		return
	}
	if r.Header.Get("X-Renki-Act") != "video" {
		http.Error(w, "Missing video request header", 403)
		return
	}
	upstreamPath := "/renki/whep"
	switch {
	case r.Method == "POST" && r.URL.Path == "/api/video/whep":
		if r.Header.Get("Content-Type") != "application/sdp" {
			http.Error(w, "Expected SDP", 415)
			return
		}
	case r.Method == "DELETE" && strings.HasPrefix(r.URL.Path, "/api/video/session/"):
		id := strings.TrimPrefix(r.URL.Path, "/api/video/session/")
		if !sessionID.MatchString(id) {
			http.Error(w, "Invalid session", 400)
			return
		}
		upstreamPath += "/" + id
	default:
		http.Error(w, "Unsupported video operation", 405)
		return
	}
	if r.URL.RawQuery != "" {
		http.Error(w, "Unexpected video query", 400)
		return
	}
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
	default:
		http.Error(w, "Video busy", 429)
		return
	}
	// Also bound slow incoming request bodies; context alone does not interrupt Read.
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(8 * time.Second))
	defer func() { _ = http.NewResponseController(w).SetReadDeadline(time.Time{}) }()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxSDP))
	if err != nil {
		http.Error(w, "Video request too large or incomplete", 400)
		return
	}
	if r.Method == "DELETE" && len(body) != 0 {
		http.Error(w, "Unexpected session body", 400)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, r.Method, fmt.Sprintf("http://127.0.0.1:%d%s", h.config.Port, upstreamPath), strings.NewReader(string(body)))
	if err != nil {
		http.Error(w, "Video unavailable", 502)
		return
	}
	if r.Method == "POST" {
		req.Header.Set("Content-Type", "application/sdp")
	}
	if h.config.Password != "" {
		req.SetBasicAuth("panel", h.config.Password)
	}
	res, err := h.client.Do(req)
	if err != nil {
		http.Error(w, "Video unavailable", 502)
		return
	}
	defer res.Body.Close()
	if r.Method == "DELETE" {
		if res.StatusCode != 200 && res.StatusCode != 404 {
			http.Error(w, "Video session unavailable", 502)
			return
		}
		w.WriteHeader(204)
		return
	}
	if res.StatusCode != 201 {
		http.Error(w, "Video source unavailable", 503)
		return
	}
	// Do not forward arbitrary upstream Locations, cookies, errors or ICE-server
	// Link headers. Ephemeral session ICE credentials necessarily live in SDP. Non-trickle host ICE needs no OPTIONS/PATCH passthrough.
	id := strings.TrimPrefix(res.Header.Get("Location"), "/renki/whep/")
	if !sessionID.MatchString(id) || res.Header.Get("Content-Type") != "application/sdp" {
		http.Error(w, "Invalid video response", 502)
		return
	}
	answer, err := io.ReadAll(io.LimitReader(res.Body, maxSDP+1))
	if err != nil || len(answer) > maxSDP {
		http.Error(w, "Invalid video response", 502)
		return
	}
	w.Header().Set("Content-Type", "application/sdp")
	w.Header().Set("Location", "/api/video/session/"+id)
	w.WriteHeader(201)
	w.Write(answer)
}
