package session

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/protocol"
)

func TestWadingCannotDispatchToOldPlugin(t *testing.T) {
	s, _, _ := setup(t, []string{protocol.TaskMoveTo}, []string{protocol.TaskMoveTo})
	pos := protocol.Vec3{1, 2, 3}
	_, err := s.Dispatch(context.Background(), Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}, Dry: true, AvoidHostiles: true, Wade: true}})
	if err == nil {
		t.Fatal("old plugin received unsupported wading mode")
	}
}

func TestWadingDispatchRetainsPolicyOnSupportingPlugin(t *testing.T) {
	s := New(&listener{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome {
		return &protocol.Welcome{AllowedTasks: []string{protocol.TaskMoveTo}}
	}}, s)
	p := bridgetest.ConnectWithFeatures(t, url, "", []string{protocol.TaskMoveTo}, []string{protocol.FeatureSupportedWading})
	waitFor(t, func() bool { _, _, ok := s.Handshake(); return ok })
	pos := protocol.Vec3{1, 2, 3}
	task, err := s.Dispatch(context.Background(), Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}, Dry: true, AvoidHostiles: true, Wade: true}})
	if err != nil {
		t.Fatal(err)
	}
	req := p.NextTask()
	decoded, err := req.DecodeArgs()
	if err != nil || req.ID != task.ID {
		t.Fatalf("invalid dispatch: %v %v", req, err)
	}
	args := decoded.(*protocol.MoveToArgs)
	if !args.Wade || !args.Dry || !args.AvoidHostiles || args.Scope != nil {
		t.Fatalf("lost wading policy: %+v", args)
	}
}
