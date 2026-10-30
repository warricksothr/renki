package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/settings"
)

// CameraOverride is a diagnostic lease, never a persisted setting or body task.
type CameraOverride struct {
	ID         string                    `json:"id"`
	Config     protocol.ScreenshotConfig `json:"config"`
	ExpiresAt  time.Time                 `json:"expiresAt"`
	Connection uint64                    `json:"connection"`
	WorldUID   string                    `json:"worldUid"`
	done       chan struct{}
}

func (a *App) cameraScopeValid(lease *CameraOverride) bool {
	st := a.Session.State()
	return st != nil && st.Phase == protocol.PhaseInWorld && st.WorldUID == lease.WorldUID && a.Session.ConnectionID() == lease.Connection
}

func (a *App) cameraSettings() (protocol.ScreenshotConfig, protocol.ScreenshotConfig, *CameraOverride) {
	a.cameraMu.Lock()
	defer a.cameraMu.Unlock()
	base := a.Settings.Values().Screenshot
	if lease := a.cameraOverride; lease != nil && time.Now().Before(lease.ExpiresAt) && a.cameraScopeValid(lease) {
		copy := *lease
		copy.done = nil
		return lease.Config, base, &copy
	}
	return base, base, nil
}

// Called with cameraMu held so expiry, settings and explicit changes keep order.
func (a *App) sendCameraLocked(cfg protocol.ScreenshotConfig, connection uint64) error {
	a.Session.ClearScreenshot()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return a.Session.ConfigureConnection(ctx, connection, protocol.Config{Screenshot: &cfg})
}

func (a *App) stopCameraOverrideLocked(reason string) {
	if lease := a.cameraOverride; lease != nil {
		a.cameraOverride = nil
		close(lease.done)
		a.cameraLog.Info("camera diagnostic ended", "id", lease.ID, "reason", reason)
	}
}

func (a *App) cameraPinsAllow(cfg, base protocol.ScreenshotConfig) error {
	for _, entry := range a.Settings.Entries() {
		if entry.Pin == nil {
			continue
		}
		if entry.Key == "screenshot.intervalS" && cfg.IntervalS != base.IntervalS || entry.Key == "screenshot.enabled" && cfg.Enabled != base.Enabled {
			return fmt.Errorf("%w: %s", settings.ErrPinned, entry.Key)
		}
	}
	return nil
}

func (a *App) startCameraOverride(interval *float64, enabled *bool, seconds float64) (*CameraOverride, error) {
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 1 || seconds > 300 {
		return nil, errors.New("durationS must be 1–300 seconds")
	}
	a.cameraMu.Lock()
	defer a.cameraMu.Unlock()
	base := a.Settings.Values().Screenshot
	cfg := base
	if interval != nil {
		cfg.IntervalS = *interval
	}
	if enabled != nil {
		cfg.Enabled = *enabled
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := a.cameraPinsAllow(cfg, base); err != nil {
		return nil, err
	}
	st := a.Session.State()
	connection := a.Session.ConnectionID()
	if st == nil || st.Phase != protocol.PhaseInWorld || st.WorldUID == "" || connection == 0 {
		return nil, errors.New("a camera diagnostic requires an in-world connection")
	}
	if lease := a.cameraOverride; lease != nil {
		if time.Now().Before(lease.ExpiresAt) && a.cameraScopeValid(lease) {
			return nil, errors.New("a camera diagnostic is already active; release its id first")
		}
		a.stopCameraOverrideLocked("expired or replaced scope")
	}
	var id [12]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, err
	}
	lease := &CameraOverride{ID: hex.EncodeToString(id[:]), Config: cfg, ExpiresAt: time.Now().Add(time.Duration(seconds * float64(time.Second))), Connection: connection, WorldUID: st.WorldUID, done: make(chan struct{})}
	a.cameraOverride = lease
	if err := a.sendCameraLocked(cfg, connection); err != nil {
		a.stopCameraOverrideLocked("configuration send failed")
		_ = a.sendCameraLocked(base, connection)
		return nil, err
	}
	a.cameraLog.Info("camera diagnostic started", "id", lease.ID, "intervalS", cfg.IntervalS, "enabled", cfg.Enabled, "expiresAt", lease.ExpiresAt)
	go a.watchCameraOverride(lease)
	copy := *lease
	copy.done = nil
	return &copy, nil
}

func (a *App) watchCameraOverride(lease *CameraOverride) {
	for {
		delay := min(time.Until(lease.ExpiresAt), time.Second)
		timer := time.NewTimer(max(delay, 0))
		select {
		case <-lease.done:
			timer.Stop()
			return
		case <-timer.C:
		}
		a.cameraMu.Lock()
		if a.cameraOverride != lease {
			a.cameraMu.Unlock()
			return
		}
		base := a.Settings.Values().Screenshot
		reason := ""
		switch {
		case !time.Now().Before(lease.ExpiresAt):
			reason = "expired"
		case !a.cameraScopeValid(lease):
			reason = "connection/world changed"
		case a.cameraPinsAllow(lease.Config, base) != nil:
			reason = "camera setting pinned"
		}
		if reason != "" {
			a.stopCameraOverrideLocked(reason)
			_ = a.sendCameraLocked(base, lease.Connection)
			a.cameraMu.Unlock()
			return
		}
		a.cameraMu.Unlock()
	}
}

func (a *App) releaseCameraOverride(id string) error {
	a.cameraMu.Lock()
	defer a.cameraMu.Unlock()
	if a.cameraOverride == nil || id == "" || a.cameraOverride.ID != id {
		return errors.New("camera diagnostic id does not match the active override")
	}
	connection := a.cameraOverride.Connection
	a.stopCameraOverrideLocked("released")
	return a.sendCameraLocked(a.Settings.Values().Screenshot, connection)
}

func (a *App) cameraCommand(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if r.Header.Get(panel.ActHeader) == "" {
		http.Error(w, "missing "+panel.ActHeader, http.StatusForbidden)
		return
	}
	var req struct {
		Action    string          `json:"action"`
		IntervalS *float64        `json:"intervalS"`
		Enabled   *bool           `json:"enabled"`
		DurationS float64         `json:"durationS"`
		ID        string          `json:"id"`
		Key       string          `json:"key"`
		Value     json.RawMessage `json:"value"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if !errors.Is(decoder.Decode(new(any)), io.EOF) {
		http.Error(w, "expected one JSON value", 400)
		return
	}
	var err error
	switch req.Action {
	case "temporary":
		if req.Key != "" || req.Value != nil || req.ID != "" {
			http.Error(w, "unexpected fields for temporary", 400)
			return
		}
		_, err = a.startCameraOverride(req.IntervalS, req.Enabled, req.DurationS)
	case "release":
		if req.IntervalS != nil || req.Enabled != nil || req.DurationS != 0 || req.Key != "" || req.Value != nil {
			http.Error(w, "unexpected fields for release", 400)
			return
		}
		err = a.releaseCameraOverride(req.ID)
	case "set", "reset":
		if req.IntervalS != nil || req.Enabled != nil || req.DurationS != 0 || req.ID != "" {
			http.Error(w, "unexpected fields for setting", 400)
			return
		}
		switch req.Key {
		case "intervalS", "enabled", "width", "quality":
		default:
			http.Error(w, "unknown camera setting", 400)
			return
		}
		if req.Action == "set" {
			err = a.Settings.Set("screenshot."+req.Key, req.Value, settings.ByAgent)
		} else if req.Value != nil {
			err = errors.New("reset must not include value")
		} else {
			err = a.Settings.Reset("screenshot."+req.Key, settings.ByAgent)
		}
	default:
		http.Error(w, "action must be temporary, release, set or reset", 400)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	cfg, base, override := a.cameraSettings()
	_ = json.NewEncoder(w).Encode(map[string]any{"config": cfg, "baseConfig": base, "override": override})
}
