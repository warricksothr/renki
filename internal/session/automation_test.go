package session

import (
	"context"
	"errors"
	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestAutomationYieldsBeforeTheNextStateFrame(t *testing.T) {
	s, _, p := setup(t, []string{"stop", "move_to", "say", "flee"}, []string{"stop", "move_to", "say", "flee"})
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "test", HP: 25})
	waitFor(t, func() bool { return s.State() != nil })
	ctx := context.Background()
	lease, err := s.TryAutomation()
	if err != nil {
		t.Fatal(err)
	}
	pos := protocol.Vec3{1, 0, 0}
	step := Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}}, TimeoutS: 10}
	if _, err = lease.Dispatch(ctx, step); err != nil {
		t.Fatal(err)
	}
	p.NextTask()
	// No state update: the plugin still appears idle to outside readers.
	if _, err = s.Dispatch(ctx, Order{Task: protocol.TaskFlee, Args: &protocol.FleeArgs{From: protocol.Target{Pos: &pos}}, TimeoutS: 10}); err != nil {
		t.Fatal(err)
	}
	if req := p.NextTask(); req.Task != protocol.TaskFlee {
		t.Fatal(req)
	}
	if lease.Valid() {
		t.Fatal("normal order left automation valid")
	}
	if _, err = lease.Dispatch(ctx, step); !errors.Is(err, ErrBodyBusy) {
		t.Fatal(err)
	}
	if _, err = lease.Dispatch(ctx, Order{Task: protocol.TaskStop}); !errors.Is(err, ErrBodyBusy) {
		t.Fatal(err)
	}
	lease.Release()
	p.Quiet(30 * time.Millisecond)
	if _, err = s.TryAutomation(); !errors.Is(err, ErrBodyBusy) {
		t.Fatal("pending flee must reserve body", err)
	}
}

func TestAutomationInvalidatesOnWorldPhaseLogoutAndReconnect(t *testing.T) {
	for _, change := range []string{"world", "dead", "menu", "logout", "reconnect"} {
		t.Run(change, func(t *testing.T) {
			s, _, p := setup(t, []string{"stop"}, nil)
			p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "old", HP: 25})
			waitFor(t, func() bool { return s.State() != nil })
			lease, err := s.TryAutomation()
			if err != nil {
				t.Fatal(err)
			}
			st := *s.State()
			st.Seq++
			switch change {
			case "world":
				st.World = "new"
				s.Message(currentConn(t, s), &st)
			case "dead":
				st.Phase = protocol.PhaseDead
				s.Message(currentConn(t, s), &st)
			case "menu":
				st.Phase = protocol.PhaseMenu
				s.Message(currentConn(t, s), &st)
			case "logout":
				if err = s.Logout(context.Background()); err != nil {
					t.Fatal(err)
				}
				p.Next()
			case "reconnect":
				s.Connected(currentConn(t, s))
			}
			if lease.Valid() {
				t.Fatal("lease survived", change)
			}
			if _, err = lease.Dispatch(context.Background(), Order{Task: protocol.TaskStop}); !errors.Is(err, ErrBodyBusy) {
				t.Fatal(err)
			}
		})
	}
}

func TestConcurrentOwnerStopAlwaysWinsWireOrder(t *testing.T) {
	s, _, p := setup(t, []string{"stop", "move_to"}, nil)
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "test", HP: 25})
	waitFor(t, func() bool { return s.State() != nil })
	for range 20 {
		lease, err := s.TryAutomation()
		if err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		var autoErr, errorStop error
		var auto, stop *Task
		go func() {
			defer wg.Done()
			<-start
			pos := protocol.Vec3{3, 0, 0}
			auto, autoErr = lease.Dispatch(context.Background(), Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}}, TimeoutS: 10})
		}()
		go func() {
			defer wg.Done()
			<-start
			stop, errorStop = s.Dispatch(context.Background(), Order{Task: protocol.TaskStop})
		}()
		close(start)
		wg.Wait()
		if errorStop != nil {
			t.Fatal(errorStop)
		}
		first := p.NextTask()
		if autoErr == nil {
			if first.ID != auto.ID || p.NextTask().ID != stop.ID {
				t.Fatal("automation overtook owner stop")
			}
			p.Report(auto.ID, protocol.StatusCancelled, protocol.ReasonStopped)
		} else if !errors.Is(autoErr, ErrBodyBusy) || first.ID != stop.ID {
			t.Fatal(autoErr, first)
		}
		p.Report(stop.ID, protocol.StatusSucceeded, "")
		waitFor(t, func() bool { s.mu.Lock(); defer s.mu.Unlock(); return len(s.tasks) == 0 })
	}
}

func TestAutomationCannotAcquireUsingAnOldWorldObservation(t *testing.T) {
	s, _, p := setup(t, []string{"stop"}, nil)
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "old", HP: 25})
	waitFor(t, func() bool { return s.State() != nil })
	hello, _, _ := s.Handshake()
	world := s.State().World
	next := *s.State()
	next.Seq++
	next.World = "new"
	s.Message(currentConn(t, s), &next)
	if _, err := s.TryAutomationFor(hello, world); !errors.Is(err, ErrBodyBusy) {
		t.Fatal("acquired from stale world", err)
	}
}

// Hold the old socket's callback until its replacement has been installed.
// This exercises the ordering that an ordinary reconnect test can miss.
type heldDisconnect struct {
	*Session
	once    sync.Once
	blocked chan struct{}
	release chan struct{}
}

func (h *heldDisconnect) Disconnected(c *bridge.Conn, err error) {
	h.once.Do(func() { close(h.blocked); <-h.release })
	h.Session.Disconnected(c, err)
}
func TestReplacementConnectionClearsOldPendingTasksBeforeItsDisconnect(t *testing.T) {
	l := &listener{}
	s := New(l, slog.New(slog.NewTextHandler(io.Discard, nil)))
	h := &heldDisconnect{Session: s, blocked: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(func() {
		select {
		case <-h.release:
		default:
			close(h.release)
		}
	})
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome {
		return &protocol.Welcome{AllowedTasks: []string{"stop", "move_to"}}
	}}, h)
	old := bridgetest.Connect(t, url, nil)
	old.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "test", HP: 25})
	waitFor(t, func() bool { return s.State() != nil })
	before, _, _ := s.Handshake()
	lease, err := s.TryAutomation()
	if err != nil {
		t.Fatal(err)
	}
	pos := protocol.Vec3{2, 0, 0}
	task, err := lease.Dispatch(context.Background(), Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}}, TimeoutS: 10})
	if err != nil {
		t.Fatal(err)
	}
	old.NextTask()
	next := bridgetest.Connect(t, url, nil)
	waitFor(t, func() bool { hello, _, ok := s.Handshake(); return ok && hello != before })
	next.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "test", HP: 25})
	waitFor(t, func() bool { return s.State() != nil })
	select {
	case <-task.Done():
	default:
		t.Fatal("replacement left old body task pending")
	}
	if lease.Valid() {
		t.Fatal("old lease still owns new session")
	}
	replacement, err := s.TryAutomation()
	if err != nil {
		t.Fatal("new idle session remains blocked", err)
	}
	replacement.Release()
	close(h.release)
	waitFor(t, func() bool { return len(l.finished()) == 1 })
	status, reason, _ := task.Status()
	if status != protocol.StatusCancelled || reason != protocol.ReasonStopped {
		t.Fatal(status, reason)
	}
	next.Quiet(20 * time.Millisecond)
}
