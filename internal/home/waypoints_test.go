package home

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestWaypointsPersistWorldsEditsAndDeletedAutomaticRoles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	automatic := Waypoint{Name: "storage", Kind: "storage", Pos: protocol.Vec3{1, 36, 3}, Radius: 6, At: now, Evidence: "Observed chest"}
	if err := s.DiscoverWaypoints("one", []Waypoint{automatic}); err != nil {
		t.Fatal(err)
	}
	if err := s.DiscoverWaypoints("two", []Waypoint{automatic}); err != nil {
		t.Fatal(err)
	}
	old, _ := s.Waypoint("one", "storage")
	edited := old
	edited.Name = "The East Store"
	edited.Pos = protocol.Vec3{10, 36, 20}
	edited.Radius = 12
	edited.At = now.Add(time.Second)
	edited.SetBy = "Nimah"
	if err := s.EditWaypoint("one", "storage", &edited, &old.At); err != nil {
		t.Fatal(err)
	}
	automatic.Pos[0] = 99
	if err := s.DiscoverWaypoints("one", []Waypoint{automatic}); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := s.Waypoint("one", "east-store")
	if !ok || got.Pos != edited.Pos || got.Radius != 12 || got.Source != "owner" || got.Evidence != "" {
		t.Fatal(got)
	}
	if _, ok := s.Waypoint("one", "storage"); ok {
		t.Fatal("recreated renamed automatic role")
	}
	if other, _ := s.Waypoint("two", "storage"); other.Pos[0] != 1 {
		t.Fatal(other)
	}
	if err := s.EditWaypoint("one", "east store", nil, &got.At); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DiscoverWaypoints("one", []Waypoint{automatic}); err != nil {
		t.Fatal(err)
	}
	if len(s.Waypoints("one")) != 0 {
		t.Fatal("deleted automatic role returned")
	}
}

func TestWaypointConflictValidationAndSaveRollback(t *testing.T) {
	s := Memory()
	now := time.Now().UTC()
	w := Waypoint{Name: "mine", Kind: "ore", Pos: protocol.Vec3{1, 2, 3}, At: now}
	if err := s.EditWaypoint("world", "", &w, nil); err != nil {
		t.Fatal(err)
	}
	edited := w
	edited.Pos[0] = 4
	edited.At = now.Add(time.Second)
	if err := s.EditWaypoint("world", "mine", &edited, &now); err != nil {
		t.Fatal(err)
	}
	if err := s.EditWaypoint("world", "mine", nil, &now); !errors.Is(err, ErrWaypointConflict) {
		t.Fatal(err)
	}
	invalid := []Waypoint{w, w, w, w}
	invalid[0].Radius = -1
	invalid[1].Pos[1] = math.NaN()
	invalid[2].Name = "home"
	invalid[3].Kind = "unrecognized"
	for _, bad := range invalid {
		if err := s.EditWaypoint("world", "mine", &bad, nil); err == nil {
			t.Fatal(bad)
		}
	}
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	s.path = filepath.Join(blocked, "memory.json")
	if err := s.EditWaypoint("world", "mine", nil, &edited.At); err == nil {
		t.Fatal("save should fail")
	}
	if got, ok := s.Waypoint("world", "mine"); !ok || got.Pos != edited.Pos {
		t.Fatal(got)
	}
	// A failed discovery cannot leave a phantom role in memory either.
	auto := Waypoint{Name: "wood", Kind: "wood", At: now}
	if err := s.DiscoverWaypoints("world", []Waypoint{auto}); err == nil {
		t.Fatal("save should fail")
	}
	if _, ok := s.Waypoint("world", "wood"); ok {
		t.Fatal("failed save leaked")
	}
}
