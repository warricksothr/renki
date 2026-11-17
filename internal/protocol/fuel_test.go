package protocol

import (
	"encoding/json"
	"testing"
)

func TestAddToRequiresBoundedExplicitFuelRequest(t *testing.T) {
	for _, raw := range []string{
		`{}`, `null`,
		`{"entity":"1:2","item":"Wood","count":0}`,
		`{"entity":"1:2","item":"Wood","count":17}`,
		`{"entity":"1:2","item":"Wood","count":1.5}`,
		`{"entity":"1:2","item":"Wood","count":"all"}`,
		`{"entity":"1:2","item":"Wood","count":null}`,
		`{"entity":" ","item":"Wood","count":2}`,
		`{"entity":"1:2","item":"Wood","count":2,"toggle":true}`,
	} {
		req := TaskRequest{Task: TaskAddTo, Args: json.RawMessage(raw)}
		if _, err := req.DecodeArgs(); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	req := TaskRequest{Task: TaskAddTo, Args: json.RawMessage(`{"entity":"1:2","item":"Wood","count":2}`)}
	args, err := req.DecodeArgs()
	if err != nil {
		t.Fatal(err)
	}
	if a := args.(*AddToArgs); a.Count != 2 || a.Entity != "1:2" || a.Item != "Wood" {
		t.Fatalf("%+v", a)
	}
	if Instant(TaskAddTo) {
		t.Fatal("fueling must own the body")
	}
}
