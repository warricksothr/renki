package burden

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

const (
	// Settle is how long after dropping it waits for the state and the
	// inventory to show the change before judging again.
	Settle = 3 * time.Second
	// SameSpot is how close a new drop must be to an old one to count as
	// the same place.
	SameSpot = 3.0
	// Spread is how far from a remembered place its things may lie: the
	// game throws a drop forward and it rolls, and later drops within
	// SameSpot join the same place.
	Spread = 5.0 + SameSpot
	// Look is how close the bot must be to a place it left things before
	// perception's word on them counts.
	Look = 15.0
	// GoneAfter is how long things must be missing from a place it is
	// looking at before they count as picked up.
	GoneAfter = 5 * time.Second
	// Retry is how long it waits after nothing could be dropped (not
	// allowed, or the plugin refused) before trying again.
	Retry = time.Minute
	// waitFor bounds the wait for one drop task's answer.
	waitFor = 5 * time.Second
)

// Session is what the unloader needs of the session.
type Session interface {
	Dispatch(ctx context.Context, o session.Order) (*session.Task, error)
	State() *protocol.State
	Inventory() *protocol.Inventory
	Perception() *protocol.Perception
	Handshake() (*protocol.Hello, *protocol.Welcome, bool)
}

// Unloader drops things when the bot carries too much, and keeps track of
// what it left until perception shows it gone.
type Unloader struct {
	on    bool
	s     Session
	store *home.Store
	log   *slog.Logger
	now   func() time.Time

	conn *protocol.Hello
	next time.Time
	// told is set once the bot has said why it is still carrying too
	// much (all gear, or it may not drop), so it says so once per overload.
	told bool
	// missing is when each remembered place's things were first missed,
	// keyed by world and position.
	missing map[string]time.Time
}

// New returns an unloader. on false leaves an overloaded bot as it is
// (config.Overload.Drop), but what it left is still tracked.
func New(on bool, s Session, store *home.Store, log *slog.Logger, now func() time.Time) *Unloader {
	if now == nil {
		now = time.Now
	}
	return &Unloader{on: on, s: s, store: store, log: log, now: now, missing: map[string]time.Time{}}
}

// Run calls Tick every second until ctx ends.
func (u *Unloader) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			u.Tick(ctx)
		}
	}
}

// Tick unloads an overloaded bot and checks on what it left behind.
func (u *Unloader) Tick(ctx context.Context) {
	// The state outlives a disconnect; there is nobody to drop anything.
	hello, welcome, ok := u.s.Handshake()
	st := u.s.State()
	if !ok || st == nil || st.Phase != protocol.PhaseInWorld || st.World == "" {
		return
	}
	if hello != u.conn {
		// A new connection: try at once, and say again what is wrong.
		u.conn, u.next, u.told = hello, time.Time{}, false
	}
	var seen protocol.PerceptionConfig
	if welcome != nil {
		seen = welcome.Perception
	}
	u.forgetGone(st, seen)
	if !st.Encumbered() {
		u.told = false
		return
	}
	inv := u.s.Inventory()
	if !u.on || inv == nil || u.now().Before(u.next) {
		return
	}
	u.next = u.now().Add(Settle)
	// The state's weight is rounded: over by less than it shows still
	// needs one thing dropped.
	drops, short := Plan(inv, max(st.Weight, st.WeightMax+0.01), st.WeightMax)
	if len(drops) == 0 {
		u.tell(ctx, "I'm carrying too much, and all of it is gear I won't leave behind.")
		return
	}
	left := map[string]int{}
	for _, d := range drops {
		n, err := u.drop(ctx, d)
		if errors.Is(err, session.ErrNotConnected) {
			// Gone mid-way: start again on the next connection.
			u.next = time.Time{}
			break
		}
		if n > 0 {
			left[d.Prefab] += n
		}
	}
	if len(left) > 0 {
		// Where it stands now: it may have walked on while dropping.
		pos := st.Pos
		if now := u.s.State(); now != nil {
			pos = now.Pos
		}
		u.remember(st.World, pos, left)
	}
	if u.next.IsZero() {
		return
	}
	if len(left) == 0 {
		// Not allowed (allowedTasks or the plugin's allowDrop), or refused:
		// say so once, and do not ask again every few seconds.
		u.next = u.now().Add(Retry)
		u.tell(ctx, "I'm carrying too much, and I can't put anything down.")
		return
	}
	line := fmt.Sprintf("I was carrying too much, so I left %s here. I'll remember where.", list(left))
	if short {
		line = fmt.Sprintf("I was carrying too much, so I left %s here. It's still too much, but the rest is gear.", list(left))
	}
	u.say(ctx, line)
}

// drop sends one drop task and returns how many it dropped, with the error
// when it could not be sent.
func (u *Unloader) drop(ctx context.Context, d Drop) (int, error) {
	t, err := u.s.Dispatch(ctx, session.Order{Task: protocol.TaskDrop, Args: &protocol.DropArgs{Prefab: d.Prefab, Count: d.Count}})
	if err != nil {
		u.log.Warn("could not drop", "prefab", d.Prefab, "count", d.Count, "err", err)
		return 0, err
	}
	wait, cancel := context.WithTimeout(ctx, waitFor)
	defer cancel()
	status, reason, err := t.Wait(wait)
	if err != nil || status == protocol.StatusCancelled {
		// No answer, or the link went before one came: drop is instant, so
		// the things may well be on the ground. Count them as left, and
		// let a later look forget them if they are not there.
		u.log.Warn("drop unconfirmed", "prefab", d.Prefab, "count", d.Count, "status", status, "err", err)
		return d.Count, nil
	}
	if status != protocol.StatusSucceeded {
		u.log.Warn("drop failed", "prefab", d.Prefab, "count", d.Count, "status", status, "reason", reason)
		return 0, nil
	}
	// "dropped N PREFAB" (7.2).
	_, _, detail := t.Status()
	var n int
	if _, err := fmt.Sscanf(detail, "dropped %d", &n); err != nil || n < 1 {
		n = d.Count
	}
	u.log.Info("dropped to lighten the load", "prefab", d.Prefab, "count", n)
	return n, nil
}

// remember adds what was left at pos to the place nearest it, or to a new one.
func (u *Unloader) remember(world string, pos protocol.Vec3, items map[string]int) {
	err := u.store.UpdateLeftBehind(world, func(caches []home.Cache) []home.Cache {
		for i := range caches {
			if caches[i].Pos.FlatDistance(pos) <= SameSpot {
				for p, n := range items {
					caches[i].Items[p] += n
				}
				caches[i].At = u.now()
				return caches
			}
		}
		return append(caches, home.Cache{Pos: pos, Items: maps.Clone(items), At: u.now(), Why: "overloaded"})
	})
	if err != nil {
		u.log.Warn("could not remember what was left", "err", err)
	}
}

// forgetGone drops from memory what perception, looking at a place up
// close, no longer sees there: someone picked it up. seen is what
// perception was told to report (welcome.perception).
func (u *Unloader) forgetGone(st *protocol.State, seen protocol.PerceptionConfig) {
	caches := u.store.LeftBehind(st.World)
	if len(caches) == 0 {
		return
	}
	p := u.s.Perception()
	if p == nil {
		return
	}
	var drops []protocol.PerceivedResource
	farthest := 0.0
	for _, r := range p.Resources {
		if r.Kind == protocol.ResourceDrop {
			drops = append(drops, r)
			// Perception ranks by distance in 3D.
			farthest = max(farthest, distance(r.Pos, st.Pos))
		}
	}
	changed := false
	for i, c := range caches {
		// Perception lists only the nearest drops within its radius. A
		// place partly beyond the radius, or beyond a full list, is not
		// all in view, and a miss there says nothing.
		reach := distance(c.Pos, st.Pos) + Spread
		blind := (seen.Radius > 0 && reach > seen.Radius) ||
			(seen.MaxPerKind > 0 && len(drops) >= seen.MaxPerKind && farthest < reach)
		if c.Pos.FlatDistance(st.Pos) > Look || blind {
			// Out of sight: a miss from an earlier visit does not count later.
			for prefab := range c.Items {
				delete(u.missing, missingKey(st.World, c.Pos, prefab))
			}
			continue
		}
		for prefab := range c.Items {
			key := missingKey(st.World, c.Pos, prefab)
			if slices.ContainsFunc(drops, func(r protocol.PerceivedResource) bool {
				return r.Prefab == prefab && r.Pos.FlatDistance(c.Pos) <= Spread
			}) {
				delete(u.missing, key)
				continue
			}
			since, ok := u.missing[key]
			if !ok {
				u.missing[key] = u.now()
				continue
			}
			if u.now().Sub(since) >= GoneAfter {
				delete(u.missing, key)
				delete(caches[i].Items, prefab)
				changed = true
				u.log.Info("what was left is gone", "prefab", prefab, "pos", c.Pos)
			}
		}
	}
	if !changed {
		return
	}
	if err := u.store.UpdateLeftBehind(st.World, func([]home.Cache) []home.Cache { return caches }); err != nil {
		u.log.Warn("could not forget what was picked up", "err", err)
	}
}

func distance(a, b protocol.Vec3) float64 {
	return math.Sqrt((a[0]-b[0])*(a[0]-b[0]) + (a[1]-b[1])*(a[1]-b[1]) + (a[2]-b[2])*(a[2]-b[2]))
}

func missingKey(world string, pos protocol.Vec3, prefab string) string {
	return fmt.Sprintf("%s|%.1f|%.1f|%s", world, pos[0], pos[2], prefab)
}

// tell says text once per overload.
func (u *Unloader) tell(ctx context.Context, text string) {
	if !u.told {
		u.told = true
		u.say(ctx, text)
	}
}

func (u *Unloader) say(ctx context.Context, text string) {
	if _, err := u.s.Dispatch(ctx, session.Order{Task: protocol.TaskSay,
		Args: &protocol.SayArgs{Text: text, Channel: protocol.ChannelShout}}); err != nil {
		u.log.Warn("could not say", "text", text, "err", err)
	}
}

// list names what was left, most first: "6 Wood and 2 Stone".
func list(items map[string]int) string {
	prefabs := slices.Collect(maps.Keys(items))
	slices.SortFunc(prefabs, func(a, b string) int {
		if items[a] != items[b] {
			return items[b] - items[a]
		}
		return strings.Compare(a, b)
	})
	parts := make([]string, len(prefabs))
	for i, p := range prefabs {
		parts[i] = fmt.Sprintf("%d %s", items[p], Name(p))
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}

// Name is a prefab as a person would say it: AxeBronze is "Axe Bronze".
func Name(prefab string) string {
	var b strings.Builder
	for i, r := range prefab {
		if i > 0 && r >= 'A' && r <= 'Z' && prefab[i-1] >= 'a' && prefab[i-1] <= 'z' {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}
