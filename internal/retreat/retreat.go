// Package retreat gets the bot out of fights: after a confirmed hostile hit,
// or on low health with a hostile creature after it, it flees toward safety and
// says so (docs/BRIDGE-PROTOCOL.md 7.2, flee).
package retreat

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"
	"sync"
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
	// HurtFor bridges AI targets that are not replicated to this client.
	// A recent attacker must still be perceived as hostile and within Near.
	HurtFor = 30 * time.Second
	// There is how near safety the bot must be to be at it already.
	There = 6.0
	// HomeReach is how far home may be to run to: farther, the run itself
	// is the danger.
	HomeReach = 150.0
	// For bounds one flee: one that cannot reach safety still ends.
	For = 60.0
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

	defended map[string]time.Time
	last     time.Time
	mu       sync.Mutex
	hurt     map[string]injury
}

type injury struct {
	world string
	at    time.Time
	until time.Time
}

// Event remembers actual attackers, since remotely owned AI does not expose
// its current target on this client. Environment damage has no attacker.
func (r *Retreat) Event(e *protocol.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e.Event == protocol.EventDied || e.Event == protocol.EventRespawned {
		r.hurt = nil
		r.defended = nil
		return
	}
	if e.Event != protocol.EventDamaged || e.By == "" || e.Amount <= 0 {
		return
	}
	st := r.s.State()
	if st == nil || st.Phase != protocol.PhaseInWorld {
		return
	}
	if r.hurt == nil {
		r.hurt = make(map[string]injury)
	}
	now := r.now()
	r.hurt[e.By] = injury{world: st.World, at: now, until: now.Add(HurtFor)}
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

// Tick reacts to a confirmed hostile attacker at any health. An AI target
// alone still requires low health. A fresh hit can bypass the retry cooldown,
// but never replaces an active flee with another one.
func (r *Retreat) Tick(ctx context.Context) {
	st, seen := r.s.State(), r.s.Perception()
	if st == nil || seen == nil || st.Phase != protocol.PhaseInWorld || st.HP <= 0 || st.HPMax <= 0 {
		return
	}
	if st.Task != nil && st.Task.Task == protocol.TaskFlee && st.Task.Status == protocol.StatusRunning {
		return
	}
	if nativeEscort(st) && st.Escort.Mode == "escape" {
		return
	}
	if r.controlledAttack(st, seen) {
		return
	}
	threat := r.threat(st, seen, st.HP/st.HPMax >= Low)
	if threat == nil {
		return
	}
	r.mu.Lock()
	hit, hurt := r.hurt[threat.ID]
	r.mu.Unlock()
	if hurt && hit.world == st.World && r.now().Before(hit.until) && r.defend(ctx, st, seen, threat) {
		return
	}
	if r.now().Sub(r.last) < Again && (!hurt || !hit.at.After(r.last)) {
		return
	}
	toward, to := r.safety(st.Pos, threat.Pos, seen)
	switch {
	case to != "" && toward.Pos != nil && st.Pos.FlatDistance(*toward.Pos) <= There:
		// An owner's presence is not protection while a threat is close.
		// Keep moving away instead of waiting beside them to be killed.
		toward, to = protocol.Target{}, ""
	case to == "" && toward.Pos != nil && st.Pos.FlatDistance(*toward.Pos) <= There:
		// Home, and the creature followed it there: away it is.
		toward = protocol.Target{}
	}
	if toward.Pos != nil && PastThreat(st.Pos, threat.Pos, *toward.Pos) {
		// The plugin would run straight away instead: say that.
		toward, to = protocol.Target{}, ""
	}
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
	if _, err := r.s.Dispatch(ctx, session.Order{Task: protocol.TaskFlee, Args: args, TimeoutS: For}); err != nil {
		r.log.Warn("could not flee", "err", err)
		return
	}
	r.log.Info("fleeing", "from", threat.ID, "prefab", threat.Prefab, "hp", st.HP, "toward", to)
	if _, err := r.s.Dispatch(ctx, session.Order{Task: protocol.TaskSay, Args: &protocol.SayArgs{Text: line}}); err != nil {
		r.log.Warn("could not say so", "err", err)
	}
}

// Try one native-confirmed defensive encounter before another open-ended chase.
// Native AllowCombat, armour and local safety checks remain authoritative. A
// refusal falls through to ordinary flee on the next tick, without retry loops.
func (r *Retreat) defend(ctx context.Context, st *protocol.State, seen *protocol.Perception, threat *protocol.PerceivedCharacter) bool {
	if threat.Prefab != "Greyling" || threat.Stars != 0 || st.Pos.FlatDistance(threat.Pos) > 6 ||
		st.HP < 30 || st.HP < st.HPMax*.65 || st.Stamina < 35 || st.Food == nil || len(st.Food.Active) == 0 {
		return false
	}
	for _, other := range seen.Characters {
		if other.ID != threat.ID && other.HPFrac > 0 && st.Pos.FlatDistance(other.Pos) < 8 {
			return false
		}
	}
	for _, structure := range seen.Structures {
		if st.Pos.FlatDistance(structure.Pos) < 8 {
			return false
		}
	}
	r.mu.Lock()
	for id, until := range r.defended {
		if !r.now().Before(until) {
			delete(r.defended, id)
		}
	}
	if until, ok := r.defended[threat.ID]; ok && r.now().Before(until) {
		r.mu.Unlock()
		return false
	}
	if r.defended == nil {
		r.defended = make(map[string]time.Time)
	}
	r.defended[threat.ID] = r.now().Add(time.Minute)
	r.mu.Unlock()
	_, err := r.s.Dispatch(ctx, session.Order{Task: protocol.TaskAttack, Args: &protocol.AttackArgs{Entity: threat.ID, Style: "melee", Defensive: true}, TimeoutS: 85})
	if err != nil {
		r.log.Warn("could not start defensive encounter", "err", err)
		return false
	}
	r.log.Info("defending against confirmed attacker", "entity", threat.ID, "hp", st.HP)
	return true
}

// Native escape is trusted only for the active matching body task.
func nativeEscort(st *protocol.State) bool {
	return st.Task != nil && st.Task.Task == protocol.TaskEscort && st.Task.Status == protocol.StatusRunning &&
		st.Escort != nil && st.Escort.ID != "" && st.Escort.ID == st.Task.ID
}

// The plugin owns a bounded explicit melee encounter and its fallback flee.
// Match the task ID so stale combat metadata cannot suppress ordinary retreat.
func (r *Retreat) controlledAttack(st *protocol.State, seen *protocol.Perception) bool {
	c := st.Combat
	if c == nil || st.Task == nil || (st.Task.Task != protocol.TaskAttack && !(nativeEscort(st) && st.Escort.Mode == "defend" && st.Escort.Entity == c.Entity)) || st.Task.Status != protocol.StatusRunning || c.ID == "" || c.ID != st.Task.ID {
		return false
	}
	if c.Mode == "retreat" {
		return true
	}
	healthReady := st.HP >= st.HPMax*.8
	if c.Defensive {
		healthReady = st.HP >= 25 && st.HP >= st.HPMax*.6
	}
	if !healthReady || st.Stamina < 10 {
		return false
	}
	rangeLimit := 12.0
	switch c.Mode {
	case "closing":
		rangeLimit = 30
	case "draw", "kite", "hold":
		rangeLimit = 40
	case "approach", "strike", "block", "dodge", "wait":
	default:
		return false
	}
	if c.Defensive {
		rangeLimit = 6
	}
	found := false
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, other := range seen.Characters {
		if other.ID == c.Entity {
			eligible := other.Prefab == "Greyling" || nativeEscort(st) && (other.Prefab == "Boar" || other.Prefab == "Neck")
			found = other.Kind == protocol.CharacterCreature && other.Hostile && eligible && other.Stars == 0 && other.HPFrac > 0 && st.Pos.FlatDistance(other.Pos) <= rangeLimit
			continue
		}
		if other.Kind != protocol.CharacterCreature || !other.Hostile || other.HPFrac <= 0 {
			continue
		}
		d := st.Pos.FlatDistance(other.Pos)
		hit, hurt := r.hurt[other.ID]
		hurt = hurt && hit.world == st.World && r.now().Before(hit.until)
		if d <= 6 || d <= Near && (other.TargetingMe || hurt) {
			return false
		}
	}
	return found
}

// pastThreatCos is the cosine of 60°: a place within that angle of the
// threat's bearing lies past it, as the plugin's Flee.PastThreat judges.
var pastThreatCos = math.Cos(60 * math.Pi / 180)

// PastThreat reports whether place lies past the threat seen from pos, where
// running to it would run into the threat.
func PastThreat(pos, threat, place protocol.Vec3) bool {
	tx, tz := threat[0]-pos[0], threat[2]-pos[2]
	px, pz := place[0]-pos[0], place[2]-pos[2]
	tl, pl := math.Hypot(tx, tz), math.Hypot(px, pz)
	if tl < 0.5 || pl < 0.5 {
		return false
	}
	return (tx*px+tz*pz)/(tl*pl) > pastThreatCos
}

// threat is the nearest hostile within Near that targets or recently hurt
// the bot. Neither an unrelated hostile nor an old attack causes a retreat.
func (r *Retreat) threat(st *protocol.State, seen *protocol.Perception, damagedOnly bool) *protocol.PerceivedCharacter {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, hit := range r.hurt {
		if hit.world != st.World || !r.now().Before(hit.until) {
			delete(r.hurt, id)
		}
	}
	var (
		best *protocol.PerceivedCharacter
		dist = Near
	)
	for i, c := range seen.Characters {
		_, hurt := r.hurt[c.ID]
		if c.Kind != protocol.CharacterCreature || !c.Hostile || c.HPFrac <= 0 || (!c.TargetingMe && !hurt) {
			continue
		}
		if damagedOnly && !hurt {
			continue
		}
		if d := st.Pos.FlatDistance(c.Pos); d <= dist {
			best, dist = &seen.Characters[i], d
		}
	}
	return best
}

// safety is where to run: the nearest owner in sight within OwnerReach,
// named in owner and with Pos set to where they stand, or home within
// HomeReach when the threat is not at it, or nowhere.
func (r *Retreat) safety(pos, threat protocol.Vec3, seen *protocol.Perception) (to protocol.Target, owner string) {
	owners := r.owners()
	best := OwnerReach
	for _, c := range seen.Characters {
		if c.Kind != protocol.CharacterPlayer || !slices.Contains(owners, c.Name) || c.HPFrac <= 0 {
			continue
		}
		if d := pos.FlatDistance(c.Pos); d <= best {
			at := c.Pos
			best, to, owner = d, protocol.Target{Entity: c.ID, Pos: &at}, c.Name
		}
	}
	if owner != "" {
		return to, owner
	}
	if r.home == nil {
		return protocol.Target{}, ""
	}
	if h, ok := r.home(); ok && h.FlatDistance(threat) > HomeClear && h.FlatDistance(pos) <= HomeReach {
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
