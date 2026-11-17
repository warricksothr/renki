package session

import (
	"errors"
	"fmt"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

// SetHarvestAreas supplies authoritative current-world base circles. Call before
// starting the bridge. Every harvest path, including raw API orders, uses it.
func (s *Session) SetHarvestAreas(areas func(world string) []protocol.HarvestExclusion) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.harvestAreas = areas
}

func (s *Session) prepareHarvest(hello *protocol.Hello, world, phase string, args *protocol.HarvestArgs, areas func(string) []protocol.HarvestExclusion) error {
	if world == "" || phase != protocol.PhaseInWorld {
		return errors.New("join a world before harvesting")
	}
	seenHello, seenWorld, seen, at := s.PerceptionObservation()
	if seen == nil || seenHello != hello || seenWorld != world || time.Since(at) < 0 || time.Since(at) > 5*time.Second {
		return errors.New("harvest requires a fresh resource observation in this world")
	}
	// Callers cannot supply a smaller policy, including an explicit empty list.
	args.Exclude = areas(world)
	for _, r := range seen.Resources {
		if r.ID != args.Entity {
			continue
		}
		if r.Kind != protocol.ResourceTree && r.Kind != protocol.ResourceLog && r.Kind != protocol.ResourceStump && r.Kind != protocol.ResourceRock {
			return errors.New("target is not a harvestable resource")
		}
		for _, area := range args.Exclude {
			if area.Contains(r.Pos) {
				return fmt.Errorf("harvest target is inside a protected saved base (%.0f m radius)", area.Radius)
			}
		}
		if !r.DryRoot() {
			return errors.New("harvest requires a resource root above known water level")
		}
		return nil
	}
	return errors.New("harvest target is not in fresh perception")
}
