package home

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func baseFixture(t *testing.T, s *Store) Base {
	t.Helper()
	now := time.Now().UTC()
	for _, p := range []Waypoint{{Name: "bedroom", Kind: "rest", Pos: protocol.Vec3{-25, 10, 0}}, {Name: "warehouse", Kind: "storage", Pos: protocol.Vec3{25, 10, 0}}} {
		p.At = now
		if err := s.EditWaypoint("one", "", &p, nil); err != nil {
			t.Fatal(err)
		}
	}
	b := Base{Name: "camp", Center: protocol.Vec3{0, 10, 0}, Radius: 60, DurationS: 600, TravelM: 800, Return: "bedroom", Sleeping: "bedroom", Storage: []string{"warehouse"}, At: now, SetBy: "owner"}
	if err := s.EditBase("one", b.Name, &b, nil, false); err != nil {
		t.Fatal(err)
	}
	return b
}
func TestBasePlacesSeparateBuildingsPersistenceAndLegacyFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	b := baseFixture(t, s)
	s.SetHome("one", Home{Pos: protocol.Vec3{1000, 0, 0}, At: b.At})
	p, err := ResolveBase(b, s.Waypoints("one"), nil)
	if err != nil || p.Sleeping.Pos.FlatDistance(p.Storage[0].Pos) != 50 {
		t.Fatal(p, err)
	}
	h, ok := s.WorkHome("one")
	if !ok || h.Pos != p.Sleeping.Pos {
		t.Fatal("return did not resolve", h)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.ActiveBase("one") == nil || s.ActiveBase("two") != nil {
		t.Fatal("lost base or world leak")
	}
	copy := s.ActiveBase("one")
	copy.Storage[0] = "changed"
	if s.ActiveBase("one").Storage[0] != "warehouse" {
		t.Fatal("caller mutated base")
	}
	wp, _ := s.Waypoint("one", "bedroom")
	next := wp
	next.Name = "other bedroom"
	next.At = next.At.Add(time.Second)
	if err = s.EditWaypoint("one", wp.Name, &next, &wp.At); err != nil {
		t.Fatal(err)
	}
	if _, ok = s.WorkHome("one"); ok {
		t.Fatal("broken return binding fell back to old home")
	}
	if _, err = ResolveBase(b, s.Waypoints("one"), nil); err == nil {
		t.Fatal("rename silently redirected")
	}
	if err = s.EditBase("one", b.Name, nil, &b.At, true); err != nil {
		t.Fatal(err)
	}
	if h, ok = s.WorkHome("one"); !ok || h.Pos[0] != 1000 {
		t.Fatal("legacy home not restored", h)
	}
}
func TestBaseRevisionInvalidBoundsAndSaveRollback(t *testing.T) {
	s := Memory()
	b := baseFixture(t, s)
	next := b
	next.Radius = 80
	next.At = b.At.Add(time.Second)
	if err := s.EditBase("one", b.Name, &next, nil, false); !errors.Is(err, ErrWaypointConflict) {
		t.Fatal("missing revision accepted", err)
	}
	if err := s.EditBase("one", b.Name, &next, &b.At, false); err != nil {
		t.Fatal(err)
	}
	if err := s.EditBase("one", b.Name, &b, &b.At, false); !errors.Is(err, ErrWaypointConflict) {
		t.Fatal("stale write accepted", err)
	}
	bad := next
	bad.Radius = 10
	bad.At = next.At.Add(time.Second)
	if err := s.EditBase("one", b.Name, &bad, &next.At, false); err == nil {
		t.Fatal("out-of-scope roles accepted")
	}
	block := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(block, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	s.path = filepath.Join(block, "memory.json")
	bad = next
	bad.Radius = 90
	bad.At = next.At.Add(time.Second)
	if err := s.EditBase("one", b.Name, &bad, &next.At, false); err == nil {
		t.Fatal("save succeeded")
	}
	if s.ActiveBase("one").Radius != 80 {
		t.Fatal("failed save changed memory")
	}
}
func TestRefreshMultiplePlacesOwnerProtectionSuppressionAndOtherBase(t *testing.T) {
	s := Memory()
	b := baseFixture(t, s)
	now := b.At.Add(time.Second)
	old := Waypoint{Name: "old storage", Kind: "storage", Pos: protocol.Vec3{500, 10, 0}, At: now}
	if err := s.EditWaypoint("one", "", &old, nil); err != nil {
		t.Fatal(err)
	}
	candidates := []Waypoint{{Kind: "storage", Pos: protocol.Vec3{25, 10, 0}, Radius: 6, At: now, Entity: "chest:owner"}, {Kind: "storage", Pos: protocol.Vec3{0, 10, 20}, Radius: 6, At: now, Entity: "chest:a"}, {Kind: "storage", Pos: protocol.Vec3{0, 10, -20}, Radius: 6, At: now, Entity: "chest:b"}}
	changes, err := s.RefreshBasePlaces("one", b, candidates)
	if err != nil || len(changes.Added) != 2 || len(changes.Protected) != 2 {
		t.Fatal(changes, err)
	}
	oldNow, _ := s.Waypoint("one", old.Name)
	if oldNow.Pos != old.Pos {
		t.Fatal("old base moved")
	}
	wp, _ := s.Waypoint("one", changes.Added[0])
	edited := wp
	edited.Pos[0] += 1
	edited.At = now.Add(time.Second)
	if err = s.EditWaypoint("one", wp.Name, &edited, &wp.At); err != nil {
		t.Fatal(err)
	}
	for i := range candidates {
		candidates[i].At = now.Add(2 * time.Second)
	}
	if _, err = s.RefreshBasePlaces("one", b, candidates); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Waypoint("one", wp.Name)
	if got.Source != "owner" || got.Pos != edited.Pos {
		t.Fatal("owner edit overwritten", got)
	}
	if err = s.EditWaypoint("one", wp.Name, nil, &got.At); err != nil {
		t.Fatal(err)
	}
	more := candidates[0]
	more.Pos = protocol.Vec3{30, 10, 25}
	more.Entity = "chest:new"
	changes, err = s.RefreshBasePlaces("one", b, []Waypoint{more})
	if err != nil || len(changes.Added) != 0 {
		t.Fatal("suppressed role recreated", changes, err)
	}
	next := b
	next.At = now.Add(4 * time.Second)
	if err = s.EditBase("one", b.Name, &next, &b.At, false); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RefreshBasePlaces("one", b, candidates); !errors.Is(err, ErrWaypointConflict) {
		t.Fatal("stale base refresh accepted", err)
	}
}

func TestRefreshMovedEntityAndFailedSaveKeepHonestPlaces(t *testing.T) {
	s := Memory()
	b := baseFixture(t, s)
	c := Waypoint{Kind: "wood", Pos: protocol.Vec3{0, 10, 20}, Radius: 15, At: b.At.Add(time.Second), Entity: "tree:1"}
	changes, err := s.RefreshBasePlaces("one", b, []Waypoint{c})
	if err != nil || len(changes.Added) != 1 {
		t.Fatal(changes, err)
	}
	name := changes.Added[0]
	c.Pos[0] = 20
	c.At = c.At.Add(time.Second)
	changes, err = s.RefreshBasePlaces("one", b, []Waypoint{c})
	w, _ := s.Waypoint("one", name)
	if err != nil || len(changes.Updated) != 1 || w.Pos != c.Pos {
		t.Fatal("moved entity stayed at old position", w, changes, err)
	}
	block := filepath.Join(t.TempDir(), "blocked")
	if err = os.WriteFile(block, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	s.path = filepath.Join(block, "memory.json")
	c.Pos[0] = 25
	c.At = c.At.Add(time.Second)
	if _, err = s.RefreshBasePlaces("one", b, []Waypoint{c}); err == nil {
		t.Fatal("save unexpectedly succeeded")
	}
	got, _ := s.Waypoint("one", name)
	if got != w {
		t.Fatal("failed refresh mutated memory", got, w)
	}
	s.path = ""
	edited := w
	edited.Base = ""
	edited.At = edited.At.Add(time.Second)
	if err = s.EditWaypoint("one", w.Name, &edited, &w.At); err != nil {
		t.Fatal(err)
	}
	c.Pos = w.Pos
	c.At = c.At.Add(time.Second)
	changes, err = s.RefreshBasePlaces("one", b, []Waypoint{c})
	if err != nil || len(changes.Added) != 0 || len(changes.Protected) != 1 {
		t.Fatal("owner edit lost protection when provenance omitted", changes, err)
	}
}
