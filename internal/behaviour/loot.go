package behaviour

import (
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
// in the FellingWindow before, was put down on purpose. Loot leaves it for
// DeliberateIgnore. The haul report (TKT-01M6DDFTDD) is to reuse this rule.
const (
	DeliberateReach  = 2.0
	FellingReach     = 10.0
	FellingWindow    = 10 * time.Second
	DeliberateIgnore = 5 * time.Minute
)

// Break is a tree, log or stump that perception stopped showing.
type Break struct {
	Pos protocol.Vec3
	At  time.Time
}

// DeliberateDrop applies the deliberate-drop rule to a drop first seen at
// pos and time at, given live owner positions and the breaks seen so far.
// A break counts when it is at most FellingWindow before at, never after.
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
		if !b.At.After(at) && at.Sub(b.At) <= FellingWindow && distance(pos, b.Pos) <= FellingReach {
			return false
		}
	}
	return true
}

// LootItems are the felling products gather_trial.py checks.
var LootItems = []string{"Wood", "Resin", "BeechSeeds", "Feathers"}

// Loot picks up drops near a live owner. One value is shared by every loot
// instance of an engine, because the drops' history outlives a run.
type Loot struct {
	mu      sync.Mutex
	world   string
	seq     uint64
	felling map[string]protocol.Vec3 // trees, logs and stumps in the last frame
	breaks  []Break
	drops   map[string]lootDrop
}
type lootDrop struct {
	seen     bool // in the last fresh frame
	lastSeen time.Time
	ignore   time.Time
}
type LootParams struct {
	Items             []string `json:"items,omitempty"`
	Radius            float64  `json:"radius,omitempty"`
	MaxWeightFraction float64  `json:"maxWeightFraction,omitempty"`
	MaxTargets        int      `json:"maxTargets,omitempty"`
	PreemptFollow     bool     `json:"preemptFollow,omitempty"`
}

func decodeLoot(raw json.RawMessage) (LootParams, error) {
	var p LootParams
	if err := Decode(raw, &p); err != nil {
		return p, err
	}
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
	return p, nil
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
	if p.MaxTargets < 1 || p.MaxTargets > 64 {
		return errors.New("loot needs 1–64 targets")
	}
	if p.PreemptFollow {
		return errors.New("loot cannot pause a follow yet: that waits on the owners' agreement (TKT-01M62VXYQB)")
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
		l.world, l.seq, l.felling, l.breaks, l.drops = v.State.World, 0, nil, nil, nil
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
		l.drops[id] = d
	}
	for _, r := range v.Perception.Resources {
		if r.Kind != protocol.ResourceDrop || r.ID == "" {
			continue
		}
		if _, ok := l.drops[r.ID]; ok {
			continue
		}
		d := lootDrop{seen: true, lastSeen: now}
		if DeliberateDrop(r.Pos, now, owners, l.breaks) {
			d.ignore = now.Add(DeliberateIgnore)
		}
		l.drops[r.ID] = d
	}
}

// eligible reports whether a drop has been observed and is not ignored.
func (l *Loot) eligible(id string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	d, ok := l.drops[id]
	return ok && d.seen && !now.Before(d.ignore)
}

// Ready lets an idle loot instance queue only when it has work.
func (l *Loot) Ready(raw json.RawMessage, v View) bool {
	if !v.Idle {
		return false
	}
	r, err := l.Start(raw)
	if err != nil {
		return false
	}
	return r.(*lootRun).pick(v) != nil
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
	l *Loot
	p LootParams
	c *collectRun // collect's capacity, take and receipt bookkeeping
}

// Recover lets a vanished drop go: the game or another player took it.
func (r *lootRun) Recover(result Result) bool {
	if r.c.task == protocol.TaskPickup && result.Status == protocol.StatusRejected && result.Reason == protocol.ReasonInvalidTarget {
		return true
	}
	return r.c.Recover(result)
}

func (r *lootRun) pick(v View) *protocol.PerceivedResource {
	if v.Inventory == nil || !freshPerception(v) {
		return nil
	}
	owners := liveOwners(v)
	var best *protocol.PerceivedResource
	for _, d := range v.Perception.Resources {
		if d.Kind != protocol.ResourceDrop || d.ID == "" || r.c.seen[d.ID] || !r.c.fits(v, d.Yield) || !r.l.eligible(d.ID, v.Now) {
			continue
		}
		near := false
		for _, o := range owners {
			if distance(d.Pos, o) <= r.p.Radius {
				near = true
				break
			}
		}
		if !near {
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

// LootDefinition is the built-in loot instance: on, below owner orders and
// gathering, above idle chores. It runs only while Tas is idle within 40 m
// of a live owner, and queues only when an eligible drop is in reach.
func LootDefinition() Definition {
	params, _ := json.Marshal(LootParams{Items: LootItems, Radius: 30, MaxWeightFraction: .8, MaxTargets: 16})
	return Definition{Template: "loot", Params: params, Trigger: Trigger{Kind: "timer", EveryS: 2},
		When: Conditions{Owner: &OwnerNear{Radius: 40}}, Repeat: Repeat{Mode: "trigger", CooldownS: 2}, Priority: -5, Enabled: true}
}

// InstallLoot creates the built-in loot instance when no instance of that
// name is saved, including an invalid one. Disable is the off switch;
// deleting the instance brings it back on the next start.
func (e *Engine) InstallLoot(by string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.instances[LootInstance]; ok {
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
