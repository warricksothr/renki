package protocol

import "math"

const TypeTerrain = "terrain"
const TerrainTileSize = 16
const TerrainSpacing = 4

// Terrain is a patch of cached loaded-heightmap observations. Tiles use a
// fixed 5x5 corner grid in row-major +z order. Unknown samples are null.
type Terrain struct {
	Envelope
	World string        `json:"world"`
	Tiles []TerrainTile `json:"tiles"`
}

func (*Terrain) MessageType() string { return TypeTerrain }

type TerrainTile struct {
	X          int        `json:"x"`
	Z          int        `json:"z"`
	WaterLevel float64    `json:"waterLevel"`
	Heights    []*float64 `json:"heights"`
	Biomes     []*string  `json:"biomes"`
}

func (t TerrainTile) Valid() bool {
	if len(t.Heights) != 25 || len(t.Biomes) != 25 || math.IsNaN(t.WaterLevel) || math.IsInf(t.WaterLevel, 0) || t.X < -10000 || t.X > 10000 || t.Z < -10000 || t.Z > 10000 {
		return false
	}
	for i, h := range t.Heights {
		if h != nil && (math.IsNaN(*h) || math.IsInf(*h, 0) || t.Biomes[i] == nil) {
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
