package video_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/video"
)

const id = "12345678-1234-1234-1234-123456789abc"

func upstream(t *testing.T, f http.HandlerFunc) (video.Config, *httptest.Server) {
	t.Helper()
	s := httptest.NewServer(f)
	t.Cleanup(s.Close)
	_, p, _ := net.SplitHostPort(strings.TrimPrefix(s.URL, "http://"))
	port, _ := strconv.Atoi(p)
	return video.Config{Port: uint16(port), Password: "server-secret"}, s
}
func request(h http.Handler, method, path, body string, auth bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/sdp")
	r.Header.Set("X-Renki-Act", "video")
	if auth {
		r.Header.Set("Authorization", "Bearer local-panel-token-123")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func scope() video.Scope {
	return video.Scope{Online: true, Connection: 7, World: "Test", WorldUID: "42"}
}
func TestAuthenticatedConstrainedSignaling(t *testing.T) {
	var calls atomic.Int32
	cfg, _ := upstream(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		user, pass, ok := r.BasicAuth()
		if !ok || user != "panel" || pass != "server-secret" {
			t.Error("missing server-side auth")
		}
		if r.Header.Get("Cookie") != "" || r.Header.Get("X-Renki-Act") != "" {
			t.Error("forwarded panel credentials/headers")
		}
		if r.Method == "DELETE" {
			if r.URL.Path != "/renki/whep/"+id {
				t.Error("wrong session path")
			}
			return
		}
		if r.URL.Path != "/renki/whep" {
			t.Error("wrong upstream path")
		}
		w.Header().Set("Location", "/renki/whep/"+id)
		w.Header().Set("Content-Type", "application/sdp")
		w.Header().Set("Set-Cookie", "secret=x")
		w.Header().Set("Link", "turn:secret")
		w.WriteHeader(201)
		fmt.Fprint(w, "v=0\r\n")
	})
	h := panel.Handler(panel.Options{Token: "local-panel-token-123", Status: func() any { return nil }, Video: video.New(cfg, scope)})
	for _, method := range []string{"GET", "POST", "DELETE"} {
		path := "/api/video"
		if method == "POST" {
			path += "/whep"
		}
		if method == "DELETE" {
			path += "/session/" + id
		}
		if w := request(h, method, path, "", false); w.Code != 401 {
			t.Fatalf("unauthenticated %s: %d", method, w.Code)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("unauthenticated upstream access")
	}
	w := request(h, "POST", "/api/video/whep", "v=0\r\n", true)
	if w.Code != 201 || w.Header().Get("Location") != "/api/video/session/"+id || w.Body.String() != "v=0\r\n" {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	for _, header := range []string{"Set-Cookie", "Link", "Authorization"} {
		if w.Header().Get(header) != "" {
			t.Error("leaked", header)
		}
	}
	if w = request(h, "DELETE", "/api/video/session/"+id, "", true); w.Code != 204 {
		t.Fatal(w.Code)
	}
	for _, path := range []string{"/api/video/whep?url=http://example.com", "/api/video/session/https:example.com", "/api/video/session/" + id + "?x=1", "/api/video/whip", "/api/video/renki/other"} {
		if w = request(h, "POST", path, "x", true); w.Code < 400 {
			t.Error("accepted", path)
		}
	}
	if w = request(h, "POST", "/api/video/whep", strings.Repeat("x", 65537), true); w.Code != 400 {
		t.Error("unbounded request", w.Code)
	}
	r := httptest.NewRequest("POST", "/api/video/whep", strings.NewReader("v=0"))
	r.Header.Set("Authorization", "Bearer local-panel-token-123")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Error("no CSRF header", w.Code)
	}
	if calls.Load() != 2 {
		t.Error("unexpected upstream calls", calls.Load())
	}
}
func TestUpstreamCannotRedirectOrSelectSessionURL(t *testing.T) {
	for _, location := range []string{"https://external.example/renki/whep/" + id, "//127.0.0.1/" + id, "/renki/whep/" + id + "?secret=x", "/other/whep/" + id} {
		t.Run(location, func(t *testing.T) {
			cfg, _ := upstream(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/sdp")
				w.Header().Set("Location", location)
				w.WriteHeader(201)
			})
			w := request(video.New(cfg, scope), "POST", "/api/video/whep", "v=0", true)
			if w.Code != 502 || strings.Contains(w.Body.String(), location) {
				t.Fatal("unsafe Location", w.Code)
			}
		})
	}
	var followed atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { followed.Store(true) }))
	defer target.Close()
	cfg, _ := upstream(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) })
	if w := request(video.New(cfg, scope), "POST", "/api/video/whep", "v=0", true); w.Code != 503 || followed.Load() {
		t.Fatal("followed redirect")
	}
}
func TestBoundedAnswerCancellationAndConcurrency(t *testing.T) {
	cfg, _ := upstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/renki/whep/"+id)
		w.Header().Set("Content-Type", "application/sdp")
		w.WriteHeader(201)
		io.WriteString(w, strings.Repeat("x", 65537))
	})
	if w := request(video.New(cfg, scope), "POST", "/api/video/whep", "v=0", true); w.Code != 502 {
		t.Error("unbounded SDP answer")
	}
	entered := make(chan struct{}, 4)
	release := make(chan struct{})
	cfg, s := upstream(t, func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	h := video.New(cfg, scope)
	done := make(chan struct{}, 4)
	for range 4 {
		go func() { request(h, "POST", "/api/video/whep", "v=0", true); done <- struct{}{} }()
		<-entered
	}
	if w := request(h, "POST", "/api/video/whep", "v=0", true); w.Code != 429 {
		t.Fatal("unbounded concurrency", w.Code)
	}
	// Status/body stop remain available while every media slot is occupied.
	panelHandler := panel.Handler(panel.Options{Token: "local-panel-token-123", Status: func() any { return nil }, Video: h, Act: func(context.Context, string) (string, error) { return "Stopped.", nil }})
	if w := request(panelHandler, "POST", "/api/act/stop", "", true); w.Code != 200 || !strings.Contains(w.Body.String(), "Stopped.") {
		t.Fatal("body stop depends on video", w.Code)
	}
	if w := request(h, "GET", "/api/video", "", true); w.Code != 200 {
		t.Error("status blocked")
	}
	close(release)
	for range 4 {
		<-done
	}
	s.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := httptest.NewRequest("POST", "/api/video/whep", strings.NewReader("v=0")).WithContext(ctx)
	r.Header.Set("X-Renki-Act", "video")
	r.Header.Set("Content-Type", "application/sdp")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 502 {
		t.Error("cancel ignored", w.Code)
	}
}
func TestContentTrustRequiresFreshMatchingCaptureAndNeverOwnsTransport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "health.json")
	cfg := video.Config{Port: 18889, Password: "private", HealthFile: path}
	s := scope()
	h := video.New(cfg, func() video.Scope { return s })
	for _, tc := range []struct {
		name   string
		health video.Health
		online bool
		want   string
	}{
		{"fresh", video.Health{Gameplay: true, Connection: 7, WorldUID: "42", ObservedAt: time.Now()}, true, "gameplay"},
		{"offline", video.Health{Gameplay: true, Connection: 7, WorldUID: "42", ObservedAt: time.Now()}, false, "standby"},
		{"wrong connection", video.Health{Gameplay: true, Connection: 6, WorldUID: "42", ObservedAt: time.Now()}, true, "standby"},
		{"wrong world", video.Health{Gameplay: true, Connection: 7, WorldUID: "43", ObservedAt: time.Now()}, true, "standby"},
		{"expired", video.Health{Gameplay: true, Connection: 7, WorldUID: "42", ObservedAt: time.Now().Add(-4 * time.Second)}, true, "standby"},
		{"future", video.Health{Gameplay: true, Connection: 7, WorldUID: "42", ObservedAt: time.Now().Add(time.Hour)}, true, "standby"},
		{"untrusted source", video.Health{Connection: 7, WorldUID: "42", ObservedAt: time.Now()}, true, "standby"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s.Online = tc.online
			b, _ := json.Marshal(tc.health)
			os.WriteFile(path, b, 0600)
			w := request(h, "GET", "/api/video", "", true)
			var data map[string]any
			json.Unmarshal(w.Body.Bytes(), &data)
			if data["content"] != tc.want || data["enabled"] != true || strings.Contains(w.Body.String(), "private") || strings.Contains(w.Body.String(), path) {
				t.Fatal(w.Body.String())
			}
		})
	}
	for _, data := range []string{"{", strings.Repeat("x", 4097)} {
		os.WriteFile(path, []byte(data), 0600)
		if w := request(h, "GET", "/api/video", "", true); strings.Contains(w.Body.String(), `"content":"gameplay"`) {
			t.Fatal("invalid health trusted")
		}
	}
}
