package home

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestPortalLinksPersistIndependentlyAndReplaceWithNewObservations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	pos := protocol.Vec3{500, 40, -100}
	p := &protocol.PortalStatus{Connected: true, TargetID: "remote", TargetPos: &pos}
	o := Landmark{Entity: "portal", Kind: protocol.StructurePortal, Prefab: "portal_wood", Tag: "mine", At: now, Portal: p}
	if err := s.RememberLandmarks("one", []Landmark{o}, now); err != nil {
		t.Fatal(err)
	}
	// Neither the caller's observation nor expanded reads may mutate memory.
	p.TargetID, pos[0] = "tampered", 999
	got := s.Landmarks("one")[0].Portal
	if got.TargetID != "remote" || got.TargetPos[0] != 500 {
		t.Fatal(got)
	}
	got.TargetID, got.TargetPos[0] = "tampered", 999
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, store := range []*Store{s, reopened} {
		p := store.Landmarks("one")[0].Portal
		if p.TargetID != "remote" || p.TargetPos[0] != 500 {
			t.Fatal(p)
		}
		if len(store.Landmarks("two")) != 0 {
			t.Fatal("portal knowledge leaked between worlds")
		}
	}
	// A real link with no delivered destination record stays explicitly unknown.
	o.At, o.Tag, o.Portal = now.Add(time.Second), "retagged", &protocol.PortalStatus{Connected: true, TargetID: "another"}
	if err := s.RememberLandmarks("one", []Landmark{o}, o.At); err != nil {
		t.Fatal(err)
	}
	if p := s.Landmarks("one")[0].Portal; p.TargetID != "another" || p.TargetPos != nil {
		t.Fatal(p)
	}
	// Newer unlink evidence must replace old coordinates. Old frames cannot restore it.
	o.At, o.Portal = now.Add(2*time.Second), &protocol.PortalStatus{}
	if err := s.RememberLandmarks("one", []Landmark{o}, o.At); err != nil {
		t.Fatal(err)
	}
	o.At, o.Portal = now, &protocol.PortalStatus{Connected: true, TargetID: "remote"}
	if err := s.RememberLandmarks("one", []Landmark{o}, o.At); err != nil {
		t.Fatal(err)
	}
	if p := s.Landmarks("one")[0].Portal; p.Connected || p.TargetID != "" || p.TargetPos != nil {
		t.Fatal(p)
	}
	// Legacy observations and duplicate tags are hints, never inferred links.
	o.Entity, o.At, o.Portal, o.Pos = "other", now.Add(3*time.Second), nil, protocol.Vec3{10, 0, 0}
	if err := s.RememberLandmarks("one", []Landmark{o}, o.At); err != nil {
		t.Fatal(err)
	}
	if p := s.Landmarks("one")[0].Portal; p != nil {
		t.Fatal("invented portal link", p)
	}
}

func TestInvalidPortalBatchAndFailedSaveKeepPriorLink(t *testing.T) {
	s := Memory()
	now := time.Now().UTC()
	valid := Landmark{Entity: "portal", Kind: protocol.StructurePortal, At: now, Portal: &protocol.PortalStatus{Connected: true, TargetID: "remote"}}
	for _, p := range []*protocol.PortalStatus{
		{Connected: true}, {TargetID: "remote"}, {Connected: true, TargetID: "portal"},
		{TargetPos: &protocol.Vec3{1, 2, 3}},
		{Connected: true, TargetID: "remote", TargetPos: &protocol.Vec3{math.NaN(), 0, 0}},
	} {
		o := valid
		o.Portal = p
		if err := s.RememberLandmarks("one", []Landmark{valid, o}, now); err == nil || len(s.Landmarks("one")) != 0 {
			t.Fatal("invalid batch accepted", p)
		}
	}
	o := valid
	o.Kind = protocol.StructureBed
	if err := s.RememberLandmarks("one", []Landmark{o}, now); err == nil {
		t.Fatal("non-portal link accepted")
	}
	if err := s.RememberLandmarks("one", []Landmark{valid}, now); err != nil {
		t.Fatal(err)
	}
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	s.path = filepath.Join(blocked, "memory.json")
	o = valid
	o.At, o.Portal = now.Add(time.Second), &protocol.PortalStatus{}
	if err := s.RememberLandmarks("one", []Landmark{o}, o.At); err == nil {
		t.Fatal("expected save failure")
	}
	if p := s.Landmarks("one")[0].Portal; !p.Connected || p.TargetID != "remote" {
		t.Fatal("failed save changed link", p)
	}
}
