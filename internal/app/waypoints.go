package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

type WaypointCommand struct {
	Action     string         `json:"action"`
	World      string         `json:"world"`
	Name       string         `json:"name"`
	Waypoint   *home.Waypoint `json:"waypoint,omitempty"`
	ExpectedAt *time.Time     `json:"expectedAt,omitempty"`
}
type NearWaypoint struct {
	Entity   string                    `json:"entity"`
	Kind     string                    `json:"kind"`
	Prefab   string                    `json:"prefab,omitempty"`
	Pos      protocol.Vec3             `json:"pos"`
	Source   string                    `json:"source"`
	At       time.Time                 `json:"at"`
	Contents *home.RememberedContainer `json:"contents,omitempty"`
}

func (a *App) WaypointHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		world := r.URL.Query().Get("world")
		if world == "" {
			world = a.Home.World()
		}
		if r.Method == http.MethodGet {
			out := struct {
				World     string          `json:"world"`
				Waypoints []home.Waypoint `json:"waypoints"`
				Nearby    []NearWaypoint  `json:"nearby,omitempty"`
			}{World: world, Waypoints: a.Memory.Store.Waypoints(world)}
			if name := r.URL.Query().Get("name"); name != "" {
				place, ok := a.Memory.Store.Waypoint(world, name)
				if !ok {
					http.Error(w, home.ErrNoWaypoint.Error(), http.StatusNotFound)
					return
				}
				out.Nearby = a.nearWaypoint(world, place)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(out)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		if r.Header.Get(panel.ActHeader) == "" {
			http.Error(w, "missing "+panel.ActHeader, 403)
			return
		}
		var command WaypointCommand
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
		d.DisallowUnknownFields()
		err := d.Decode(&command)
		if err == nil && !errors.Is(d.Decode(new(any)), io.EOF) {
			err = errors.New("expected one JSON value")
		}
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if command.World == "" {
			http.Error(w, "a world is required", 400)
			return
		}
		switch command.Action {
		case "set":
			if command.Waypoint == nil {
				err = errors.New("a waypoint is required")
				break
			}
			next := *command.Waypoint
			next.SetBy, next.At = "panel", time.Now().UTC()
			err = a.Memory.Store.EditWaypoint(command.World, command.Name, &next, command.ExpectedAt)
		case "delete":
			err = a.Memory.Store.EditWaypoint(command.World, command.Name, nil, command.ExpectedAt)
		case "go":
			place, ok := a.Memory.Store.Waypoint(command.World, command.Name)
			if !ok {
				err = home.ErrNoWaypoint
				break
			}
			st := a.Session.State()
			_, _, connected := a.Session.Handshake()
			if !connected || st == nil || st.World != command.World || st.Phase != protocol.PhaseInWorld {
				err = errors.New("Tas must be in this world before moving")
				break
			}
			// Dispatch resolves the latest stored coordinate and checks the world again.
			_, err = a.Session.Dispatch(r.Context(), session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Entity: place.Name}}, Origin: "panel waypoint", TimeoutS: 90})
			if err == nil {
				a.Home.Touch()
				a.Home.Ordered()
				a.Home.Told()
			}
		default:
			err = errors.New("action must be set, delete or go")
		}
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, home.ErrWaypointConflict) {
				status = 409
			}
			http.Error(w, err.Error(), status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"world": command.World, "waypoints": a.Memory.Store.Waypoints(command.World)})
	})
}

// nearby is last-known context, never a claim of complete or current stock.
func (a *App) nearWaypoint(world string, place home.Waypoint) []NearWaypoint {
	radius := place.Radius
	if radius == 0 {
		radius = 15
	}
	found := map[string]NearWaypoint{}
	containers := a.Memory.Store.RememberedContainers(world)
	for _, c := range containers {
		found[c.Entity] = NearWaypoint{Entity: c.Entity, Kind: c.Kind, Prefab: c.Prefab, Pos: c.Pos, Source: "remembered", At: c.At}
	}
	for _, l := range a.Memory.Store.Landmarks(world) {
		if old, ok := found[l.Entity]; ok && old.At.After(l.At) {
			continue
		}
		found[l.Entity] = NearWaypoint{Entity: l.Entity, Kind: l.Kind, Prefab: l.Prefab, Pos: l.Pos, Source: "remembered", At: l.At}
	}
	_, seenWorld, seen, at := a.Session.PerceptionObservation()
	if seen != nil && seenWorld == world && time.Since(at) <= 5*time.Second {
		for _, s := range seen.Structures {
			delete(found, s.ID)
			if s.Pos.FlatDistance(place.Pos) <= radius {
				found[s.ID] = NearWaypoint{Entity: s.ID, Kind: s.Kind, Prefab: s.Prefab, Pos: s.Pos, Source: "current", At: at}
			}
		}
		for _, s := range seen.Resources {
			delete(found, s.ID)
			if s.Pos.FlatDistance(place.Pos) <= radius {
				found[s.ID] = NearWaypoint{Entity: s.ID, Kind: s.Kind, Prefab: s.Prefab, Pos: s.Pos, Source: "current", At: at}
			}
		}
	}
	for _, c := range containers {
		if n, ok := found[c.Entity]; ok {
			copy := c
			n.Contents = &copy
			found[c.Entity] = n
		}
	}
	out := make([]NearWaypoint, 0, len(found))
	for _, n := range found {
		if n.Pos.FlatDistance(place.Pos) <= radius {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		di, dj := out[i].Pos.FlatDistance(place.Pos), out[j].Pos.FlatDistance(place.Pos)
		if di == dj {
			return out[i].Entity < out[j].Entity
		}
		return di < dj
	})
	if len(out) > 24 {
		out = out[:24]
	}
	return out
}

func (a *App) discoverWaypoints(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var previous *protocol.Perception
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, world, seen, at := a.Session.PerceptionObservation()
			if seen == nil || seen == previous || world == "" || time.Since(at) > 5*time.Second {
				continue
			}
			st := a.Session.State()
			if st == nil || st.World != world || st.Phase != protocol.PhaseInWorld {
				continue
			}
			candidates := standardWaypoints(seen, st.Pos, at)
			if err := a.Memory.Store.DiscoverWaypoints(world, candidates); err != nil {
				a.Landmarks.Logger.Warn("could not remember standard places", "err", err)
				continue
			}
			previous = seen
		}
	}
}

// Standard places come from actual objects, closest first, and are never
// inferred from missing perception or invented coordinates.
func standardWaypoints(seen *protocol.Perception, bot protocol.Vec3, at time.Time) []home.Waypoint {
	type candidate struct {
		kind, prefab string
		pos          protocol.Vec3
	}
	all := []candidate{}
	for _, s := range seen.Structures {
		if s.Kind == protocol.StructureContainer {
			all = append(all, candidate{"storage", s.Prefab, s.Pos})
		}
		lower := strings.ToLower(s.Prefab)
		if strings.Contains(lower, "cooking") || strings.Contains(lower, "cauldron") {
			all = append(all, candidate{"cooking", s.Prefab, s.Pos})
		}
	}
	for _, r := range seen.Resources {
		lower := strings.ToLower(r.Prefab)
		kind := ""
		switch {
		case r.Kind == protocol.ResourceTree:
			kind = "wood"
		case r.Kind == protocol.ResourceRock:
			kind = "stone"
			for _, ore := range []string{"copper", "tin", "silver", "iron", "obsidian"} {
				if strings.Contains(lower, ore) {
					kind = "ore"
				}
			}
		case r.Kind == protocol.ResourcePickable && strings.Contains(lower, "flint"):
			kind = "flint"
		case r.Kind == protocol.ResourcePickable && (strings.Contains(lower, "berry") || strings.Contains(lower, "berries")):
			kind = "berries"
		}
		if kind != "" {
			all = append(all, candidate{kind, r.Prefab, r.Pos})
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].pos.FlatDistance(bot) < all[j].pos.FlatDistance(bot) })
	chosen := map[string]bool{}
	out := []home.Waypoint{}
	for _, c := range all {
		if chosen[c.kind] {
			continue
		}
		chosen[c.kind] = true
		radius := 15.0
		if c.kind == "storage" || c.kind == "cooking" {
			radius = 6
		}
		out = append(out, home.Waypoint{Name: c.kind, Kind: c.kind, Pos: c.pos, Radius: radius, At: at, Evidence: fmt.Sprintf("Observed %s", c.prefab)})
	}
	return out
}
