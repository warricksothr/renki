package protocol

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var senders = map[string]Sender{"plugin": FromPlugin, "controller": FromController}

func fixtures(t *testing.T, dir string) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("testdata", dir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatalf("no fixtures in testdata/%s", dir)
	}
	return paths
}

func read(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// sameJSON compares two documents as values, so spacing, key order and 3.0
// against 3 do not matter but a missing or changed field does.
func sameJSON(t *testing.T, want, got []byte) {
	t.Helper()
	var w, g any
	if err := json.Unmarshal(want, &w); err != nil {
		t.Fatalf("want: %v", err)
	}
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("got: %v", err)
	}
	if !reflect.DeepEqual(w, g) {
		t.Errorf("round trip changed the message\nwant %s\n got %s", compact(t, want), got)
	}
}

func compact(t *testing.T, data []byte) string {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	out, _ := json.Marshal(v)
	return string(out)
}

func TestFixturesRoundTrip(t *testing.T) {
	for name, from := range senders {
		for _, path := range fixtures(t, name) {
			t.Run(name+"/"+filepath.Base(path), func(t *testing.T) {
				data := read(t, path)
				msg, err := Decode(data, from)
				if err != nil {
					t.Fatalf("decode: %v", err)
				}
				out, err := Encode(msg)
				if err != nil {
					t.Fatalf("encode: %v", err)
				}
				sameJSON(t, data, out)
			})
		}
	}
}

func TestFixturesNameTheirType(t *testing.T) {
	// A fixture's file name starts with its type, so a reader of either test
	// suite can find the example for a message without opening every file.
	for name, from := range senders {
		for _, path := range fixtures(t, name) {
			msg, err := Decode(read(t, path), from)
			if err != nil {
				t.Fatalf("%s: %v", path, err)
			}
			if base := filepath.Base(path); !strings.HasPrefix(base, msg.MessageType()) {
				t.Errorf("%s holds a %q message", path, msg.MessageType())
			}
		}
	}
}

func TestUnknownFieldsAreIgnored(t *testing.T) {
	for name, from := range senders {
		for _, path := range fixtures(t, filepath.Join("lenient", name)) {
			t.Run(name+"/"+filepath.Base(path), func(t *testing.T) {
				msg, err := Decode(read(t, path), from)
				if err != nil {
					t.Fatalf("decode: %v", err)
				}
				out, err := Encode(msg)
				if err != nil {
					t.Fatalf("encode: %v", err)
				}
				sameJSON(t, read(t, filepath.Join("testdata", name, filepath.Base(path))), out)
			})
		}
	}
}

func TestRejects(t *testing.T) {
	for name, from := range senders {
		for _, path := range fixtures(t, filepath.Join("reject", name)) {
			base := filepath.Base(path)
			t.Run(name+"/"+base, func(t *testing.T) {
				msg, err := Decode(read(t, path), from)
				if err == nil {
					t.Fatalf("decoded as %T; want an error", msg)
				}
				var verr *VersionError
				switch {
				case strings.HasPrefix(base, "version_"):
					if !errors.As(err, &verr) {
						t.Errorf("got %v; want a *VersionError", err)
					}
				case strings.HasPrefix(base, "unknown_type"), strings.HasPrefix(base, "wrong_direction"):
					if !errors.Is(err, ErrUnknownType) {
						t.Errorf("got %v; want ErrUnknownType", err)
					}
				default:
					t.Fatalf("no expectation for %s; name it after the README", base)
				}
			})
		}
	}
}

func TestTaskFixturesHaveValidArgs(t *testing.T) {
	for _, path := range fixtures(t, "controller") {
		if !strings.HasPrefix(filepath.Base(path), "task_") {
			continue
		}
		msg, err := Decode(read(t, path), FromController)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if _, err := msg.(*TaskRequest).DecodeArgs(); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
}

func TestDecodeArgs(t *testing.T) {
	pos := Vec3{1, 2, 3}
	tests := []struct {
		task string
		args string
		want any
		err  error
	}{
		{TaskStop, ``, &StopArgs{}, nil},
		{TaskMoveTo, `{"to":[1,2,3]}`, &MoveToArgs{To: Target{Pos: &pos}}, nil},
		{TaskMoveTo, `{"to":"0:7"}`, &MoveToArgs{To: Target{Entity: "0:7"}}, nil},
		{TaskMoveTo, `{}`, nil, ErrBadArgs},
		{TaskMoveTo, `{"to":{"x":1}}`, nil, ErrBadArgs},
		{TaskFollow, `{"player":"Sothr"}`, &FollowArgs{Player: "Sothr"}, nil},
		{TaskFollow, `{"player":"Sothr","run":false}`, &FollowArgs{Player: "Sothr", Run: RunWalk}, nil},
		{TaskFollow, `{"player":"Sothr","run":"sometimes"}`, nil, ErrBadArgs},
		{TaskFollow, `{"distance":3}`, nil, ErrBadArgs},
		{TaskLookAt, `{}`, nil, ErrBadArgs},
		{TaskSay, `{"text":"hi","channel":"shout"}`, &SayArgs{Text: "hi", Channel: ChannelShout}, nil},
		{TaskSay, `{"text":"hi","channel":"whisper"}`, nil, ErrBadArgs},
		{TaskSay, `{"text":"hi","channel":"ping"}`, nil, ErrBadArgs},
		{TaskSay, `{"text":""}`, nil, ErrBadArgs},
		{TaskEmote, `{"name":"wave"}`, &EmoteArgs{Name: "wave"}, nil},
		{"harvest", `{"entity":"0:7001"}`, nil, ErrUnknownTask},
	}
	for _, tc := range tests {
		t.Run(tc.task+" "+tc.args, func(t *testing.T) {
			req := &TaskRequest{Task: tc.task, Args: json.RawMessage(tc.args)}
			got, err := req.DecodeArgs()
			if tc.err != nil {
				if !errors.Is(err, tc.err) {
					t.Fatalf("got %v, %v; want %v", got, err, tc.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
		})
	}
}

func TestNewTaskEncodes(t *testing.T) {
	d := 3.5
	req, err := NewTask("t-17", TaskFollow, &FollowArgs{Player: "Sothr", Distance: &d}, 0)
	if err != nil {
		t.Fatal(err)
	}
	req.Seq, req.T = 90, 18234.0
	out, err := Encode(req)
	if err != nil {
		t.Fatal(err)
	}
	sameJSON(t, read(t, "testdata/controller/task_follow.json"), out)
}

func TestTargetWithBothIsAnError(t *testing.T) {
	p := Vec3{}
	if _, err := json.Marshal(Target{Pos: &p, Entity: "0:1"}); err == nil {
		t.Fatal("want an error for a target that is both a position and an entity")
	}
}
