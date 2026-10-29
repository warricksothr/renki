package food

import (
	"context"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

type fake struct {
	st                 *protocol.State
	hello              *protocol.Hello
	connected, allowed bool
	orders             []session.Order
	status             string
}

func (f *fake) State() *protocol.State { return f.st }
func (f *fake) Handshake() (*protocol.Hello, *protocol.Welcome, bool) {
	return f.hello, nil, f.connected
}
func (f *fake) Allowed(string) bool { return f.allowed }
func (f *fake) Dispatch(_ context.Context, o session.Order) (*session.Task, error) {
	f.orders = append(f.orders, o)
	return session.Finished(o.Task, nil, f.status, ""), nil
}
func scene() (*fake, *Keeper, *time.Time) {
	now := time.Unix(1000, 0)
	f := &fake{hello: &protocol.Hello{}, connected: true, allowed: true, status: protocol.StatusSucceeded,
		st: &protocol.State{Envelope: protocol.Envelope{Seq: 1}, Phase: protocol.PhaseInWorld, World: "world", HP: 25, HPMax: 25,
			Food: &protocol.FoodStatus{Carried: []protocol.CarriedFood{{Prefab: "Meat", Health: 35, Regen: 2, CanEat: true}, {Prefab: "Berry", Health: 10, Regen: 1, CanEat: true}}}}}
	k := New(f, func() []string { return []string{"Nimah"} }, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
	return f, k, &now
}
func fresh(f *fake) { copy := *f.st; copy.Seq++; f.st = &copy }
func TestFillEmptySlotsBeforeRefreshingAndIgnoreIneligible(t *testing.T) {
	s := &protocol.FoodStatus{Active: []protocol.ActiveFood{{Prefab: "Meat"}}, Carried: []protocol.CarriedFood{
		{Prefab: "Meat", Health: 35, CanEat: true}, {Prefab: "Berry", Health: 10, CanEat: true}, {Prefab: "Locked", Health: 99}}}
	if got := Choose(s); got != "Berry" {
		t.Fatal(got)
	}
	s.Active = append(s.Active, protocol.ActiveFood{Prefab: "Two"}, protocol.ActiveFood{Prefab: "Three"})
	if got := Choose(s); got != "Meat" {
		t.Fatal(got)
	}
}
func TestOneActionWaitsForNewObservationAndPreservesBody(t *testing.T) {
	f, k, now := scene()
	f.st.Task = &protocol.TaskRef{ID: "walk", Task: protocol.TaskMoveTo, Status: protocol.StatusRunning}
	k.Tick(context.Background())
	k.Tick(context.Background())
	*now = now.Add(time.Second)
	k.Tick(context.Background())
	if len(f.orders) != 1 || f.orders[0].Task != protocol.TaskUseItem || f.st.Task.ID != "walk" {
		t.Fatal(f.orders)
	}
	fresh(f)
	f.st.Food = &protocol.FoodStatus{Active: []protocol.ActiveFood{{Prefab: "Meat"}}, Carried: []protocol.CarriedFood{{Prefab: "Berry", Health: 10, CanEat: true}}}
	k.Tick(context.Background())
	*now = now.Add(3 * time.Second)
	fresh(f)
	k.Tick(context.Background())
	if len(f.orders) != 2 || f.orders[1].Args.(*protocol.UseItemArgs).Item != "Berry" {
		t.Fatal(f.orders)
	}
}
func TestPermissionDisconnectMenuAndPriorityActions(t *testing.T) {
	for _, mode := range []string{"permission", "disconnect", "menu", "dead", "attack", "sleep", "flee", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			f, k, _ := scene()
			switch mode {
			case "permission":
				f.allowed = false
			case "disconnect":
				f.connected = false
			case "menu":
				f.st.Phase = protocol.PhaseMenu
			case "dead":
				f.st.HP = 0
			case "attack":
				f.st.Flags = []string{protocol.FlagAttacking}
			case "sleep":
				f.st.Flags = []string{protocol.FlagSleeping}
			case "flee":
				f.st.Task = &protocol.TaskRef{Task: protocol.TaskFlee}
			case "unknown":
				f.st.Food = nil
			}
			k.Tick(context.Background())
			if len(f.orders) != 0 {
				t.Fatal(f.orders)
			}
		})
	}
}
func TestFailedFoodBacksOffAndStaleSnapshotDoesNotRetry(t *testing.T) {
	f, k, now := scene()
	f.status = protocol.StatusFailed
	k.Tick(context.Background())
	fresh(f)
	k.Tick(context.Background())
	*now = now.Add(time.Minute)
	k.Tick(context.Background()) // First observes this unchanged pointer as stale.
	if len(f.orders) != 1 {
		t.Fatal(f.orders)
	}
	fresh(f)
	k.Tick(context.Background())
	if len(f.orders) != 2 {
		t.Fatal(f.orders)
	}
}
func TestAskOwnerOnlyWhenHungryInjuredAndRateLimited(t *testing.T) {
	f, k, now := scene()
	f.st.Food = &protocol.FoodStatus{}
	f.st.HP = 10
	k.Tick(context.Background())
	k.Tick(context.Background())
	*now = now.Add(time.Minute)
	fresh(f)
	k.Tick(context.Background())
	if len(f.orders) != 1 || f.orders[0].Task != protocol.TaskSay || !strings.Contains(f.orders[0].Args.(*protocol.SayArgs).Text, "Nimah") {
		t.Fatal(f.orders)
	}
	*now = now.Add(Notice)
	fresh(f)
	k.Tick(context.Background())
	if len(f.orders) != 2 {
		t.Fatal(f.orders)
	}
	// Fresh active food is already helping recovery; don't ask needlessly.
	fresh(f)
	f.st.Food = &protocol.FoodStatus{Active: []protocol.ActiveFood{{Prefab: "Meat"}}}
	*now = now.Add(Notice)
	k.Tick(context.Background())
	if len(f.orders) != 2 {
		t.Fatal(f.orders)
	}
}
func TestNewConnectionDoesNotReuseOldPendingAction(t *testing.T) {
	f, k, _ := scene()
	k.Tick(context.Background())
	f.hello = &protocol.Hello{}
	fresh(f)
	f.st.World = "other"
	f.st.Food = &protocol.FoodStatus{Carried: []protocol.CarriedFood{{Prefab: "OtherFood", Health: 10, CanEat: true}}}
	k.Tick(context.Background())
	if len(f.orders) != 2 || f.orders[1].Args.(*protocol.UseItemArgs).Item != "OtherFood" {
		t.Fatal(f.orders)
	}
}
