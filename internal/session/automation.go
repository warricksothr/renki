package session

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/warricksothr/renki/internal/protocol"
)

// Automation owns successive steps of one autonomous run. Normal body
// orders invalidate it before sending, even before a new state arrives.
// Keep this handle for the entire run; never reacquire to finish old work.
type Automation struct {
	s    *Session
	loan *loan // set when the handle borrowed an owner's follow
}

// loan is the owner's follow an automation borrowed.
type loan struct {
	task     *Task
	returned bool
}

// Follow is an owner's follow the body is running.
type Follow struct {
	TaskID string
	Player string
}

// ErrNoLoan is Return on a handle that did not borrow a follow, or that
// already gave it back.
var ErrNoLoan = errors.New("automation holds no lent follow")

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

// OwnerFollow reports the follow the body is running for an owner order:
// a follow sent outside automation (or given back by Return), still the
// state's current task, and the only body task in flight.
func (s *Session) OwnerFollow() (Follow, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.ownerFollow()
	if t == nil {
		return Follow{}, false
	}
	var args protocol.FollowArgs
	if json.Unmarshal(t.Args, &args) != nil || args.Player == "" {
		return Follow{}, false
	}
	return Follow{TaskID: t.ID, Player: args.Player}, true
}

// ownerFollow holds s.mu.
func (s *Session) ownerFollow() *Task {
	if s.conn == nil || s.state == nil || s.state.Task == nil || protocol.Final(s.state.Task.Status) {
		return nil
	}
	var follow *Task
	for _, t := range s.tasks {
		if protocol.Instant(t.Name) {
			continue
		}
		if follow != nil || t.Name != protocol.TaskFollow || t.automation != nil || t.ID != s.state.Task.ID {
			return nil
		}
		follow = t
	}
	if follow == nil {
		return nil
	}
	if status, _, _ := follow.Status(); protocol.Final(status) {
		return nil
	}
	return follow
}

// TryLoanFrom reserves a living body that is running the owner's follow
// followID, as TryAutomationFor reserves an idle one. The handle's first
// body task replaces the follow, which ends superseded and reports Lent.
// Return sends the same follow again. Any owner order revokes the handle
// as it does every automation, and then nothing is returned.
func (s *Session) TryLoanFrom(hello *protocol.Hello, world, followID string) (*Automation, error) {
	s.dispatchMu.Lock()
	defer s.dispatchMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return nil, ErrNotConnected
	}
	if hello == nil || s.conn.Hello != hello || s.state == nil || s.state.World != world {
		return nil, ErrBodyBusy
	}
	if s.automation != nil || s.state.World == "" || s.state.Phase != protocol.PhaseInWorld || s.state.HP <= 0 || s.state.Stranded() {
		return nil, ErrBodyBusy
	}
	t := s.ownerFollow()
	if t == nil || t.ID != followID {
		return nil, ErrBodyBusy
	}
	a := &Automation{s: s, loan: &loan{task: t}}
	s.automation = a
	return a, nil
}

// Lent is the follow this handle borrowed, or false for an ordinary one.
func (a *Automation) Lent() (Follow, bool) {
	if a == nil || a.loan == nil {
		return Follow{}, false
	}
	var args protocol.FollowArgs
	_ = json.Unmarshal(a.loan.task.Args, &args)
	return Follow{TaskID: a.loan.task.ID, Player: args.Player}, true
}

// Return sends the borrowed follow again, for the same player with the
// same arguments, timeout and origin, while the handle is still valid. It
// is not a new owner order: ControlRevision stays. The new follow is the
// owner's, and the handle ends with it. A revoked handle returns nothing.
// A handle that never sent a body task leaves the follow running, releases
// itself and returns a nil task.
func (a *Automation) Return(ctx context.Context) (*Task, error) {
	a.s.dispatchMu.Lock()
	defer a.s.dispatchMu.Unlock()
	if a.loan == nil || a.loan.returned {
		return nil, ErrNoLoan
	}
	t := a.loan.task
	a.loan.returned = true
	if !t.Lent() {
		// Nothing replaced the follow, so it is still running: let it be.
		a.Release()
		return nil, nil
	}
	return a.s.dispatch(ctx, Order{giveBack: true, Task: protocol.TaskFollow, Args: json.RawMessage(t.Args), TimeoutS: t.timeoutS, Origin: t.Origin}, a)
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
