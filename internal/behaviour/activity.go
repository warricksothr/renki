package behaviour

import (
	"cmp"
	"slices"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

const (
	ActivityTTL   = 30 * time.Second
	ActivityReach = 12.0
	activityGap   = 5 * time.Second
)

// Activity is inferred nearby work, not proof of who caused a resource to vanish.
// It cannot grant permission to harvest or dispatch any work.
type Activity struct {
	Kind   string        `json:"kind"`
	Player string        `json:"player"`
	Target string        `json:"target"`
	Prefab string        `json:"prefab"`
	Pos    protocol.Vec3 `json:"pos"`
	At     time.Time     `json:"at"`
	AgeS   float64       `json:"ageS"`
}

// ActivityTracker consumes ordered fresh frames. Missing, capped or widely
// separated scans establish a new baseline rather than inventing break events.
type ActivityTracker struct {
	world, uid, player string
	conn, seq          uint64
	at                 time.Time
	gameT              float64
	bot, anchor        protocol.Vec3
	previous           map[string]protocol.PerceivedResource
	activity           *Activity
}

func activityKind(kind string) string {
	switch kind {
	case protocol.ResourceTree, protocol.ResourceLog, protocol.ResourceStump:
		return "felling"
	case protocol.ResourceRock:
		return "mining"
	case protocol.ResourcePickable:
		return "picking"
	}
	return ""
}
func (t *ActivityTracker) Observe(v View) *Activity {
	if v.State == nil || v.State.Phase != protocol.PhaseInWorld || v.State.HP <= 0 {
		*t = ActivityTracker{}
		return nil
	}
	player, anchor, ok := lootAnchor(v)
	if t.world != v.State.World || t.uid != v.WorldUID || t.conn != v.Connection || t.player != player {
		*t = ActivityTracker{world: v.State.World, uid: v.WorldUID, conn: v.Connection, player: player}
	}
	if !ok || !diagnosticFresh(v) {
		t.previous = nil
		return nil
	}
	snapshot := func() *Activity {
		if t.activity == nil || v.Now.Before(t.activity.At) || v.Now.Sub(t.activity.At) >= ActivityTTL {
			return nil
		}
		a := *t.activity
		a.AgeS = v.Now.Sub(a.At).Seconds()
		return &a
	}
	if t.previous != nil && v.Perception.Seq <= t.seq {
		if v.Perception.Seq < t.seq {
			return nil
		}
		return snapshot()
	}
	current := map[string]protocol.PerceivedResource{}
	counts := map[string]int{}
	for _, r := range v.Perception.Resources {
		counts[r.Kind]++
		if r.ID != "" && activityKind(r.Kind) != "" {
			current[r.ID] = r
		}
	}
	paired := t.previous != nil && v.Now.Sub(t.at) > 0 && v.Now.Sub(t.at) <= activityGap && v.Perception.T > t.gameT && distance(v.State.Pos, t.bot) <= 4
	var changes []protocol.PerceivedResource
	if paired {
		for id, r := range t.previous {
			if _, seen := current[id]; seen || distance(r.Pos, t.anchor) > ActivityReach || distance(r.Pos, anchor) > ActivityReach {
				continue
			}
			if v.PerceptionBounds.MaxPerKind > 0 && counts[r.Kind] >= v.PerceptionBounds.MaxPerKind {
				continue
			}
			if v.PerceptionBounds.Radius > 0 && distance(v.State.Pos, r.Pos) > v.PerceptionBounds.Radius-4 {
				continue
			}
			changes = append(changes, r)
		}
	}
	// Simultaneous mixed evidence picks the closest change, then kind and ID.
	slices.SortFunc(changes, func(a, b protocol.PerceivedResource) int {
		if c := cmp.Compare(distance(a.Pos, anchor), distance(b.Pos, anchor)); c != 0 {
			return c
		}
		if c := cmp.Compare(activityKind(a.Kind), activityKind(b.Kind)); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	if len(changes) > 0 {
		r := changes[0]
		observed := v.PerceptionAt
		if observed.IsZero() {
			observed = v.Now
		}
		t.activity = &Activity{Kind: activityKind(r.Kind), Player: player, Target: r.ID, Prefab: r.Prefab, Pos: r.Pos, At: observed}
	}
	t.previous = current
	t.at = v.Now
	t.seq = v.Perception.Seq
	t.gameT = v.Perception.T
	t.bot = v.State.Pos
	t.anchor = anchor
	return snapshot()
}
