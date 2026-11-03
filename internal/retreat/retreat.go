// Package retreat protects the bot from known pursuit or a confirmed hostile hit.
// It uses bounded native defense when prepared, otherwise flees toward safety and
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
	// Low is retained as the shared low-health threshold; known pursuit now
	// triggers protection at any health.
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

	defended                 map[string]time.Time
	last                     time.Time
	mu                       sync.Mutex
	hurt                     map[string]injury
	escape                   *escapeEpisode
	defense                  *defenseAttempt
	scope                    *protocol.Hello
	scopeWorld               string
	lifecycle, lifecycleSeen uint64
}

type defenseAttempt struct {
	scope    *protocol.Hello
	task     *session.Task
	world    string
	entity   string
	revision uint64
	at       time.Time
}

type escapeEpisode struct {
	scope         *protocol.Hello
	task          *session.Task
	threat, world string
	started       time.Time
	attempts      int
	revision      uint64
}

func (r *Retreat) revision() uint64 {
	if s, ok := r.s.(interface{ ControlRevision() uint64 }); ok {
		return s.ControlRevision()
	}
	return 0
}

func (r *Retreat) dispatch(ctx context.Context, o session.Order, revision uint64) (*session.Task, error) {
	if s, ok := r.s.(interface {
		DispatchObserved(context.Context, session.Order, *protocol.Hello, string, uint64) (*session.Task, error)
	}); ok {
		return s.DispatchObserved(ctx, o, r.scope, r.scopeWorld, revision)
	}
	if s, ok := r.s.(interface {
		DispatchAtRevision(context.Context, session.Order, uint64) (*session.Task, error)
	}); ok {
		return s.DispatchAtRevision(ctx, o, revision)
	}
	return r.s.Dispatch(ctx, o)
}

// Real sessions expose arrival timestamps and the perception's world. Never
// turn a stalled stream or a previous world's creatures into a new body order.
func (r *Retreat) observations() (*protocol.State, *protocol.Perception) {
	r.scope, r.scopeWorld = nil, ""
	st, seen := r.s.State(), r.s.Perception()
	if s, ok := r.s.(interface {
		StateObservation() (*protocol.State, time.Time)
		PerceptionObservation() (*protocol.Hello, string, *protocol.Perception, time.Time)
	}); ok {
		var at time.Time
		st, at = s.StateObservation()
		hello, world, perception, perceivedAt := s.PerceptionObservation()
		now := r.now()
		if st == nil || hello == nil || world != st.World || at.IsZero() || perceivedAt.IsZero() ||
			now.Sub(at) < 0 || now.Sub(at) > 3*time.Second || now.Sub(perceivedAt) < 0 || now.Sub(perceivedAt) > 3*time.Second ||
			perception == nil || math.Abs(st.T-perception.T) > 3 {
			return nil, nil
		}
		seen = perception
		r.scope, r.scopeWorld = hello, world
	}
	return st, seen
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
		r.lifecycle++
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

// Run checks pursuit four times a second until ctx ends.
func (r *Retreat) Run(ctx context.Context) {
	t := time.NewTicker(250 * time.Millisecond)
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

// Tick reacts before contact to known targeting, or to a recent actual hit.
// Only a controller-owned escape can be retried or re-aimed, once per episode.
func (r *Retreat) Tick(ctx context.Context) {
	revision := r.revision()
	st, seen := r.observations()
	r.mu.Lock()
	lifecycle := r.lifecycle
	r.mu.Unlock()
	if lifecycle != r.lifecycleSeen {
		r.escape, r.defense = nil, nil
		r.last = time.Time{}
		r.lifecycleSeen = lifecycle
	}
	if st == nil || seen == nil || st.Phase != protocol.PhaseInWorld || st.HP <= 0 || st.HPMax <= 0 {
		return
	}
	if s, ok := r.s.(interface {
		InspectionSnapshot() session.InspectionSnapshot
	}); ok {
		body := s.InspectionSnapshot().Task
		if body != nil && body.Name == protocol.TaskFlee && !protocol.Final(body.Status) &&
			(r.escape == nil || r.escape.task.ID != body.ID) {
			return
		}
	}
	if r.escape != nil && (r.escape.world != st.World || r.escape.revision != revision || r.escape.scope != r.scope) {
		if r.escape.world != st.World || r.escape.scope != r.scope {
			r.last = time.Time{}
		}
		r.escape = nil
		return // A newer owner order cancels delayed recovery.
	}
	if d := r.defense; d != nil {
		status, _, _ := d.task.Status()
		pendingSafe := st.HP >= st.HPMax*.6 && st.Stamina >= 10
		for _, other := range seen.Characters {
			r.mu.Lock()
			hit, hurt := r.hurt[other.ID]
			r.mu.Unlock()
			hurt = hurt && hit.world == st.World && r.now().Before(hit.until)
			if other.ID != d.entity && other.Kind == protocol.CharacterCreature && other.Hostile && other.HPFrac > 0 &&
				st.Pos.FlatDistance(other.Pos) <= Near && (other.TargetingMe || hurt || st.Pos.FlatDistance(other.Pos) <= 6) {
				pendingSafe = false
			}
		}
		if pendingSafe && d.scope == r.scope && d.world == st.World && d.revision == revision && !protocol.Final(status) &&
			r.now().Sub(d.at) < 3*time.Second && (st.Task == nil || st.Task.ID == d.task.ID) {
			return
		}
		r.defense = nil
	}
	// The native escort protects its charges even when injured or outnumbered.
	// A matching task owns its tactics; stale metadata never suppresses retreat.
	if nativeEscort(st) {
		return
	}
	if nativeTravel(st) {
		return
	}
	if r.controlledAttack(st, seen) {
		return
	}
	threat := r.threat(st, seen)
	if threat == nil {
		return
	}
	r.mu.Lock()
	hit, hurt := r.hurt[threat.ID]
	r.mu.Unlock()
	hurt = hurt && hit.world == st.World && r.now().Before(hit.until)
	alternate := false
	episode := r.escape
	activeFlee := st.Task != nil && st.Task.Task == protocol.TaskFlee && st.Task.Status == protocol.StatusRunning
	if episode != nil && (r.now().Sub(episode.started) < Again || activeFlee) {
		status, reason, _ := episode.task.Status()
		owned := st.Task == nil || st.Task.ID == episode.task.ID
		if !owned {
			return
		}
		newAttacker := hurt && threat.ID != episode.threat && hit.at.After(r.last)
		newHit := hurt && hit.at.After(r.last) && !activeFlee && protocol.Final(status)
		blocked := status == protocol.StatusFailed && slices.Contains([]string{protocol.ReasonBlocked, protocol.ReasonUnreachable, protocol.ReasonStuck}, reason)
		if episode.attempts >= 2 || (!newAttacker && !newHit && !blocked) {
			return
		}
		alternate = blocked && !newAttacker
	} else {
		if activeFlee {
			return
		} // Never replace an unrelated/manual flee.
		if r.defend(ctx, st, seen, threat, revision) {
			return
		}
		if r.now().Sub(r.last) < Again && (!hurt || !hit.at.After(r.last)) {
			return
		}
		episode = nil
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
	if alternate {
		// A tangent destination differs from the failed straight/home goal.
		// Native dry/hostile/physical checks still authorize every step.
		dx, dz := st.Pos[0]-threat.Pos[0], st.Pos[2]-threat.Pos[2]
		length := math.Hypot(dx, dz)
		if length < .5 {
			dx, dz, length = 1, 0, 1
		}
		target := protocol.Vec3{st.Pos[0] - dz/length*Distance, st.Pos[1], st.Pos[2] + dx/length*Distance}
		args.Toward = &protocol.Target{Pos: &target}
		line = "Escape blocked. Trying a different direction."
	}
	r.last = r.now()
	task, err := r.dispatch(ctx, session.Order{Task: protocol.TaskFlee, Args: args, TimeoutS: For}, revision)
	if err != nil {
		r.log.Warn("could not flee", "err", err)
		return
	}
	if episode == nil {
		episode = &escapeEpisode{scope: r.scope, world: st.World, started: r.now()}
	}
	episode.task, episode.threat, episode.revision = task, threat.ID, revision+1
	// Test doubles need not implement control revisions.
	if _, ok := r.s.(interface{ ControlRevision() uint64 }); !ok {
		episode.revision = 0
	}
	episode.attempts++
	r.escape = episode
	r.log.Info("fleeing", "from", threat.ID, "prefab", threat.Prefab, "hp", st.HP, "toward", to)
	if _, err := r.s.Dispatch(ctx, session.Order{Task: protocol.TaskSay, Args: &protocol.SayArgs{Text: line}}); err != nil {
		r.log.Warn("could not say so", "err", err)
	}
}

// Try one native-confirmed defensive encounter before another open-ended chase.
// Native AllowCombat, armour and local safety checks remain authoritative. A
// refusal falls through to ordinary flee on the next tick, without retry loops.
func (r *Retreat) defend(ctx context.Context, st *protocol.State, seen *protocol.Perception, threat *protocol.PerceivedCharacter, revision uint64) bool {
	if threat.Prefab != "Greyling" || threat.Stars != 0 || st.Pos.FlatDistance(threat.Pos) > 6 ||
		st.HP <= 0 || st.HP < st.HPMax*.65 || st.Stamina < 35 || st.Food == nil || len(st.Food.Active) == 0 {
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
	task, err := r.dispatch(ctx, session.Order{Task: protocol.TaskAttack, Args: &protocol.AttackArgs{Entity: threat.ID, Style: "melee", Defensive: true}, TimeoutS: 85}, revision)
	if err != nil {
		r.log.Warn("could not start defensive encounter", "err", err)
		return false
	}
	nextRevision := revision + 1
	if _, ok := r.s.(interface{ ControlRevision() uint64 }); !ok {
		nextRevision = 0
	}
	r.defense = &defenseAttempt{scope: r.scope, task: task, world: st.World, entity: threat.ID, revision: nextRevision, at: r.now()}
	r.log.Info("defending against hostile pursuit", "entity", threat.ID, "hp", st.HP)
	return true
}

// Native escape is trusted only for the active matching body task.
func nativeEscort(st *protocol.State) bool {
	return st.Task != nil && st.Task.Task == protocol.TaskEscort && st.Task.Status == protocol.StatusRunning &&
		st.Escort != nil && st.Escort.ID != "" && st.Escort.ID == st.Task.ID
}

// The matching native travel owner handles defense/withdrawal every physics
// step. Do not replace it with a second controller-owned body after a small hit.
func nativeTravel(st *protocol.State) bool {
	j := st.Journey
	return st.Task != nil && slices.Contains([]string{protocol.TaskMoveTo, protocol.TaskPickup, protocol.TaskInteract}, st.Task.Task) && st.Task.Status == protocol.StatusRunning &&
		j != nil && j.ID != "" && j.ID == st.Task.ID && j.Task == st.Task.Task && j.Attempts >= 0 && j.Attempts <= 3 &&
		slices.Contains([]string{"walk", "defend", "escape", "observe", "blocked"}, j.Mode) &&
		st.HP >= math.Max(20, st.HPMax*.4) && st.Stamina >= 8 && st.Food != nil && len(st.Food.Active) > 0 &&
		!st.Encumbered() && !st.Stranded() && !slices.Contains(st.Flags, protocol.FlagSwimming) &&
		(st.Location == nil || !st.Location.InWater)
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
		healthReady = st.HP > 0 && st.HP >= st.HPMax*.6
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
			eligible := other.Prefab == "Greyling" || (nativeEscort(st) || c.Peel != nil) && (other.Prefab == "Boar" || other.Prefab == "Neck")
			found = other.Kind == protocol.CharacterCreature && other.Hostile && eligible && other.Stars == 0 && other.HPFrac > 0 && st.Pos.FlatDistance(other.Pos) <= rangeLimit
			continue
		}
		if other.Kind != protocol.CharacterCreature || !other.Hostile || other.HPFrac <= 0 {
			continue
		}
		d := st.Pos.FlatDistance(other.Pos)
		hit, hurt := r.hurt[other.ID]
		hurt = hurt && hit.world == st.World && r.now().Before(hit.until)
		peeling := c.Peel != nil && c.Peel.Members > 0 && c.Peel.Members <= 32 &&
			slices.Contains([]string{"draw", "lead"}, c.Peel.Phase) &&
			slices.Contains([]string{"draw", "kite"}, c.Mode)
		if d <= 6 || !peeling && d <= Near && (other.TargetingMe || hurt) {
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
func (r *Retreat) threat(st *protocol.State, seen *protocol.Perception) *protocol.PerceivedCharacter {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, hit := range r.hurt {
		if hit.world != st.World || !r.now().Before(hit.until) {
			delete(r.hurt, id)
		}
	}
	var (
		best     *protocol.PerceivedCharacter
		dist     = Near
		bestHurt bool
		newest   time.Time
	)
	for i, c := range seen.Characters {
		hit, hurt := r.hurt[c.ID]
		if c.Kind != protocol.CharacterCreature || !c.Hostile || c.HPFrac <= 0 || (!(c.TargetingMe && (c.TargetingKnown == nil || *c.TargetingKnown)) && !hurt) {
			continue
		}

		if d := st.Pos.FlatDistance(c.Pos); d <= Near && (best == nil || hurt && (!bestHurt || hit.at.After(newest)) || hurt == bestHurt && (!hurt || hit.at.Equal(newest)) && d <= dist) {
			best, dist, bestHurt, newest = &seen.Characters[i], d, hurt, hit.at
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
