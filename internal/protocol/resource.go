package protocol

import "math"

// DryRoot requires direct water evidence at the resource root. Dry ground or
// supported flooring nearby cannot establish that a submerged resource is dry.
func (r PerceivedResource) DryRoot() bool {
	for _, n := range r.Pos {
		if !finite(n) || math.Abs(n) > 1e6 {
			return false
		}
	}
	return r.WaterLevel != nil && finite(*r.WaterLevel) && r.Pos[1] > *r.WaterLevel+.2
}
