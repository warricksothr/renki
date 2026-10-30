package protocol

import (
	"math"
	"strconv"
	"strings"
	"unicode"
)

const TypeTerrain = "terrain"
const TerrainTileSize = 16
const TerrainSpacing = 4

// Terrain is a patch of cached loaded-heightmap observations. Tiles use a
// fixed 5x5 corner grid in row-major +z order. Unknown samples are null.
type Terrain struct {
	Envelope
	World      string        `json:"world"`
	Scope      *TerrainScope `json:"scope,omitempty"`
	CapturedAt float64       `json:"capturedAt,omitempty"`
	Tiles      []TerrainTile `json:"tiles"`
}

// TerrainScope identifies the save lineage and producer, never just its name.
type TerrainScope struct {
	Namespace string `json:"namespace"`
	WorldUID  string `json:"worldUid"`
	Epoch     string `json:"epoch"`
	Source    string `json:"source"`
	Stream    string `json:"stream"`
	Game      string `json:"game"`
	Generator int    `json:"generator"`
	Mods      string `json:"mods"`
}

func (s TerrainScope) Valid() bool {
	uid, err := strconv.ParseInt(s.WorldUID, 10, 64)
	if err != nil || uid == 0 || strconv.FormatInt(uid, 10) != s.WorldUID || s.Generator < 0 {
		return false
	}
	for _, v := range []string{s.Namespace, s.Epoch, s.Source, s.Stream, s.Game, s.Mods} {
		if strings.TrimSpace(v) == "" || len(v) > 128 || strings.ContainsFunc(v, unicode.IsControl) {
			return false
		}
	}
	return true
}

// ValidMeasurements deliberately excludes legacy tiles from durable ingestion.
func (m *Terrain) ValidMeasurements() bool {
	if m.Scope == nil || !m.Scope.Valid() || !finite(m.CapturedAt) || m.CapturedAt < 0 {
		return false
	}
	for _, tile := range m.Tiles {
		if !tile.Valid() || len(tile.MeasuredAt) != 25 {
			return false
		}
		for i, at := range tile.MeasuredAt {
			if at != nil && (!finite(*at) || *at < 0 || *at > m.CapturedAt || tile.Heights[i] == nil || tile.Biomes[i] == nil) {
				return false
			}
			if at == nil && (tile.Heights[i] != nil || tile.Biomes[i] != nil) {
				return false
			}
		}
	}
	return true
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func (*Terrain) MessageType() string { return TypeTerrain }

type TerrainTile struct {
	X          int        `json:"x"`
	Z          int        `json:"z"`
	WaterLevel float64    `json:"waterLevel"`
	Heights    []*float64 `json:"heights"`
	Biomes     []*string  `json:"biomes"`
	MeasuredAt []*float64 `json:"measuredAt,omitempty"`
}

func (t TerrainTile) Valid() bool {
	if len(t.Heights) != 25 || len(t.Biomes) != 25 || math.IsNaN(t.WaterLevel) || math.IsInf(t.WaterLevel, 0) || t.X < -10000 || t.X > 10000 || t.Z < -10000 || t.Z > 10000 {
		return false
	}
	for i, h := range t.Heights {
		if h != nil && (!finite(*h) || t.Biomes[i] == nil || len(*t.Biomes[i]) > 64 || *t.Biomes[i] == "") {
			return false
		}
	}
	return true
}

type Location struct {
	GroundHeight *float64 `json:"groundHeight,omitempty"`
	Support      string   `json:"support"`
	InWater      bool     `json:"inWater"`
	WaterDepth   float64  `json:"waterDepth"`
}
