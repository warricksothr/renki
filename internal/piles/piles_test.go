package piles

import (
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

var (
	start = time.Unix(1000, 0)
	seen  = protocol.PerceptionConfig{Radius: 40, MaxPerKind: 24}
	hello = &protocol.Hello{}
)

func drop(id, prefab string, x, z float64, stack int) protocol.PerceivedResource {
	return protocol.PerceivedResource{ID: id, Kind: protocol.ResourceDrop, Prefab: prefab, Pos: protocol.Vec3{x, 0, z}, Stack: stack}
}

func frameOf(drops ...protocol.PerceivedResource) *protocol.Perception {
	return &protocol.Perception{Resources: drops}
}

func find(t *testing.T, s Snapshot, prefab string, cell [2]int) Pile {
	t.Helper()
	for _, p := range s.Piles(newest(s)) {
		if p.Prefab == prefab && p.Cell == cell {
			return p
		}
	}
	t.Fatalf("no %s pile in cell %v: %+v", prefab, cell, s.Piles(newest(s)))
	return Pile{}
}

// newest is the time of the newest frame, or zero.
func newest(s Snapshot) time.Time {
	if len(s.frames) == 0 {
		return time.Time{}
	}
	return s.frames[len(s.frames)-1].at
}

func TestGroupsDropsPerCellAndPrefab(t *testing.T) {
	var tr Tracker
	tr.Observe(hello, "w", protocol.Vec3{}, 0, seen, frameOf(
		drop("a", "Wood", 1, 1, 10),
		drop("b", "Wood", 3, 3, 5),
		drop("c", "Stone", 2, 2, 0), // a stack of 0 is still one item
		drop("d", "Wood", 5, 1, 7),  // the next cell along x
		drop("e", "Wood", 50, 0, 9), // beyond the perception radius
		protocol.PerceivedResource{ID: "f", Kind: protocol.ResourcePickable, Prefab: "Wood", Pos: protocol.Vec3{1, 0, 1}, Stack: 99},
	), start)
	s := tr.Snapshot(hello, "w")
	if got := len(s.Piles(start)); got != 3 {
		t.Fatalf("piles = %d, want 3: %+v", got, s.Piles(start))
	}
	wood := find(t, s, "Wood", [2]int{0, 0})
	if wood.Total != 15 || !wood.InView || wood.Pos[0] < 1 || wood.Pos[0] > 3 {
		t.Fatalf("wood = %+v", wood)
	}
	if stone := find(t, s, "Stone", [2]int{0, 0}); stone.Total != 1 {
		t.Fatalf("stone = %+v", stone)
	}
	if next := find(t, s, "Wood", [2]int{1, 0}); next.Total != 7 {
		t.Fatalf("next cell = %+v", next)
	}
}

func TestTrendOverFiveMinutes(t *testing.T) {
	var tr Tracker
	at := start
	see := func(drops ...protocol.PerceivedResource) Snapshot {
		at = at.Add(time.Second)
		tr.Observe(hello, "w", protocol.Vec3{}, 0, seen, frameOf(drops...), at)
		return tr.Snapshot(hello, "w")
	}
	see()
	if p := find(t, see(drop("a", "Wood", 1, 1, 5)), "Wood", [2]int{0, 0}); p.Trend != Rising {
		t.Fatalf("a pile that appeared is %s", p.Trend)
	}
	// Five minutes on, the empty frame has left the window.
	for range 300 {
		see(drop("a", "Wood", 1, 1, 5))
	}
	if p := find(t, see(drop("a", "Wood", 1, 1, 5)), "Wood", [2]int{0, 0}); p.Trend != Steady || p.Total != 5 {
		t.Fatalf("an unchanged pile is %+v", p)
	}
	if p := find(t, see(drop("a", "Wood", 1, 1, 8)), "Wood", [2]int{0, 0}); p.Trend != Rising {
		t.Fatalf("a growing pile is %s", p.Trend)
	}
	if p := find(t, see(drop("a", "Wood", 1, 1, 2)), "Wood", [2]int{0, 0}); p.Trend != Falling {
		t.Fatalf("a shrinking pile is %s", p.Trend)
	}
	if p := find(t, see(), "Wood", [2]int{0, 0}); p.Trend != Gone || p.Total != 0 || !p.InView {
		t.Fatalf("a picked-up pile is %+v", p)
	}
	// Walking away: the cell is out of sight, so it is unknown, not gone.
	at = at.Add(time.Second)
	tr.Observe(hello, "w", protocol.Vec3{100, 0, 0}, 0, seen, frameOf(), at)
	if p := find(t, tr.Snapshot(hello, "w"), "Wood", [2]int{0, 0}); p.Trend != Unknown || p.InView {
		t.Fatalf("an unseen pile is %+v", p)
	}
	// Five minutes later still, it is forgotten.
	for range 301 {
		see()
	}
	if got := see().Piles(at); len(got) != 0 {
		t.Fatalf("piles outlived the window: %+v", got)
	}
}

func TestFullListHidesWhatLiesBeyondIt(t *testing.T) {
	var tr Tracker
	small := protocol.PerceptionConfig{Radius: 40, MaxPerKind: 2}
	tr.Observe(hello, "w", protocol.Vec3{}, 0, small, frameOf(drop("a", "Wood", 30, 1, 5)), start)
	// Two drops fill the list, so ground beyond the farther one is unseen.
	tr.Observe(hello, "w", protocol.Vec3{}, 0, small, frameOf(drop("b", "Stone", 1, 1, 1), drop("c", "Stone", 6, 1, 1)), start.Add(time.Second))
	s := tr.Snapshot(hello, "w")
	if p := find(t, s, "Wood", [2]int{7, 0}); p.Trend != Unknown || p.Total != 5 {
		t.Fatalf("wood beyond a full list is %+v", p)
	}
	if n := s.Near(start.Add(time.Second), protocol.Vec3{30, 0, 1}, 4, nil); n.State != Unseen {
		t.Fatalf("near beyond a full list = %+v", n)
	}
}

func TestScopedToConnectionAndWorld(t *testing.T) {
	var tr Tracker
	tr.Observe(hello, "w", protocol.Vec3{}, 0, seen, frameOf(drop("a", "Wood", 1, 1, 5)), start)
	if got := tr.Snapshot(hello, "other").Piles(start.Add(time.Second)); len(got) != 0 {
		t.Fatalf("piles crossed worlds: %+v", got)
	}
	other := &protocol.Hello{}
	if got := tr.Snapshot(other, "w").Piles(start.Add(time.Second)); len(got) != 0 {
		t.Fatalf("piles crossed connections: %+v", got)
	}
	tr.Observe(other, "w", protocol.Vec3{}, 0, seen, frameOf(), start.Add(time.Second))
	if got := tr.Snapshot(other, "w").Piles(start.Add(time.Second)); len(got) != 0 {
		t.Fatalf("a new connection kept old piles: %+v", got)
	}
	if got := tr.Snapshot(hello, "w").Piles(start.Add(time.Second)); len(got) != 0 {
		t.Fatalf("the old connection still answers: %+v", got)
	}
}

func TestBoundedInSize(t *testing.T) {
	var tr Tracker
	many := make([]protocol.PerceivedResource, 0, 400)
	for i := range 400 {
		many = append(many, drop("d", "Wood", float64(i%20)*4-38, float64(i/20)*4-38, 1))
	}
	open := protocol.PerceptionConfig{Radius: 1000}
	// A fast stream: many more frames than the cap within the window.
	for i := range MaxFrames * 3 {
		tr.Observe(hello, "w", protocol.Vec3{}, 0, open, frameOf(many...), start.Add(time.Duration(i)*100*time.Millisecond))
	}
	tr.mu.Lock()
	frames := len(tr.frames)
	piles := 0
	for _, f := range tr.frames {
		piles = max(piles, len(f.piles))
	}
	last := tr.frames[len(tr.frames)-1]
	tr.mu.Unlock()
	if frames > MaxFrames || piles > MaxPilesPerFrame {
		t.Fatalf("frames = %d, piles per frame = %d", frames, piles)
	}
	if last.reach != 0 {
		t.Fatal("a frame cut short claimed to see all its ground")
	}
	// The same frame twice is one observation.
	p := frameOf(drop("a", "Wood", 1, 1, 5))
	tr.Observe(hello, "w", protocol.Vec3{}, 0, seen, p, start.Add(time.Hour))
	tr.Observe(hello, "w", protocol.Vec3{}, 0, seen, p, start.Add(time.Hour+time.Second))
	if got := len(tr.Snapshot(hello, "w").frames); got != 1 {
		t.Fatalf("frames after a repeat = %d", got)
	}
}

func TestNearAnOwner(t *testing.T) {
	var tr Tracker
	owner := protocol.Vec3{5, 0, 5}
	wood := func(prefab string) bool { return prefab == "Wood" }
	at := start
	see := func(bot protocol.Vec3, drops ...protocol.PerceivedResource) Snapshot {
		at = at.Add(time.Second)
		tr.Observe(hello, "w", bot, 0, seen, frameOf(drops...), at)
		return tr.Snapshot(hello, "w")
	}
	if n := (Snapshot{}).Near(at, owner, 8, wood); n.State != Unseen {
		t.Fatalf("nothing observed = %+v", n)
	}
	s := see(protocol.Vec3{}, drop("a", "Wood", 6, 6, 4), drop("b", "Wood", 7, 4, 3), drop("c", "Stone", 5, 5, 9))
	if n := s.Near(at, owner, 8, wood); n.State != Present || n.Total != 7 {
		t.Fatalf("pile near owner = %+v", n)
	}
	// Only stone is left, which is not eligible.
	s = see(protocol.Vec3{}, drop("c", "Stone", 5, 5, 9))
	gone := at
	if n := s.Near(at, owner, 8, wood); n.State != Absent || !n.Since.Equal(gone) {
		t.Fatalf("pile picked up = %+v, want absent since %v", n, gone)
	}
	for range 30 {
		s = see(protocol.Vec3{})
	}
	if n := s.Near(at, owner, 8, wood); n.State != Absent || !n.Since.Equal(gone) {
		t.Fatalf("still clear = %+v, want absent since %v", n, gone)
	}
	// A stale frame proves nothing about now.
	if n := s.Near(at.Add(Gap+time.Second), owner, 8, wood); n.State != Unseen {
		t.Fatalf("stale = %+v", n)
	}
	// The bot walks off: part of the circle lies beyond its sight. That is
	// unknown, never gone, even though the frame lists no drops at all.
	s = see(protocol.Vec3{40, 0, 0})
	if n := s.Near(at, owner, 8, wood); n.State != Unseen {
		t.Fatalf("circle partly out of sight = %+v", n)
	}
	// Back in sight: the clear run starts again, not from before the gap.
	s = see(protocol.Vec3{})
	back := at
	s = see(protocol.Vec3{})
	if n := s.Near(at, owner, 8, wood); n.State != Absent || !n.Since.Equal(back) {
		t.Fatalf("back in sight = %+v, want absent since %v", n, back)
	}
	// A gap in perception breaks the run too.
	at = at.Add(Gap + time.Second)
	s = see(protocol.Vec3{})
	if n := s.Near(at, owner, 8, wood); n.State != Absent || !n.Since.Equal(at) {
		t.Fatalf("after a gap = %+v, want absent since %v", n, at)
	}
	// Without a perception radius, no ground is known to be clear.
	var blind Tracker
	blind.Observe(hello, "w", protocol.Vec3{}, 0, protocol.PerceptionConfig{}, frameOf(), at)
	if n := blind.Snapshot(hello, "w").Near(at, owner, 8, wood); n.State != Unseen {
		t.Fatalf("no radius = %+v", n)
	}
}

func TestDriftAndStaleFramesSeeLess(t *testing.T) {
	var tr Tracker
	owner := protocol.Vec3{5, 0, 5}
	// Owner circle 7.07 + 8 from the bot fits a 40 m reach, but not one
	// shrunk by 30 m of possible movement since the scan.
	tr.Observe(hello, "w", protocol.Vec3{}, 30, seen, frameOf(), start)
	if n := tr.Snapshot(hello, "w").Near(start, owner, 8, nil); n.State != Unseen {
		t.Fatalf("drifted frame = %+v", n)
	}
	// A full list ending 30 m from the paired position: with 5 m of drift
	// the scan's own list may end at 25 m, and a circle around the paired
	// position is safe only to 20 m. 7.07 + 13 = 20.07 is too far.
	full := protocol.PerceptionConfig{Radius: 40, MaxPerKind: 1}
	tr.Observe(hello, "w", protocol.Vec3{}, 5, full, frameOf(drop("f", "Stone", 30, 0, 1)), start.Add(time.Second/2))
	if n := tr.Snapshot(hello, "w").Near(start.Add(time.Second/2), owner, 13, nil); n.State != Unseen {
		t.Fatalf("full list with drift = %+v", n)
	}
	if n := tr.Snapshot(hello, "w").Near(start.Add(time.Second/2), owner, 12.9, nil); n.State != Absent {
		t.Fatalf("full list with drift, inside = %+v", n)
	}
	tr.Observe(hello, "w", protocol.Vec3{}, 0, seen, frameOf(drop("a", "Wood", 1, 1, 5)), start.Add(time.Second))
	s := tr.Snapshot(hello, "w")
	if p := find(t, s, "Wood", [2]int{0, 0}); !p.InView {
		t.Fatalf("fresh pile = %+v", p)
	}
	// Perception stalls: the last frame is no longer what is there now.
	for _, p := range s.Piles(start.Add(time.Second + Gap + time.Second)) {
		if p.InView || p.Trend != Unknown {
			t.Fatalf("stale pile = %+v", p)
		}
	}
}
