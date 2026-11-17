package home

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func routeStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "memory.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return s, path
}

func walk(at time.Time, points ...protocol.Vec3) []RouteStep {
	var steps []RouteStep
	for i := 1; i < len(points); i++ {
		steps = append(steps, RouteStep{From: points[i-1], To: points[i], At: at})
	}
	return steps
}

func links(nodes []RouteNode) map[string]int {
	out := map[string]int{}
	for _, n := range nodes {
		for _, l := range n.Links {
			out[n.Key+">"+l.To] = l.N
		}
	}
	return out
}

func TestRouteCellsKeepFloorsApart(t *testing.T) {
	// The new home's stair foot and stair top, 2026-11-04.
	foot, top := RouteCellOf(protocol.Vec3{-189.51, 31.26, -471.72}), RouteCellOf(protocol.Vec3{-193.01, 32.97, -471.11})
	if foot[1] == top[1] {
		t.Fatal("stair foot and top share a height level", foot, top)
	}
	// The same spot on the floor above is another cell.
	if RouteCellOf(protocol.Vec3{1.2, 30.97, 5.5}) == RouteCellOf(protocol.Vec3{1.2, 32.97, 5.5}) {
		t.Fatal("two floors share a cell")
	}
	if RouteCellOf(protocol.Vec3{-0.2, 0, 0.9}) != (RouteCell{-1, 0, 0}) {
		t.Fatal(RouteCellOf(protocol.Vec3{-0.2, 0, 0.9}))
	}
}

func TestRememberRouteLinksLevelGroundBothWaysAndADropOneWay(t *testing.T) {
	s, path := routeStore(t)
	now := time.Now()
	level := walk(now, protocol.Vec3{0.5, 30, 0.5}, protocol.Vec3{1.5, 30.2, 0.5}, protocol.Vec3{2.5, 30.6, 0.5})
	drop := walk(now, protocol.Vec3{2.5, 30.6, 0.5}, protocol.Vec3{3.5, 28.5, 0.5})
	jump := walk(now, protocol.Vec3{3.5, 28.5, 0.5}, protocol.Vec3{40.5, 28.5, 0.5})
	if err := s.RememberRoute("w", append(append(level, drop...), jump...), now); err != nil {
		t.Fatal(err)
	}
	got := links(s.Routes("w"))
	a, b, c, d := "0,60,0", "1,60,0", "2,61,0", "3,57,0"
	if got[a+">"+b] != 1 || got[b+">"+c] != 1 || got[c+">"+d] != 1 {
		t.Fatal("walked links missing", got)
	}
	if n, ok := got[b+">"+a]; !ok || n != 0 {
		t.Fatal("level ground has no inferred way back", got)
	}
	if _, ok := got[d+">"+c]; ok {
		t.Fatal("a 2 m drop was recorded as climbable", got)
	}
	// A teleport marks both ends walked and joins nothing.
	if len(s.Routes("w")) != 5 || len(got) != 5 {
		t.Fatal(len(s.Routes("w")), got)
	}
	// Walking it again counts, and survives a restart.
	if err := s.RememberRoute("w", level, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if again := links(reopened.Routes("w")); again[a+">"+b] != 2 || again[b+">"+a] != 0 {
		t.Fatal(again)
	}
	if len(reopened.Routes("other")) != 0 {
		t.Fatal("worlds share a graph")
	}
}

func TestRememberRouteKeepsTheLastExactPositionAndRejectsBadSteps(t *testing.T) {
	s, _ := routeStore(t)
	now := time.Now()
	_ = s.RememberRoute("w", walk(now, protocol.Vec3{0.1, 30, 0.1}, protocol.Vec3{1.5, 30, 0.5}), now)
	_ = s.RememberRoute("w", walk(now.Add(time.Second), protocol.Vec3{0.9, 30, 0.8}, protocol.Vec3{1.5, 30, 0.5}), now)
	_ = s.RememberRoute("w", walk(now.Add(-time.Hour), protocol.Vec3{0.3, 30, 0.3}, protocol.Vec3{1.5, 30, 0.5}), now)
	if n := s.Routes("w")[0]; n.Key != "0,60,0" || n.Pos != (protocol.Vec3{0.9, 30, 0.8}) {
		t.Fatalf("%+v", n)
	}
	nan := 0.0
	for _, bad := range [][]RouteStep{
		{{From: protocol.Vec3{0, 0, 0}, To: protocol.Vec3{1, 0, 0}}},
		{{From: protocol.Vec3{nan / nan, 0, 0}, To: protocol.Vec3{1, 0, 0}, At: now}},
		{{From: protocol.Vec3{0, 0, 0}, To: protocol.Vec3{1e9, 0, 0}, At: now}},
	} {
		if s.RememberRoute("w", bad, now) == nil {
			t.Fatal("accepted", bad)
		}
	}
	if s.RememberRoute("", walk(now, protocol.Vec3{}, protocol.Vec3{1, 0, 0}), now) == nil {
		t.Fatal("accepted a step with no world")
	}
}

func TestRouteMemoryForgetsOldGroundAndItsLinks(t *testing.T) {
	s, _ := routeStore(t)
	now := time.Now()
	old := now.Add(-KeepRoutes - time.Hour)
	_ = s.RememberRoute("w", walk(old, protocol.Vec3{0.5, 30, 0.5}, protocol.Vec3{1.5, 30, 0.5}), old)
	_ = s.RememberRoute("w", walk(now, protocol.Vec3{1.5, 30, 0.5}, protocol.Vec3{2.5, 30, 0.5}), now)
	nodes := s.Routes("w")
	if len(nodes) != 2 || nodes[0].Key != "1,60,0" {
		t.Fatalf("%+v", nodes)
	}
	for _, n := range nodes {
		for _, l := range n.Links {
			if l.To == "0,60,0" {
				t.Fatal("a link to forgotten ground remains")
			}
		}
	}
}

func TestMarkRouteFailedCountsOnlyAKnownLink(t *testing.T) {
	s, _ := routeStore(t)
	now := time.Now()
	a, b := protocol.Vec3{0.5, 30, 0.5}, protocol.Vec3{1.5, 30, 0.5}
	_ = s.RememberRoute("w", walk(now, a, b), now)
	if known, err := s.MarkRouteFailed("w", a, b, now); !known || err != nil {
		t.Fatal(known, err)
	}
	if known, _ := s.MarkRouteFailed("w", a, protocol.Vec3{9.5, 30, 0.5}, now); known {
		t.Fatal("marked a link that was never walked")
	}
	if known, _ := s.MarkRouteFailed("other", a, b, now); known {
		t.Fatal("marked a link in another world")
	}
	stats := SummariseRoutes(s.Routes("w"))
	if stats.Failed != 1 || stats.Links != 2 || stats.Walked != 1 {
		t.Fatalf("%+v", stats)
	}
}

func TestSummariseRoutesShowsLevelsAndSeparateGroups(t *testing.T) {
	s, _ := routeStore(t)
	now := time.Now()
	// A ground floor walk, and an upper floor nobody has walked up to.
	ground := walk(now, protocol.Vec3{0.5, 31, 0.5}, protocol.Vec3{1.5, 31, 0.5}, protocol.Vec3{2.5, 31, 0.5})
	upper := walk(now.Add(time.Second), protocol.Vec3{0.5, 33, 0.5}, protocol.Vec3{1.5, 33, 0.5})
	_ = s.RememberRoute("w", append(ground, upper...), now)
	stats := SummariseRoutes(s.Routes("w"))
	if stats.Cells != 5 || len(stats.Levels) != 2 || stats.Levels[0] != (RouteLevel{Y: 31, Cells: 3}) || stats.Levels[1].Y != 33 {
		t.Fatalf("%+v", stats)
	}
	if len(stats.Groups) != 2 || stats.Groups[0].Cells != 3 || stats.Groups[1].MinY != 33 || stats.Groups[0].Centre != (protocol.Vec3{1.5, 31, 0.5}) {
		t.Fatalf("%+v", stats.Groups)
	}
	if !stats.Newest.Equal(now.Add(time.Second)) {
		t.Fatal(stats.Newest)
	}
	// Walking the stairs joins them.
	stairs := walk(now, protocol.Vec3{2.5, 31, 0.5}, protocol.Vec3{2.5, 31.6, 1.5}, protocol.Vec3{2.5, 32.3, 2.5}, protocol.Vec3{1.5, 33, 0.5})
	_ = s.RememberRoute("w", stairs, now)
	if joined := SummariseRoutes(s.Routes("w")); len(joined.Groups) != 1 || joined.Groups[0].Cells != 7 {
		t.Fatalf("%+v", joined.Groups)
	}
	if empty := SummariseRoutes(nil); empty.Cells != 0 || empty.Levels == nil || empty.Groups == nil {
		t.Fatalf("%+v", empty)
	}
}
