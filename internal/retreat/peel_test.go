package retreat

import (
	"context"
	"testing"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestNativePeelOwnsDrawingButCannotSuppressConvergenceOrStaleIdentity(t *testing.T) {
	for _, mode := range []string{"draw", "close", "stale", "fight"} {
		t.Run(mode, func(t *testing.T) {
			f := fighting()
			f.st.Combat.Mode = "kite"
			f.st.Combat.Peel = &protocol.PeelStatus{Phase: "lead", Members: 2}
			f.seen.Characters[0].Pos = protocol.Vec3{8, 30, 0}
			other := protocol.PerceivedCharacter{ID: "other", Kind: protocol.CharacterCreature, Prefab: "Greyling", HPFrac: 1, Hostile: true, TargetingMe: true, Pos: protocol.Vec3{12, 30, 0}}
			if mode == "close" {
				other.Pos[0] = 5
			}
			if mode == "stale" {
				f.st.Combat.ID = "old"
			}
			if mode == "fight" {
				f.st.Combat.Peel.Phase = "fight"
				f.st.Combat.Mode = "strike"
			}
			f.seen.Characters = append(f.seen.Characters, other)
			r := New(f, func() []string { return nil }, nil, quiet, nil)
			r.Event(&protocol.Event{Event: protocol.EventDamaged, By: "other", Amount: 1})
			r.Tick(context.Background())
			if (len(f.sent) == 0) != (mode == "draw") {
				t.Fatal(mode, f.sent)
			}
		})
	}
}
