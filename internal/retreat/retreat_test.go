package retreat

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
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

func fighting() *fake {
	f := scene(25)
	f.st.World, f.st.Stamina = "one", 50
	f.seen.Characters = f.seen.Characters[1:2]
	f.seen.Characters[0].HPFrac = 1
	f.st.Task = &protocol.TaskRef{ID: "a-1", Task: protocol.TaskAttack, Status: protocol.StatusRunning}
	f.st.Combat = &protocol.CombatStatus{ID: "a-1", Entity: "0:7", Mode: "block"}
	return f
}

func TestHealthyExplicitMeleeIsNotCancelledOnEveryConfirmedHit(t *testing.T) {
	f := fighting()
	r := New(f, func() []string { return nil }, nil, quiet, nil)
	r.Event(&protocol.Event{Event: protocol.EventDamaged, By: "0:7", Amount: 1})
	r.Tick(context.Background())
	if len(f.sent) != 0 {
		t.Fatal(f.sent)
	}
	// The same confirmed attacker now requires escape, even above old Low.
	f.st.HP = 19
	r.Tick(context.Background())
	if len(f.sent) != 2 || !strings.HasPrefix(f.sent[0], "flee 0:7") {
		t.Fatal(f.sent)
	}
}

func TestUnsafeOrStaleAttackCannotSuppressRetreat(t *testing.T) {
	for _, change := range []string{"stamina", "stale id", "unknown mode", "stronger target", "additional attacker", "stop"} {
		t.Run(change, func(t *testing.T) {
			f := fighting()
			r := New(f, func() []string { return nil }, nil, quiet, nil)
			r.Event(&protocol.Event{Event: protocol.EventDamaged, By: "0:7", Amount: 1})
			switch change {
			case "stamina":
				f.st.Stamina = 9
			case "stale id":
				f.st.Combat.ID = "old"
			case "unknown mode":
				f.st.Combat.Mode = "unknown"
			case "stronger target":
				f.seen.Characters[0].Stars = 1
			case "additional attacker":
				f.seen.Characters = append(f.seen.Characters, protocol.PerceivedCharacter{ID: "0:99", Kind: protocol.CharacterCreature, Prefab: "Greyling", Hostile: true, HPFrac: 1, Pos: protocol.Vec3{3, 30, 0}})
			case "stop":
				f.st.Task = nil
			}
			r.Tick(context.Background())
			if len(f.sent) != 2 || !strings.HasPrefix(f.sent[0], "flee 0:7") {
				t.Fatal(f.sent)
			}
		})
	}
}

func TestPluginFallbackFleeIsNotRepeatedlyReplaced(t *testing.T) {
	f := fighting()
	f.st.HP, f.st.Stamina, f.st.Combat.Mode = 5, 5, "retreat"
	r := New(f, func() []string { return nil }, nil, quiet, nil)
	r.Event(&protocol.Event{Event: protocol.EventDamaged, By: "0:7", Amount: 3})
	r.Tick(context.Background())
	if len(f.sent) != 0 {
		t.Fatal(f.sent)
	}
}

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

	// At the owner's side with the threat still close: keep getting away.
	f.sent, f.st.Pos = nil, protocol.Vec3{-17, 30, 4}
	f.seen.Characters[1].Pos = protocol.Vec3{-14, 30, 4}
	c.t = c.t.Add(Again + time.Second)
	r.Tick(ctx)
	if want := []string{"flee 0:7 25 toward nowhere", "say: Too much for me. Getting away from the greyling."}; !slices.Equal(f.sent, want) {
		t.Fatalf("waited beside an owner in danger: %v", f.sent)
	}

	// Home, with the creature come after it but home not where it stood:
	// straight away.
	f.seen.Characters, f.sent = f.seen.Characters[:3], nil
	f.st.Pos, f.seen.Characters[1].Pos = protocol.Vec3{-38, 31, 0}, protocol.Vec3{-23.5, 30, 0}
	c.t = c.t.Add(Again + time.Second)
	r.Tick(ctx)
	if want := []string{"flee 0:7 25 toward nowhere", "say: Too much for me. Getting away from the greyling."}; !slices.Equal(f.sent, want) {
		t.Fatalf("at home: sent %q\nwant %q", f.sent, want)
	}
}

func TestRemoteAttackerWithoutTargetingFlag(t *testing.T) {
	c := &clock{time.Unix(1000, 0)}
	// Positions, health and missing targetingMe from the fatal 17:15:00Z
	// scene. The owner is visible but past the two attackers.
	f := &fake{
		st: &protocol.State{Phase: protocol.PhaseInWorld, World: "DizzysHellheim", HP: 5.4, HPMax: 25, Pos: protocol.Vec3{-28.81, 34.47, -0.48}},
		seen: &protocol.Perception{Characters: []protocol.PerceivedCharacter{
			{ID: "709168501:274", Kind: protocol.CharacterCreature, Prefab: "Greyling", Pos: protocol.Vec3{-28.5, 34.03, 2.54}, Hostile: true},
			{ID: "709168501:983", Kind: protocol.CharacterCreature, Prefab: "Greyling", Pos: protocol.Vec3{-27.4, 34.16, 2.16}, Hostile: true},
			{ID: "709168501:394", Kind: protocol.CharacterPlayer, Name: "Nimah", Pos: protocol.Vec3{-35.27, 36.09, 8.43}, HPFrac: .3},
		}},
	}
	r := New(f, func() []string { return []string{"Nimah"} }, nil, quiet, c.now)
	r.Tick(context.Background())
	if len(f.sent) != 0 {
		t.Fatal("unrelated hostiles caused retreat", f.sent)
	}
	r.Event(&protocol.Event{Event: protocol.EventDamaged, By: "709168501:274", Amount: 5})
	r.Tick(context.Background())
	if len(f.sent) != 2 || f.sent[0] != "flee 709168501:274 25 toward nowhere" {
		t.Fatalf("did not escape the known attacker: %v", f.sent)
	}
}

func TestFirstConfirmedHitRetreatsBeforeLowHealth(t *testing.T) {
	c := &clock{time.Unix(1000, 0)}
	f := scene(20.4)
	f.st.World = "DizzysHellheim"
	f.seen.Characters[0].TargetingMe = false
	r := New(f, func() []string { return nil }, nil, quiet, c.now)
	r.Tick(context.Background())
	if len(f.sent) != 0 {
		t.Fatal("healthy targeting alone triggered retreat")
	}
	// The real harvesting incident had remote damage without targetingMe.
	// Prefer that actual attacker over a closer AI that merely targets Tas.
	r.Event(&protocol.Event{Event: protocol.EventDamaged, By: "0:9", Amount: 4.6})
	r.Tick(context.Background())
	if len(f.sent) != 2 || !strings.HasPrefix(f.sent[0], "flee 0:9 ") {
		t.Fatalf("first hit did not trigger escape: %v", f.sent)
	}
	// No repeat from the same stored injury. A genuinely new hit after a
	// completed flee must not wait out ten seconds while the bot is hurt.
	r.Tick(context.Background())
	if len(f.sent) != 2 {
		t.Fatalf("stored injury repeated: %v", f.sent)
	}
	c.t = c.t.Add(2 * time.Second)
	r.Event(&protocol.Event{Event: protocol.EventDamaged, By: "0:9", Amount: 3})
	f.st.Task = &protocol.TaskRef{Task: protocol.TaskFlee, Status: protocol.StatusRunning}
	r.Tick(context.Background())
	if len(f.sent) != 2 {
		t.Fatalf("replaced a running escape: %v", f.sent)
	}
	f.st.Task = nil
	r.Tick(context.Background())
	if len(f.sent) != 4 {
		t.Fatalf("fresh hit waited through cooldown: %v", f.sent)
	}
}

func TestRecentDamageIsBoundedAndCleared(t *testing.T) {
	for _, tc := range []string{"expiry", "world", "died", "respawned", "environment", "player", "gone", "far"} {
		t.Run(tc, func(t *testing.T) {
			c := &clock{time.Unix(1000, 0)}
			f := scene(5)
			f.st.World = "world-a"
			f.seen.Characters[1].TargetingMe = false
			r := New(f, func() []string { return nil }, nil, quiet, c.now)
			by := "0:7"
			if tc == "environment" {
				by = ""
			}
			r.Event(&protocol.Event{Event: protocol.EventDamaged, By: by, Amount: 5})
			switch tc {
			case "expiry":
				c.t = c.t.Add(HurtFor)
			case "world":
				f.st.World = "world-b"
			case "died":
				r.Event(&protocol.Event{Event: protocol.EventDied})
			case "respawned":
				r.Event(&protocol.Event{Event: protocol.EventRespawned})
			case "player":
				f.seen.Characters[1].Kind = protocol.CharacterPlayer
			case "gone":
				f.seen.Characters = f.seen.Characters[:1]
			case "far":
				f.seen.Characters[1].Pos[0] = 30
			}
			r.Tick(context.Background())
			if len(f.sent) != 0 {
				t.Fatal("stale or unrelated damage caused retreat", f.sent)
			}
		})
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

func TestNoRunningThroughTheThreat(t *testing.T) {
	c := &clock{time.Unix(1000, 0)}
	f := scene(5)
	// Home past the Greyling (at 4, 0) but clear of it: not that way.
	r := New(f, func() []string { return nil }, func() (protocol.Vec3, bool) { return protocol.Vec3{40, 30, 3}, true }, quiet, c.now)
	r.Tick(context.Background())
	if want := []string{"flee 0:7 25 toward nowhere", "say: Too much for me. Getting away from the greyling."}; !slices.Equal(f.sent, want) {
		t.Fatalf("sent %q\nwant %q", f.sent, want)
	}
	// Home too far to run to.
	f.sent, c.t = nil, c.t.Add(Again+time.Second)
	r.home = func() (protocol.Vec3, bool) { return protocol.Vec3{-400, 30, 0}, true }
	r.Tick(context.Background())
	if want := []string{"flee 0:7 25 toward nowhere", "say: Too much for me. Getting away from the greyling."}; !slices.Equal(f.sent, want) {
		t.Fatalf("far from home: sent %q\nwant %q", f.sent, want)
	}
	if PastThreat(protocol.Vec3{}, protocol.Vec3{5, 0, 0}, protocol.Vec3{0, 0, 30}) {
		t.Error("off to the side counted as past the threat")
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
