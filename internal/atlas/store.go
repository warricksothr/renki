// Package atlas retains measured terrain in a bounded, world-scoped SQLite atlas.
// It is visual/survey evidence, never a claim about navigability.
package atlas

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
	_ "modernc.org/sqlite"
)

type Policy struct {
	StaleDays   int   `json:"staleDays"`
	KeepDays    int   `json:"keepDays"`
	HistoryDays int   `json:"historyDays"`
	Revisions   int   `json:"revisions"`
	MaxSamples  int   `json:"maxSamples"`
	MaxHistory  int   `json:"maxHistory"`
	MaxBytes    int64 `json:"maxBytes"`
}

func Defaults() Policy { return Policy{30, 180, 30, 8, 250000, 100000, 128 << 20} }
func (p Policy) Validate() error {
	if p.StaleDays < 1 || p.KeepDays < p.StaleDays || p.KeepDays > 3650 || p.HistoryDays < 1 || p.HistoryDays > p.KeepDays || p.Revisions < 1 || p.Revisions > 64 || p.MaxSamples < 25 || p.MaxSamples > 4000000 || p.MaxHistory < 1 || p.MaxHistory > 4000000 || p.MaxBytes < 8<<20 || p.MaxBytes > 4<<30 {
		return errors.New("invalid atlas retention/quota policy")
	}
	return nil
}

type Store struct {
	db     *sql.DB
	policy Policy
	mu     sync.Mutex
}

func Key(s protocol.TerrainScope) string {
	b, _ := json.Marshal([]string{s.Namespace, s.WorldUID, s.Epoch})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func Open(path string, p Policy) (*Store, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if path != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return nil, err
		}
	} else {
		path = ":memory:"
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, policy: p}
	fail := func(err error) (*Store, error) { db.Close(); return nil, err }
	for _, q := range []string{"PRAGMA busy_timeout=3000", "PRAGMA foreign_keys=ON", "PRAGMA auto_vacuum=INCREMENTAL", "PRAGMA journal_mode=WAL", "PRAGMA synchronous=FULL", "PRAGMA wal_autocheckpoint=256", "PRAGMA journal_size_limit=0"} {
		if _, err = db.Exec(q); err != nil {
			return fail(err)
		}
	}
	var version, pageSize int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fail(err)
	}
	if version > 1 {
		return fail(fmt.Errorf("atlas schema %d is newer than supported schema 1", version))
	}
	if err = db.QueryRow("PRAGMA page_size").Scan(&pageSize); err != nil {
		return fail(err)
	}
	// Reserve room for the WAL, frame headers and maintenance. One connection
	// prevents readers retaining a WAL while the writer checkpoints each batch.
	pages := (p.MaxBytes/2 - (1 << 20)) / int64(pageSize)
	var actualPages int64
	if err = db.QueryRow(fmt.Sprintf("PRAGMA max_page_count=%d", pages)).Scan(&actualPages); err != nil {
		return fail(err)
	}
	if actualPages > pages {
		return fail(errors.New("existing atlas exceeds configured disk budget"))
	}
	if version == 0 {
		tx, err := db.Begin()
		if err != nil {
			return fail(err)
		}
		_, err = tx.Exec(schema)
		if err == nil {
			err = tx.Commit()
		} else {
			tx.Rollback()
		}
		if err != nil {
			return fail(err)
		}
	}
	if err = s.Maintain(context.Background(), time.Now()); err != nil {
		return fail(err)
	}
	return s, nil
}
func (s *Store) Close() error { return s.db.Close() }

const schema = `
CREATE TABLE worlds(id TEXT PRIMARY KEY, name TEXT NOT NULL, scope TEXT NOT NULL, seen INTEGER NOT NULL, cx REAL NOT NULL, cz REAL NOT NULL);
CREATE TABLE sources(world TEXT NOT NULL REFERENCES worlds(id) ON DELETE CASCADE,source TEXT NOT NULL,stream TEXT NOT NULL,manifest TEXT NOT NULL,PRIMARY KEY(world,source,stream)) WITHOUT ROWID;
CREATE TABLE samples(world TEXT NOT NULL REFERENCES worlds(id) ON DELETE CASCADE, x INTEGER NOT NULL,z INTEGER NOT NULL,h REAL NOT NULL,biome TEXT NOT NULL,water REAL NOT NULL,hash TEXT NOT NULL,observed INTEGER NOT NULL,received INTEGER NOT NULL,source TEXT NOT NULL,stream TEXT NOT NULL,tick REAL NOT NULL,PRIMARY KEY(world,x,z)) WITHOUT ROWID;
CREATE INDEX samples_age ON samples(observed);
CREATE INDEX samples_source ON samples(world,source,stream);
CREATE TABLE history(id INTEGER PRIMARY KEY,world TEXT NOT NULL REFERENCES worlds(id) ON DELETE CASCADE,x INTEGER NOT NULL,z INTEGER NOT NULL,h REAL NOT NULL,biome TEXT NOT NULL,water REAL NOT NULL,hash TEXT NOT NULL,observed INTEGER NOT NULL,received INTEGER NOT NULL,source TEXT NOT NULL,stream TEXT NOT NULL,tick REAL NOT NULL,replaced INTEGER NOT NULL);
CREATE INDEX history_cell ON history(world,x,z,id DESC);
CREATE INDEX history_age ON history(replaced);
CREATE INDEX history_source ON history(world,source,stream);
PRAGMA user_version=1;`

func content(h float64, b string, w float64) string {
	v, _ := json.Marshal([]any{h, b, w})
	sum := sha256.Sum256(v)
	return hex.EncodeToString(sum[:])
}

// Ingest accepts only validated local bridge measurements. Original producer
// ticks make cached resends idempotent even after reopening the database.
func (s *Store) Ingest(ctx context.Context, b session.TerrainObservation) error {
	m := protocol.Terrain{Scope: &b.Scope, CapturedAt: b.CapturedAt, Tiles: b.Tiles}
	if b.World == "" || len(b.World) > 128 || len(b.Tiles) > 128 || !m.ValidMeasurements() || b.ReceivedAt.IsZero() {
		return errors.New("invalid atlas measurement batch")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	key := Key(b.Scope)
	manifest, _ := json.Marshal(b.Scope)
	cx, cz := 0.0, 0.0
	if len(b.Tiles) > 0 {
		cx = float64(b.Tiles[0].X*16 + 8)
		cz = float64(b.Tiles[0].Z*16 + 8)
	}
	var previousManifest string
	err = tx.QueryRowContext(ctx, "SELECT scope FROM worlds WHERE id=?", key).Scan(&previousManifest)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		var old protocol.TerrainScope
		if err = json.Unmarshal([]byte(previousManifest), &old); err != nil {
			return err
		}
		if old.Generator != b.Scope.Generator {
			return errors.New("world generator changed; select a new atlas epoch")
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO worlds VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,scope=excluded.scope,seen=MAX(seen,excluded.seen),cx=CASE WHEN excluded.seen>=seen THEN excluded.cx ELSE cx END,cz=CASE WHEN excluded.seen>=seen THEN excluded.cz ELSE cz END`, key, b.World, string(manifest), b.ReceivedAt.UnixMilli(), cx, cz)
	if err != nil {
		return err
	}
	// The manifest for every producer stream is retained with its evidence.
	if _, err = tx.ExecContext(ctx, "INSERT INTO sources VALUES(?,?,?,?) ON CONFLICT(world,source,stream) DO NOTHING", key, b.Scope.Source, b.Scope.Stream, string(manifest)); err != nil {
		return err
	}
	for _, tile := range b.Tiles {
		for i, tick := range tile.MeasuredAt {
			if tick == nil {
				continue
			}
			age := b.CapturedAt - *tick
			if age > float64(s.policy.KeepDays)*86400 {
				continue
			}
			observed := b.ReceivedAt.Add(-time.Duration(age * float64(time.Second))).UnixMilli()
			x, z := tile.X*4+i%5, tile.Z*4+i/5
			hash := content(*tile.Heights[i], *tile.Biomes[i], tile.WaterLevel)
			var oldHash, source, stream string
			var oldTick float64
			var oldAt int64
			err = tx.QueryRowContext(ctx, "SELECT hash,source,stream,tick,observed FROM samples WHERE world=? AND x=? AND z=?", key, x, z).Scan(&oldHash, &source, &stream, &oldTick, &oldAt)
			exists := err == nil
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			if exists && ((source == b.Scope.Source && stream == b.Scope.Stream && *tick <= oldTick) || observed < oldAt) {
				continue
			}
			if exists && oldHash != hash {
				if _, err = tx.ExecContext(ctx, `INSERT INTO history(world,x,z,h,biome,water,hash,observed,received,source,stream,tick,replaced) SELECT world,x,z,h,biome,water,hash,observed,received,source,stream,tick,? FROM samples WHERE world=? AND x=? AND z=?`, b.ReceivedAt.UnixMilli(), key, x, z); err != nil {
					return err
				}
				if _, err = tx.ExecContext(ctx, `DELETE FROM history WHERE world=? AND x=? AND z=? AND id NOT IN (SELECT id FROM history WHERE world=? AND x=? AND z=? ORDER BY id DESC LIMIT ?)`, key, x, z, key, x, z, s.policy.Revisions); err != nil {
					return err
				}
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO samples VALUES(?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(world,x,z) DO UPDATE SET h=excluded.h,biome=excluded.biome,water=excluded.water,hash=excluded.hash,observed=excluded.observed,received=excluded.received,source=excluded.source,stream=excluded.stream,tick=excluded.tick`, key, x, z, *tile.Heights[i], *tile.Biomes[i], tile.WaterLevel, hash, observed, b.ReceivedAt.UnixMilli(), b.Scope.Source, b.Scope.Stream, *tick)
			if err != nil {
				return err
			}
		}
	}
	if err = s.prune(ctx, tx, b.ReceivedAt); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")
	return err
}

func (s *Store) prune(ctx context.Context, tx *sql.Tx, now time.Time) error {
	for _, q := range []struct {
		query  string
		cutoff int64
	}{
		{"DELETE FROM samples WHERE observed<?", now.AddDate(0, 0, -s.policy.KeepDays).UnixMilli()},
		{"DELETE FROM history WHERE replaced<?", now.AddDate(0, 0, -s.policy.HistoryDays).UnixMilli()},
	} {
		if _, err := tx.ExecContext(ctx, q.query, q.cutoff); err != nil {
			return err
		}
	}
	// Only order/evict when over budget; unchanged confirmations must not sort
	// the whole atlas on every batch.
	for _, q := range []struct {
		table  string
		limit  int
		delete string
	}{
		{"samples", s.policy.MaxSamples, `DELETE FROM samples WHERE (world,x,z) IN (SELECT world,x,z FROM samples ORDER BY observed,world,x,z LIMIT ?)`},
		{"history", s.policy.MaxHistory, `DELETE FROM history WHERE id IN (SELECT id FROM history ORDER BY id LIMIT ?)`},
		{"worlds", 256, `DELETE FROM worlds WHERE id IN (SELECT id FROM worlds ORDER BY seen,id LIMIT ?)`},
	} {
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+q.table).Scan(&count); err != nil {
			return err
		}
		if count > q.limit {
			if _, err := tx.ExecContext(ctx, q.delete, count-q.limit); err != nil {
				return err
			}
		}
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM sources WHERE NOT EXISTS(SELECT 1 FROM samples WHERE samples.world=sources.world AND samples.source=sources.source AND samples.stream=sources.stream) AND NOT EXISTS(SELECT 1 FROM history WHERE history.world=sources.world AND history.source=sources.source AND history.stream=sources.stream)`)
	return err
}

func (s *Store) Maintain(ctx context.Context, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = s.prune(ctx, tx, now); err != nil {
		return err
	}
	// Reclaim before SQLite's page ceiling is reached, including history space.
	var pages, free, limit int
	for q, out := range map[string]*int{"PRAGMA page_count": &pages, "PRAGMA freelist_count": &free, "PRAGMA max_page_count": &limit} {
		if err = tx.QueryRowContext(ctx, q).Scan(out); err != nil {
			return err
		}
	}
	if pages-free > limit*8/10 {
		if _, err = tx.ExecContext(ctx, `DELETE FROM history WHERE id IN (SELECT id FROM history ORDER BY id LIMIT (SELECT MAX(1,COUNT(*)/4) FROM history))`); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM samples WHERE (world,x,z) IN (SELECT world,x,z FROM samples ORDER BY observed LIMIT (SELECT MAX(1,COUNT(*)/10) FROM samples))`); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	for _, q := range []string{"PRAGMA wal_checkpoint(TRUNCATE)", "PRAGMA incremental_vacuum(256)", "PRAGMA wal_checkpoint(TRUNCATE)"} {
		if _, err = s.db.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

type World struct {
	Key    string                `json:"key"`
	Name   string                `json:"name"`
	Scope  protocol.TerrainScope `json:"scope"`
	Seen   time.Time             `json:"seen"`
	Center [2]float64            `json:"center"`
}

func (s *Store) Worlds(ctx context.Context) ([]World, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id,name,scope,seen,cx,cz FROM worlds ORDER BY seen DESC,id LIMIT 256")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []World{}
	for rows.Next() {
		var w World
		var scope string
		var seen int64
		if err = rows.Scan(&w.Key, &w.Name, &scope, &seen, &w.Center[0], &w.Center[1]); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(scope), &w.Scope); err != nil {
			return nil, err
		}
		w.Seen = time.UnixMilli(seen).UTC()
		out = append(out, w)
	}
	return out, rows.Err()
}

type Bounds struct{ MinX, MinZ, MaxX, MaxZ float64 }

func (b Bounds) Valid() bool {
	return !math.IsNaN(b.MinX) && !math.IsNaN(b.MinZ) && !math.IsNaN(b.MaxX) && !math.IsNaN(b.MaxZ) && b.MinX >= -160000 && b.MaxX <= 160000 && b.MinZ >= -160000 && b.MaxZ <= 160000 && b.MaxX >= b.MinX && b.MaxZ >= b.MinZ && b.MaxX-b.MinX <= 512 && b.MaxZ-b.MinZ <= 512
}

type Tile struct {
	protocol.TerrainTile
	ObservedAt []*time.Time `json:"observedAt"`
	Sources    []*string    `json:"sources"`
	Stale      []bool       `json:"stale"`
}

func floorDiv(n, d int) int { return int(math.Floor(float64(n) / float64(d))) }
func (s *Store) Tiles(ctx context.Context, key string, b Bounds, now time.Time) ([]Tile, error) {
	if !b.Valid() {
		return nil, errors.New("atlas viewport must be finite and at most 512 m per side")
	}
	// Query whole edge tiles, then reconstruct their corners from canonical keys.
	minTX, maxTX := int(math.Floor(b.MinX/16)), int(math.Floor(b.MaxX/16))
	minTZ, maxTZ := int(math.Floor(b.MinZ/16)), int(math.Floor(b.MaxZ/16))
	rows, err := s.db.QueryContext(ctx, `SELECT x,z,h,biome,water,observed,source FROM samples WHERE world=? AND x BETWEEN ? AND ? AND z BETWEEN ? AND ? AND observed>=? AND observed<=? ORDER BY x,z LIMIT 20000`, key, minTX*4, (maxTX+1)*4, minTZ*4, (maxTZ+1)*4, now.AddDate(0, 0, -s.policy.KeepDays).UnixMilli(), now.UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tiles := map[[2]int]*Tile{}
	for rows.Next() {
		var x, z int
		var h, w float64
		var biome, source string
		var observed int64
		if err = rows.Scan(&x, &z, &h, &biome, &w, &observed, &source); err != nil {
			return nil, err
		}
		tx, tz := floorDiv(x, 4), floorDiv(z, 4)
		xs, zs := []int{tx}, []int{tz}
		if x%4 == 0 {
			xs = append(xs, tx-1)
		}
		if z%4 == 0 {
			zs = append(zs, tz-1)
		}
		for _, xx := range xs {
			for _, zz := range zs {
				if xx < minTX || xx > maxTX || zz < minTZ || zz > maxTZ {
					continue
				}
				k := [2]int{xx, zz}
				t := tiles[k]
				if t == nil {
					t = &Tile{TerrainTile: protocol.TerrainTile{X: xx, Z: zz, WaterLevel: w, Heights: make([]*float64, 25), Biomes: make([]*string, 25)}, ObservedAt: make([]*time.Time, 25), Sources: make([]*string, 25), Stale: make([]bool, 25)}
					tiles[k] = t
				}
				i := (z-zz*4)*5 + x - xx*4
				at := time.UnixMilli(observed).UTC()
				t.Heights[i] = &h
				t.Biomes[i] = &biome
				t.ObservedAt[i] = &at
				t.Sources[i] = &source
				t.Stale[i] = now.Sub(at) > time.Duration(s.policy.StaleDays)*24*time.Hour
			}
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	out := make([]Tile, 0, len(tiles))
	for x := minTX; x <= maxTX; x++ {
		for z := minTZ; z <= maxTZ; z++ {
			if t := tiles[[2]int{x, z}]; t != nil {
				out = append(out, *t)
			}
		}
	}
	return out, nil
}
