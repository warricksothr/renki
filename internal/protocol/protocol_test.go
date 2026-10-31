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
		{TaskMoveTo, `{"to":[1,2,3],"dry":true}`, &MoveToArgs{To: Target{Pos: &pos}, Dry: true}, nil},
		{TaskMoveTo, `{"to":[1,2,3],"dry":"yes"}`, nil, ErrBadArgs},
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
		{TaskInteract, `{"entity":"1234:310","alt":true}`, &InteractArgs{Entity: "1234:310", Alt: true}, nil},
		{TaskInteract, `{"hold":true}`, nil, ErrBadArgs},
		{TaskInteract, `{"entity":"1234:310","sleep":true}`, &InteractArgs{Entity: "1234:310", Sleep: true}, nil},
		{TaskInteract, `{"entity":"1234:310","sleep":true,"hold":true}`, nil, ErrBadArgs},
		{TaskInteract, `{"entity":"1234:310","sleep":true,"alt":true}`, nil, ErrBadArgs},
		{TaskFlee, `{"from":"0:5521","toward":[1,2,3]}`, &FleeArgs{From: Target{Entity: "0:5521"}, Toward: &Target{Pos: &Vec3{1, 2, 3}}}, nil},
		{TaskFlee, `{"distance":20}`, nil, ErrBadArgs},
		{TaskFlee, `{"from":"0:1","distance":0}`, nil, ErrBadArgs},
		{TaskEquip, `{"item":"AxeStone"}`, &EquipArgs{Item: "AxeStone"}, nil},
		{TaskEquip, `{}`, nil, ErrBadArgs},
		{TaskUseItem, `{"item":"Raspberry"}`, &UseItemArgs{Item: "Raspberry"}, nil},
		{TaskUseItem, `{}`, nil, ErrBadArgs},
		{TaskUseItem, `{"item":7}`, nil, ErrBadArgs},
		{TaskUnequip, `{"slot":"hands"}`, &UnequipArgs{Slot: "hands"}, nil},
		{TaskUnequip, `{"slot":"tail"}`, nil, ErrBadArgs},
		{TaskDeposit, `{"container":"1:2","items":[{"prefab":"Wood","count":"all"}]}`, &TransferArgs{Container: "1:2", Items: []TransferItem{{Prefab: "Wood", Count: AllItems}}}, nil},
		{TaskWithdraw, `{"container":"1:2","items":[{"prefab":"Wood","count":5}]}`, &TransferArgs{Container: "1:2", Items: []TransferItem{{Prefab: "Wood", Count: 5}}}, nil},
		{TaskDeposit, `{"items":[{"prefab":"Wood","count":1}]}`, nil, ErrBadArgs},
		{TaskWithdraw, `{"container":"1:2","items":[]}`, nil, ErrBadArgs},
		{TaskDeposit, `{"container":"1:2","items":[{"prefab":"Wood","count":0}]}`, nil, ErrBadArgs},
		{TaskDeposit, `{"container":"1:2","items":[{"prefab":"Wood","count":-1}]}`, nil, ErrBadArgs},
		{TaskDeposit, `{"container":"1:2","items":[{"prefab":"Wood","count":1.5}]}`, nil, ErrBadArgs},
		{TaskDeposit, `{"container":"1:2","items":[{"prefab":"Wood","count":2147483648}]}`, nil, ErrBadArgs},
		{TaskDeposit, `{"container":"1:2","items":[{"prefab":"Wood","count":"5"}]}`, nil, ErrBadArgs},
		{TaskDeposit, `{"container":"1:2","items":[{"prefab":"Wood","count":1},{"prefab":"Wood","count":2}]}`, nil, ErrBadArgs},
		{TaskHarvest, `{"entity":"0:7001"}`, &HarvestArgs{Entity: "0:7001"}, nil},
		{TaskHarvest, `{"entity":"0:7001","collect":false}`, &HarvestArgs{Entity: "0:7001", Collect: func() *bool { v := false; return &v }()}, nil},
		{TaskHarvest, `{}`, nil, ErrBadArgs},
		{TaskAttack, `{"entity":"1:2"}`, &AttackArgs{Entity: "1:2"}, nil},
		{TaskAttack, `{"entity":"1:2","style":"auto"}`, &AttackArgs{Entity: "1:2", Style: "auto"}, nil},
		{TaskAttack, `{"entity":"1:2","style":"melee"}`, &AttackArgs{Entity: "1:2", Style: "melee"}, nil},
		{TaskAttack, `{"kite":true}`, &AttackArgs{Kite: true}, nil},
		{TaskAttack, `{"kite":true,"searchCenter":[80,30,0]}`, &AttackArgs{Kite: true, SearchCenter: &Vec3{80, 30, 0}}, nil},
		{TaskAttack, `{"kite":true,"searchCenter":null}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchCenter":[80,30,0],"searchAnchor":[0,30,0]}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchCenter":[80,30,0],"searchLimit":148}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchCenter":[80,30,0],"searchLimit":null}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchCenter":[80,30,0],"searchAnchor":[0,30,0],"searchLimit":100}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchCenter":[80,30,0],"searchAnchor":[0,30,0],"searchLimit":501}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchAnchor":[0,30,0],"searchLimit":148}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchCenter":[80,30,0],"searchAnchor":[0,null,0],"searchLimit":148}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchCenter":[80,30,0],"searchAnchor":[0,30,0],"searchLimit":"148"}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchCenter":[80,null,0]}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchCenter":{"entity":"1:2"}}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchCenter":[80,30,0],"searchRadius":60}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchCenter":[80,30,0],"searchSector":0}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchCenter":[80,30,0],"searchOffset":0}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"entity":"1:2","searchCenter":[80,30,0]}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"at":[0,30,0],"searchCenter":[80,30,0]}`, nil, ErrBadArgs},
		{TaskAttack, `{"entity":"1:2","searchCenter":[80,30,0]}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchRadius":60}`, &AttackArgs{Kite: true, SearchRadius: func() *float64 { v := 60.0; return &v }()}, nil},
		{TaskAttack, `{"kite":true,"searchSector":0}`, &AttackArgs{Kite: true, SearchSector: func() *int { v := 0; return &v }()}, nil},
		{TaskAttack, `{"kite":true,"searchSector":0,"searchOffset":60}`, &AttackArgs{Kite: true, SearchSector: func() *int { v := 0; return &v }(), SearchOffset: func() *float64 { v := 60.0; return &v }()}, nil},
		{TaskAttack, `{"kite":true,"searchSector":0,"searchOffset":61}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchSector":0,"searchOffset":-1}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchOffset":40}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchOffset":null}`, &AttackArgs{Kite: true}, nil},
		{TaskAttack, `{"kite":true,"searchSector":0,"searchOffset":60,"searchRadius":80}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchSector":0,"searchOffset":"40"}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchSector":7}`, &AttackArgs{Kite: true, SearchSector: func() *int { v := 7; return &v }()}, nil},
		{TaskAttack, `{"kite":true,"searchSector":8}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchSector":-1}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchSector":1.5}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchSector":"1"}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchSector":null}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"entity":"1:2","searchSector":1}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"at":[1,2,3],"searchSector":1}`, nil, ErrBadArgs},
		{TaskAttack, `{"entity":"1:2","searchSector":1}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchRadius":81}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"searchRadius":11}`, nil, ErrBadArgs},
		{TaskAttack, `{"entity":"1:2","searchRadius":60}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"at":[1,2,3],"searchRadius":60}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"at":[1,2,3]}`, &AttackArgs{Kite: true, At: &Vec3{1, 2, 3}}, nil},
		{TaskAttack, `{"entity":"1:2","kite":true}`, &AttackArgs{Entity: "1:2", Kite: true}, nil},
		{TaskAttack, `{"kite":"true"}`, nil, ErrBadArgs},
		{TaskAttack, `{"entity":"1:2","at":[1,2,3]}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"at":[1,2]}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"at":[1,2,3,4]}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"at":"1:2"}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"at":null}`, nil, ErrBadArgs},
		{TaskAttack, `{"kite":true,"at":[1,null,3]}`, nil, ErrBadArgs},
		{TaskAttack, `{}`, nil, ErrBadArgs},
		{TaskAttack, `{"entity":"   "}`, nil, ErrBadArgs},
		{TaskAttack, `{"entity":"1:2","style":"ranged"}`, nil, ErrBadArgs},
		{TaskAttack, `{"entity":5}`, nil, ErrBadArgs},
		{TaskHarvest, `{"entity":"0:7001","collect":"no"}`, nil, ErrBadArgs},
		{"craft", `{}`, nil, ErrUnknownTask},
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
