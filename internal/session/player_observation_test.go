package session

import (
	"math"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/protocol"
)

func TestPlayerChatObservationRejectsStaleUnlocatedAndReplayedMessages(t *testing.T) {
	for _, scenario := range []string{"valid", "replaced", "no world", "menu", "stale state", "stale chat", "NaN time", "ping", "panel", "old seq", "replay"} {
		t.Run(scenario, func(t *testing.T) {
			s, _, _ := setup(t, nil, nil)
			conn := currentConn(t, s)
			now := time.Now().UTC()
			s.mu.Lock()
			s.state = &protocol.State{Envelope: protocol.Envelope{T: 100, Seq: 20}, Phase: "in_world", World: "one", HP: 25}
			s.stateAt = now
			s.mu.Unlock()
			c := &protocol.Chat{Envelope: protocol.Envelope{T: 100, Seq: 21}, Channel: "normal", From: protocol.ChatSender{Name: "Nimah"}}
			switch scenario {
			case "replaced":
				conn = &bridge.Conn{Hello: &protocol.Hello{}}
			case "no world":
				s.state.World = ""
			case "menu":
				s.state.Phase = "menu"
			case "stale state":
				now = now.Add(4 * time.Second)
			case "stale chat":
				c.T = 90
			case "NaN time":
				c.T = math.NaN()
			case "ping":
				c.Channel = "ping"
			case "panel":
				c.Channel = "panel"
			case "old seq":
				c.Seq = 19
			case "replay":
				s.ObservePlayerChat(conn, c, now)
			}
			observation, ok := s.ObservePlayerChat(conn, c, now)
			if ok != (scenario == "valid") {
				t.Fatal(observation, ok)
			}
			if ok && (observation.World != "one" || observation.Hello != conn.Hello || !observation.At.Equal(now)) {
				t.Fatal(observation)
			}
		})
	}
}
