package session

import (
	"context"
	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/protocol"
	"math"
	"testing"
	"time"
)

func TestPickupIngressScopesDeduplicatesAndRejectsInference(t *testing.T) {
	s := New(&listener{}, nil)
	c := &bridge.Conn{Hello: &protocol.Hello{}, Welcome: &protocol.Welcome{}}
	s.Connected(c)
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 10, T: 10}, World: "one", WorldUID: "uid", Phase: protocol.PhaseInWorld, HP: 50})
	pos := protocol.Vec3{1, 2, 3}
	e := &protocol.Event{Envelope: protocol.Envelope{Seq: 11, T: 10.1}, Event: protocol.EventItemPickedUp, World: "one", WorldUID: "uid", Prefab: "Coal", Count: 7, Pos: &pos}
	s.Message(c, e)
	s.Message(c, e)
	_, got := s.PickupObservations()
	if len(got) != 1 || got[0].Count != 7 || got[0].WorldUID != "uid" || got[0].Pos != pos {
		t.Fatal(got)
	}
	for _, mode := range []string{"generic_gain", "stale", "future", "bad_pos", "zero", "world", "world_uid"} {
		copy := *e
		copy.Seq++
		now := time.Now()
		if mode == "generic_gain" {
			copy.Event = protocol.EventItemAdded
		}
		if mode == "stale" {
			now = now.Add(4 * time.Second)
		}
		if mode == "future" {
			copy.T = 14
		}
		if mode == "bad_pos" {
			p := protocol.Vec3{math.NaN(), 0, 0}
			copy.Pos = &p
		}
		if mode == "zero" {
			copy.Count = 0
		}
		if mode == "world" {
			copy.World = "two"
		}
		if mode == "world_uid" {
			copy.WorldUID = "foreign"
		}
		s.observePickup(c, &copy, now)
	}
	_, got = s.PickupObservations()
	if len(got) != 1 {
		t.Fatal(got)
	}
	replacement := &bridge.Conn{Hello: &protocol.Hello{}, Welcome: &protocol.Welcome{}}
	s.Connected(replacement)
	s.Message(c, e)
	_, got = s.PickupObservations()
	if len(got) != 0 {
		t.Fatal("replaced connection leaked", got)
	}
}
func TestConditionalDepositCannotBeSentToAnOldPlugin(t *testing.T) {
	s, _, _ := setup(t, []string{protocol.TaskDeposit}, []string{protocol.TaskDeposit})
	_, err := s.Dispatch(context.Background(), Order{Task: protocol.TaskDeposit, Args: &protocol.TransferArgs{Container: "a", MatchExisting: true, Items: []protocol.TransferItem{{Prefab: "Coal", Count: 1}}}})
	if err == nil {
		t.Fatal("old plugin received an unsupported destructive fallback")
	}
}
