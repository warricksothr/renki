package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func TestPlayerRecorderRetainsFailedBatchAndRecentReferenceNeverRevives(t *testing.T) {
	now := time.Now().UTC()
	folder := filepath.Join(t.TempDir(), "data")
	store, e := home.Open(filepath.Join(folder, "memory.json"))
	if e != nil {
		t.Fatal(e)
	}
	p := &Players{Store: store}
	hello := &protocol.Hello{Character: "Tas"}
	p.ObserveChat(&protocol.Chat{From: protocol.ChatSender{Name: "Nimah"}, Pos: protocol.Vec3{100, 0, 0}}, session.PlayerChatObservation{Hello: hello, World: "one", At: now})
	if last, ok := p.Last("one", "nimah", now); !ok || last.Source != "heard" {
		t.Fatal("pending not queryable", last)
	}
	if _, ok := p.Reference("one", "Nimah", hello, now); !ok {
		t.Fatal("recent speech not usable")
	}
	if _, ok := p.Reference("two", "Nimah", hello, now); ok {
		t.Fatal("cross world reference")
	}
	if _, ok := p.Reference("one", "Nimah", &protocol.Hello{Character: "new"}, now); ok {
		t.Fatal("reconnect reference")
	}
	if _, ok := p.Reference("one", "Nimah", hello, now.Add(PlayerReferenceAge+time.Second)); ok {
		t.Fatal("stale reference")
	}
	if e = os.WriteFile(folder, []byte("blocked"), 0600); e != nil {
		t.Fatal(e)
	}
	if p.Flush(now) || len(p.pending) != 1 {
		t.Fatal("failed save lost pending")
	}
	if e = os.Remove(folder); e != nil {
		t.Fatal(e)
	}
	if !p.Flush(now) || len(p.pending) != 0 {
		t.Fatal("retry failed")
	}
	restarted := &Players{Store: store}
	if _, ok := restarted.Last("one", "Nimah", now); !ok {
		t.Fatal("history lost")
	}
	if _, ok := restarted.Reference("one", "Nimah", hello, now); ok {
		t.Fatal("disk revived a current location")
	}
}

func TestPlayerSightingsUseNewFreshFramesAndOriginalWorld(t *testing.T) {
	s := session.New(listener{}, nil)
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome { return &protocol.Welcome{} }}, s)
	bot := bridgetest.Connect(t, url, nil)
	p := &Players{Session: s, Store: home.Memory()}
	bot.Send(&protocol.State{Envelope: protocol.Envelope{T: 100}, Phase: "in_world", World: "one", HP: 25, Character: "Tas"})
	bot.Send(&protocol.Perception{Envelope: protocol.Envelope{T: 100}, Characters: []protocol.PerceivedCharacter{{Kind: "player", Name: "Nimah", Pos: protocol.Vec3{50, 0, 0}}, {Kind: "creature", Name: "Greyling"}}})
	wait(t, func() bool { h, _, _, _ := s.PerceptionObservation(); return h != nil })
	now := time.Now()
	p.Tick(now)
	first, ok := p.Last("one", "Nimah", now)
	if !ok || first.Source != "seen" {
		t.Fatal(first)
	}
	p.Tick(now.Add(time.Second))
	if len(p.pending) != 1 {
		t.Fatal("same frame refreshed", p.pending)
	}
	bot.Send(&protocol.State{Envelope: protocol.Envelope{T: 110}, Phase: "in_world", World: "two", HP: 25})
	bot.Send(&protocol.Perception{Envelope: protocol.Envelope{T: 110}, Characters: []protocol.PerceivedCharacter{{Kind: "player", Name: "Sothr"}}})
	wait(t, func() bool { _, w, _, _ := s.PerceptionObservation(); return w == "two" })
	p.Tick(time.Now().Add(5 * time.Second))
	if _, ok := p.Last("two", "Sothr", time.Now().Add(5*time.Second)); ok {
		t.Fatal("stale frame accepted")
	}
	p.Tick(time.Now())
	if _, ok := p.Last("two", "Sothr", time.Now()); !ok {
		t.Fatal("fresh frame lost")
	}
	if _, ok := p.Last("two", "Nimah", time.Now()); ok {
		t.Fatal("cross world history")
	}
	if !p.Flush(time.Now()) || len(p.Store.PlayerHistory("one", time.Now())) != 1 {
		t.Fatal("original world lost")
	}
}

func TestPlayerFlushDoesNotLoseSpeechArrivingAfterSweepTimestamp(t *testing.T) {
	now := time.Now().UTC()
	p := &Players{Store: home.Memory()}
	hello := &protocol.Hello{Character: "Tas"}
	p.ObserveChat(&protocol.Chat{From: protocol.ChatSender{Name: "Nimah"}}, session.PlayerChatObservation{Hello: hello, World: "one", At: now.Add(time.Millisecond)})
	if !p.Flush(now) || len(p.pending) != 1 {
		t.Fatal("future speech was removed", p.pending)
	}
	if !p.Flush(now.Add(time.Second)) || len(p.pending) != 0 {
		t.Fatal("deferred speech not saved")
	}
	if _, ok := p.Last("one", "Nimah", now.Add(time.Second)); !ok {
		t.Fatal("speech lost")
	}
}
