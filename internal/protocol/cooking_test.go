package protocol

import "testing"

func TestCookingArgumentsAreExplicitAndExcludeOtherUseModes(t *testing.T) {
	for _, args := range []string{
		`{"entity":"1:2","cook":"RawMeat"}`,
		`{"entity":"1:2","cook":"DeerMeat"}`,
		`{"entity":"1:2","cook":"NeckTail"}`,
	} {
		req := TaskRequest{Task: TaskInteract, Args: []byte(args)}
		if _, err := req.DecodeArgs(); err != nil {
			t.Fatal(args, err)
		}
	}
	for _, args := range []string{
		`{"entity":"1:2","cook":true}`,
		`{"entity":"1:2","cook":"Coal"}`,
		`{"entity":"1:2","cook":"CookedMeat"}`,
		`{"entity":"1:2","cook":"RawMeat","sleep":true}`,
		`{"entity":"1:2","cook":"RawMeat","repair":true}`,
		`{"entity":"1:2","cook":"RawMeat","hold":true}`,
		`{"entity":"1:2","cook":"RawMeat","alt":true}`,
	} {
		req := TaskRequest{Task: TaskInteract, Args: []byte(args)}
		if _, err := req.DecodeArgs(); err == nil {
			t.Fatal("accepted", args)
		}
	}
}
