package behaviour

import (
	"fmt"
	"math"
	"strings"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/route"
	"github.com/warricksothr/renki/internal/session"
)

// RouteMemory is what a run needs of route memory (docs/ROUTES.md): the walked
// graph to plan over, and a way to say a link did not carry the body.
type RouteMemory struct {
	Graph  func(world string) []home.RouteNode
	Failed func(world string, from, to protocol.Vec3) (bool, error)
}

const (
	// An intermediate leg ends this near its cell.
	routeLegStopM = .5
	// A leg has arrived when the body is within its stop distance plus this,
	// and within routeFloorM of the leg end's height: on the intended floor,
	// not under or over it.
	routeArriveSlackM = 1.0
	routeFloorM       = 1.0
	// Plans made after the first, each after a leg failed.
	routeMaxReplans = 2
	routeLegTimeout = 30
)

// routeWalk takes the body to a position over walked ground: one ordinary
// move per leg of a plan, each checked on arrival. A leg that fails marks its
// link failed and the way is planned again from where the body stopped. With
// no plan it makes the one direct move a run would have made without route
// memory, so new ground can still be learnt.
type routeWalk struct {
	mem   RouteMemory
	world string
	to    protocol.Vec3
	stop  float64

	plan    route.Plan
	leg     int
	replans int
	direct  bool
	done    bool
	arrived bool
	// notes is the walk's story, one short entry per plan or failure.
	notes []string
	// afford, when set, is asked before a walk planned again is taken up:
	// the run charges the new way's length, or refuses it.
	afford func(lengthM float64) bool
}

// groundFailure reports a result that says something about the ground: the
// move found no way, got stuck, or reported success short of the leg's end or
// on another floor. A timeout, a refusal or a stop says nothing lasting, and
// marking a link for one would close good ground after a passing threat.
func groundFailure(result *Result) bool {
	if result == nil {
		return false
	}
	switch result.Reason {
	case protocol.ReasonUnreachable, protocol.ReasonStuck, protocol.ReasonBlocked, protocol.ReasonDeepWater:
		return true
	}
	return result.Status == protocol.StatusSucceeded
}

// abandon ends the walk where the body stands, with the reason in its story.
func (w *routeWalk) abandon(why string) {
	if !w.done {
		w.done = true
		w.notes = append(w.notes, why)
	}
}

func newRouteWalk(mem RouteMemory, world string, from, to protocol.Vec3, stop float64) *routeWalk {
	w := &routeWalk{mem: mem, world: world, to: to, stop: stop}
	w.replan(from)
	return w
}

// replan plans from where the body is. It reports whether there is a plan.
func (w *routeWalk) replan(from protocol.Vec3) bool {
	w.leg = 0
	if w.mem.Graph == nil {
		w.direct = true
		w.notes = append(w.notes, "direct, no route memory")
		return false
	}
	w.plan = route.Find(w.mem.Graph(w.world), from, w.to)
	if !w.plan.Found {
		w.notes = append(w.notes, "no route ("+w.plan.Reason+")")
		return false
	}
	w.notes = append(w.notes, fmt.Sprintf("route %d legs %.0f m", len(w.plan.Legs), w.plan.LengthM))
	return true
}

// lengthM is how far the walk is expected to go from a position.
func (w *routeWalk) lengthM(from protocol.Vec3) float64 {
	if w.plan.Found {
		return w.plan.LengthM
	}
	return distance(from, w.to)
}

// order is the move for the current leg, or the direct move.
func (w *routeWalk) order() *session.Order {
	to, stop := w.to, w.stop
	if !w.direct && w.leg < len(w.plan.Legs)-1 {
		to, stop = w.plan.Legs[w.leg].To, routeLegStopM
	}
	return &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &to}, StopDistance: &stop}, TimeoutS: routeLegTimeout}
}

// first starts the walk. Without a plan the walk is one direct move.
func (w *routeWalk) first() *session.Order {
	if !w.plan.Found {
		w.direct = true
	}
	return w.order()
}

// step takes the result of the last move and returns the next, or nil when
// the walk is over: w.arrived says how it ended.
func (w *routeWalk) step(v View, result *Result) *session.Order {
	if w.done {
		return nil
	}
	body := v.State.Pos
	target, stop := w.to, w.stop
	last := w.direct || w.leg >= len(w.plan.Legs)-1
	if !last {
		target, stop = w.plan.Legs[w.leg].To, routeLegStopM
	}
	here := result != nil && result.Status == protocol.StatusSucceeded &&
		body.FlatDistance(target) <= stop+routeArriveSlackM && math.Abs(body[1]-target[1]) <= math.Max(routeFloorM, stop)
	if here && !last {
		// Near enough to the end already: the legs left would only shuffle
		// the body about inside the distance it was asked to stop at.
		if body.FlatDistance(w.to) <= w.stop && math.Abs(body[1]-w.to[1]) <= math.Max(routeFloorM, w.stop) {
			w.done, w.arrived = true, true
			return nil
		}
		w.leg++
		return w.order()
	}
	if here || w.direct {
		w.done, w.arrived = true, here
		if !here {
			w.notes = append(w.notes, "direct move "+outcomeOf(result))
		}
		return nil
	}
	note := fmt.Sprintf("leg %d of %d %s", w.leg+1, len(w.plan.Legs), outcomeOf(result))
	if from, to, ok := w.plan.FailedLink(w.leg, body); ok && w.mem.Failed != nil && groundFailure(result) {
		if known, err := w.mem.Failed(w.world, from, to); err == nil && known {
			note += ", link marked failed"
		}
	}
	w.notes = append(w.notes, note)
	if w.replans >= routeMaxReplans {
		w.done = true
		return nil
	}
	w.replans++
	if !w.replan(body) {
		w.done = true
		return nil
	}
	if w.afford != nil && !w.afford(w.plan.LengthM) {
		w.abandon("no budget for the new way")
		return nil
	}
	return w.order()
}

func outcomeOf(result *Result) string {
	switch {
	case result == nil:
		return "had no result"
	case result.Status == protocol.StatusSucceeded:
		return "stopped short"
	case result.Reason != "":
		return "failed (" + result.Reason + ")"
	}
	return "failed (" + result.Status + ")"
}

// story is the walk in one line, for a run's summary.
func (w *routeWalk) story() string {
	end := "did not arrive"
	if w.arrived {
		end = "arrived"
	} else if !w.done {
		end = "walking"
	}
	return strings.Join(append(append([]string{}, w.notes...), end), "; ")
}
