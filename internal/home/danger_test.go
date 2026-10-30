package home

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestDangerPersistsDecaysAndIsolatesWorldsAndFloors(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "memory.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	observations := []Harm{
		{World: "one", At: now.Add(-DangerHalfLife), Pos: protocol.Vec3{2, 0, 0}, Event: protocol.EventDamaged, Amount: 10, By: "greyling", Prefab: "Greyling"},
		{World: "one", At: now, Pos: protocol.Vec3{4, 0, 0}, Event: protocol.EventDied, By: "greyling", Prefab: "Greyling"},
		{World: "one", At: now, Pos: protocol.Vec3{2, 8, 0}, Event: protocol.EventDamaged, Amount: 10},
		{World: "other", At: now, Pos: protocol.Vec3{2, 0, 0}, Event: protocol.EventDied},
	}
	if err = s.RememberHarms(observations, now); err != nil {
		t.Fatal(err)
	}
	observations[1].By = "changed"
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	zones := s.DangerZones("one", now)
	if len(zones) != 2 || zones[0].Deaths != 1 || zones[0].Hits != 1 || zones[0].By != "greyling" || math.Abs(zones[0].Score-5.5) > 1e-9 {
		t.Fatal(zones)
	}
	if s.Harms("one")[0].By != "greyling" || len(s.DangerZones("other", now)) != 1 || len(s.DangerZones("unknown", now)) != 0 {
		t.Fatal("world/input isolation")
	}
	harms := s.Harms("one")
	harms[0].By = "changed"
	zones[0].Score = 100
	if s.Harms("one")[0].By != "greyling" || s.DangerZones("one", now)[0].Score != 5.5 {
		t.Fatal("read aliases store")
	}
	if got := s.DangerZones("one", now.Add(DangerHalfLife))[0].Score; math.Abs(got-2.75) > 1e-9 {
		t.Fatal(got)
	}
	if len(s.DangerZones("one", now.Add(KeepDanger+time.Second))) != 0 {
		t.Fatal("expired zones")
	}
	if len(s.DangerZones("one", now.Add(-KeepDanger))) != 0 {
		t.Fatal("future zones")
	}
	if DangerRisk(s.DangerZones("one", now), protocol.Vec3{4, 20, 0}) != 0 || DangerRisk(s.DangerZones("one", now), protocol.Vec3{50, 0, 0}) != 0 || DangerRisk(s.DangerZones("one", now), protocol.Vec3{4, 0, 0}) <= 0 {
		t.Fatal("risk height/radius")
	}
}

func TestDangerBoundsRejectsInvalidAndRollsBackFailedSave(t *testing.T) {
	now := time.Now().UTC()
	folder := filepath.Join(t.TempDir(), "data")
	s, err := Open(filepath.Join(folder, "memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	sample := Harm{World: "one", At: now, Event: protocol.EventDamaged, Amount: 10}
	if err = s.RememberHarms([]Harm{sample}, now); err != nil {
		t.Fatal(err)
	}
	for _, edit := range []func(*Harm){func(h *Harm) { h.Amount = math.NaN() }, func(h *Harm) { h.Amount = -1 }, func(h *Harm) { h.Amount = 0 }, func(h *Harm) { h.World = "" }, func(h *Harm) { h.Event = "respawned" }, func(h *Harm) { h.At = now.Add(time.Second) }, func(h *Harm) { h.Pos[0] = math.Inf(1) }} {
		bad := sample
		edit(&bad)
		if s.RememberHarms([]Harm{sample, bad}, now) == nil || len(s.Harms("one")) != 1 {
			t.Fatal("invalid batch mutated store", bad)
		}
	}
	if err = os.Rename(folder, folder+"-saved"); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(folder, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	if s.RememberHarms([]Harm{sample}, now) == nil || len(s.Harms("one")) != 1 {
		t.Fatal("save rollback")
	}
	if err = os.Remove(folder); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(folder+"-saved", folder); err != nil {
		t.Fatal(err)
	}
	many := []Harm{}
	for i := range MaxHarms + 20 {
		h := sample
		h.Pos[0] = float64(i) * 30
		h.At = now.Add(-time.Duration(i) * time.Second)
		many = append(many, h)
	}
	expired := sample
	expired.At = now.Add(-KeepDanger - time.Second)
	many = append(many, expired)
	if err = s.RememberHarms(many, now); err != nil {
		t.Fatal(err)
	}
	if len(s.Harms("one")) != MaxHarms || len(s.DangerZones("one", now)) != MaxDangerZones {
		t.Fatal("unbounded history/zones")
	}
}
