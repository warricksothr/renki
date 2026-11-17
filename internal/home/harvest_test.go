package home

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestResourceSitesRequireDryStandingRoots(t *testing.T) {
	water := 30.0
	for _, tc := range []struct {
		name, kind, prefab string
		y                  float64
		water              *float64
		want               string
	}{
		{"dry beech", "tree", "Beech1", 30.86, &water, "wood"},
		{"submerged captured log", "tree", "FirTree_oldLog", 25.29, &water, ""},
		{"dry static log", "tree", "FirTree_oldLog", 35, &water, ""},
		{"dynamic log", "log", "Beech_log", 35, &water, ""},
		{"stump", "stump", "Beech_Stub", 35, &water, ""},
		{"unknown water", "tree", "Beech1", 35, nil, ""},
		{"waterline", "tree", "Beech1", 30.2, &water, ""},
		{"dry rock", "rock", "Rock_3_frac", 32, &water, "stone"},
		{"wet captured rock", "rock", "Rock_3_frac", 28, &water, ""},
		{"dry ore", "rock", "rock_copper", 32, &water, "ore"},
		{"berries unaffected", "pickable", "RaspberryBush", 34, nil, "berries"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := protocol.PerceivedResource{Kind: tc.kind, Prefab: tc.prefab, Pos: protocol.Vec3{-216.17, tc.y, -489.48}, WaterLevel: tc.water}
			if got := ResourcePlaceKind(r); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
			places := ObservedPlaces(&protocol.Perception{Resources: []protocol.PerceivedResource{r}}, time.Now())
			if (len(places) > 0) != (tc.want != "") {
				t.Fatal(places)
			}
		})
	}
	for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if (protocol.PerceivedResource{Pos: protocol.Vec3{0, 40, 0}, WaterLevel: &bad}).DryRoot() {
			t.Fatal("nonfinite water accepted")
		}
	}
}

func TestBaseResourceMigrationPreservesOwnerRolesAndDiscovery(t *testing.T) {
	s := Memory()
	now := time.Now().UTC()
	seed := []Waypoint{
		{Name: "inside wood", Kind: "wood", Pos: protocol.Vec3{10, 35, 0}},
		{Name: "edge stone", Kind: "stone", Pos: protocol.Vec3{60, 35, 0}},
		{Name: "inside ore", Kind: "ore", Pos: protocol.Vec3{0, 35, 10}},
		{Name: "outside wood", Kind: "wood", Pos: protocol.Vec3{70, 35, 0}},
		{Name: "wet stone", Kind: "stone", Pos: protocol.Vec3{100, 28, 0}},
		{Name: "old log", Kind: "wood", Pos: protocol.Vec3{110, 25, 0}, Evidence: "Observed FirTree_oldLog"},
		{Name: "owner wood", Kind: "wood", Source: "owner", Pos: protocol.Vec3{15, 35, 0}},
		{Name: "bound wood", Kind: "wood", Pos: protocol.Vec3{20, 35, 0}},
		{Name: "unobserved stone", Kind: "stone", Pos: protocol.Vec3{150, 35, 0}},
	}
	if err := s.editWaypoints("one", func(w *waypointWorld) error {
		for _, p := range seed {
			if p.Source == "" {
				p.Source = "bot"
			}
			p.At = now
			w.Places[p.Name] = p
		}
		w.Suppressed = []string{"flint"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	b := baseFixture(t, s)
	b.Return = "bound wood"
	b.At = b.At.Add(time.Second)
	prior := s.ActiveBase("one")
	if err := s.EditBase("one", b.Name, &b, &prior.At, false); err != nil {
		t.Fatal(err)
	}
	water := 30.0
	seen := &protocol.Perception{Resources: []protocol.PerceivedResource{{Kind: "rock", Pos: protocol.Vec3{100, 28, 0}, WaterLevel: &water}}}
	if err := s.PruneHarvestPlaces("one", seen); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"inside wood", "edge stone", "inside ore", "wet stone", "old log"} {
		if _, ok := s.Waypoint("one", name); ok {
			t.Fatal("invalid marker survived", name)
		}
	}
	for _, name := range []string{"outside wood", "owner wood", "bound wood", "unobserved stone", "bedroom", "warehouse"} {
		if _, ok := s.Waypoint("one", name); !ok {
			t.Fatal("protected/unrelated marker lost", name)
		}
	}
	if !reflect.DeepEqual(s.mem.Waypoints["one"].Suppressed, []string{"flint"}) {
		t.Fatal("pruning changed suppression")
	}
	// Even callers supplying unfiltered candidates cannot recreate inside sites.
	if err := s.DiscoverWaypoints("one", []Waypoint{{Name: "stone", Kind: "stone", Pos: protocol.Vec3{0, 35, 0}, At: now}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Waypoint("one", "stone"); ok {
		t.Fatal("inside site recreated")
	}
	changes, err := s.RefreshBasePlaces("one", b, []Waypoint{{Kind: "wood", Pos: protocol.Vec3{0, 35, 0}, At: now, Entity: "tree"}})
	if err != nil || len(changes.Added) != 0 || changes.Skipped != 1 {
		t.Fatal(changes, err)
	}
	// No stone suppression was introduced; discovery outside this base still works.
	if err := s.DiscoverWaypoints("two", []Waypoint{{Name: "wood", Kind: "wood", Pos: protocol.Vec3{0, 35, 0}, At: now}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Waypoint("two", "wood"); !ok {
		t.Fatal("base leaked across worlds")
	}
	if err := s.DiscoverWaypoints("one", []Waypoint{{Name: "ore", Kind: "ore", Pos: protocol.Vec3{61, 35, 0}, At: now}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Waypoint("one", "ore"); !ok {
		t.Fatal("valid outside site suppressed")
	}
}

func TestPruningIsAtomicAndDoesNotWriteUnchangedMemories(t *testing.T) {
	s := Memory()
	if err := s.editWaypoints("one", func(w *waypointWorld) error {
		w.Places["old log"] = Waypoint{Name: "old log", Kind: "wood", Source: "bot", Evidence: "Observed FirTree_oldLog", At: time.Now()}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	block := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(block, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	s.path = filepath.Join(block, "memory.json")
	if err := s.PruneHarvestPlaces("one", nil); err == nil {
		t.Fatal("save succeeded")
	}
	if _, ok := s.Waypoint("one", "old log"); !ok {
		t.Fatal("failed save mutated memory")
	}
	// A clean world does not attempt a save, even when the path is unwritable.
	if err := s.PruneHarvestPlaces("clean", nil); err != nil {
		t.Fatal(err)
	}
}
