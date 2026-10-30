package app

import (
	"github.com/warricksothr/renki/internal/atlas"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"sort"
	"time"
)

type MapSnapshot struct {
	WorldUID   string                     `json:"worldUid,omitempty"`
	WorldKey   string                     `json:"worldKey,omitempty"`
	Danger     []home.DangerZone          `json:"danger"`
	World      string                     `json:"world"`
	Connection uint64                     `json:"connection"`
	Tiles      []protocol.TerrainTile     `json:"tiles"`
	Containers []home.RememberedContainer `json:"containers"`
	Landmarks  []home.Landmark            `json:"landmarks"`
	Waypoints  []home.Waypoint            `json:"waypoints"`
	Perception *protocol.Perception       `json:"perception,omitempty"`
	Walls      []protocol.Wall            `json:"walls"`
}

func (a *App) MapSnapshot() any {
	world, conn, scope, tiles := a.Session.TerrainView()
	out := MapSnapshot{World: world, Connection: conn, Tiles: tiles, Landmarks: []home.Landmark{}, Containers: []home.RememberedContainer{}}
	if state := a.Session.State(); state != nil && state.World == world {
		out.WorldUID = state.WorldUID
	}
	if scope != nil {
		out.WorldKey = atlas.Key(*scope)
	}
	out.Perception, out.Walls = a.Session.MapObjects(world, conn)
	if world != "" {
		out.Danger = a.Memory.Store.DangerZones(world, time.Now())
		out.Waypoints = a.Memory.Store.Waypoints(world)
		out.Landmarks = a.Landmarks.Store.Landmarks(world)
		if state := a.Session.State(); state != nil && state.World == world {
			sort.Slice(out.Landmarks, func(i, j int) bool {
				return out.Landmarks[i].Pos.FlatDistance(state.Pos) < out.Landmarks[j].Pos.FlatDistance(state.Pos)
			})
		}
		if len(out.Landmarks) > 512 {
			out.Landmarks = out.Landmarks[:512]
		}
		// Include only mapped entities, keeping the snapshot bounded by its
		// landmarks and visible structures. This is the bot's shared memory.
		mapped := make(map[string]bool, len(out.Landmarks))
		for _, landmark := range out.Landmarks {
			mapped[landmark.Entity] = true
		}
		if out.Perception != nil {
			for _, structure := range out.Perception.Structures {
				mapped[structure.ID] = true
			}
		}
		for _, chest := range a.Memory.Store.RememberedContainers(world) {
			if mapped[chest.Entity] {
				out.Containers = append(out.Containers, chest)
			}
		}
	}
	return out
}
