// Package session is the controller's view of the plugin: the newest state,
// and every task the controller has sent with what the plugin said about it
// (docs/BRIDGE-PROTOCOL.md 6.1, 6.6 and 7.1). Everything that sends a task
// goes through Dispatch, which is where the capability and allowedTasks
// checks live.
package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/protocol"
)

// Errors from Dispatch.
var (
	ErrNotConnected = errors.New("the plugin is not connected")
	ErrNotAllowed   = errors.New("task is not allowed in this session")
)

// Listener hears what the session does not handle itself. Calls arrive on
// the connection's read goroutine, so a listener must not block on slow
// work such as an LLM call.
type Listener interface {
	Chat(*protocol.Chat)
	Event(*protocol.Event)
	// TaskDone is called once per task, when it reaches a final status.
	TaskDone(*Task)
}

// Order is a task the caller wants run.
type Order struct {
	Task     string
	Args     any
	TimeoutS float64
	// Origin is the caller's own note on why it was sent, such as the chat
	// line that asked for it. The session only carries it.
	Origin any
}

// Task is a task the session sent and what became of it.
type Task struct {
	ID     string
	Name   string
	Origin any

	mu     sync.Mutex
	status string
	reason string
	detail string
	done   chan struct{}
}

// Status returns the newest status, reason and detail.
func (t *Task) Status() (status, reason, detail string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.status, t.reason, t.detail
}

// Done is closed when the task reaches a final status.
func (t *Task) Done() <-chan struct{} { return t.done }

// Wait blocks until the task is final or ctx ends.
func (t *Task) Wait(ctx context.Context) (status, reason string, err error) {
	select {
	case <-t.done:
		s, r, _ := t.Status()
		return s, r, nil
	case <-ctx.Done():
		return "", "", ctx.Err()
	}
}

// update records a status and reports whether it made the task final. A
// task that is already final ignores later reports.
func (t *Task) update(status, reason, detail string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if protocol.Final(t.status) {
		return false
	}
	t.status, t.reason, t.detail = status, reason, detail
	if protocol.Final(status) {
		close(t.done)
		return true
	}
	return false
}

// Session tracks one bot.
type Session struct {
	log      *slog.Logger
	listener Listener

	mu       sync.Mutex
	conn     *bridge.Conn
	state    *protocol.State
	seen     *protocol.Perception
	carried  *protocol.Inventory
	allowed  []string
	tasks    map[string]*Task
	nextTask uint64
}

// New returns a session that reports to l.
func New(l Listener, log *slog.Logger) *Session {
	if log == nil {
		log = slog.Default()
	}
	return &Session{log: log, listener: l, tasks: map[string]*Task{}}
}

// SetListener replaces the listener. Call it before the bridge starts.
func (s *Session) SetListener(l Listener) { s.listener = l }

// Connected implements bridge.Handler.
func (s *Session) Connected(c *bridge.Conn) {
	allowed := slices.DeleteFunc(slices.Clone(c.Welcome.AllowedTasks), func(t string) bool {
		return !slices.Contains(c.Hello.Capabilities, t)
	})
	s.mu.Lock()
	s.conn, s.state, s.seen, s.carried, s.allowed = c, nil, nil, nil, allowed
	s.mu.Unlock()
	s.log.Info("session started", "allowed", allowed)
}

// Disconnected implements bridge.Handler. Every unfinished task of that
// connection is reported stopped, and none is sent again on reconnect: the
// plugin's dead-man rule has already stopped the body.
func (s *Session) Disconnected(c *bridge.Conn, err error) {
	s.mu.Lock()
	if s.conn != c {
		s.mu.Unlock()
		return
	}
	s.conn, s.allowed = nil, nil
	var ended []*Task
	for id, t := range s.tasks {
		if t.update(protocol.StatusCancelled, protocol.ReasonStopped, "disconnected") {
			ended = append(ended, t)
		}
		delete(s.tasks, id)
	}
	s.mu.Unlock()
	for _, t := range ended {
		s.listener.TaskDone(t)
	}
}

// Message implements bridge.Handler.
func (s *Session) Message(c *bridge.Conn, m protocol.Message) {
	switch m := m.(type) {
	case *protocol.State:
		s.mu.Lock()
		if s.conn == c && (s.state == nil || m.Seq > s.state.Seq) {
			s.state = m
		}
		s.mu.Unlock()
	case *protocol.TaskStatus:
		s.taskStatus(m)
	case *protocol.Chat:
		s.listener.Chat(m)
	case *protocol.Event:
		s.listener.Event(m)
	case *protocol.Perception:
		s.mu.Lock()
		if s.conn == c && (s.seen == nil || m.Seq > s.seen.Seq) {
			s.seen = m
		}
		s.mu.Unlock()
	case *protocol.Inventory:
		s.mu.Lock()
		if s.conn == c && (s.carried == nil || m.Seq > s.carried.Seq) {
			s.carried = m
		}
		s.mu.Unlock()
	case *protocol.Trace:
		// For the audit log and the panel, which have them from the
		// bridge already.
	case *protocol.Error:
		s.log.Warn("plugin reported an error", "re", m.Re, "code", m.Code, "detail", m.Detail)
	default:
		s.log.Debug("message not used", "type", m.MessageType())
	}
}

func (s *Session) taskStatus(m *protocol.TaskStatus) {
	s.mu.Lock()
	t, ok := s.tasks[m.Re]
	if ok && protocol.Final(m.Status) {
		delete(s.tasks, m.Re)
	}
	s.mu.Unlock()
	if !ok {
		s.log.Debug("status for a task this session does not hold", "re", m.Re, "status", m.Status)
		return
	}
	s.log.Info("task", "id", t.ID, "task", t.Name, "status", m.Status, "reason", m.Reason, "detail", m.Detail)
	if t.update(m.Status, m.Reason, m.Detail) {
		s.listener.TaskDone(t)
	}
}

// State returns the newest state, or nil before the first one.
func (s *Session) State() *protocol.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// Perception returns the newest perception snapshot, or nil before the
// first. Like State, it outlives a disconnect and resets on a new
// connection.
func (s *Session) Perception() *protocol.Perception {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seen
}

// Inventory returns what the bot carries, from the newest inventory
// message, or nil before the first. Like State, it outlives a disconnect
// and resets on a new connection.
func (s *Session) Inventory() *protocol.Inventory {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.carried
}

// Handshake returns the current connection's hello and welcome, and false
// when no plugin is connected.
func (s *Session) Handshake() (hello *protocol.Hello, welcome *protocol.Welcome, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return nil, nil, false
	}
	return s.conn.Hello, s.conn.Welcome, true
}

// Allowed reports whether task may be sent now: it is in both the plugin's
// capabilities and welcome.allowedTasks.
func (s *Session) Allowed(task string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Contains(s.allowed, task)
}

// AllowedTasks lists what may be sent now.
func (s *Session) AllowedTasks() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.allowed)
}

// Logout asks the plugin to save and return to the main menu. Whatever is
// running stops with it.
func (s *Session) Logout(ctx context.Context) error {
	s.mu.Lock()
	conn := s.conn
	s.nextTask++
	id := fmt.Sprintf("l-%d", s.nextTask)
	s.mu.Unlock()
	if conn == nil {
		return ErrNotConnected
	}
	s.log.Info("asking the plugin to log out", "id", id)
	return conn.Send(ctx, &protocol.Logout{Envelope: protocol.Envelope{ID: id}})
}

// Dispatch checks o and sends it. It refuses a task outside capabilities ∩
// allowedTasks, and arguments that DecodeArgs rejects, before anything
// reaches the plugin.
func (s *Session) Dispatch(ctx context.Context, o Order) (*Task, error) {
	s.mu.Lock()
	conn := s.conn
	if conn == nil {
		s.mu.Unlock()
		return nil, ErrNotConnected
	}
	if !slices.Contains(s.allowed, o.Task) {
		s.mu.Unlock()
		return nil, fmt.Errorf("%w: %q", ErrNotAllowed, o.Task)
	}
	s.nextTask++
	id := fmt.Sprintf("t-%d", s.nextTask)
	s.mu.Unlock()

	req, err := protocol.NewTask(id, o.Task, o.Args, o.TimeoutS)
	if err != nil {
		return nil, err
	}
	if _, err := req.DecodeArgs(); err != nil {
		return nil, err
	}
	t := &Task{ID: id, Name: o.Task, Origin: o.Origin, done: make(chan struct{})}

	s.mu.Lock()
	if s.conn != conn {
		s.mu.Unlock()
		return nil, ErrNotConnected
	}
	s.tasks[id] = t
	s.mu.Unlock()

	if err := conn.Send(ctx, req); err != nil {
		s.mu.Lock()
		delete(s.tasks, id)
		s.mu.Unlock()
		return nil, fmt.Errorf("send %s: %w", o.Task, err)
	}
	s.log.Info("task sent", "id", id, "task", o.Task)
	return t, nil
}

// Finished returns a task that has already ended, for tests of listeners.
func Finished(name string, origin any, status, reason string) *Task {
	t := &Task{ID: "t-test", Name: name, Origin: origin, done: make(chan struct{})}
	t.update(status, reason, "")
	return t
}
