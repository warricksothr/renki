package presence

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/access"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

type fake struct {
	hello      *protocol.Hello
	world      string
	state      *protocol.State
	seen       *protocol.Perception
	at, seenAt time.Time
	connected  bool
	orders     []session.Order
	configs    []protocol.Config
}

func (f *fake) Handshake() (*protocol.Hello, *protocol.Welcome, bool) {
	return f.hello, nil, f.connected
}
func (f *fake) StateObservation() (*protocol.State, time.Time) { return f.state, f.at }
func (f *fake) PerceptionObservation() (*protocol.Hello, string, *protocol.Perception, time.Time) {
	return f.hello, f.world, f.seen, f.seenAt
}
func (f *fake) DispatchFor(_ context.Context, o session.Order, hello *protocol.Hello, world string) (*session.Task, error) {
	if !f.connected || hello != f.hello || world != f.world {
		return nil, session.ErrBodyBusy
	}
	f.orders = append(f.orders, o)
	return nil, nil
}
func (f *fake) Configure(_ context.Context, c protocol.Config) error {
	f.configs = append(f.configs, c)
	return nil
}
func fixture(t *testing.T) (*Manager, *fake, *Live, *time.Time) {
	t.Helper()
	now := time.Now().UTC()
	hidden := false
	f := &fake{hello: &protocol.Hello{}, world: "one", connected: true, at: now, seenAt: now, state: &protocol.State{Envelope: protocol.Envelope{T: 100}, Phase: protocol.PhaseInWorld, World: "one", Character: "Tasleroch", HP: 25, SharePosition: &hidden}, seen: &protocol.Perception{Envelope: protocol.Envelope{T: 100}}}
	live := &Live{Owners: []string{"Nimah"}, Nicknames: []string{"Tas"}, Players: access.Rules{Profiles: map[string]string{"Friend": "companion", "Guest": "guest", "Pilot": "operator"}}, Presence: config.Presence{MarcoPolo: true}}
	m := New(Options{Session: f, Live: func() Live { return *live }, Now: func() time.Time { return now }, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	m.Tick(context.Background())
	return m, f, live, &now
}
func advance(f *fake, now *time.Time, d time.Duration) {
	*now = now.Add(d)
	f.at = *now
	f.seenAt = *now
	f.state.T += d.Seconds()
	f.seen.T = f.state.T
}
func shout(f *fake, name, text string, pos protocol.Vec3) *protocol.Chat {
	return &protocol.Chat{Envelope: protocol.Envelope{T: f.state.T}, From: protocol.ChatSender{Name: name, PlayerID: name}, Channel: protocol.ChannelShout, Text: text, Pos: pos}
}
func say(t *testing.T, f *fake) string {
	t.Helper()
	if len(f.orders) == 0 {
		t.Fatal("no reply")
	}
	last := f.orders[len(f.orders)-1]
	if last.Task != "say" || last.Args.(*protocol.SayArgs).Channel != "shout" {
		t.Fatal(last)
	}
	return last.Args.(*protocol.SayArgs).Text
}
func TestAddressedMarcoGivesBotToCallerDirectionWithoutMovement(t *testing.T) {
	m, f, _, _ := fixture(t)
	if !m.Chat(shout(f, "Friend", "Tas, MARCO!", protocol.Vec3{100, 200, 0})) {
		t.Fatal("not consumed")
	}
	m.Tick(context.Background())
	if got := say(t, f); !strings.Contains(got, "Polo!") || !strings.Contains(got, "100 m east of me") {
		t.Fatal(got)
	}
	if len(f.orders) != 1 {
		t.Fatal(f.orders)
	}
}
func TestJoinGraceVisibilityAndPublicPoloReverseDirection(t *testing.T) {
	m, f, _, now := fixture(t)
	m.Event(&protocol.Event{Event: protocol.EventPlayerJoined, Name: "Friend"})
	m.Tick(context.Background())
	if len(f.orders) != 0 {
		t.Fatal("no grace")
	}
	advance(f, now, 2*time.Second)
	m.Tick(context.Background())
	if got := say(t, f); got != "Marco!" {
		t.Fatal(got)
	}
	advance(f, now, 2*time.Second)
	m.Chat(shout(f, "Stranger", "Polo!", protocol.Vec3{100, 40, 0}))
	m.Tick(context.Background())
	if got := say(t, f); !strings.Contains(got, "100 m west to reach me") {
		t.Fatal(got)
	}
	advance(f, now, 2*time.Second)
	m.Chat(shout(f, "Stranger", "Polo!", protocol.Vec3{100, 40, 0}))
	m.Tick(context.Background())
	if len(f.orders) != 2 {
		t.Fatal("reply spam")
	}
	advance(f, now, 3*time.Minute)
	m.Chat(shout(f, "Other", "Polo", protocol.Vec3{100, 40, 0}))
	m.Tick(context.Background())
	if len(f.orders) != 2 {
		t.Fatal("expired Marco answered")
	}
}
func TestProfilesAndVisibilityControlGreetingsAndReplies(t *testing.T) {
	for _, scenario := range []string{"visible", "guest join", "unknown join", "unknown call", "disabled", "revoked", "left", "stale perception", "changed world", "reconnect"} {
		t.Run(scenario, func(t *testing.T) {
			m, f, live, now := fixture(t)
			if scenario == "unknown call" {
				m.Chat(shout(f, "Other", "tas Marco", protocol.Vec3{100, 0, 0}))
			} else {
				name := "Friend"
				if scenario == "guest join" {
					name = "Guest"
				}
				if scenario == "unknown join" {
					name = "Other"
				}
				m.Event(&protocol.Event{Event: protocol.EventPlayerJoined, Name: name})
			}
			switch scenario {
			case "visible":
				f.seen.Characters = []protocol.PerceivedCharacter{{Kind: "player", Name: "Friend"}}
			case "disabled":
				live.Presence.MarcoPolo = false
			case "revoked":
				delete(live.Players.Profiles, "Friend")
			case "left":
				m.Event(&protocol.Event{Event: protocol.EventPlayerLeft, Name: "Friend"})
			case "changed world":
				f.world = "two"
				f.state.World = "two"
			case "reconnect":
				f.hello = &protocol.Hello{Character: "new"}
			}
			advance(f, now, 6*time.Second)
			if scenario == "stale perception" {
				f.seenAt = now.Add(-10 * time.Second)
			}
			m.Tick(context.Background())
			if len(f.orders) != 0 {
				t.Fatal(f.orders)
			}
		})
	}
}
func TestMarkerConfigReconcilesAndStaleOrNonShoutMessagesDoNothing(t *testing.T) {
	m, f, live, now := fixture(t)
	live.Presence.SharePosition = true
	m.Tick(context.Background())
	if len(f.configs) != 1 || !*f.configs[0].SharePosition {
		t.Fatal(f.configs)
	}
	c := shout(f, "Guest", "tas Marco", protocol.Vec3{0, 0, 20})
	c.Channel = "normal"
	if m.Chat(c) {
		t.Fatal("normal chat consumed")
	}
	c.Channel = "shout"
	c.T -= 20
	m.Chat(c)
	m.Tick(context.Background())
	if len(f.orders) != 0 {
		t.Fatal("stale chat used")
	}
	advance(f, now, 4*time.Second)
	f.at = now.Add(-4 * time.Second)
	m.Chat(shout(f, "Guest", "tas Marco", protocol.Vec3{0, 0, 20}))
	m.Tick(context.Background())
	if len(f.orders) != 0 {
		t.Fatal("stalled state used")
	}
}
func TestAllCompassBearingsAndOpposites(t *testing.T) {
	for _, c := range []struct {
		pos              protocol.Vec3
		forward, reverse string
	}{{protocol.Vec3{0, 50, 100}, "north", "south"}, {protocol.Vec3{100, 0, 100}, "north-east", "south-west"}, {protocol.Vec3{100, 0, 0}, "east", "west"}, {protocol.Vec3{100, 0, -100}, "south-east", "north-west"}, {protocol.Vec3{0, 0, -100}, "south", "north"}, {protocol.Vec3{-100, 0, -100}, "south-west", "north-east"}, {protocol.Vec3{-100, 0, 0}, "west", "east"}, {protocol.Vec3{-100, 0, 100}, "north-west", "south-east"}} {
		if got := direction("Nimah", protocol.Vec3{}, c.pos, true); !strings.Contains(got, " "+c.forward+" of me") {
			t.Fatal(got)
		}
		if got := direction("Nimah", c.pos, protocol.Vec3{}, false); !strings.Contains(got, " "+c.reverse+" to reach me") {
			t.Fatal(got)
		}
	}
}
