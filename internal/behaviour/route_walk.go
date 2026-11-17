package behaviour

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/route"
	"github.com/warricksothr/renki/internal/session"
)

// Navigator plans a way for the body and learns from a leg that failed. Route
// memory is the first one; another may plan over ground only seen.
type Navigator interface {
	// Plan returns legs, each an ordinary move, or the reason there are none.
	Plan(world string, from, to protocol.Vec3) route.Plan
	// Failed says the body could not get from one position on a plan to the
	// next. It reports whether the navigator knew such a link.
	Failed(world string, from, to protocol.Vec3) (bool, error)
}

// RouteMemory is the Navigator over route memory (docs/ROUTES.md): the walked
// graph to plan over, and a way to mark a link that did not carry the body.
type RouteMemory struct {
	Graph      func(world string) []home.RouteNode
	MarkFailed func(world string, from, to protocol.Vec3) (bool, error)
}

// ReasonNoRouteMemory is the reason a RouteMemory with no graph gives.
const ReasonNoRouteMemory = "no_route_memory"

func (m RouteMemory) Plan(world string, from, to protocol.Vec3) route.Plan {
	if m.Graph == nil {
		return route.Plan{Reason: ReasonNoRouteMemory, Explain: "Route memory is not available.", Legs: []route.Leg{}}
	}
	return route.Find(m.Graph(world), from, to)
}

func (m RouteMemory) Failed(world string, from, to protocol.Vec3) (bool, error) {
	if m.MarkFailed == nil {
		return false, nil
	}
	return m.MarkFailed(world, from, to)
}

// MoveStyle is how a walk's moves are made. The zero value is a plain move
// with the walk's own leg timeout.
type MoveStyle struct {
	// Dry keeps the body out of water and AvoidHostiles away from threats:
	// the native move then refuses ground an ordinary one would cross.
	Dry, AvoidHostiles bool
	Run                *bool
	// LegTimeoutS bounds each move. Zero means routeLegTimeout.
	LegTimeoutS float64
}

func (s MoveStyle) order(to protocol.Vec3, stop float64) *session.Order {
	timeout := s.LegTimeoutS
	if timeout <= 0 {
		timeout = routeLegTimeout
	}
	return &session.Order{Task: protocol.TaskMoveTo, TimeoutS: timeout,
		Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &to}, StopDistance: &stop, Dry: s.Dry, AvoidHostiles: s.AvoidHostiles, Run: s.Run}}
}

// Arrived reports whether a move toward target ended there: it succeeded and
// the body stands within the stop distance and on the target's floor.
func Arrived(body, target protocol.Vec3, stop float64, result *Result) bool {
	return result != nil && result.Status == protocol.StatusSucceeded &&
		body.FlatDistance(target) <= stop+routeArriveSlackM && math.Abs(body[1]-target[1]) <= math.Max(routeFloorM, stop)
}

// NextLeg is the walker without memory of its own, for a run that keeps its
// state elsewhere or must survive a restart: it plans from where the body
// stands and returns the first move toward to, with the plan it came from.
// With no plan the move is the direct one. Call it again after each move; the
// run is there once Arrived says so for to itself.
func NextLeg(nav Navigator, world string, from, to protocol.Vec3, stop float64, style MoveStyle) (*session.Order, route.Plan) {
	plan := route.Plan{Reason: ReasonNoRouteMemory, Legs: []route.Leg{}}
	if nav != nil {
		plan = nav.Plan(world, from, to)
	}
	if !plan.Found || len(plan.Legs) < 2 {
		return style.order(to, stop), plan
	}
	return style.order(plan.Legs[0].To, routeLegStopM), plan
}

// LegFailed tells the navigator about a move from NextLeg that did not
// arrive, when the result says something about the ground. plan is the one
// NextLeg returned and body where the move ended. It reports whether a link
// was marked.
func LegFailed(nav Navigator, world string, plan route.Plan, body protocol.Vec3, result *Result) bool {
	if nav == nil || !plan.Found || !groundFailure(result) {
		return false
	}
	from, to, ok := plan.FailedLink(0, body)
	if !ok {
		return false
	}
	known, err := nav.Failed(world, from, to)
	return err == nil && known
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
	nav   Navigator
	world string
	to    protocol.Vec3
	stop  float64
	style MoveStyle

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

func newRouteWalk(nav Navigator, world string, from, to protocol.Vec3, stop float64, style MoveStyle) *routeWalk {
	w := &routeWalk{nav: nav, world: world, to: to, stop: stop, style: style}
	w.replan(from)
	return w
}

// replan plans from where the body is. It reports whether there is a plan.
func (w *routeWalk) replan(from protocol.Vec3) bool {
	w.leg = 0
	if w.nav == nil {
		w.direct = true
		w.notes = append(w.notes, "direct, no navigator")
		return false
	}
	w.plan = w.nav.Plan(w.world, from, w.to)
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
	return w.style.order(to, stop)
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
	here := Arrived(body, target, stop, result)
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
	if from, to, ok := w.plan.FailedLink(w.leg, body); ok && groundFailure(result) {
		if known, err := w.nav.Failed(w.world, from, to); err == nil && known {
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

// detour is the walk over known ground a run falls back on when its native
// move to a position found no way. The run keeps the native move, with its
// water and hostile avoidance, as its first choice; a detour is ordinary
// moves over ground the body has walked, tried once.
type detour struct{ walk *routeWalk }

// foundNoWay reports a reason that says the native move found no way there.
func foundNoWay(reason string) bool {
	return reason == protocol.ReasonUnreachable || reason == protocol.ReasonStuck || reason == protocol.ReasonBlocked
}

// recovers reports whether a failed result should go to next rather than end
// the run: a walk under way answers for its own legs, and before one a move
// that found no way may have a walked one.
func (d *detour) recovers(result Result) bool {
	if d.walk != nil {
		return !d.walk.done
	}
	return result.Status == protocol.StatusFailed && foundNoWay(result.Reason)
}

// next takes the result of the native move, or of a detour's leg. It returns
// the next move to make, or that the body arrived, or an error saying what
// did not succeed: the native reason, or the walk's story.
func (d *detour) next(v View, world string, to protocol.Vec3, stop float64, result *Result, what string) (*session.Order, bool, error) {
	if result == nil {
		return nil, false, errors.New(what + " did not succeed")
	}
	if d.walk != nil {
		if next := d.walk.step(v, result); next != nil {
			return next, false, nil
		}
		if !d.walk.arrived {
			return nil, false, errors.New(what + " did not succeed: " + d.walk.story())
		}
		return nil, true, nil
	}
	if result.Status == protocol.StatusSucceeded {
		return nil, true, nil
	}
	if v.Routes != nil && foundNoWay(result.Reason) {
		walk := newRouteWalk(v.Routes, world, v.State.Pos, to, stop, MoveStyle{})
		if walk.plan.Found && len(walk.plan.Legs) >= 2 {
			d.walk = walk
			return walk.first(), false, nil
		}
	}
	return nil, false, errors.New(strings.TrimSpace(what + " did not succeed: " + result.Reason + " " + result.Detail))
}
