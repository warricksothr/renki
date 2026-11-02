package behaviour

import (
	"math"

	"github.com/warricksothr/renki/internal/piles"
	"github.com/warricksothr/renki/internal/protocol"
)

const (
	// pileSpeed bounds, in metres per second, how fast the bot moves without
	// jumping: faster than running, so a cart or a boat stays inside it.
	pileSpeed = 15.0
	// pilePairing is how much newer than a scan, in game seconds, a state
	// may be and still be paired with it.
	pilePairing = 1.0
	// pileStates bounds the recent states kept to find one before a scan.
	pileStates = 32
)

type pileState struct {
	pos protocol.Vec3
	t   float64
}

// pileWatch is the engine's drop-pile tracker with the recent states it saw,
// so the distance from a scan to the state paired with it can be bounded.
type pileWatch struct {
	piles.Tracker
	conn   *protocol.Hello
	world  string
	states []pileState
}

// observePiles feeds a fresh perception frame to the drop-pile tracker and
// gives the view its snapshot. view() has already dropped a perception from
// another connection or world.
//
// Perception carries no origin, so a frame is paired with a state at most
// pilePairing newer, and the tracker shrinks its reach by how far that state
// may be from where the scan ran. Moving without jumping, that is pileSpeed
// times the time between them. A jump (a portal, a respawn, a teleport) may
// fall before or after the scan. If before, the scan ran near the paired
// state and that bound holds. If after, the scan ran within pileSpeed times
// the time since the last state at or before it of that state, so the
// distance from there to the paired state counts too. The larger of the two
// holds either way. With no state seen at or before the scan the frame is
// not paired. An older frame teaches nothing, and Near answers unknown once
// the newest is too old.
func (e *Engine) observePiles(v *View, hello *protocol.Hello) {
	if v.State == nil || hello == nil {
		return
	}
	w, st := &e.piles, v.State
	world := st.World
	if w.conn != hello || w.world != world || len(w.states) > 0 && st.T < w.states[len(w.states)-1].t {
		w.conn, w.world, w.states = hello, world, nil
	}
	if n := len(w.states); n == 0 || st.T > w.states[n-1].t {
		w.states = append(w.states, pileState{st.Pos, st.T})
		cut := 0
		for cut < len(w.states)-1 && (st.T-w.states[cut+1].t > pilePairing+1 || len(w.states)-cut > pileStates) {
			cut++
		}
		w.states = append([]pileState(nil), w.states[cut:]...)
	}
	if p := v.Perception; p != nil && st.T-p.T >= 0 && st.T-p.T <= pilePairing {
		var before *pileState
		for i := len(w.states) - 1; i >= 0; i-- {
			if w.states[i].t <= p.T {
				before = &w.states[i]
				break
			}
		}
		if before != nil {
			speed := max(pileSpeed, math.Sqrt(st.Vel[0]*st.Vel[0]+st.Vel[1]*st.Vel[1]+st.Vel[2]*st.Vel[2]))
			drift := max(speed*(st.T-p.T), distance(st.Pos, before.pos)+speed*(p.T-before.t))
			var seen protocol.PerceptionConfig
			if _, welcome, ok := e.o.Session.Handshake(); ok && welcome != nil {
				seen = welcome.Perception
			}
			w.Observe(hello, world, st.Pos, drift, seen, p, v.PerceptionAt)
		}
	}
	v.Piles = w.Snapshot(hello, world)
}
