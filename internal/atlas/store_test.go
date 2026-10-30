package atlas

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func observation(now time.Time, x, z int, tick, h float64) session.TerrainObservation {
	scope := protocol.TerrainScope{Namespace: "server", WorldUID: "9223372036854775807", Epoch: "1", Source: "bot", Stream: "stream-1", Game: "test", Generator: 2, Mods: "hash"}
	biome := "Meadows"
	tile := protocol.TerrainTile{X: x, Z: z, WaterLevel: 30, Heights: make([]*float64, 25), Biomes: make([]*string, 25), MeasuredAt: make([]*float64, 25)}
	for i := range tile.Heights {
		tile.Heights[i] = &h
		tile.Biomes[i] = &biome
		tile.MeasuredAt[i] = &tick
	}
	return session.TerrainObservation{World: "Same name", Scope: scope, CapturedAt: tick, ReceivedAt: now, Tiles: []protocol.TerrainTile{tile}}
}
func openTest(t *testing.T, p Policy) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "atlas.sqlite"), p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func ingest(t *testing.T, s *Store, b session.TerrainObservation) {
	t.Helper()
	if err := s.Ingest(context.Background(), b); err != nil {
		t.Fatal(err)
	}
}
func count(t *testing.T, s *Store, table string) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
func view(t *testing.T, s *Store, b session.TerrainObservation, now time.Time) []Tile {
	t.Helper()
	tiles, err := s.Tiles(context.Background(), Key(b.Scope), Bounds{-16, -16, 32, 16}, now)
	if err != nil {
		t.Fatal(err)
	}
	return tiles
}

func TestRestartMergeBordersAndCachedResends(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)
	path := filepath.Join(t.TempDir(), "atlas.sqlite")
	s, err := Open(path, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	b := observation(now, -1, 0, 10, 40)
	ingest(t, s, b)
	// Same sample timestamp received much later is a resend, not a confirmation.
	resend := b
	resend.ReceivedAt = now.Add(time.Hour)
	resend.CapturedAt += 3600
	ingest(t, s, resend)
	var at int64
	s.db.QueryRow("SELECT MIN(observed) FROM samples").Scan(&at)
	if at != now.UnixMilli() || count(t, s, "history") != 0 {
		t.Fatal("resend renewed age or added history", at)
	}
	s.Close()
	s, err = Open(path, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	worlds, err := s.Worlds(ctx)
	if err != nil || len(worlds) != 1 || worlds[0].Scope.WorldUID != "9223372036854775807" {
		t.Fatal(worlds, err)
	}
	// Adjacent transport tiles share five border samples.
	adjacent := observation(now.Add(2*time.Hour), 0, 0, 7210, 42)
	ingest(t, s, adjacent)
	if count(t, s, "samples") != 45 || count(t, s, "history") != 5 {
		t.Fatal("border duplication", count(t, s, "samples"), count(t, s, "history"))
	}
	tiles := view(t, s, b, now.Add(2*time.Hour))
	found := false
	for _, tile := range tiles {
		if tile.X == -1 && tile.Z == 0 {
			found = true
			if *tile.Heights[4] != 42 || *tile.Heights[0] != 40 {
				t.Fatal("border failed to reconcile")
			}
		}
	}
	if !found {
		t.Fatal("negative tile missing")
	}
	// An unloaded corner retains its original timestamp even while its neighbour changes.
	partial := observation(now.Add(3*time.Hour), -1, 0, 10810, 50)
	partial.Tiles[0].MeasuredAt[0] = b.Tiles[0].MeasuredAt[0]
	partial.Tiles[0].Heights[0] = b.Tiles[0].Heights[0]
	ingest(t, s, partial)
	for _, tile := range view(t, s, b, now.Add(3*time.Hour)) {
		if tile.X == -1 && tile.Z == 0 && (!tile.ObservedAt[0].Equal(now) || !tile.ObservedAt[1].Equal(partial.ReceivedAt)) {
			t.Fatal("partial update renewed unmeasured age")
		}
	}
}

func TestEqualFreshConfirmationsHistoryCapsAndExpiry(t *testing.T) {
	p := Defaults()
	p.Revisions = 2
	p.StaleDays = 1
	p.KeepDays = 3
	p.HistoryDays = 2
	s := openTest(t, p)
	now := time.Now().UTC().Truncate(time.Millisecond)
	b := observation(now, 0, 0, 1, 40)
	ingest(t, s, b)
	fresh := observation(now.Add(time.Hour), 0, 0, 3601, 40)
	ingest(t, s, fresh)
	if count(t, s, "history") != 0 {
		t.Fatal("equal confirmation duplicated history")
	}
	for i := 1; i <= 4; i++ {
		ingest(t, s, observation(now.Add(time.Duration(i+1)*time.Hour), 0, 0, float64((i+1)*3600+1), float64(40+i)))
	}
	if count(t, s, "history") != 50 {
		t.Fatal("revision cap", count(t, s, "history"))
	}
	// A stale sample remains visibly stale, then hard expiry omits it on reads
	// even before maintenance. History is never queried as current state.
	tiles := view(t, s, b, now.Add(2*24*time.Hour))
	if len(tiles) == 0 || !tiles[0].Stale[24] {
		t.Fatal("stale evidence not labelled")
	}
	if got := view(t, s, b, now.Add(4*24*time.Hour)); len(got) != 0 {
		t.Fatal("expired samples visible")
	}
	if err := s.Maintain(context.Background(), now.Add(4*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if count(t, s, "samples") != 0 || count(t, s, "history") != 0 {
		t.Fatal("expired state/history retained")
	}
}

func TestWorldEpochIsolationValidationAndQuota(t *testing.T) {
	p := Defaults()
	p.MaxSamples = 30
	p.MaxHistory = 10
	s := openTest(t, p)
	now := time.Now().UTC()
	b := observation(now, 0, 0, 1, 40)
	ingest(t, s, b)
	other := observation(now.Add(time.Second), 10, 0, 2, 50)
	other.Scope.WorldUID = "42"
	ingest(t, s, other)
	epoch := other
	epoch.Scope.Epoch = "2"
	epoch.ReceivedAt = now.Add(2 * time.Second)
	ingest(t, s, epoch)
	worlds, err := s.Worlds(context.Background())
	if err != nil || len(worlds) != 3 || Key(other.Scope) == Key(epoch.Scope) {
		t.Fatal("save identity collapsed", worlds, err)
	}
	if count(t, s, "samples") > 30 {
		t.Fatal("sample quota exceeded")
	}
	invalid := b
	invalid.Tiles = []protocol.TerrainTile{{}}
	if s.Ingest(context.Background(), invalid) == nil {
		t.Fatal("invalid legacy batch accepted")
	}
	if _, err = s.Tiles(context.Background(), Key(b.Scope), Bounds{0, 0, 513, 0}, now); err == nil {
		t.Fatal("unbounded viewport")
	}
	incompatible := b
	incompatible.Scope.Generator++
	if s.Ingest(context.Background(), incompatible) == nil {
		t.Fatal("incompatible generation merged")
	}
}

func TestTransactionRollbackSchemaRecoveryAndDiskBound(t *testing.T) {
	now := time.Now().UTC()
	path := filepath.Join(t.TempDir(), "atlas.sqlite")
	p := Defaults()
	p.MaxBytes = 8 << 20
	s, err := Open(path, p)
	if err != nil {
		t.Fatal(err)
	}
	b := observation(now, 0, 0, 1, 40)
	ingest(t, s, b)
	// Force a mid-write error. No partial samples, world metadata or history may commit.
	_, err = s.db.Exec(`CREATE TRIGGER fail_insert BEFORE INSERT ON samples WHEN NEW.x=2 BEGIN SELECT RAISE(ABORT,'injected'); END`)
	if err != nil {
		t.Fatal(err)
	}
	if s.Ingest(context.Background(), observation(now.Add(time.Second), 0, 0, 2, 99)) == nil {
		t.Fatal("injection did not fail")
	}
	var h float64
	s.db.QueryRow("SELECT h FROM samples WHERE x=0 AND z=0").Scan(&h)
	if h != 40 || count(t, s, "history") != 0 {
		t.Fatal("partial transaction committed")
	}
	s.db.Exec("DROP TRIGGER fail_insert")
	for i := 1; i < 50; i++ {
		ingest(t, s, observation(now.Add(time.Duration(i)*time.Second), i*2, 0, float64(i+1), 40))
	}
	if err = s.Maintain(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	var total int64
	for _, file := range []string{path, path + "-wal", path + "-shm"} {
		if st, err := os.Stat(file); err == nil {
			total += st.Size()
		}
	}
	if total > p.MaxBytes {
		t.Fatal("physical storage budget exceeded", total)
	}
	s.db.Exec("PRAGMA user_version=99")
	s.Close()
	if next, err := Open(path, p); err == nil {
		next.Close()
		t.Fatal("future schema opened")
	}
}

func BenchmarkAtlasIngest(b *testing.B) {
	ctx := context.Background()
	path := filepath.Join(b.TempDir(), "atlas.sqlite")
	s, err := Open(path, Defaults())
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	now := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err = s.Ingest(ctx, observation(now.Add(time.Duration(i)*time.Millisecond), i*2, 0, float64(i+1), 40)); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	var n int
	s.db.QueryRow("SELECT COUNT(*) FROM samples").Scan(&n)
	if st, err := os.Stat(path); err == nil && n > 0 {
		b.ReportMetric(float64(st.Size())/float64(n), "bytes/sample")
	}
}
