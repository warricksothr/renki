package home

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestReconcilePositiveStructureIdentitiesAcrossRestart(t *testing.T) {
	for _, kind := range []string{protocol.StructureContainer, protocol.StructureBed, protocol.StructureDoor,
		protocol.StructureStation, protocol.StructureFire, protocol.StructurePortal, protocol.ResourcePickable} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "memory.json")
			s, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC()
			old := Landmark{Entity: "old", Kind: kind, Prefab: "same-prefab", Pos: protocol.Vec3{-37.09, 36.09, 3.42}, At: now}
			if err := s.RememberLandmarks("world", []Landmark{old}, now); err != nil {
				t.Fatal(err)
			}
			if err := s.RememberLandmarks("other", []Landmark{old}, now); err != nil {
				t.Fatal(err)
			}
			current := old
			current.Entity, current.At = "current", now.Add(time.Minute)
			if err := s.RememberLandmarks("world", []Landmark{current}, current.At); err != nil {
				t.Fatal(err)
			}
			s, err = Open(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := s.Landmarks("world"); len(got) != 1 || got[0].Entity != "current" || !got[0].At.Equal(current.At) {
				t.Fatal("superseded identity survived", got)
			}
			if got := s.Landmarks("other"); len(got) != 1 || got[0].Entity != "old" {
				t.Fatal("changed another world", got)
			}
		})
	}
}

func TestReconcileKeepsDistinctAmbiguousCoObservedAndNewerObjects(t *testing.T) {
	now := time.Now().UTC()
	old := Landmark{Entity: "old", Kind: protocol.StructureContainer, Prefab: "chest", At: now}
	current := old
	current.Entity, current.At = "current", now.Add(time.Minute)
	for _, mode := range []string{"floor", "nearby", "type", "kind", "unknown-prefab", "co-observed", "older-frame", "ambiguous-current", "ambiguous-old", "absence"} {
		t.Run(mode, func(t *testing.T) {
			s := Memory()
			initial := []Landmark{old}
			observed := []Landmark{current}
			switch mode {
			case "floor":
				observed[0].Pos[1] = 3
			case "nearby":
				observed[0].Pos[0] = .75
			case "type":
				observed[0].Prefab = "another-chest"
			case "kind":
				observed[0].Kind = protocol.StructureStation
			case "unknown-prefab":
				observed[0].Prefab = ""
			case "co-observed":
				both := old
				both.At = current.At
				observed = append(observed, both)
			case "older-frame":
				observed[0].At = now.Add(-time.Minute)
			case "ambiguous-current":
				another := current
				another.Entity = "another-current"
				observed = append(observed, another)
			case "ambiguous-old":
				another := old
				another.Entity = "another-old"
				initial = append(initial, another)
			case "absence":
				observed = nil
			}
			if err := s.RememberLandmarks("world", initial, now); err != nil {
				t.Fatal(err)
			}
			if err := s.RememberLandmarks("world", observed, current.At); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, l := range s.Landmarks("world") {
				found = found || l.Entity == "old"
			}
			if !found {
				t.Fatal("lost a distinct or unproven old object", s.Landmarks("world"))
			}
		})
	}
}

func TestReconciledChestContentsRetainProvenanceUntilActuallyOpened(t *testing.T) {
	s := Memory()
	now := time.Now().UTC()
	old := Landmark{Entity: "old", Kind: protocol.StructureContainer, Prefab: "chest", Pos: protocol.Vec3{2.1, 3.1, 4.1}, At: now}
	if err := s.RememberLandmarks("world", []Landmark{old}, now); err != nil {
		t.Fatal(err)
	}
	contents := RememberedContainer{Entity: "old", Prefab: "chest", Pos: old.Pos, Size: [2]int{8, 4},
		Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 20}}, At: now.Add(-time.Minute)}
	if err := s.RememberContainers("world", []RememberedContainer{contents}, now); err != nil {
		t.Fatal(err)
	}
	old.Entity, old.At = "current", now.Add(time.Second)
	if err := s.RememberLandmarks("world", []Landmark{old}, old.At); err != nil {
		t.Fatal(err)
	}
	got := s.RememberedContainers("world")
	if len(got) != 1 || got[0].Entity != "current" || got[0].ObservedEntity != "old" ||
		!got[0].At.Equal(contents.At) || got[0].Inventory().Count("Wood") != 20 || got[0].Fresh(now, time.Hour) {
		t.Fatal("old contents were lost or promoted to current evidence", got)
	}
	contents.Entity, contents.At, contents.Items = "current", now.Add(time.Minute), nil
	if err := s.RememberContainers("world", []RememberedContainer{contents}, contents.At); err != nil {
		t.Fatal(err)
	}
	got = s.RememberedContainers("world")
	if got[0].ObservedEntity != "" || got[0].Inventory().Count("Wood") != 0 || !got[0].Fresh(contents.At, time.Hour) {
		t.Fatal("fresh empty snapshot did not replace the hint", got)
	}
}

func TestReconciliationRollsBackLandmarksAndContentsTogether(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	old := Landmark{Entity: "old", Kind: protocol.StructureContainer, Prefab: "chest", At: now}
	if err := s.RememberLandmarks("world", []Landmark{old}, now); err != nil {
		t.Fatal(err)
	}
	if err := s.RememberContainers("world", []RememberedContainer{{Entity: "old", Prefab: "chest", Size: [2]int{1, 1}, At: now}}, now); err != nil {
		t.Fatal(err)
	}
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	s.path = filepath.Join(blocked, "memory.json")
	old.Entity, old.At = "current", now.Add(time.Minute)
	if err := s.RememberLandmarks("world", []Landmark{old}, old.At); err == nil {
		t.Fatal("expected failed save")
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, view := range []*Store{s, reopened} {
		if got := view.Landmarks("world"); len(got) != 1 || got[0].Entity != "old" {
			t.Fatal("failed save changed landmarks", got)
		}
		if got := view.RememberedContainers("world"); len(got) != 1 || got[0].Entity != "old" || got[0].ObservedEntity != "" {
			t.Fatal("failed save changed chest provenance", got)
		}
	}
}

func TestReconciliationPreservesDirectCurrentChestObservation(t *testing.T) {
	s := Memory()
	now := time.Now().UTC()
	old := Landmark{Entity: "old", Kind: protocol.StructureContainer, Prefab: "chest", At: now}
	if err := s.RememberLandmarks("world", []Landmark{old}, now); err != nil {
		t.Fatal(err)
	}
	// Even a newer old-ID hint cannot supersede directly observed empty contents.
	for _, c := range []RememberedContainer{
		{Entity: "old", Size: [2]int{1, 1}, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 20}}, At: now},
		{Entity: "current", Size: [2]int{1, 1}, At: now.Add(-time.Second)},
	} {
		if err := s.RememberContainers("world", []RememberedContainer{c}, now); err != nil {
			t.Fatal(err)
		}
	}
	old.Entity, old.At = "current", now.Add(time.Minute)
	if err := s.RememberLandmarks("world", []Landmark{old}, old.At); err != nil {
		t.Fatal(err)
	}
	got := s.RememberedContainers("world")
	if len(got) != 1 || got[0].Entity != "current" || got[0].ObservedEntity != "" ||
		len(got[0].Items) != 0 || !got[0].At.Equal(now.Add(-time.Second)) {
		t.Fatal("reconciliation replaced directly observed contents", got)
	}
}
