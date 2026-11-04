package home

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

type PlaceChanges struct {
	Added     []string `json:"added"`
	Updated   []string `json:"updated"`
	Protected []string `json:"protected"`
	Skipped   int      `json:"skipped"`
	Unchanged int      `json:"unchanged"`
}

// ObservedPlaces describes actual objects only. Grouping into named places is
// separate from the object/chest memories; it makes no claim about stock.
func ObservedPlaces(seen *protocol.Perception, at time.Time) []Waypoint {
	out := []Waypoint{}
	if seen == nil {
		return out
	}
	add := func(kind, prefab, id string, p protocol.Vec3) {
		r := 15.0
		if kind == "storage" || kind == "cooking" || kind == "rest" {
			r = 6
		}
		out = append(out, Waypoint{Kind: kind, Pos: p, Radius: r, At: at, Entity: id, Evidence: "Observed " + prefab})
	}
	for _, s := range seen.Structures {
		if s.Kind == protocol.StructureContainer {
			add("storage", s.Prefab, s.ID, s.Pos)
		}
		if s.Kind == protocol.StructureBed {
			add("rest", s.Prefab, s.ID, s.Pos)
		}
		lower := strings.ToLower(s.Prefab)
		if strings.Contains(lower, "cooking") || strings.Contains(lower, "cauldron") {
			add("cooking", s.Prefab, s.ID, s.Pos)
		}
	}
	for _, r := range seen.Resources {
		kind := ResourcePlaceKind(r)
		if kind != "" {
			add(kind, r.Prefab, r.ID, r.Pos)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Entity != b.Entity {
			return a.Entity < b.Entity
		}
		if a.Pos[0] != b.Pos[0] {
			return a.Pos[0] < b.Pos[0]
		}
		return a.Pos[2] < b.Pos[2]
	})
	return out
}

// RefreshBasePlaces is explicit learning, not background discovery. It checks
// base revision and current owner protection while holding the store lock.
func (s *Store) RefreshBasePlaces(world string, b Base, candidates []Waypoint) (PlaceChanges, error) {
	changes := PlaceChanges{}
	if err := b.Validate(); err != nil {
		return changes, err
	}
	if len(candidates) > 4096 {
		return changes, fmt.Errorf("too many place observations")
	}
	err := s.editWaypoints(world, func(current *waypointWorld) error {
		bw := s.mem.Bases[world]
		active, ok := bw.Areas[b.Name]
		if !ok || bw.Active != b.Name || !active.At.Equal(b.At) {
			return ErrWaypointConflict
		}
		names := slices.Sorted(func(yield func(string) bool) {
			for name := range current.Places {
				if !yield(name) {
					return
				}
			}
		})
		for _, name := range names {
			w := current.Places[name]
			if w.Source != "bot" && !b.Contains(w.Pos) {
				changes.Protected = append(changes.Protected, name)
			}
		}
		for _, c := range candidates {
			if !b.Contains(c.Pos) || c.At.IsZero() || !finitePosition(c.Pos) || c.Entity == "" || slices.Contains(current.Suppressed, c.Kind) || s.protectedHarvestPlace(world, c) {
				changes.Skipped++
				continue
			}
			c.Name = "observed"
			if err := validateWaypoint(world, c); err != nil {
				return err
			}
			match := ""
			for _, name := range names {
				old := current.Places[name]
				bound := old.Base == b.Name || old.Source != "bot" && b.Contains(old.Pos) || slices.Contains(b.Storage, old.Name) || b.Sleeping == old.Name || b.Return == old.Name
				if bound && old.Kind == c.Kind && (old.Entity == c.Entity || old.Pos.FlatDistance(c.Pos) <= old.Radius && absHeight(old.Pos, c.Pos) <= 2) {
					match = name
					break
				}
			}
			if match != "" {
				old := current.Places[match]
				if old.Source != "bot" || slices.Contains(b.Storage, old.Name) || b.Sleeping == old.Name || b.Return == old.Name {
					if !slices.Contains(changes.Protected, match) {
						changes.Protected = append(changes.Protected, match)
					}
					continue
				}
				if !c.At.After(old.At) {
					changes.Unchanged++
					continue
				}
				// Follow a moved observed entity. Other members of the same
				// cluster refresh evidence without shifting its representative.
				if old.Entity == c.Entity {
					old.Pos = c.Pos
				}
				old.At = c.At
				old.Evidence = c.Evidence
				old.Entity = c.Entity
				current.Places[match] = old
				if !slices.Contains(changes.Updated, match) {
					changes.Updated = append(changes.Updated, match)
				}
				continue
			}
			if len(current.Places) >= MaxWaypoints {
				changes.Skipped++
				continue
			}
			for n := 1; n <= MaxWaypoints; n++ {
				name, err := WaypointName(fmt.Sprintf("%s %s %d", b.Name, c.Kind, n))
				if err != nil {
					return err
				}
				if _, exists := current.Places[name]; exists {
					continue
				}
				c.Name = name
				c.Base = b.Name
				c.Source = "bot"
				c.SetBy = "bot"
				current.Places[name] = c
				names = append(names, name)
				changes.Added = append(changes.Added, name)
				break
			}
		}
		return nil
	})
	if err != nil {
		return PlaceChanges{}, err
	}
	return changes, nil
}
func absHeight(a, b protocol.Vec3) float64 {
	d := a[1] - b[1]
	if d < 0 {
		return -d
	}
	return d
}
