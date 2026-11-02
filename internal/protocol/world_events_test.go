package protocol

import (
	"math"
	"testing"
)

func TestAreaEventMatchesGameMembership(t *testing.T) {
	e := &AreaEvent{Name: "army_greydwarf", Center: Vec3{10, 30, 20}, Radius: 96}
	for _, tc := range []struct {
		p      Vec3
		inside bool
	}{
		{Vec3{10, 30, 20}, true}, {Vec3{105.99, -500, 20}, true},
		{Vec3{106, 30, 20}, false}, {Vec3{10, 3000, 20}, true},
		{Vec3{10, 3000.01, 20}, false}, {Vec3{math.NaN(), 30, 20}, false},
	} {
		if e.Contains(tc.p) != tc.inside {
			t.Fatalf("membership %v", tc)
		}
	}
	e.Radius = math.Inf(1)
	if e.Contains(Vec3{10, 30, 20}) {
		t.Fatal("accepted malformed area")
	}
}

func TestWorldEventKnowledgeDistinguishesUnavailableClearAndLocal(t *testing.T) {
	var unknown *WorldEventStatus
	if unknown.Valid() || !(&WorldEventStatus{}).Valid() {
		t.Fatal("unknown and observed clear confused")
	}
	local := &WorldEventStatus{LocalEvent: &EventNotice{Name: "boss", Kind: "local"}}
	if !local.Valid() || local.RandomEvent != nil {
		t.Fatal("local notice acquired invented bounds")
	}
}
