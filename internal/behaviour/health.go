package behaviour

import "math"

type healthSample struct {
	hp, maximum float64
	food        map[string]float64
	injury      *uint64
}

func sampleHealth(v View) healthSample {
	s := healthSample{hp: v.State.HP, maximum: v.State.HPMax, food: map[string]float64{}}
	if v.InjurySeq != nil {
		copy := *v.InjurySeq
		s.injury = &copy
	}
	if v.State.Food != nil {
		for _, f := range v.State.Food.Active {
			s.food[f.Prefab] = f.RemainingS
		}
	}
	return s
}

func (before healthSample) lost(after healthSample) bool {
	if before.injury != nil {
		if after.injury == nil || *before.injury != *after.injury {
			return true
		}
		// A meal can first clamp HP to its decayed maximum and then raise
		// the maximum again between observations. Accepted injury events
		// distinguish that from a real hit during the same food refresh.
		if after.maximum > before.maximum {
			for name, remaining := range after.food {
				old, known := before.food[name]
				if !known || remaining > old+1 {
					return false
				}
			}
		}
	}
	return after.hp < math.Min(before.hp, after.maximum)-.15
}
