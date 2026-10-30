package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/protocol"
)

func TestPlayerProfilesAuthorizeCommandsAndRouteMarcoWithoutVoice(t *testing.T) {
	cfg := config.Default()
	cfg.Owners = []string{"Nimah"}
	cfg.Nicknames = []string{"Tas"}
	a := New(cfg, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil)
	if e := a.Settings.Set("players.profiles", json.RawMessage(`{"Friend":"companion","Pilot":"operator"}`), "panel"); e != nil {
		t.Fatal(e)
	}
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome { return Welcome(cfg, a.Settings.Values()) }}, a.Session)
	p := bridgetest.Connect(t, url, []string{"say", "follow", "stop"})
	if p.Welcome.SharePosition == nil || *p.Welcome.SharePosition {
		t.Fatal(p.Welcome)
	}
	p.Send(&protocol.State{Envelope: protocol.Envelope{T: 100}, World: "one", Phase: "in_world", Character: "Tasleroch", HP: 25})
	until := time.Now().Add(time.Second)
	for a.Session.State() == nil && time.Now().Before(until) {
		time.Sleep(time.Millisecond)
	}
	send := func(name, text string) {
		p.Send(&protocol.Chat{Envelope: protocol.Envelope{T: 100}, From: protocol.ChatSender{Name: name}, Channel: "shout", Text: text, Pos: protocol.Vec3{100, 0, 0}})
	}
	send("Friend", "Tas follow me")
	refused := p.NextTask()
	if refused.Task != "say" {
		t.Fatal(refused)
	}
	p.Report(refused.ID, "succeeded", "")
	send("Pilot", "Tas follow me")
	follow := p.NextTask()
	if follow.Task != "follow" {
		t.Fatal(follow)
	}
	speech := p.NextTask()
	p.Report(speech.ID, "succeeded", "")
	revision := a.Session.ControlRevision()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.Presence.Run(ctx)
	send("Friend", "Tas Marco")
	var reply *protocol.TaskRequest
	reply = p.NextTask()
	if reply.Task != "say" || !strings.Contains(string(reply.Args), "Polo!") || !strings.Contains(string(reply.Args), "100 m east") {
		t.Fatal(reply)
	}
	if a.Session.ControlRevision() != revision {
		t.Fatal("routine replaced body control")
	}
	if e := a.Settings.Set("presence.sharePosition", json.RawMessage(`true`), "panel"); e != nil {
		t.Fatal(e)
	}
	c, ok := p.Next().(*protocol.Config)
	if !ok || c.SharePosition == nil || !*c.SharePosition {
		t.Fatal(c)
	}
}
