package home

import (
	"cmp"
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

type fakeSession struct {
	// final makes Dispatch return tasks that have already failed, for
	// reason, or stuck when that is empty.
	final  bool
	reason string
	mu     sync.Mutex
	hello  *protocol.Hello
	state  *protocol.State
	orders []session.Order
}

func (f *fakeSession) Dispatch(_ context.Context, o session.Order) (*session.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.hello == nil {
		return nil, session.ErrNotConnected
	}
	f.orders = append(f.orders, o)
	if f.final {
		return session.Finished(o.Task, o.Origin, protocol.StatusFailed, cmp.Or(f.reason, protocol.ReasonStuck)), nil
	}
	return session.Finished(o.Task, o.Origin, protocol.StatusRunning, ""), nil
}

func (f *fakeSession) State() *protocol.State {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state
}

func (f *fakeSession) Handshake() (*protocol.Hello, *protocol.Welcome, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.hello, nil, f.hello != nil
}

func (f *fakeSession) set(hello *protocol.Hello, st *protocol.State) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hello, f.state = hello, st
}

// sent lists the tasks sent, as "move_to" or "say: text".
func (f *fakeSession) sent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, o := range f.orders {
		if a, ok := o.Args.(*protocol.SayArgs); ok {
			out = append(out, "say: "+a.Text)
		} else {
			out = append(out, o.Task)
		}
	}
	return out
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

var rules = config.HomeRules{WhenLost: true, WhenStuck: true, AfterOutage: true, IdleMinutes: 10, AfterDeath: config.AfterDeathHome}

func inWorld(pos protocol.Vec3, task *protocol.TaskRef) *protocol.State {
	return &protocol.State{Phase: protocol.PhaseInWorld, World: "Midgard", Pos: pos, Task: task}
}

func keeper(t *testing.T, store *Store) (*Keeper, *fakeSession, *clock) {
	t.Helper()
	f := &fakeSession{}
	c := &clock{t: time.Date(2026, 10, 28, 17, 0, 0, 0, time.UTC)}
	return New(store, rules, f, slog.New(slog.NewTextHandler(io.Discard, nil)), c.now), f, c
}

func TestHomesSurviveARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "memory.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	k, f, _ := keeper(t, store)
	if _, err := k.SetHome(protocol.Vec3{1, 2, 3}, "Nimah"); err != ErrNoWorld {
		t.Errorf("set home with no world: %v", err)
	}
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{10, 30, 10}, nil))
	if world, err := k.SetHome(protocol.Vec3{100, 30, -50}, "Nimah"); err != nil || world != "Midgard" {
		t.Fatalf("set home: %q %v", world, err)
	}

	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	h, ok := again.Home("Midgard")
	if !ok || h.Pos != (protocol.Vec3{100, 30, -50}) || h.SetBy != "Nimah" {
		t.Errorf("after reopening: %+v %v", h, ok)
	}
	if _, ok := again.Home("Elsewhere"); ok {
		t.Error("a home leaked into another world")
	}
}

func TestGoHomeWalksThereAndRunsWhenFar(t *testing.T) {
	k, f, _ := keeper(t, Memory())
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{0, 0, 0}, nil))
	if _, err := k.GoHome(context.Background(), nil, "test"); err != ErrNoHome {
		t.Errorf("go home with none: %v", err)
	}
	k.SetHome(protocol.Vec3{100, 0, 0}, "Nimah")
	task, err := k.GoHome(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	a := f.orders[0].Args.(*protocol.MoveToArgs)
	if *a.To.Pos != (protocol.Vec3{100, 0, 0}) || *a.StopDistance != Arrive || !*a.Run {
		t.Errorf("move_to %+v", a)
	}
	if !k.Homeward(task.ID) {
		t.Error("the trip home is not marked homeward")
	}
}

// TestATripHomeStaysHomewardAfterItEnds: the brain asks Homeward from
// TaskDone, after the task's Done channel has closed, so the mark must
// outlive the task.
func TestATripHomeStaysHomewardAfterItEnds(t *testing.T) {
	k, f, _ := keeper(t, Memory())
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{0, 0, 0}, nil))
	k.SetHome(protocol.Vec3{100, 0, 0}, "Nimah")
	f.final = true
	ended, err := k.GoHome(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	<-ended.Done()
	time.Sleep(20 * time.Millisecond)
	if !k.Homeward(ended.ID) {
		t.Error("a finished trip home lost its homeward mark")
	}
	if d, ok := k.HomeDistance(); !ok || d != 100 {
		t.Errorf("distance %v %v", d, ok)
	}
}

func TestCombatOutageReturnsHomeWithoutResumingAttack(t *testing.T) {
	store := Memory()
	k, f, _ := keeper(t, store)
	ctx := context.Background()
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{}, nil))
	k.SetHome(protocol.Vec3{50, 0, 0}, "Nimah")
	k.Tick(ctx)
	f.set(f.hello, inWorld(protocol.Vec3{20, 0, 0}, &protocol.TaskRef{ID: "a", Task: protocol.TaskAttack, Status: protocol.StatusRunning}))
	k.Tick(ctx)
	if !store.Interrupted() {
		t.Fatal("combat interruption was not remembered")
	}
	f.set(nil, nil)
	k.Tick(ctx)
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{20, 0, 0}, nil))
	k.Tick(ctx)
	if got := f.sent(); len(got) != 2 || got[0] != "move_to" {
		t.Fatal(got)
	}
}

func TestOutageSendsTheBotHome(t *testing.T) {
	store := Memory()
	k, f, _ := keeper(t, store)
	ctx := context.Background()
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{0, 0, 0}, nil))
	k.SetHome(protocol.Vec3{50, 0, 0}, "Nimah")
	k.Tick(ctx)

	// Following when the connection drops.
	f.set(f.hello, inWorld(protocol.Vec3{20, 0, 0}, &protocol.TaskRef{ID: "t-1", Task: "follow", Status: "running"}))
	k.Tick(ctx)
	if !store.Interrupted() {
		t.Fatal("a running follow did not set the interrupted flag")
	}
	f.set(nil, nil)
	k.Tick(ctx)

	// Back on a new connection, at the menu and then in the world.
	f.set(&protocol.Hello{}, &protocol.State{Phase: protocol.PhaseMenu})
	k.Tick(ctx)
	if len(f.sent()) != 0 {
		t.Fatalf("sent %v before the bot was in the world", f.sent())
	}
	f.set(f.hello, inWorld(protocol.Vec3{20, 0, 0}, nil))
	k.Tick(ctx)
	if got := f.sent(); len(got) != 2 || got[0] != "move_to" || got[1] != "say: Something cut me off. Heading home." {
		t.Errorf("after the outage: %v", got)
	}
	if store.Interrupted() {
		t.Error("the flag is still set")
	}
	k.Tick(ctx)
	if len(f.sent()) != 2 {
		t.Errorf("went home twice: %v", f.sent())
	}
}

func TestOutageWaitsUntilTheBotCanMove(t *testing.T) {
	store := Memory()
	k, f, _ := keeper(t, store)
	ctx := context.Background()
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{0, 0, 0}, nil))
	k.SetHome(protocol.Vec3{50, 0, 0}, "Nimah")
	k.Tick(ctx)
	store.SetInterrupted(true)

	// Back on a new connection, carrying too much to move.
	st := inWorld(protocol.Vec3{20, 0, 0}, nil)
	st.Flags = []string{protocol.FlagEncumbered, protocol.FlagStranded}
	f.set(&protocol.Hello{}, st)
	k.Tick(ctx)
	if len(f.sent()) != 0 || !store.Interrupted() {
		t.Fatalf("sent %v, interrupted %v, while stranded", f.sent(), store.Interrupted())
	}
	if _, err := k.GoHome(ctx, nil, "test"); err != ErrStranded {
		t.Errorf("GoHome while stranded: %v", err)
	}
	st.Flags = []string{protocol.FlagEncumbered}
	k.Tick(ctx)
	if got := f.sent(); len(got) != 2 || got[1] != "say: Something cut me off. Heading home." {
		t.Errorf("once it can move: %v", got)
	}
}

func TestAFollowThatEndsStrandedIsNoOutage(t *testing.T) {
	store := Memory()
	k, f, _ := keeper(t, store)
	ctx := context.Background()
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{0, 0, 0}, &protocol.TaskRef{ID: "t-1", Task: "follow", Status: "running"}))
	k.SetHome(protocol.Vec3{50, 0, 0}, "Nimah")
	k.Tick(ctx)
	if !store.Interrupted() {
		t.Fatal("a running follow did not set the interrupted flag")
	}
	// The follow fails with encumbered: a normal end, not an outage.
	st := inWorld(protocol.Vec3{20, 0, 0}, nil)
	st.Flags = []string{protocol.FlagEncumbered, protocol.FlagStranded}
	f.set(f.hello, st)
	k.Tick(ctx)
	if store.Interrupted() {
		t.Error("the flag outlived a follow that ended while stranded")
	}
}

func TestATaskThatEndsNormallyIsNoOutage(t *testing.T) {
	store := Memory()
	k, f, _ := keeper(t, store)
	ctx := context.Background()
	hello := &protocol.Hello{}
	f.set(hello, inWorld(protocol.Vec3{0, 0, 0}, &protocol.TaskRef{ID: "t-1", Task: "follow"}))
	k.SetHome(protocol.Vec3{50, 0, 0}, "Nimah")
	k.Tick(ctx)
	f.set(hello, inWorld(protocol.Vec3{0, 0, 0}, nil))
	k.Tick(ctx)
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{0, 0, 0}, nil)) // a later reconnect
	k.Tick(ctx)
	if len(f.sent()) != 0 || store.Interrupted() {
		t.Errorf("sent %v, interrupted %v", f.sent(), store.Interrupted())
	}
}

func TestIdleAwayFromHomeGoesHome(t *testing.T) {
	k, f, c := keeper(t, Memory())
	ctx := context.Background()
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{0, 0, 0}, nil))
	k.SetHome(protocol.Vec3{3, 0, 0}, "Nimah") // close by
	k.Tick(ctx)
	c.advance(11 * time.Minute)
	k.Tick(ctx)
	if len(f.sent()) != 0 {
		t.Fatalf("went home from %v away: %v", 3, f.sent())
	}

	k.SetHome(protocol.Vec3{80, 0, 0}, "Nimah")
	c.advance(9 * time.Minute)
	k.Touch() // an order resets the clock
	c.advance(9 * time.Minute)
	k.Tick(ctx)
	if len(f.sent()) != 0 {
		t.Fatalf("went home before idleMinutes: %v", f.sent())
	}
	c.advance(2 * time.Minute)
	k.Tick(ctx)
	if got := f.sent(); len(got) != 2 || got[1] != "say: Nothing to do. Heading home." {
		t.Errorf("idle: %v", got)
	}
}

// TestIdleUnderTheFloorGoesHome: seen on the VM, the bot under a raised
// floor, 1.9 m across and 3.7 m below home, is not home.
func TestIdleUnderTheFloorGoesHome(t *testing.T) {
	k, f, c := keeper(t, Memory())
	ctx := context.Background()
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{-37.76, 34.0, 6.36}, nil))
	k.SetHome(protocol.Vec3{-35.9, 37.66, 6.53}, "Nimah")
	if d, _ := k.HomeDistance(); d < 4 {
		t.Errorf("home distance %.1f; want the straight line, over 4", d)
	}
	k.Tick(ctx)
	c.advance(11 * time.Minute)
	k.Tick(ctx)
	if got := f.sent(); len(got) != 2 || got[1] != "say: Nothing to do. Heading home." {
		t.Errorf("idle under the floor: %v", got)
	}
}

// TestIdleDoesNotRetryAnUnreachableHome: once a trip home finds no way
// there, idling waits for the bot to move rather than shouting every spell.
func TestIdleDoesNotRetryAnUnreachableHome(t *testing.T) {
	for _, reason := range []string{protocol.ReasonUnreachable, protocol.ReasonDeepWater} {
		t.Run(reason, func(t *testing.T) { idleDoesNotRetry(t, reason) })
	}
}

func idleDoesNotRetry(t *testing.T, reason string) {
	k, f, c := keeper(t, Memory())
	ctx := context.Background()
	f.final, f.reason = true, reason
	hello := &protocol.Hello{}
	f.set(hello, inWorld(protocol.Vec3{-37.76, 34.0, 6.36}, nil))
	k.SetHome(protocol.Vec3{-35.9, 36.09, 6.53}, "Nimah")
	k.Tick(ctx)
	c.advance(11 * time.Minute)
	k.Tick(ctx)
	if n := len(f.sent()); n != 2 {
		t.Fatalf("first idle: %v", f.sent())
	}
	waitFor(t, func() bool { k.mu.Lock(); defer k.mu.Unlock(); return k.cutOff != nil })
	c.advance(11 * time.Minute)
	k.Tick(ctx)
	if n := len(f.sent()); n != 2 {
		t.Fatalf("idled home again from the same spot: %v", f.sent())
	}
	f.set(hello, inWorld(protocol.Vec3{-20, 33, 6}, nil))
	c.advance(11 * time.Minute)
	k.Tick(ctx)
	if n := len(f.sent()); n != 4 {
		t.Errorf("moved away, did not try again: %v", f.sent())
	}
	// That trip failed too, from over there; back at the first spot, the
	// bot has moved since, so idling tries again.
	waitFor(t, func() bool { k.mu.Lock(); defer k.mu.Unlock(); return k.cutOff != nil && k.cutOff[0] == -20 })
	f.set(hello, inWorld(protocol.Vec3{-37.76, 34.0, 6.36}, nil))
	c.advance(11 * time.Minute)
	k.Tick(ctx)
	if n := len(f.sent()); n < 6 {
		t.Errorf("back at the old spot after moving away, did not try: %v", f.sent())
	}
}

func TestStrandedBotDoesNotSetOffHome(t *testing.T) {
	k, f, c := keeper(t, Memory())
	ctx := context.Background()
	st := inWorld(protocol.Vec3{0, 0, 0}, nil)
	st.Flags, st.Weight, st.WeightMax = []string{protocol.FlagEncumbered, protocol.FlagStranded}, 305, 300
	f.set(&protocol.Hello{}, st)
	k.SetHome(protocol.Vec3{80, 0, 0}, "Nimah")
	k.Tick(ctx)
	c.advance(11 * time.Minute)
	k.Tick(ctx)
	if len(f.sent()) != 0 {
		t.Fatalf("set off home with no stamina, overloaded: %v", f.sent())
	}
	// With stamina it can still walk, slowly.
	st.Flags = []string{protocol.FlagEncumbered}
	c.advance(11 * time.Minute)
	k.Tick(ctx)
	if got := f.sent(); len(got) != 2 {
		t.Errorf("overloaded but able to walk: %v", got)
	}
}

func TestReturnOnFollowsTheRules(t *testing.T) {
	k, _, _ := keeper(t, Memory())
	if !k.ReturnOn(protocol.ReasonTargetLost) || !k.ReturnOn(protocol.ReasonStuck) || k.ReturnOn(protocol.ReasonStopped) {
		t.Error("default rules")
	}
	change(k, func(r *config.HomeRules) { r.WhenStuck = false })
	if k.ReturnOn(protocol.ReasonStuck) {
		t.Error("whenStuck off")
	}
}

// dead and back drive the keeper through a death as the state reports it.
func dead(f *fakeSession) {
	f.set(f.hello, &protocol.State{Phase: protocol.PhaseDead, World: "Midgard"})
}

func back(f *fakeSession, task *protocol.TaskRef) {
	f.set(f.hello, inWorld(protocol.Vec3{-1, 0, -5}, task))
}

func TestRespawnSendsTheBotHomeOnceItIsBack(t *testing.T) {
	k, f, _ := keeper(t, Memory())
	ctx := context.Background()
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{0, 0, 0}, nil))
	k.SetHome(protocol.Vec3{40, 0, 0}, "Nimah")
	k.Tick(ctx)

	dead(f)
	k.Tick(ctx)
	k.Tick(ctx)
	if len(f.sent()) != 0 {
		t.Fatalf("sent %v while dead", f.sent())
	}
	back(f, nil)
	k.Tick(ctx)
	if got := f.sent(); len(got) != 2 || got[0] != "move_to" || got[1] != "say: I'm back. Heading home." {
		t.Errorf("after the respawn: %v", got)
	}
	k.Tick(ctx)
	if len(f.sent()) != 2 {
		t.Errorf("went home twice: %v", f.sent())
	}

	change(k, func(r *config.HomeRules) { r.AfterDeath = config.AfterDeathStay })
	dead(f)
	k.Tick(ctx)
	back(f, nil)
	k.Tick(ctx)
	if len(f.sent()) != 2 {
		t.Errorf("afterDeath stay still went home: %v", f.sent())
	}
}

// TestNoGoingBackToWhatKilledItAtHome: on 2026-10-29 a Greyling at home
// killed the bot, which respawned, went home and was killed again, three
// times. A death near home keeps it away, and every death is a grave to
// fetch from.
func TestNoGoingBackToWhatKilledItAtHome(t *testing.T) {
	k, f, c := keeper(t, Memory())
	ctx := context.Background()
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{0, 0, 0}, nil))
	k.SetHome(protocol.Vec3{10, 0, 0}, "Nimah")
	k.Tick(ctx)

	f.set(f.hello, &protocol.State{Phase: protocol.PhaseDead, World: "Midgard", Pos: protocol.Vec3{8, 0, 2}})
	k.Tick(ctx)
	back(f, nil)
	k.Tick(ctx)
	want := []string{`say: Something killed me at home, so I'll wait here. Say "get your things" when it's safe.`}
	if got := f.sent(); !slices.Equal(got, want) {
		t.Fatalf("after dying at home: %q", got)
	}
	// Nor does idling take it back, until an order is carried out.
	c.advance(11 * time.Minute)
	k.Tick(ctx)
	if got := f.sent(); len(got) != 1 {
		t.Fatalf("idled home to what killed it: %q", got)
	}
	k.Ordered()
	k.Told()
	c.advance(11 * time.Minute)
	k.Tick(ctx)
	if got := f.sent(); len(got) != 3 || got[1] != "move_to" {
		t.Fatalf("told, still kept away: %q", got)
	}
	// With afterDeath stay it says nothing, and idling still keeps away.
	change(k, func(r *config.HomeRules) { r.AfterDeath = config.AfterDeathStay })
	f.set(f.hello, &protocol.State{Phase: protocol.PhaseDead, World: "Midgard", Pos: protocol.Vec3{12, 0, 0}})
	k.Tick(ctx)
	back(f, nil)
	k.Tick(ctx)
	c.advance(11 * time.Minute)
	k.Tick(ctx)
	if got := f.sent(); len(got) != 3 {
		t.Fatalf("with stay, went home to what killed it: %q", got)
	}
	// A later death far from home does not make home safe.
	change(k, func(r *config.HomeRules) { r.AfterDeath = config.AfterDeathHome })
	f.set(f.hello, &protocol.State{Phase: protocol.PhaseDead, World: "Midgard", Pos: protocol.Vec3{300, 0, 0}})
	k.Tick(ctx)
	back(f, nil)
	k.Tick(ctx)
	if got := f.sent(); len(got) != 4 || got[3] != `say: Something killed me at home, so I'll wait here. Say "get your things" when it's safe.` {
		t.Fatalf("after a later death far off: %q", got)
	}
	// Nor does an outage.
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{-1, 0, -5}, nil))
	k.setInterrupted(true)
	k.Tick(ctx)
	if got := f.sent(); len(got) != 4 {
		t.Fatalf("an outage sent it home to what killed it: %q", got)
	}
	k.ForgetGrave(protocol.Vec3{300, 0, 0}, 1)
	g, ok := k.Grave()
	if !ok || g.Pos != (protocol.Vec3{12, 0, 0}) {
		t.Fatalf("newest grave %+v, %v", g, ok)
	}
	k.ForgetGrave(g.Pos, 1)
	g, ok = k.Grave()
	if !ok || g.Pos != (protocol.Vec3{8, 0, 2}) || !g.At.Equal(c.now().Add(-33*time.Minute)) {
		t.Fatalf("grave %+v, %v", g, ok)
	}
	k.ForgetGrave(protocol.Vec3{9, 0, 2}, 3)
	if _, ok := k.Grave(); ok {
		t.Error("the fetched grave is still remembered")
	}
}

func TestAnOrderAfterTheDeathWins(t *testing.T) {
	k, f, _ := keeper(t, Memory())
	ctx := context.Background()
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{0, 0, 0}, nil))
	k.SetHome(protocol.Vec3{40, 0, 0}, "Nimah")
	k.Tick(ctx)

	// "wait here" while the bot is dead: over before it is back.
	dead(f)
	k.Tick(ctx)
	k.Ordered()
	back(f, nil)
	k.Tick(ctx)
	if len(f.sent()) != 0 {
		t.Errorf("went home after an order given while dead: %v", f.sent())
	}

	// The same, before Tick has seen the death.
	dead(f)
	k.Ordered()
	k.Tick(ctx)
	back(f, nil)
	k.Tick(ctx)
	if len(f.sent()) != 0 {
		t.Errorf("went home after an order given before Tick saw the death: %v", f.sent())
	}

	// "follow me" as it comes back, then "stop".
	dead(f)
	k.Tick(ctx)
	back(f, &protocol.TaskRef{ID: "t-9", Task: "follow", Status: "running"})
	k.Tick(ctx)
	back(f, nil)
	k.Tick(ctx)
	if len(f.sent()) != 0 {
		t.Errorf("went home after a newer order ended: %v", f.sent())
	}

	// "stop" on the tick it is back, before Tick acts.
	dead(f)
	k.Tick(ctx)
	back(f, nil)
	k.Ordered()
	k.Tick(ctx)
	if len(f.sent()) != 0 {
		t.Errorf("went home after an order as it came back: %v", f.sent())
	}

	// An order while alive is not one given while dead: the next death
	// still sends it home.
	k.Ordered()
	dead(f)
	k.Tick(ctx)
	back(f, nil)
	k.Tick(ctx)
	if got := f.sent(); len(got) != 2 || got[1] != "say: I'm back. Heading home." {
		t.Errorf("an order before the death cancelled the respawn: %v", got)
	}
}

func TestRespawnResumesOrFallsBackToHome(t *testing.T) {
	k, f, _ := keeper(t, Memory())
	ctx := context.Background()
	change(k, func(r *config.HomeRules) { r.AfterDeath = config.AfterDeathResume })
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{0, 0, 0}, nil))
	k.SetHome(protocol.Vec3{40, 0, 0}, "Nimah")
	k.Tick(ctx)

	resumed := 0
	die := func() {
		dead(f)
		k.Tick(ctx)
		back(f, nil)
		k.Tick(ctx)
	}
	k.ResumeAfterDeath(func(context.Context) bool { resumed++; return true })
	die()
	if resumed != 1 || len(f.sent()) != 0 {
		t.Fatalf("resumed %d, sent %v", resumed, f.sent())
	}

	// Nothing to resume, or a resume that cannot start: go home.
	die()
	k.ResumeAfterDeath(func(context.Context) bool { return false })
	die()
	if got := f.sent(); len(got) != 4 || got[1] != "say: I'm back. Heading home." || resumed != 1 {
		t.Errorf("fallbacks: %v, resumed %d", got, resumed)
	}

	// An order while dead drops the resume for good: the next death,
	// with nothing to resume, goes home.
	k.ResumeAfterDeath(func(context.Context) bool { resumed++; return true })
	dead(f)
	k.Tick(ctx)
	k.Ordered()
	back(f, nil)
	k.Tick(ctx)
	die()
	if resumed != 1 || len(f.sent()) != 6 {
		t.Errorf("a dropped resume ran: resumed %d, sent %v", resumed, f.sent())
	}
}

// change alters the keeper's rules in place, as a runtime setting does.
func change(k *Keeper, f func(*config.HomeRules)) {
	r := k.Rules()
	f(&r)
	k.rules = func() config.HomeRules { return r }
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition never held")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestTheShunSurvivesARestart: a controller restart does not forget that
// home is where something killed the bot.
func TestTheShunSurvivesARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	k, f, _ := keeper(t, store)
	ctx := context.Background()
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{0, 0, 0}, nil))
	k.SetHome(protocol.Vec3{10, 0, 0}, "Nimah")
	f.set(f.hello, &protocol.State{Phase: protocol.PhaseDead, World: "Midgard", Pos: protocol.Vec3{8, 0, 2}})
	k.Tick(ctx)

	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	k2, f2, c2 := keeper(t, again)
	f2.set(&protocol.Hello{}, inWorld(protocol.Vec3{-60, 0, 0}, nil))
	k2.Tick(ctx)
	c2.advance(11 * time.Minute)
	k2.Tick(ctx)
	if got := f2.sent(); len(got) != 0 {
		t.Fatalf("after a restart, idled home to what killed it: %q", got)
	}
	k2.Ordered()
	if !again.Shunned("Midgard") {
		t.Fatal("an attempted order ended the shun before it was carried out")
	}
	k2.Told()
	if again.Shunned("Midgard") {
		t.Error("an order did not end the shun")
	}

	// An order that comes before Tick sees the death still leaves a grave.
	f2.set(f2.hello, &protocol.State{Phase: protocol.PhaseDead, World: "Midgard", Pos: protocol.Vec3{50, 0, 50}})
	k2.Ordered()
	if g, ok := k2.Grave(); !ok || g.Pos != (protocol.Vec3{50, 0, 50}) {
		t.Errorf("grave %+v, %v", g, ok)
	}
}

func TestAShunnedHomeDoesNotPreventResumingAFollow(t *testing.T) {
	store := Memory()
	k, f, _ := keeper(t, store)
	ctx := context.Background()
	change(k, func(r *config.HomeRules) { r.AfterDeath = config.AfterDeathResume })
	f.set(&protocol.Hello{}, inWorld(protocol.Vec3{}, nil))
	k.SetHome(protocol.Vec3{}, "Nimah")
	k.Tick(ctx)
	resumed := false
	k.ResumeAfterDeath(func(context.Context) bool { resumed = true; return true })
	dead(f)
	k.Tick(ctx)
	back(f, nil)
	k.Tick(ctx)
	if !resumed || !store.Shunned("Midgard") || len(f.sent()) != 0 {
		t.Fatalf("resumed %v, shunned %v, sent %v", resumed, store.Shunned("Midgard"), f.sent())
	}
}

func TestACarriedOutOrderUnshunsOnlyItsWorld(t *testing.T) {
	store := Memory()
	k, f, _ := keeper(t, store)
	for _, world := range []string{"Midgard", "Other"} {
		f.set(&protocol.Hello{}, &protocol.State{Phase: protocol.PhaseInWorld, World: world})
		k.SetHome(protocol.Vec3{}, "Nimah")
		if err := store.Shun(world); err != nil {
			t.Fatal(err)
		}
	}
	k.Told()
	if !store.Shunned("Midgard") || store.Shunned("Other") {
		t.Fatalf("shunned Midgard=%v Other=%v", store.Shunned("Midgard"), store.Shunned("Other"))
	}
}
