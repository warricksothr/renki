// Package retreat gets the bot out of fights it is losing: on low health,
// with a hostile creature after it, it flees toward its owner or home, and
// says so (docs/BRIDGE-PROTOCOL.md 7.2, flee).
package retreat

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

const (
	// Low is the share of health below which the bot gets away.
	Low = 0.4
	// Near is how close a hostile creature that has picked the bot must be
	// to count as a threat.
	Near = 15.0
	// Distance is how far from the threat the bot runs.
	Distance = 25.0
	// OwnerReach is how far an owner in sight may be to run to.
	OwnerReach = 60.0
	// HomeClear is how far from home the threat must be for home to be
	// safety: a creature at home makes it no refuge.
	HomeClear = 15.0
	// Again is how long after one retreat before another may start.
	Again = 10 * time.Second
)

// Session is what the retreat needs of the session.
type Session interface {
	Dispatch(ctx context.Context, o session.Order) (*session.Task, error)
	State() *protocol.State
	Perception() *protocol.Perception
}

// Retreat watches the bot's health and what is after it.
type Retreat struct {
	s      Session
	owners func() []string
	home   func() (protocol.Vec3, bool)
	log    *slog.Logger
	now    func() time.Time

	last time.Time
}

// New returns a retreat. owners and home, where the bot lives in the
// current world if anywhere, are read each time; home may be nil.
func New(s Session, owners func() []string, home func() (protocol.Vec3, bool), log *slog.Logger, now func() time.Time) *Retreat {
	if now == nil {
		now = time.Now
	}
	return &Retreat{s: s, owners: owners, home: home, log: log, now: now}
}

// Run calls Tick every second until ctx ends.
func (r *Retreat) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.Tick(ctx)
		}
	}
}

// Tick sends the bot running when it is low on health with a threat near,
// unless it is already running.
func (r *Retreat) Tick(ctx context.Context) {
	st, seen := r.s.State(), r.s.Perception()
	if st == nil || seen == nil || st.Phase != protocol.PhaseInWorld || st.HPMax <= 0 {
		return
	}
	if st.HP/st.HPMax >= Low || r.now().Sub(r.last) < Again {
		return
	}
	if st.Task != nil && st.Task.Task == protocol.TaskFlee && st.Task.Status == protocol.StatusRunning {
		return
	}
	threat := Threat(st.Pos, seen)
	if threat == nil {
		return
	}
	toward, to := r.safety(st.Pos, threat.Pos, seen)
	args := &protocol.FleeArgs{From: protocol.Target{Entity: threat.ID}, Distance: ptr(Distance)}
	line := fmt.Sprintf("Too much for me. Getting away from the %s.", name(threat))
	switch {
	case to != "":
		args.Toward = &protocol.Target{Entity: toward.Entity}
		line = fmt.Sprintf("Too much for me. Falling back to you, %s.", to)
	case toward.Pos != nil:
		args.Toward = &toward
		line = "Too much for me. Heading home."
	}
	r.last = r.now()
	if _, err := r.s.Dispatch(ctx, session.Order{Task: protocol.TaskFlee, Args: args}); err != nil {
		r.log.Warn("could not flee", "err", err)
		return
	}
	r.log.Info("fleeing", "from", threat.ID, "prefab", threat.Prefab, "hp", st.HP, "toward", to)
	if _, err := r.s.Dispatch(ctx, session.Order{Task: protocol.TaskSay, Args: &protocol.SayArgs{Text: line}}); err != nil {
		r.log.Warn("could not say so", "err", err)
	}
}

// Threat is the nearest hostile creature within Near that has picked the
// bot, or nil.
func Threat(pos protocol.Vec3, seen *protocol.Perception) *protocol.PerceivedCharacter {
	var (
		best *protocol.PerceivedCharacter
		dist = Near
	)
	for i, c := range seen.Characters {
		if c.Kind != protocol.CharacterCreature || !c.Hostile || !c.TargetingMe {
			continue
		}
		if d := pos.FlatDistance(c.Pos); d <= dist {
			best, dist = &seen.Characters[i], d
		}
	}
	return best
}

// safety is where to run: the nearest owner in sight within OwnerReach,
// named in owner, or home when the threat is not at it, or nowhere.
func (r *Retreat) safety(pos, threat protocol.Vec3, seen *protocol.Perception) (to protocol.Target, owner string) {
	owners := r.owners()
	best := OwnerReach
	for _, c := range seen.Characters {
		if c.Kind != protocol.CharacterPlayer || !slices.Contains(owners, c.Name) || c.HPFrac <= 0 {
			continue
		}
		if d := pos.FlatDistance(c.Pos); d <= best {
			best, to, owner = d, protocol.Target{Entity: c.ID}, c.Name
		}
	}
	if owner != "" {
		return to, owner
	}
	if r.home == nil {
		return protocol.Target{}, ""
	}
	if h, ok := r.home(); ok && h.FlatDistance(threat) > HomeClear {
		return protocol.Target{Pos: &h}, ""
	}
	return protocol.Target{}, ""
}

func name(c *protocol.PerceivedCharacter) string {
	if c.Prefab == "" {
		return "creature"
	}
	return strings.ToLower(c.Prefab)
}

func ptr(f float64) *float64 { return &f }
