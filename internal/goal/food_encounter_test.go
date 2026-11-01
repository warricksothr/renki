package goal

import (
	"encoding/json"
	"testing"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func TestHouseholdTravelRecoveryRequiresMatchingCurrentNativeTask(t *testing.T) {
	for _, change := range []string{"matching", "other_task", "stationary", "finished", "old_run", "opt_out", "injury"} {
		t.Run(change, func(t *testing.T) {
			host, r, v, id := stockFixture(t)
			g, _ := host.Store.Get(id)
			pos := protocol.Vec3{30, 0, 0}
			o, err := r.foodDispatch(g, v, "search_food", "", "marker:patch", &session.Order{Task: protocol.TaskMoveTo,
				Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}, Dry: true, AvoidHostiles: true}, TimeoutS: 180})
			if err != nil || o == nil || !o.Args.(*protocol.MoveToArgs).RecoverThreats {
				t.Fatal(o, err)
			}
			if err = r.Dispatched("journey"); err != nil {
				t.Fatal(err)
			}
			v.State.Task = &protocol.TaskRef{ID: "journey", Task: protocol.TaskMoveTo, Status: protocol.StatusRunning}
			v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "neck", Prefab: "Neck", Hostile: true, TargetingMe: true, HPFrac: 1, Pos: protocol.Vec3{10, 0, 0}}}
			switch change {
			case "other_task":
				v.State.Task.ID = "owner-order"
			case "stationary":
				v.State.Task.Task = protocol.TaskInteract
			case "finished":
				v.State.Task.Status = protocol.StatusSucceeded
			}
			if err = host.Store.update(id, func(g *Record) error {
				n := uint64(0)
				g.Food.Injury = &n
				if change == "old_run" {
					g.Pending.Run = "prior-controller"
				}
				if change == "opt_out" {
					a := *o.Args.(*protocol.MoveToArgs)
					a.RecoverThreats = false
					g.Pending.Params, _ = json.Marshal(a)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if change == "injury" {
				n := uint64(1)
				v.InjurySeq = &n
			}
			err = r.Check(v)
			if (err == nil) != (change == "matching") {
				t.Fatal(change, err)
			}
			g, _ = host.Store.Get(id)
			if g.Status != "running" || g.Pending.TaskID != "journey" || len(g.Food.Receipts) != 0 {
				t.Fatal("guard mutated the task or invented production", g)
			}
		})
	}
}

func TestDistantTargetingDoesNotTerminateHouseholdWork(t *testing.T) {
	host, r, v, id := stockFixture(t)
	v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "neck", Prefab: "Neck", Hostile: true, TargetingMe: true, HPFrac: 1, Pos: protocol.Vec3{22, 0, 0}}}
	if err := r.Check(v); err != nil {
		t.Fatal(err)
	}
	o, err := r.Next(v, nil)
	g, _ := host.Store.Get(id)
	if err != nil || o == nil || o.Task != protocol.TaskWithdraw || g.Status != "running" || len(g.Food.Receipts) != 0 {
		t.Fatal("distant awareness cancelled work or became a receipt", o, g, err)
	}
}

func TestUnfedBootstrapDoesNotOptIntoFedTravelRecovery(t *testing.T) {
	host, r, v, id := stockFixture(t)
	v.State.Food = nil
	g, _ := host.Store.Get(id)
	pos := protocol.Vec3{5, 0, 0}
	o, err := r.foodDispatch(g, v, "home", "", "", &session.Order{Task: protocol.TaskMoveTo,
		Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}, Dry: true, AvoidHostiles: true}, TimeoutS: 60})
	if err != nil || o == nil || o.Args.(*protocol.MoveToArgs).RecoverThreats {
		t.Fatal("unfed bootstrap acquired a food-dependent encounter policy", o, err)
	}
}
