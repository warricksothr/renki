package session

import (
	"github.com/warricksothr/renki/internal/protocol"
	"time"
)

// ContainerObservation keeps arrival time independent of persistence retries.
type ContainerObservation struct {
	Frame *protocol.Container
	At    time.Time
}

// ContainerObservations scopes every frame and its metadata to one atomic
// connection/world snapshot. Disconnected/menu snapshots are never reused.
func (s *Session) ContainerObservations() (*protocol.Hello, string, *protocol.Perception, []ContainerObservation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil || s.state == nil || s.state.Phase != protocol.PhaseInWorld || s.state.World == "" {
		return nil, "", nil, nil
	}
	out := make([]ContainerObservation, 0, len(s.containers))
	for id, frame := range s.containers {
		out = append(out, ContainerObservation{frame, s.containerAt[id]})
	}
	return s.conn.Hello, s.state.World, s.seen, out
}
