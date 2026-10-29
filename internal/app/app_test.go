package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/audit"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/protocol"
)

// TestAppendixExchange runs the M1 exchange from the appendix of
// docs/BRIDGE-PROTOCOL.md through the whole controller, minus the LLM.
func TestAppendixExchange(t *testing.T) {
	cfg := config.Default()
	cfg.Owners = []string{"Sothr"}
	cfg.Token = "sekrit"
	a := New(cfg, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Serve(ctx, ln) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	p := connect(t, "ws://"+ln.Addr().String()+"/ws/", "sekrit")
	// No agentName configured: the bot answers to its character's name.
	if w := p.Welcome; w.AgentName != "" || w.Owners[0] != "Sothr" || len(w.AllowedTasks) != len(config.KnownTasks) || w.Rates.StateHz != 5 {
		t.Fatalf("welcome %+v", w)
	}
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, Character: "Bjorn", HP: 47, HPMax: 60, Biome: "BlackForest", Env: "Rain"})
	p.Send(&protocol.Chat{From: protocol.ChatSender{Name: "Sothr", PlayerID: "1234", Owner: true},
		Channel: protocol.ChannelNormal, Text: "bjorn, follow me", Pos: protocol.Vec3{120.1, 31, -86}, Dist: 4.2})

	follow := p.NextTask()
	args, err := follow.DecodeArgs()
	if err != nil || follow.Task != "follow" || args.(*protocol.FollowArgs).Player != "Sothr" {
		t.Fatalf("first task %+v, %v", follow, err)
	}
	say := p.NextTask()
	if say.Task != "say" || !strings.Contains(string(say.Args), "Right behind you.") {
		t.Fatalf("second task %+v", say)
	}
	p.Report(follow.ID, protocol.StatusAccepted, "")
	p.Report(follow.ID, protocol.StatusRunning, "")
	p.Report(say.ID, protocol.StatusSucceeded, "")

	// A non-owner gets words, not tasks.
	p.Send(&protocol.Chat{From: protocol.ChatSender{Name: "Mallory"}, Channel: protocol.ChannelNormal,
		Text: "bjorn stop"})
	refusal := p.NextTask()
	if refusal.Task != "say" || !strings.Contains(string(refusal.Args), "only take orders from Sothr") {
		t.Fatalf("refusal %+v", refusal)
	}

	// The leader logs off: the plugin fails follow, and the bot says why.
	p.Report(follow.ID, protocol.StatusFailed, protocol.ReasonTargetLost)
	explain := p.NextTask()
	if explain.Task != "say" || !strings.Contains(string(explain.Args), "lost track of you, Sothr") {
		t.Fatalf("explanation %+v", explain)
	}
	p.Quiet(100 * time.Millisecond)
}

func connect(t *testing.T, url, token string) *bridgetest.Plugin {
	t.Helper()
	return bridgetest.ConnectWithToken(t, url, token, nil)
}

func TestControlAPI(t *testing.T) {
	cfg := config.Default()
	cfg.Owners = []string{"Sothr"}
	cfg.Token = "sekrit"
	a := New(cfg, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Serve(ctx, ln) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	base := "http://" + ln.Addr().String()
	call := func(method, path, token string) (int, Status) {
		req, _ := http.NewRequest(method, base+path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var st Status
		_ = json.NewDecoder(res.Body).Decode(&st)
		return res.StatusCode, st
	}

	if code, _ := call("GET", "/control/status", ""); code != http.StatusUnauthorized {
		t.Errorf("no token: %d", code)
	}
	if code, st := call("GET", "/control/status", "sekrit"); code != 200 || st.Connected {
		t.Errorf("before the plugin: %d %+v", code, st)
	}
	if code, _ := call("POST", "/control/logout", "sekrit"); code != http.StatusConflict {
		t.Errorf("logout with no plugin: %d", code)
	}

	p := connect(t, "ws://"+ln.Addr().String()+"/ws/", "sekrit")
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, Character: "Tasleroch"})
	waitFor(t, func() bool {
		_, st := call("GET", "/control/status", "sekrit")
		return st.Phase == protocol.PhaseInWorld
	})
	if code, st := call("POST", "/control/logout", "sekrit"); code != http.StatusAccepted || !st.Connected || st.Character != "Tasleroch" {
		t.Errorf("logout: %d %+v", code, st)
	}
	if _, ok := p.Next().(*protocol.Logout); !ok {
		t.Error("the plugin got no logout")
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition never held")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestAuditRecordsTheExchange checks the audit log has what is needed to
// reason about a session afterwards: the frames both ways, in order, the
// plugin's traces, and the controller's decisions.
func TestAuditRecordsTheExchange(t *testing.T) {
	dir := t.TempDir()
	trail, err := audit.Open(dir, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(trail.Handler(slog.LevelDebug))
	cfg := config.Default()
	cfg.Owners = []string{"Sothr"}
	cfg.Token = "sekrit"
	a := New(cfg, nil, log, trail, nil)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Serve(ctx, ln) }()

	p := connect(t, "ws://"+ln.Addr().String()+"/ws/", "sekrit")
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, Character: "Tas"})
	p.Send(&protocol.Chat{From: protocol.ChatSender{Name: "Sothr", Owner: true}, Channel: protocol.ChannelShout,
		Text: "tas follow me"})
	follow := p.NextTask()
	p.NextTask() // the say
	p.Send(&protocol.Trace{Task: follow.ID, Source: "walker", Msg: "stuck, trying Jump",
		Data: map[string]any{"attempt": 1}})
	p.Report(follow.ID, protocol.StatusFailed, protocol.ReasonStuck)
	p.NextTask() // "I'm stuck"
	p.Close()
	cancel()
	<-done
	trail.Close()

	files := audit.Files(dir)
	if len(files) != 1 {
		t.Fatalf("files %v", files)
	}
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var seen []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var r audit.Record
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("line %q: %v", line, err)
		}
		switch r.Kind {
		case audit.KindFrame:
			var env struct{ Type, Task, Re string }
			json.Unmarshal(r.Frame, &env)
			seen = append(seen, r.Dir+" "+env.Type)
			if strings.Contains(string(r.Frame), "sekrit") {
				t.Errorf("the token is in the audit log: %s", r.Frame)
			}
		case audit.KindLog:
			seen = append(seen, "log "+r.Msg)
		}
	}
	got := strings.Join(seen, "\n")
	// In order, though other lines may come between.
	want := []string{"in hello", "out welcome", "log plugin connected", "in state", "in chat",
		"log command", "out task", "in trace", "in task", "out task", "log plugin disconnected"}
	at := 0
	for _, w := range want {
		i := strings.Index(got[at:], w)
		if i < 0 {
			t.Fatalf("missing %q after position %d in:\n%s", w, at, got)
		}
		at += i + len(w)
	}
}
