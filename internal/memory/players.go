package memory

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

const PlayerReferenceAge = 30 * time.Second
const maxPendingPlayers = 4096

type recentPlayer struct {
	home.PlayerEncounter
	hello *protocol.Hello
}

// Players records only evidence from perception and located speech. The read
// goroutine appends to bounded memory; a single worker owns disk IO and Tick.
type Players struct {
	Session *session.Session
	Store   *home.Store
	Logger  *slog.Logger
	mu      sync.Mutex
	pending []home.PlayerEncounter
	dropped uint64
	recent  map[string]recentPlayer
	frame   *protocol.Perception
	hello   *protocol.Hello
}

func (p *Players) ObserveChat(c *protocol.Chat, scope session.PlayerChatObservation) {
	p.observe(home.PlayerEncounter{World: scope.World, Name: c.From.Name, Pos: c.Pos, At: scope.At, Source: "heard"}, scope.Hello)
}

func (p *Players) observe(o home.PlayerEncounter, hello *protocol.Hello) {
	if o.World == "" || hello == nil || !home.ValidPlayerEncounter(o) {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.recent == nil {
		p.recent = map[string]recentPlayer{}
	}
	if prior, ok := p.recent[o.Name]; !ok || !prior.At.After(o.At) {
		if !ok && len(p.recent) >= home.MaxPlayers {
			oldest := ""
			var at time.Time
			for name, r := range p.recent {
				if oldest == "" || r.At.Before(at) {
					oldest, at = name, r.At
				}
			}
			delete(p.recent, oldest)
		}
		p.recent[o.Name] = recentPlayer{o, hello}
	}
	if len(p.pending) < maxPendingPlayers {
		p.pending = append(p.pending, o)
	} else {
		p.dropped++
	}
}

func (p *Players) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	next := time.Now().Add(30 * time.Second)
	for {
		select {
		case <-ctx.Done():
			p.Tick(time.Now())
			p.Flush(time.Now())
			return
		case now := <-t.C:
			p.Tick(now)
			if !now.Before(next) {
				p.Flush(now)
				next = now.Add(30 * time.Second)
			}
		}
	}
}

func (p *Players) Tick(now time.Time) {
	h, world, f, at := p.Session.PerceptionObservation()
	if h == nil || f == nil || world == "" || now.Sub(at) < 0 || now.Sub(at) > 3*time.Second || h == p.hello && f == p.frame {
		return
	}
	st, stateAt := p.Session.StateObservation()
	if st == nil || st.World != world || st.Phase != protocol.PhaseInWorld || st.HP <= 0 || now.Sub(stateAt) < 0 || now.Sub(stateAt) > 3*time.Second || st.T-f.T < -5 || st.T-f.T > 5 {
		return
	}
	p.hello, p.frame = h, f
	for _, c := range f.Characters {
		if c.Kind == protocol.CharacterPlayer && c.Name != st.Character {
			p.observe(home.PlayerEncounter{World: world, Name: c.Name, Pos: c.Pos, At: at, Source: "seen"}, h)
		}
	}
}

// Flush retains failed batches and original times. No recorder mutex is held
// over disk IO, so speech collection remains nonblocking.
func (p *Players) Flush(now time.Time) bool {
	p.mu.Lock()
	snapshot := append([]home.PlayerEncounter(nil), p.pending...)
	dropped := p.dropped
	p.dropped = 0
	p.mu.Unlock()
	if dropped > 0 && p.Logger != nil {
		p.Logger.Warn("player history observations dropped", "count", dropped)
	}
	var batch, deferred []home.PlayerEncounter
	for _, o := range snapshot {
		if o.At.After(now) {
			deferred = append(deferred, o)
		} else {
			batch = append(batch, o)
		}
	}
	if err := p.Store.RememberPlayers(batch, now); err != nil {
		if p.Logger != nil {
			p.Logger.Warn("could not remember players", "err", err)
		}
		return false
	}
	p.mu.Lock()
	p.pending = append(deferred, p.pending[len(snapshot):]...)
	p.mu.Unlock()
	return true
}

// History includes unsaved observations and applies the same fading policy.
func (p *Players) History(world string, now time.Time) []home.PlayerEncounter {
	p.mu.Lock()
	var pending []home.PlayerEncounter
	for _, o := range p.pending {
		if o.World == world {
			pending = append(pending, o)
		}
	}
	p.mu.Unlock()
	return home.CompactPlayerHistory(append(p.Store.PlayerHistory(world, now), pending...), now)
}

// Reference never uses a persisted or previous-connection position. Hearing
// the actual question updates this evidence before the routine processes it.
func (p *Players) Reference(world, name string, hello *protocol.Hello, now time.Time) (protocol.Vec3, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.recent[name]
	return r.Pos, ok && r.World == world && r.hello == hello && now.Sub(r.At) >= 0 && now.Sub(r.At) <= PlayerReferenceAge
}

// Last accepts exact names, or an unambiguous case-insensitive match.
func (p *Players) Last(world, name string, now time.Time) (home.PlayerEncounter, bool) {
	history := p.History(world, now)
	for _, o := range history {
		if o.Name == name {
			return o, true
		}
	}
	found := ""
	var last home.PlayerEncounter
	for _, o := range history {
		if !strings.EqualFold(o.Name, name) {
			continue
		}
		if found != "" && found != o.Name {
			return home.PlayerEncounter{}, false
		}
		if found == "" {
			found, last = o.Name, o
		}
	}
	return last, found != ""
}
