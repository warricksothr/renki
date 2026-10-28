package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/warricksothr/renki/internal/protocol"
)

// recorder is a Handler that keeps what it was told.
type recorder struct {
	mu        sync.Mutex
	connected []*Conn
	messages  []protocol.Message
	gone      chan error
}

func newRecorder() *recorder { return &recorder{gone: make(chan error, 4)} }

func (r *recorder) Connected(c *Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connected = append(r.connected, c)
}

func (r *recorder) Message(c *Conn, m protocol.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages = append(r.messages, m)
}

func (r *recorder) Disconnected(c *Conn, err error) { r.gone <- err }

func (r *recorder) received() []protocol.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]protocol.Message(nil), r.messages...)
}

func start(t *testing.T, opts Options) (*recorder, string) {
	t.Helper()
	if opts.Welcome == nil {
		opts.Welcome = func(*protocol.Hello) *protocol.Welcome {
			return &protocol.Welcome{AgentName: "Bjorn", Owners: []string{"Sothr"},
				Rates: protocol.Rates{StateHz: 5, PerceptionHz: 1}, AllowedTasks: []string{"stop"}}
		}
	}
	opts.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := newRecorder()
	srv := httptest.NewServer(NewServer(opts, rec))
	t.Cleanup(srv.Close)
	return rec, "ws" + strings.TrimPrefix(srv.URL, "http") + Path
}

func dial(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ws.CloseNow() })
	return ws
}

func send(t *testing.T, ws *websocket.Conn, frame string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := ws.Write(ctx, websocket.MessageText, []byte(frame)); err != nil {
		t.Fatal(err)
	}
}

// next reads frames until one that is not a ping arrives.
func next(t *testing.T, ws *websocket.Conn) protocol.Message {
	t.Helper()
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, data, err := ws.Read(ctx)
		cancel()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		m, err := protocol.Decode(data, protocol.FromController)
		if err != nil {
			t.Fatalf("decode %s: %v", data, err)
		}
		if _, ok := m.(*protocol.Ping); !ok {
			return m
		}
	}
}

func closeCode(t *testing.T, ws *websocket.Conn) websocket.StatusCode {
	t.Helper()
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, _, err := ws.Read(ctx)
		cancel()
		if err != nil {
			return websocket.CloseStatus(err)
		}
	}
}

const hello = `{"v":1,"type":"hello","seq":1,"t":12.5,"token":"sekrit","plugin":"0.1.0","game":"0.221.12",
	"phase":"in_world","character":"Bjorn","world":"Midgard","capabilities":["stop","follow"]}`

func TestHandshake(t *testing.T) {
	rec, url := start(t, Options{Token: "sekrit"})
	ws := dial(t, url)
	send(t, ws, hello)
	w, ok := next(t, ws).(*protocol.Welcome)
	if !ok {
		t.Fatal("first reply is not welcome")
	}
	if w.AgentName != "Bjorn" || w.Seq != 1 || w.T != 12.5 {
		t.Errorf("welcome = %+v; want Bjorn, seq 1, and the plugin's game time", w)
	}
	send(t, ws, `{"v":1,"type":"chat","seq":2,"t":13,"from":{"name":"Sothr","playerId":"1","owner":true},"channel":"normal","text":"hi","pos":[0,0,0],"dist":1}`)
	waitFor(t, func() bool { return len(rec.received()) == 1 })
	if c, ok := rec.received()[0].(*protocol.Chat); !ok || c.Text != "hi" {
		t.Errorf("handler got %#v", rec.received()[0])
	}
	if rec.connected[0].Hello.Character != "Bjorn" {
		t.Errorf("Conn.Hello not set")
	}
}

func TestBadTokenCloses4401(t *testing.T) {
	_, url := start(t, Options{Token: "other"})
	ws := dial(t, url)
	send(t, ws, hello)
	if code := closeCode(t, ws); code != CloseBadToken {
		t.Errorf("close code %d; want %d", code, CloseBadToken)
	}
}

func TestUnknownMajorVersionIsRefused(t *testing.T) {
	_, url := start(t, Options{})
	ws := dial(t, url)
	send(t, ws, strings.Replace(hello, `"v":1`, `"v":2`, 1))
	e, ok := next(t, ws).(*protocol.Error)
	if !ok || e.Code != "unsupported_version" {
		t.Fatalf("got %#v; want an unsupported_version error", e)
	}
	if code := closeCode(t, ws); code != CloseBadHello {
		t.Errorf("close code %d; want %d", code, CloseBadHello)
	}
}

func TestFirstMessageMustBeHello(t *testing.T) {
	_, url := start(t, Options{})
	ws := dial(t, url)
	send(t, ws, `{"v":1,"type":"state","seq":1,"t":1}`)
	if _, ok := next(t, ws).(*protocol.Error); !ok {
		t.Fatal("want an error reply")
	}
	if code := closeCode(t, ws); code != CloseBadHello {
		t.Errorf("close code %d; want %d", code, CloseBadHello)
	}
}

func TestUnknownTypeGetsErrorAndStaysOpen(t *testing.T) {
	rec, url := start(t, Options{})
	ws := dial(t, url)
	send(t, ws, hello)
	next(t, ws)
	send(t, ws, `{"v":1,"type":"dance","id":"d-1","seq":2,"t":1}`)
	e, ok := next(t, ws).(*protocol.Error)
	if !ok || e.Code != protocol.CodeBadMessage || e.Re != "d-1" {
		t.Fatalf("got %#v; want bad_message re d-1", e)
	}
	send(t, ws, `{"v":1,"type":"event","seq":3,"t":2,"event":"died"}`)
	waitFor(t, func() bool { return len(rec.received()) == 1 })
}

func TestPingsAndPongs(t *testing.T) {
	rec, url := start(t, Options{PingInterval: 20 * time.Millisecond})
	ws := dial(t, url)
	send(t, ws, hello)
	next(t, ws)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var ping protocol.Ping
		if json.Unmarshal(data, &ping) == nil && ping.Type == protocol.TypePing {
			send(t, ws, `{"v":1,"type":"pong","seq":9,"t":3,"re":"`+ping.ID+`"}`)
			break
		}
	}
	time.Sleep(50 * time.Millisecond)
	if n := len(rec.received()); n != 0 {
		t.Errorf("the handler saw %d messages; pongs are the bridge's own", n)
	}
}

func TestSilentPluginIsDeclaredDead(t *testing.T) {
	rec, url := start(t, Options{DeadAfter: 100 * time.Millisecond, PingInterval: time.Hour})
	ws := dial(t, url)
	send(t, ws, hello)
	next(t, ws)
	select {
	case err := <-rec.gone:
		if err == nil || !strings.Contains(err.Error(), "no message") {
			t.Errorf("disconnect reason %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("silent connection was never noticed")
	}
}

func TestNewConnectionReplacesOld(t *testing.T) {
	_, url := start(t, Options{})
	old := dial(t, url)
	send(t, old, hello)
	next(t, old)
	fresh := dial(t, url)
	send(t, fresh, hello)
	next(t, fresh)
	if code := closeCode(t, old); code != CloseReplaced {
		t.Errorf("old connection closed with %d; want %d", code, CloseReplaced)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal(errors.New("condition never held"))
		}
		time.Sleep(5 * time.Millisecond)
	}
}
