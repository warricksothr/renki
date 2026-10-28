// Package home remembers where the bot lives in each world, sends it there
// when told, and sends it there by itself when it has lost the plot: after
// an outage cut a task short, or when it has stood idle away from home.
// Going home after a failed follow is the chat brain's call, since it knows
// who asked and on which channel; it asks Keeper to do it.
package home

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// Errors from Keeper.
var (
	ErrNoWorld = errors.New("the bot is not in a world")
	ErrNoHome  = errors.New("no home is set in this world")
)

// Arrive is how close to home counts as being there.
const Arrive = 2.0

// awayFrom is how far from home the bot must be before idling sends it
// back, so it does not shuffle on the spot.
const awayFrom = 5.0

// Session is what Keeper needs of the session.
type Session interface {
	Dispatch(ctx context.Context, o session.Order) (*session.Task, error)
	State() *protocol.State
	Handshake() (*protocol.Hello, *protocol.Welcome, bool)
}

// Keeper owns the bot's homes and the rules for going back to them.
type Keeper struct {
	store *Store
	rules config.HomeRules
	s     Session
	log   *slog.Logger
	now   func() time.Time

	mu sync.Mutex
	// homeward holds the ids of tasks sent to take the bot home.
	homeward map[string]bool
	// conn is the hello of the connection Tick last saw, to notice a new one.
	conn *protocol.Hello
	// checked is set once the outage check has run for conn.
	checked bool
	// active is the last time the bot had a task or was given an order.
	active time.Time
	// respawned is set by Respawned and cleared once Tick has acted on it.
	respawned bool
}

// New returns a keeper over store. now may be nil.
func New(store *Store, rules config.HomeRules, s Session, log *slog.Logger, now func() time.Time) *Keeper {
	if log == nil {
		log = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	return &Keeper{store: store, rules: rules, s: s, log: log, now: now, homeward: map[string]bool{}, active: now()}
}

// World is the name of the world the bot is in, or "".
func (k *Keeper) World() string {
	if st := k.s.State(); st != nil && st.World != "" {
		return st.World
	}
	if hello, _, ok := k.s.Handshake(); ok {
		return hello.World
	}
	return ""
}

// SetHome makes pos home in the bot's current world, and returns the world.
func (k *Keeper) SetHome(pos protocol.Vec3, by string) (string, error) {
	world := k.World()
	if world == "" {
		return "", ErrNoWorld
	}
	if err := k.store.SetHome(world, Home{Pos: pos, SetBy: by, At: k.now()}); err != nil {
		return world, err
	}
	k.log.Info("home set", "world", world, "pos", pos, "by", by)
	return world, nil
}

// Current returns the home in the bot's current world.
func (k *Keeper) Current() (world string, h Home, ok bool) {
	world = k.World()
	if world == "" {
		return "", Home{}, false
	}
	h, ok = k.store.Home(world)
	return world, h, ok
}

// Rules are the rules Keeper follows.
func (k *Keeper) Rules() config.HomeRules { return k.rules }

// HomeDistance is how far the bot is from home, flat, and false when
// either is unknown.
func (k *Keeper) HomeDistance() (float64, bool) {
	st := k.s.State()
	if st == nil || st.Phase != protocol.PhaseInWorld {
		return 0, false
	}
	h, ok := k.store.Home(k.World())
	if !ok {
		return 0, false
	}
	return flat(st.Pos, h.Pos), true
}

// GoHome sends the bot home. origin rides on the task as with any order.
func (k *Keeper) GoHome(ctx context.Context, origin any, why string) (*session.Task, error) {
	world := k.World()
	if world == "" {
		return nil, ErrNoWorld
	}
	h, ok := k.store.Home(world)
	if !ok {
		return nil, ErrNoHome
	}
	stop := Arrive
	run := true
	if d, ok := k.HomeDistance(); ok && d < 15 {
		run = false
	}
	t, err := k.s.Dispatch(ctx, session.Order{Task: protocol.TaskMoveTo, Origin: origin,
		Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &h.Pos}, StopDistance: &stop, Run: &run}})
	if err != nil {
		return nil, err
	}
	k.mu.Lock()
	// Kept for the life of the controller: the brain asks Homeward from
	// TaskDone, which runs after the task's Done channel has closed, so
	// forgetting the id when it closes would race that question. Ids are
	// unique per run and there is one per trip home.
	k.homeward[t.ID] = true
	k.active = k.now()
	k.mu.Unlock()
	k.log.Info("going home", "why", why, "id", t.ID, "world", world, "home", h.Pos)
	return t, nil
}

// Homeward reports whether the task with id was sent to take the bot home.
func (k *Keeper) Homeward(id string) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.homeward[id]
}

// ReturnOn reports whether a body task failing for reason should send the
// bot home.
func (k *Keeper) ReturnOn(reason string) bool {
	switch reason {
	case protocol.ReasonTargetLost:
		return k.rules.WhenLost
	case protocol.ReasonStuck:
		return k.rules.WhenStuck
	}
	return false
}

// Respawned notes that the bot came back from death. Tick sends it home
// once it is in the world with nothing to do, if the rules allow.
func (k *Keeper) Respawned() {
	k.mu.Lock()
	k.respawned = true
	k.mu.Unlock()
}

// Touch counts as activity: a player gave an order.
func (k *Keeper) Touch() {
	k.mu.Lock()
	k.active = k.now()
	k.mu.Unlock()
}

// Run calls Tick every second until ctx ends.
func (k *Keeper) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			k.Tick(ctx)
		}
	}
}

// Tick looks at the newest state: it notices outages and idling and keeps
// the interrupted flag.
func (k *Keeper) Tick(ctx context.Context) {
	hello, _, ok := k.s.Handshake()
	st := k.s.State()
	if !ok || st == nil {
		return
	}
	now := k.now()
	k.mu.Lock()
	if hello != k.conn {
		// A new connection: the old one's tasks are gone, and the idle
		// clock starts again.
		k.conn, k.checked, k.active = hello, false, now
	}
	k.mu.Unlock()
	if st.Phase != protocol.PhaseInWorld {
		return
	}

	k.mu.Lock()
	first := !k.checked
	k.checked = true
	k.mu.Unlock()
	if first && k.store.Interrupted() {
		k.setInterrupted(false)
		if k.rules.AfterOutage && st.Task == nil {
			k.announce(ctx, "after an outage", "Something cut me off. Heading home.")
			return
		}
	}

	k.mu.Lock()
	respawned := k.respawned && st.Task == nil
	if respawned {
		k.respawned = false
	}
	k.mu.Unlock()
	if respawned && k.rules.AfterDeath {
		k.announce(ctx, "after a respawn", "I'm back. Heading home.")
		return
	}

	busy := st.Task != nil && body(st.Task.Task)
	k.setInterrupted(busy)
	k.mu.Lock()
	if st.Task != nil {
		k.active = now
	}
	idle := now.Sub(k.active)
	k.mu.Unlock()
	if k.rules.IdleMinutes <= 0 || st.Task != nil || idle < time.Duration(k.rules.IdleMinutes*float64(time.Minute)) {
		return
	}
	k.Touch() // whatever happens, wait another spell before trying again
	if d, ok := k.HomeDistance(); ok && d > awayFrom {
		k.announce(ctx, "idle", "Nothing to do. Heading home.")
	}
}

// announce sends the bot home and says so, for trips nobody asked for.
func (k *Keeper) announce(ctx context.Context, why, line string) {
	if _, err := k.GoHome(ctx, nil, why); err != nil {
		if !errors.Is(err, ErrNoHome) {
			k.log.Warn("could not go home", "why", why, "err", err)
		}
		return
	}
	if _, err := k.s.Dispatch(ctx, session.Order{Task: protocol.TaskSay, Args: &protocol.SayArgs{Text: line}}); err != nil {
		k.log.Warn("could not say", "text", line, "err", err)
	}
}

func (k *Keeper) setInterrupted(v bool) {
	if err := k.store.SetInterrupted(v); err != nil {
		k.log.Warn("could not save the interrupted flag", "err", err)
	}
}

// body reports whether task moves the bot, so an outage could leave it
// somewhere odd.
func body(task string) bool {
	return task == protocol.TaskFollow || task == protocol.TaskMoveTo
}

func flat(a, b protocol.Vec3) float64 { return math.Hypot(a[0]-b[0], a[2]-b[2]) }
