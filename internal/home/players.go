package home

import (
	"errors"
	"maps"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/warricksothr/renki/internal/protocol"
)

const (
	MaxPlayerWorlds  = 16
	MaxPlayers       = 128
	MaxPlayerHistory = 64
	KeepPlayers      = 14 * 24 * time.Hour
)

// PlayerEncounter remembers evidence, never a player's current whereabouts.
// ResolutionM is a cell width; TimeResolutionS describes the rounded-down At.
type PlayerEncounter struct {
	World           string        `json:"-"`
	Name            string        `json:"name"`
	Pos             protocol.Vec3 `json:"pos"`
	At              time.Time     `json:"at"`
	Source          string        `json:"source"`
	ResolutionM     float64       `json:"resolutionM"`
	TimeResolutionS int           `json:"timeResolutionS"`
}

func ValidPlayerEncounter(o PlayerEncounter) bool {
	if o.Name == "" || strings.TrimSpace(o.Name) != o.Name || len(o.Name) > 64 || strings.ContainsFunc(o.Name, unicode.IsControl) || o.At.IsZero() || (o.Source != "heard" && o.Source != "seen") {
		return false
	}
	switch o.ResolutionM {
	case 0, 5, 25, 100, 500:
	default:
		return false
	}
	switch o.TimeResolutionS {
	case 0, 1, 60, 900, 3600:
	default:
		return false
	}
	for _, v := range o.Pos {
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 1e7 {
			return false
		}
	}
	return true
}

// CompactPlayerHistory ages on reads as well as writes. Nested cells/buckets
// prevent repeated rounding from drifting or recovering discarded precision.
func CompactPlayerHistory(observations []PlayerEncounter, now time.Time) []PlayerEncounter {
	input := append([]PlayerEncounter(nil), observations...)
	sort.SliceStable(input, func(i, j int) bool { return input[i].At.After(input[j].At) })
	type key struct {
		name, source string
		at           int64
		resolution   float64
	}
	seen := map[key]bool{}
	names := map[string]bool{}
	out := []PlayerEncounter{}
	for _, o := range input {
		age := now.Sub(o.At)
		if !ValidPlayerEncounter(o) || age < 0 || age > KeepPlayers {
			continue
		}
		grid, bucket := 5.0, 1
		switch {
		case age >= 2*24*time.Hour:
			grid, bucket = 500, 3600
		case age >= 6*time.Hour:
			grid, bucket = 100, 900
		case age >= 10*time.Minute:
			grid, bucket = 25, 60
		}
		// Never regain precision after a backwards wall-clock adjustment.
		if o.ResolutionM > grid {
			grid = o.ResolutionM
		}
		if o.TimeResolutionS > bucket {
			bucket = o.TimeResolutionS
		}
		for i, v := range o.Pos {
			o.Pos[i] = math.Floor(v/grid)*grid + grid/2
		}
		o.At = o.At.UTC().Truncate(time.Duration(bucket) * time.Second)
		o.ResolutionM, o.TimeResolutionS = grid, bucket
		k := key{o.Name, o.Source, o.At.Unix(), grid}
		if seen[k] {
			continue
		}
		if !names[o.Name] && len(names) >= MaxPlayers {
			continue
		}
		seen[k] = true
		names[o.Name] = true
		out = append(out, o)
	}
	// Reserve history for each age tier. Even sampling preserves its newest
	// and oldest evidence, allowing old encounters to survive a busy present.
	tiers := map[string][4][]PlayerEncounter{}
	for _, o := range out {
		tier := 0
		switch age := now.Sub(o.At); {
		case age >= 2*24*time.Hour:
			tier = 3
		case age >= 6*time.Hour:
			tier = 2
		case age >= 10*time.Minute:
			tier = 1
		}
		buckets := tiers[o.Name]
		buckets[tier] = append(buckets[tier], o)
		tiers[o.Name] = buckets
	}
	out = nil
	for _, buckets := range tiers {
		for tier, bucket := range buckets {
			limit := []int{32, 16, 8, 8}[tier]
			if len(bucket) <= limit {
				out = append(out, bucket...)
				continue
			}
			for i := 0; i < limit; i++ {
				out = append(out, bucket[i*(len(bucket)-1)/(limit-1)])
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].At.Equal(out[j].At) {
			return out[i].At.After(out[j].At)
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Source < out[j].Source
	})
	return out
}

// PlayerHistory returns independent observations, newest first, for one world.
func (s *Store) PlayerHistory(world string, now time.Time) []PlayerEncounter {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := CompactPlayerHistory(s.mem.Players[world], now)
	for i := range out {
		out[i].World = world
	}
	return out
}

// RememberPlayers atomically batches worlds and physically removes precision
// and expired evidence, including during quiet periods with no new events.
func (s *Store) RememberPlayers(observations []PlayerEncounter, now time.Time) error {
	for _, o := range observations {
		if o.World == "" || !ValidPlayerEncounter(o) {
			return errors.New("invalid player encounter")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.mem.Players
	next := maps.Clone(previous)
	if next == nil {
		next = map[string][]PlayerEncounter{}
	}
	grouped := map[string][]PlayerEncounter{}
	for _, o := range observations {
		world := o.World
		o.World = ""
		grouped[world] = append(grouped[world], o)
	}
	for world, batch := range grouped {
		// Do not append into the previous map's slice (save may fail).
		next[world] = append(append([]PlayerEncounter(nil), next[world]...), batch...)
	}
	for world, history := range next {
		next[world] = CompactPlayerHistory(history, now)
		if len(next[world]) == 0 {
			delete(next, world)
		}
	}
	worlds := make([]string, 0, len(next))
	for world := range next {
		worlds = append(worlds, world)
	}
	sort.Slice(worlds, func(i, j int) bool {
		a, b := next[worlds[i]][0].At, next[worlds[j]][0].At
		if !a.Equal(b) {
			return a.After(b)
		}
		return worlds[i] < worlds[j]
	})
	for _, world := range worlds[min(len(worlds), MaxPlayerWorlds):] {
		delete(next, world)
	}
	if reflect.DeepEqual(previous, next) || len(previous) == 0 && len(next) == 0 {
		return nil
	}
	s.mem.Players = next
	if err := s.save(); err != nil {
		s.mem.Players = previous
		return err
	}
	return nil
}
