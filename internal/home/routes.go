package home

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"sort"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

// Route memory keeps where the body has actually walked, per world, so the
// controller can plan through buildings, up stairs and between floors over
// ground that is known to carry it. It records; planning is in package route.
//
// The world is cut into cells one metre square and half a metre high, so two
// floors of one building never share a cell. A cell remembers the last exact
// position walked inside it, which is a better place to send the body than
// the cell's centre: the centre may be inside a wall.
const (
	RouteCellM  = 1.0
	RouteFloorM = 0.5
	// Samples farther apart than this are not joined: the body teleported,
	// went through a portal or the recording had a gap.
	RouteLinkMaxM = 3.0
	// A step no higher than this is taken to be walkable both ways. A
	// bigger drop is recorded one way only, so a fall is never planned as
	// a climb.
	RouteClimbM   = 0.75
	MaxRouteCells = 20000
	KeepRoutes    = 60 * 24 * time.Hour
)

// RouteCell is a cell's integer address: x, height level, z.
type RouteCell [3]int32

func RouteCellOf(p protocol.Vec3) RouteCell {
	return RouteCell{int32(math.Floor(p[0] / RouteCellM)), int32(math.Round(p[1] / RouteFloorM)), int32(math.Floor(p[2] / RouteCellM))}
}

func (c RouteCell) Key() string { return fmt.Sprintf("%d,%d,%d", c[0], c[1], c[2]) }

// RouteStep is one stretch the body walked between two samples.
type RouteStep struct {
	From, To protocol.Vec3
	At       time.Time
}

type routeLink struct {
	// N counts times walked in this direction. A link with N of 0 is the
	// inferred reverse of a walked one.
	N  int       `json:"n,omitempty"`
	At time.Time `json:"at"`
	// WalkedAt is when the body last went along the link in this direction.
	// It stays zero for a link only ever inferred.
	WalkedAt time.Time `json:"walkedAt,omitzero"`
	Fail     int       `json:"fail,omitempty"`
	FailAt   time.Time `json:"failAt,omitzero"`
}

type routeCell struct {
	Pos   protocol.Vec3        `json:"pos"`
	At    time.Time            `json:"at"`
	Links map[string]routeLink `json:"links,omitempty"`
}

type routeWorld struct {
	Cells map[string]routeCell `json:"cells"`
}

// RouteEdge and RouteNode are an independent copy of the graph for planning
// and inspection.
type RouteEdge struct {
	To       string    `json:"to"`
	N        int       `json:"n"`
	At       time.Time `json:"at"`
	WalkedAt time.Time `json:"walkedAt,omitzero"`
	Fail     int       `json:"fail,omitempty"`
	FailAt   time.Time `json:"failAt,omitzero"`
}

type RouteNode struct {
	Key   string        `json:"key"`
	Pos   protocol.Vec3 `json:"pos"`
	At    time.Time     `json:"at"`
	Links []RouteEdge   `json:"links,omitempty"`
}

// Routes returns the world's graph, cells ordered by key.
func (s *Store) Routes(world string) []RouteNode {
	s.mu.Lock()
	defer s.mu.Unlock()
	cells := s.mem.Routes[world].Cells
	out := make([]RouteNode, 0, len(cells))
	for key, c := range cells {
		n := RouteNode{Key: key, Pos: c.Pos, At: c.At}
		for to, l := range c.Links {
			n.Links = append(n.Links, RouteEdge{To: to, N: l.N, At: l.At, WalkedAt: l.WalkedAt, Fail: l.Fail, FailAt: l.FailAt})
		}
		sort.Slice(n.Links, func(i, j int) bool { return n.Links[i].To < n.Links[j].To })
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func finiteVec(p protocol.Vec3) bool {
	for _, n := range p {
		if math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > 1e6 {
			return false
		}
	}
	return true
}

func stepLength(a, b protocol.Vec3) float64 {
	return math.Sqrt((a[0]-b[0])*(a[0]-b[0]) + (a[1]-b[1])*(a[1]-b[1]) + (a[2]-b[2])*(a[2]-b[2]))
}

// RememberRoute adds walked steps. Steps longer than RouteLinkMaxM mark both
// ends as walked but join nothing. Failed persistence changes nothing.
func (s *Store) RememberRoute(world string, steps []RouteStep, now time.Time) error {
	if world == "" {
		return errors.New("route observation needs a world")
	}
	for _, st := range steps {
		if st.At.IsZero() || !finiteVec(st.From) || !finiteVec(st.To) {
			return errors.New("route step needs finite positions and a time")
		}
	}
	if len(steps) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cells := maps.Clone(s.mem.Routes[world].Cells)
	if cells == nil {
		cells = map[string]routeCell{}
	}
	touched := map[string]bool{}
	cell := func(p protocol.Vec3, at time.Time) string {
		key := RouteCellOf(p).Key()
		c := cells[key]
		if !touched[key] {
			// Copy on first write: the stored links map is shared with
			// the previous snapshot until this batch is saved.
			c.Links = maps.Clone(c.Links)
			touched[key] = true
		}
		if !c.At.After(at) {
			c.Pos, c.At = p, at
		}
		cells[key] = c
		return key
	}
	link := func(from, to string, at time.Time, walked bool) {
		c := cells[from]
		if c.Links == nil {
			c.Links = map[string]routeLink{}
		}
		l := c.Links[to]
		if walked {
			l.N++
			if !l.WalkedAt.After(at) {
				l.WalkedAt = at
			}
		}
		if !l.At.After(at) {
			l.At = at
		}
		c.Links[to] = l
		cells[from] = c
	}
	for _, st := range steps {
		a, b := cell(st.From, st.At), cell(st.To, st.At)
		if a == b || stepLength(st.From, st.To) > RouteLinkMaxM {
			continue
		}
		link(a, b, st.At, true)
		if math.Abs(st.To[1]-st.From[1]) <= RouteClimbM {
			link(b, a, st.At, false)
		}
	}
	pruneRoutes(cells, now)
	return s.saveRoutes(world, cells)
}

// pruneRoutes drops cells not walked for KeepRoutes, then the oldest beyond
// MaxRouteCells, and every link that pointed at a dropped cell.
func pruneRoutes(cells map[string]routeCell, now time.Time) {
	dropped := false
	for key, c := range cells {
		if now.Sub(c.At) > KeepRoutes {
			delete(cells, key)
			dropped = true
		}
	}
	if len(cells) > MaxRouteCells {
		keys := make([]string, 0, len(cells))
		for key := range cells {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			a, b := cells[keys[i]], cells[keys[j]]
			if !a.At.Equal(b.At) {
				return a.At.Before(b.At)
			}
			return keys[i] < keys[j]
		})
		for _, key := range keys[:len(cells)-MaxRouteCells] {
			delete(cells, key)
		}
		dropped = true
	}
	if !dropped {
		return
	}
	for key, c := range cells {
		var dangling []string
		for to := range c.Links {
			if _, ok := cells[to]; !ok {
				dangling = append(dangling, to)
			}
		}
		if len(dangling) == 0 {
			continue
		}
		c.Links = maps.Clone(c.Links)
		for _, to := range dangling {
			delete(c.Links, to)
		}
		cells[key] = c
	}
}

func (s *Store) saveRoutes(world string, cells map[string]routeCell) error {
	previous := s.mem.Routes
	next := maps.Clone(previous)
	if next == nil {
		next = map[string]routeWorld{}
	}
	next[world] = routeWorld{Cells: cells}
	s.mem.Routes = next
	if err := s.save(); err != nil {
		s.mem.Routes = previous
		return err
	}
	return nil
}

// MarkRouteFailed records that the body could not get from one cell to the
// next, in that direction. It reports whether such a link was known.
func (s *Store) MarkRouteFailed(world string, from, to protocol.Vec3, now time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, b := RouteCellOf(from).Key(), RouteCellOf(to).Key()
	cells := maps.Clone(s.mem.Routes[world].Cells)
	c, ok := cells[a]
	if !ok {
		return false, nil
	}
	l, ok := c.Links[b]
	if !ok {
		return false, nil
	}
	c.Links = maps.Clone(c.Links)
	l.Fail++
	l.FailAt = now
	c.Links[b] = l
	cells[a] = c
	return true, s.saveRoutes(world, cells)
}

// RouteStats summarises a graph for inspection.
type RouteStats struct {
	Cells int `json:"cells"`
	Links int `json:"links"`
	// Walked counts links the body went along; the rest are inferred reverses.
	Walked int `json:"walked"`
	Failed int `json:"failed"`
	// Levels lists the heights walked, lowest first, with their cell counts.
	Levels []RouteLevel `json:"levels"`
	// Groups are sets of cells joined to each other, largest first. More
	// than one means there are places with no walked way between them.
	Groups []RouteGroup `json:"groups"`
	Newest time.Time    `json:"newest,omitzero"`
}

type RouteLevel struct {
	Y     float64 `json:"y"`
	Cells int     `json:"cells"`
}

type RouteGroup struct {
	Cells int `json:"cells"`
	// Centre is the mean walked position; MinY and MaxY its height range.
	Centre protocol.Vec3 `json:"centre"`
	MinY   float64       `json:"minY"`
	MaxY   float64       `json:"maxY"`
}

// SummariseRoutes computes RouteStats. Groups ignore link direction.
func SummariseRoutes(nodes []RouteNode) RouteStats {
	out := RouteStats{Cells: len(nodes), Levels: []RouteLevel{}, Groups: []RouteGroup{}}
	index := map[string]int{}
	for i, n := range nodes {
		index[n.Key] = i
	}
	parent := make([]int, len(nodes))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	levels := map[float64]int{}
	for i, n := range nodes {
		levels[math.Round(n.Pos[1]/RouteFloorM)*RouteFloorM]++
		if n.At.After(out.Newest) {
			out.Newest = n.At
		}
		for _, l := range n.Links {
			out.Links++
			if l.N > 0 {
				out.Walked++
			}
			if l.Fail > 0 {
				out.Failed++
			}
			if j, ok := index[l.To]; ok {
				parent[find(i)] = find(j)
			}
		}
	}
	for y, n := range levels {
		out.Levels = append(out.Levels, RouteLevel{Y: y, Cells: n})
	}
	sort.Slice(out.Levels, func(i, j int) bool { return out.Levels[i].Y < out.Levels[j].Y })
	groups := map[int]*RouteGroup{}
	var roots []int
	for i, n := range nodes {
		root := find(i)
		g := groups[root]
		if g == nil {
			g = &RouteGroup{MinY: n.Pos[1], MaxY: n.Pos[1]}
			groups[root] = g
			roots = append(roots, root)
		}
		g.Cells++
		for k := range g.Centre {
			g.Centre[k] += n.Pos[k]
		}
		g.MinY, g.MaxY = math.Min(g.MinY, n.Pos[1]), math.Max(g.MaxY, n.Pos[1])
	}
	for _, root := range roots {
		g := groups[root]
		for k := range g.Centre {
			g.Centre[k] = math.Round(g.Centre[k]/float64(g.Cells)*100) / 100
		}
		out.Groups = append(out.Groups, *g)
	}
	sort.Slice(out.Groups, func(i, j int) bool {
		a, b := out.Groups[i], out.Groups[j]
		if a.Cells != b.Cells {
			return a.Cells > b.Cells
		}
		return a.Centre[0] < b.Centre[0]
	})
	return out
}
