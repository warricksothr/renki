package goal

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestAreaSelectionPrefersNearbyUnusedAreaWithFixedWorldCenters(t *testing.T) {
	g := Record{World: "one", Radius: 200, Center: protocol.Vec3{10, 30, 20}}
	pos := protocol.Vec3{90, 31, 20}
	first, raw, _ := nextArea(g, pos)
	if first == nil || first.Sector != 0 || *first.Center != (protocol.Vec3{90, 30, 20}) {
		t.Fatal(first)
	}
	var params map[string]json.RawMessage
	if err := json.Unmarshal(raw, &params); err != nil {
		t.Fatal(err)
	}
	var args protocol.AttackArgs
	delete(params, "world")
	params["kite"] = json.RawMessage("true")
	encoded, _ := json.Marshal(params)
	request := protocol.TaskRequest{Task: protocol.TaskAttack, Args: encoded}
	parsed, err := request.DecodeArgs()
	if err != nil {
		t.Fatal(err)
	}
	args = *parsed.(*protocol.AttackArgs)
	if params["searchAnchor"] == nil || params["searchLimit"] == nil || args.SearchAnchor == nil || *args.SearchAnchor != g.Center || *args.SearchLimit != 148 {
		t.Fatal("missing native absolute envelope", params)
	}
	// Both unreachable and genuinely empty attempts consume this area. A
	// changing bot position cannot slide the pool or select it again.
	for _, empty := range []bool{false, true} {
		g.Searches = []SearchAttempt{{Plan: *first, Empty: empty}}
		next, _, _ := nextArea(g, pos)
		if next == nil || next.Sector == 0 || math.Abs(distance(*next.Center, g.Center)-80) > 1e-6 {
			t.Fatal(next)
		}
		if distance(*next.Center, pos) > 70 {
			t.Fatal("chose opposite distant area instead of neighbour", next)
		}
	}
	// Existing saved sector attempts are respected when a goal resumes on
	// the regional implementation. They are not silently relabelled empty.
	g.Searches = []SearchAttempt{{Plan: SearchPlan{Sector: 0, Origin: g.Center, Radius: 48}}}
	next, _, _ := nextArea(g, pos)
	if next == nil || next.Sector == 0 {
		t.Fatal(next)
	}
	g.Radius = 63
	if next, _, _ := nextArea(g, pos); next != nil {
		t.Fatal("invented room outside a small envelope")
	}
}
