package home

import (
	"errors"
	"maps"
	"slices"
	"sort"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

const MaxContainers = 4096
const KeepContainers = 30 * 24 * time.Hour

// RememberedContainer retains an observation, including its source when a
// reconciled place now has another object. At never advances on reconciliation.
type RememberedContainer struct {
	Entity string `json:"entity"`
	// ObservedEntity is set when contents came from a superseded network
	// object at this place. They remain a hint until this entity is opened.
	ObservedEntity string                   `json:"observedEntity,omitempty"`
	Kind           string                   `json:"kind"`
	Prefab         string                   `json:"prefab,omitempty"`
	Pos            protocol.Vec3            `json:"pos"`
	Size           [2]int                   `json:"size"`
	Items          []protocol.InventoryItem `json:"items"`
	At             time.Time                `json:"at"`
}

func (c RememberedContainer) Inventory() *protocol.Inventory {
	return &protocol.Inventory{Size: c.Size, Items: slices.Clone(c.Items)}
}
func (c RememberedContainer) Room(prefab string, stackMax int) int {
	return c.Inventory().Room(prefab, stackMax)
}
func (c RememberedContainer) Fresh(now time.Time, maxAge time.Duration) bool {
	age := now.Sub(c.At)
	return (c.ObservedEntity == "" || c.ObservedEntity == c.Entity) && !c.At.IsZero() && age >= 0 && age <= maxAge
}

// RememberedContainers returns independent observations, newest first.
func (s *Store) RememberedContainers(world string) []RememberedContainer {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]RememberedContainer, 0, len(s.mem.Containers[world]))
	for _, c := range s.mem.Containers[world] {
		c.Items = slices.Clone(c.Items)
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].At.Equal(out[j].At) {
			return out[i].At.After(out[j].At)
		}
		return out[i].Entity < out[j].Entity
	})
	return out
}

// RememberContainers replaces whole snapshots, pruning oldest observations.
// Failed persistence leaves both the file and the in-memory view unchanged.
func (s *Store) RememberContainers(world string, observations []RememberedContainer, now time.Time) error {
	if world == "" {
		return errors.New("container observation needs a world")
	}
	for _, c := range observations {
		if c.Entity == "" || c.At.IsZero() || c.Size[0] <= 0 || c.Size[1] <= 0 {
			return errors.New("container observation needs entity, time and positive dimensions")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := maps.Clone(s.mem.Containers[world])
	if next == nil {
		next = map[string]RememberedContainer{}
	}
	for _, c := range observations {
		old := next[c.Entity]
		if old.At.After(c.At) {
			continue
		}
		if c.Prefab == "" {
			c.Prefab = old.Prefab
		}
		if c.Kind == "" {
			c.Kind = old.Kind
		}
		if c.Kind == "" {
			c.Kind = protocol.StructureContainer
		}
		c.Items = slices.Clone(c.Items)
		next[c.Entity] = c
	}
	for id, c := range next {
		if now.Sub(c.At) > KeepContainers {
			delete(next, id)
		}
	}
	if len(next) > MaxContainers {
		ordered := make([]RememberedContainer, 0, len(next))
		for _, c := range next {
			ordered = append(ordered, c)
		}
		sort.Slice(ordered, func(i, j int) bool {
			if !ordered[i].At.Equal(ordered[j].At) {
				return ordered[i].At.Before(ordered[j].At)
			}
			return ordered[i].Entity < ordered[j].Entity
		})
		for _, c := range ordered[:len(next)-MaxContainers] {
			delete(next, c.Entity)
		}
	}
	previous := s.mem.Containers
	worlds := maps.Clone(previous)
	if worlds == nil {
		worlds = map[string]map[string]RememberedContainer{}
	}
	worlds[world] = next
	s.mem.Containers = worlds
	if err := s.save(); err != nil {
		s.mem.Containers = previous
		return err
	}
	return nil
}
