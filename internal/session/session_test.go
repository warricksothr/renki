package session

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/protocol"
)

type listener struct {
	mu    sync.Mutex
	done  []*Task
	chats []*protocol.Chat
}

func (l *listener) Chat(c *protocol.Chat) { l.mu.Lock(); l.chats = append(l.chats, c); l.mu.Unlock() }
func (l *listener) Event(*protocol.Event) {}
func (l *listener) TaskDone(t *Task)      { l.mu.Lock(); l.done = append(l.done, t); l.mu.Unlock() }
func (l *listener) finished() []*Task {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]*Task(nil), l.done...)
}

func setup(t *testing.T, allowed []string, caps []string) (*Session, *listener, *bridgetest.Plugin) {
	t.Helper()
	l := &listener{}
	s := New(l, slog.New(slog.NewTextHandler(io.Discard, nil)))
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome {
		return &protocol.Welcome{AgentName: "Bjorn", AllowedTasks: allowed}
	}}, s)
	p := bridgetest.Connect(t, url, caps)
	waitFor(t, func() bool { _, _, ok := s.Handshake(); return ok })
	return s, l, p
}

func TestDispatchAndLifecycle(t *testing.T) {
	s, l, p := setup(t, []string{"stop", "follow"}, nil)
	task, err := s.Dispatch(context.Background(), Order{Task: "follow",
		Args: &protocol.FollowArgs{Player: "Sothr"}, Origin: "chat"})
	if err != nil {
		t.Fatal(err)
	}
	req := p.NextTask()
	if req.ID != task.ID || req.Task != "follow" {
		t.Fatalf("plugin got %+v", req)
	}
	p.Report(task.ID, protocol.StatusAccepted, "")
	p.Report(task.ID, protocol.StatusRunning, "")
	waitFor(t, func() bool { st, _, _ := task.Status(); return st == protocol.StatusRunning })
	p.Report(task.ID, protocol.StatusFailed, protocol.ReasonTargetLost)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	status, reason, err := task.Wait(ctx)
	if err != nil || status != protocol.StatusFailed || reason != protocol.ReasonTargetLost {
		t.Fatalf("Wait = %s, %s, %v", status, reason, err)
	}
	waitFor(t, func() bool { return len(l.finished()) == 1 })
	if l.finished()[0].Origin != "chat" {
		t.Error("Origin was not carried")
	}
	// A late report for a finished task changes nothing.
	p.Report(task.ID, protocol.StatusSucceeded, "")
	time.Sleep(20 * time.Millisecond)
	if st, _, _ := task.Status(); st != protocol.StatusFailed {
		t.Errorf("status moved on to %s after it was final", st)
	}
}

func TestDispatchRefusesOutsideCapabilitiesAndAllowed(t *testing.T) {
	// The controller allows emote, but the plugin cannot do it; the plugin
	// can move_to, but the controller does not allow it.
	s, _, p := setup(t, []string{"stop", "emote"}, []string{"stop", "move_to"})
	for _, name := range []string{"emote", "move_to", "harvest"} {
		if _, err := s.Dispatch(context.Background(), Order{Task: name}); !errors.Is(err, ErrNotAllowed) {
			t.Errorf("%s: err %v; want ErrNotAllowed", name, err)
		}
	}
	if _, err := s.Dispatch(context.Background(), Order{Task: "stop"}); err != nil {
		t.Fatal(err)
	}
	p.NextTask()
}

func TestDispatchRefusesBadArgs(t *testing.T) {
	s, _, p := setup(t, []string{"follow"}, nil)
	_, err := s.Dispatch(context.Background(), Order{Task: "follow", Args: &protocol.FollowArgs{}})
	if !errors.Is(err, protocol.ErrBadArgs) {
		t.Fatalf("err %v; want ErrBadArgs", err)
	}
	p.Quiet(50 * time.Millisecond)
}

func TestStaleStateIsDropped(t *testing.T) {
	s, _, p := setup(t, nil, nil)
	p.Send(&protocol.State{Envelope: protocol.Envelope{Seq: 0}, HP: 10})
	p.Send(&protocol.State{HP: 20})
	waitFor(t, func() bool { st := s.State(); return st != nil && st.HP == 20 })
	// Simulate a reordered frame: lower seq than the one held.
	s.Message(currentConn(t, s), &protocol.State{Envelope: protocol.Envelope{Seq: 1}, HP: 5})
	if s.State().HP != 20 {
		t.Errorf("an older state replaced a newer one")
	}
}

func TestTheNewestPerceptionIsKept(t *testing.T) {
	s, _, p := setup(t, nil, nil)
	if s.Perception() != nil {
		t.Fatal("perception before any was sent")
	}
	p.Send(&protocol.Perception{Characters: []protocol.PerceivedCharacter{{ID: "1:2", Kind: "player", Name: "Nimah"}}})
	waitFor(t, func() bool { pc := s.Perception(); return pc != nil && len(pc.Characters) == 1 })
	s.Message(currentConn(t, s), &protocol.Perception{Envelope: protocol.Envelope{Seq: 0}})
	if len(s.Perception().Characters) != 1 {
		t.Error("an older perception replaced a newer one")
	}

	// A new connection starts its sequence again: the old snapshot goes.
	s.Connected(currentConn(t, s))
	if s.Perception() != nil {
		t.Error("the snapshot outlived a new connection")
	}
}

func TestTheNewestInventoryIsKept(t *testing.T) {
	s, _, p := setup(t, nil, nil)
	p.Send(&protocol.Inventory{Size: [2]int{8, 4}, Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 3}}})
	waitFor(t, func() bool { inv := s.Inventory(); return inv != nil && inv.Count("Wood") == 3 })
	s.Message(currentConn(t, s), &protocol.Inventory{Envelope: protocol.Envelope{Seq: 0}})
	if s.Inventory().Count("Wood") != 3 {
		t.Error("an older inventory replaced a newer one")
	}
	s.Connected(currentConn(t, s))
	if s.Inventory() != nil {
		t.Error("the inventory outlived a new connection")
	}
}

func TestChestSnapshotsAreOrderedAndScopedToConnectionAndWorld(t *testing.T) {
	s, _, p := setup(t, nil, nil)
	p.Send(&protocol.State{World: "Midgard"})
	p.Send(&protocol.Container{Entity: "1:2", Items: []protocol.InventoryItem{{Prefab: "Wood", Stack: 5}}})
	waitFor(t, func() bool { return len(s.Containers()) == 1 })
	s.Message(currentConn(t, s), &protocol.Container{Envelope: protocol.Envelope{Seq: 0}, Entity: "1:2"})
	if s.Containers()[0].Inventory().Count("Wood") != 5 {
		t.Fatal("stale chest contents replaced the newer snapshot")
	}
	p.Send(&protocol.State{World: "Other"})
	waitFor(t, func() bool { return len(s.Containers()) == 0 })
	p.Send(&protocol.Container{Entity: "1:2"})
	waitFor(t, func() bool { return len(s.Containers()) == 1 })
	s.Connected(currentConn(t, s))
	if len(s.Containers()) != 0 {
		t.Fatal("chests outlived a new connection")
	}
}

func TestDisconnectStopsTasksAndSendsNothingAfter(t *testing.T) {
	s, l, p := setup(t, []string{"follow", "say"}, nil)
	task, err := s.Dispatch(context.Background(), Order{Task: "follow", Args: &protocol.FollowArgs{Player: "Sothr"}})
	if err != nil {
		t.Fatal(err)
	}
	p.NextTask()
	p.Close()
	select {
	case <-task.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("task never ended after the disconnect")
	}
	if st, reason, _ := task.Status(); st != protocol.StatusCancelled || reason != protocol.ReasonStopped {
		t.Errorf("status %s/%s; want cancelled/stopped", st, reason)
	}
	waitFor(t, func() bool { return len(l.finished()) == 1 })
	if _, err := s.Dispatch(context.Background(), Order{Task: "say", Args: &protocol.SayArgs{Text: "hi"}}); !errors.Is(err, ErrNotConnected) {
		t.Errorf("dispatch after disconnect: %v", err)
	}
}

func currentConn(t *testing.T, s *Session) *bridge.Conn {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition never held")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
