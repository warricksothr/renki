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
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/audit"
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
