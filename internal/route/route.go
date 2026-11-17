// Package route plans over route memory: the ground the body has walked
// (internal/home/routes.go, docs/ROUTES.md). It is pure. Given the graph, a
// start and a target it returns legs for ordinary moves, or says why it
// cannot, in terms an operator or agent can act on.
package route

import (
	"container/heap"
	"fmt"
	"math"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

const (
	// A position joins the graph at the nearest walked cell within SnapM,
	// and within SnapHeightM in height so it joins on its own floor.
	SnapM       = 4.0
	SnapHeightM = 1.25
	// An inferred way back costs a little more than a way that was walked.
	inferredCost = 1.25
	// Legs are no longer than this, so every leg is a short, checkable move.
	MaxLegM = 10.0
	// A cell is dropped from a leg when it lies this close to the straight
	// line between the leg's ends. Height counts double: a stair's top and
	// foot must stay.
	straightM = 0.6
)

// Why a plan was not found.
const (
	ReasonNone         = ""
	ReasonEmpty        = "nothing_walked"
	ReasonStartOff     = "start_off_walked_ground"
	ReasonTargetOff    = "target_off_walked_ground"
	ReasonDisconnected = "no_walked_way_between"
	// ReasonUnknownPlace is set by callers that resolve a place name.
	ReasonUnknownPlace = "unknown_place"
)

// Anchor is where a position meets the graph.
type Anchor struct {
	Pos protocol.Vec3 `json:"pos"`
	// Cell is the walked cell used, CellPos its walked position and
	// DistanceM how far that is from Pos. Cell is empty when no walked cell
	// is near enough; CellPos and DistanceM then describe the nearest one
	// anywhere, if there is any.
	Cell      string         `json:"cell,omitempty"`
	CellPos   *protocol.Vec3 `json:"cellPos,omitempty"`
	DistanceM float64        `json:"distanceM"`
}

// Leg is one ordinary move: go to To.
type Leg struct {
	To      protocol.Vec3 `json:"to"`
	LengthM float64       `json:"lengthM"`
	// Climb is the height gained, negative going down.
	Climb float64 `json:"climb"`
	// Cells is how many walked cells the leg passes through.
	Cells int `json:"cells"`
	// Inferred counts links taken against the direction they were walked.
	Inferred int `json:"inferred,omitempty"`
	// Path is the walked position of each cell on the leg, first to last. It
	// is empty for a leg that leaves walked ground at either end of a plan.
	Path []protocol.Vec3 `json:"-"`
}

// FailedLink names the link a leg's move most likely failed on: the one out
// of the leg's cell nearest where the body stopped. It reports false for a
// leg off walked ground, which has no link to blame.
func (p Plan) FailedLink(leg int, body protocol.Vec3) (from, to protocol.Vec3, ok bool) {
	if leg < 0 || leg >= len(p.Legs) || len(p.Legs[leg].Path) < 2 {
		return from, to, false
	}
	path := p.Legs[leg].Path
	at := 0
	for i := range path[:len(path)-1] {
		if dist(path[i], body) < dist(path[at], body) {
			at = i
		}
	}
	return path[at], path[at+1], true
}

// Plan is a route, or the explanation for there being none.
type Plan struct {
	Found   bool    `json:"found"`
	Reason  string  `json:"reason,omitempty"`
	Explain string  `json:"explain"`
	Start   Anchor  `json:"start"`
	Target  Anchor  `json:"target"`
	Legs    []Leg   `json:"legs"`
	LengthM float64 `json:"lengthM"`
	Cells   int     `json:"cells"`
	// Avoided counts failed links the search skipped. They need not lie
	// near the route found, so it says the graph has failures, not where.
	Avoided int `json:"avoided"`
	// Closest is the walked cell joined to the start that lies nearest the
	// target, when there is no way through: where a walk would have to be
	// extended from. GapM is how far it is from the target.
	Closest *protocol.Vec3 `json:"closest,omitempty"`
	GapM    float64        `json:"gapM,omitempty"`
}

func dist(a, b protocol.Vec3) float64 {
	return math.Sqrt((a[0]-b[0])*(a[0]-b[0]) + (a[1]-b[1])*(a[1]-b[1]) + (a[2]-b[2])*(a[2]-b[2]))
}

// blocked reports a link a move failed along that has not been walked, in
// that direction, since. Walking the other way proves nothing about this one:
// a gate may open from one side only.
func blocked(e home.RouteEdge) bool { return e.Fail > 0 && !e.WalkedAt.After(e.FailAt) }

type graph struct {
	nodes []home.RouteNode
	index map[string]int
	// leaves and enters mark cells with a usable link out and a usable link
	// in. A cell can have neither: the far end of a teleport is walked but
	// joined to nothing.
	leaves, enters []bool
}

func newGraph(nodes []home.RouteNode) graph {
	g := graph{nodes: nodes, index: make(map[string]int, len(nodes)), leaves: make([]bool, len(nodes)), enters: make([]bool, len(nodes))}
	for i, n := range nodes {
		g.index[n.Key] = i
	}
	for i, n := range nodes {
		for _, e := range n.Links {
			if j, ok := g.index[e.To]; ok && !blocked(e) {
				g.leaves[i], g.enters[j] = true, true
			}
		}
	}
	return g
}

// anchor finds the walked cell a position joins at: the nearest within reach
// on its own floor, preferring a cell that is joined to others (joined says
// which cells are) over a nearer one that leads nowhere. With none in reach,
// it reports the nearest cell anywhere.
func (g graph) anchor(pos protocol.Vec3, joined []bool) (Anchor, int) {
	a := Anchor{Pos: pos}
	best, bestD, lone, loneD, any, anyD := -1, math.Inf(1), -1, math.Inf(1), -1, math.Inf(1)
	for i, n := range g.nodes {
		d := dist(n.Pos, pos)
		if d < anyD {
			any, anyD = i, d
		}
		if d > SnapM || math.Abs(n.Pos[1]-pos[1]) > SnapHeightM {
			continue
		}
		if joined[i] && d < bestD {
			best, bestD = i, d
		}
		if d < loneD {
			lone, loneD = i, d
		}
	}
	if best < 0 {
		best, bestD = lone, loneD
	}
	if best >= 0 {
		p := g.nodes[best].Pos
		a.Cell, a.CellPos, a.DistanceM = g.nodes[best].Key, &p, bestD
	} else if any >= 0 {
		p := g.nodes[any].Pos
		a.CellPos, a.DistanceM = &p, anyD
	}
	return a, best
}

type item struct {
	node     int
	priority float64
}
type queue []item

func (q queue) Len() int           { return len(q) }
func (q queue) Less(i, j int) bool { return q[i].priority < q[j].priority }
func (q queue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *queue) Push(x any)        { *q = append(*q, x.(item)) }
func (q *queue) Pop() any {
	old := *q
	x := old[len(old)-1]
	*q = old[:len(old)-1]
	return x
}

// search runs A* from start to goal. It returns the cells on the way, the
// cells reached at all, and how many links it left out as blocked.
func (g graph) search(start, goal int) (path []int, reached []bool, avoided int) {
	cost := make([]float64, len(g.nodes))
	from := make([]int, len(g.nodes))
	reached = make([]bool, len(g.nodes))
	done := make([]bool, len(g.nodes))
	for i := range cost {
		cost[i], from[i] = math.Inf(1), -1
	}
	cost[start], reached[start] = 0, true
	q := &queue{{start, 0}}
	for q.Len() > 0 {
		cur := heap.Pop(q).(item).node
		if done[cur] {
			continue
		}
		done[cur] = true
		if cur == goal {
			break
		}
		for _, e := range g.nodes[cur].Links {
			next, ok := g.index[e.To]
			if !ok {
				continue
			}
			if blocked(e) {
				avoided++
				continue
			}
			step := dist(g.nodes[cur].Pos, g.nodes[next].Pos)
			if e.N == 0 {
				step *= inferredCost
			}
			if c := cost[cur] + step; c < cost[next] {
				cost[next], from[next], reached[next] = c, cur, true
				heap.Push(q, item{next, c + dist(g.nodes[next].Pos, g.nodes[goal].Pos)})
			}
		}
	}
	if !done[goal] {
		// The queue ran dry, so reached is everything joined to the start.
		return nil, reached, avoided
	}
	for at := goal; at >= 0; at = from[at] {
		path = append([]int{at}, path...)
	}
	return path, reached, avoided
}

// offLine is how far p lies from the straight line a to b, with height
// counted double.
func offLine(p, a, b protocol.Vec3) float64 {
	scale := func(v protocol.Vec3) protocol.Vec3 { return protocol.Vec3{v[0], v[1] * 2, v[2]} }
	p, a, b = scale(p), scale(a), scale(b)
	length := dist(a, b)
	if length == 0 {
		return dist(p, a)
	}
	t := ((p[0]-a[0])*(b[0]-a[0]) + (p[1]-a[1])*(b[1]-a[1]) + (p[2]-a[2])*(b[2]-a[2])) / (length * length)
	t = math.Max(0, math.Min(1, t))
	return dist(p, protocol.Vec3{a[0] + t*(b[0]-a[0]), a[1] + t*(b[1]-a[1]), a[2] + t*(b[2]-a[2])})
}

// legs cuts a path of cells into moves. A leg ends where going on would
// take the walked way more than straightM off the straight line, or make the
// leg longer than MaxLegM. Corners, stair tops and stair feet become leg
// ends; a straight corridor is one leg.
func (g graph) legs(path []int) []Leg {
	var out []Leg
	start := 0
	for start < len(path)-1 {
		end := start + 1
		for next := end + 1; next < len(path); next++ {
			a, b := g.nodes[path[start]].Pos, g.nodes[path[next]].Pos
			ok := dist(a, b) <= MaxLegM
			for mid := start + 1; ok && mid < next; mid++ {
				ok = offLine(g.nodes[path[mid]].Pos, a, b) <= straightM
			}
			if !ok {
				break
			}
			end = next
		}
		leg := Leg{To: g.nodes[path[end]].Pos, Cells: end - start, Climb: g.nodes[path[end]].Pos[1] - g.nodes[path[start]].Pos[1]}
		for i := start; i <= end; i++ {
			leg.Path = append(leg.Path, g.nodes[path[i]].Pos)
		}
		for i := start; i < end; i++ {
			leg.LengthM += dist(g.nodes[path[i]].Pos, g.nodes[path[i+1]].Pos)
			for _, e := range g.nodes[path[i]].Links {
				if e.To == g.nodes[path[i+1]].Key && e.N == 0 {
					leg.Inferred++
				}
			}
		}
		out = append(out, leg)
		start = end
	}
	return out
}

// Find plans from one position to another over walked ground.
func Find(nodes []home.RouteNode, from, to protocol.Vec3) Plan {
	g := newGraph(nodes)
	p := Plan{Legs: []Leg{}}
	var start, goal int
	p.Start, start = g.anchor(from, g.leaves)
	p.Target, goal = g.anchor(to, g.enters)
	switch {
	case len(nodes) == 0:
		p.Reason, p.Explain = ReasonEmpty, "No ground has been walked in this world yet. Walk the body there, or have it follow someone."
		return p
	case start < 0:
		p.Reason = ReasonStartOff
		p.Explain = fmt.Sprintf("The body is not on walked ground: the nearest walked cell on its level is more than %.0f m away (nearest anywhere is %.1f m off). Move it onto known ground first.", SnapM, p.Start.DistanceM)
		return p
	case goal < 0:
		p.Reason = ReasonTargetOff
		p.Explain = fmt.Sprintf("Nothing has been walked within %.0f m of the target on its level (nearest walked cell anywhere is %.1f m off). Walk there once to teach the way.", SnapM, p.Target.DistanceM)
		return p
	}
	path, reached, avoided := g.search(start, goal)
	p.Avoided = avoided
	if path == nil {
		p.Reason = ReasonDisconnected
		best, bestD := -1, math.Inf(1)
		for i, ok := range reached {
			if d := dist(g.nodes[i].Pos, to); ok && d < bestD {
				best, bestD = i, d
			}
		}
		closest := g.nodes[best].Pos
		p.Closest, p.GapM = &closest, bestD
		p.Explain = fmt.Sprintf("Walked ground near the target is not joined to the body's. The joined cell nearest the target is %.1f m short of it; walk that gap once to join them.", bestD)
		if avoided > 0 {
			p.Explain += fmt.Sprintf(" The search skipped %d failed links; one of them may be the missing way.", avoided)
		}
		return p
	}
	p.Found = true
	p.Cells = len(path)
	p.Legs = g.legs(path)
	// The body is near the first walked cell, not on it: start there, so the
	// first long move begins on walked ground.
	if head := dist(from, g.nodes[start].Pos); head > 0.5 {
		p.Legs = append([]Leg{{To: g.nodes[start].Pos, LengthM: head, Climb: g.nodes[start].Pos[1] - from[1]}}, p.Legs...)
	}
	// The last walked cell is near the target, not on it: finish there.
	if tail := dist(g.nodes[goal].Pos, to); tail > 0.5 {
		p.Legs = append(p.Legs, Leg{To: to, LengthM: tail, Climb: to[1] - g.nodes[goal].Pos[1]})
	}
	for _, leg := range p.Legs {
		p.LengthM += leg.LengthM
	}
	p.Explain = fmt.Sprintf("%d legs, %.0f m over %d walked cells.", len(p.Legs), p.LengthM, p.Cells)
	if avoided > 0 {
		p.Explain += fmt.Sprintf(" The search skipped %d failed links.", avoided)
	}
	return p
}
