// Package bridgetest is a fake plugin for tests: it dials a bridge, does the
// handshake, and lets a test send and read protocol messages. Pings are
// answered and hidden, as the real plugin would.
package bridgetest

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/protocol"
)

// Capabilities is what the fake plugin claims by default: the M1 tasks.
var Capabilities = []string{"stop", "move_to", "follow", "look_at", "say", "emote"}

// Serve starts a bridge for h on a test server and returns its URL.
func Serve(t testing.TB, opts bridge.Options, h bridge.Handler) string {
	t.Helper()
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	srv := httptest.NewServer(bridge.NewServer(opts, h))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http") + bridge.Path
}

// Plugin is a connected fake plugin.
type Plugin struct {
	t       testing.TB
	ws      *websocket.Conn
	seq     atomic.Uint64
	in      chan protocol.Message
	Welcome *protocol.Welcome
}

// Connect dials url, sends hello with capabilities (Capabilities if nil),
// and waits for welcome.
func Connect(t testing.TB, url string, capabilities []string) *Plugin {
	t.Helper()
	return ConnectWithToken(t, url, "", capabilities)
}

// ConnectWithToken is Connect with a token in hello.
func ConnectWithToken(t testing.TB, url, token string, capabilities []string) *Plugin {
	t.Helper()
	if capabilities == nil {
		capabilities = Capabilities
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := &Plugin{t: t, ws: ws, in: make(chan protocol.Message, 64)}
	t.Cleanup(func() { ws.CloseNow() })
	go p.read()
	p.Send(&protocol.Hello{Token: token, Plugin: "test", Game: "0.221.12", Phase: protocol.PhaseInWorld,
		Character: "Bjorn", World: "Midgard", Capabilities: capabilities})
	w, ok := p.Next().(*protocol.Welcome)
	if !ok {
		t.Fatal("no welcome")
	}
	p.Welcome = w
	return p
}

func (p *Plugin) read() {
	defer close(p.in)
	for {
		_, data, err := p.ws.Read(context.Background())
		if err != nil {
			return
		}
		m, err := protocol.Decode(data, protocol.FromController)
		if err != nil {
			p.t.Errorf("fake plugin could not decode %s: %v", data, err)
			continue
		}
		if ping, ok := m.(*protocol.Ping); ok {
			p.Send(&protocol.Pong{Envelope: protocol.Envelope{Re: ping.ID}})
			continue
		}
		p.in <- m
	}
}

// Send writes m with the next seq and a game time.
func (p *Plugin) Send(m protocol.Message) {
	p.t.Helper()
	env := protocol.EnvelopeOf(m)
	env.Seq = p.seq.Add(1)
	if env.T == 0 {
		env.T = float64(env.Seq)
	}
	data, err := protocol.Encode(m)
	if err != nil {
		p.t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := p.ws.Write(ctx, websocket.MessageText, data); err != nil {
		p.t.Errorf("fake plugin send: %v", err)
	}
}

// Next returns the next message from the controller, failing the test after
// two seconds.
func (p *Plugin) Next() protocol.Message {
	p.t.Helper()
	select {
	case m, ok := <-p.in:
		if !ok {
			p.t.Fatal("connection closed")
		}
		return m
	case <-time.After(2 * time.Second):
		p.t.Fatal("nothing from the controller for 2s")
	}
	return nil
}

// NextTask returns the next task request, failing on anything else.
func (p *Plugin) NextTask() *protocol.TaskRequest {
	p.t.Helper()
	m := p.Next()
	req, ok := m.(*protocol.TaskRequest)
	if !ok {
		p.t.Fatalf("got %s; want a task", m.MessageType())
	}
	return req
}

// Quiet fails the test if the controller sends anything within d.
func (p *Plugin) Quiet(d time.Duration) {
	p.t.Helper()
	select {
	case m, ok := <-p.in:
		if ok {
			data, _ := protocol.Encode(m)
			p.t.Fatalf("controller sent %s; want nothing", data)
		}
	case <-time.After(d):
	}
}

// Report sends a status for task id.
func (p *Plugin) Report(id, status, reason string) {
	p.Send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: id}, Status: status, Reason: reason})
}

// Close drops the connection without a bye, as a crashed game would.
func (p *Plugin) Close() { p.ws.CloseNow() }
