package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/chat"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
)

type unusedPlayerVoice struct{ calls atomic.Int32 }

func (v *unusedPlayerVoice) Reply(context.Context, chat.Request) (chat.Reply, error) {
	v.calls.Add(1)
	return chat.Reply{Say: "unexpected model call"}, nil
}

func TestSpeechMemoryAndDeterministicLocationQuestionsIncludingPanel(t *testing.T) {
	cfg := config.Default()
	cfg.Owners = []string{"Nimah"}
	cfg.Nicknames = []string{"Tas"}
	cfg.Presence.MarcoPolo = false
	voice := &unusedPlayerVoice{}
	a := New(cfg, voice, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil)
	if e := a.Settings.Set("voice", json.RawMessage(`true`), "panel"); e != nil {
		t.Fatal(e)
	}
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome { return Welcome(cfg, a.Settings.Values()) }}, a.Session)
	p := bridgetest.Connect(t, url, []string{"say", "follow", "stop"})
	p.Send(&protocol.State{Envelope: protocol.Envelope{T: 100}, Phase: "in_world", World: "one", HP: 25, Character: "Tasleroch"})
	until := time.Now().Add(time.Second)
	for a.Session.State() == nil && time.Now().Before(until) {
		time.Sleep(time.Millisecond)
	}
	// Neither an addressed command nor Marco/Polo: every located speech counts.
	p.Send(&protocol.Chat{Envelope: protocol.Envelope{T: 100}, From: protocol.ChatSender{Name: "Sothr"}, Channel: "normal", Text: "Building over here", Pos: protocol.Vec3{300, 0, 0}})
	for time.Now().Before(until) {
		if _, ok := a.Players.Last("one", "Sothr", time.Now()); ok {
			break
		}
		time.Sleep(time.Millisecond)
	}
	last, ok := a.Players.Last("one", "Sothr", time.Now())
	if !ok || last.Source != "heard" {
		t.Fatal("ordinary chat not remembered", last)
	}
	revision := a.Session.ControlRevision()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.Presence.Run(ctx)
	p.Send(&protocol.Chat{Envelope: protocol.Envelope{T: 100}, From: protocol.ChatSender{Name: "Nimah"}, Channel: "shout", Text: "Tas where is Sothr?", Pos: protocol.Vec3{100, 0, 0}})
	reply := p.NextTask()
	if reply.Task != "say" || !strings.Contains(string(reply.Args), "heard from Sothr") || !strings.Contains(string(reply.Args), "200 m east from your last known position") {
		t.Fatal(reply, string(reply.Args))
	}
	p.Report(reply.ID, "succeeded", "")
	if a.Session.ControlRevision() != revision {
		t.Fatal("query interrupted the body")
	}
	if e := a.panelSay("Nimah", "where is Sothr?"); e != nil {
		t.Fatal(e)
	}
	p.Quiet(30 * time.Millisecond)
	if voice.calls.Load() != 0 {
		t.Fatal("query called model")
	}
	if !a.Players.Flush(time.Now()) {
		t.Fatal("flush failed")
	}
	handler := panel.Handler(panel.Options{Token: "test", Status: func() any { return nil }, PlayerMemory: a.PlayerMemory})
	for _, authorized := range []bool{false, true} {
		r := httptest.NewRequest("GET", "/api/memory/players", nil)
		if authorized {
			r.Header.Set("Authorization", "Bearer test")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if !authorized {
			if w.Code == 200 {
				t.Fatal("unauthenticated memory exposed")
			}
			continue
		}
		var got PlayerMemoryStatus
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &got) != nil || got.World != "one" || len(got.Encounters) != 2 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
