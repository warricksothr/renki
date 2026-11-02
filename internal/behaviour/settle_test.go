package behaviour

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

var testArea = SettleArea{Radius: 6, MaxRadius: 10, Slack: 2, Delay: 5 * time.Second}

// walk feeds one sighting a second from t0 and returns the tracker.
func walk(s *Settle, t0 time.Time, positions ...protocol.Vec3) time.Time {
	at := t0
	for _, p := range positions {
		s.Observe(p, at, testArea)
		at = at.Add(time.Second)
	}
	return at.Add(-time.Second)
}

func still(x float64, n int) []protocol.Vec3 {
	out := make([]protocol.Vec3, n)
	for i := range out {
		out[i] = protocol.Vec3{x, 0, 0}
	}
	return out
}

func TestSettleGrowsReachToTheCapWhileStandingStill(t *testing.T) {
	p := lootDefaults(LootParams{})
	var s Settle
	t0 := time.Unix(1000, 0)
	last := walk(&s, t0, still(0, 300)...)
	for _, c := range []struct {
		at   time.Duration
		want float64
	}{{0, 8}, {5 * time.Second, 8}, {65 * time.Second, 19}, {125 * time.Second, 30}, {299 * time.Second, 30}} {
		if got := LootReach(s.Settled(t0.Add(c.at)), p); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("after %v reach %v, want %v", c.at, got, c.want)
		}
	}
	if s.Settled(last) != 299*time.Second || s.Radius != 6 {
		t.Fatal(s.Settled(last), s.Radius)
	}
}

func TestSettleKeepsItsAreaWhileThePlayerDrifts(t *testing.T) {
	var s Settle
	t0 := time.Unix(1000, 0)
	// Pacing between -4 and 4 m keeps the area and its centre near 0.
	pacing := []protocol.Vec3{{}}
	for i := range 60 {
		pacing = append(pacing, protocol.Vec3{float64(i%9 - 4), 0, float64(i%5 - 2)})
	}
	last := walk(&s, t0, pacing...)
	if s.Since != t0 || distance(s.Centre, protocol.Vec3{}) > 1 || s.Radius != 6 {
		t.Fatalf("since %v centre %v radius %v", s.Since, s.Centre, s.Radius)
	}
	// Moving the work over by a few metres draws the centre along.
	last = walk(&s, last.Add(time.Second), still(5, 120)...)
	if s.Since != t0 || s.Centre[0] < 3 {
		t.Fatalf("since %v centre %v", s.Since, s.Centre)
	}
}

func TestSettleWidensForAStepOutAndResetsOnlyOnAFullExit(t *testing.T) {
	var s Settle
	t0 := time.Unix(1000, 0)
	last := walk(&s, t0, still(0, 30)...)
	// 7 m is past the 6 m edge but inside the 2 m slack: kept, and widened.
	last = walk(&s, last.Add(time.Second), protocol.Vec3{7, 0, 0})
	if s.Since != t0 || s.Radius < 6.9 || s.Radius > 7 {
		t.Fatalf("a step out: since %v radius %v", s.Since, s.Radius)
	}
	// Widening stops at the maximum.
	last = walk(&s, last.Add(time.Second), protocol.Vec3{-8.5, 0, 0}, protocol.Vec3{0, 0, 10.5})
	if s.Since != t0 || s.Radius != 10 {
		t.Fatalf("widening: since %v radius %v", s.Since, s.Radius)
	}
	// Beyond the edge and its slack is a full exit: a fresh area opens there.
	exit := last.Add(time.Second)
	walk(&s, exit, protocol.Vec3{0, 0, 30})
	if s.Since != exit || s.Centre != (protocol.Vec3{0, 0, 30}) || s.Radius != 6 || s.Settled(exit) != 0 {
		t.Fatalf("exit: %+v", s)
	}
}

func TestSettleGivesAWalkerNoReach(t *testing.T) {
	p := lootDefaults(LootParams{})
	var s Settle
	t0 := time.Unix(1000, 0)
	// A jog at 4 m/s, sighted twice a second: each area is left before the
	// delay, so it neither widens nor gives any reach.
	for i := range 240 {
		at := t0.Add(time.Duration(i) * 500 * time.Millisecond)
		s.Observe(protocol.Vec3{float64(i) * 2, 0, 0}, at, testArea)
		if r := LootReach(s.Settled(at), p); r != p.BaseRadius || s.Radius != testArea.Radius {
			t.Fatalf("reach %v radius %v at step %d", r, s.Radius, i)
		}
	}
}

func TestSettleForgetsAPlayerUnseenForItsGap(t *testing.T) {
	var s Settle
	t0 := time.Unix(1000, 0)
	last := walk(&s, t0, still(0, 60)...)
	if s.Settled(last.Add(SettleGap+time.Second)) != 0 {
		t.Fatal("an unseen player is not settled")
	}
	back := last.Add(SettleGap + time.Second)
	s.Observe(protocol.Vec3{}, back, testArea)
	if s.Since != back {
		t.Fatal("back after the gap opens a fresh area", s.Since)
	}
}

func TestLootReachesAFarDropOnlyOnceTheAnchorHasSettled(t *testing.T) {
	l := &Loot{}
	// Nimah stands at x=10; the wood lies 20 m away.
	v := lootView(1, target("wood", protocol.ResourceDrop, "Wood", 30))
	l.Observe(v)
	if l.Ready(json.RawMessage(`{}`), v) {
		t.Fatal("a drop 20 m away is out of reach on arrival")
	}
	for i := 2; i <= 80; i++ {
		v.Now = v.Now.Add(time.Second)
		v.Perception.Seq = uint64(i)
		l.Observe(v)
		// The reach is base+22*(settled-5)/120; 20 m needs 70 s settled.
		if got, want := l.Ready(json.RawMessage(`{}`), v), i >= 72; got != want {
			t.Fatalf("after %d s ready %v", i-1, got)
		}
	}
	o := nextTask(t, lootRunner(t, l, LootParams{}), v, nil, protocol.TaskPickup)
	if o.Args.(*protocol.PickupArgs).Entity != "wood" {
		t.Fatal(o.Args)
	}
	// Walking off resets the reach: from x=50 the wood is again 20 m away,
	// inside the reach just grown but outside the base.
	v.Now = v.Now.Add(time.Second)
	v.Perception.Seq++
	v.Perception.Characters[0].Pos = protocol.Vec3{50, 0, 0}
	l.Observe(v)
	if l.Ready(json.RawMessage(`{}`), v) || l.reach("Nimah", lootDefaults(LootParams{}), v.Now) != 8 {
		t.Fatal("a full exit shrinks the reach back to its base")
	}
}

func TestLootTracksEachAreaShapeAnInstanceAsksFor(t *testing.T) {
	l := &Loot{}
	v := lootView(1)
	l.Observe(v)
	wide := lootDefaults(LootParams{SettleRadius: 3, SettleMaxRadius: 4})
	if r := l.reach("Nimah", wide, v.Now); r != 8 {
		t.Fatal(r)
	}
	for i := 2; i <= 70; i++ {
		v.Now = v.Now.Add(time.Second)
		v.Perception.Seq = uint64(i)
		l.Observe(v)
	}
	// The default shape has tracked Nimah from the first frame; the other
	// from the frame after it was asked for.
	if d, w := l.reach("Nimah", lootDefaults(LootParams{}), v.Now), l.reach("Nimah", wide, v.Now); d <= w || w <= 8 {
		t.Fatal(d, w)
	}
}
