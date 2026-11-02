package behaviour

import (
	"github.com/warricksothr/renki/internal/protocol"
)

// observePiles feeds a fresh perception frame to the drop-pile tracker and
// gives the view its snapshot. Like drop_appeared, it trusts only a frame no
// more than five seconds behind state; an older one teaches nothing, and
// Near then answers unknown once the newest frame is too old.
// view() has already dropped a perception from another connection or world.
func (e *Engine) observePiles(v *View, hello *protocol.Hello) {
	if v.State == nil || hello == nil {
		return
	}
	world := v.State.World
	if p := v.Perception; p != nil && v.State.T-p.T >= 0 && v.State.T-p.T <= 5 {
		var seen protocol.PerceptionConfig
		if _, welcome, ok := e.o.Session.Handshake(); ok && welcome != nil {
			seen = welcome.Perception
		}
		e.piles.Observe(hello, world, v.State.Pos, seen, p, v.PerceptionAt)
	}
	v.Piles = e.piles.Snapshot(hello, world)
}
