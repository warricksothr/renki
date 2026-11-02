// Package piles sums the drops the bot can see per 4 m cell and prefab, and
// follows each total over the last five minutes. It is a live signal kept in
// memory only, for one connection and world at a time, and never a durable
// record: a pile nobody is looking at is unknown, not gone.
package piles

import (
	"math"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

const (
	// Cell is the side of the square, in metres on the ground, that drops
	// are summed over.
	Cell = 4.0
	// Window is how far back a trend looks.
	Window = 5 * time.Minute
	// MaxFrames bounds the history. Perception runs at 1 Hz by default, so
	// this is ten minutes of frames, more than Window ever keeps.
	MaxFrames = 600
	// MaxPilesPerFrame bounds one frame. Perception lists at most
	// maxPerKind drops (24 by default), so only a misbehaving plugin
	// reaches it, and a frame cut short sees nothing as gone.
	MaxPilesPerFrame = 128
	// Gap is the longest pause between two frames that still counts as
	// watching without a break.
	Gap = 5 * time.Second
	// maxStack bounds one drop's stack so a sum cannot overflow.
	maxStack = 1_000_000
)

// halfDiagonal is the reach from a cell's centre to its corners.
var halfDiagonal = Cell * math.Sqrt2 / 2

// Trends of a pile over Window.
const (
	Rising  = "rising"
	Steady  = "steady"
	Falling = "falling"
	Gone    = "gone"
	// Unknown is a pile whose cell is not all in sight now.
	Unknown = "unknown"
)

// States of the ground near a point, answered by Snapshot.Near.
const (
	Present = "present"
	Absent  = "absent"
	// Unseen: some of the circle is outside what perception reports.
	Unseen = "unknown"
)

type key struct {
	X, Z   int
	Prefab string
}

type entry struct {
	key
	pos   protocol.Vec3
	total int
}

type frame struct {
	at  time.Time
	bot protocol.Vec3
	// reach is the distance from bot within which every drop is listed.
	// Zero means no ground is known to be empty.
	reach float64
	piles []entry
}

// Tracker turns perception frames into piles. The zero value is ready.
type Tracker struct {
	mu     sync.Mutex
	conn   *protocol.Hello
	world  string
	last   *protocol.Perception
	frames []frame
}

// Observe records one perception frame seen at `at`, with the bounds the
// welcome gave perception. Perception does not say where it was scanned
// from, so bot is the nearest known position and drift how far from the
// scan the bot may have been there; the reach shrinks by drift, so ground
// the scan may not have covered is never taken as seen. A frame already
// observed is ignored, and a new connection or world starts over.
func (t *Tracker) Observe(conn *protocol.Hello, world string, bot protocol.Vec3, drift float64, seen protocol.PerceptionConfig, p *protocol.Perception, at time.Time) {
	if conn == nil || world == "" || p == nil || at.IsZero() || !finite(bot) || !finiteF(drift) || drift < 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if conn != t.conn || world != t.world {
		t.conn, t.world, t.last, t.frames = conn, world, nil, nil
	}
	if p == t.last {
		return
	}
	t.last = p
	if n := len(t.frames); n > 0 && at.Before(t.frames[n-1].at) {
		return
	}
	f := group(bot, seen, p, at)
	f.reach = max(f.reach-drift, 0)
	t.frames = append(t.frames, f)
	cut := 0
	for cut < len(t.frames) && (at.Sub(t.frames[cut].at) > Window || len(t.frames)-cut > MaxFrames) {
		cut++
	}
	if cut > 0 {
		// Snapshots share the old backing array, so never shift in place.
		t.frames = slices.Clone(t.frames[cut:])
	}
}

func group(bot protocol.Vec3, seen protocol.PerceptionConfig, p *protocol.Perception, at time.Time) frame {
	f := frame{at: at, bot: bot}
	if seen.Radius > 0 && finiteF(seen.Radius) {
		f.reach = seen.Radius
	}
	type sum struct {
		weighted protocol.Vec3
		total    int
	}
	sums := map[key]*sum{}
	drops, farthest := 0, 0.0
	for _, r := range p.Resources {
		if r.Kind != protocol.ResourceDrop {
			continue
		}
		drops++
		if r.Prefab == "" || len(r.Prefab) > 100 || !finite(r.Pos) {
			continue
		}
		d := distance(bot, r.Pos)
		farthest = max(farthest, d)
		if f.reach > 0 && d > f.reach {
			continue
		}
		k := key{int(math.Floor(r.Pos[0] / Cell)), int(math.Floor(r.Pos[2] / Cell)), r.Prefab}
		s := sums[k]
		if s == nil {
			s = &sum{}
			sums[k] = s
		}
		n := min(max(r.Stack, 1), maxStack)
		for i := range s.weighted {
			s.weighted[i] += r.Pos[i] * float64(n)
		}
		s.total += n
	}
	// Perception lists only the nearest drops. A full list says nothing
	// about ground beyond the farthest one it lists.
	if seen.MaxPerKind > 0 && drops >= seen.MaxPerKind {
		f.reach = min(f.reach, farthest)
	}
	for k, s := range sums {
		var pos protocol.Vec3
		for i := range pos {
			pos[i] = s.weighted[i] / float64(s.total)
		}
		f.piles = append(f.piles, entry{key: k, pos: pos, total: s.total})
	}
	sort.Slice(f.piles, func(i, j int) bool {
		a, b := f.piles[i], f.piles[j]
		if da, db := distance(bot, a.pos), distance(bot, b.pos); da != db {
			return da < db
		}
		return less(a.key, b.key)
	})
	if len(f.piles) > MaxPilesPerFrame {
		f.piles = f.piles[:MaxPilesPerFrame]
		f.reach = 0
	}
	return f
}

// Snapshot returns the piles for conn and world, or none when the tracker
// is following another connection or world.
func (t *Tracker) Snapshot(conn *protocol.Hello, world string) Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	if conn == nil || conn != t.conn || world != t.world {
		return Snapshot{}
	}
	// Frames are never changed once appended, so sharing them is safe.
	return Snapshot{frames: t.frames[:len(t.frames):len(t.frames)]}
}

// Snapshot is an immutable view of the tracker at one moment.
type Snapshot struct {
	frames []frame
}

// Pile is the drops of one prefab in one cell.
type Pile struct {
	Prefab string        `json:"prefab"`
	Cell   [2]int        `json:"cell"`
	Pos    protocol.Vec3 `json:"pos"`
	// Total is the stack total now, or when the cell was last all in sight
	// if InView is false.
	Total  int    `json:"total"`
	Trend  string `json:"trend"`
	InView bool   `json:"inView"`
}

// Piles lists every pile seen within Window, largest first. When the newest
// frame is more than Gap old at now, nothing is in view.
func (s Snapshot) Piles(now time.Time) []Pile {
	if len(s.frames) == 0 {
		return nil
	}
	fresh := s.fresh(now)
	where := map[key]protocol.Vec3{}
	for _, f := range s.frames {
		for _, e := range f.piles {
			where[e.key] = e.pos
		}
	}
	out := make([]Pile, 0, len(where))
	for k, pos := range where {
		p := Pile{Prefab: k.Prefab, Cell: [2]int{k.X, k.Z}, Pos: pos, Trend: Unknown}
		centre := protocol.Vec3{(float64(k.X) + .5) * Cell, pos[1], (float64(k.Z) + .5) * Cell}
		baseline, current := -1, -1
		for i, f := range s.frames {
			if !covers(f, centre, halfDiagonal) {
				continue
			}
			n := totalOf(f, k)
			if baseline < 0 {
				baseline = n
			}
			current = n
			p.InView = fresh && i == len(s.frames)-1
		}
		if current < 0 {
			// Never all in sight: report the newest partial count.
			for _, f := range s.frames {
				if n := totalOf(f, k); n > 0 {
					p.Total = n
				}
			}
		} else {
			p.Total = current
		}
		switch {
		case !p.InView:
		case current == 0:
			p.Trend = Gone
		case current > baseline:
			p.Trend = Rising
		case current < baseline:
			p.Trend = Falling
		default:
			p.Trend = Steady
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Total != out[j].Total {
			return out[i].Total > out[j].Total
		}
		return less(key{out[i].Cell[0], out[i].Cell[1], out[i].Prefab}, key{out[j].Cell[0], out[j].Cell[1], out[j].Prefab})
	})
	return out
}

// Nearby answers whether an eligible pile lies within a circle.
type Nearby struct {
	// State is Present, Absent or Unseen.
	State string `json:"state"`
	// Total is the eligible stack total in the circle when Present.
	Total int `json:"total,omitempty"`
	// Since is, when Absent, the first frame of an unbroken run of frames
	// that saw all of the circle with no eligible pile in it. The run is
	// limited to Window, so Since is never older than that.
	Since time.Time `json:"since,omitzero"`
}

// Near asks whether a pile whose prefab is eligible (every prefab when
// eligible is nil) lies within r metres, on the ground, of pos. Any part of
// the circle outside what the newest frame reports, or a newest frame older
// than Gap at now, answers Unseen: that ground is not known to be clear.
func (s Snapshot) Near(now time.Time, pos protocol.Vec3, r float64, eligible func(prefab string) bool) Nearby {
	n := len(s.frames)
	if !s.fresh(now) || !finite(pos) || !finiteF(r) || r < 0 {
		return Nearby{State: Unseen}
	}
	inCircle := func(f frame) int {
		total := 0
		for _, e := range f.piles {
			if (eligible == nil || eligible(e.Prefab)) && e.pos.FlatDistance(pos) <= r {
				total += e.total
			}
		}
		return total
	}
	latest := s.frames[n-1]
	if !covers(latest, pos, r) {
		return Nearby{State: Unseen}
	}
	if total := inCircle(latest); total > 0 {
		return Nearby{State: Present, Total: total}
	}
	since := latest.at
	for i := n - 2; i >= 0; i-- {
		f := s.frames[i]
		if s.frames[i+1].at.Sub(f.at) > Gap || !covers(f, pos, r) || inCircle(f) > 0 {
			break
		}
		since = f.at
	}
	return Nearby{State: Absent, Since: since}
}

// fresh reports whether the newest frame is no more than Gap old at now.
func (s Snapshot) fresh(now time.Time) bool {
	n := len(s.frames)
	return n > 0 && !s.frames[n-1].at.After(now) && now.Sub(s.frames[n-1].at) <= Gap
}

// covers reports whether every drop within r of pos would be listed in f.
func covers(f frame, pos protocol.Vec3, r float64) bool {
	return f.reach > 0 && distance(f.bot, pos)+r <= f.reach
}

func totalOf(f frame, k key) int {
	for _, e := range f.piles {
		if e.key == k {
			return e.total
		}
	}
	return 0
}

func less(a, b key) bool {
	if a.Prefab != b.Prefab {
		return a.Prefab < b.Prefab
	}
	if a.X != b.X {
		return a.X < b.X
	}
	return a.Z < b.Z
}

func distance(a, b protocol.Vec3) float64 {
	return math.Sqrt((a[0]-b[0])*(a[0]-b[0]) + (a[1]-b[1])*(a[1]-b[1]) + (a[2]-b[2])*(a[2]-b[2]))
}

func finite(v protocol.Vec3) bool { return finiteF(v[0]) && finiteF(v[1]) && finiteF(v[2]) }

func finiteF(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }
