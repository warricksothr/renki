package behaviour

import (
	"github.com/warricksothr/renki/internal/protocol"
	"math"
	"slices"
	"time"
)

// View contains immutable protocol snapshots, scoped to the current session
// and world. A missing observation fails a condition that needs it.
type View struct {
	Now        time.Time
	State      *protocol.State
	Inventory  *protocol.Inventory
	Perception *protocol.Perception
	Containers []*protocol.Container
	Owners     []string
	Home       *protocol.Vec3
	Idle       bool
}

func (c Conditions) Match(v View) bool {
	s := v.State
	if s == nil || s.Phase != protocol.PhaseInWorld || s.HP <= 0 {
		return false
	}
	if len(c.Items) > 0 || c.MinFreeSlots > 0 {
		if v.Inventory == nil {
			return false
		}
		if v.Inventory.Free() < c.MinFreeSlots {
			return false
		}
		for p, n := range c.Items {
			if v.Inventory.Count(p) < n {
				return false
			}
		}
	}
	if c.MaxWeightFraction != nil && (s.WeightMax <= 0 || s.Weight/s.WeightMax > *c.MaxWeightFraction) {
		return false
	}
	if c.Encumbered != nil && s.Encumbered() != *c.Encumbered {
		return false
	}
	if c.Stranded != nil && s.Stranded() != *c.Stranded {
		return false
	}
	if c.Night != nil && s.Time.Night != *c.Night {
		return false
	}
	if c.Near != nil && (s.World != c.Near.World || distance(s.Pos, c.Near.Pos) > c.Near.Radius) {
		return false
	}
	if c.AtHomeRadius != nil && (v.Home == nil || distance(s.Pos, *v.Home) > *c.AtHomeRadius) {
		return false
	}
	if c.Owner != nil {
		found := false
		if v.Perception != nil && s.T-v.Perception.T >= 0 && s.T-v.Perception.T <= 5 {
			for _, p := range v.Perception.Characters {
				if p.Kind == protocol.CharacterPlayer && p.HPFrac > 0 && slices.Contains(v.Owners, p.Name) && (c.Owner.Name == "" || c.Owner.Name == p.Name) && distance(s.Pos, p.Pos) <= c.Owner.Radius {
					found = true
					break
				}
			}
		}
		if !found {
			return false
		}
	}
	for _, stock := range c.Stock {
		found := false
		for _, chest := range v.Containers {
			age := s.T - chest.T
			if chest.Entity == stock.Container && age >= 0 && age <= stock.MaxAgeS && chest.Inventory().Count(stock.Prefab) >= stock.MinCount {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func distance(a, b protocol.Vec3) float64 {
	return math.Sqrt((a[0]-b[0])*(a[0]-b[0]) + (a[1]-b[1])*(a[1]-b[1]) + (a[2]-b[2])*(a[2]-b[2]))
}
