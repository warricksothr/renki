package session

import (
	"sort"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/protocol"
)

const MaxTerrainTiles = 256
const MaxTerrainPending = 2048

type TerrainObservation struct {
	World      string
	Scope      protocol.TerrainScope
	CapturedAt float64
	ReceivedAt time.Time
	Tiles      []protocol.TerrainTile
}

// DrainTerrain transfers bounded, already validated batches to the disk worker.
// Pending observations keep their accepted scope even across reconnections.
func (s *Session) DrainTerrain() ([]TerrainObservation, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out, dropped := s.terrainPending, s.terrainDropped
	s.terrainPending, s.terrainDropped = nil, 0
	return out, dropped
}

func (s *Session) MapObjects(world string, connection uint64) (*protocol.Perception, []protocol.Wall) {
	s.mu.Lock()
	defer s.mu.Unlock()
	walls := []protocol.Wall{}
	if s.conn == nil || s.conn.ID() != connection || s.state == nil || s.state.World != world || s.state.Phase != protocol.PhaseInWorld {
		return nil, walls
	}
	if s.layout != nil {
		walls = s.layout.Walls
	}
	if s.seenWorld != world {
		return nil, walls
	}
	return s.seen, walls
}

func (s *Session) ConnectionID() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return 0
	}
	return s.conn.ID()
}

func (s *Session) rememberTerrain(c *bridge.Conn, m *protocol.Terrain) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != c || s.state == nil || s.state.Phase != protocol.PhaseInWorld || m.World == "" || m.World != s.state.World || m.Seq <= s.terrainSeq || (len(m.Tiles) == 0 || len(m.Tiles) > 128) {
		return
	}
	for _, tile := range m.Tiles {
		if !tile.Valid() {
			return
		}
	}
	if m.Scope != nil {
		if !m.ValidMeasurements() || m.Scope.WorldUID != s.state.WorldUID {
			return
		}
		if s.terrainScope != nil && *s.terrainScope != *m.Scope {
			s.terrain = nil
		}
		scope := *m.Scope
		s.terrainScope = &scope
		count := len(m.Tiles)
		for _, batch := range s.terrainPending {
			count += len(batch.Tiles)
		}
		if count <= MaxTerrainPending {
			s.terrainPending = append(s.terrainPending, TerrainObservation{World: m.World, Scope: scope,
				CapturedAt: m.CapturedAt, ReceivedAt: time.Now().UTC(), Tiles: m.Tiles})
		} else {
			s.terrainDropped += uint64(len(m.Tiles))
		}
	} else if s.terrainScope != nil {
		// A legacy payload cannot silently downgrade an identified stream.
		return
	}
	s.terrainSeq = m.Seq
	if s.terrain == nil {
		s.terrain = map[[2]int]protocol.TerrainTile{}
	}
	for _, tile := range m.Tiles {
		s.terrain[[2]int{tile.X, tile.Z}] = tile
	}
	if len(s.terrain) > MaxTerrainTiles {
		keys := make([][2]int, 0, len(s.terrain))
		for k := range s.terrain {
			keys = append(keys, k)
		}
		pos := s.state.Pos
		distance := func(k [2]int) float64 {
			dx, dz := float64(k[0]*16+8)-pos[0], float64(k[1]*16+8)-pos[2]
			return dx*dx + dz*dz
		}
		sort.Slice(keys, func(i, j int) bool {
			if a, b := distance(keys[i]), distance(keys[j]); a != b {
				return a < b
			}
			if keys[i][0] != keys[j][0] {
				return keys[i][0] < keys[j][0]
			}
			return keys[i][1] < keys[j][1]
		})
		for _, k := range keys[MaxTerrainTiles:] {
			delete(s.terrain, k)
		}
	}
}

// TerrainSnapshot is atomic and scoped to a live connection/world. Tiles and
// their slices are immutable, as are the session's other protocol snapshots.
func (s *Session) TerrainSnapshot() (string, uint64, []protocol.TerrainTile) {
	world, connection, _, tiles := s.TerrainView()
	return world, connection, tiles
}

func (s *Session) TerrainView() (string, uint64, *protocol.TerrainScope, []protocol.TerrainTile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []protocol.TerrainTile{}
	if s.conn == nil || s.state == nil || s.state.Phase != protocol.PhaseInWorld || s.state.World == "" {
		return "", 0, nil, out
	}
	for _, tile := range s.terrain {
		out = append(out, tile)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].X != out[j].X {
			return out[i].X < out[j].X
		}
		return out[i].Z < out[j].Z
	})
	var scope *protocol.TerrainScope
	if s.terrainScope != nil {
		value := *s.terrainScope
		scope = &value
	}
	return s.state.World, s.conn.ID(), scope, out
}
