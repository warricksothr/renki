// Package panel is the controller's web control panel: a page for watching
// the bot live, served on its own address (the VM's Tailscale address) behind
// a login token. docs/PANEL.md describes it.
//
// The page's files are embedded, and it gets everything over two endpoints:
// GET /api/status for a snapshot of the bot, and GET /api/events, a stream of
// server-sent events carrying the audit log's records as they are written
// (internal/audit), after the recent ones, and a status snapshot each second.
package panel

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/warricksothr/renki/internal/audit"
	"github.com/warricksothr/renki/internal/protocol"
)

//go:embed static
var static embed.FS

// Cookie is the name of the login cookie.
const Cookie = "renki_panel"

// Options configure the panel.
type Options struct {
	// Token is what the login form wants. It must not be empty.
	Token string
	// Log is the feed of records.
	Log *audit.Log
	// Status returns a snapshot of the bot as JSON-able data.
	Status func() any
	Logger *slog.Logger
	// StatusEvery is how often the event stream sends a status. Default 1 s.
	StatusEvery time.Duration
}

// Handler returns the panel's HTTP handler.
func Handler(o Options) http.Handler {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.StatusEvery == 0 {
		o.StatusEvery = time.Second
	}
	if o.Token == "" {
		panic("panel: empty token")
	}
	p := &panel{o: o}
	files, _ := fs.Sub(static, "static")
	authed := http.NewServeMux()
	authed.Handle("GET /", http.FileServerFS(files))
	authed.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(o.Status())
	})
	authed.HandleFunc("GET /api/events", p.events)

	mux := http.NewServeMux()
	// The stylesheet holds nothing private, and the login page needs it.
	mux.Handle("GET /style.css", http.FileServerFS(files))
	mux.HandleFunc("GET /login", p.loginForm)
	mux.HandleFunc("POST /login", p.login)
	mux.HandleFunc("POST /logout", p.logout)
	mux.Handle("/", p.auth(authed))
	return headers(mux)
}

type panel struct {
	o Options
}

// headers adds the security headers every response carries.
func headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// auth lets a request through with the token in the login cookie or as a
// bearer token, sends a page to the login form, and refuses anything else.
func (p *panel) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := ""
		if c, err := r.Cookie(Cookie); err == nil {
			got = c.Value
		} else if h := r.Header.Get("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
			got = h[7:]
		}
		if p.valid(got) {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodGet && (r.URL.Path == "/" || r.URL.Path == "/index.html") {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		http.Error(w, "log in first", http.StatusUnauthorized)
	})
}

func (p *panel) valid(token string) bool {
	return token != "" && subtle.ConstantTimeCompare([]byte(token), []byte(p.o.Token)) == 1
}

var loginPage = template.Must(template.New("login").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>renki: log in</title><link rel="stylesheet" href="/style.css"></head>
<body class="login"><form method="post" action="/login">
<h1>renki</h1>
{{if .}}<p class="error">{{.}}</p>{{end}}
<label>Token <input type="password" name="token" autocomplete="current-password" autofocus required></label>
<button>Log in</button>
</form></body></html>`))

func (p *panel) loginForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	loginPage.Execute(w, "")
}

func (p *panel) login(w http.ResponseWriter, r *http.Request) {
	token := r.PostFormValue("token")
	if !p.valid(token) {
		p.o.Logger.Warn("panel login refused", "remote", r.RemoteAddr)
		time.Sleep(time.Second) // slow guessing down
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		loginPage.Execute(w, "That token is not right.")
		return
	}
	p.o.Logger.Info("panel login", "remote", r.RemoteAddr)
	http.SetCookie(w, &http.Cookie{Name: Cookie, Value: token, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteStrictMode, MaxAge: 30 * 24 * 3600})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (p *panel) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: Cookie, Value: "", Path: "/", HttpOnly: true,
		SameSite: http.SameSiteStrictMode, MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Event is a record as the page gets it: the record, the timeline's line
// for it, and a category for the page's filters and map.
type Event struct {
	audit.Record
	Line string `json:"line"`
	Cat  string `json:"cat"`
}

// Categories of Event.
const (
	CatChat       = "chat"
	CatTask       = "task"
	CatTrace      = "trace"
	CatDecision   = "decision"
	CatState      = "state"
	CatPerception = "perception"
	CatOther      = "other"
)

func event(r audit.Record) Event {
	return Event{Record: r, Line: audit.Line(r), Cat: category(r)}
}

func category(r audit.Record) string {
	switch r.Kind {
	case audit.KindLog:
		return CatDecision
	case audit.KindFrame:
		var env struct {
			Type string `json:"type"`
		}
		json.Unmarshal(r.Frame, &env)
		switch env.Type {
		case protocol.TypeChat:
			return CatChat
		case protocol.TypeTask:
			return CatTask
		case protocol.TypeTrace:
			return CatTrace
		case protocol.TypeState:
			return CatState
		case protocol.TypePerception:
			return CatPerception
		}
	}
	return CatOther
}

// events streams the recent records, then new ones as they come, with a
// status every StatusEvery, until the client goes.
func (p *panel) events(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")

	// Subscribe before reading the recent records, so nothing falls between.
	ch, cancel := p.o.Log.Subscribe()
	defer cancel()
	send := func(name string, v any) bool {
		data, err := json.Marshal(v)
		if err != nil {
			return true
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, data); err != nil {
			return false
		}
		return true
	}
	for _, rec := range p.o.Log.RecentRecords() {
		if !send("record", event(rec)) {
			return
		}
	}
	send("status", p.o.Status())
	fl.Flush()

	tick := time.NewTicker(p.o.StatusEvery)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case rec := <-ch:
			if !send("record", event(rec)) {
				return
			}
		case <-tick.C:
			if !send("status", p.o.Status()) {
				return
			}
		}
		fl.Flush()
	}
}
