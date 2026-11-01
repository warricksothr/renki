package session

import (
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestWorldEventsRequireFreshLiveWorldAndConnection(t *testing.T) {
	s, _, p := setup(t, nil, nil)
	p.Send(&protocol.State{World: "one", Phase: protocol.PhaseInWorld})
	waitFor(t, func() bool { return s.State() != nil })
	c := currentConn(t, s)
	events := &protocol.WorldEventStatus{RandomEvent: &protocol.AreaEvent{Name: "army_greydwarf", Radius: 96}}
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 100}, World: "one", Phase: protocol.PhaseInWorld, WorldEvents: events})
	_, at := s.StateObservation()
	if s.WorldEvents("one", c.ID(), at) == nil {
		t.Fatal("live area lost")
	}
	for _, tc := range []struct {
		world string
		conn  uint64
		now   time.Time
	}{
		{"two", c.ID(), at}, {"one", c.ID() + 1, at},
		{"one", c.ID(), at.Add(4 * time.Second)}, {"one", c.ID(), at.Add(-time.Second)},
	} {
		if s.WorldEvents(tc.world, tc.conn, tc.now) != nil {
			t.Fatal("stale or unscoped event", tc)
		}
	}
	// Fresh empty status is a distinct, observed ending. It does not assert
	// that event-spawned creatures or other dangers have disappeared.
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 101}, World: "one", Phase: protocol.PhaseInWorld, WorldEvents: &protocol.WorldEventStatus{}})
	_, at = s.StateObservation()
	if clear := s.WorldEvents("one", c.ID(), at); clear == nil || clear.RandomEvent != nil {
		t.Fatal("clear became unknown")
	}
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 102}, World: "one", Phase: protocol.PhaseMenu, WorldEvents: events})
	if s.WorldEvents("one", c.ID(), time.Now()) != nil {
		t.Fatal("menu event is live")
	}
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 103}, World: "one", Phase: protocol.PhaseInWorld, WorldEvents: events})
	s.Disconnected(c, nil)
	if s.WorldEvents("one", c.ID(), time.Now()) != nil {
		t.Fatal("offline event is live")
	}
}
