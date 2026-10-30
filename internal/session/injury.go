package session

import (
	"math"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/protocol"
)

// InjuryObservation captures scope on the connection's read goroutine, before
// any asynchronous recorder or reconnect can change the current world.
type InjuryObservation struct {
	World string
	Pos   protocol.Vec3
	At    time.Time
}

// InjuryListener is optional: existing listeners continue to receive Event.
type InjuryListener interface {
	Injury(*protocol.Event, InjuryObservation)
}

// InjurySequence is a connection-scoped marker of accepted injury events.
// Zero is known no injury; nil means no current in-world body observation.
func (s *Session) InjurySequence() *uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil || s.state == nil || s.state.Phase != protocol.PhaseInWorld {
		return nil
	}
	seq := s.injurySeq
	return &seq
}

// ObserveInjury attaches an injury event to fresh current-world state. Events
// carry precise position on new plugins; older plugins use the last state.
// No store or disk work is done under the session lock.
func (s *Session) ObserveInjury(c *bridge.Conn, e *protocol.Event, now time.Time) (InjuryObservation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.state
	if s.conn == nil || s.conn != c || (e.Event != protocol.EventDamaged && e.Event != protocol.EventDied) || e.Seq <= s.injurySeq || st == nil || st.Phase != protocol.PhaseInWorld || st.World == "" || now.Sub(s.stateAt) < 0 || now.Sub(s.stateAt) > 5*time.Second || e.Seq <= st.Seq || math.IsNaN(e.T) || math.IsInf(e.T, 0) || e.T < st.T || e.T-st.T > 3 {
		return InjuryObservation{}, false
	}
	pos := st.Pos
	if e.Pos != nil {
		pos = *e.Pos
	}
	s.injurySeq = e.Seq
	return InjuryObservation{World: st.World, Pos: pos, At: now}, true
}
