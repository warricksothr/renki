package protocol

import (
	"encoding/json"
	"testing"
)

func TestHarvestExclusionsValidateBounds(t *testing.T) {
	for _, raw := range []string{
		`{"entity":"1:2","exclude":[{"center":[0,0,0],"radius":60}]}`,
		`{"entity":"1:2","exclude":[]}`,
	} {
		req := TaskRequest{Task: TaskHarvest, Args: json.RawMessage(raw)}
		if _, err := req.DecodeArgs(); err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{
		`{"entity":"1:2","exclude":[{"center":[0,0,0],"radius":0}]}`,
		`{"entity":"1:2","exclude":[{"radius":60}]}`,
		`{"entity":"1:2","exclude":[{"center":[0,0,0,1],"radius":60}]}`,
		`{"entity":"1:2","exclude":[null]}`,
		`{"entity":"1:2","exclude":[{"center":[null,0,0],"radius":60}]}`,
		`{"entity":"1:2","exclude":[{"center":[0,0,0],"radius":251}]}`,
		`{"entity":"1:2","exclude":[{"center":[1000001,0,0],"radius":60}]}`,
		`{"entity":"1:2","exclude":[{"center":[0,0,0],"radius":60,"typo":true}]}`,
	} {
		req := TaskRequest{Task: TaskHarvest, Args: json.RawMessage(raw)}
		if _, err := req.DecodeArgs(); err == nil {
			t.Fatal("invalid exclusion accepted", raw)
		}
	}
	req, err := NewTask("t", TaskHarvest, HarvestArgs{Entity: "1:2", Exclude: make([]HarvestExclusion, 17)}, 60)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = req.DecodeArgs(); err == nil {
		t.Fatal("unbounded exclusions accepted")
	}
}
