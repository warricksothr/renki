package protocol

import (
	"encoding/json"
	"os"
	"testing"
)

func TestInspectionArgumentsMatchSharedContract(t *testing.T) {
	raw, err := os.ReadFile("testdata/inspection-args.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Args  json.RawMessage `json:"args"`
		Valid bool            `json:"valid"`
	}
	if err = json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		_, err := (&TaskRequest{Task: TaskInspectContainer, Args: c.Args}).DecodeArgs()
		if (err == nil) != c.Valid {
			t.Errorf("%s: valid=%v, err=%v", c.Args, c.Valid, err)
		}
	}
	if Instant(TaskInspectContainer) {
		t.Fatal("inspection must own the body")
	}
}
