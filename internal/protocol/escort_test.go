package protocol

import (
	"encoding/json"
	"testing"
)

func TestEscortArgsAndStatus(t *testing.T) {
	for _, raw := range []string{`{}`, `{"player":" "}`, `{"player":"Nimah","distance":9}`, `{"player":"Nimah","protect":[1]}`, `{"player":"Nimah","collect":"yes"}`} {
		req := TaskRequest{Task: TaskEscort, Args: json.RawMessage(raw)}
		if _, err := req.DecodeArgs(); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	req := TaskRequest{Task: TaskEscort, Args: json.RawMessage(`{"player":"Nimah","protect":["Nimah","Sothr"]}`)}
	got, err := req.DecodeArgs()
	if err != nil {
		t.Fatal(err)
	}
	args := got.(*EscortArgs)
	if args.Player != "Nimah" || len(args.Protect) != 2 || args.Collect != nil || args.Defend != nil {
		t.Fatal(args)
	}
}
