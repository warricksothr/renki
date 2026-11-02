package goal

import (
	"errors"
	"testing"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/protocol"
)

func TestCookingReturnWaitsForStateAfterItsNativeCompletionReceipt(t *testing.T) {
	host, r, v, id := stockFixture(t)
	home := protocol.Vec3{-35.9, 37.66, 6.53}
	v.Home = &home
	v.State.Pos = protocol.Vec3{-33.1, 36.15, 9.75}
	v.State.Seq = 2330
	for i := range v.Perception.Structures {
		v.Perception.Structures[i].Pos = protocol.Vec3{home[0] + float64(i+1), 36.09, home[2]}
	}
	if err := host.Store.update(id, func(g *Record) error {
		g.Center, g.Home = home, &home
		g.Food.Stock.ReturnAfterCooking = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskMoveTo {
		t.Fatal(o, err)
	}
	if err = r.Dispatched("cook-return"); err != nil {
		t.Fatal(err)
	}
	foodFresh(&v)
	// Captured state2341 precedes native success2342. Tas is still81cm
	// from home here, although movement succeeds before the next state.
	v.State.Seq, v.State.Pos = 2341, protocol.Vec3{-35.29, 36.09, 7.07}
	last := &behaviour.Result{TaskID: "cook-return", Status: protocol.StatusSucceeded, ReceiptSeq: 2342}
	o, err = r.Next(v, last)
	var delay *behaviour.Delay
	if o != nil || !errors.As(err, &delay) {
		t.Fatal("pre-arrival state caused failure or another action", o, err)
	}
	g, _ := host.Store.Get(id)
	if g.Status == "needs_plan" || g.Pending == nil || g.Pending.TaskID != "cook-return" {
		t.Fatal(g)
	}
	foodFresh(&v)
	v.State.Seq, v.State.Pos = 2344, protocol.Vec3{-35.63, 36.09, 6.77}
	_, err = r.Next(v, last)
	if err != nil && !errors.As(err, &delay) {
		t.Fatal(err)
	}
	g, _ = host.Store.Get(id)
	if g.Status == "needs_plan" || g.Food.Stock.ReturnAfterCooking || len(g.History) != 1 || g.History[0].Template != "cook_return" || g.History[0].Status != protocol.StatusSucceeded {
		t.Fatal(g)
	}
}
