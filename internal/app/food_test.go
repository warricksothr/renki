package app

import (
	"context"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func TestEscortSupplyBoundaryDoesNotSurviveANewerOwnerOrder(t *testing.T) {
	a, p, _, _ := goalFixture(t)
	oldHello, _, _ := a.Session.Handshake()
	q := escortSupplyRequest{args: protocol.EscortArgs{Player: "Nimah", Resupply: true}, hello: oldHello, world: "one", revision: a.Session.ControlRevision()}
	if _, err := a.Session.Dispatch(context.Background(), session.Order{Task: protocol.TaskStop}); err != nil {
		t.Fatal(err)
	}
	p.NextTask()
	if _, err := a.startFoodFor(context.Background(), "Nimah", &q.args, &q); err == nil {
		t.Fatal("queued old escort after stop")
	}
	if len(a.Goals.List()) != 0 {
		t.Fatal("stale escort created food work")
	}
}
func TestFoodGoalAPIKeepsFullCheckpointSeparateFromStatusSummary(t *testing.T) {
	a, _, call, _ := goalFixture(t)
	if _, err := a.Home.SetHome(protocol.Vec3{}, "test"); err != nil {
		t.Fatal(err)
	}
	response := call("POST", `{"action":"start","spec":{"kind":"food_resupply","world":"one","center":[0,0,0],"radius":150,"reserve":2}}`, true, true)
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	g := createdGoal(t, response)
	if g.Food == nil || g.Food.Reserve != 2 || g.Kind != "food_resupply" {
		t.Fatal(g)
	}
	if _, err := a.startFood(context.Background(), "Visitor", nil); err == nil {
		t.Fatal("visitor admitted")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := a.goalAction(ctx, g.ID, "stop"); err != nil {
		t.Fatal(err)
	}
}
