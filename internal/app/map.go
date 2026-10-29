package app

import (
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"sort"
)

type MapSnapshot struct {
	World      string                 `json:"world"`
	Connection uint64                 `json:"connection"`
	Tiles      []protocol.TerrainTile `json:"tiles"`
	Landmarks  []home.Landmark        `json:"landmarks"`
	Perception *protocol.Perception   `json:"perception,omitempty"`
	Walls      []protocol.Wall        `json:"walls"`
}

func (a *App) MapSnapshot() any {
	world, conn, tiles := a.Session.TerrainSnapshot()
	out := MapSnapshot{World: world, Connection: conn, Tiles: tiles, Landmarks: []home.Landmark{}}
	out.Perception, out.Walls = a.Session.MapObjects(world, conn)
	if world != "" {
		out.Landmarks = a.Landmarks.Store.Landmarks(world)
		if state := a.Session.State(); state != nil && state.World == world {
			sort.Slice(out.Landmarks, func(i, j int) bool {
				return out.Landmarks[i].Pos.FlatDistance(state.Pos) < out.Landmarks[j].Pos.FlatDistance(state.Pos)
			})
		}
		if len(out.Landmarks) > 512 {
			out.Landmarks = out.Landmarks[:512]
		}
	}
	return out
}
