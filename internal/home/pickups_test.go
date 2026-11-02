package home

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPickupOriginsPersistDeduplicatePruneAndRollback(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "memory.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	p := Pickup{ID: "1", World: "one", WorldUID: "uid", Prefab: "Coal", Count: 4, At: now}
	if err = s.RememberPickups([]Pickup{p, p}, now); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Open(path)
	if err != nil || len(reloaded.Pickups("one")) != 1 || len(reloaded.Pickups("two")) != 0 {
		t.Fatal(err)
	}
	got := s.Pickups("one")
	got[0].Count = 99
	if s.Pickups("one")[0].Count != 4 {
		t.Fatal("shared memory")
	}
	backup := dir + "-saved"
	if err = os.Rename(dir, backup); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(dir, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	p.ID = "2"
	if err = s.RememberPickups([]Pickup{p}, now); err == nil || len(s.Pickups("one")) != 1 {
		t.Fatal("failed save changed memory", err)
	}
	if err = os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(backup, dir); err != nil {
		t.Fatal(err)
	}
	p.At = now.Add(KeepPickups + time.Second)
	if err = s.RememberPickups([]Pickup{p}, p.At); err != nil || len(s.Pickups("one")) != 1 || s.Pickups("one")[0].ID != "2" {
		t.Fatal("expiry", err)
	}
}
