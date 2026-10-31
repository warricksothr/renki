package goal

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/protocol"
)

func returnFixture(t *testing.T) (*Template, *runner, behaviour.View, string) {
	t.Helper()
	h, r, v, id := fixture(t)
	if err := h.Store.update(id, func(g *Record) error { g.ReturnHome = true; g.MaxTasks = 7; return nil }); err != nil {
		t.Fatal(err)
	}
	home := protocol.Vec3{5, 2, 0}
	v.Home = &home
	return h, r, v, id
}

func killForReturn(t *testing.T, r *runner, v *behaviour.View) {
	t.Helper()
	attack(t, r, *v, "fight")
	v.Perception.Characters = nil
	_, err := r.Next(*v, &behaviour.Result{TaskID: "fight", Status: "succeeded", Detail: "observed Greyling death"})
	var delay *behaviour.Delay
	if !errors.As(err, &delay) {
		t.Fatalf("expected fresh post-kill wait: %v", err)
	}
	fresh(v, 2)
}

func TestCombatRoundTripRequiresSeparateMatchingReturnAndFreshArrival(t *testing.T) {
	h, r, v, id := returnFixture(t)
	killForReturn(t, r, &v)
	g, _ := h.Store.Get(id)
	if g.Status != "running" || g.Proof.TaskID != "fight" || g.Home == nil || g.PostCombat.Phase != "recover" {
		t.Fatal(g)
	}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskMoveTo || *o.Args.(*protocol.MoveToArgs).To.Pos != *v.Home {
		t.Fatal(o, err)
	}
	if err = r.Dispatched("return"); err != nil {
		t.Fatal(err)
	}
	fresh(&v, 10)
	v.State.Pos = *v.Home
	_, err = r.Next(v, &behaviour.Result{TaskID: "return", Status: "succeeded"})
	var delay *behaviour.Delay
	if !errors.As(err, &delay) {
		t.Fatalf("arrival must use post-receipt state: %v", err)
	}
	g, _ = h.Store.Get(id)
	if g.Status != "running" || g.PostCombat.Receipt.TaskID != "return" || g.Pending != nil {
		t.Fatal(g)
	}
	fresh(&v, 2)
	o, err = r.Next(v, nil)
	if err != nil || o != nil {
		t.Fatal(o, err)
	}
	g, _ = h.Store.Get(id)
	if g.Status != "succeeded" || g.Proof.TaskID != "fight" || g.PostCombat.Phase != "complete" || g.Attempts != 1 || g.TaskCount != 2 {
		t.Fatal(g)
	}
	s, err := Open(h.Store.path)
	if err != nil {
		t.Fatal(err)
	}
	g, _ = s.Get(id)
	if g.Proof.TaskID != "fight" || g.PostCombat.Receipt.TaskID != "return" || g.PostCombat.Phase != "complete" {
		t.Fatal(g)
	}
}

func TestReturnFailuresAndFalseArrivalKeepKillWithoutAnotherAttack(t *testing.T) {
	for _, cause := range []string{"failure", "wrong_receipt", "far", "wrong_floor", "changed_home", "nan"} {
		t.Run(cause, func(t *testing.T) {
			h, r, v, id := returnFixture(t)
			killForReturn(t, r, &v)
			if _, err := r.Next(v, nil); err != nil {
				t.Fatal(err)
			}
			if err := r.Dispatched("return"); err != nil {
				t.Fatal(err)
			}
			result := &behaviour.Result{TaskID: "return", Status: "succeeded"}
			if cause == "failure" {
				result.Status = "failed"
				result.Reason = "unreachable"
			}
			if cause == "wrong_receipt" {
				result.TaskID = "other"
			}
			fresh(&v, 10)
			v.State.Pos = *v.Home
			_, _ = r.Next(v, result)
			fresh(&v, 2)
			switch cause {
			case "far":
				v.State.Pos[0] += 4
			case "wrong_floor":
				v.State.Pos[1] -= 3
			case "changed_home":
				copy := *v.Home
				copy[0]++
				v.Home = &copy
			case "nan":
				v.State.Pos[0] = math.NaN()
			}
			o, err := r.Next(v, nil)
			if err != nil || o != nil {
				t.Fatal(o, err)
			}
			g, _ := h.Store.Get(id)
			if g.Status != "needs_plan" || g.Proof.TaskID != "fight" || g.Attempts != 1 || g.TaskCount != 2 {
				t.Fatal(g)
			}
		})
	}
}

func TestPostCombatRecoveryRetainsBoundsAndRechecksReadiness(t *testing.T) {
	for _, cause := range []string{"stamina", "health", "threat", "stale", "food", "encumbered", "water", "deadline", "budget", "scope"} {
		t.Run(cause, func(t *testing.T) {
			h, r, v, id := returnFixture(t)
			killForReturn(t, r, &v)
			switch cause {
			case "stamina":
				v.State.Stamina = 20
			case "health":
				v.State.HP = 50
			case "threat":
				v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "other", Kind: "creature", Hostile: true, HPFrac: 1, Pos: protocol.Vec3{4, 0, 0}}}
			case "stale":
				v.PerceptionAt = v.Now.Add(-10 * time.Second)
			case "food":
				v.State.Food = nil
			case "encumbered":
				v.State.Flags = []string{protocol.FlagEncumbered}
			case "water":
				v.State.Flags = []string{protocol.FlagSwimming}
			case "deadline":
				fresh(&v, 181)
			case "budget":
				_ = h.Store.update(id, func(g *Record) error { g.WaitS = 120; return nil })
				v.State.Stamina = 20
			case "scope":
				v.State.Pos[0] = 101
			}
			o, err := r.Next(v, nil)
			var delay *behaviour.Delay
			waiting := cause == "stamina" || cause == "health" || cause == "stale"
			if o != nil || (waiting && !errors.As(err, &delay)) || (!waiting && err != nil) {
				t.Fatal(o, err)
			}
			g, _ := h.Store.Get(id)
			if g.Proof == nil || g.TaskCount != 1 || g.Attempts != 1 {
				t.Fatal(g)
			}
			if waiting {
				fresh(&v, 61)
				if cause == "stale" {
					v.PerceptionAt = v.Now.Add(-10 * time.Second)
					_ = h.Store.update(id, func(g *Record) error { g.WaitS = 120; return nil })
				}
				_, _ = r.Next(v, nil)
				g, _ = h.Store.Get(id)
			}
			if g.Status != "needs_plan" {
				t.Fatal(g)
			}
		})
	}
}

func TestReturnIntentIsDurableCannotReplayAndPreservesKillOnCancellation(t *testing.T) {
	for _, cause := range []string{"restart", "cancel", "disk"} {
		t.Run(cause, func(t *testing.T) {
			h, r, v, id := returnFixture(t)
			killForReturn(t, r, &v)
			if cause == "disk" {
				h.Store.write = func([]byte) error { return errors.New("disk full") }
				o, err := r.Next(v, nil)
				if err == nil || o != nil {
					t.Fatal(o, err)
				}
				g, _ := h.Store.Get(id)
				if g.Pending != nil || g.PostCombat.Phase != "recover" || g.Proof == nil {
					t.Fatal(g)
				}
				return
			}
			if _, err := r.Next(v, nil); err != nil {
				t.Fatal(err)
			}
			if err := r.Dispatched("return"); err != nil {
				t.Fatal(err)
			}
			if cause == "cancel" {
				if err := h.Cancel(id); err != nil {
					t.Fatal(err)
				}
				g, _ := h.Store.Get(id)
				if g.Status != "cancelled" || g.Proof == nil {
					t.Fatal(g)
				}
			} else {
				s, err := Open(h.Store.path)
				if err != nil {
					t.Fatal(err)
				}
				g, _ := s.Get(id)
				if g.Status != "needs_plan" || g.Proof == nil || g.Pending.TaskID != "return" {
					t.Fatal(g)
				}
				if err := (&Template{Store: s}).Queue(id); err == nil || !strings.Contains(err.Error(), "uncertain") {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestLastCombatAttemptStillHasExactlyOneReservedReturnTask(t *testing.T) {
	h, r, v, id := returnFixture(t)
	killForReturn(t, r, &v)
	if err := h.Store.update(id, func(g *Record) error { g.Attempts = 6; g.TaskCount = 6; return nil }); err != nil {
		t.Fatal(err)
	}
	o, err := r.Next(v, nil)
	g, _ := h.Store.Get(id)
	if err != nil || o == nil || o.Task != "move_to" || g.Attempts != 6 || g.TaskCount != 7 {
		t.Fatal(o, err, g)
	}
	if err := r.Dispatched("return"); err != nil {
		t.Fatal(err)
	}
	h.Store.write = func([]byte) error { return errors.New("disk full") }
	_, err = r.Next(v, &behaviour.Result{TaskID: "return", Status: "succeeded"})
	g, _ = h.Store.Get(id)
	if err == nil || g.Pending == nil || g.PostCombat.Receipt != nil || g.Proof == nil {
		t.Fatal(err, g)
	}
}

func TestReturnHomeMustBePinnedInsideScopeBeforeCombat(t *testing.T) {
	for _, cause := range []string{"missing", "outside", "changed"} {
		t.Run(cause, func(t *testing.T) {
			h, r, v, id := returnFixture(t)
			if cause == "changed" {
				attack(t, r, v, "fight")
				_, _ = r.Next(v, &behaviour.Result{TaskID: "fight", Status: "failed", Reason: "unreachable"})
				fresh(&v, 5)
				v.Home[0]++
			}
			if cause == "missing" {
				v.Home = nil
			}
			if cause == "outside" {
				v.Home[0] = 101
			}
			o, err := r.Next(v, nil)
			g, _ := h.Store.Get(id)
			if o != nil || err != nil || g.Status != "needs_plan" {
				t.Fatal(o, err, g)
			}
		})
	}
	g, err := NewRecord(Spec{Kind: "combat_trial", World: "one", Radius: 100, ReturnHome: true}, time.Now())
	if err != nil || g.MaxAttempts != 6 || g.MaxTasks != 7 {
		t.Fatal(g, err)
	}
	raw, _ := json.Marshal(g)
	var copy Record
	_ = json.Unmarshal(raw, &copy)
	if !copy.ReturnHome {
		t.Fatal(copy)
	}
}
