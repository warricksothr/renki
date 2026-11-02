package behaviour

import (
	"math"

	"github.com/warricksothr/renki/internal/protocol"
)

// pileSpeed bounds, in metres per second, how fast the bot may move between
// a perception scan and the state paired with it: faster than running, so a
// cart or a boat stays inside it for the short pairing window allowed.
const pileSpeed = 15.0

// observePiles feeds a fresh perception frame to the drop-pile tracker and
// gives the view its snapshot. Perception carries no origin, so it pairs a
// frame only with a state at most a second newer and lets the tracker shrink
// the frame's reach by how far the bot could have moved in between. An older
// frame teaches nothing, and Near answers unknown once the newest is too old.
// view() has already dropped a perception from another connection or world.
func (e *Engine) observePiles(v *View, hello *protocol.Hello) {
	if v.State == nil || hello == nil {
		return
	}
	world := v.State.World
	if p := v.Perception; p != nil && v.State.T-p.T >= 0 && v.State.T-p.T <= 1 {
		var seen protocol.PerceptionConfig
		if _, welcome, ok := e.o.Session.Handshake(); ok && welcome != nil {
			seen = welcome.Perception
		}
		vel := v.State.Vel
		speed := max(pileSpeed, math.Sqrt(vel[0]*vel[0]+vel[1]*vel[1]+vel[2]*vel[2]))
		e.piles.Observe(hello, world, v.State.Pos, speed*(v.State.T-p.T), seen, p, v.PerceptionAt)
	}
	v.Piles = e.piles.Snapshot(hello, world)
}
