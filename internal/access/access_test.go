package access

import (
	"slices"
	"testing"
)

func TestProfilesSeparateCommandsFromRoutinesAndOwnersKeepAuthority(t *testing.T) {
	r := Rules{Profiles: map[string]string{"Pilot": "operator", "Friend": "companion", "Visitor": "guest", "Nimah": "guest"}}
	for _, c := range []struct {
		name               string
		cmd, locate, greet bool
	}{{"Pilot", true, true, true}, {"Friend", false, true, true}, {"Visitor", false, true, false}, {"Nimah", true, true, true}, {"Other", false, false, false}} {
		for permission, want := range map[Permission]bool{Command: c.cmd, Locate: c.locate, Greet: c.greet} {
			if got := r.Has([]string{"Nimah"}, c.name, permission); got != want {
				t.Fatalf("%s %s: %v", c.name, permission, got)
			}
		}
	}
	if got := r.Commanders([]string{"Nimah"}); !slices.Equal(got, []string{"Nimah", "Pilot"}) {
		t.Fatal(got)
	}
	for _, bad := range []Rules{{Profiles: map[string]string{" X": "operator"}}, {Profiles: map[string]string{"X": "admin"}}, {Profiles: map[string]string{"": "guest"}}} {
		if bad.Validate() == nil {
			t.Fatal(bad)
		}
	}
}
