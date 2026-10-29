package home

import (
	"github.com/warricksothr/renki/internal/protocol"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRememberedContainersReplaceContentsAndSurviveRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	chest := RememberedContainer{Entity: "1:2", Prefab: "piece_chest_wood", Pos: protocol.Vec3{1, 2, 3}, Size: [2]int{2, 2}, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 30, StackMax: 50}, {Prefab: "Stone", Stack: 50, StackMax: 50}}, At: now}
	if err = s.RememberContainers("world", []RememberedContainer{chest}, now); err != nil {
		t.Fatal(err)
	}
	if room := s.RememberedContainers("world")[0].Room("Wood", 50); room != 120 {
		t.Fatal(room)
	}
	chest.Items[0].Stack = 999
	saved := s.RememberedContainers("world")
	if saved[0].Items[0].Stack != 30 {
		t.Fatal("input alias")
	}
	saved[0].Items[0].Stack = 111
	restored, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := restored.RememberedContainers("world"); len(got) != 1 || got[0].Items[0].Stack != 30 || !got[0].At.Equal(now) {
		t.Fatal(got)
	}
	empty := chest
	empty.Items = nil
	empty.Prefab = ""
	empty.At = now.Add(time.Second)
	if err = s.RememberContainers("world", []RememberedContainer{empty}, empty.At); err != nil {
		t.Fatal(err)
	}
	current := s.RememberedContainers("world")[0]
	if len(current.Items) != 0 || current.Prefab != "piece_chest_wood" || current.Room("Wood", 50) != 200 {
		t.Fatal(current)
	}
	if !current.Fresh(empty.At.Add(5*time.Second), 5*time.Second) || current.Fresh(empty.At.Add(6*time.Second), 5*time.Second) || current.Fresh(now, time.Minute) {
		t.Fatal("invalid age/freshness")
	}
}
func TestContainerWorldIsolationAndFailedSaveRollback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	c := RememberedContainer{Entity: "same-id", Size: [2]int{1, 1}, At: now}
	if err = s.RememberContainers("one", []RememberedContainer{c}, now); err != nil {
		t.Fatal(err)
	}
	c.Items = []protocol.InventoryItem{{Prefab: "Wood", Stack: 3}}
	if err = s.RememberContainers("two", []RememberedContainer{c}, now); err != nil {
		t.Fatal(err)
	}
	if len(s.RememberedContainers("one")[0].Items) != 0 || len(s.RememberedContainers("two")[0].Items) != 1 {
		t.Fatal("worlds mixed")
	}
	blocked := filepath.Join(t.TempDir(), "file")
	if err = os.WriteFile(blocked, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	s.path = filepath.Join(blocked, "memory.json")
	c.Items = nil
	c.At = now.Add(time.Second)
	if err = s.RememberContainers("two", []RememberedContainer{c}, now); err == nil {
		t.Fatal("save should fail")
	}
	if got := s.RememberedContainers("two")[0]; len(got.Items) != 1 || !got.At.Equal(now) {
		t.Fatal("failed save mutated memory", got)
	}
	persisted, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted.RememberedContainers("two")[0].Items) != 1 {
		t.Fatal("failed save changed file")
	}
}
func TestContainerRetentionIsBoundedAndOlderFramesCannotReplaceNewer(t *testing.T) {
	s := Memory()
	now := time.Now()
	updates := make([]RememberedContainer, MaxContainers+2)
	for i := range updates {
		updates[i] = RememberedContainer{Entity: time.Unix(int64(i), 0).String(), Size: [2]int{1, 1}, At: now.Add(time.Duration(i) * time.Second)}
	}
	if err := s.RememberContainers("world", updates, now); err != nil {
		t.Fatal(err)
	}
	got := s.RememberedContainers("world")
	if len(got) != MaxContainers {
		t.Fatal(len(got))
	}
	older := got[0]
	older.At = now.Add(-time.Second)
	older.Items = []protocol.InventoryItem{{Prefab: "Wrong", Stack: 1}}
	if err := s.RememberContainers("world", []RememberedContainer{older}, now); err != nil {
		t.Fatal(err)
	}
	if len(s.RememberedContainers("world")[0].Items) != 0 {
		t.Fatal("old observation replaced new")
	}
	if err := s.RememberContainers("world", nil, now.Add(KeepContainers+time.Hour*2)); err != nil {
		t.Fatal(err)
	}
	if len(s.RememberedContainers("world")) != 0 {
		t.Fatal("expired records retained")
	}
}
