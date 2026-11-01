package session

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestEncounterChildrenRequireCurrentCapabilitiesWelcomeAndPermit(t *testing.T) {
	for _, task := range []string{protocol.TaskMoveTo, protocol.TaskPickup, protocol.TaskInteract} {
		for _, layer := range []string{"capability", "welcome", "permit", "allowed"} {
			for _, child := range []string{protocol.TaskFlee, protocol.TaskAttack} {
				t.Run(task+"/"+layer+"/"+child, func(t *testing.T) {
					allowed := []string{task, protocol.TaskFlee, protocol.TaskAttack}
					caps := slices.Clone(allowed)
					if layer == "capability" {
						caps = slices.DeleteFunc(caps, func(s string) bool { return s == child })
					}
					if layer == "welcome" {
						allowed = slices.DeleteFunc(allowed, func(s string) bool { return s == child })
					}
					s, _, p := setup(t, allowed, caps)
					if layer == "permit" {
						s.SetPermit(func(s string) bool { return s != child })
					}
					var args any
					switch task {
					case protocol.TaskMoveTo:
						args = &protocol.MoveToArgs{To: protocol.Target{Pos: &protocol.Vec3{1, 2, 3}}, Dry: true, AvoidHostiles: true, RecoverThreats: true, DefendThreats: true}
					case protocol.TaskPickup:
						args = &protocol.PickupArgs{Entity: "1:2", RecoverThreats: true, DefendThreats: true}
					case protocol.TaskInteract:
						args = &protocol.InteractArgs{Entity: "1:2", RecoverThreats: true, DefendThreats: true}
					}
					_, err := s.Dispatch(context.Background(), Order{Task: task, Args: args})
					if layer == "allowed" {
						if err != nil {
							t.Fatal(err)
						}
						if p.NextTask().Task != task {
							t.Fatal("wrong body sent")
						}
					} else {
						if !errors.Is(err, ErrNotAllowed) {
							t.Fatal(err)
						}
						p.Quiet(10 * time.Millisecond)
					}
				})
			}
		}
	}
}
