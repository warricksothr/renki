package app

import (
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"math"
	"time"
)

type ContainerMemoryEntry struct {
	home.RememberedContainer
	AgeS float64 `json:"ageS"`
}
type ContainerMemoryStatus struct {
	World      string                 `json:"world"`
	Containers []ContainerMemoryEntry `json:"containers"`
	Pickups    []home.Pickup          `json:"pickups"`
}

func (a *App) ContainerMemory() any {
	out := ContainerMemoryStatus{Containers: []ContainerMemoryEntry{}}
	state := a.Session.State()
	if state == nil {
		return out
	}
	out.World = state.World
	out.Pickups = a.Memory.Store.Pickups(out.World)
	now := time.Now()
	for _, c := range a.Memory.Store.RememberedContainers(out.World) {
		out.Containers = append(out.Containers, ContainerMemoryEntry{c, now.Sub(c.At).Seconds()})
	}
	return out
}

type LandmarkMemoryEntry struct {
	home.Landmark
	AgeS float64 `json:"ageS"`
}

type LandmarkMemoryStatus struct {
	World     string                `json:"world"`
	Landmarks []LandmarkMemoryEntry `json:"landmarks"`
}

func (a *App) LandmarkMemory() any {
	out := LandmarkMemoryStatus{Landmarks: []LandmarkMemoryEntry{}}
	state := a.Session.State()
	if state == nil || state.Phase != protocol.PhaseInWorld {
		return out
	}
	out.World = state.World
	now := time.Now()
	for _, landmark := range a.Landmarks.Store.Landmarks(out.World) {
		out.Landmarks = append(out.Landmarks, LandmarkMemoryEntry{landmark, now.Sub(landmark.At).Seconds()})
	}
	return out
}

type PlayerMemoryStatus struct {
	World      string                 `json:"world"`
	Encounters []home.PlayerEncounter `json:"encounters"`
}

func (a *App) PlayerMemory() any {
	out := PlayerMemoryStatus{Encounters: []home.PlayerEncounter{}}
	st := a.Session.State()
	if st == nil || st.Phase != protocol.PhaseInWorld {
		return out
	}
	out.World = st.World
	out.Encounters = a.Players.History(st.World, time.Now())
	if out.Encounters == nil {
		out.Encounters = []home.PlayerEncounter{}
	}
	return out
}

// RouteMemoryStatus is what an operator or agent reads to see where the body
// has walked and whether it stands on known ground.
type RouteMemoryStatus struct {
	World string           `json:"world"`
	Stats home.RouteStats  `json:"stats"`
	Body  *RouteBody       `json:"body,omitempty"`
	Cells []home.RouteNode `json:"cells,omitempty"`
}

// RouteBody places the body in the graph. Known means its own cell has been
// walked; Nearest is the closest walked cell and how far it is.
type RouteBody struct {
	Pos      protocol.Vec3  `json:"pos"`
	Cell     string         `json:"cell"`
	Known    bool           `json:"known"`
	Nearest  *protocol.Vec3 `json:"nearest,omitempty"`
	NearestM float64        `json:"nearestM,omitempty"`
}

func (a *App) RouteMemory(cells bool) any {
	out := RouteMemoryStatus{Stats: home.SummariseRoutes(nil)}
	state := a.Session.State()
	if state == nil || state.Phase != protocol.PhaseInWorld {
		return out
	}
	out.World = state.World
	nodes := a.Routes.Store.Routes(out.World)
	out.Stats = home.SummariseRoutes(nodes)
	body := &RouteBody{Pos: state.Pos, Cell: home.RouteCellOf(state.Pos).Key()}
	best := -1.0
	for _, n := range nodes {
		if n.Key == body.Cell {
			body.Known = true
		}
		if d := math.Sqrt(math.Pow(n.Pos[0]-state.Pos[0], 2) + math.Pow(n.Pos[1]-state.Pos[1], 2) + math.Pow(n.Pos[2]-state.Pos[2], 2)); best < 0 || d < best {
			best = d
			pos := n.Pos
			body.Nearest, body.NearestM = &pos, d
		}
	}
	out.Body = body
	if cells {
		out.Cells = nodes
	}
	return out
}
