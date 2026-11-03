package behaviour

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// The deliberate-drop rule. A drop first seen within DeliberateReach of a
// live owner, with no tree, log or stump breaking within FellingReach of it
// from FellingWindow before to FellingLag after, was put down on purpose.
// The lag covers a split log whose wood is seen a frame before the log goes.
// Loot leaves a deliberate drop for DeliberateIgnore. The haul report (TKT-01M6DDFTDD) is to reuse this rule.
const (
	DeliberateReach  = 2.0
	FellingReach     = 10.0
	FellingWindow    = 10 * time.Second
	FellingLag       = 2 * time.Second
	DeliberateIgnore = 5 * time.Minute
)

// Break is a tree, log or stump that perception stopped showing.
type Break struct {
	Pos protocol.Vec3
	At  time.Time
}

// DeliberateDrop applies the deliberate-drop rule to a drop first seen at
// pos and time at, given live owner positions and the breaks seen so far.
// A break counts from FellingWindow before at until FellingLag after it.
func DeliberateDrop(pos protocol.Vec3, at time.Time, owners []protocol.Vec3, breaks []Break) bool {
	byOwner := false
	for _, o := range owners {
		if distance(pos, o) <= DeliberateReach {
			byOwner = true
			break
		}
	}
	if !byOwner {
		return false
	}
	for _, b := range breaks {
		if !b.At.After(at.Add(FellingLag)) && at.Sub(b.At) <= FellingWindow && distance(pos, b.Pos) <= FellingReach {
			return false
		}
	}
	return true
}

// LootItems are the felling products gather_trial.py checks.
var LootItems = []string{"Wood", "Resin", "BeechSeeds", "Feathers"}

// Loot picks up drops near the player Tas is with (lootAnchor), within a
// reach that grows while that player stays settled in one place (Settle,
// LootReach). One value
// is shared by every loot instance of an engine, because the drops' history
// outlives a run.
type Loot struct {
	mu      sync.Mutex
	world   string
	seq     uint64
	felling map[string]protocol.Vec3 // trees, logs and stumps in the last frame
	breaks  []Break
	drops   map[string]lootDrop
	settles map[settleKey]*Settle
	asked   map[settleKey]time.Time // when an instance last asked for a non-default shape
}

// settleKey names one settle tracker: a player and the area shape an
// instance asked for. Observe feeds every shape asked for so far.
type settleKey struct {
	player string
	area   SettleArea
}

// settleForget drops a tracker whose player has been unseen this long, or
// whose non-default shape no instance has asked for this long.
const settleForget = 10 * time.Minute

type lootDrop struct {
	seen     bool // in the last fresh frame
	pos      protocol.Vec3
	first    time.Time
	owners   []protocol.Vec3 // live owners at the first sighting
	lastSeen time.Time
	ignore   time.Time
}
type LootParams struct {
	Items             []string `json:"items,omitempty"`
	Radius            float64  `json:"radius,omitempty"`
	MaxWeightFraction float64  `json:"maxWeightFraction,omitempty"`
	MaxTargets        int      `json:"maxTargets,omitempty"`
	// Reach: BaseRadius until the anchor has been settled SettleDelayS,
	// then growing to Radius over GrowS (LootReach). The settle area opens
	// at SettleRadius and, after SettleDelayS, widens to SettleMaxRadius;
	// SettleSlack is how far past its edge the anchor may step before it
	// resets (Settle). Zero means the default.
	BaseRadius      float64 `json:"baseRadius,omitempty"`
	GrowS           float64 `json:"growS,omitempty"`
	SettleDelayS    float64 `json:"settleDelayS,omitempty"`
	SettleRadius    float64 `json:"settleRadius,omitempty"`
	SettleMaxRadius float64 `json:"settleMaxRadius,omitempty"`
	SettleSlack     float64 `json:"settleSlack,omitempty"`
}

func (p LootParams) area() SettleArea {
	return SettleArea{Radius: p.SettleRadius, MaxRadius: p.SettleMaxRadius, Slack: p.SettleSlack,
		Delay: time.Duration(p.SettleDelayS * float64(time.Second))}
}

// defaultSettleArea is the shape Observe tracks for every commander in
// sight, so their settle time accrues before any instance asks.
var defaultSettleArea = lootDefaults(LootParams{}).area()

func decodeLoot(raw json.RawMessage) (LootParams, error) {
	var p LootParams
	if err := Decode(raw, &p); err != nil {
		return p, err
	}
	return lootDefaults(p), nil
}

func lootDefaults(p LootParams) LootParams {
	if p.Items == nil {
		p.Items = slices.Clone(LootItems)
	}
	if p.Radius == 0 {
		p.Radius = 30
	}
	if p.MaxWeightFraction == 0 {
		p.MaxWeightFraction = .8
	}
	if p.MaxTargets == 0 {
		p.MaxTargets = 16
	}
	// A radius under the usual base keeps its flat reach.
	if p.BaseRadius == 0 {
		p.BaseRadius = min(8, p.Radius)
	}
	for _, d := range []struct {
		field *float64
		value float64
	}{{&p.GrowS, 120}, {&p.SettleDelayS, 5}, {&p.SettleRadius, 6}, {&p.SettleSlack, 2}} {
		if *d.field == 0 {
			*d.field = d.value
		}
	}
	// A wider opening circle than the usual maximum does not widen.
	if p.SettleMaxRadius == 0 {
		p.SettleMaxRadius = max(10, p.SettleRadius)
	}
	return p
}
func (*Loot) Validate(raw json.RawMessage) error {
	p, err := decodeLoot(raw)
	if err != nil {
		return err
	}
	if len(p.Items) == 0 || len(p.Items) > 64 {
		return errors.New("loot needs 1–64 item prefabs")
	}
	for _, name := range p.Items {
		if name == "" || slices.Contains([]string{"*", "wood", "ore", "food"}, name) {
			return errors.New("loot needs exact item prefabs")
		}
	}
	if !finite(p.Radius) || p.Radius <= 0 || p.Radius > 40 {
		return errors.New("loot radius must be above 0 and at most 40 m")
	}
	if !finite(p.MaxWeightFraction) || p.MaxWeightFraction < .1 || p.MaxWeightFraction > .9 {
		return errors.New("loot weight fraction must be between 0.1 and 0.9")
	}
	if !finite(p.BaseRadius) || p.BaseRadius <= 0 || p.BaseRadius > p.Radius {
		return errors.New("loot base radius must be above 0 and at most its radius")
	}
	if !finite(p.GrowS) || p.GrowS <= 0 || p.GrowS > 600 {
		return errors.New("loot growS must be above 0 and at most 600 s")
	}
	if !finite(p.SettleDelayS) || p.SettleDelayS <= 0 || p.SettleDelayS > 120 {
		return errors.New("loot settleDelayS must be above 0 and at most 120 s")
	}
	if !finite(p.SettleRadius) || p.SettleRadius <= 0 || !finite(p.SettleMaxRadius) || p.SettleMaxRadius < p.SettleRadius || p.SettleMaxRadius > 20 {
		return errors.New("loot settle radius must be above 0, and its maximum at least that and at most 20 m")
	}
	if !finite(p.SettleSlack) || p.SettleSlack <= 0 || p.SettleSlack > 10 {
		return errors.New("loot settleSlack must be above 0 and at most 10 m")
	}
	if p.MaxTargets < 1 || p.MaxTargets > 64 {
		return errors.New("loot needs 1–64 targets")
	}
	return nil
}
func (l *Loot) Start(raw json.RawMessage) (Runner, error) {
	if err := l.Validate(raw); err != nil {
		return nil, err
	}
	p, _ := decodeLoot(raw)
	c := &collectRun{p: CollectParams{Items: p.Items, MaxWeightFraction: p.MaxWeightFraction, Drops: true, CarryOnly: true}, seen: map[string]bool{}}
	return &lootRun{l: l, p: p, c: c}, nil
}

// Observe records breaks and first sightings from each new fresh frame.
func (l *Loot) Observe(v View) {
	if !freshPerception(v) {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if v.State.World != l.world {
		l.world, l.seq, l.felling, l.breaks, l.drops, l.settles, l.asked = v.State.World, 0, nil, nil, nil, nil, nil
	}
	if l.drops != nil && v.Perception.Seq == l.seq {
		return
	}
	l.seq = v.Perception.Seq
	now := v.Now
	felling := map[string]protocol.Vec3{}
	drops := map[string]bool{}
	for _, r := range v.Perception.Resources {
		switch r.Kind {
		case protocol.ResourceTree, protocol.ResourceLog, protocol.ResourceStump:
			felling[r.ID] = r.Pos
		case protocol.ResourceDrop:
			drops[r.ID] = true
		}
	}
	for id, pos := range l.felling {
		if _, ok := felling[id]; !ok {
			l.breaks = append(l.breaks, Break{Pos: pos, At: now})
		}
	}
	l.felling = felling
	l.observeSettles(v)
	l.breaks = slices.DeleteFunc(l.breaks, func(b Break) bool { return now.Sub(b.At) > FellingWindow })
	owners := liveOwners(v)
	if l.drops == nil {
		l.drops = map[string]lootDrop{}
	}
	for id, d := range l.drops {
		// A drop is judged once: its verdict outlives a missed frame or a
		// walk out of range, and is forgotten only after DeliberateIgnore away.
		if drops[id] {
			d.lastSeen = now
		} else if now.Sub(d.lastSeen) > DeliberateIgnore && !now.Before(d.ignore) {
			delete(l.drops, id)
			continue
		}
		d.seen = drops[id]
		// Within FellingLag of its first sighting, a later break can still
		// show that a drop came from felling.
		if !d.ignore.IsZero() && now.Sub(d.first) <= FellingLag && !DeliberateDrop(d.pos, d.first, d.owners, l.breaks) {
			d.ignore = time.Time{}
		}
		l.drops[id] = d
	}
	for _, r := range v.Perception.Resources {
		if r.Kind != protocol.ResourceDrop || r.ID == "" {
			continue
		}
		if _, ok := l.drops[r.ID]; ok {
			continue
		}
		d := lootDrop{seen: true, pos: r.Pos, first: now, owners: owners, lastSeen: now}
		if DeliberateDrop(r.Pos, now, owners, l.breaks) {
			d.ignore = now.Add(DeliberateIgnore)
		}
		l.drops[r.ID] = d
	}
}

// observeSettles feeds each live commander in sight to their trackers.
func (l *Loot) observeSettles(v View) {
	if l.settles == nil {
		l.settles = map[settleKey]*Settle{}
	}
	for _, p := range v.Perception.Characters {
		if p.Kind != protocol.CharacterPlayer || p.HPFrac <= 0 || !slices.Contains(v.Owners, p.Name) {
			continue
		}
		if k := (settleKey{p.Name, defaultSettleArea}); l.settles[k] == nil {
			l.settles[k] = &Settle{}
		}
		for k, s := range l.settles {
			if k.player == p.Name {
				s.Observe(p.Pos, v.Now, k.area)
			}
		}
	}
	for k, s := range l.settles {
		if v.Now.Sub(s.Last) > settleForget || k.area != defaultSettleArea && v.Now.Sub(l.asked[k]) > settleForget {
			delete(l.settles, k)
			delete(l.asked, k)
		}
	}
}

// reach is loot's radius around player under p at now. An area shape not
// tracked yet starts being tracked, from the next frame.
func (l *Loot) reach(player string, p LootParams, now time.Time) float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return LootReach(l.tracker(player, p.area(), now).Settled(now), p)
}

// Settled implements Passive with the tracker for the instance's area
// shape: the player has stayed settleDelayS in their current area.
func (l *Loot) Settled(raw json.RawMessage, player string, now time.Time) (time.Time, bool) {
	p, err := decodeLoot(raw)
	if err != nil {
		return time.Time{}, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	s := l.tracker(player, p.area(), now)
	settled := s.Settled(now)
	return s.Since, settled > 0 && settled >= p.area().Delay
}

// tracker is player's settle tracker for area, holding l.mu. A shape not
// tracked yet starts being tracked, from the next frame.
func (l *Loot) tracker(player string, area SettleArea, now time.Time) *Settle {
	k := settleKey{player, area}
	if k.area != defaultSettleArea {
		if l.asked == nil {
			l.asked = map[settleKey]time.Time{}
		}
		l.asked[k] = now
	}
	s := l.settles[k]
	if s == nil {
		if l.settles == nil {
			l.settles = map[settleKey]*Settle{}
		}
		s = &Settle{Last: now}
		l.settles[k] = s
	}
	return s
}

// eligible reports whether a drop has been observed and is not ignored.
func (l *Loot) eligible(id string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	d, ok := l.drops[id]
	return ok && d.seen && !now.Before(d.ignore)
}

// Ready lets a loot instance queue only when it has work, and only while
// Tas is idle or following (the engine lends a follow to passive ones).
func (l *Loot) Ready(raw json.RawMessage, v View) bool {
	if !v.Idle && v.Following == nil {
		return false
	}
	r, err := l.Start(raw)
	if err != nil {
		return false
	}
	return r.(*lootRun).pick(v) != nil
}

func sameDefinition(a, b Definition) bool {
	x, errA := json.Marshal(a)
	y, errB := json.Marshal(b)
	return errA == nil && errB == nil && bytes.Equal(x, y)
}

func liveOwners(v View) []protocol.Vec3 {
	var out []protocol.Vec3
	if !freshPerception(v) {
		return nil
	}
	for _, p := range v.Perception.Characters {
		if p.Kind == protocol.CharacterPlayer && p.HPFrac > 0 && slices.Contains(v.Owners, p.Name) {
			out = append(out, p.Pos)
		}
	}
	return out
}

type lootRun struct {
	l      *Loot
	p      LootParams
	c      *collectRun // collect's capacity, take and receipt bookkeeping
	anchor string      // the player the last pick worked around
	picked int         // pickups the pack confirmed
}

func (r *lootRun) Summary() string {
	if r.anchor == "" {
		return "completed"
	}
	return fmt.Sprintf("picked up %d near %s", r.picked, r.anchor)
}

// Recover lets a vanished drop go: the game or another player took it.
func (r *lootRun) Recover(result Result) bool {
	if r.c.task == protocol.TaskPickup && result.Status == protocol.StatusRejected && result.Reason == protocol.ReasonInvalidTarget {
		return true
	}
	return r.c.Recover(result)
}

// lootAnchor finds the player loot works around: the one View.Anchor names,
// while they still command Tas and are alive in sight. Until anyone has
// given an order, a single commander in sight stands in; with several in
// sight and none chosen, loot waits rather than guess.
func lootAnchor(v View) (string, protocol.Vec3, bool) {
	if !freshPerception(v) {
		return "", protocol.Vec3{}, false
	}
	var name string
	var pos protocol.Vec3
	found := 0
	for _, p := range v.Perception.Characters {
		if p.Kind != protocol.CharacterPlayer || p.HPFrac <= 0 || !slices.Contains(v.Owners, p.Name) {
			continue
		}
		if v.Anchor != "" {
			if p.Name == v.Anchor {
				return p.Name, p.Pos, true
			}
			continue
		}
		name, pos = p.Name, p.Pos
		found++
	}
	return name, pos, v.Anchor == "" && found == 1
}

func (r *lootRun) pick(v View) *protocol.PerceivedResource {
	if v.Inventory == nil || !freshPerception(v) {
		return nil
	}
	anchor, at, ok := lootAnchor(v)
	if !ok {
		return nil
	}
	r.anchor = anchor
	reach := r.l.reach(anchor, r.p, v.Now)
	var best *protocol.PerceivedResource
	for _, d := range v.Perception.Resources {
		if d.Kind != protocol.ResourceDrop || d.ID == "" || r.c.seen[d.ID] || !r.c.fits(v, d.Yield) || !r.l.eligible(d.ID, v.Now) {
			continue
		}
		if distance(d.Pos, at) > reach {
			continue
		}
		if best == nil || targetCost(v, d.Pos) < targetCost(v, best.Pos) || targetCost(v, d.Pos) == targetCost(v, best.Pos) && d.ID < best.ID {
			copy := d
			best = &copy
		}
	}
	return best
}

func (r *lootRun) Next(v View, result *Result) (*session.Order, error) {
	if v.State == nil {
		return nil, ErrWait
	}
	if v.Inventory == nil {
		return nil, ErrWait
	}
	if r.c.task != "" {
		if result == nil {
			return nil, errors.New("missing pickup result")
		}
		if result.Status == protocol.StatusSucceeded {
			// The pickup's receipt is a newer pack that holds more of it.
			if v.Inventory.Seq <= r.c.stepSeq {
				return nil, ErrWait
			}
			if r.c.productGain(v) <= 0 {
				return nil, fmt.Errorf("pickup of %s succeeded but the pack did not gain any", r.c.yield.Prefab)
			}
			r.picked++
		}
		r.c.task = ""
	}
	if !freshPerception(v) {
		return nil, ErrWait
	}
	if r.c.targets >= r.p.MaxTargets {
		return nil, nil
	}
	target := r.pick(v)
	if target == nil {
		return nil, nil
	}
	return r.c.take(v, *target), nil
}

// LootInstance is the built-in loot instance's name.
const LootInstance = "Loot"

// LootDefinition is the built-in loot instance: on and passive, below
// owner orders and gathering, above idle chores. It runs while Tas is idle
// within 40 m of a live owner, or borrows an owner's follow while the
// followed player is settled, and queues only when an eligible drop is in
// reach of its anchor player.
func LootDefinition() Definition {
	d := idleLootDefinition()
	d.Mode = ModePassive
	return d
}

// idleLootDefinition is the built-in instance as installed before it was
// passive, which InstallLoot upgrades.
func idleLootDefinition() Definition {
	// The reach fields stay unset, so they follow the defaults and an owner
	// who edits only radius is not held to a saved base.
	params, _ := json.Marshal(LootParams{Items: LootItems, Radius: 30, MaxWeightFraction: .8, MaxTargets: 16})
	return Definition{Template: "loot", Params: params, Trigger: Trigger{Kind: "timer", EveryS: 2},
		When: Conditions{Owner: &OwnerNear{Radius: 40}}, Repeat: Repeat{Mode: "trigger", CooldownS: 2}, Priority: -5, Enabled: true}
}

// InstallLoot creates the built-in loot instance when no instance of that
// name is saved, including an invalid one. Disable is the off switch;
// deleting the instance brings it back on the next start. A saved Loot
// that is still the old idle built-in, on or off, becomes passive; one an
// owner edited stays as they left it.
func (e *Engine) InstallLoot(by string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if i, ok := e.instances[LootInstance]; ok {
		old := idleLootDefinition()
		old.Enabled = i.definition.Enabled
		if !sameDefinition(i.definition, old) {
			return nil
		}
		d := LootDefinition()
		d.Enabled = i.definition.Enabled
		if err := e.put(context.Background(), LootInstance, d, by); err != nil {
			e.o.Logger.Warn("could not make the built-in loot instance passive", "err", err)
			return err
		}
		e.o.Logger.Info("built-in behaviour made passive", "instance", LootInstance)
		return nil
	}
	if _, ok := e.o.Store.Instances()[LootInstance]; ok {
		return nil
	}
	if err := e.put(context.Background(), LootInstance, LootDefinition(), by); err != nil {
		e.o.Logger.Warn("could not install the built-in loot instance", "err", err)
		return err
	}
	e.o.Logger.Info("built-in behaviour installed", "instance", LootInstance)
	return nil
}
