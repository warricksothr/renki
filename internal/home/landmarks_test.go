package home

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestLandmarksPersistSharedTypesAndIndependentWorlds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	tool := 2
	first := Landmark{Entity: "ore1", Kind: protocol.ResourceRock, Prefab: "silvervein", MinTool: &tool, Pos: protocol.Vec3{-1.26, 2.24, 3.76}, At: now}
	second := first
	second.Entity = "ore2"
	if err := s.RememberLandmarks("one", []Landmark{first, second}, now); err != nil {
		t.Fatal(err)
	}
	tool = 9
	if len(s.mem.Landmarks["one"].Types) != 1 {
		t.Fatal("identical prefab metadata was duplicated")
	}
	got := s.Landmarks("one")
	if got[0].Entity != "ore1" || got[0].Pos != (protocol.Vec3{-1.5, 2, 4}) || *got[0].MinTool != 2 {
		t.Fatal(got)
	}
	*got[0].MinTool = 7
	got[0].Pos[0] = 999
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Landmarks("one"); len(got) != 2 || *got[0].MinTool != 2 || !got[0].At.Equal(now) {
		t.Fatal(got)
	}
	if *s.Landmarks("one")[0].MinTool != 2 {
		t.Fatal("read alias")
	}
	first.Kind, first.Prefab, first.MinTool = protocol.StructurePortal, "portal_wood", nil
	first.Tag = "home"
	if err := s.RememberLandmarks("two", []Landmark{first}, now); err != nil {
		t.Fatal(err)
	}
	if s.Landmarks("one")[0].Kind != protocol.ResourceRock || s.Landmarks("two")[0].Tag != "home" {
		t.Fatal("worlds mixed")
	}
	// An updated position/tag replaces that entity only. Old frames cannot undo it.
	first.At, first.Pos, first.Tag = now.Add(time.Second), protocol.Vec3{9, 1, 8}, "new"
	if err := s.RememberLandmarks("two", []Landmark{first}, first.At); err != nil {
		t.Fatal(err)
	}
	first.At, first.Tag = now, "old"
	if err := s.RememberLandmarks("two", []Landmark{first}, now); err != nil {
		t.Fatal(err)
	}
	if got := s.Landmarks("two")[0]; got.Tag != "new" || got.Pos[0] != 9 {
		t.Fatal(got)
	}
}

func TestLandmarksBoundRecordsAndCatalogueAndRollBackFailures(t *testing.T) {
	s := Memory()
	now := time.Now().UTC()
	batch := make([]Landmark, MaxLandmarks+2)
	for i := range batch {
		batch[i] = Landmark{Entity: fmt.Sprint(i), Kind: protocol.StructureStation, Prefab: fmt.Sprint(i), At: now.Add(time.Duration(i) * time.Second)}
	}
	if err := s.RememberLandmarks("world", batch, now); err != nil {
		t.Fatal(err)
	}
	if len(s.Landmarks("world")) != MaxLandmarks || len(s.mem.Landmarks["world"].Types) != MaxLandmarks {
		t.Fatal("record or metadata bound failed")
	}
	for _, l := range s.Landmarks("world") {
		if l.Entity == "0" || l.Entity == "1" {
			t.Fatal("did not evict oldest")
		}
	}
	if err := s.RememberLandmarks("world", nil, now.Add(KeepLandmarks+24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if len(s.Landmarks("world")) != 0 || len(s.mem.Landmarks["world"].Types) != 0 {
		t.Fatal("expired entities or metadata remain")
	}
	path := filepath.Join(t.TempDir(), "memory.json")
	s.path = path
	o := Landmark{Entity: "portal", Kind: protocol.StructurePortal, Prefab: "portal_wood", At: now, Tag: "old"}
	if err := s.RememberLandmarks("world", []Landmark{o}, now); err != nil {
		t.Fatal(err)
	}
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	s.path = filepath.Join(blocked, "memory.json")
	o.Tag, o.Prefab = "new", "new_type"
	if err := s.RememberLandmarks("world", []Landmark{o}, now); err == nil {
		t.Fatal("expected persistence failure")
	}
	if got := s.Landmarks("world")[0]; got.Tag != "old" || got.Prefab != "portal_wood" || len(s.mem.Landmarks["world"].Types) != 1 {
		t.Fatal("failed save changed memory", got)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Landmarks("world")[0].Tag != "old" {
		t.Fatal("failed save changed disk")
	}
}

func TestLandmarksRejectInvalidBatchesAtomically(t *testing.T) {
	s := Memory()
	valid := Landmark{Entity: "chest", Kind: protocol.StructureContainer, At: time.Now()}
	for _, invalid := range []Landmark{{}, {Entity: "x", Kind: "rock", At: valid.At, Pos: protocol.Vec3{math.Inf(1), 0, 0}}, {Entity: "x", Kind: "rock", At: valid.At, Pos: protocol.Vec3{math.MaxFloat64, 0, 0}}} {
		if err := s.RememberLandmarks("world", []Landmark{valid, invalid}, valid.At); err == nil || len(s.Landmarks("world")) != 0 {
			t.Fatal("invalid batch partially accepted")
		}
	}
}
