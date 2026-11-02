package session

import (
	"math"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/protocol"
)

func TestInjuryTotalsUseActualAmountsNotWireSpacingOrHealedHealth(t *testing.T) {
	s := New(&listener{}, nil)
	c := &bridge.Conn{Hello: &protocol.Hello{}, Welcome: &protocol.Welcome{}}
	s.Connected(c)
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 10, T: 10}, Phase: protocol.PhaseInWorld, World: "one", HP: 50})
	s.Message(c, &protocol.Event{Envelope: protocol.Envelope{Seq: 111, T: 10.1}, Event: protocol.EventDamaged, Amount: 4})
	s.Message(c, &protocol.Event{Envelope: protocol.Envelope{Seq: 111, T: 10.1}, Event: protocol.EventDamaged, Amount: 4})
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 800, T: 11}, Phase: protocol.PhaseInWorld, World: "one", HP: 50})
	s.Message(c, &protocol.Event{Envelope: protocol.Envelope{Seq: 900, T: 11.1}, Event: protocol.EventDamaged, Amount: 7})
	got := s.InjuryStatus()
	if got == nil || !got.Complete || got.Sequence != 900 || got.Damage != 11 {
		t.Fatal(got)
	}
	s.Connected(&bridge.Conn{Hello: &protocol.Hello{}, Welcome: &protocol.Welcome{}})
	if s.InjuryStatus() != nil {
		t.Fatal("old injury allowance crossed reconnect")
	}
}

func TestUnknownDamageOrDeathPermanentlyInvalidatesConnectionDamageAllowance(t *testing.T) {
	for _, amount := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		s := New(&listener{}, nil)
		c := &bridge.Conn{Hello: &protocol.Hello{}, Welcome: &protocol.Welcome{}}
		s.Connected(c)
		s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 1, T: 1}, Phase: protocol.PhaseInWorld, World: "one", HP: 50})
		s.Message(c, &protocol.Event{Envelope: protocol.Envelope{Seq: 2, T: 1.1}, Event: protocol.EventDamaged, Amount: amount})
		s.Message(c, &protocol.Event{Envelope: protocol.Envelope{Seq: 3, T: 1.2}, Event: protocol.EventDamaged, Amount: 1})
		if got := s.InjuryStatus(); got == nil || got.Complete || got.Damage != 1 {
			t.Fatal(got)
		}
	}
	s := New(&listener{}, nil)
	c := &bridge.Conn{Hello: &protocol.Hello{}, Welcome: &protocol.Welcome{}}
	s.Connected(c)
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 1, T: 1}, Phase: protocol.PhaseInWorld, World: "one", HP: 0})
	s.Message(c, &protocol.Event{Envelope: protocol.Envelope{Seq: 2, T: 1.1}, Event: protocol.EventDied})
	if got := s.InjuryStatus(); got == nil || got.Complete {
		t.Fatal(got)
	}
}

type injuryListener struct {
	events   int
	injuries []InjuryObservation
}

func (*injuryListener) Chat(*protocol.Chat)     {}
func (*injuryListener) TaskDone(*Task)          {}
func (l *injuryListener) Event(*protocol.Event) { l.events++ }
func (l *injuryListener) Injury(_ *protocol.Event, o InjuryObservation) {
	l.injuries = append(l.injuries, o)
}

func TestInjuryContextFreshnessReconnectAndEventSource(t *testing.T) {
	l := &injuryListener{}
	s := New(l, nil)
	c := &bridge.Conn{Hello: &protocol.Hello{}, Welcome: &protocol.Welcome{}}
	s.Connected(c)
	event := &protocol.Event{Envelope: protocol.Envelope{Seq: 11, T: 10.2}, Event: protocol.EventDamaged, Amount: 5}
	check := func(ok bool, now time.Time) InjuryObservation {
		t.Helper()
		got, valid := s.ObserveInjury(c, event, now)
		if valid != ok {
			t.Fatalf("accepted=%v want=%v", valid, ok)
		}
		return got
	}
	check(false, time.Now())
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 10, T: 10}, Phase: protocol.PhaseInWorld, World: "one", Pos: protocol.Vec3{1, 2, 3}, HP: 25})
	observation := check(true, time.Now())
	if observation.World != "one" || observation.Pos != (protocol.Vec3{1, 2, 3}) {
		t.Fatal(observation)
	}
	check(false, time.Now()) // replayed sequence
	event.Seq++
	check(false, time.Now().Add(6*time.Second))
	check(false, time.Now().Add(-time.Minute))
	event.T = 14
	check(false, time.Now())
	event.T = 9
	check(false, time.Now())
	event.T = 10.2
	precise := protocol.Vec3{4, 5, 6}
	event.Pos = &precise
	if got := check(true, time.Now()); got.Pos != precise {
		t.Fatal(got)
	}
	// A death immediately after a zero-HP state still has a usable location.
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 13, T: 10.3}, Phase: protocol.PhaseInWorld, World: "one", HP: 0})
	event.Seq, event.T, event.Event = 14, 10.3, protocol.EventDied
	s.Message(c, event)
	if len(l.injuries) != 1 || l.injuries[0].World != "one" {
		t.Fatal(l.injuries)
	}
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 15, T: 10.4}, Phase: protocol.PhaseMenu})
	event.Seq = 16
	check(false, time.Now())
	replacement := &bridge.Conn{Hello: &protocol.Hello{}, Welcome: &protocol.Welcome{}}
	s.Connected(replacement)
	check(false, time.Now())
	l.events = 0
	s.Message(c, event)
	if l.events != 0 {
		t.Fatal("replaced connection delivered an event")
	}
	s.Message(replacement, &protocol.State{Envelope: protocol.Envelope{Seq: 1, T: 10}, Phase: protocol.PhaseInWorld, World: "two", HP: 25})
	s.Message(c, event)
	if len(l.injuries) != 1 {
		t.Fatal("old socket event entered replacement world")
	}
	event.Seq, event.T = 2, 10
	s.Message(replacement, event)
	if l.events != 1 || len(l.injuries) != 2 || l.injuries[1].World != "two" {
		t.Fatal(l.events, l.injuries)
	}
	if observation.World != "one" {
		t.Fatal("captured observation changed after reconnect")
	}
}

func TestInjuryMarkerWorksWithoutRecorderAndIsUnavailableAcrossReconnect(t *testing.T) {
	s := New(&listener{}, nil)
	c := &bridge.Conn{Hello: &protocol.Hello{}, Welcome: &protocol.Welcome{}}
	s.Connected(c)
	if s.InjurySequence() != nil {
		t.Fatal("unobserved body has an injury baseline")
	}
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 10, T: 10}, Phase: protocol.PhaseInWorld, World: "one", HP: 25})
	if seq := s.InjurySequence(); seq == nil || *seq != 0 {
		t.Fatal(seq)
	}
	s.Message(c, &protocol.Event{Envelope: protocol.Envelope{Seq: 11, T: 10.1}, Event: protocol.EventDamaged, Amount: 1})
	if seq := s.InjurySequence(); seq == nil || *seq != 11 {
		t.Fatal(seq)
	}
	s.Connected(&bridge.Conn{Hello: &protocol.Hello{}, Welcome: &protocol.Welcome{}})
	if s.InjurySequence() != nil {
		t.Fatal("old injury marker crossed a reconnect")
	}
}
