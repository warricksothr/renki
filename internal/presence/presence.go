// Package presence handles player-specific greetings and location exchanges.
// It never takes body control or invokes a model.
package presence

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/access"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

type Session interface {
	Handshake() (*protocol.Hello, *protocol.Welcome, bool)
	StateObservation() (*protocol.State, time.Time)
	PerceptionObservation() (*protocol.Hello, string, *protocol.Perception, time.Time)
	DispatchFor(context.Context, session.Order, *protocol.Hello, string) (*session.Task, error)
	Configure(context.Context, protocol.Config) error
}
type Live struct {
	Owners    []string
	Players   access.Rules
	Presence  config.Presence
	AgentName string
	Nicknames []string
}
type Options struct {
	Session Session
	Live    func() Live
	Now     func() time.Time
	Logger  *slog.Logger
}
type request struct {
	kind, player, key, world string
	hello                    *protocol.Hello
	at                       time.Time
	pos                      protocol.Vec3
}
type Manager struct {
	o                                          Options
	queue                                      chan request
	pending                                    []request
	hello                                      *protocol.Hello
	world                                      string
	marcoUntil, lastMarco, nextSay, nextConfig time.Time
	replies, joins                             map[string]time.Time
}

func New(o Options) *Manager {
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	return &Manager{o: o, queue: make(chan request, 64), replies: map[string]time.Time{}, joins: map[string]time.Time{}}
}
func normalized(s string) string {
	s = strings.Map(func(r rune) rune {
		if strings.ContainsRune(",.!?:;", r) {
			return ' '
		}
		return r
	}, s)
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}
func (m *Manager) scope() (request, bool) {
	h, _, connected := m.o.Session.Handshake()
	st, at := m.o.Session.StateObservation()
	now := m.o.Now()
	if !connected || h == nil || st == nil || st.Phase != protocol.PhaseInWorld || st.HP <= 0 || st.World == "" || now.Sub(at) < 0 || now.Sub(at) > 3*time.Second {
		return request{}, false
	}
	return request{hello: h, world: st.World, at: now}, true
}
func (m *Manager) enqueue(r request) {
	select {
	case m.queue <- r:
	default:
		m.o.Logger.Warn("presence queue full; message skipped")
	}
}

// Chat returns true for the exact routine phrases it consumes. Permission
// checks happen again at execution, so revocation also drops queued replies.
func (m *Manager) Chat(c *protocol.Chat) bool {
	live := m.o.Live()
	if !live.Presence.MarcoPolo || c.Channel != protocol.ChannelShout {
		return false
	}
	text := normalized(c.Text)
	kind := ""
	if text == "polo" {
		kind = "answer"
	} else {
		name := live.AgentName
		if name == "" {
			if st, _ := m.o.Session.StateObservation(); st != nil {
				name = st.Character
			}
		}
		for _, alias := range append(slices.Clone(live.Nicknames), name) {
			if alias != "" && text == normalized(alias)+" marco" {
				kind = "call"
				break
			}
		}
	}
	if kind == "" {
		return false
	}
	if kind == "call" && !live.Players.Has(live.Owners, c.From.Name, access.Locate) {
		return true
	}
	r, ok := m.scope()
	if !ok {
		return true
	}
	st, _ := m.o.Session.StateObservation()
	if st == nil || st.World != r.world || c.T-st.T < -5 || c.T-st.T > 5 {
		return true
	}
	r.kind, r.player, r.key, r.pos = kind, c.From.Name, c.From.PlayerID+":"+c.From.Name, c.Pos
	if len(r.player) > 64 || len(r.key) > 160 {
		return true
	}
	for _, n := range r.pos {
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return true
		}
	}
	m.enqueue(r)
	return true
}
func (m *Manager) Event(e *protocol.Event) {
	if e.Event != protocol.EventPlayerJoined && e.Event != protocol.EventPlayerLeft {
		return
	}
	live := m.o.Live()
	if !live.Presence.MarcoPolo || !live.Players.Has(live.Owners, e.Name, access.Greet) {
		return
	}
	r, ok := m.scope()
	if !ok {
		return
	}
	r.kind = "join"
	if e.Event == protocol.EventPlayerLeft {
		r.kind = "left"
	}
	r.player = e.Name
	m.enqueue(r)
}
func (m *Manager) Run(ctx context.Context) {
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			step, cancel := context.WithTimeout(ctx, time.Second)
			m.Tick(step)
			cancel()
		}
	}
}

// Tick has one caller (Run); callbacks only write to the bounded queue.
func (m *Manager) Tick(ctx context.Context) {
	now := m.o.Now()
	current, ok := m.scope()
	if !ok {
		m.pending = nil
		m.drain(false)
		m.hello = nil
		m.world = ""
		m.marcoUntil = time.Time{}
		return
	}
	if current.hello != m.hello || current.world != m.world {
		m.pending = nil
		m.hello, m.world = current.hello, current.world
		m.marcoUntil = time.Time{}
		m.lastMarco = time.Time{}
		m.nextSay = time.Time{}
		m.nextConfig = time.Time{}
		m.replies = map[string]time.Time{}
		m.joins = map[string]time.Time{}
	}
	m.drain(true)
	live := m.o.Live()
	st, _ := m.o.Session.StateObservation()
	if st == nil || st.World != current.world || st.Phase != protocol.PhaseInWorld || st.HP <= 0 {
		m.pending = nil
		return
	}
	if !now.Before(m.nextConfig) && st.SharePosition != nil && *st.SharePosition != live.Presence.SharePosition {
		m.nextConfig = now.Add(5 * time.Second)
		desired := live.Presence.SharePosition
		if err := m.o.Session.Configure(ctx, protocol.Config{SharePosition: &desired}); err != nil {
			m.o.Logger.Debug("marker update deferred", "err", err)
		}
	}
	if !live.Presence.MarcoPolo {
		m.pending = nil
		m.marcoUntil = time.Time{}
		return
	}
	pending := m.pending
	m.pending = nil
	for _, r := range pending {
		if r.hello != current.hello || r.world != current.world || now.Sub(r.at) < 0 || now.Sub(r.at) > 10*time.Second {
			continue
		}
		if now.Before(m.nextSay) {
			m.pending = append(m.pending, r)
			continue
		}
		text := ""
		switch r.kind {
		case "join":
			if !live.Players.Has(live.Owners, r.player, access.Greet) || now.Sub(m.joins[r.player]) < 60*time.Second {
				continue
			}
			if now.Sub(r.at) < 2*time.Second {
				m.pending = append(m.pending, r)
				continue
			}
			h, w, seen, at := m.o.Session.PerceptionObservation()
			if h != current.hello || w != current.world || seen == nil || now.Sub(at) < 0 || now.Sub(at) > 3*time.Second || st.T-seen.T < 0 || st.T-seen.T > 5 {
				if now.Sub(r.at) < 5*time.Second {
					m.pending = append(m.pending, r)
				}
				continue
			}
			visible := false
			for _, c := range seen.Characters {
				if c.Kind == protocol.CharacterPlayer && c.Name == r.player {
					visible = true
					break
				}
			}
			if visible {
				continue
			}
			boundedRemember(m.joins, r.player, now)
			if now.Sub(m.lastMarco) < 5*time.Second {
				continue
			}
			text = "Marco!"
		case "call":
			if !live.Players.Has(live.Owners, r.player, access.Locate) || now.Sub(m.replies[r.key]) < 15*time.Second {
				continue
			}
			text = "Polo! " + direction(r.player, st.Pos, r.pos, true)
		case "answer":
			if !now.Before(m.marcoUntil) || now.Sub(m.replies[r.key]) < 15*time.Second {
				continue
			}
			text = direction(r.player, r.pos, st.Pos, false)
		}
		if text == "" {
			continue
		}
		if _, err := m.o.Session.DispatchFor(ctx, session.Order{Task: protocol.TaskSay, Args: &protocol.SayArgs{Text: text, Channel: protocol.ChannelShout}, TimeoutS: 5, Origin: "presence"}, current.hello, current.world); err != nil {
			m.o.Logger.Debug("presence reply skipped", "err", err)
			continue
		}
		m.nextSay = now.Add(2 * time.Second)
		if r.kind == "join" {
			m.lastMarco = now
			m.marcoUntil = now.Add(2 * time.Minute)
		} else {
			boundedRemember(m.replies, r.key, now)
		}
	}
}
func (m *Manager) drain(keep bool) {
	for range cap(m.queue) {
		select {
		case r := <-m.queue:
			if !keep {
				continue
			}
			if r.kind == "left" {
				m.pending = slices.DeleteFunc(m.pending, func(old request) bool {
					return old.kind == "join" && old.player == r.player && old.hello == r.hello && old.world == r.world
				})
				continue
			}
			if len(m.pending) < 64 {
				m.pending = append(m.pending, r)
			}
		default:
			return
		}
	}
}
func boundedRemember(values map[string]time.Time, key string, now time.Time) {
	if len(values) >= 128 {
		oldest := ""
		var at time.Time
		for k, t := range values {
			if oldest == "" || t.Before(at) {
				oldest, at = k, t
			}
		}
		delete(values, oldest)
	}
	values[key] = now
}

// Both directions are horizontal offsets, not claims about a navigable route.
func direction(name string, from, to protocol.Vec3, toCaller bool) string {
	dx, dz := to[0]-from[0], to[2]-from[2]
	distance := math.Hypot(dx, dz)
	if distance < 5 {
		if toCaller {
			return fmt.Sprintf("%s, you're right beside me.", name)
		}
		return fmt.Sprintf("%s, I'm right beside you.", name)
	}
	dirs := []string{"north", "north-east", "east", "south-east", "south", "south-west", "west", "north-west"}
	angle := math.Atan2(dx, dz)
	index := int(math.Round(angle/(math.Pi/4))+8) % 8
	metres := math.Round(distance/5) * 5
	if toCaller {
		return fmt.Sprintf("%s, you're about %.0f m %s of me, straight-line.", name, metres, dirs[index])
	}
	return fmt.Sprintf("%s, travel about %.0f m %s to reach me, straight-line.", name, metres, dirs[index])
}
