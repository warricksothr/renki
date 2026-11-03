package atlas

import (
	"context"
	"database/sql"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestMeasuredCoverageCountsNewCellsNotConfirmationsOrOtherObservers(t *testing.T) {
	s := openTest(t, Defaults())
	now := time.Now().UTC().Truncate(time.Millisecond)
	b := observation(now, -1, 0, 1, 40)
	ingest(t, s, b)
	since := now.Add(time.Second)
	fresh := observation(now.Add(2*time.Second), -1, 0, 3, 40)
	ingest(t, s, fresh)
	ingest(t, s, b)
	c, points, err := s.SurveyCoverage(context.Background(), Key(b.Scope), b.Scope.Source, protocol.Vec3{-8, 0, 8}, 12, since, now.Add(3*time.Second))
	if err != nil || c.New != 0 || c.Refreshed != len(points) || c.MeasuredM2 != len(points)*16 || c.Unknown != c.Total-c.Measured {
		t.Fatal(c, err)
	}
	other := observation(now.Add(4*time.Second), 1, 0, 5, 40)
	other.Scope.Source = "someone else"
	ingest(t, s, other)
	newCells := observation(now.Add(5*time.Second), 0, 0, 6, 40)
	ingest(t, s, newCells)
	c, _, err = s.SurveyCoverage(context.Background(), Key(b.Scope), b.Scope.Source, protocol.Vec3{}, 60, since, now.Add(6*time.Second))
	if err != nil || c.New != 15 {
		t.Fatal("shared border or other observer counted", c, err)
	}
	for _, radius := range []float64{251, math.NaN(), math.Inf(1)} {
		if _, _, err = s.SurveyCoverage(context.Background(), Key(b.Scope), b.Scope.Source, protocol.Vec3{}, radius, since, now.Add(6*time.Second)); err == nil {
			t.Fatal("invalid radius accepted", radius)
		}
	}
	if _, _, err = s.SurveyCoverage(context.Background(), Key(b.Scope), b.Scope.Source, protocol.Vec3{math.NaN(), 0, 0}, 60, since, now.Add(6*time.Second)); err == nil {
		t.Fatal("invalid center accepted")
	}
}
func TestSchemaOneCoverageMigrationPreservesFirstReceipt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.sqlite")
	s, err := Open(path, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	b := observation(now, 0, 0, 1, 40)
	ingest(t, s, b)
	s.Close()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`ALTER TABLE samples DROP COLUMN first_received; ALTER TABLE samples DROP COLUMN first_source; PRAGMA user_version=1;`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	s, err = Open(path, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var wrong int
	if err = s.db.QueryRow("SELECT count(*) FROM samples WHERE first_received!=received OR first_source!=source").Scan(&wrong); err != nil || wrong != 0 {
		t.Fatal(wrong, err)
	}
	fresh := observation(now.Add(time.Second), 0, 0, 2, 40)
	ingest(t, s, fresh)
	c, _, err := s.SurveyCoverage(context.Background(), Key(b.Scope), b.Scope.Source, protocol.Vec3{}, 60, now.Add(time.Millisecond), now.Add(2*time.Second))
	if err != nil || c.New != 0 || c.Refreshed != 25 {
		t.Fatal(c, err)
	}
}
