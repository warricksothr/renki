package home

import (
	"slices"
	"strings"

	"github.com/warricksothr/renki/internal/protocol"
)

// HarvestKind includes all mined resources; ore must not bypass camp protection.
func HarvestKind(kind string) bool { return kind == "wood" || kind == "stone" || kind == "ore" }

func HarvestProtected(bases []Base, pos protocol.Vec3) bool {
	for _, b := range bases {
		if b.Contains(pos) {
			return true
		}
	}
	return false
}

func FallenWood(prefab string) bool {
	lower := strings.ToLower(prefab)
	return strings.Contains(lower, "log") || strings.Contains(lower, "stub") || strings.Contains(lower, "stump")
}

// ResourcePlaceKind describes durable work sites, not every harvestable object.
func ResourcePlaceKind(r protocol.PerceivedResource) string {
	lower := strings.ToLower(r.Prefab)
	switch r.Kind {
	case protocol.ResourceTree:
		if !r.DryRoot() || FallenWood(r.Prefab) {
			return ""
		}
		return "wood"
	case protocol.ResourceRock:
		if !r.DryRoot() {
			return ""
		}
		for _, ore := range []string{"copper", "tin", "silver", "iron", "obsidian"} {
			if strings.Contains(lower, ore) {
				return "ore"
			}
		}
		return "stone"
	case protocol.ResourcePickable:
		if strings.Contains(lower, "flint") {
			return "flint"
		}
		if strings.Contains(lower, "berry") || strings.Contains(lower, "berries") {
			return "berries"
		}
	}
	return ""
}

// Caller holds the store lock. Role bindings survive even for automatic places.
func (s *Store) boundPlace(world, name string) bool {
	for _, b := range s.mem.Bases[world].Areas {
		if b.Return == name || b.Sleeping == name || slices.Contains(b.Storage, name) {
			return true
		}
	}
	return false
}

// Caller holds the store lock, so saving or switching a base cannot race discovery.
func (s *Store) protectedHarvestPlace(world string, w Waypoint) bool {
	if !HarvestKind(w.Kind) {
		return false
	}
	for _, b := range s.mem.Bases[world].Areas {
		if b.Contains(w.Pos) {
			return true
		}
	}
	return false
}

// PruneHarvestPlaces removes only positively invalid automatic sites. It does
// not use owner deletion, which suppresses all future discovery of that kind.
// Unobserved sites outside bases remain last-known hints, never dry-land claims.
func (s *Store) PruneHarvestPlaces(world string, seen *protocol.Perception) error {
	invalid := func(w Waypoint) bool {
		if w.Source != "bot" || !HarvestKind(w.Kind) || s.boundPlace(world, w.Name) {
			return false
		}
		if s.protectedHarvestPlace(world, w) || w.Kind == "wood" && FallenWood(strings.TrimPrefix(w.Evidence, "Observed ")) {
			return true
		}
		if seen != nil {
			for _, r := range seen.Resources {
				if r.Kind != protocol.ResourceTree && r.Kind != protocol.ResourceRock && r.Kind != protocol.ResourceLog && r.Kind != protocol.ResourceStump {
					continue
				}
				if w.Entity != "" && w.Entity == r.ID || w.Pos.FlatDistance(r.Pos) < .25 && absHeight(w.Pos, r.Pos) < .5 {
					// Missing telemetry is unknown, not proof that an existing site is wet.
					if r.WaterLevel != nil && !r.DryRoot() || w.Kind == "wood" && (r.Kind != protocol.ResourceTree || FallenWood(r.Prefab)) {
						return true
					}
				}
			}
		}
		return false
	}
	// Avoid a disk write every perception once the migration has converged.
	s.mu.Lock()
	needed := false
	for _, w := range s.mem.Waypoints[world].Places {
		if invalid(w) {
			needed = true
			break
		}
	}
	s.mu.Unlock()
	if !needed {
		return nil
	}
	return s.editWaypoints(world, func(current *waypointWorld) error {
		for name, w := range current.Places {
			if invalid(w) {
				delete(current.Places, name)
			}
		}
		return nil
	})
}
