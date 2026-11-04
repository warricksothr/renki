package panel

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSharingPersistsRotationRevocationAndFailedWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sharing.json")
	s, err := OpenSharing(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.valid("") {
		t.Fatal("disabled sharing accepted empty token")
	}
	if err = s.change("rotate"); err != nil {
		t.Fatal(err)
	}
	first := s.token
	restored, err := OpenSharing(path)
	if err != nil || !restored.valid(first) {
		t.Fatalf("persisted access: %v", err)
	}
	if info, _ := os.Stat(path); info.Mode().Perm()&0077 != 0 {
		t.Fatalf("credential permissions: %v", info.Mode())
	}
	if err = s.change("rotate"); err != nil {
		t.Fatal(err)
	}
	if s.valid(first) || !s.valid(s.token) {
		t.Fatal("rotation failed")
	}
	current := s.token
	s.path = filepath.Join(path, "impossible")
	if err = s.change("disable"); err == nil || !s.valid(current) {
		t.Fatal("failed persistence changed active credential")
	}
	s.path = path
	if err = s.change("disable"); err != nil {
		t.Fatal(err)
	}
	restored, err = OpenSharing(path)
	if err != nil || restored.valid(current) {
		t.Fatal("revocation did not persist")
	}
}

func TestViewerAuthorizationIsAllowlistedAndSeparate(t *testing.T) {
	sharing, _ := OpenSharing(filepath.Join(t.TempDir(), "viewer.json"))
	if err := sharing.change("rotate"); err != nil {
		t.Fatal(err)
	}
	privateCalls := 0
	data := func() any { privateCalls++; return map[string]any{"secret": "operator"} }
	mediaCalls := 0
	media := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mediaCalls++; w.WriteHeader(204) })
	h := Handler(Options{Token: token, Sharing: sharing, Status: data, Inventory: data, Map: data, ViewerStatus: func() any { return map[string]any{"character": "Tas"} }, ViewerMap: func() any { return map[string]any{"tiles": []int{1}} }, ViewerAtlas: media, Video: media})
	call := func(method, path, body, credential string, operator, header bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if credential != "" {
			r.AddCookie(&http.Cookie{Name: ViewerCookie, Value: credential})
		}
		if operator {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if header {
			r.Header.Set(ActHeader, "test")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for _, path := range []string{"/view", "/app.js", "/terrain.js"} {
		if w := call("GET", path, "", "", false, false); w.Code != 200 {
			t.Fatalf("bootstrap %s: %d", path, w.Code)
		}
	}
	for _, path := range []string{"/api/status", "/api/map", "/api/video", "/api/atlas/worlds"} {
		if w := call("GET", path, "", "", false, false); w.Code != 401 {
			t.Fatalf("anonymous %s: %d", path, w.Code)
		}
		w := call("GET", path, "", sharing.token, false, false)
		if w.Code >= 300 || strings.Contains(w.Body.String(), "secret") {
			t.Fatalf("viewer %s: %d %s", path, w.Code, w.Body.String())
		}
	}
	for _, path := range []string{"/api/task", "/api/act/stop", "/api/setting", "/api/settings", "/api/say", "/api/sharing", "/api/inventory", "/api/inspection", "/api/goals", "/api/behaviours", "/api/days", "/api/timeline", "/api/memory/players", "/api/camera", "/api/waypoints", "/api/bases", "/api/video/other"} {
		for _, method := range []string{"GET", "POST", "DELETE"} {
			w := call(method, path, `{}`, sharing.token, false, true)
			if w.Code != 403 {
				t.Fatalf("viewer escaped allowlist %s %s: %d", method, path, w.Code)
			}
		}
	}
	for _, c := range []struct{ method, path string }{{"POST", "/api/video/whep"}, {"DELETE", "/api/video/session/example"}} {
		if w := call(c.method, c.path, "", sharing.token, false, true); w.Code != 204 {
			t.Fatalf("playback %s: %d", c.path, w.Code)
		}
	}
	if privateCalls != 0 || mediaCalls != 4 {
		t.Fatalf("unexpected handlers: private %d media %d", privateCalls, mediaCalls)
	}
	// The viewer token never authenticates to the operator mux as a bearer token.
	r := httptest.NewRequest("GET", "/api/settings", nil)
	r.Header.Set("Authorization", "Bearer "+sharing.token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("viewer bearer operator access: %d", w.Code)
	}
	if w := call("POST", "/api/sharing", `{"action":"rotate"}`, "", true, false); w.Code != 403 {
		t.Fatalf("sharing CSRF: %d", w.Code)
	}
	login := call("POST", "/api/viewer/login", `{"token":"`+sharing.token+`"}`, "", false, true)
	if login.Code != 204 || len(login.Result().Cookies()) != 1 || !login.Result().Cookies()[0].HttpOnly || login.Result().Cookies()[0].Name != ViewerCookie {
		t.Fatalf("viewer login: %d %v", login.Code, login.Result().Cookies())
	}
	if w := call("POST", "/api/viewer/login", `{"token":"`+token+`"}`, "", false, true); w.Code != 401 {
		t.Fatal("operator token accepted as viewer")
	}
	old := sharing.token
	if w := call("POST", "/api/sharing", `{"action":"rotate"}`, "", true, true); w.Code != 200 {
		t.Fatal("operator could not rotate")
	}
	if w := call("GET", "/api/status", "", old, false, false); w.Code != 401 {
		t.Fatal("revoked cookie accepted")
	}
}

func TestViewerStreamOnlyObservationsAndClosesAfterRevocation(t *testing.T) {
	sharing, _ := OpenSharing(filepath.Join(t.TempDir(), "viewer.json"))
	if err := sharing.change("rotate"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(Handler(Options{Token: token, Sharing: sharing, StatusEvery: 5 * time.Millisecond, Status: func() any { t.Error("private status called"); return nil }, ViewerStatus: func() any { return map[string]any{"character": "Tas"} }}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/viewer/events", nil)
	r.AddCookie(&http.Cookie{Name: ViewerCookie, Value: sharing.token})
	res, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	scanner := bufio.NewScanner(res.Body)
	if !scanner.Scan() || scanner.Text() != "event: status" {
		t.Fatal("missing observation event")
	}
	if !scanner.Scan() || !strings.Contains(scanner.Text(), `"character":"Tas"`) {
		t.Fatal("missing observation data")
	}
	if err = sharing.change("disable"); err != nil {
		t.Fatal(err)
	}
	tail, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("stream did not close on revocation: %v", err)
	}
	if strings.Contains(string(tail), "record") {
		t.Fatal("audit record leaked")
	}
}

func TestOperatorViewerPreviewGetsFilteredData(t *testing.T) {
	h := Handler(Options{Token: token, Status: func() any { return map[string]any{"owners": "secret"} }, ViewerStatus: func() any { return map[string]any{"character": "Tas"} }})
	r := httptest.NewRequest("GET", "/api/status", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("X-Renki-Viewer", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["owners"] != nil || body["character"] != "Tas" {
		t.Fatalf("preview unfiltered: %v", body)
	}
}
