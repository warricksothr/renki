package behaviour

import (
	"math"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

// SettleGap is how long a player may go unseen before their settle area is
// forgotten and the next sighting opens a fresh one.
const SettleGap = 30 * time.Second

// SettleDrift is the time constant of the area's centre: it moves toward
// the player's position as an average weighted this far back, slowly
// enough that a player walking through leaves the area rather than
// dragging it along.
const SettleDrift = 30 * time.Second

// SettleArea is the shape of a settle area: it opens at Radius around the
// player, may widen to MaxRadius once they have stayed Delay, and lets them
// stray Slack past its edge before it resets.
type SettleArea struct {
	Radius, MaxRadius, Slack float64
	Delay                    time.Duration
}

// Settle tracks how long a player has stayed in one small area. Feed it
// their positions in time order with Observe. The area's centre follows
// their positions (SettleDrift). Once they have stayed Delay, a step into
// the slack band past its edge widens it to that distance, up to
// MaxRadius. A step beyond the band, or SettleGap unseen, opens a fresh
// area where they stand.
type Settle struct {
	Centre protocol.Vec3
	Radius float64
	Since  time.Time // when this area opened
	Last   time.Time // the latest sighting
	seen   bool
}

// Observe adds a sighting of the player at pos.
func (s *Settle) Observe(pos protocol.Vec3, at time.Time, a SettleArea) {
	if !s.seen || at.Sub(s.Last) > SettleGap || distance(pos, s.Centre) > s.Radius+a.Slack {
		*s = Settle{Centre: pos, Radius: a.Radius, Since: at, Last: at, seen: true}
		return
	}
	if d := distance(pos, s.Centre); d > s.Radius && at.Sub(s.Since) >= a.Delay {
		s.Radius = min(a.MaxRadius, d)
	}
	if dt := at.Sub(s.Last); dt > 0 {
		pull := 1 - math.Exp(-dt.Seconds()/SettleDrift.Seconds())
		for i := range s.Centre {
			s.Centre[i] += pull * (pos[i] - s.Centre[i])
		}
		s.Last = at
	}
}

// Settled is how long the player has been in the current area at now, or
// zero if they have not been seen within SettleGap.
func (s *Settle) Settled(now time.Time) time.Duration {
	if !s.seen || now.Sub(s.Last) > SettleGap || now.Before(s.Since) {
		return 0
	}
	return now.Sub(s.Since)
}

// LootReach is how far from its anchor loot looks after the anchor has been
// settled for settled: BaseRadius until SettleDelayS has passed, so a player
// walking through gains nothing, then linearly up to Radius over GrowS.
func LootReach(settled time.Duration, p LootParams) float64 {
	grown := (settled.Seconds() - p.SettleDelayS) / p.GrowS
	if grown <= 0 {
		return p.BaseRadius
	}
	return p.BaseRadius + (p.Radius-p.BaseRadius)*min(1, grown)
}
