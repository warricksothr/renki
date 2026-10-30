package session

import (
	"context"
	"errors"
	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/protocol"
	"testing"
	"time"
)

func TestChatFromReplacedConnectionCannotReachRoutines(t *testing.T) {
	s, l, _ := setup(t, nil, nil)
	old := currentConn(t, s)
	newer := &bridge.Conn{Hello: &protocol.Hello{}, Welcome: &protocol.Welcome{}}
	s.Connected(newer)
	s.Message(old, &protocol.Chat{Text: "Tas Marco"})
	s.Message(newer, &protocol.Chat{Text: "Polo"})
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.chats) != 1 || l.chats[0].Text != "Polo" {
		t.Fatal(l.chats)
	}
}

type scopedReply struct{ change func() }

func (s scopedReply) MarshalJSON() ([]byte, error) {
	s.change()
	return []byte(`{"text":"Polo!","channel":"shout"}`), nil
}
func TestScopedReplyCannotCrossWorldChangeDuringArgumentEncoding(t *testing.T) {
	s, _, p := setup(t, []string{"say"}, []string{"say"})
	p.Send(&protocol.State{Phase: "in_world", World: "one", HP: 25})
	waitFor(t, func() bool { return s.State() != nil })
	hello, _, _ := s.Handshake()
	conn := currentConn(t, s)
	args := scopedReply{change: func() {
		s.Message(conn, &protocol.State{Envelope: protocol.Envelope{Seq: 100}, Phase: "in_world", World: "two", HP: 25})
	}}
	if _, e := s.DispatchFor(context.Background(), Order{Task: "say", Args: args}, hello, "one"); !errors.Is(e, ErrBodyBusy) {
		t.Fatal(e)
	}
	p.Quiet(20 * time.Millisecond)
}
