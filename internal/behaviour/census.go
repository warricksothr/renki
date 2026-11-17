package behaviour

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// Census opens every container within a radius, nearest first, without
// taking anything. Remembered contents are a hint with an age: after time
// away, on a first visit, or once memory has proved wrong, this is how the
// controller learns what storage holds and which remembered containers are
// gone. It shares search_chests' admission, budgets and safety stops.
//
// Landmarks supplies remembered places. Containers known only as landmarks
// are asked for too, so one that has gone is found out and forgotten.
type Census struct {
	Landmarks func(world string) []home.Landmark
	Routes    RouteMemory
}

type CensusParams struct {
	World         string         `json:"world"`
	Center        *protocol.Vec3 `json:"center,omitempty"`
	Radius        float64        `json:"radius,omitempty"`
	MaxContainers int            `json:"maxContainers,omitempty"`
	DurationS     float64        `json:"durationS,omitempty"`
	TravelM       float64        `json:"travelM,omitempty"`
}

const (
	// A container is reported gone only when the plugin finds none under
	// its id while the body stands this close, well inside the distance at
	// which the client keeps objects loaded.
	censusLoadedM = 30
	// Ordinary walks to a verified standing position, per container.
	censusMaxWalks = 2
	// Refusals in a row, not counting gone containers, before giving up.
	censusMaxFailures = 6
	// The plugin verifies standing positions within 12 m of the body. A
	// farther one is not walked to.
	censusMaxWalkM = 15
	// Records of one container can differ by this much: landmark positions
	// are rounded to half a metre.
	censusSamePlaceM = .6
	// Closer than this, walking toward the container itself gains nothing.
	censusNearM = 4
	// An ordinary move returns no travel receipt and exists to go round an
	// obstacle, so each walk is charged this multiple of its straight line.
	censusWalkCharge = 2
)

func (p *CensusParams) defaults() {
	if p.Radius == 0 {
		p.Radius = 30
	}
	if p.MaxContainers == 0 {
		p.MaxContainers = 64
	}
	if p.DurationS == 0 {
		p.DurationS = 600
	}
	if p.TravelM == 0 {
		p.TravelM = 600
	}
}

func (Census) Validate(raw json.RawMessage) error {
	var p CensusParams
	if err := Decode(raw, &p); err != nil {
		return err
	}
	p.defaults()
	if strings.TrimSpace(p.World) == "" || !finite(p.Radius) || p.Radius < 10 || p.Radius > 100 ||
		p.MaxContainers < 1 || p.MaxContainers > 96 || !finite(p.DurationS) || p.DurationS < 30 || p.DurationS > 1200 ||
		!finite(p.TravelM) || p.TravelM < 10 || p.TravelM > 2000 {
		return errors.New("census needs a world, radius 10–100m, 1–96 containers, 30–1200s and 10–2000m travel")
	}
	if p.Center != nil {
		for _, n := range *p.Center {
			if !finite(n) || math.Abs(n) > 1e6 {
				return errors.New("invalid census center")
			}
		}
	}
	return nil
}

func (t Census) Start(raw json.RawMessage) (Runner, error) {
	if err := t.Validate(raw); err != nil {
		return nil, err
	}
	var p CensusParams
	_ = Decode(raw, &p)
	p.defaults()
	// The search run supplies admission, budget accounting and discovery.
	// Its item is never matched here.
	base := chestSearchRun{p: SearchChestsParams{World: p.World, Item: "census", Center: p.Center, Radius: p.Radius,
		MaxChests: 32, DurationS: p.DurationS, TravelM: p.TravelM}, seen: map[string]bool{}}
	return &censusRun{p: p, base: base, landmarks: t.Landmarks, routes: t.Routes}, nil
}

func (t Census) Ready(raw json.RawMessage, v View) bool {
	r, err := t.Start(raw)
	return err == nil && r.(*censusRun).base.Check(v) == nil
}

// CensusEntry is one container's outcome. Outcome is opened, gone or refused.
type CensusEntry struct {
	Entity  string        `json:"entity"`
	Pos     protocol.Vec3 `json:"pos"`
	Outcome string        `json:"outcome"`
	Reason  string        `json:"reason,omitempty"`
	Stacks  int           `json:"stacks"`
	First   bool          `json:"first,omitempty"`
	Walks   int           `json:"walks,omitempty"`
	// Route tells the walks made for the container: each plan over walked
	// ground, any leg that failed, and how the walk ended.
	Route string `json:"route,omitempty"`
}

type censusRun struct {
	p         CensusParams
	base      chestSearchRun
	entries   []CensusEntry
	pending   *chestCandidate
	known     bool
	walking   bool
	landmarks func(world string) []home.Landmark
	routes    RouteMemory
	// walk is the walk under way, and stories those made for the pending
	// container.
	walk    *routeWalk
	stories []string
	// walked holds the positions walked to for the pending container.
	walked   []protocol.Vec3
	failures int
	// walkCharged is the travel total once the current walk was charged.
	walkCharged float64
	reported    bool
}

// Recover keeps the run going after any failed walk to a standing spot: the
// walk is only a means, and the container is asked for again from wherever
// the body stopped. Inspections recover as the search's do.
func (r *censusRun) Recover(result Result) bool { return r.walking || r.base.Recover(result) }
func (r *censusRun) Dispatched(id string) error { r.base.taskID = id; return nil }

// Entries returns the outcomes so far, for status and tests.
func (r *censusRun) Entries() []CensusEntry { return slices.Clone(r.entries) }

type inspectionReceipt struct {
	Travel   *float64        `json:"travel"`
	Standing []protocol.Vec3 `json:"standing"`
}

func parseInspection(detail string) (inspectionReceipt, bool) {
	var receipt inspectionReceipt
	if !strings.HasPrefix(detail, "inspection:") || json.Unmarshal([]byte(strings.TrimPrefix(detail, "inspection:")), &receipt) != nil ||
		receipt.Travel == nil || !finite(*receipt.Travel) || *receipt.Travel < 0 {
		return inspectionReceipt{}, false
	}
	return receipt, true
}

// walkTarget picks where an ordinary move should take the body before the
// container is asked for again, and how near it must get. The inspection's
// own bounded walk has already failed, or the container was too far to be
// loaded; an ordinary move may go round what stopped it.
//
// A position the plugin verified for reach and body clearance comes first.
// Without one, as when the bounded walker finds no route between buildings
// and never measures a standing position, the body walks toward the container
// itself and stops within reach of it.
func (r *censusRun) walkTarget(from protocol.Vec3, spots []protocol.Vec3) (protocol.Vec3, float64, bool) {
	affordable := func(to protocol.Vec3) bool {
		ok := to.FlatDistance(r.base.center) <= r.p.Radius && r.base.travel+censusWalkCharge*distance(from, to) < r.p.TravelM
		for _, n := range to {
			ok = ok && finite(n) && math.Abs(n) <= 1e6
		}
		return ok && !slices.Contains(r.walked, to)
	}
	for _, spot := range spots {
		if affordable(spot) && distance(from, spot) <= censusMaxWalkM {
			return spot, .4, true
		}
	}
	if to := r.pending.pos; distance(from, to) > censusNearM && affordable(to) {
		return to, 2.5, true
	}
	return protocol.Vec3{}, 0, false
}

// addLandmarks queues containers remembered only as places. One that stands
// where a known container does is the same box under an older id.
func (r *censusRun) addLandmarks(v View) {
	if r.landmarks == nil {
		return
	}
	b := &r.base
	var known []protocol.Vec3
	for _, c := range v.Containers {
		known = append(known, c.Pos)
	}
	for _, c := range v.Perception.Structures {
		if c.Kind == protocol.StructureContainer {
			known = append(known, c.Pos)
		}
	}
	for _, c := range b.queue {
		known = append(known, c.pos)
	}
	for _, l := range r.landmarks(r.p.World) {
		if l.Kind != protocol.StructureContainer || b.seen[l.Entity] || l.Pos.FlatDistance(b.center) > r.p.Radius {
			continue
		}
		b.seen[l.Entity] = true
		same := false
		for _, pos := range known {
			same = same || distance(pos, l.Pos) <= censusSamePlaceM
		}
		if !same {
			b.queue = append(b.queue, chestCandidate{l.Entity, l.Pos})
			known = append(known, l.Pos)
		}
	}
}

func (r *censusRun) record(outcome, reason string, stacks int) {
	r.entries = append(r.entries, CensusEntry{Entity: r.pending.id, Pos: r.pending.pos, Outcome: outcome, Reason: reason,
		Stacks: stacks, First: outcome == "opened" && !r.known, Walks: len(r.walked), Route: strings.Join(r.stories, ", then ")})
	if outcome == "refused" {
		r.failures++
	} else if outcome == "opened" {
		r.failures = 0
	}
	if outcome == "opened" {
		// Another id queued for this same place is this container.
		at := r.pending.pos
		r.base.queue = slices.DeleteFunc(r.base.queue, func(c chestCandidate) bool { return distance(c.pos, at) <= censusSamePlaceM })
	}
	r.pending, r.walked, r.walking, r.walk, r.stories = nil, nil, false, nil, nil
}

func (r *censusRun) inspect(v View) *session.Order {
	b := &r.base
	b.legStartTravel = b.travel
	b.beforeSeq = 0
	for _, old := range v.Containers {
		if old.Entity == r.pending.id && old.Seq > b.beforeSeq {
			b.beforeSeq = old.Seq
		}
	}
	seconds := math.Min(45, r.p.DurationS-v.Now.Sub(b.began).Seconds())
	return &session.Order{Task: protocol.TaskInspectContainer, Args: &protocol.InspectContainerArgs{Container: r.pending.id,
		Scope: &protocol.MoveScope{Center: b.center, Radius: r.p.Radius, TravelM: math.Min(500, r.p.TravelM-b.travel), DurationS: seconds}}, TimeoutS: seconds}
}

func (r *censusRun) spent(v View) bool {
	b := &r.base
	return r.p.DurationS-v.Now.Sub(b.began).Seconds() < 1 || r.p.TravelM-b.travel < 1
}

func (r *censusRun) Next(v View, result *Result) (*session.Order, error) {
	b := &r.base
	if err := b.Check(v); err != nil {
		return nil, err
	}
	if b.began.IsZero() {
		b.center = v.State.Pos
		if r.p.Center != nil {
			b.center = *r.p.Center
		}
		if v.State.Pos.FlatDistance(b.center) > r.p.Radius {
			return nil, errors.New("body is outside census radius")
		}
		b.began = v.Now
		b.lastPos = v.State.Pos
		b.health = sampleHealth(v)
		b.discover(v)
		r.addLandmarks(v)
	}
	if r.reported {
		return nil, nil
	}
	if r.pending != nil {
		if result == nil || result.TaskID != b.taskID {
			return nil, errors.New("missing matching census outcome")
		}
		if r.walking {
			if next := r.walk.step(v, result); next != nil {
				return next, nil
			}
			// Whatever the walk achieved, ask again from where the body
			// now stands: the inspection re-measures reach itself. The
			// walk was charged in full when ordered, so the displacement
			// Check just added is not counted a second time.
			r.walking = false
			r.stories = append(r.stories, r.walk.story())
			b.travel = r.walkCharged
			if r.spent(v) {
				r.record("refused", "budget ended before a second look", 0)
			} else {
				return r.inspect(v), nil
			}
		} else if err := r.settle(v, result); err != nil || r.walking {
			if err != nil {
				return nil, err
			}
			return r.walk.first(), nil
		}
	}
	b.discover(v)
	if b.stop || len(b.queue) == 0 || len(r.entries) >= r.p.MaxContainers || r.failures >= censusMaxFailures || r.spent(v) {
		r.reported = true
		line := r.line()
		if len([]rune(line)) > 200 {
			line = string([]rune(line)[:197]) + "..."
		}
		return &session.Order{Task: protocol.TaskSay, Args: &protocol.SayArgs{Text: line}, TimeoutS: 5}, nil
	}
	// Nearest to where the body is now, not to where it began: a walk round
	// a row changes which neighbour is next.
	next := 0
	for i, c := range b.queue {
		a, best := distance(v.State.Pos, c.pos), distance(v.State.Pos, b.queue[next].pos)
		if a < best || a == best && c.id < b.queue[next].id {
			next = i
		}
	}
	c := b.queue[next]
	b.queue = slices.Delete(b.queue, next, next+1)
	r.pending, r.known = &c, false
	for _, old := range v.Containers {
		r.known = r.known || old.Entity == c.id
	}
	return r.inspect(v), nil
}

// settle applies an inspection's result. It leaves r.walking set, with the
// target appended to r.walked, when the next order is an ordinary walk.
func (r *censusRun) settle(v View, result *Result) error {
	b := &r.base
	receipt, measured := parseInspection(result.Detail)
	if measured {
		b.travel = math.Max(b.travel, b.legStartTravel+*receipt.Travel)
	} else if result.Status != protocol.StatusRejected {
		// As in search_chests: without native travel accounting the budget
		// cannot be trusted, so nothing further is asked.
		b.stop = true
	}
	switch {
	case result.Status == protocol.StatusSucceeded:
		var seen *protocol.Container
		for _, c := range v.Containers {
			if c.Entity == r.pending.id && c.Task == b.taskID && c.Direction == protocol.TaskInspectContainer && c.Seq > b.beforeSeq && math.Abs(v.State.T-c.T) <= 3 {
				seen = c
				break
			}
		}
		if seen == nil {
			return ErrWait
		}
		r.record("opened", "", len(seen.Items))
	case result.Reason == protocol.ReasonInvalidTarget && distance(v.State.Pos, r.pending.pos) <= censusLoadedM:
		r.record("gone", "", 0)
	case result.Reason == protocol.ReasonUnreachable && measured || result.Reason == protocol.ReasonInvalidTarget:
		// Unreachable from here, or too far away for the game to have it
		// loaded: walk nearer and ask again.
		if !b.stop && len(r.walked) < censusMaxWalks && !r.spent(v) {
			if to, stop, ok := r.walkTarget(v.State.Pos, receipt.Standing); ok {
				// Over walked ground when route memory knows a way, and
				// otherwise the one direct move.
				walk := newRouteWalk(r.routes, r.p.World, v.State.Pos, to, stop)
				charge := censusWalkCharge * math.Max(distance(v.State.Pos, to), walk.lengthM(v.State.Pos))
				if b.travel+charge < r.p.TravelM {
					r.walked = append(r.walked, to)
					r.walk, r.walking = walk, true
					// Charged now, in full, whether or not the walk
					// arrives or is planned again.
					b.travel += charge
					r.walkCharged = b.travel
					return nil
				}
			}
		}
		r.record("refused", result.Reason, 0)
	default:
		r.record("refused", result.Reason, 0)
	}
	return nil
}

// Gone names the containers this run proved absent, for memory to forget.
func (r *censusRun) Gone() (string, []string) {
	var ids []string
	for _, e := range r.entries {
		if e.Outcome == "gone" {
			ids = append(ids, e.Entity)
		}
	}
	return r.p.World, ids
}

func (r *censusRun) counts() (opened, first, gone, refused, stacks int) {
	for _, e := range r.entries {
		switch e.Outcome {
		case "opened":
			opened++
			stacks += e.Stacks
			if e.First {
				first++
			}
		case "gone":
			gone++
		default:
			refused++
		}
	}
	return
}

func (r *censusRun) unchecked() int {
	n := len(r.base.queue) + r.base.unknown
	if r.pending != nil {
		n++
	}
	return n
}

func (r *censusRun) line() string {
	opened, first, gone, refused, _ := r.counts()
	return fmt.Sprintf("Census: opened %d containers (%d for the first time), %d gone, %d I could not open, %d not checked.",
		opened, first, gone, refused, r.unchecked())
}

func (r *censusRun) InterruptedSummary() string { return r.Summary() }

func (r *censusRun) Summary() string {
	_, _, _, _, stacks := r.counts()
	parts := []string{r.line(), fmt.Sprintf("%d stacks seen.", stacks)}
	var gone, refused, walks []string
	for _, e := range r.entries {
		switch e.Outcome {
		case "gone":
			gone = append(gone, e.Entity)
		case "refused":
			refused = append(refused, e.Entity+": "+e.Reason)
		}
		if e.Route != "" {
			walks = append(walks, e.Entity+": "+e.Route)
		}
	}
	if len(gone) > 0 {
		parts = append(parts, "Gone, and forgotten: "+strings.Join(gone, ", ")+".")
	}
	if len(refused) > 0 {
		parts = append(parts, "Not opened: "+strings.Join(refused, "; ")+".")
	}
	if len(walks) > 0 {
		parts = append(parts, "Walks: "+strings.Join(walks, " | ")+".")
	}
	if r.base.stop {
		parts = append(parts, "Stopped: native travel receipt missing.")
	}
	return strings.Join(parts, " ")
}
