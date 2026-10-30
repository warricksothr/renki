// Package panel is the controller's web control panel: a page for watching
// the bot live, served on its own address (the VM's Tailscale address) behind
// a login token. docs/PANEL.md describes it.
//
// The page's files are embedded, and it gets everything over two endpoints:
// GET /api/status for a snapshot of the bot, and GET /api/events, a stream of
// server-sent events carrying the audit log's records as they are written
// (internal/audit), after the recent ones, and a status snapshot each second.
// POST /api/act/{action} asks the bot to stop, go home or log out.
// GET /api/settings and POST /api/setting read and change the runtime
// settings (internal/settings), and GET /api/models lists the voice's
// models. GET /api/days, /api/timeline and /api/task read the audit files
// of past days.
package panel

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/audit"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/settings"
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
	// Act does one of Actions and returns what to show for it, or an
	// error to show instead. Nil turns the buttons' endpoint off.
	Act func(ctx context.Context, action string) (string, error)
	// Settings is the store the page reads and changes, as the panel.
	// Nil turns those endpoints off.
	Settings *settings.Store
	// Models lists the voice's models; nil when there is no voice.
	Models func(ctx context.Context) ([]string, error)
	// AuditDir holds the audit files the timeline reads; "" turns it off.
	AuditDir string
	// Say takes words typed into the page as if the owner as had said them
	// to the bot; nil turns the command box off.
	Say func(as, text string) error
	// Dispatch sends an explicit task through normal capability/safety checks.
	Dispatch func(context.Context, TaskCommand) (string, error)
	// Behaviours serves engine controls behind the panel's authentication.
	Behaviours      http.Handler
	ContainerMemory func() any
	LandmarkMemory  func() any
	Map             func() any
	Inventory       func() any
	Waypoints       http.Handler
}

// TaskCommand is an explicit order for repeatable game checks. IDs and
// protocol envelope fields belong to the session, never the caller.
type TaskCommand struct {
	Task     string          `json:"task"`
	Args     json.RawMessage `json:"args,omitempty"`
	TimeoutS float64         `json:"timeoutS,omitempty"`
}

func (c TaskCommand) Validate() error {
	if c.TimeoutS < 0 || c.TimeoutS > 600 {
		return fmt.Errorf("timeoutS must be between 0 and 600")
	}
	_, err := (&protocol.TaskRequest{Task: c.Task, Args: c.Args}).DecodeArgs()
	return err
}

// Actions are what the page's buttons ask for.
var Actions = []string{"stop", "home", "logout"}

// ErrUnknownAction is what Act returns for an action not in Actions.
var ErrUnknownAction = errors.New("no such action")

// ActHeader must be on every POST to /api/act/: a page on another site
// cannot set it without the browser asking first, which this server never
// allows, so a forged form cannot press the buttons.
const ActHeader = "X-Renki-Act"

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
	if o.Inventory != nil {
		authed.HandleFunc("GET /api/inventory", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_ = json.NewEncoder(w).Encode(o.Inventory())
		})
	}
	if o.Map != nil {
		authed.HandleFunc("GET /api/map", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_ = json.NewEncoder(w).Encode(o.Map())
		})
	}
	if o.Waypoints != nil {
		authed.Handle("GET /api/waypoints", o.Waypoints)
		authed.Handle("POST /api/waypoints", o.Waypoints)
	}
	if o.ContainerMemory != nil {
		authed.HandleFunc("GET /api/memory/containers", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(o.ContainerMemory())
		})
	}
	if o.LandmarkMemory != nil {
		authed.HandleFunc("GET /api/memory/landmarks", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(o.LandmarkMemory())
		})
	}
	if o.Behaviours != nil {
		authed.Handle("GET /api/behaviours", o.Behaviours)
		authed.Handle("POST /api/behaviours", o.Behaviours)
	}
	if o.Act != nil {
		authed.HandleFunc("POST /api/act/{action}", p.act)
	}
	if o.Settings != nil {
		authed.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(o.Settings.Entries())
		})
		authed.HandleFunc("POST /api/setting", p.set)
	}
	authed.HandleFunc("GET /api/models", p.models)
	if o.Say != nil {
		authed.HandleFunc("POST /api/say", p.say)
	}
	if o.Dispatch != nil {
		authed.HandleFunc("POST /api/task", p.dispatch)
	}
	if o.AuditDir != "" {
		authed.HandleFunc("GET /api/days", p.days)
		authed.HandleFunc("GET /api/timeline", p.timeline)
		authed.HandleFunc("GET /api/task", p.task)
	}

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

func (p *panel) dispatch(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(ActHeader) == "" {
		http.Error(w, "missing "+ActHeader, http.StatusForbidden)
		return
	}
	var command TaskCommand
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&command)
	if err == nil {
		err = command.Validate()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	id, err := p.o.Dispatch(ctx, command)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	p.o.Logger.Info("panel task", "id", id, "task", command.Task, "remote", r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"id": id, "task": command.Task})
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

// act runs one of the buttons' actions and answers {"text": ...} with
// 200, or with 404 for an unknown action and 409 when it could not be done.
func (p *panel) act(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(ActHeader) == "" {
		http.Error(w, "missing "+ActHeader, http.StatusForbidden)
		return
	}
	action := r.PathValue("action")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	text, err := p.o.Act(ctx, action)
	code := http.StatusOK
	switch {
	case errors.Is(err, ErrUnknownAction):
		code, text = http.StatusNotFound, err.Error()
	case err != nil:
		code, text = http.StatusConflict, err.Error()
	}
	p.o.Logger.Info("panel action", "action", action, "remote", r.RemoteAddr, "status", code, "text", text)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"text": text})
}

// set changes one setting as the panel: {"key": …, "value": …} sets it,
// {"key": …, "reset": true} returns it to the config's value, and
// {"key": …, "pin": true|false} pins it against the agent or lets it go.
// It answers {"text": …}: 200, or 400 for a bad value, 404 for an unknown
// key, 409 for a pinned one.
func (p *panel) set(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(ActHeader) == "" {
		http.Error(w, "missing "+ActHeader, http.StatusForbidden)
		return
	}
	var req struct {
		Key   string          `json:"key"`
		Value json.RawMessage `json:"value"`
		Reset bool            `json:"reset"`
		Pin   *bool           `json:"pin"`
	}
	code, text := http.StatusOK, ""
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		code, text = http.StatusBadRequest, "not a setting: "+err.Error()
	} else if err := p.change(req.Key, req.Value, req.Reset, req.Pin); err != nil {
		code, text = http.StatusBadRequest, err.Error()
		switch {
		case errors.Is(err, settings.ErrUnknownKey):
			code = http.StatusNotFound
		case errors.Is(err, settings.ErrPinned), errors.Is(err, settings.ErrNotAllowed):
			code = http.StatusConflict
		}
	} else {
		switch {
		case req.Reset:
			text = req.Key + " is back to the config's value"
		case req.Pin != nil && *req.Pin:
			text = req.Key + " is pinned"
		case req.Pin != nil:
			text = req.Key + " is no longer pinned"
		default:
			text = req.Key + " is now " + string(req.Value)
		}
	}
	p.o.Logger.Info("panel setting", "key", req.Key, "value", string(req.Value), "remote", r.RemoteAddr, "status", code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"text": text})
}

func (p *panel) change(key string, value json.RawMessage, reset bool, pin *bool) error {
	switch {
	case reset:
		return p.o.Settings.Reset(key, settings.ByPanel)
	case pin != nil && *pin:
		return p.o.Settings.Pin(key, settings.ByPanel)
	case pin != nil:
		return p.o.Settings.Unpin(key, settings.ByPanel)
	}
	return p.o.Settings.Set(key, value, settings.ByPanel)
}

// models answers {"models": [...]} or {"error": …}, always with 200: a
// backend that cannot list its models leaves the page a text field.
func (p *panel) models(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{"models": []string{}}
	if p.o.Models == nil {
		out["error"] = "the bot has no LLM voice"
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if names, err := p.o.Models(ctx); err != nil {
			out["error"] = err.Error()
		} else {
			out["models"] = names
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// say passes {"as": owner, "text": …} to the bot as that owner's words.
// The bot answers in the feed, as "said to the panel".
func (p *panel) say(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(ActHeader) == "" {
		http.Error(w, "missing "+ActHeader, http.StatusForbidden)
		return
	}
	var req struct {
		As   string `json:"as"`
		Text string `json:"text"`
	}
	code, text := http.StatusOK, ""
	switch err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&req); {
	case err != nil:
		code, text = http.StatusBadRequest, err.Error()
	case strings.TrimSpace(req.Text) == "":
		code, text = http.StatusBadRequest, "nothing to say"
	default:
		// Logged first, so the feed shows the words before the answer.
		p.o.Logger.Info("panel said", "as", req.As, "text", req.Text, "remote", r.RemoteAddr)
		if err := p.o.Say(req.As, req.Text); err != nil {
			code, text = http.StatusConflict, err.Error()
			p.o.Logger.Info("panel said: refused", "as", req.As, "err", err)
		}
	}
	writeJSON(w, code, map[string]string{"text": text})
}

// Day is one audit file, for the page's day picker.
type Day struct {
	Day   string `json:"day"`
	Bytes int64  `json:"bytes"`
}

// days lists the audit files, newest first.
func (p *panel) days(w http.ResponseWriter, r *http.Request) {
	out := []Day{}
	files := audit.Files(p.o.AuditDir)
	for i := len(files) - 1; i >= 0; i-- {
		if day := audit.Day(files[i]); day != "" {
			var size int64
			if st, err := os.Stat(files[i]); err == nil {
				size = st.Size()
			}
			out = append(out, Day{Day: day, Bytes: size})
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// dayFile is the audit file for day, which must be a date, so a request
// cannot name any other file.
func (p *panel) dayFile(day string) (string, error) {
	if _, err := time.Parse(time.DateOnly, day); err != nil {
		return "", fmt.Errorf("not a day: %q", day)
	}
	return filepath.Join(p.o.AuditDir, audit.Prefix+day+audit.Suffix), nil
}

// TimelineLimit is how many records /api/timeline returns at most.
const TimelineLimit = 2000

// Timeline is a stretch of a day: its newest records before a time, oldest
// first, and whether there are older ones.
type Timeline struct {
	Events []Event `json:"events"`
	More   bool    `json:"more"`
}

// timeline answers ?day=YYYY-MM-DD[&before=RFC 3339&skip=N] with the day's
// newest TimelineLimit records up to that time, without state or
// perception. skip is how many records at exactly that time the page has
// already, so a page boundary between records of one instant loses none.
func (p *panel) timeline(w http.ResponseWriter, r *http.Request) {
	path, err := p.dayFile(r.URL.Query().Get("day"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f := audit.Filter{}
	if b := r.URL.Query().Get("before"); b != "" {
		if f.Until, err = time.Parse(time.RFC3339Nano, b); err != nil {
			http.Error(w, "before: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	recs, err := audit.ReadDay(path, f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if skip, _ := strconv.Atoi(r.URL.Query().Get("skip")); skip > 0 {
		// The page has the last skip records at f.Until; they are the
		// newest ones there, so the tail.
		n := 0
		for i := len(recs) - 1; i >= 0 && n < skip && recs[i].At.Equal(f.Until); i-- {
			n++
		}
		recs = recs[:len(recs)-n]
	}
	out := Timeline{Events: []Event{}}
	if len(recs) > TimelineLimit {
		recs, out.More = recs[len(recs)-TimelineLimit:], true
	}
	for _, rec := range recs {
		out.Events = append(out.Events, event(rec))
	}
	writeJSON(w, http.StatusOK, out)
}

// task answers ?id=t-3&at=RFC 3339 with what concerns that task in the
// controller run that was going at that time: ids start again with each
// run, so t-3 of an earlier run is another task.
func (p *panel) task(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	at, err := time.Parse(time.RFC3339Nano, r.URL.Query().Get("at"))
	if id == "" || err != nil {
		http.Error(w, "want id and at", http.StatusBadRequest)
		return
	}
	path, _ := p.dayFile(at.Local().Format(time.DateOnly))
	recs, err := audit.ReadDay(path, audit.Filter{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	// A run that began before midnight began in the day before's file, and
	// its task may have too.
	if !slices.ContainsFunc(recs, func(r audit.Record) bool { return r.Kind == audit.KindStart && !r.At.After(at) }) {
		prev, _ := p.dayFile(at.Local().AddDate(0, 0, -1).Format(time.DateOnly))
		if earlier, err := audit.ReadDay(prev, audit.Filter{}); err == nil {
			recs = append(earlier, recs...)
		}
	}
	// A task that runs past midnight goes on in the next day's file.
	next, _ := p.dayFile(at.Local().AddDate(0, 0, 1).Format(time.DateOnly))
	if more, err := audit.ReadDay(next, audit.Filter{}); err == nil {
		recs = append(recs, more...)
	}
	// The run is from the last start at or before at to the next one. It is
	// cut out first, so the task's records in other runs cannot stretch
	// the window the filter looks for its cause in.
	from, to := 0, len(recs)
	for i, rec := range recs {
		if rec.Kind != audit.KindStart {
			continue
		}
		if !rec.At.After(at) {
			from = i
		} else {
			to = i
			break
		}
	}
	out := []Event{}
	for _, rec := range (audit.Filter{Task: id}).Apply(recs[from:to]) {
		out = append(out, event(rec))
	}
	writeJSON(w, http.StatusOK, out)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// Event is a record as the page gets it: the record, the timeline's line
// for it, and a category for the page's filters and map.
type Event struct {
	audit.Record
	Line string `json:"line"`
	Cat  string `json:"cat"`
	// Task is the task the record is about, if any (audit.TaskID).
	Task string `json:"task,omitempty"`
}

// Categories of Event.
const (
	CatChat       = "chat"
	CatTask       = "task"
	CatTrace      = "trace"
	CatDiagnostic = "diagnostic"
	CatDecision   = "decision"
	CatState      = "state"
	CatPerception = "perception"
	CatOther      = "other"
)

func event(r audit.Record) Event {
	return Event{Record: r, Line: audit.Line(r), Cat: category(r), Task: audit.TaskID(r)}
}

func category(r audit.Record) string {
	switch r.Kind {
	case audit.KindLog:
		// The panel's words and the bot's answers to them are chat.
		if r.Msg == "panel said" || r.Msg == "panel said: refused" || r.Msg == "said to the panel" {
			return CatChat
		}
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
			var trace protocol.Trace
			if json.Unmarshal(r.Frame, &trace) == nil && ((trace.Source == "perception" && trace.Msg == "scan") ||
				(trace.Source == "navigation" && trace.Msg == "obstacle cache") || (trace.Source == "terrain" && trace.Msg == "cache")) {
				return CatDiagnostic
			}
			return CatTrace
		case protocol.TypeState:
			return CatState
		case protocol.TypePerception, protocol.TypeLayout, protocol.TypeTerrain:
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
