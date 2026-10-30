package home

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/warricksothr/renki/internal/protocol"
)

const MaxWaypoints = 128

var WaypointKinds = []string{"place", "storage", "cooking", "wood", "stone", "flint", "berries", "ore"}
var ErrWaypointConflict = errors.New("the waypoint changed; reload it before editing")
var ErrNoWaypoint = errors.New("no waypoint with that name in this world")

// Waypoint is an owner-editable named point, or a circular zone when Radius > 0.
type Waypoint struct {
	Name     string        `json:"name"`
	Kind     string        `json:"kind"`
	Pos      protocol.Vec3 `json:"pos"`
	Radius   float64       `json:"radius"`
	Source   string        `json:"source"` // owner or bot
	SetBy    string        `json:"setBy"`
	At       time.Time     `json:"at"`
	Evidence string        `json:"evidence,omitempty"`
}
type waypointWorld struct {
	Places     map[string]Waypoint `json:"places"`
	Suppressed []string            `json:"suppressed,omitempty"`
}

// WaypointName uses one spelling in chat, settings, the map and movement.
func WaypointName(name string) (string, error) {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case unicode.IsSpace(r) || r == '-' || r == '_':
			b.WriteByte(' ')
		default:
			return "", errors.New("waypoint names use letters, numbers and spaces")
		}
	}
	name = strings.Join(strings.Fields(b.String()), " ")
	for strings.HasPrefix(name, "the ") {
		name = strings.TrimPrefix(name, "the ")
	}
	if name == "" || len(name) > 64 || name == "home" {
		return "", errors.New("choose a waypoint name of 1–64 characters other than home")
	}
	return name, nil
}
func validateWaypoint(world string, w Waypoint) error {
	if strings.TrimSpace(world) == "" || len(world) > 128 {
		return errors.New("a world is required")
	}
	if !slices.Contains(WaypointKinds, w.Kind) {
		return errors.New("unknown waypoint kind")
	}
	if math.IsNaN(w.Radius) || math.IsInf(w.Radius, 0) || w.Radius < 0 || w.Radius > 100 {
		return errors.New("zone radius must be 0–100 m")
	}
	for _, v := range w.Pos {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return errors.New("waypoint coordinates must be finite")
		}
	}
	if w.At.IsZero() {
		return errors.New("waypoint needs an update time")
	}
	return nil
}
func (s *Store) Waypoints(world string) []Waypoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Waypoint, 0, len(s.mem.Waypoints[world].Places))
	for _, w := range s.mem.Waypoints[world].Places {
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func (s *Store) Waypoint(world, name string) (Waypoint, bool) {
	name, err := WaypointName(name)
	if err != nil {
		return Waypoint{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.mem.Waypoints[world].Places[name]
	return w, ok
}

// editWaypoints commits the whole world or restores the old in-memory state.
func (s *Store) editWaypoints(world string, edit func(*waypointWorld) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.mem.Waypoints
	next := old[world]
	next.Places = maps.Clone(next.Places)
	next.Suppressed = slices.Clone(next.Suppressed)
	if next.Places == nil {
		next.Places = map[string]Waypoint{}
	}
	if err := edit(&next); err != nil {
		return err
	}
	worlds := maps.Clone(old)
	if worlds == nil {
		worlds = map[string]waypointWorld{}
	}
	worlds[world] = next
	s.mem.Waypoints = worlds
	if err := s.save(); err != nil {
		s.mem.Waypoints = old
		return err
	}
	return nil
}

// EditWaypoint creates (oldName empty), updates/renames, or deletes (next nil).
// An owner change takes precedence over future automatic discovery.
func (s *Store) EditWaypoint(world, oldName string, next *Waypoint, expected *time.Time) error {
	if next == nil && oldName == "" {
		return ErrNoWaypoint
	}
	var err error
	if oldName != "" {
		oldName, err = WaypointName(oldName)
		if err != nil {
			return err
		}
	}
	var w Waypoint
	if next != nil {
		w = *next
		w.Name, err = WaypointName(w.Name)
		if err != nil {
			return err
		}
		if w.Kind == "" {
			w.Kind = "place"
		}
		w.Source = "owner"
		w.Evidence = ""
		if err := validateWaypoint(world, w); err != nil {
			return err
		}
	} else if strings.TrimSpace(world) == "" {
		return errors.New("a world is required")
	}
	return s.editWaypoints(world, func(current *waypointWorld) error {
		old, exists := current.Places[oldName]
		if oldName != "" && !exists {
			return ErrNoWaypoint
		}
		if expected != nil && (!exists || !old.At.Equal(*expected)) {
			return ErrWaypointConflict
		}
		if next != nil && w.Name != oldName {
			if _, exists := current.Places[w.Name]; exists {
				return fmt.Errorf("a waypoint named %q already exists", w.Name)
			}
		}
		if next != nil && !exists && len(current.Places) >= MaxWaypoints {
			return errors.New("too many waypoints in this world")
		}
		if oldName != "" {
			delete(current.Places, oldName)
		}
		if next == nil {
			if old.Kind != "place" && !slices.Contains(current.Suppressed, old.Kind) {
				current.Suppressed = append(current.Suppressed, old.Kind)
			}
		} else {
			current.Places[w.Name] = w
			// Changing an automatic role must not immediately recreate its old role.
			if exists && old.Kind != w.Kind && old.Kind != "place" && !slices.Contains(current.Suppressed, old.Kind) {
				current.Suppressed = append(current.Suppressed, old.Kind)
			}
		}
		return nil
	})
}

// DiscoverWaypoints adds evidenced standard places only when that role is
// missing and has not been deleted. It never moves or overwrites an edit.
func (s *Store) DiscoverWaypoints(world string, candidates []Waypoint) error {
	// Avoid a disk write for every perception when nothing can be added.
	s.mu.Lock()
	current := s.mem.Waypoints[world]
	needed := false
	for _, w := range candidates {
		occupied := slices.Contains(current.Suppressed, w.Kind)
		for _, old := range current.Places {
			if old.Kind == w.Kind || old.Name == w.Name {
				occupied = true
			}
		}
		if !occupied && len(current.Places) < MaxWaypoints {
			needed = true
			break
		}
	}
	s.mu.Unlock()
	if !needed {
		return nil
	}
	return s.editWaypoints(world, func(current *waypointWorld) error {
		for _, w := range candidates {
			name, err := WaypointName(w.Name)
			if err != nil {
				return err
			}
			w.Name = name
			if err := validateWaypoint(world, w); err != nil {
				return err
			}
			if len(current.Places) >= MaxWaypoints {
				break
			}
			occupied := slices.Contains(current.Suppressed, w.Kind)
			for _, old := range current.Places {
				if old.Kind == w.Kind || old.Name == w.Name {
					occupied = true
				}
			}
			if occupied {
				continue
			}
			w.Source, w.SetBy = "bot", "bot"
			current.Places[w.Name] = w
		}
		return nil
	})
}
