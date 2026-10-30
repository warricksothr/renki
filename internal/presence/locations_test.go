package presence

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/memory"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func playerMemory(m *Manager) *memory.Players {
	p := &memory.Players{Store: home.Memory()}
	m.o.Players = p
	return p
}

func TestLocationReferenceRequiresRecentSameWorldAndConnectionEvidence(t *testing.T) {
	for _, scenario := range []string{"recent", "near Tas", "missing", "stale", "reconnected", "other world"} {
		t.Run(scenario, func(t *testing.T) {
			m, f, _, now := fixture(t)
			p := playerMemory(m)
			if e := p.Store.RememberPlayers([]home.PlayerEncounter{{World: "one", Name: "Sothr", Source: "heard", Pos: protocol.Vec3{200, 0, 0}, At: now.Add(-2 * time.Minute)}}, *now); e != nil {
				t.Fatal(e)
			}
			scope := session.PlayerChatObservation{Hello: f.hello, World: f.world, At: *now}
			caller := protocol.Vec3{100, 0, 0}
			switch scenario {
			case "near Tas":
				caller[0] = 2
			case "stale":
				scope.At = now.Add(-31 * time.Second)
			case "reconnected":
				scope.Hello = &protocol.Hello{Character: "old"}
			case "other world":
				scope.World = "two"
			}
			if scenario != "missing" {
				p.ObserveChat(&protocol.Chat{From: protocol.ChatSender{Name: "Friend"}, Pos: caller}, scope)
			}
			text, handled := m.PanelQuery("Friend", "where is Sothr?")
			if !handled || !strings.Contains(text, "heard from Sothr about 2 minutes ago") || !strings.Contains(text, "straight-line") {
				t.Fatal(text)
			}
			if scenario == "recent" {
				if !strings.Contains(text, "100 m east from your last known position") {
					t.Fatal(text)
				}
			} else {
				if !strings.Contains(text, "200 m east from me") {
					t.Fatal(text)
				}
			}
			if scenario != "recent" && scenario != "near Tas" && !strings.Contains(text, "don't have a recent position for you") {
				t.Fatal(text)
			}
		})
	}
}

func TestLocationQuestionsAreScopedThrottledAndIndependentOfMarco(t *testing.T) {
	for _, scenario := range []string{"normal", "whisper", "disabled Marco", "unassigned", "revoked", "reconnect", "world change", "stale request"} {
		t.Run(scenario, func(t *testing.T) {
			m, f, live, now := fixture(t)
			p := playerMemory(m)
			p.Store.RememberPlayers([]home.PlayerEncounter{{World: "one", Name: "Nimah", Source: "seen", Pos: protocol.Vec3{0, 0, 150}, At: now.Add(-time.Hour)}}, *now)
			c := shout(f, "Friend", "Tas, where is Nimah?", protocol.Vec3{})
			switch scenario {
			case "normal":
				c.Channel = "normal"
			case "whisper":
				c.Channel = "whisper"
			case "disabled Marco":
				live.Presence.MarcoPolo = false
			case "unassigned":
				c.From.Name = "Stranger"
			case "stale request":
				c.T -= 30
			}
			if !m.Chat(c) {
				t.Fatal("query went to the model")
			}
			switch scenario {
			case "revoked":
				delete(live.Players.Profiles, "Friend")
			case "reconnect":
				f.hello = &protocol.Hello{Character: "new"}
			case "world change":
				f.world = "two"
				f.state.World = "two"
			}
			m.Tick(context.Background())
			switch scenario {
			case "unassigned", "revoked", "reconnect", "world change", "stale request":
				if len(f.orders) != 0 {
					t.Fatal(f.orders)
				}
				return
			}
			if len(f.orders) != 1 || f.orders[0].Task != "say" {
				t.Fatal(f.orders)
			}
			args := f.orders[0].Args.(*protocol.SayArgs)
			if args.Channel != c.Channel || !strings.Contains(args.Text, "last saw Nimah") || !strings.Contains(args.Text, "within about 18 m") {
				t.Fatal(args)
			}
			if c.Channel == "whisper" && args.To != "Friend" {
				t.Fatal(args)
			}
			m.Chat(c)
			m.Tick(context.Background())
			if len(f.orders) != 1 {
				t.Fatal("not throttled")
			}
		})
	}
}

func TestUnknownExpiredAmbiguousAndCoarseMemoriesDoNotInventLocations(t *testing.T) {
	m, f, _, now := fixture(t)
	p := playerMemory(m)
	for _, name := range []string{"Unknown", "Expired", "OtherWorld"} {
		if name == "Expired" {
			p.Store.RememberPlayers([]home.PlayerEncounter{{World: "one", Name: name, Source: "seen", At: now.Add(-15 * 24 * time.Hour)}}, *now)
		}
		if name == "OtherWorld" {
			p.Store.RememberPlayers([]home.PlayerEncounter{{World: "two", Name: name, Source: "seen", At: *now}}, *now)
		}
		if text, _ := m.PanelQuery("Friend", "where is "+name); !strings.Contains(text, "no remembered location") {
			t.Fatal(text)
		}
	}
	p.Store.RememberPlayers([]home.PlayerEncounter{{World: "one", Name: "Sothr", Source: "heard", Pos: protocol.Vec3{1200, 0, 0}, At: now.Add(-3 * 24 * time.Hour)}, {World: "one", Name: "ALICE", Source: "seen", At: *now}, {World: "one", Name: "Alice", Source: "seen", At: *now}}, *now)
	if text, _ := m.PanelQuery("Friend", "where is Sothr"); !strings.Contains(text, "about 3 days ago") || !strings.Contains(text, "within about 354 m") {
		t.Fatal(text)
	}
	if _, ok := p.Last(f.world, "alice", *now); ok {
		t.Fatal("ambiguous name guessed")
	}
	for _, text := range []string{"where is Nimah", "Tasleroch where is Nimah", "Tas, where's Nimah?", "Tas when did you last see Nimah?"} {
		if _, ok := m.PanelQuery("Nimah", text); !ok {
			t.Fatal(text)
		}
	}
	if m.Chat(shout(f, "Friend", "where is Nimah", protocol.Vec3{})) {
		t.Fatal("unaddressed query intercepted")
	}
}
