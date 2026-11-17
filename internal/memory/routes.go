package memory

import (
	"context"
	"log/slog"
	"math"
	"slices"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

const (
	routeBatchEvery = 30 * time.Second
	// A new sample is kept once the body has moved this far from the last.
	routeSampleM = 0.5
	// A longer silence between samples breaks the chain: what happened in
	// between was not seen.
	routeSampleGap = 3 * time.Second
	maxRouteBatch  = 4096
)

// Routes records where the body walks, without issuing tasks. Call Tick and
// Flush from one goroutine. Whatever moved the body counts: an order, a
// behaviour, or following an owner through a building.
type Routes struct {
	Session *session.Session
	Store   *home.Store
	Logger  *slog.Logger
	world   string
	chain   routeChain
	seen    *protocol.State
	pending []home.RouteStep
	since   time.Time
}

func (r *Routes) Run(ctx context.Context) {
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			r.Tick()
			r.Flush()
			return
		case <-t.C:
			r.Tick()
		}
	}
}

func (r *Routes) Tick() { r.tick(time.Now()) }

// walkable reports whether a state frame shows the body standing on ground
// it could be sent back to.
func walkable(s *protocol.State) bool {
	return s != nil && s.Phase == protocol.PhaseInWorld && s.World != "" && s.HP > 0 &&
		slices.Contains(s.Flags, protocol.FlagGrounded) && !slices.Contains(s.Flags, protocol.FlagSwimming) &&
		(s.Location == nil || !s.Location.InWater)
}

func (r *Routes) tick(now time.Time) {
	state, at := r.Session.StateObservation()
	if state == nil {
		r.chain.broken()
	} else if state != r.seen {
		r.seen = state
		if state.World != r.world {
			// Do not give a batch the next world's name.
			if !r.flush(now) {
				return
			}
			r.world = state.World
			r.chain.broken()
		}
		if step, ok := r.chain.observe(state, at, r.Session.ConnectionID()); ok && len(r.pending) < maxRouteBatch {
			if len(r.pending) == 0 {
				r.since = now
			}
			r.pending = append(r.pending, step)
		}
	}
	r.flushDue(now)
}

// routeChain turns successive state frames into walked steps. It joins two
// samples only when it saw the body on the ground at both, on one connection,
// close together in space and time.
type routeChain struct {
	conn   uint64
	last   *protocol.Vec3
	lastAt time.Time
}

func (c *routeChain) broken() { c.last = nil }

func (c *routeChain) observe(state *protocol.State, at time.Time, conn uint64) (home.RouteStep, bool) {
	if conn != c.conn {
		c.conn, c.last = conn, nil
	}
	if !walkable(state) {
		// In the air, in water, or not in the world: the chain is broken
		// here, so a jump down or a swim is never joined into a walk.
		c.last = nil
		return home.RouteStep{}, false
	}
	pos := state.Pos
	switch {
	case c.last == nil || at.Sub(c.lastAt) > routeSampleGap || at.Before(c.lastAt) || distance3(*c.last, pos) > home.RouteLinkMaxM:
		c.last, c.lastAt = &pos, at
	case distance3(*c.last, pos) >= routeSampleM:
		step := home.RouteStep{From: *c.last, To: pos, At: at}
		c.last, c.lastAt = &pos, at
		return step, true
	default:
		// Standing still keeps the chain alive.
		c.lastAt = at
	}
	return home.RouteStep{}, false
}

func distance3(a, b protocol.Vec3) float64 {
	return math.Sqrt((a[0]-b[0])*(a[0]-b[0]) + (a[1]-b[1])*(a[1]-b[1]) + (a[2]-b[2])*(a[2]-b[2]))
}

func (r *Routes) flushDue(now time.Time) {
	if len(r.pending) > 0 && now.Sub(r.since) >= routeBatchEvery {
		r.flush(now)
	}
}

// Flush persists the batch; failure retains it.
func (r *Routes) Flush() bool { return r.flush(time.Now()) }

func (r *Routes) flush(now time.Time) bool {
	if len(r.pending) == 0 {
		return true
	}
	if err := r.Store.RememberRoute(r.world, r.pending, now); err != nil {
		if r.Logger != nil {
			r.Logger.Warn("could not save walked route", "err", err)
		}
		r.since = now
		return false
	}
	r.pending = nil
	return true
}
