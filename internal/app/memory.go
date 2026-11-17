package app

import (
	"errors"
	"fmt"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/route"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
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

// RoutePlanStatus is a plan with what it was asked for.
type RoutePlanStatus struct {
	World string `json:"world"`
	// Place is the named place the target came from, if any.
	Place string `json:"place,omitempty"`
	route.Plan
}

func parseVec(text string) (protocol.Vec3, error) {
	var v protocol.Vec3
	parts := strings.Split(text, ",")
	if len(parts) != 3 {
		return v, errors.New("a position is x,y,z")
	}
	for i, part := range parts {
		n, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 1e6 {
			return v, errors.New("a position is three finite numbers: x,y,z")
		}
		v[i] = n
	}
	return v, nil
}

// unknownPlace answers a plan for a name that resolves to nothing: a reply an
// agent can read, with the names that would work, rather than an HTTP error.
func (a *App) unknownPlace(out RoutePlanStatus, from protocol.Vec3, place, why string) RoutePlanStatus {
	var names []string
	for _, w := range a.Routes.Store.Waypoints(out.World) {
		names = append(names, w.Name)
	}
	sort.Strings(names)
	if _, ok := a.Routes.Store.Home(out.World); ok {
		names = append([]string{"home"}, names...)
	}
	if len(names) > 25 {
		names = append(names[:25], fmt.Sprintf("and %d more", len(names)-25))
	}
	saved := " No places are saved in this world."
	if len(names) > 0 {
		saved = " Saved places: " + strings.Join(names, ", ") + "."
	}
	if why == "" {
		why = fmt.Sprintf("No saved place is named %q.", place)
	}
	out.Place = place
	out.Plan = route.Plan{Reason: route.ReasonUnknownPlace, Legs: []route.Leg{}, Start: route.Anchor{Pos: from},
		Explain: why + saved}
	return out
}

// RoutePlan answers GET /api/routes/plan. The target is `to=x,y,z` or
// `place=NAME`: "home", or a saved place by name. `from=x,y,z` replaces the
// body's position, so a plan can be asked for somewhere the body is not. It
// sends no game task.
func (a *App) RoutePlan(q url.Values) (any, error) {
	state := a.Session.State()
	if state == nil || state.Phase != protocol.PhaseInWorld {
		return nil, errors.New("the bot is not in a world")
	}
	out := RoutePlanStatus{World: state.World}
	from := state.Pos
	if text := q.Get("from"); text != "" {
		var err error
		if from, err = parseVec(text); err != nil {
			return nil, err
		}
	}
	var to protocol.Vec3
	switch place, text := strings.TrimSpace(q.Get("place")), q.Get("to"); {
	case place != "" && text != "":
		return nil, errors.New("give to or place, not both")
	case text != "":
		var err error
		if to, err = parseVec(text); err != nil {
			return nil, err
		}
	case strings.EqualFold(place, "home"):
		h, ok := a.Routes.Store.Home(state.World)
		if !ok {
			return a.unknownPlace(out, from, place, "No home is saved in this world."), nil
		}
		to, out.Place = h.Pos, "home"
	case place != "":
		found := false
		for _, w := range a.Routes.Store.Waypoints(state.World) {
			if strings.EqualFold(w.Name, place) {
				to, out.Place, found = w.Pos, w.Name, true
				break
			}
		}
		if !found {
			return a.unknownPlace(out, from, place, ""), nil
		}
	default:
		return nil, errors.New("give a target: to=x,y,z or place=NAME")
	}
	out.Plan = route.Find(a.Routes.Store.Routes(state.World), from, to)
	return out, nil
}
