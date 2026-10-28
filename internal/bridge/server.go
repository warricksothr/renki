// Package bridge is the controller's end of the plugin connection: the
// WebSocket server, the hello/welcome handshake, and keeping the connection
// alive (docs/BRIDGE-PROTOCOL.md sections 2 and 5).
package bridge

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"github.com/warricksothr/renki/internal/protocol"
)

// Path is where the plugin dials.
const Path = "/ws/"

// Close codes beyond the standard ones (section 2).
const (
	// CloseBadHello closes a connection whose first message is not a usable
	// `hello`, including an unsupported major version.
	CloseBadHello websocket.StatusCode = 4400
	// CloseBadToken closes a connection whose `hello` carries the wrong token.
	CloseBadToken websocket.StatusCode = 4401
	// CloseReplaced closes the older connection when a new plugin connects.
	CloseReplaced websocket.StatusCode = 4409
)

// Handler receives what happens on the connection. Calls for one connection
// arrive in order, on that connection's read goroutine, so a slow handler
// delays reading.
type Handler interface {
	// Connected is called after `welcome` has gone out.
	Connected(c *Conn)
	// Message is called for every decoded message after `hello`, except
	// `pong`, which the bridge consumes.
	Message(c *Conn, m protocol.Message)
	// Disconnected is called once, when the connection is gone.
	Disconnected(c *Conn, err error)
}

// Options configure a Server.
type Options struct {
	// Token is what `hello` must carry. Empty accepts any token.
	Token string
	// Welcome builds the reply to a hello.
	Welcome func(*protocol.Hello) *protocol.Welcome
	// HelloTimeout bounds the wait for `hello` after the socket opens.
	HelloTimeout time.Duration
	// PingInterval is how often the controller pings. It must stay well
	// under the plugin's 10 s dead-man limit.
	PingInterval time.Duration
	// DeadAfter is how long the controller waits for any message before it
	// gives up on the connection.
	DeadAfter time.Duration
	Logger    *slog.Logger
	// Fallback serves every path but Path, such as the controller's control
	// API. Nil answers them with 404.
	Fallback http.Handler
	// Tap sees every frame as it went over the wire, received ("in") or
	// sent ("out"), such as for the audit log. It must not keep frame.
	Tap func(conn uint64, dir string, frame []byte)
}

func (o *Options) defaults() {
	if o.HelloTimeout == 0 {
		o.HelloTimeout = 10 * time.Second
	}
	if o.PingInterval == 0 {
		o.PingInterval = 3 * time.Second
	}
	if o.DeadAfter == 0 {
		o.DeadAfter = 15 * time.Second
	}
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
}

// Server accepts one plugin at a time. A new connection replaces the old
// one, because the bot has one body and a reconnecting plugin should not
// wait for the old socket to time out.
type Server struct {
	opts    Options
	handler Handler

	mu      sync.Mutex
	current *Conn
	nextID  atomic.Uint64
	// conns counts connections being served, so Serve can let them finish.
	conns sync.WaitGroup
}

// NewServer returns a server that reports to h.
func NewServer(opts Options, h Handler) *Server {
	opts.defaults()
	return &Server{opts: opts, handler: h}
}

// ServeHTTP accepts the plugin's WebSocket on Path.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != Path {
		if s.opts.Fallback != nil {
			s.opts.Fallback.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
		return
	}
	s.conns.Add(1)
	defer s.conns.Done()
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		s.opts.Logger.Warn("websocket accept failed", "remote", r.RemoteAddr, "err", err)
		return
	}
	ws.SetReadLimit(1 << 20)
	id := s.nextID.Add(1)
	c := &Conn{
		id:     id,
		ws:     ws,
		remote: r.RemoteAddr,
		log:    s.opts.Logger.With("conn", id, "remote", r.RemoteAddr),
		tap:    s.opts.Tap,
	}
	s.serve(r.Context(), c)
}

// ListenAndServe serves on addr until ctx ends.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.Serve(ctx, ln)
}

// Serve serves on ln until ctx ends.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	go func() {
		<-ctx.Done()
		s.mu.Lock()
		if s.current != nil {
			s.current.Bye("shutdown")
		}
		s.mu.Unlock()
		shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	s.opts.Logger.Info("bridge listening", "addr", ln.Addr().String(), "path", Path)
	err := srv.Serve(ln)
	// Let the connections finish, so their last words reach the logs.
	finished := make(chan struct{})
	go func() {
		s.conns.Wait()
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		s.opts.Logger.Warn("connections still open at shutdown")
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) serve(ctx context.Context, c *Conn) {
	hello, err := s.handshake(ctx, c)
	if err != nil {
		c.log.Warn("handshake refused", "err", err)
		return
	}
	c.Hello = hello
	c.log = c.log.With("character", hello.Character, "plugin", hello.Plugin)

	s.mu.Lock()
	old := s.current
	s.current = c
	s.mu.Unlock()
	if old != nil {
		old.log.Info("replaced by a newer connection")
		old.close(CloseReplaced, "replaced by a newer connection")
	}

	c.log.Info("plugin connected", "game", hello.Game, "phase", hello.Phase, "world", hello.World,
		"capabilities", hello.Capabilities)
	s.handler.Connected(c)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go s.pingLoop(ctx, c)
	err = s.readLoop(ctx, c)

	s.mu.Lock()
	if s.current == c {
		s.current = nil
	}
	s.mu.Unlock()
	c.close(websocket.StatusNormalClosure, "")
	c.log.Info("plugin disconnected", "err", err)
	s.handler.Disconnected(c, err)
}

func (s *Server) handshake(ctx context.Context, c *Conn) (*protocol.Hello, error) {
	readCtx, cancel := context.WithTimeout(ctx, s.opts.HelloTimeout)
	defer cancel()
	_, data, err := c.ws.Read(readCtx)
	if err != nil {
		c.ws.CloseNow()
		return nil, fmt.Errorf("waiting for hello: %w", err)
	}
	c.observeFrame("in", data)
	msg, err := protocol.Decode(data, protocol.FromPlugin)
	var verr *protocol.VersionError
	if errors.As(err, &verr) {
		c.sendError("", "unsupported_version", verr.Error())
		c.close(CloseBadHello, "unsupported protocol version")
		return nil, err
	}
	hello, ok := msg.(*protocol.Hello)
	if err != nil || !ok {
		if err == nil {
			err = fmt.Errorf("first message was %q, not hello", msg.MessageType())
		}
		c.sendError("", protocol.CodeBadMessage, err.Error())
		c.close(CloseBadHello, "expected hello")
		return nil, err
	}
	c.observe(hello)
	if s.opts.Token != "" && subtle.ConstantTimeCompare([]byte(hello.Token), []byte(s.opts.Token)) != 1 {
		c.close(CloseBadToken, "bad token")
		return nil, errors.New("bad token")
	}
	welcome := s.opts.Welcome(hello)
	if err := c.Send(ctx, welcome); err != nil {
		return nil, fmt.Errorf("sending welcome: %w", err)
	}
	c.Welcome = welcome
	return hello, nil
}

func (s *Server) readLoop(ctx context.Context, c *Conn) error {
	for {
		readCtx, cancel := context.WithTimeout(ctx, s.opts.DeadAfter)
		_, data, err := c.ws.Read(readCtx)
		timedOut := readCtx.Err() == context.DeadlineExceeded
		cancel()
		if err != nil {
			if timedOut {
				return fmt.Errorf("no message for %s", s.opts.DeadAfter)
			}
			return err
		}
		c.observeFrame("in", data)
		msg, err := protocol.Decode(data, protocol.FromPlugin)
		if err != nil {
			c.log.Warn("bad message from plugin", "err", err, "frame", truncate(data, 200))
			c.sendError(protocol.IDOf(data), protocol.CodeBadMessage, err.Error())
			continue
		}
		c.observe(msg)
		if _, ok := msg.(*protocol.Pong); ok {
			continue
		}
		if bye, ok := msg.(*protocol.Bye); ok {
			return fmt.Errorf("plugin said bye: %s", bye.Reason)
		}
		s.handler.Message(c, msg)
	}
}

func (s *Server) pingLoop(ctx context.Context, c *Conn) {
	t := time.NewTicker(s.opts.PingInterval)
	defer t.Stop()
	var n uint64
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			n++
			if err := c.Send(ctx, &protocol.Ping{Envelope: protocol.Envelope{ID: fmt.Sprintf("p-%d", n)}}); err != nil {
				return
			}
		}
	}
}

// Conn is one plugin connection.
type Conn struct {
	id     uint64
	ws     *websocket.Conn
	remote string
	log    *slog.Logger

	// Hello and Welcome are the handshake, set before Handler.Connected.
	Hello   *protocol.Hello
	Welcome *protocol.Welcome

	tap       func(conn uint64, dir string, frame []byte)
	seq       atomic.Uint64
	lastT     atomic.Uint64 // math.Float64bits of the newest `t` the plugin sent
	closeOnce sync.Once
}

// ID tells connections apart in logs and in a Handler.
func (c *Conn) ID() uint64 { return c.id }

// Send stamps msg with the next seq and the newest game time the plugin has
// reported, then writes it. It is safe to call from any goroutine.
func (c *Conn) Send(ctx context.Context, msg protocol.Message) error {
	env := protocol.EnvelopeOf(msg)
	env.Seq = c.seq.Add(1)
	env.T = math.Float64frombits(c.lastT.Load())
	data, err := protocol.Encode(msg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := c.ws.Write(ctx, websocket.MessageText, data); err != nil {
		return err
	}
	c.observeFrame("out", data)
	return nil
}

// observeFrame hands a frame to the tap, if there is one.
func (c *Conn) observeFrame(dir string, data []byte) {
	if c.tap != nil {
		c.tap(c.id, dir, data)
	}
}

// Bye tells the plugin the controller is leaving, then closes.
func (c *Conn) Bye(reason string) {
	_ = c.Send(context.Background(), &protocol.Bye{Reason: reason})
	c.close(websocket.StatusNormalClosure, reason)
}

func (c *Conn) observe(m protocol.Message) {
	if t := protocol.EnvelopeOf(m).T; t > 0 {
		c.lastT.Store(math.Float64bits(t))
	}
}

func (c *Conn) sendError(re, code, detail string) {
	e := &protocol.Error{Envelope: protocol.Envelope{Re: re}, Code: code, Detail: detail}
	if err := c.Send(context.Background(), e); err != nil {
		c.log.Debug("could not send error", "err", err)
	}
}

func (c *Conn) close(code websocket.StatusCode, reason string) {
	c.closeOnce.Do(func() {
		go c.ws.Close(code, reason)
	})
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
