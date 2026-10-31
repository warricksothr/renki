package goal

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/warricksothr/renki/internal/protocol"
)

// Fixed areas belong to the goal anchor, not the bot's drifting task origin.
// A failed route consumes an attempted area, never evidence of empty coverage.
func nextArea(g Record, pos protocol.Vec3) (*SearchPlan, json.RawMessage, string) {
	radius := math.Min(24, (g.Radius-52)/4)
	reach := math.Min(80, g.Radius-52-radius)
	if radius < 12 || reach < radius*2 {
		return nil, nil, ""
	}
	used := map[int]bool{}
	for _, search := range g.Searches {
		used[search.Plan.Sector] = true
	}
	var best *SearchPlan
	bestDistance := math.Inf(1)
	for _, sector := range []int{0, 4, 2, 6, 1, 5, 3, 7} {
		if used[sector] {
			continue
		}
		angle := float64(sector) * math.Pi / 4
		center := protocol.Vec3{g.Center[0] + math.Cos(angle)*reach, g.Center[1], g.Center[2] + math.Sin(angle)*reach}
		gap := math.Hypot(center[0]-pos[0], center[2]-pos[2])
		if gap > 140 || gap >= bestDistance-1e-6 {
			continue
		}
		bestDistance = gap
		best = &SearchPlan{Sector: sector, Origin: pos, Center: &center, Radius: radius}
	}
	if best == nil {
		return nil, nil, ""
	}
	raw, _ := json.Marshal(map[string]any{"world": g.World, "searchCenter": best.Center, "searchRadius": best.Radius, "searchAnchor": g.Center, "searchLimit": g.Radius - 52})
	return best, raw, fmt.Sprintf("area:%d:%g:%g", best.Sector, (*best.Center)[0], (*best.Center)[2])
}

func areaSector(anchor, center protocol.Vec3) int {
	angle := math.Atan2(center[2]-anchor[2], center[0]-anchor[0])
	return (int(math.Round(angle/(math.Pi/4))) + 8) % 8
}
