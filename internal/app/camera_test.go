package app

import (
	"bytes"
	"encoding/json"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/settings"
)

func TestCameraAuthenticationMetadataImageAndDisable(t *testing.T) {
	cfg := config.Default()
	cfg.Screenshot.Enabled = true
	a := New(cfg, nil, nil, nil, nil, nil)
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome { return &protocol.Welcome{} }}, a.Session)
	p := bridgetest.Connect(t, url, nil)
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one", WorldUID: "uid"})
	until := time.Now().Add(2 * time.Second)
	for a.Session.State() == nil {
		if time.Now().After(until) {
			t.Fatal("state missing")
		}
		time.Sleep(time.Millisecond)
	}
	var jpegData bytes.Buffer
	_ = jpeg.Encode(&jpegData, image.NewRGBA(image.Rect(0, 0, 8, 4)), nil)
	p.Send(&protocol.Screenshot{Envelope: protocol.Envelope{Seq: 10}, World: "one", WorldUID: "uid", Width: 8, Height: 4, CapturedAt: time.Now(), JPEG: jpegData.Bytes()})
	for a.Session.Screenshot() == nil {
		if time.Now().After(until) {
			t.Fatal("image missing")
		}
		time.Sleep(time.Millisecond)
	}
	handler := panel.Handler(panel.Options{Token: "test", Camera: a.CameraHandler()})
	get := func(path string, auth bool) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("GET", path, nil)
		if auth {
			r.Header.Set("Authorization", "Bearer test")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if get("/api/camera.jpg", false).Code == http.StatusOK {
		t.Fatal("image exposed without authentication")
	}
	w := get("/api/camera", true)
	if w.Code != 200 || bytes.Contains(w.Body.Bytes(), []byte(`"jpeg"`)) {
		t.Fatal("bad metadata", w.Body.String())
	}
	var metadata struct {
		Available bool
		ImageURL  string `json:"imageUrl"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &metadata); err != nil || !metadata.Available {
		t.Fatal(err, w.Body.String())
	}
	w = get(metadata.ImageURL, true)
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), jpegData.Bytes()) || w.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatal("JPEG unavailable", w.Code)
	}
	if get("/api/camera.jpg?seq=9", true).Code != 409 {
		t.Fatal("old frame URL served new image")
	}
	if err := a.Settings.Set("screenshot.enabled", json.RawMessage(`false`), settings.ByPanel); err != nil {
		t.Fatal(err)
	}
	if get("/api/camera.jpg", true).Code != 404 {
		t.Fatal("disabled image exposed")
	}
	if a.Session.Screenshot() != nil {
		t.Fatal("disabled image retained")
	}
}
