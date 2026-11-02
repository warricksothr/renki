package home

import (
	"errors"
	"maps"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

const MaxPickups = 256
const KeepPickups = 14 * 24 * time.Hour

// Pickup is origin evidence, never an instruction to return or an assignment
// of a nearby chest. Count is the measured gain, not current carried stock.
type Pickup struct {
	ID       string        `json:"id"`
	World    string        `json:"world"`
	WorldUID string        `json:"worldUid"`
	Prefab   string        `json:"prefab"`
	Entity   string        `json:"entity,omitempty"`
	Pos      protocol.Vec3 `json:"pos"`
	Count    int           `json:"count"`
	At       time.Time     `json:"at"`
}

func (s *Store) Pickups(world string) []Pickup {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.mem.Pickups[world])
}

func (s *Store) RememberPickups(observations []Pickup, now time.Time) error {
	for _, p := range observations {
		if p.ID == "" || p.World == "" || p.WorldUID == "" || p.Prefab == "" || len(p.Prefab) > 100 || p.Count <= 0 || p.Count > 1000000 || p.At.IsZero() || p.At.After(now) {
			return errors.New("pickup requires scoped identity, item, count and observation time")
		}
		for _, n := range p.Pos {
			if math.IsNaN(n) || math.IsInf(n, 0) {
				return errors.New("pickup position must be finite")
			}
		}
	}
	if len(observations) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.mem.Pickups
	next := maps.Clone(old)
	if next == nil {
		next = map[string][]Pickup{}
	}
	worlds := map[string]bool{}
	for _, p := range observations {
		worlds[p.World] = true
	}
	for world := range worlds {
		entries := map[string]Pickup{}
		for _, p := range next[world] {
			entries[p.ID] = p
		}
		for _, p := range observations {
			if p.World == world {
				if _, exists := entries[p.ID]; !exists {
					entries[p.ID] = p
				}
			}
		}
		kept := []Pickup{}
		for _, p := range entries {
			if !p.At.After(now) && now.Sub(p.At) <= KeepPickups {
				kept = append(kept, p)
			}
		}
		sort.Slice(kept, func(i, j int) bool {
			if !kept[i].At.Equal(kept[j].At) {
				return kept[i].At.After(kept[j].At)
			}
			return kept[i].ID < kept[j].ID
		})
		if len(kept) > MaxPickups {
			kept = kept[:MaxPickups]
		}
		next[world] = kept
	}
	s.mem.Pickups = next
	if err := s.save(); err != nil {
		s.mem.Pickups = old
		return err
	}
	return nil
}
