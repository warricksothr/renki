package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// session is a stuck follow as the audit log records it.
func session(t *testing.T) []Record {
	t.Helper()
	base := time.Date(2026, 10, 28, 16, 42, 0, 0, time.Local)
	at := func(s float64) time.Time { return base.Add(time.Duration(s * float64(time.Second))) }
	frame := func(s float64, dir, f string) Record {
		return Record{At: at(s), Kind: KindFrame, Conn: 1, Dir: dir, Frame: json.RawMessage(f)}
	}
	return []Record{
		{At: at(0), Kind: KindStart, Msg: "renki controller test"},
		frame(1, In, `{"v":1,"type":"chat","seq":1,"t":0,"from":{"name":"Ylva"},"channel":"normal","text":"nice day","pos":[0,0,0],"dist":30}`),
		frame(10, In, `{"v":1,"type":"chat","seq":2,"t":0,"from":{"name":"Nimah","owner":true},"channel":"shout","text":"tas follow me","pos":[0,0,0],"dist":42}`),
		{At: at(10.001), Kind: KindLog, Conn: 0, Level: "INFO", Msg: "command", Attrs: map[string]any{"from": "Nimah", "command": "follow"}},
		frame(10.002, Out, `{"v":1,"type":"task","seq":3,"t":0,"id":"t-6","task":"follow","args":{"player":"Nimah"},"timeoutS":0}`),
		{At: at(10.003), Kind: KindLog, Level: "INFO", Msg: "task sent", Attrs: map[string]any{"id": "t-6", "task": "follow"}},
		frame(10.01, In, `{"v":1,"type":"task","seq":3,"t":0,"re":"t-6","status":"running"}`),
		frame(10.2, In, `{"v":1,"type":"state","seq":4,"t":0,"phase":"in_world","pos":[1,2,3],"vel":[3,0,4],"hp":47,"hpMax":60,"task":{"id":"t-6","task":"follow","status":"running"}}`),
		frame(13, In, `{"v":1,"type":"trace","seq":5,"t":0,"task":"t-6","source":"walker","msg":"stuck; trying Jump","data":{"attempt":1,"pos":[1,2,3]}}`),
		frame(20, Out, `{"v":1,"type":"task","seq":6,"t":0,"id":"t-7","task":"say","args":{"text":"hi"},"timeoutS":0}`),
		frame(40, In, `{"v":1,"type":"task","seq":7,"t":0,"re":"t-6","status":"failed","reason":"stuck","detail":"no progress after 9 recoveries"}`),
		frame(50, In, `{"v":1,"type":"chat","seq":8,"t":0,"from":{"name":"Ylva"},"channel":"normal","text":"later","pos":[0,0,0],"dist":30}`),
	}
}

func lines(recs []Record) string {
	var out []string
	for _, r := range recs {
		out = append(out, Line(r)[13:]) // drop the clock
	}
	return strings.Join(out, "\n")
}

func TestTaskFilterTellsTheTasksStory(t *testing.T) {
	got := lines(Filter{Task: "t-6"}.Apply(session(t)))
	want := strings.Join([]string{
		`[1] <- chat      Nimah (shout, 42 m): tas follow me`,
		`[ ] INFO  command command=follow from=Nimah`,
		`[1] -> task      t-6 follow {"player":"Nimah"}`,
		`[ ] INFO  task sent id=t-6 task=follow`,
		`[1] <- task      t-6 running`,
		`[1] <- trace     t-6 walker: stuck; trying Jump attempt=1 pos=(1, 2, 3)`,
		`[1] <- task      t-6 failed (stuck): no progress after 9 recoveries`,
	}, "\n")
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestTransferReceiptAppearsOnlyInItsOwnTaskStory(t *testing.T) {
	receipt := Record{Kind: KindFrame, At: time.Now(), Frame: json.RawMessage(`{"v":1,"type":"container","task":"t-6","entity":"chest","direction":"withdraw","moved":{"Honey":4}}`)}
	if TaskID(receipt) != "t-6" || len((Filter{Task: "t-6"}).Apply([]Record{receipt})) != 1 {
		t.Fatal("matching transfer receipt hidden")
	}
	if len((Filter{Task: "t-7"}).Apply([]Record{receipt})) != 0 {
		t.Fatal("unrelated transfer receipt included")
	}
}

func TestStateIsHiddenUnlessAskedForAndSampled(t *testing.T) {
	recs := session(t)
	if strings.Contains(lines(Filter{}.Apply(recs)), "state") {
		t.Error("state shown without -state")
	}
	got := lines(Filter{State: time.Second}.Apply(recs))
	if !strings.Contains(got, "<- state     in_world at (1, 2, 3), speed 5.0, hp 47/60, task t-6 follow running") {
		t.Errorf("state line missing:\n%s", got)
	}
}

func TestTimeWindow(t *testing.T) {
	recs := session(t)
	base := recs[0].At
	got := Filter{Since: base.Add(15 * time.Second), Until: base.Add(45 * time.Second)}.Apply(recs)
	if len(got) != 2 || !strings.Contains(Line(got[0]), "t-7 say") || !strings.Contains(Line(got[1]), "failed (stuck)") {
		t.Errorf("window:\n%s", lines(got))
	}
}

func TestReadFilesMergesDaysAndSkipsJunk(t *testing.T) {
	dir := t.TempDir()
	recs := session(t)
	write := func(name string, rs []Record, extra string) {
		var b strings.Builder
		for _, r := range rs {
			line, _ := encode(r)
			b.Write(line)
		}
		b.WriteString(extra)
		os.WriteFile(filepath.Join(dir, name), []byte(b.String()), 0o644)
	}
	write("audit-2026-10-29.jsonl", recs[6:], "")
	write("audit-2026-10-28.jsonl", recs[:6], "not a record\n")
	got, skipped, err := ReadFiles([]string{dir})
	if err != nil || skipped != 1 || len(got) != len(recs) {
		t.Fatalf("read %d records, skipped %d, err %v", len(got), skipped, err)
	}
	for i := range got {
		if !got[i].At.Equal(recs[i].At) {
			t.Fatalf("record %d out of order", i)
		}
	}
}
