package session

import (
	"context"

	"github.com/warricksothr/renki/internal/protocol"
)

// Automation owns successive steps of one autonomous run. Normal body
// orders invalidate it before sending, even before a new state arrives.
// Keep this handle for the entire run; never reacquire to finish old work.
type Automation struct{ s *Session }

// ControlRevision changes even when an order arrives between runs, so a
// scheduler can discard queued work as well as a running continuation.
func (s *Session) ControlRevision() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.orders
}

// TryAutomation reserves an idle, living body. Pending dispatched tasks
// count as busy even while the streamed state still says idle.
func (s *Session) TryAutomation() (*Automation, error) {
	return s.TryAutomationFor(nil, "")
}

// TryAutomationFor also pins planning observations to their connection and
// world. A reconnect/world change between observing and acquiring is busy.
func (s *Session) TryAutomationFor(hello *protocol.Hello, world string) (*Automation, error) {
	s.dispatchMu.Lock()
	defer s.dispatchMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return nil, ErrNotConnected
	}
	if hello != nil && (s.conn.Hello != hello || s.state == nil || s.state.World != world) {
		return nil, ErrBodyBusy
	}
	if s.automation != nil || s.state == nil || s.state.World == "" || s.state.Phase != protocol.PhaseInWorld || s.state.HP <= 0 || s.state.Stranded() {
		return nil, ErrBodyBusy
	}
	if t := s.state.Task; t != nil && !protocol.Final(t.Status) {
		return nil, ErrBodyBusy
	}
	for _, t := range s.tasks {
		if !protocol.Instant(t.Name) {
			return nil, ErrBodyBusy
		}
	}
	a := &Automation{s: s}
	s.automation = a
	return a, nil
}

func (a *Automation) Valid() bool {
	a.s.mu.Lock()
	defer a.s.mu.Unlock()
	return a.s.automation == a && a.s.conn != nil
}

func (a *Automation) Dispatch(ctx context.Context, o Order) (*Task, error) {
	a.s.dispatchMu.Lock()
	defer a.s.dispatchMu.Unlock()
	return a.s.dispatch(ctx, o, a)
}

// Release never sends stop: a stale handle must not stop a newer order.
func (a *Automation) Release() {
	a.s.mu.Lock()
	defer a.s.mu.Unlock()
	if a.s.automation == a {
		a.s.automation = nil
	}
}
