package panel

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/audit"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/settings"
)

const token = "0123456789abcdef-test"

func server(t *testing.T) (*httptest.Server, *audit.Log) {
	t.Helper()
	l, err := audit.Open("", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(Handler(Options{
		Token:       token,
		Log:         l,
		Status:      func() any { return map[string]any{"connected": true, "character": "Tasleroch"} },
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		StatusEvery: 50 * time.Millisecond,
		Act: func(_ context.Context, action string) (string, error) {
			switch action {
			case "stop":
				return "Stopped.", nil
			case "home":
				return "", errors.New("no home in this world yet")
			}
			return "", ErrUnknownAction
		},
	}))
	t.Cleanup(srv.Close)
	return srv, l
}

// client follows no redirects, so tests see where they point.
func client() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// TestButtons: each answers with text to show; a request without the
// header, as a forged form from another site would be, does nothing.
func TestButtons(t *testing.T) {
	srv, _ := server(t)
	post := func(path string, header, login bool) (int, string) {
		req, _ := http.NewRequest("POST", srv.URL+path, nil)
		if login {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if header {
			req.Header.Set(ActHeader, "1")
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var body struct{ Text string }
		json.NewDecoder(res.Body).Decode(&body)
		return res.StatusCode, body.Text
	}
	for _, c := range []struct {
		path          string
		header, login bool
		code          int
		text          string
	}{
		{"/api/act/stop", true, true, 200, "Stopped."},
		{"/api/act/home", true, true, 409, "no home in this world yet"},
		{"/api/act/dance", true, true, 404, "no such action"},
		{"/api/act/stop", false, true, 403, ""},
		{"/api/act/stop", true, false, 401, ""},
	} {
		code, text := post(c.path, c.header, c.login)
		if code != c.code || text != c.text {
			t.Errorf("%s header=%v login=%v: %d %q; want %d %q", c.path, c.header, c.login, code, text, c.code, c.text)
		}
	}
}

func TestNothingWithoutLoggingIn(t *testing.T) {
	srv, _ := server(t)
	c := client()
	res, _ := c.Get(srv.URL + "/")
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/login" {
		t.Errorf("/: %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	for _, path := range []string{"/api/status", "/api/events", "/app.js", "/index.html?x"} {
		res, _ := c.Get(srv.URL + path)
		if res.StatusCode != http.StatusUnauthorized && res.StatusCode != http.StatusSeeOther {
			t.Errorf("%s served without a login: %d", path, res.StatusCode)
		}
	}
	res, _ = c.Get(srv.URL + "/style.css")
	if res.StatusCode != 200 {
		t.Errorf("the login page's stylesheet: %d", res.StatusCode)
	}
	if res.Header.Get("Content-Security-Policy") == "" || res.Header.Get("X-Frame-Options") != "DENY" {
		t.Errorf("security headers missing: %v", res.Header)
	}
}

func TestLoginLogout(t *testing.T) {
	srv, _ := server(t)
	c := client()
	res, _ := c.PostForm(srv.URL+"/login", url.Values{"token": {"wrong"}})
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("a wrong token: %d", res.StatusCode)
	}
	res, _ = c.PostForm(srv.URL+"/login", url.Values{"token": {token}})
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("login: %d", res.StatusCode)
	}
	cookie := res.Cookies()[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("cookie %+v", cookie)
	}
	res, _ = c.Get(srv.URL + "/")
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || !strings.Contains(string(body), `<canvas id="map"`) {
		t.Fatalf("the page after login: %d", res.StatusCode)
	}
	res, _ = c.Get(srv.URL + "/api/status")
	var st map[string]any
	json.NewDecoder(res.Body).Decode(&st)
	if st["character"] != "Tasleroch" {
		t.Errorf("status %v", st)
	}
	c.Post(srv.URL+"/logout", "", nil)
	if res, _ := c.Get(srv.URL + "/api/status"); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("after logout: %d", res.StatusCode)
	}

	// Scripts can use the token as a bearer token instead.
	req, _ := http.NewRequest("GET", srv.URL+"/api/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if res, _ := http.DefaultClient.Do(req); res.StatusCode != 200 {
		t.Errorf("bearer: %d", res.StatusCode)
	}
}

func TestEventsReplayThenStream(t *testing.T) {
	srv, l := server(t)
	l.Frame(1, audit.In, []byte(`{"v":1,"type":"chat","seq":1,"t":0,"from":{"name":"Nimah"},"channel":"shout","text":"tas follow me","pos":[0,0,0],"dist":3}`))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/events", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if ct := res.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type %q", ct)
	}
	events := make(chan [2]string)
	go func() {
		s := bufio.NewScanner(res.Body)
		var name string
		for s.Scan() {
			line := s.Text()
			switch {
			case strings.HasPrefix(line, "event: "):
				name = line[7:]
			case strings.HasPrefix(line, "data: "):
				events <- [2]string{name, line[6:]}
			}
		}
		close(events)
	}()
	next := func(want string) Event {
		t.Helper()
		for e := range events {
			if e[0] != want {
				continue
			}
			var ev Event
			json.Unmarshal([]byte(e[1]), &ev)
			return ev
		}
		t.Fatalf("no %s event", want)
		return Event{}
	}

	if ev := next("record"); ev.Cat != CatChat || !strings.Contains(ev.Line, "Nimah (shout, 3 m): tas follow me") {
		t.Errorf("replayed %+v", ev)
	}
	next("status")
	l.Frame(1, audit.In, []byte(`{"v":1,"type":"trace","seq":2,"t":0,"task":"t-1","source":"walker","msg":"stuck; trying Jump","data":{"attempt":1}}`))
	if ev := next("record"); ev.Cat != CatTrace || !strings.Contains(ev.Line, "t-1 walker: stuck; trying Jump attempt=1") {
		t.Errorf("live %+v", ev)
	}
}

// TestSettingsFromThePage: a change is the panel's, the page reads it back
// with who made it, and bad, unknown and owner-only changes say why.
func TestSettingsFromThePage(t *testing.T) {
	cfg := config.Default()
	cfg.Owners = []string{"Sothr"}
	cfg.LLM = &config.LLM{Terva: json.RawMessage(`{"provider":"openai-compatible","model":"gemma","baseURL":"http://cpa/v1"}`)}
	store, err := settings.Open("", cfg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	l, _ := audit.Open("", 0, nil)
	srv := httptest.NewServer(Handler(Options{
		Token: token, Log: l, Status: func() any { return nil }, Settings: store,
		Models: func(context.Context) ([]string, error) { return []string{"claude-haiku-5-5", "gemma"}, nil },
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}))
	defer srv.Close()
	do := func(method, path, body string) (int, string) {
		req, _ := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(ActHeader, "1")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(b)
	}
	for _, c := range []struct {
		body string
		code int
	}{
		{`{"key":"model.name","value":"claude-haiku-5-5"}`, 200},
		{`{"key":"home.idleMinutes","value":"soon"}`, 400},
		{`{"key":"colour","value":"red"}`, 404},
		{`{"key":"model.name"`, 400},
		{`{"key":"overload.drop","value":false}`, 200},
		{`{"key":"overload.drop","pin":true}`, 200},
		{`{"key":"overload.drop","reset":true}`, 200},
		{`{"key":"overload.drop","pin":false}`, 200},
	} {
		if code, body := do("POST", "/api/setting", c.body); code != c.code {
			t.Errorf("%s: %d %s; want %d", c.body, code, body, c.code)
		}
	}
	if m := store.Values().Model; m.Name != "claude-haiku-5-5" {
		t.Errorf("model %+v", m)
	}
	if !store.Values().Overload.Drop {
		t.Error("reset did not return overload.drop to the config's value")
	}
	if _, body := do("GET", "/api/settings", ""); !strings.Contains(body, `"by":"panel"`) {
		t.Errorf("settings do not say the panel changed one: %s", body)
	}
	if _, body := do("GET", "/api/models", ""); !strings.Contains(body, `"models":["claude-haiku-5-5","gemma"]`) {
		t.Errorf("models %s", body)
	}
}

// TestPastDays: the day picker lists the files, a day reads without state,
// and a task is the one of the run going at the time asked about.
func TestPastDays(t *testing.T) {
	dir := t.TempDir()
	lines := []string{
		`{"at":"2026-10-28T10:00:00Z","kind":"start","msg":"renki controller"}`,
		`{"at":"2026-10-28T10:00:02Z","kind":"frame","dir":"out","frame":{"v":1,"type":"task","id":"t-1","task":"follow","args":{}}}`,
		`{"at":"2026-10-28T10:00:03Z","kind":"frame","dir":"in","frame":{"v":1,"type":"state","seq":1,"t":0}}`,
		`{"at":"2026-10-28T10:00:04Z","kind":"frame","dir":"in","frame":{"v":1,"type":"task","re":"t-1","status":"failed","reason":"stuck"}}`,
		`{"at":"2026-10-28T11:00:00Z","kind":"start","msg":"renki controller"}`,
		`{"at":"2026-10-28T11:00:02Z","kind":"frame","dir":"out","frame":{"v":1,"type":"task","id":"t-1","task":"move_to","args":{}}}`,
	}
	day := time.Date(2026, 10, 28, 11, 0, 2, 0, time.UTC).Local().Format(time.DateOnly)
	if err := os.WriteFile(filepath.Join(dir, "audit-"+day+".jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	l, _ := audit.Open("", 0, nil)
	srv := httptest.NewServer(Handler(Options{Token: token, Log: l, Status: func() any { return nil }, AuditDir: dir,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}))
	defer srv.Close()
	get := func(path string, v any) int {
		req, _ := http.NewRequest("GET", srv.URL+path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		json.NewDecoder(res.Body).Decode(v)
		return res.StatusCode
	}
	var days []Day
	if get("/api/days", &days); len(days) != 1 || days[0].Day != day {
		t.Errorf("days %+v", days)
	}
	var tl Timeline
	get("/api/timeline?day="+day, &tl)
	if len(tl.Events) != 5 || tl.More {
		t.Errorf("timeline: %d events, more %v", len(tl.Events), tl.More)
	}
	var evs []Event
	get("/api/task?id=t-1&at=2026-10-28T11:00:02Z", &evs)
	if len(evs) != 1 || !strings.Contains(evs[0].Line, "move_to") || evs[0].Task != "t-1" {
		t.Errorf("t-1 of the second run: %+v", evs)
	}
	// Paging back from 10:00:04, of which the page has its one record:
	// the start and the request are left.
	get("/api/timeline?day="+day+"&before=2026-10-28T10:00:04Z&skip=1", &tl)
	if len(tl.Events) != 2 || tl.Events[1].Task != "t-1" || !strings.Contains(tl.Events[1].Line, "follow") {
		t.Errorf("earlier page: %+v", tl.Events)
	}
	if code := get("/api/timeline?day=../../etc/passwd", &tl); code != http.StatusBadRequest {
		t.Errorf("a path for a day: %d", code)
	}
}

func TestSayFromThePage(t *testing.T) {
	var heard []string
	l, _ := audit.Open("", 0, nil)
	srv := httptest.NewServer(Handler(Options{Token: token, Log: l, Status: func() any { return nil },
		Say: func(as, text string) error {
			if as != "Sothr" {
				return errors.New("not an owner: " + as)
			}
			heard = append(heard, text)
			return nil
		},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}))
	defer srv.Close()
	for _, c := range []struct {
		body   string
		header bool
		code   int
	}{
		{`{"as":"Sothr","text":"follow me"}`, true, 200},
		{`{"as":"Mallory","text":"follow me"}`, true, 409},
		{`{"as":"Sothr","text":"  "}`, true, 400},
		{`{"as":"Sothr","text":"stop"}`, false, 403},
	} {
		req, _ := http.NewRequest("POST", srv.URL+"/api/say", strings.NewReader(c.body))
		req.Header.Set("Authorization", "Bearer "+token)
		if c.header {
			req.Header.Set(ActHeader, "1")
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != c.code {
			t.Errorf("%s header=%v: %d; want %d", c.body, c.header, res.StatusCode, c.code)
		}
	}
	if len(heard) != 1 || heard[0] != "follow me" {
		t.Errorf("heard %q", heard)
	}
}
