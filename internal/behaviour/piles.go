package behaviour

import (
	"math"

	"github.com/warricksothr/renki/internal/piles"
	"github.com/warricksothr/renki/internal/protocol"
)

// pileSpeed bounds, in metres per second, how fast the bot may move between
// a perception scan and the state paired with it: faster than running, so a
// cart or a boat stays inside it for the short pairing window allowed.
const pileSpeed = 15.0

// pileWatch is the engine's drop-pile tracker, with the last state it saw so
// a jump (a portal, a respawn, a teleport) is not mistaken for movement.
type pileWatch struct {
	piles.Tracker
	conn   *protocol.Hello
	world  string
	pos    protocol.Vec3
	t      float64
	jumped float64
	seen   bool
}

// observePiles feeds a fresh perception frame to the drop-pile tracker and
// gives the view its snapshot. Perception carries no origin, so it pairs a
// frame only with a state at most a second newer and lets the tracker shrink
// the frame's reach by how far the bot could have moved in between. A frame
// scanned before the bot jumped further than that is never paired, since it
// says nothing about where the bot is now. An older frame teaches nothing,
// and Near answers unknown once the newest is too old. view() has already
// dropped a perception from another connection or world.
func (e *Engine) observePiles(v *View, hello *protocol.Hello) {
	if v.State == nil || hello == nil {
		return
	}
	w, st := &e.piles, v.State
	world := st.World
	speed := max(pileSpeed, math.Sqrt(st.Vel[0]*st.Vel[0]+st.Vel[1]*st.Vel[1]+st.Vel[2]*st.Vel[2]))
	if !w.seen || w.conn != hello || w.world != world || st.T < w.t {
		w.conn, w.world, w.jumped = hello, world, st.T
	} else if st.T > w.t && distance(st.Pos, w.pos) > speed*(st.T-w.t)+1 {
		w.jumped = st.T
	}
	w.pos, w.t, w.seen = st.Pos, st.T, true
	if p := v.Perception; p != nil && p.T >= w.jumped && st.T-p.T >= 0 && st.T-p.T <= 1 {
		var seen protocol.PerceptionConfig
		if _, welcome, ok := e.o.Session.Handshake(); ok && welcome != nil {
			seen = welcome.Perception
		}
		w.Observe(hello, world, st.Pos, speed*(st.T-p.T), seen, p, v.PerceptionAt)
	}
	v.Piles = w.Snapshot(hello, world)
}
