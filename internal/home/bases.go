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

	"github.com/warricksothr/renki/internal/protocol"
)

// Base keeps an owner's survey boundary separate from the places work uses.
// Role bindings reference waypoint names. Rename/deletion invalidates a binding
// until explicitly edited; it never silently redirects work to another place.
type Base struct {
	Name      string        `json:"name"`
	Center    protocol.Vec3 `json:"center"`
	Radius    float64       `json:"radius"`
	DurationS float64       `json:"durationS"`
	TravelM   float64       `json:"travelM"`
	Return    string        `json:"return"` // "home", or a named place
	Sleeping  string        `json:"sleeping,omitempty"`
	Storage   []string      `json:"storage,omitempty"` // owner preference order
	At        time.Time     `json:"at"`
	SetBy     string        `json:"setBy"`
}
type baseWorld struct {
	Active string          `json:"active"`
	Areas  map[string]Base `json:"areas"`
}
type BasePlaces struct {
	Base     Base       `json:"base"`
	Return   Waypoint   `json:"return"`
	Sleeping *Waypoint  `json:"sleeping,omitempty"`
	Storage  []Waypoint `json:"storage"`
}

func cloneBase(b Base) Base                  { b.Storage = slices.Clone(b.Storage); return b }
func (b Base) Contains(p protocol.Vec3) bool { return b.Center.FlatDistance(p) <= b.Radius }
func finitePosition(p protocol.Vec3) bool {
	for _, v := range p {
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 1e6 {
			return false
		}
	}
	return true
}
func (b Base) Validate() error {
	name, err := WaypointName(b.Name)
	if err != nil || name != b.Name || len(b.Name) > 40 {
		return errors.New("choose a normalized base name other than home")
	}
	if !finitePosition(b.Center) || math.IsNaN(b.Radius) || math.IsInf(b.Radius, 0) || b.Radius < 10 || b.Radius > 250 {
		return errors.New("base radius must be 10–250 m with a finite center")
	}
	if math.IsNaN(b.DurationS) || math.IsInf(b.DurationS, 0) || b.DurationS < 120 || b.DurationS > 1200 || math.IsNaN(b.TravelM) || math.IsInf(b.TravelM, 0) || b.TravelM < 50 || b.TravelM > 4000 {
		return errors.New("survey limits must be 120–1200 seconds and 50–4000 m")
	}
	if b.Return == "" || b.At.IsZero() || len(b.Storage) > 16 {
		return errors.New("base needs a return place, update time and at most 16 storage places")
	}
	seen := map[string]bool{}
	for _, n := range append(append([]string{b.Return, b.Sleeping}, b.Storage...), b.Name) {
		if n == "" || n == "home" {
			continue
		}
		normalized, e := WaypointName(n)
		if e != nil || normalized != n {
			return errors.New("base bindings need normalized waypoint names")
		}
	}
	for _, n := range b.Storage {
		if n == "" || seen[n] {
			return errors.New("storage bindings must be unique nonempty places")
		}
		seen[n] = true
	}
	return nil
}

// ResolveBase snapshots all role coordinates and versions. Home is only the
// return default: sleeping and storage are never inferred from its proximity.
func ResolveBase(b Base, places []Waypoint, h *protocol.Vec3) (BasePlaces, error) {
	out := BasePlaces{Base: cloneBase(b)}
	if err := b.Validate(); err != nil {
		return out, err
	}
	resolve := func(name string) (Waypoint, error) {
		if name == "home" && h != nil && b.Contains(*h) {
			return Waypoint{Name: "home", Kind: "place", Pos: *h}, nil
		}
		for _, w := range places {
			if w.Name == name && b.Contains(w.Pos) {
				return w, nil
			}
		}
		return Waypoint{}, fmt.Errorf("base place %q is missing or outside its boundary; update the base binding", name)
	}
	var err error
	if out.Return, err = resolve(b.Return); err != nil {
		return out, err
	}
	if b.Sleeping != "" {
		w, e := resolve(b.Sleeping)
		if e != nil {
			return out, e
		}
		out.Sleeping = &w
	}
	for _, n := range b.Storage {
		w, e := resolve(n)
		if e != nil {
			return out, e
		}
		if w.Kind != "storage" {
			return out, fmt.Errorf("%q must have storage purpose", n)
		}
		out.Storage = append(out.Storage, w)
	}
	return out, nil
}
func (s *Store) Bases(world string) ([]Base, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w := s.mem.Bases[world]
	out := []Base{}
	for _, b := range w.Areas {
		out = append(out, cloneBase(b))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, w.Active
}
func (s *Store) ActiveBase(world string) *Base {
	s.mu.Lock()
	defer s.mu.Unlock()
	w := s.mem.Bases[world]
	b, ok := w.Areas[w.Active]
	if !ok {
		return nil
	}
	b = cloneBase(b)
	return &b
}

// WorkHome resolves the selected return role, retaining the legacy home when
// no base is selected. A broken binding fails closed instead of falling back.
func (s *Store) WorkHome(world string) (Home, bool) {
	b := s.ActiveBase(world)
	if b == nil || b.Return == "home" {
		h, ok := s.Home(world)
		if b != nil && ok && !b.Contains(h.Pos) {
			return Home{}, false
		}
		return h, ok
	}
	w, ok := s.Waypoint(world, b.Return)
	if !ok || !b.Contains(w.Pos) {
		return Home{}, false
	}
	return Home{Pos: w.Pos, At: w.At, SetBy: w.SetBy}, true
}

// EditBase saves and selects a base, or selects/deletes an existing one. Every
// change is compare-and-swap and rolls back the entire memory on save failure.
func (s *Store) EditBase(world, name string, next *Base, expected *time.Time, remove bool) error {
	if strings.TrimSpace(world) == "" || len(world) > 128 {
		return errors.New("world is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.mem.Bases
	w := old[world]
	w.Areas = maps.Clone(w.Areas)
	if w.Areas == nil {
		w.Areas = map[string]Base{}
	}
	prior, exists := w.Areas[name]
	if exists && (expected == nil || !prior.At.Equal(*expected)) || !exists && expected != nil {
		return ErrWaypointConflict
	}
	if next != nil {
		b := cloneBase(*next)
		if b.Name != name {
			return errors.New("base name must match; create a separate base to rename it")
		}
		if err := b.Validate(); err != nil {
			return err
		}
		places := []Waypoint{}
		for _, p := range s.mem.Waypoints[world].Places {
			places = append(places, p)
		}
		var h *protocol.Vec3
		if at, ok := s.mem.Homes[world]; ok {
			p := at.Pos
			h = &p
		}
		if _, err := ResolveBase(b, places, h); err != nil {
			return err
		}
		if !exists && len(w.Areas) >= 16 {
			return errors.New("at most 16 bases per world")
		}
		w.Areas[name] = b
		w.Active = name
	} else {
		if !exists {
			return errors.New("unknown base")
		}
		if remove {
			delete(w.Areas, name)
			if w.Active == name {
				w.Active = ""
			}
		} else {
			w.Active = name
			b := w.Areas[name]
			b.At = time.Now().UTC()
			w.Areas[name] = b
		}
	}
	all := maps.Clone(old)
	if all == nil {
		all = map[string]baseWorld{}
	}
	all[world] = w
	s.mem.Bases = all
	if err := s.save(); err != nil {
		s.mem.Bases = old
		return err
	}
	return nil
}
