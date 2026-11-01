package session

import (
	"math"
	"slices"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/protocol"
)

// PickupObservation records an actual native ground pickup, not an inferred
// inventory increase or a chest withdrawal. Scope is captured on ingress.
type PickupObservation struct {
	World, WorldUID, Entity, Prefab string
	Pos                             protocol.Vec3
	Count                           int
	At                              time.Time
	Seq, Connection                 uint64
}

func (s *Session) observePickup(c *bridge.Conn, e *protocol.Event, now time.Time) {
	if e.Event != protocol.EventItemPickedUp {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.state
	if st == nil || e.World != st.World || e.WorldUID != st.WorldUID {
		return
	}
	if s.conn == nil || c != s.conn || st == nil || st.Phase != protocol.PhaseInWorld || st.World == "" || st.WorldUID == "" || st.HP <= 0 || now.Sub(s.stateAt) < 0 || now.Sub(s.stateAt) > 3*time.Second || e.Seq <= st.Seq || e.Seq <= s.pickupSeq || math.IsNaN(e.T) || math.IsInf(e.T, 0) || e.T < st.T || e.T-st.T > 3 || e.Pos == nil || e.Prefab == "" || len(e.Prefab) > 100 || e.Count <= 0 || e.Count > 1000000 {
		return
	}
	for _, n := range *e.Pos {
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return
		}
	}
	s.pickupSeq = e.Seq
	s.pickups = append(s.pickups, PickupObservation{World: st.World, WorldUID: st.WorldUID, Entity: e.By, Prefab: e.Prefab, Pos: *e.Pos, Count: e.Count, At: now, Seq: e.Seq, Connection: c.ID()})
	if len(s.pickups) > 256 {
		s.pickups = slices.Clone(s.pickups[len(s.pickups)-256:])
	}
}

func (s *Session) PickupObservations() (*protocol.Hello, []PickupObservation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return nil, nil
	}
	return s.conn.Hello, slices.Clone(s.pickups)
}
