package audit

import (
	"bufio"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func open(t *testing.T) (*Log, *clock, string) {
	t.Helper()
	dir := t.TempDir()
	c := &clock{t: time.Date(2026, 10, 28, 16, 0, 0, 0, time.Local)}
	l, err := Open(dir, 3, c.now)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	return l, c, dir
}

func read(t *testing.T, name string) []Record {
	t.Helper()
	f, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []Record
	s := bufio.NewScanner(f)
	for s.Scan() {
		var r Record
		if err := json.Unmarshal(s.Bytes(), &r); err != nil {
			t.Fatalf("line %q: %v", s.Text(), err)
		}
		out = append(out, r)
	}
	return out
}

func TestFramesAreRecordedRedactedAndWithoutPings(t *testing.T) {
	l, _, dir := open(t)
	l.Frame(1, In, []byte(`{"v":1,"type":"hello","seq":1,"t":0,"token":"sekrit","plugin":"0.1.0"}`))
	l.Frame(1, Out, []byte(`{"v":1,"type":"ping","seq":2,"t":0,"id":"p-1"}`))
	l.Frame(1, In, []byte(`{"v":1,"type":"pong","seq":2,"t":0,"re":"p-1"}`))
	l.Frame(1, In, []byte("{ \"v\": 1,\n \"type\": \"trace\", \"source\": \"walker\", \"msg\": \"stuck\" }"))
	l.Frame(2, In, []byte(`not json`))

	recs := read(t, filepath.Join(dir, "audit-2026-10-28.jsonl"))
	if len(recs) != 3 {
		t.Fatalf("got %d records: %+v", len(recs), recs)
	}
	if s := string(recs[0].Frame); strings.Contains(s, "sekrit") || !strings.Contains(s, `"token":"[redacted]"`) {
		t.Errorf("hello not redacted: %s", s)
	}
	if recs[1].Kind != KindFrame || recs[1].Dir != In || string(recs[1].Frame) != `{"v":1,"type":"trace","source":"walker","msg":"stuck"}` {
		t.Errorf("trace frame %+v (%s)", recs[1], recs[1].Frame)
	}
	if recs[2].Conn != 2 || recs[2].Bad != "not json" || recs[2].Frame != nil {
		t.Errorf("bad frame %+v", recs[2])
	}
}

func TestDailyFilesAndPruning(t *testing.T) {
	l, c, dir := open(t)
	for _, old := range []string{"audit-2026-10-20.jsonl", "audit-2026-10-26.jsonl", "notes.txt"} {
		os.WriteFile(filepath.Join(dir, old), []byte("{}\n"), 0o644)
	}
	l.Write(Record{Kind: KindStart, Msg: "day one"})
	c.t = c.t.Add(10 * time.Hour) // past midnight
	l.Write(Record{Kind: KindLog, Msg: "day two"})

	var names []string
	for _, f := range Files(dir) {
		names = append(names, filepath.Base(f))
	}
	// keepDays 3 on the 29th keeps the 27th to the 29th.
	if strings.Join(names, " ") != "audit-2026-10-28.jsonl audit-2026-10-29.jsonl" {
		t.Errorf("files %v", names)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Error("pruning removed a file that is not an audit file")
	}
	if r := read(t, filepath.Join(dir, "audit-2026-10-29.jsonl")); len(r) != 1 || r[0].Msg != "day two" {
		t.Errorf("day two %+v", r)
	}
}

func TestHandlerWritesLogRecords(t *testing.T) {
	l, _, dir := open(t)
	log := slog.New(l.Handler(slog.LevelDebug)).With("part", "bridge", "conn", uint64(4))
	log.Debug("plugin connected", "err", errors.New("boom"), "wait", 2*time.Second,
		slog.Group("pos", "x", 1.5), "tasks", []string{"stop"})
	slog.New(l.Handler(slog.LevelInfo)).Debug("below the level")

	recs := read(t, filepath.Join(dir, "audit-2026-10-28.jsonl"))
	if len(recs) != 1 {
		t.Fatalf("records %+v", recs)
	}
	r := recs[0]
	if r.Kind != KindLog || r.Level != "DEBUG" || r.Msg != "plugin connected" || r.Conn != 4 {
		t.Errorf("record %+v", r)
	}
	want := map[string]any{"part": "bridge", "err": "boom", "wait": "2s", "pos.x": 1.5, "tasks": []any{"stop"}}
	got, _ := json.Marshal(r.Attrs)
	exp, _ := json.Marshal(want)
	if string(got) != string(exp) {
		t.Errorf("attrs %s\nwant  %s", got, exp)
	}
}

func TestNilLogDiscards(t *testing.T) {
	var l *Log
	l.Frame(1, In, []byte(`{}`))
	l.Write(Record{})
	if l.Dropped() != 0 || l.Close() != nil {
		t.Error("a nil log should do nothing")
	}
}
