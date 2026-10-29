package retreat

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

type fake struct {
	st   *protocol.State
	seen *protocol.Perception
	sent []string
}

func (f *fake) State() *protocol.State           { return f.st }
func (f *fake) Perception() *protocol.Perception { return f.seen }

func (f *fake) Dispatch(_ context.Context, o session.Order) (*session.Task, error) {
	switch a := o.Args.(type) {
	case *protocol.FleeArgs:
		toward := "nowhere"
		if a.Toward != nil {
			toward = a.Toward.Entity
			if a.Toward.Pos != nil {
				toward = fmt.Sprint(*a.Toward.Pos)
			}
		}
		f.sent = append(f.sent, fmt.Sprintf("flee %s %v toward %s", a.From.Entity, *a.Distance, toward))
	case *protocol.SayArgs:
		f.sent = append(f.sent, "say: "+a.Text)
	}
	return session.Finished(o.Task, nil, protocol.StatusAccepted, ""), nil
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

// The bot at home, a Greyling after it: what killed it on 2026-10-29.
func scene(hp float64) *fake {
	return &fake{
		st: &protocol.State{Phase: protocol.PhaseInWorld, Pos: protocol.Vec3{0, 30, 0}, HP: hp, HPMax: 25},
		seen: &protocol.Perception{Characters: []protocol.PerceivedCharacter{
			{ID: "0:9", Kind: protocol.CharacterCreature, Prefab: "Greyling", Pos: protocol.Vec3{12, 30, 0}, Hostile: true},
			{ID: "0:7", Kind: protocol.CharacterCreature, Prefab: "Greyling", Pos: protocol.Vec3{4, 30, 0}, Hostile: true, TargetingMe: true},
			{ID: "0:8", Kind: protocol.CharacterCreature, Prefab: "Neck", Pos: protocol.Vec3{2, 30, 0}, TargetingMe: true},
		}},
	}
}

func TestRetreatOnLowHealthWithAThreat(t *testing.T) {
	c := &clock{time.Unix(1000, 0)}
	home := protocol.Vec3{-40, 31, 0}
	f := scene(20)
	r := New(f, func() []string { return []string{"Sothr"} }, func() (protocol.Vec3, bool) { return home, true }, quiet, c.now)
	ctx := context.Background()

	r.Tick(ctx)
	if len(f.sent) != 0 {
		t.Fatalf("healthy enough, yet: %v", f.sent)
	}
	// Low: from the Greyling that has picked it, not the nearer neck that
	// is not hostile nor the one that has not; home is clear of it.
	f.st.HP = 9
	r.Tick(ctx)
	want := []string{"flee 0:7 25 toward [-40 31 0]", "say: Too much for me. Heading home."}
	if !slices.Equal(f.sent, want) {
		t.Fatalf("sent %q\nwant %q", f.sent, want)
	}
	// Not again at once, nor while the flee runs.
	r.Tick(ctx)
	c.t = c.t.Add(Again + time.Second)
	f.st.Task = &protocol.TaskRef{Task: protocol.TaskFlee, Status: protocol.StatusRunning}
	r.Tick(ctx)
	if len(f.sent) != 2 {
		t.Fatalf("again: %v", f.sent)
	}

	// An owner in sight is better than home.
	f.st.Task, f.sent = nil, nil
	f.seen.Characters = append(f.seen.Characters,
		protocol.PerceivedCharacter{ID: "1:2", Kind: protocol.CharacterPlayer, Name: "Sothr", Pos: protocol.Vec3{-20, 30, 5}, HPFrac: 1})
	r.Tick(ctx)
	if want := []string{"flee 0:7 25 toward 1:2", "say: Too much for me. Falling back to you, Sothr."}; !slices.Equal(f.sent, want) {
		t.Fatalf("sent %q\nwant %q", f.sent, want)
	}
}

func TestNoRefugeWhereTheThreatIs(t *testing.T) {
	c := &clock{time.Unix(1000, 0)}
	f := scene(5)
	// Home is where the Greyling stands, and nobody else is about.
	r := New(f, func() []string { return []string{"Sothr"} }, func() (protocol.Vec3, bool) { return protocol.Vec3{6, 30, 2}, true }, quiet, c.now)
	r.Tick(context.Background())
	if want := []string{"flee 0:7 25 toward nowhere", "say: Too much for me. Getting away from the greyling."}; !slices.Equal(f.sent, want) {
		t.Fatalf("sent %q\nwant %q", f.sent, want)
	}
}

func TestNoThreatNoRetreat(t *testing.T) {
	c := &clock{time.Unix(1000, 0)}
	f := scene(5)
	f.seen.Characters[1].Pos = protocol.Vec3{30, 30, 0} // out of Near
	r := New(f, func() []string { return nil }, nil, quiet, c.now)
	r.Tick(context.Background())
	f.st.Phase = protocol.PhaseDead
	f.seen.Characters[1].Pos = protocol.Vec3{3, 30, 0}
	r.Tick(context.Background())
	if len(f.sent) != 0 {
		t.Fatalf("sent %v", f.sent)
	}
}
