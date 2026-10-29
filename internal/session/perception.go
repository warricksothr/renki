package session

import (
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

// PerceptionObservation returns one immutable frame with its arrival world
// and wall time, atomically. Menu, pre-state and disconnected frames are not
// observations of the current world and must never become fresh memories.
func (s *Session) PerceptionObservation() (*protocol.Hello, string, *protocol.Perception, time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil || s.state == nil || s.state.Phase != protocol.PhaseInWorld || s.state.World == "" || s.seen == nil || s.seenWorld != s.state.World {
		return nil, "", nil, time.Time{}
	}
	return s.conn.Hello, s.seenWorld, s.seen, s.seenAt
}
