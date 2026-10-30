package session

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestMoveWaypointResolvesBeforeWireAndDoesNotMutateCaller(t *testing.T) {
	s, _, p := setup(t, []string{protocol.TaskMoveTo}, nil)
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one"})
	waitFor(t, func() bool { return s.State() != nil })
	s.SetMovePlace(func(world, name string) (protocol.Vec3, error) {
		if world != "one" || name != "mine" {
			t.Errorf("%s/%s", world, name)
		}
		return protocol.Vec3{1, 36, 3}, nil
	})
	args := &protocol.MoveToArgs{To: protocol.Target{Entity: "mine"}}
	if _, err := s.Dispatch(context.Background(), Order{Task: protocol.TaskMoveTo, Args: args}); err != nil {
		t.Fatal(err)
	}
	req := p.NextTask()
	decoded, err := req.DecodeArgs()
	if err != nil {
		t.Fatal(err)
	}
	if got := decoded.(*protocol.MoveToArgs).To; got.Pos == nil || *got.Pos != (protocol.Vec3{1, 36, 3}) || got.Entity != "" {
		t.Fatal(got)
	}
	if args.To.Entity != "mine" || args.To.Pos != nil {
		t.Fatal("caller mutated")
	}
	if _, err := s.Dispatch(context.Background(), Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Entity: "1:9"}}}); err != nil {
		t.Fatal(err)
	}
	if got, err := p.NextTask().DecodeArgs(); err != nil || got.(*protocol.MoveToArgs).To.Entity != "1:9" {
		t.Fatal(got, err)
	}
}

func TestWaypointWorldChangeDuringResolutionSendsNothing(t *testing.T) {
	s, _, p := setup(t, []string{protocol.TaskMoveTo}, nil)
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one"})
	waitFor(t, func() bool { return s.State() != nil })
	s.SetMovePlace(func(world, name string) (protocol.Vec3, error) {
		s.Message(currentConn(t, s), &protocol.State{Envelope: protocol.Envelope{Seq: 999}, Phase: protocol.PhaseInWorld, World: "two"})
		return protocol.Vec3{1, 2, 3}, nil
	})
	_, err := s.Dispatch(context.Background(), Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Entity: "mine"}}})
	if err == nil || !strings.Contains(err.Error(), "world changed") {
		t.Fatal(err)
	}
	p.Quiet(30 * time.Millisecond)
}
