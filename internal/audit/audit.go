// Package audit is the controller's audit log: everything the bot saw,
// decided and did, in order, so a session can be reasoned about afterwards.
//
// It is JSON Lines, one file per local day (audit-2026-10-28.jsonl),
// appended to and never rewritten, and files older than KeepDays are
// removed. Each line is a Record:
//
//   - "start": the controller started, with its version and configuration.
//   - "frame": a bridge frame, as sent ("out") or received ("in"). Pings and
//     pongs are left out; the token in a hello is redacted.
//   - "log": a controller log record, at debug level and up. These are the
//     controller's decisions: commands heard, refusals, tasks sent,
//     connections opened and closed.
//
// The plugin's own decisions arrive as `trace` frames.
//
// The log also keeps its recent records in memory and hands every new one
// to subscribers, which is how the control panel watches the bot live. With
// no directory it does only that.
package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Kinds of Record.
const (
	KindStart = "start"
	KindFrame = "frame"
	KindLog   = "log"
)

// Directions of a frame.
const (
	In  = "in"
	Out = "out"
)

// Record is one line of the audit log.
type Record struct {
	At   time.Time `json:"at"`
	Kind string    `json:"kind"`
	// Conn is the bridge connection a frame or log record belongs to.
	Conn uint64 `json:"conn,omitempty"`
	// Dir is In or Out, for a frame.
	Dir string `json:"dir,omitempty"`
	// Frame is the frame as it went over the wire.
	Frame json.RawMessage `json:"frame,omitempty"`
	// Bad holds a received frame that was not JSON, as text.
	Bad   string         `json:"bad,omitempty"`
	Level string         `json:"level,omitempty"`
	Msg   string         `json:"msg,omitempty"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// Log writes the audit log. Its methods are safe for concurrent use, and a
// nil *Log discards everything.
type Log struct {
	dir  string
	keep int
	now  func() time.Time

	mu   sync.Mutex
	f    *os.File
	day  string
	errs int

	// recent holds the newest records but state and perception, which
	// come too often to keep many of; last holds the newest of each, and
	// trail one state a second for a few minutes, for the panel's map.
	recent []Record
	last   map[string]Record
	trail  []Record
	subs   map[chan Record]struct{}
	missed int
}

// Recent is how many records RecentRecords keeps, beside the newest state
// and perception and the trail.
const Recent = 500

// Trail is how many states RecentRecords keeps, one a second at most.
const Trail = 300

// Prefix and Suffix frame a day's file name.
const (
	Prefix = "audit-"
	Suffix = ".jsonl"
)

// Open starts an audit log in dir, creating it if need be. keepDays is how
// many days of files to keep, today included; 0 keeps everything. An empty
// dir writes no files but still keeps recent records and feeds
// subscribers. now may be nil.
func Open(dir string, keepDays int, now func() time.Time) (*Log, error) {
	if now == nil {
		now = time.Now
	}
	l := &Log{dir: dir, keep: keepDays, now: now, last: map[string]Record{}, subs: map[chan Record]struct{}{}}
	if dir == "" {
		return l, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.rotate(now()); err != nil {
		return nil, err
	}
	return l, nil
}

// Subscribe returns a channel that gets every record written from now on,
// and a function that ends the subscription. A subscriber that falls
// behind misses records rather than holding up the log.
func (l *Log) Subscribe() (<-chan Record, func()) {
	ch := make(chan Record, 256)
	l.mu.Lock()
	l.subs[ch] = struct{}{}
	l.mu.Unlock()
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			l.mu.Lock()
			delete(l.subs, ch)
			l.mu.Unlock()
		})
	}
}

// RecentRecords returns the newest records, oldest first, with the newest
// state and perception among them.
func (l *Log) RecentRecords() []Record {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Record, 0, len(l.recent)+len(l.last)+len(l.trail))
	out = append(out, l.recent...)
	out = append(out, l.trail...)
	for typ, r := range l.last {
		if typ == "state" && len(l.trail) > 0 && l.trail[len(l.trail)-1].At.Equal(r.At) {
			continue
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out
}

// Missed is how many records subscribers have missed by falling behind.
func (l *Log) Missed() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.missed
}

// remember keeps r among the recent records and hands it to subscribers.
// l.mu is held.
func (l *Log) remember(r Record) {
	switch typ := frameType(r); typ {
	case "state", "perception":
		l.last[typ] = r
		if typ == "state" && (len(l.trail) == 0 || r.At.Sub(l.trail[len(l.trail)-1].At) >= time.Second) {
			l.trail = append(l.trail, r)
			if len(l.trail) > Trail {
				l.trail = append(l.trail[:0], l.trail[len(l.trail)-Trail:]...)
			}
		}
	default:
		l.recent = append(l.recent, r)
		if len(l.recent) > Recent {
			l.recent = append(l.recent[:0], l.recent[len(l.recent)-Recent:]...)
		}
	}
	for ch := range l.subs {
		select {
		case ch <- r:
		default:
			l.missed++
		}
	}
}

// Close closes the current file.
func (l *Log) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f = nil
	return err
}

// Write appends r, stamping At when it is zero.
func (l *Log) Write(r Record) {
	if l == nil {
		return
	}
	if r.At.IsZero() {
		r.At = l.now()
	}
	line, err := encode(r)
	if err != nil {
		line, _ = encode(Record{At: r.At, Kind: KindLog, Level: "ERROR", Msg: "audit: unencodable record",
			Attrs: map[string]any{"kind": r.Kind, "err": err.Error()}})
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.remember(r)
	if l.dir == "" {
		return
	}
	if err := l.rotate(r.At); err == nil {
		_, err = l.f.Write(line)
		if err == nil {
			return
		}
	}
	// The audit log must never take the controller down; count the loss.
	l.errs++
}

// encode writes r as one line, leaving <, > and & as they are so chat
// reads as it was typed.
func encode(r Record) ([]byte, error) {
	var b bytes.Buffer
	e := json.NewEncoder(&b)
	e.SetEscapeHTML(false)
	if err := e.Encode(r); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Dropped is how many records could not be written.
func (l *Log) Dropped() int {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.errs
}

// Frame records a bridge frame. It is shaped to be a bridge.Options.Tap.
func (l *Log) Frame(conn uint64, dir string, data []byte) {
	if l == nil {
		return
	}
	var env struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(data, &env) != nil {
		l.Write(Record{Kind: KindFrame, Conn: conn, Dir: dir, Bad: string(data)})
		return
	}
	switch env.Type {
	case "ping", "pong":
		return
	case "hello":
		data = redact(data)
	}
	l.Write(Record{Kind: KindFrame, Conn: conn, Dir: dir, Frame: compact(data)})
}

// redact blanks the token in a hello.
func redact(data []byte) []byte {
	var m map[string]json.RawMessage
	if json.Unmarshal(data, &m) != nil {
		return data
	}
	if _, ok := m["token"]; !ok {
		return data
	}
	m["token"] = json.RawMessage(`"[redacted]"`)
	out, err := json.Marshal(m)
	if err != nil {
		return data
	}
	return out
}

func compact(data []byte) json.RawMessage {
	var b bytes.Buffer
	if json.Compact(&b, data) != nil {
		return data
	}
	return b.Bytes()
}

// rotate opens the file for t's day if it is not the one open, and prunes
// old files when it does. l.mu is held.
func (l *Log) rotate(t time.Time) error {
	day := t.Local().Format(time.DateOnly)
	if l.f != nil && day == l.day {
		return nil
	}
	if l.f != nil {
		l.f.Close()
		l.f = nil
	}
	f, err := os.OpenFile(filepath.Join(l.dir, Prefix+day+Suffix), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	l.f, l.day = f, day
	l.prune(t)
	return nil
}

// prune removes the files of days more than keep days before t's.
func (l *Log) prune(t time.Time) {
	if l.keep <= 0 {
		return
	}
	oldest := t.Local().AddDate(0, 0, -(l.keep - 1)).Format(time.DateOnly)
	for _, name := range Files(l.dir) {
		if day := Day(name); day != "" && day < oldest {
			os.Remove(name)
		}
	}
}

// Files lists the audit files in dir, oldest first.
func Files(dir string) []string {
	names, _ := filepath.Glob(filepath.Join(dir, Prefix+"*"+Suffix))
	sort.Strings(names)
	return names
}

// Day is the date in an audit file's name, or "" if it is not one.
func Day(name string) string {
	base := filepath.Base(name)
	if !strings.HasPrefix(base, Prefix) || !strings.HasSuffix(base, Suffix) {
		return ""
	}
	day := strings.TrimSuffix(strings.TrimPrefix(base, Prefix), Suffix)
	if _, err := time.Parse(time.DateOnly, day); err != nil {
		return ""
	}
	return day
}

// Handler returns a slog.Handler that writes records at level and above to
// the audit log as KindLog records.
func (l *Log) Handler(level slog.Leveler) slog.Handler {
	return &handler{log: l, level: level}
}

type handler struct {
	log   *Log
	level slog.Leveler
	attrs []slog.Attr
	group string
}

func (h *handler) Enabled(_ context.Context, level slog.Level) bool {
	return h.log != nil && level >= h.level.Level()
}

func (h *handler) Handle(_ context.Context, r slog.Record) error {
	attrs := map[string]any{}
	for _, a := range h.attrs {
		put(attrs, "", a)
	}
	r.Attrs(func(a slog.Attr) bool {
		put(attrs, h.group, a)
		return true
	})
	rec := Record{At: r.Time, Kind: KindLog, Level: r.Level.String(), Msg: r.Message}
	// The bridge tags its records with the connection; lift it out so a
	// log record and the frames around it share a key.
	if c, ok := attrs["conn"].(uint64); ok {
		rec.Conn = c
		delete(attrs, "conn")
	}
	if len(attrs) > 0 {
		rec.Attrs = attrs
	}
	h.log.Write(rec)
	return nil
}

func (h *handler) WithAttrs(as []slog.Attr) slog.Handler {
	n := *h
	n.attrs = append([]slog.Attr(nil), h.attrs...)
	for _, a := range as {
		if h.group != "" {
			a.Key = h.group + "." + a.Key
		}
		n.attrs = append(n.attrs, a)
	}
	return &n
}

func (h *handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	n := *h
	if h.group != "" {
		name = h.group + "." + name
	}
	n.group = name
	return &n
}

// put adds a to m under its key, prefixed by group, flattening groups with
// dots and turning values JSON cannot carry into text.
func put(m map[string]any, group string, a slog.Attr) {
	v := a.Value.Resolve()
	key := a.Key
	if group != "" {
		key = group + "." + key
	}
	switch v.Kind() {
	case slog.KindGroup:
		for _, g := range v.Group() {
			put(m, key, g)
		}
		return
	case slog.KindDuration:
		m[key] = v.Duration().String()
		return
	case slog.KindTime:
		m[key] = v.Time()
		return
	case slog.KindAny:
		switch x := v.Any().(type) {
		case error:
			m[key] = x.Error()
		case fmt.Stringer:
			m[key] = x.String()
		case json.Marshaler, []string, map[string]any, []any:
			m[key] = x
		default:
			if _, err := json.Marshal(x); err != nil {
				m[key] = fmt.Sprint(x)
			} else {
				m[key] = x
			}
		}
		return
	}
	m[key] = v.Any()
}
