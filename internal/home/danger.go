package home

import (
	"errors"
	"maps"
	"math"
	"sort"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

const MaxHarms = 256
const MaxDangerZones = 128
const KeepDanger = 7 * 24 * time.Hour
const DangerHalfLife = 24 * time.Hour

// Harm is an observed injury, not an inference from falling food maximums.
// Empty By/Prefab means the attacker/cause was not identified by the game.
type Harm struct {
	World  string        `json:"world"`
	Pos    protocol.Vec3 `json:"pos"`
	At     time.Time     `json:"at"`
	Event  string        `json:"event"`
	Amount float64       `json:"amount,omitempty"`
	By     string        `json:"by,omitempty"`
	Prefab string        `json:"prefab,omitempty"`
}

// DangerZone summarizes nearby injuries at read time, with a decayed score.
type DangerZone struct {
	Pos    protocol.Vec3 `json:"pos"`
	Radius float64       `json:"radius"`
	Score  float64       `json:"score"`
	At     time.Time     `json:"at"`
	Hits   int           `json:"hits"`
	Deaths int           `json:"deaths"`
	By     string        `json:"by,omitempty"`
	Prefab string        `json:"prefab,omitempty"`
}

// RememberHarms atomically persists a cross-world batch. A failed save can be
// retried without double-counting. Readers and callers never share slices.
func (s *Store) RememberHarms(observations []Harm, now time.Time) error {
	for _, o := range observations {
		if o.World == "" || o.At.IsZero() || o.At.After(now) || (o.Event != protocol.EventDamaged && o.Event != protocol.EventDied) || math.IsNaN(o.Amount) || math.IsInf(o.Amount, 0) || o.Amount < 0 || (o.Event == protocol.EventDamaged && o.Amount == 0) {
			return errors.New("harm needs world, valid time, injury event and finite positive damage")
		}
		for _, v := range o.Pos {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return errors.New("harm position must be finite")
			}
		}
	}
	if len(observations) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.mem.Harms
	next := maps.Clone(old)
	if next == nil {
		next = map[string][]Harm{}
	}
	// Clone before append, including when the old slice has spare capacity.
	touched := map[string]bool{}
	for _, o := range observations {
		if !touched[o.World] {
			next[o.World] = append([]Harm(nil), next[o.World]...)
			touched[o.World] = true
		}
		if now.Sub(o.At) <= KeepDanger {
			next[o.World] = append(next[o.World], o)
		}
	}
	for world := range touched {
		entries := next[world]
		sort.SliceStable(entries, func(i, j int) bool { return entries[i].At.After(entries[j].At) })
		kept := make([]Harm, 0, min(len(entries), MaxHarms))
		for _, entry := range entries {
			if !entry.At.After(now) && now.Sub(entry.At) <= KeepDanger && len(kept) < MaxHarms {
				kept = append(kept, entry)
			}
		}
		next[world] = kept
	}
	s.mem.Harms = next
	if err := s.save(); err != nil {
		s.mem.Harms = old
		return err
	}
	return nil
}

func (s *Store) Harms(world string) []Harm {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Harm(nil), s.mem.Harms[world]...)
}

// DangerZones clusters newest-first around fixed observed positions. Heights
// remain separate so harm on another storey does not poison a nearby floor.
func (s *Store) DangerZones(world string, now time.Time) []DangerZone {
	zones := []DangerZone{}
	for _, h := range s.Harms(world) {
		age := now.Sub(h.At)
		if age < 0 || age > KeepDanger {
			continue
		}
		severity := math.Min(h.Amount/10, 5)
		if h.Event == protocol.EventDied {
			severity = 5
		}
		score := severity * math.Exp2(-float64(age)/float64(DangerHalfLife))
		if score <= 0 {
			continue
		}
		i := -1
		for j, z := range zones {
			if z.Pos.FlatDistance(h.Pos) <= 8 && math.Abs(z.Pos[1]-h.Pos[1]) <= 4 {
				i = j
				break
			}
		}
		if i < 0 {
			if len(zones) == MaxDangerZones {
				continue
			}
			zones = append(zones, DangerZone{Pos: h.Pos, Radius: 12, At: h.At, By: h.By, Prefab: h.Prefab})
			i = len(zones) - 1
		}
		zones[i].Score = math.Min(10, zones[i].Score+score)
		if h.Event == protocol.EventDied {
			zones[i].Deaths++
		} else {
			zones[i].Hits++
		}
	}
	return zones
}

// DangerRisk is a soft target-selection penalty; it does not certify routes.
func DangerRisk(zones []DangerZone, pos protocol.Vec3) float64 {
	risk := 0.0
	for _, z := range zones {
		if z.Radius <= 0 || z.Score <= 0 || math.Abs(z.Pos[1]-pos[1]) > 4 {
			continue
		}
		d := z.Pos.FlatDistance(pos)
		if d < z.Radius {
			risk += z.Score * (1 - d/z.Radius)
		}
	}
	return risk
}
