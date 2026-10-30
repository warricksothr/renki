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
	"strings"
	"sync"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/protocol"
)

// Errors from Dispatch.
var (
	ErrNotConnected = errors.New("the plugin is not connected")
	ErrNotAllowed   = errors.New("task is not allowed in this session")
	ErrBodyBusy     = errors.New("the body is busy or automation was interrupted")
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
	scopeHello *protocol.Hello
	scopeWorld string
	Task       string
	Args       any
	TimeoutS   float64
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
	entity string
	done   chan struct{}
}

// Status returns the newest status, reason and detail.
func (t *Task) Status() (status, reason, detail string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.status, t.reason, t.detail
}

// Entity returns the target bound by the native task's final receipt.
func (t *Task) Entity() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.entity
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
func (t *Task) update(status, reason, detail string, entity ...string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if protocol.Final(t.status) {
		return false
	}
	t.status, t.reason, t.detail = status, reason, detail
	if protocol.Final(status) {
		if len(entity) > 0 {
			t.entity = entity[0]
		}
		close(t.done)
		return true
	}
	return false
}

// Session tracks one bot.
type Session struct {
	log      *slog.Logger
	listener Listener

	mu             sync.Mutex
	dispatchMu     sync.Mutex // serialize wire order and automation ownership
	automation     *Automation
	orders         uint64
	conn           *bridge.Conn
	state          *protocol.State
	stateAt        time.Time
	injurySeq      uint64
	playerChatSeq  uint64
	seen           *protocol.Perception
	seenAt         time.Time
	seenWorld      string
	carried        *protocol.Inventory
	containers     map[string]*protocol.Container
	containerAt    map[string]time.Time
	terrain        map[[2]int]protocol.TerrainTile
	terrainSeq     uint64
	terrainScope   *protocol.TerrainScope
	terrainPending []TerrainObservation
	terrainDropped uint64
	screenshot     *CameraFrame
	layout         *protocol.Layout
	allowed        []string
	permit         func(task string) bool
	movePlace      func(world, name string) (protocol.Vec3, error)
	tasks          map[string]*Task
	nextTask       uint64
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

// SetMovePlace resolves controller-owned waypoint names before a move goes
// on the bridge. Entity IDs contain a colon and keep their wire semantics.
func (s *Session) SetMovePlace(resolve func(world, name string) (protocol.Vec3, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.movePlace = resolve
}

// SetPermit narrows what may be sent, read on every Dispatch: a task must
// also pass permit, such as the runtime settings' allowedTasks. What the
// plugin agreed to in welcome still bounds it.
func (s *Session) SetPermit(permit func(task string) bool) {
	s.mu.Lock()
	s.permit = permit
	s.mu.Unlock()
}

// permitted reports whether task may be sent. s.mu is held.
func (s *Session) permitted(task string) bool {
	return slices.Contains(s.allowed, task) && (s.permit == nil || s.permit(task))
}

// Connected implements bridge.Handler.
func (s *Session) Connected(c *bridge.Conn) {
	allowed := slices.DeleteFunc(slices.Clone(c.Welcome.AllowedTasks), func(t string) bool {
		return !slices.Contains(c.Hello.Capabilities, t)
	})
	s.mu.Lock()
	// The bridge closes the old socket without waiting for its read loop.
	// Its Disconnected callback may arrive after this one and correctly
	// ignore the replaced connection. Finish its tasks here as well.
	var ended []*Task
	for id, t := range s.tasks {
		if t.update(protocol.StatusCancelled, protocol.ReasonStopped, "connection replaced") {
			ended = append(ended, t)
		}
		delete(s.tasks, id)
	}
	s.conn, s.state, s.seen, s.carried, s.allowed = c, nil, nil, nil, allowed
	s.seenAt, s.seenWorld = time.Time{}, ""
	s.injurySeq = 0
	s.playerChatSeq = 0
	s.terrain, s.terrainSeq = nil, 0
	s.terrainScope = nil
	s.layout = nil
	s.screenshot = nil
	s.automation = nil
	s.containers = make(map[string]*protocol.Container)
	s.containerAt = make(map[string]time.Time)
	s.mu.Unlock()
	s.log.Info("session started", "allowed", allowed)
	for _, t := range ended {
		if s.listener != nil {
			s.listener.TaskDone(t)
		}
	}
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
	s.screenshot = nil
	s.automation = nil
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
	case *protocol.Screenshot:
		s.rememberScreenshot(c, m)
	case *protocol.State:
		s.mu.Lock()
		if s.conn == c && (s.state == nil || m.Seq > s.state.Seq) {
			if s.state != nil && (s.state.World != m.World || s.state.WorldUID != m.WorldUID) {
				s.containers = make(map[string]*protocol.Container)
				s.containerAt = make(map[string]time.Time)
				s.seen, s.carried, s.automation = nil, nil, nil
				s.terrain, s.terrainSeq = nil, 0
				s.terrainScope = nil
				s.layout = nil
				s.screenshot = nil
			}
			if m.Phase != protocol.PhaseInWorld || m.HP <= 0 {
				s.automation = nil
			}
			if m.Phase != protocol.PhaseInWorld {
				s.seenWorld = ""
				s.terrain, s.terrainSeq = nil, 0
				s.terrainScope = nil
				s.layout = nil
				s.screenshot = nil
			}
			s.state = m
			s.stateAt = time.Now().UTC()
		}
		s.mu.Unlock()
	case *protocol.TaskStatus:
		s.taskStatus(m)
	case *protocol.Chat:
		if observer, ok := s.listener.(PlayerChatListener); ok {
			if observation, valid := s.ObservePlayerChat(c, m, time.Now().UTC()); valid {
				observer.PlayerChat(m, observation)
			}
		}
		s.mu.Lock()
		current := s.conn == c
		s.mu.Unlock()
		if current {
			s.listener.Chat(m)
		}
	case *protocol.Event:
		if observation, valid := s.ObserveInjury(c, m, time.Now().UTC()); valid {
			if observer, ok := s.listener.(InjuryListener); ok {
				observer.Injury(m, observation)
			}
		}
		s.mu.Lock()
		current := s.conn == c
		s.mu.Unlock()
		if current {
			s.listener.Event(m)
		}
	case *protocol.Perception:
		s.mu.Lock()
		if s.conn == c && (s.seen == nil || m.Seq > s.seen.Seq) {
			s.seen = m
			s.seenAt, s.seenWorld = time.Now().UTC(), ""
			if s.state != nil && s.state.Phase == protocol.PhaseInWorld {
				s.seenWorld = s.state.World
			}
		}
		s.mu.Unlock()
	case *protocol.Inventory:
		s.mu.Lock()
		if s.conn == c && (s.carried == nil || m.Seq > s.carried.Seq) {
			s.carried = m
		}
		s.mu.Unlock()
	case *protocol.Container:
		s.mu.Lock()
		if s.conn == c && m.Entity != "" && (s.containers[m.Entity] == nil || m.Seq > s.containers[m.Entity].Seq) {
			s.containers[m.Entity] = m
			s.containerAt[m.Entity] = time.Now().UTC()
		}
		s.mu.Unlock()
	case *protocol.Terrain:
		s.rememberTerrain(c, m)
	case *protocol.Layout:
		s.mu.Lock()
		if s.conn == c && s.state != nil && s.state.Phase == protocol.PhaseInWorld && (s.layout == nil || m.Seq > s.layout.Seq) {
			s.layout = m
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
	if t.update(m.Status, m.Reason, m.Detail, m.Entity) {
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

// PackSnapshot scopes inventory and carrying capacity to one live body/world.
// A missing Inventory means it has not been observed in that scope yet.
type PackSnapshot struct {
	World      string              `json:"world"`
	Connection uint64              `json:"connection"`
	Weight     float64             `json:"weight"`
	WeightMax  float64             `json:"weightMax"`
	Inventory  *protocol.Inventory `json:"inventory"`
}

func (s *Session) PackSnapshot() PackSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil || s.state == nil || s.state.Phase != protocol.PhaseInWorld || s.state.World == "" {
		return PackSnapshot{}
	}
	return PackSnapshot{World: s.state.World, Connection: s.conn.ID(), Weight: s.state.Weight, WeightMax: s.state.WeightMax, Inventory: s.carried}
}

// Containers lists chest snapshots from this connection; snapshots are immutable.
func (s *Session) Containers() []*protocol.Container {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*protocol.Container, 0, len(s.containers))
	for _, c := range s.containers {
		out = append(out, c)
	}
	return out
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
// capabilities and welcome.allowedTasks, and passes the permit.
func (s *Session) Allowed(task string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.permitted(task)
}

// AllowedTasks lists what may be sent now.
func (s *Session) AllowedTasks() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.DeleteFunc(slices.Clone(s.allowed), func(t string) bool { return !s.permitted(t) })
}

// Logout asks the plugin to save and return to the main menu. Whatever is
// running stops with it.
func (s *Session) Logout(ctx context.Context) error {
	s.dispatchMu.Lock()
	defer s.dispatchMu.Unlock()
	s.mu.Lock()
	s.automation = nil
	s.orders++
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
	s.dispatchMu.Lock()
	defer s.dispatchMu.Unlock()
	return s.dispatch(ctx, o, nil)
}

func (s *Session) dispatch(ctx context.Context, o Order, automation *Automation) (*Task, error) {
	s.mu.Lock()
	conn := s.conn
	if conn == nil {
		s.mu.Unlock()
		return nil, ErrNotConnected
	}
	if !s.orderScopeMatches(o, conn) {
		s.mu.Unlock()
		return nil, ErrBodyBusy
	}
	if !s.permitted(o.Task) {
		s.mu.Unlock()
		return nil, fmt.Errorf("%w: %q", ErrNotAllowed, o.Task)
	}
	s.nextTask++
	id := fmt.Sprintf("t-%d", s.nextTask)
	resolve := s.movePlace
	world, phase := "", ""
	if s.state != nil {
		world, phase = s.state.World, s.state.Phase
	}
	s.mu.Unlock()

	req, err := protocol.NewTask(id, o.Task, o.Args, o.TimeoutS)
	if err != nil {
		return nil, err
	}
	decoded, err := req.DecodeArgs()
	if err != nil {
		return nil, err
	}
	resolved := false
	if args, ok := decoded.(*protocol.MoveToArgs); ok && resolve != nil && args.To.Entity != "" && !strings.Contains(args.To.Entity, ":") {
		if world == "" || phase != protocol.PhaseInWorld {
			return nil, errors.New("join a world before using a waypoint")
		}
		pos, err := resolve(world, args.To.Entity)
		if err != nil {
			return nil, err
		}
		args.To = protocol.Target{Pos: &pos}
		req, err = protocol.NewTask(id, o.Task, args, o.TimeoutS)
		if err != nil {
			return nil, err
		}
		if _, err := req.DecodeArgs(); err != nil {
			return nil, err
		}
		resolved = true
	}
	t := &Task{ID: id, Name: o.Task, Origin: o.Origin, done: make(chan struct{})}

	s.mu.Lock()
	if s.conn != conn {
		s.mu.Unlock()
		return nil, ErrNotConnected
	}
	if !s.orderScopeMatches(o, conn) {
		s.mu.Unlock()
		return nil, ErrBodyBusy
	}
	if resolved && (s.state == nil || s.state.World != world || s.state.Phase != protocol.PhaseInWorld) {
		s.mu.Unlock()
		return nil, errors.New("world changed while resolving the waypoint")
	}
	if automation != nil && s.automation != automation {
		s.mu.Unlock()
		return nil, ErrBodyBusy
	}
	if automation == nil && !protocol.Instant(o.Task) {
		s.automation = nil
		s.orders++
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
	return FinishedWith(name, origin, status, reason, "")
}

// FinishedWith is Finished with the plugin's detail, for tests.
func FinishedWith(name string, origin any, status, reason, detail string) *Task {
	t := &Task{ID: "t-test", Name: name, Origin: origin, done: make(chan struct{})}
	t.update(status, reason, detail)
	return t
}

// DispatchFor sends an instant reply only in the observed connection/world.
// It checks scope alongside connection selection and again before publishing.
func (s *Session) DispatchFor(ctx context.Context, o Order, hello *protocol.Hello, world string) (*Task, error) {
	if hello == nil || world == "" || !protocol.Instant(o.Task) {
		return nil, ErrBodyBusy
	}
	o.scopeHello, o.scopeWorld = hello, world
	return s.Dispatch(ctx, o)
}

// orderScopeMatches holds s.mu.
func (s *Session) orderScopeMatches(o Order, conn *bridge.Conn) bool {
	if o.scopeHello == nil {
		return true
	}
	return conn != nil && conn.Hello == o.scopeHello && s.state != nil && s.state.World == o.scopeWorld && s.state.Phase == protocol.PhaseInWorld && s.state.HP > 0 && time.Since(s.stateAt) >= 0 && time.Since(s.stateAt) <= 3*time.Second
}
