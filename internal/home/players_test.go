package home

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func encounter(at time.Time) PlayerEncounter {
	return PlayerEncounter{World: "one", Name: "Nimah", At: at, Pos: protocol.Vec3{-13, 42, 129}, Source: "heard"}
}

func TestPlayerHistoryFadesConsolidatesExpiresAndNeverRegainsPrecision(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		age     time.Duration
		grid    float64
		seconds int
	}{{time.Minute, 5, 1}, {20 * time.Minute, 25, 60}, {7 * time.Hour, 100, 900}, {3 * 24 * time.Hour, 500, 3600}} {
		original := encounter(now.Add(-c.age))
		got := CompactPlayerHistory([]PlayerEncounter{original}, now)
		if len(got) != 1 || got[0].ResolutionM != c.grid || got[0].TimeResolutionS != c.seconds {
			t.Fatal(got)
		}
		if got[0].Pos.FlatDistance(original.Pos) > c.grid/math.Sqrt2 || got[0].At.After(original.At) {
			t.Fatal("misstated uncertainty", got)
		}
		if again := CompactPlayerHistory(got, now); !reflect.DeepEqual(got, again) {
			t.Fatal("rounding drift", got, again)
		}
		if back := CompactPlayerHistory(got, got[0].At.Add(time.Second)); back[0].ResolutionM != c.grid {
			t.Fatal("precision regained", back)
		}
	}
	old := encounter(now.Add(-20 * time.Minute))
	newer := old
	newer.At = newer.At.Add(2 * time.Second)
	newer.Pos = protocol.Vec3{20, 0, 0}
	got := CompactPlayerHistory([]PlayerEncounter{old, newer}, now)
	if len(got) != 1 || got[0].Pos[0] != 12.5 {
		t.Fatal("not consolidated to latest in time bucket", got)
	}
	if len(CompactPlayerHistory([]PlayerEncounter{encounter(now.Add(-KeepPlayers - time.Hour)), encounter(now.Add(time.Minute))}, now)) != 0 {
		t.Fatal("expired/future knowledge survived")
	}
}

func TestPlayerHistoryBoundsPreserveOldEvidenceDuringContinuousSightings(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	var history []PlayerEncounter
	for i := 0; i < 1300; i++ {
		at := now.Add(time.Duration(i) * time.Second)
		history = CompactPlayerHistory(append(history, encounter(at)), at)
		if len(history) > MaxPlayerHistory {
			t.Fatal("history unbounded", len(history))
		}
	}
	if history[len(history)-1].At.After(now.Add(time.Minute)) {
		t.Fatal("busy present erased oldest evidence", history)
	}
	var batch []PlayerEncounter
	for i := 0; i < MaxPlayers+10; i++ {
		o := encounter(now)
		o.Name = fmt.Sprint("player-", i)
		batch = append(batch, o)
	}
	if len(CompactPlayerHistory(batch, now)) != MaxPlayers {
		t.Fatal("player bound")
	}
	store := Memory()
	for i := 0; i < MaxPlayerWorlds+2; i++ {
		o := encounter(now.Add(time.Duration(i) * time.Second))
		o.World = fmt.Sprint("world-", i)
		if e := store.RememberPlayers([]PlayerEncounter{o}, o.At); e != nil {
			t.Fatal(e)
		}
	}
	if len(store.mem.Players) != MaxPlayerWorlds || len(store.PlayerHistory("world-0", now.Add(time.Minute))) != 0 {
		t.Fatal("world bound", len(store.mem.Players))
	}
}

func TestPlayerPersistenceWorldIsolationAndFailedSaveRollback(t *testing.T) {
	now := time.Now().UTC()
	folder := filepath.Join(t.TempDir(), "data")
	path := filepath.Join(folder, "memory.json")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	o := encounter(now)
	if e = s.RememberPlayers([]PlayerEncounter{o}, now); e != nil {
		t.Fatal(e)
	}
	reopened, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	if len(reopened.PlayerHistory("one", now)) != 1 || len(reopened.PlayerHistory("two", now)) != 0 {
		t.Fatal("lost scope")
	}
	copy := reopened.PlayerHistory("one", now)
	copy[0].Name = "mutated"
	if reopened.PlayerHistory("one", now)[0].Name != "Nimah" {
		t.Fatal("read aliases store")
	}
	if e = os.Remove(path); e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(folder); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(folder, []byte("blocked"), 0600); e != nil {
		t.Fatal(e)
	}
	updated := o
	updated.Name = "Sothr"
	if e = s.RememberPlayers([]PlayerEncounter{updated}, now); e == nil || len(s.PlayerHistory("one", now)) != 1 {
		t.Fatal("failed save mutated memory", e)
	}
	if e = os.Remove(folder); e != nil {
		t.Fatal(e)
	}
	if e = s.RememberPlayers(nil, now.Add(7*time.Hour)); e != nil {
		t.Fatal(e)
	}
	reopened, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	if got := reopened.PlayerHistory("one", now.Add(7*time.Hour)); got[0].ResolutionM != 100 {
		t.Fatal("quiet sweep did not persist fading", got)
	}
	invalid := o
	invalid.Pos[0] = math.NaN()
	if s.RememberPlayers([]PlayerEncounter{invalid}, now) == nil {
		t.Fatal("accepted NaN")
	}
}
