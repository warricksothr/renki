package memory

import (
	"math"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

func standing(x, y, z float64, flags ...string) *protocol.State {
	if flags == nil {
		flags = []string{protocol.FlagGrounded}
	}
	return &protocol.State{Phase: protocol.PhaseInWorld, World: "w", HP: 20, Pos: protocol.Vec3{x, y, z}, Flags: flags}
}

func TestRouteChainJoinsOnlyGroundedSamplesCloseInSpaceAndTime(t *testing.T) {
	var c routeChain
	t0 := time.Now()
	at := func(s float64) time.Time { return t0.Add(time.Duration(s * float64(time.Second))) }
	if _, ok := c.observe(standing(0, 30, 0), at(0), 1); ok {
		t.Fatal("the first sample made a step")
	}
	// Too small a move is not a step, but keeps the chain alive.
	if _, ok := c.observe(standing(.2, 30, 0), at(2.5), 1); ok {
		t.Fatal("a shuffle made a step")
	}
	step, ok := c.observe(standing(1, 30, 0), at(5), 1)
	if !ok || step.From != (protocol.Vec3{0, 30, 0}) || step.To != (protocol.Vec3{1, 30, 0}) || !step.At.Equal(at(5)) {
		t.Fatal(step, ok)
	}
	for _, tc := range []struct {
		name string
		next func() (home.RouteStep, bool)
	}{
		{"teleport", func() (home.RouteStep, bool) { return c.observe(standing(60, 30, 0), at(6), 1) }},
		{"silence", func() (home.RouteStep, bool) { return c.observe(standing(61, 30, 0), at(20), 1) }},
		{"reconnect", func() (home.RouteStep, bool) { return c.observe(standing(62, 30, 0), at(21), 2) }},
		{"backwards", func() (home.RouteStep, bool) { return c.observe(standing(63, 30, 0), at(1), 2) }},
	} {
		if _, ok := tc.next(); ok {
			t.Fatal(tc.name, "was joined into a walk")
		}
	}
	// The chain restarts from the last accepted sample.
	if step, ok = c.observe(standing(64, 30, 0), at(1.5), 2); !ok || step.From != (protocol.Vec3{63, 30, 0}) {
		t.Fatal(step, ok)
	}
}

func TestRouteChainDropsPositionsTheStoreWouldRefuse(t *testing.T) {
	var c routeChain
	t0 := time.Now()
	c.observe(standing(0, 30, 0), t0, 1)
	for _, x := range []float64{2e6, math.NaN(), math.Inf(1)} {
		if _, ok := c.observe(standing(x, 30, 0), t0.Add(time.Second), 1); ok {
			t.Fatal("made a step to", x)
		}
		if step, ok := c.observe(standing(1, 30, 0), t0.Add(2*time.Second), 1); ok {
			t.Fatal("joined across a bad position", step)
		}
		c.observe(standing(0, 30, 0), t0, 1)
	}
}

func TestRouteChainNeverJoinsAcrossAJumpASwimOrDeath(t *testing.T) {
	t0 := time.Now()
	for name, between := range map[string]*protocol.State{
		"airborne": standing(1, 31, 0, protocol.FlagRunning),
		"swimming": standing(1, 30, 0, protocol.FlagGrounded, protocol.FlagSwimming),
		"menu":     {Phase: "menu", World: "w", HP: 20, Flags: []string{protocol.FlagGrounded}},
		"dead":     {Phase: protocol.PhaseInWorld, World: "w", Flags: []string{protocol.FlagGrounded}},
	} {
		var c routeChain
		c.observe(standing(0, 30, 0), t0, 1)
		if _, ok := c.observe(between, t0.Add(time.Second), 1); ok {
			t.Fatal(name, "made a step")
		}
		// Landing two metres on, a second later, is not a walked link.
		if _, ok := c.observe(standing(2, 28, 0), t0.Add(2*time.Second), 1); ok {
			t.Fatal(name, "was bridged by the next grounded sample")
		}
	}
}
