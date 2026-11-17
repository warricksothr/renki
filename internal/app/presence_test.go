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
	"github.com/warricksothr/renki/internal/session"
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

func TestNightInvitationKeepsAnActiveBodyOrderAndControlRevision(t *testing.T) {
	cfg := config.Default()
	cfg.Presence.Bedtime = protocol.ChannelShout
	a := New(cfg, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil)
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome { return Welcome(cfg, a.Settings.Values()) }}, a.Session)
	p := bridgetest.Connect(t, url, []string{protocol.TaskSay, protocol.TaskFollow})
	others := 1
	p.Send(&protocol.State{Envelope: protocol.Envelope{T: 100}, World: "one", WorldUID: "123", Character: "Tas", Phase: protocol.PhaseInWorld, HP: 25, OtherPlayers: &others, Time: protocol.GameTime{Day: 213, DayFraction: .8, CalendarFraction: clockFraction(.9), Night: true}})
	waitFor(t, func() bool { return a.Session.State() != nil })
	follow, err := a.Session.Dispatch(context.Background(), session.Order{Task: protocol.TaskFollow, Args: protocol.FollowArgs{Player: "Nimah"}})
	if err != nil {
		t.Fatal(err)
	}
	p.NextTask()
	revision := a.Session.ControlRevision()
	a.Presence.Tick(context.Background())
	invited := p.NextTask()
	if invited.Task != protocol.TaskSay || !strings.Contains(string(invited.Args), "go to bed") {
		t.Fatal(invited)
	}
	if a.Session.ControlRevision() != revision {
		t.Fatal("invitation changed body ownership")
	}
	if status, _, _ := follow.Status(); protocol.Final(status) {
		t.Fatal("invitation cancelled follow")
	}
	a.Presence.Tick(context.Background())
	p.Quiet(50 * time.Millisecond)
}

func clockFraction(n float64) *float64 { return &n }
