package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/settings"
)

func cameraTestApp(t *testing.T) (*App, *bridgetest.Plugin, string, string) {
	t.Helper()
	cfg := config.Default()
	cfg.Screenshot.Enabled = true
	path := filepath.Join(t.TempDir(), "settings.json")
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	set, err := settings.Open(path, cfg, log, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := New(cfg, nil, log, nil, nil, set)
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome { return &protocol.Welcome{} }}, a.Session)
	p := bridgetest.Connect(t, url, nil)
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one", WorldUID: "uid"})
	cameraWait(t, func() bool { return a.Session.State() != nil })
	t.Cleanup(func() { a.cameraMu.Lock(); a.stopCameraOverrideLocked("test cleanup"); a.cameraMu.Unlock() })
	return a, p, url, path
}

func cameraWait(t *testing.T, f func() bool) {
	t.Helper()
	until := time.Now().Add(2 * time.Second)
	for !f() {
		if time.Now().After(until) {
			t.Fatal("camera condition timed out")
		}
		time.Sleep(time.Millisecond)
	}
}

func cameraWire(t *testing.T, p *bridgetest.Plugin, interval float64, enabled bool) {
	t.Helper()
	msg, ok := p.Next().(*protocol.Config)
	if !ok || msg.Screenshot == nil || msg.Screenshot.IntervalS != interval || msg.Screenshot.Enabled != enabled {
		t.Fatalf("unexpected camera config: %+v", msg)
	}
}

func cameraPost(a *App, body string, auth, action bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/camera", strings.NewReader(body))
	if auth {
		r.Header.Set("Authorization", "Bearer test")
	}
	if action {
		r.Header.Set(panel.ActHeader, "test")
	}
	w := httptest.NewRecorder()
	panel.Handler(panel.Options{Token: "test", Camera: a.CameraHandler()}).ServeHTTP(w, r)
	return w
}

func TestCameraCommandAuthenticationAndBounds(t *testing.T) {
	a, p, _, _ := cameraTestApp(t)
	body := `{"action":"temporary","intervalS":1,"durationS":30}`
	if w := cameraPost(a, body, false, true); w.Code == 200 {
		t.Fatal("unauthenticated mutation")
	}
	if w := cameraPost(a, body, true, false); w.Code != 403 {
		t.Fatal(w.Code)
	}
	for _, body := range []string{
		`{"action":"temporary","intervalS":0.5,"durationS":30}`,
		`{"action":"temporary","intervalS":301,"durationS":30}`,
		`{"action":"temporary","intervalS":1,"durationS":0}`,
		`{"action":"temporary","intervalS":1,"durationS":301}`,
		`{"action":"temporary","durationS":30,"unknown":true}`,
		`{"action":"release","id":"stale"}`,
		`{"action":"release","durationS":30}`,
		`{"action":"set","key":"owners","value":[]}`,
		`{"action":"set","key":"intervalS","value":0}`,
		`{"action":"reset","key":"intervalS","value":5}`,
		`{"action":"reset","key":"intervalS"} {}`,
	} {
		if w := cameraPost(a, body, true, true); w.Code < 400 {
			t.Errorf("accepted %s: %s", body, w.Body.String())
		}
	}
	p.Quiet(20 * time.Millisecond)
}

func TestCameraExpiryRestoresBaseWithoutPersistenceOrBodyControl(t *testing.T) {
	a, p, _, path := cameraTestApp(t)
	// Create the file before leasing so byte equality checks persistence too.
	if err := a.Settings.Set("screenshot.intervalS", json.RawMessage(`5`), settings.ByPanel); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rev := a.Session.ControlRevision()
	w := cameraPost(a, `{"action":"temporary","intervalS":1,"durationS":1}`, true, true)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var status struct {
		Config, BaseConfig protocol.ScreenshotConfig
		Override           *CameraOverride
	}
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Config.IntervalS != 1 || status.BaseConfig.IntervalS != 5 || status.Override == nil {
		t.Fatal(w.Body.String())
	}
	cameraWire(t, p, 1, true)
	cameraWire(t, p, 5, true) // Actual bounded expiry, no manual restoration.
	if cfg, _, override := a.cameraSettings(); cfg.IntervalS != 5 || override != nil {
		t.Fatal(cfg, override)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("lease persisted settings", err)
	}
	fresh, err := settings.Open(path, a.Config, a.cameraLog, nil)
	if err != nil {
		t.Fatal(err)
	}
	restarted := New(a.Config, nil, a.cameraLog, nil, nil, fresh)
	if cfg, _, override := restarted.cameraSettings(); cfg.IntervalS != 5 || override != nil {
		t.Fatal("lease survived restart")
	}
	if rev != a.Session.ControlRevision() {
		t.Fatal("camera changed body authority")
	}
}

func TestCameraReleaseCannotCancelReplacementAndSettingsDisplace(t *testing.T) {
	a, p, _, _ := cameraTestApp(t)
	fast := 1.0
	first, err := a.startCameraOverride(&fast, nil, 30)
	if err != nil {
		t.Fatal(err)
	}
	cameraWire(t, p, 1, true)
	if _, err := a.startCameraOverride(&fast, nil, 30); err == nil {
		t.Fatal("overlapping diagnostic")
	}
	if err := a.releaseCameraOverride(first.ID); err != nil {
		t.Fatal(err)
	}
	cameraWire(t, p, 5, true)
	second, err := a.startCameraOverride(&fast, nil, 30)
	if err != nil {
		t.Fatal(err)
	}
	cameraWire(t, p, 1, true)
	if err := a.releaseCameraOverride(first.ID); err == nil {
		t.Fatal("old id cancelled replacement")
	}
	if _, _, lease := a.cameraSettings(); lease == nil || lease.ID != second.ID {
		t.Fatal("replacement disappeared")
	}
	w := cameraPost(a, `{"action":"set","key":"intervalS","value":7}`, true, true)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cameraWire(t, p, 7, true)
	if _, _, lease := a.cameraSettings(); lease != nil {
		t.Fatal("normal setting did not displace lease")
	}
	w = cameraPost(a, `{"action":"reset","key":"intervalS"}`, true, true)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cameraWire(t, p, 5, true)
	if _, err = a.startCameraOverride(&fast, nil, 30); err != nil {
		t.Fatal(err)
	}
	cameraWire(t, p, 1, true)
	// Even an explicit same-value normal settings update takes precedence.
	if err = a.Settings.Set("screenshot.intervalS", json.RawMessage(`5`), settings.ByPanel); err != nil {
		t.Fatal(err)
	}
	cameraWire(t, p, 5, true)
	if _, _, lease := a.cameraSettings(); lease != nil {
		t.Fatal("same-value setting did not displace")
	}
}

func TestCameraPinsWorldAndConnectionScope(t *testing.T) {
	a, p, url, _ := cameraTestApp(t)
	fast := 1.0
	if err := a.Settings.Pin("screenshot.intervalS", settings.ByPanel); err != nil {
		t.Fatal(err)
	}
	if _, err := a.startCameraOverride(&fast, nil, 30); err == nil {
		t.Fatal("lease bypassed pin")
	}
	if w := cameraPost(a, `{"action":"set","key":"intervalS","value":1}`, true, true); w.Code != 409 {
		t.Fatal("agent bypassed pin", w.Code)
	}
	if err := a.Settings.Unpin("screenshot.intervalS", settings.ByPanel); err != nil {
		t.Fatal(err)
	}
	if _, err := a.startCameraOverride(&fast, nil, 30); err != nil {
		t.Fatal(err)
	}
	cameraWire(t, p, 1, true)
	if err := a.Settings.Pin("screenshot.intervalS", settings.ByPanel); err != nil {
		t.Fatal(err)
	}
	cameraWire(t, p, 5, true)
	if err := a.Settings.Unpin("screenshot.intervalS", settings.ByPanel); err != nil {
		t.Fatal(err)
	}
	off := false
	if _, err := a.startCameraOverride(nil, &off, 30); err != nil {
		t.Fatal(err)
	}
	cameraWire(t, p, 5, false)
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "two", WorldUID: "other"})
	cameraWait(t, func() bool { return a.Session.State().WorldUID == "other" })
	cameraWire(t, p, 5, true)
	lease, err := a.startCameraOverride(&fast, nil, 30)
	if err != nil {
		t.Fatal(err)
	}
	cameraWire(t, p, 1, true)
	p.Close()
	cameraWait(t, func() bool { return a.Session.ConnectionID() == 0 })
	next := bridgetest.Connect(t, url, nil)
	next.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one", WorldUID: "uid"})
	cameraWait(t, func() bool { return a.Session.State() != nil })
	cameraWait(t, func() bool { a.cameraMu.Lock(); defer a.cameraMu.Unlock(); return a.cameraOverride == nil })
	next.Quiet(30 * time.Millisecond) // Expiry/scope cleanup cannot send to this connection.
	if err := a.Session.ConfigureConnection(context.Background(), lease.Connection, protocol.Config{Screenshot: &lease.Config}); err == nil {
		t.Fatal("stale configuration reached replacement")
	}
	next.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "legacy"})
	cameraWait(t, func() bool { return a.Session.State().WorldUID == "" })
	if _, err := a.startCameraOverride(&fast, nil, 30); err == nil {
		t.Fatal("unscoped world accepted")
	}
}
