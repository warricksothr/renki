package route

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

// walked builds a graph the way the recorder does, through the store.
func walked(t *testing.T, walks ...[]protocol.Vec3) (*home.Store, []home.RouteNode) {
	t.Helper()
	s, err := home.Open(filepath.Join(t.TempDir(), "memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, points := range walks {
		var steps []home.RouteStep
		for i := 1; i < len(points); i++ {
			steps = append(steps, home.RouteStep{From: points[i-1], To: points[i], At: now})
		}
		if err = s.RememberRoute("w", steps, now); err != nil {
			t.Fatal(err)
		}
	}
	return s, s.Routes("w")
}

func line(from, to protocol.Vec3, n int) []protocol.Vec3 {
	var out []protocol.Vec3
	for i := 0; i <= n; i++ {
		f := float64(i) / float64(n)
		out = append(out, protocol.Vec3{from[0] + f*(to[0]-from[0]), from[1] + f*(to[1]-from[1]), from[2] + f*(to[2]-from[2])})
	}
	return out
}

// A house: a ground floor corridor, stairs at its end, and an upper floor
// that doubles back above the corridor.
func house() [][]protocol.Vec3 {
	ground := line(protocol.Vec3{0.5, 31, 0.5}, protocol.Vec3{12.5, 31, 0.5}, 12)
	stairs := line(protocol.Vec3{12.5, 31, 0.5}, protocol.Vec3{12.5, 33, 4.5}, 4)
	upper := line(protocol.Vec3{12.5, 33, 4.5}, protocol.Vec3{0.5, 33, 4.5}, 12)
	return [][]protocol.Vec3{ground, stairs, upper}
}

func TestPlanGoesRoundByTheStairsNotStraightUp(t *testing.T) {
	_, nodes := walked(t, house()...)
	// The target is 4.5 m away through the ceiling; the walked way is 30 m.
	p := Find(nodes, protocol.Vec3{0.6, 31, 0.6}, protocol.Vec3{0.6, 33, 4.4})
	if !p.Found || p.Reason != ReasonNone || p.LengthM < 25 || p.LengthM > 34 {
		t.Fatalf("%+v", p)
	}
	// Corridor, stairs, upper floor: the stair foot and top end legs.
	var foot, top bool
	for _, leg := range p.Legs {
		if leg.LengthM > MaxLegM+1e-9 {
			t.Fatal("a leg is longer than the limit", leg)
		}
		foot = foot || leg.To == (protocol.Vec3{12.5, 31, 0.5})
		top = top || leg.To == (protocol.Vec3{12.5, 33, 4.5})
	}
	if !foot || !top {
		t.Fatalf("stair foot %v or top %v is not a leg end: %+v", foot, top, p.Legs)
	}
	var climb float64
	for _, leg := range p.Legs {
		climb += leg.Climb
	}
	if climb < 1.9 || climb > 2.1 {
		t.Fatal("legs do not add up to the floor above", climb)
	}
	if !strings.Contains(p.Explain, "legs") || p.Start.Cell == "" || p.Target.Cell == "" {
		t.Fatalf("%+v", p)
	}
}

func TestAStraightCorridorIsFewLegsAndTheWayBackIsInferred(t *testing.T) {
	_, nodes := walked(t, line(protocol.Vec3{0.5, 31, 0.5}, protocol.Vec3{8.5, 31, 0.5}, 8))
	out := Find(nodes, protocol.Vec3{0.5, 31, 0.5}, protocol.Vec3{8.5, 31, 0.5})
	if !out.Found || len(out.Legs) != 1 || out.Legs[0].Inferred != 0 || out.Legs[0].Cells != 8 {
		t.Fatalf("%+v", out)
	}
	back := Find(nodes, protocol.Vec3{8.5, 31, 0.5}, protocol.Vec3{0.5, 31, 0.5})
	if !back.Found || len(back.Legs) != 1 || back.Legs[0].Inferred != 8 {
		t.Fatalf("%+v", back)
	}
}

func TestADropIsNeverPlannedAsAClimb(t *testing.T) {
	ledge := line(protocol.Vec3{0.5, 35, 0.5}, protocol.Vec3{3.5, 35, 0.5}, 3)
	fall := []protocol.Vec3{{3.5, 35, 0.5}, {4.5, 32.6, 0.5}}
	below := line(protocol.Vec3{4.5, 32.6, 0.5}, protocol.Vec3{7.5, 32.6, 0.5}, 3)
	_, nodes := walked(t, ledge, fall, below)
	if down := Find(nodes, protocol.Vec3{0.5, 35, 0.5}, protocol.Vec3{7.5, 32.6, 0.5}); !down.Found {
		t.Fatalf("%+v", down)
	}
	up := Find(nodes, protocol.Vec3{7.5, 32.6, 0.5}, protocol.Vec3{0.5, 35, 0.5})
	if up.Found || up.Reason != ReasonDisconnected || up.Closest == nil || *up.Closest != (protocol.Vec3{4.5, 32.6, 0.5}) {
		t.Fatalf("%+v", up)
	}
	if up.GapM < 3 || !strings.Contains(up.Explain, "not joined") {
		t.Fatalf("%+v", up)
	}
}

func TestPlanExplainsEveryWayThereIsNoRoute(t *testing.T) {
	if p := Find(nil, protocol.Vec3{}, protocol.Vec3{1, 0, 0}); p.Found || p.Reason != ReasonEmpty || p.Legs == nil {
		t.Fatalf("%+v", p)
	}
	_, nodes := walked(t, house()...)
	// Standing on the upper floor's height but far from anything walked.
	off := Find(nodes, protocol.Vec3{40, 31, 40}, protocol.Vec3{0.5, 31, 0.5})
	if off.Found || off.Reason != ReasonStartOff || off.Start.Cell != "" || off.Start.CellPos == nil || off.Start.DistanceM < 30 {
		t.Fatalf("%+v", off)
	}
	// Directly under the upper floor is not on it: the floors stay apart.
	under := Find(nodes, protocol.Vec3{0.5, 31, 0.5}, protocol.Vec3{6.5, 36, 4.5})
	if under.Found || under.Reason != ReasonTargetOff || !strings.Contains(under.Explain, "Walk there once") {
		t.Fatalf("%+v", under)
	}
	// Two rooms never walked between.
	_, apart := walked(t, line(protocol.Vec3{0.5, 31, 0.5}, protocol.Vec3{4.5, 31, 0.5}, 4), line(protocol.Vec3{20.5, 31, 0.5}, protocol.Vec3{24.5, 31, 0.5}, 4))
	gap := Find(apart, protocol.Vec3{0.5, 31, 0.5}, protocol.Vec3{24.5, 31, 0.5})
	if gap.Found || gap.Reason != ReasonDisconnected || *gap.Closest != (protocol.Vec3{4.5, 31, 0.5}) || gap.GapM != 20 {
		t.Fatalf("%+v", gap)
	}
}

func TestAFailedLinkIsAvoidedUntilItIsWalkedAgain(t *testing.T) {
	// Two ways from A to B: straight, and a longer way round.
	straight := line(protocol.Vec3{0.5, 31, 0.5}, protocol.Vec3{6.5, 31, 0.5}, 6)
	round := []protocol.Vec3{{0.5, 31, 0.5}, {0.5, 31, 2.5}, {2.5, 31, 4.5}, {4.5, 31, 4.5}, {6.5, 31, 2.5}, {6.5, 31, 0.5}}
	s, nodes := walked(t, straight, round)
	from, to := protocol.Vec3{0.5, 31, 0.5}, protocol.Vec3{6.5, 31, 0.5}
	direct := Find(nodes, from, to)
	if !direct.Found || direct.LengthM > 6.1 || direct.Avoided != 0 {
		t.Fatalf("%+v", direct)
	}
	now := time.Now().Add(time.Minute)
	if known, err := s.MarkRouteFailed("w", protocol.Vec3{2.5, 31, 0.5}, protocol.Vec3{3.5, 31, 0.5}, now); !known || err != nil {
		t.Fatal(known, err)
	}
	detour := Find(s.Routes("w"), from, to)
	if !detour.Found || detour.LengthM < 9 || detour.Avoided == 0 || !strings.Contains(detour.Explain, "failed links avoided") {
		t.Fatalf("%+v", detour)
	}
	// Walking the failed stretch again clears it.
	later := now.Add(time.Minute)
	if err := s.RememberRoute("w", []home.RouteStep{{From: protocol.Vec3{2.5, 31, 0.5}, To: protocol.Vec3{3.5, 31, 0.5}, At: later}}, later); err != nil {
		t.Fatal(err)
	}
	if again := Find(s.Routes("w"), from, to); again.LengthM > 6.1 || again.Avoided != 0 {
		t.Fatalf("%+v", again)
	}
}

func TestPlanFinishesAtTheTargetWhenTheLastCellIsShortOfIt(t *testing.T) {
	_, nodes := walked(t, line(protocol.Vec3{0.5, 31, 0.5}, protocol.Vec3{4.5, 31, 0.5}, 4))
	p := Find(nodes, protocol.Vec3{0.5, 31, 0.5}, protocol.Vec3{6.5, 31, 1.5})
	last := p.Legs[len(p.Legs)-1]
	if !p.Found || last.To != (protocol.Vec3{6.5, 31, 1.5}) || last.Cells != 0 || p.Target.DistanceM < 2 {
		t.Fatalf("%+v", p)
	}
}
