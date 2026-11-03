package behaviour

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

type LootAnchorStatus struct {
	Name string        `json:"name"`
	Pos  protocol.Vec3 `json:"pos"`
}
type LootSettleStatus struct {
	Centre   protocol.Vec3 `json:"centre"`
	Radius   float64       `json:"radius"`
	Slack    float64       `json:"slack"`
	Since    time.Time     `json:"since"`
	SettledS float64       `json:"settledS"`
	Ready    bool          `json:"ready"`
}
type LootDropStatus struct {
	ID       string        `json:"id"`
	Prefab   string        `json:"prefab"`
	Pos      protocol.Vec3 `json:"pos"`
	Decision string        `json:"decision"`
}

// LootStatus explains candidates only. Admission, instance enablement, and the
// native task's safety checks still decide whether a pickup can actually run.
type LootStatus struct {
	Connection uint64            `json:"connection"`
	World      string            `json:"world"`
	WorldUID   string            `json:"worldUid,omitempty"`
	At         time.Time         `json:"at"`
	FrameAt    time.Time         `json:"frameAt"`
	Seq        uint64            `json:"seq"`
	Fresh      bool              `json:"fresh"`
	Anchor     *LootAnchorStatus `json:"anchor,omitempty"`
	Settle     *LootSettleStatus `json:"settle,omitempty"`
	Reach      float64           `json:"reach"`
	Params     LootParams        `json:"params"`
	Defaults   LootParams        `json:"defaults"`
	Drops      []LootDropStatus  `json:"drops"`
}

func diagnosticFresh(v View) bool {
	if !freshPerception(v) || v.State.Phase != protocol.PhaseInWorld || v.State.HP <= 0 {
		return false
	}
	for _, at := range []time.Time{v.StateAt, v.PerceptionAt} {
		if !at.IsZero() && (v.Now.Before(at) || v.Now.Sub(at) > 5*time.Second) {
			return false
		}
	}
	return true
}
func (l *Loot) Inspect(raw json.RawMessage, v View, runner Runner) *LootStatus {
	p, err := decodeLoot(raw)
	if err != nil {
		return nil
	}
	s := &LootStatus{Connection: v.Connection, WorldUID: v.WorldUID, At: v.Now, FrameAt: v.PerceptionAt, Params: p, Defaults: lootDefaults(LootParams{}), Drops: []LootDropStatus{}}
	if v.State != nil {
		s.World = v.State.World
	}
	if !diagnosticFresh(v) {
		return s
	}
	l.Observe(v)
	s.Seq = v.Perception.Seq
	s.Fresh = true
	player, at, ok := lootAnchor(v)
	if !ok {
		return s
	}
	s.Anchor = &LootAnchorStatus{Name: player, Pos: at}
	l.mu.Lock()
	settle := *l.tracker(player, p.area(), v.Now)
	l.mu.Unlock()
	settled := settle.Settled(v.Now)
	if !settle.Since.IsZero() {
		s.Settle = &LootSettleStatus{Centre: settle.Centre, Radius: settle.Radius, Slack: p.SettleSlack, Since: settle.Since, SettledS: settled.Seconds(), Ready: settled >= p.area().Delay}
	}
	s.Reach = LootReach(settled, p)
	r, ok := runner.(*lootRun)
	if !ok {
		started, err := l.Start(raw)
		if err != nil {
			return nil
		}
		r = started.(*lootRun)
	}
	for _, d := range v.Perception.Resources {
		if d.Kind != protocol.ResourceDrop || d.ID == "" {
			continue
		}
		prefab := d.Prefab
		if d.Yield != nil {
			prefab = d.Yield.Prefab
		}
		s.Drops = append(s.Drops, LootDropStatus{ID: d.ID, Prefab: prefab, Pos: d.Pos, Decision: r.decision(v, d, at, s.Reach)})
	}
	slices.SortFunc(s.Drops, func(a, b LootDropStatus) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return s
}
func (r *lootRun) decision(v View, d protocol.PerceivedResource, at protocol.Vec3, reach float64) string {
	if d.Kind != protocol.ResourceDrop || d.ID == "" {
		return "not_drop"
	}
	if r.c.seen[d.ID] {
		return "attempted"
	}
	r.l.mu.Lock()
	observed, ok := r.l.drops[d.ID]
	r.l.mu.Unlock()
	if !ok || !observed.seen {
		return "unobserved"
	}
	if v.Now.Before(observed.ignore) {
		return "deliberate"
	}
	if distance(d.Pos, at) > reach {
		return "out_of_reach"
	}
	if d.Yield == nil {
		return "unknown_yield"
	}
	if !slices.Contains(r.p.Items, d.Yield.Prefab) {
		return "item_not_selected"
	}
	if v.Inventory == nil {
		return "inventory_unknown"
	}
	if !r.c.fits(v, d.Yield) {
		return "capacity"
	}
	if r.c.targets >= r.p.MaxTargets {
		return "target_limit"
	}
	return "eligible"
}
