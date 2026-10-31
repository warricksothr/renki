package goal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/protocol"
)

func fixture(t *testing.T) (*Template, *runner, behaviour.View, string) {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "goals.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	g, err := NewRecord(Spec{Kind: "combat_trial", World: "one", Radius: 100}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Add(g); err != nil {
		t.Fatal(err)
	}
	host := &Template{Store: s}
	if err = host.Queue(g.ID); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(params{g.ID})
	rr, err := host.Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	v := behaviour.View{Now: now, StateAt: now, PerceptionAt: now, State: &protocol.State{Envelope: protocol.Envelope{T: 20}, World: "one", Phase: protocol.PhaseInWorld, HP: 80, HPMax: 80, Stamina: 60, Equipped: map[string]string{"right": "AxeFlint"}, Food: &protocol.FoodStatus{Active: []protocol.ActiveFood{{Prefab: "Raspberry"}}}}, Inventory: &protocol.Inventory{}, Perception: &protocol.Perception{Envelope: protocol.Envelope{T: 20}, Characters: []protocol.PerceivedCharacter{{ID: "grey", Kind: protocol.CharacterCreature, Prefab: "Greyling", Hostile: true, HPFrac: 1, Pos: protocol.Vec3{10, 0, 0}}}}}
	return host, rr.(*runner), v, g.ID
}
func fresh(v *behaviour.View, seconds float64) {
	v.Now = v.Now.Add(time.Duration(seconds * float64(time.Second)))
	v.StateAt = v.Now
	v.PerceptionAt = v.Now
	v.State.T += seconds
	v.Perception.T = v.State.T
}
func attack(t *testing.T, r *runner, v behaviour.View, id string) {
	t.Helper()
	o, e := r.Next(v, nil)
	if e != nil || o == nil || o.Task != "attack" {
		t.Fatalf("attack: %+v %v", o, e)
	}
	if e = r.Dispatched(id); e != nil {
		t.Fatal(e)
	}
}
func TestRecoveryWaitsForChangedObservationThenCompletesWithExactReceipt(t *testing.T) {
	host, r, v, id := fixture(t)
	attack(t, r, v, "task-1")
	g, _ := host.Store.Get(id)
	if g.Pending.TaskID != "task-1" || g.Attempts != 1 {
		t.Fatal(g)
	}
	o, err := r.Next(v, &behaviour.Result{Status: "failed", Reason: "unreachable", Detail: "travel limit", TaskID: "task-1"})
	var wait *behaviour.Delay
	if o != nil || !errors.As(err, &wait) {
		t.Fatalf("%+v %v", o, err)
	}
	g, _ = host.Store.Get(id)
	if g.Pending != nil || g.Last.TaskID != "task-1" || g.Attempts != 1 || len(g.History) != 1 {
		t.Fatal(g)
	}
	fresh(&v, 5)
	v.Perception.Characters[0].Pos[0] = 6
	attack(t, r, v, "task-2")
	o, err = r.Next(v, &behaviour.Result{Status: "succeeded", Detail: "observed Greyling death", TaskID: "task-2"})
	if err != nil || o != nil {
		t.Fatal(o, err)
	}
	g, _ = host.Store.Get(id)
	if g.Status != "succeeded" || g.Proof.TaskID != "task-2" || g.Proof.Entity != "grey" || g.Attempts != 2 {
		t.Fatal(g)
	}
	reopened, e := Open(host.Store.path)
	if e != nil {
		t.Fatal(e)
	}
	g, _ = reopened.Get(id)
	if g.Status != "succeeded" || g.Attempts != 2 {
		t.Fatal(g)
	}
}
func TestRestartNeverReplaysUncertainIntent(t *testing.T) {
	host, r, v, id := fixture(t)
	attack(t, r, v, "task-1")
	reopened, e := Open(host.Store.path)
	if e != nil {
		t.Fatal(e)
	}
	g, _ := reopened.Get(id)
	if g.Status != "needs_plan" || g.Pending.TaskID != "task-1" || g.TaskCount != 1 || !strings.Contains(g.Reason, "uncertain") {
		t.Fatal(g)
	}
	if e = (&Template{Store: reopened}).Queue(id); e == nil {
		t.Fatal("uncertain action resumed")
	}
}
func TestReceiptCheckpointFailureStopsBeforeAnotherAction(t *testing.T) {
	host, r, v, id := fixture(t)
	attack(t, r, v, "task-1")
	host.Store.write = func([]byte) error { return errors.New("disk full") }
	o, e := r.Next(v, &behaviour.Result{Status: "failed", Reason: "unreachable", TaskID: "task-1"})
	if e == nil || o != nil {
		t.Fatal(o, e)
	}
	g, _ := host.Store.Get(id)
	if g.Pending == nil || g.Attempts != 1 {
		t.Fatal(g)
	}
}
func TestIntentCheckpointFailurePreventsDispatch(t *testing.T) {
	host, r, v, id := fixture(t)
	host.Store.write = func([]byte) error { return errors.New("disk full") }
	o, e := r.Next(v, nil)
	if e == nil || o != nil {
		t.Fatal(o, e)
	}
	g, _ := host.Store.Get(id)
	if g.Attempts != 0 || g.Pending != nil {
		t.Fatal(g)
	}
}
func TestSuccessClaimsAndReceiptMismatchCannotProveKill(t *testing.T) {
	for _, test := range []struct{ name, status, detail, receipt string }{{"unloaded", "succeeded", "target disappeared", "task-1"}, {"wrong receipt", "succeeded", "observed Greyling death", "other"}} {
		t.Run(test.name, func(t *testing.T) {
			host, r, v, id := fixture(t)
			attack(t, r, v, "task-1")
			o, e := r.Next(v, &behaviour.Result{Status: test.status, Detail: test.detail, TaskID: test.receipt})
			if e != nil || o != nil {
				t.Fatal(o, e)
			}
			g, _ := host.Store.Get(id)
			if g.Status != "needs_plan" || g.Proof != nil {
				t.Fatal(g)
			}
		})
	}
	host, r, v, id := fixture(t)
	source := "def step(context, state, outcome):\n    return {\"action\": \"done\"}\n"
	_ = host.Store.update(id, func(g *Record) error { g.Policy = source; g.PolicyHash = hash(source); return nil })
	_, _ = r.Next(v, nil)
	g, _ := host.Store.Get(id)
	if g.Status != "needs_plan" || g.Proof != nil {
		t.Fatal(g)
	}
}
func TestWorldFreshnessScopeReadinessAndEligibility(t *testing.T) {
	tests := map[string]func(*behaviour.View){
		"world":            func(v *behaviour.View) { v.State.World = "other" },
		"stale state":      func(v *behaviour.View) { v.StateAt = v.Now.Add(-4 * time.Second) },
		"scope":            func(v *behaviour.View) { v.State.Pos[0] = 49 },
		"no food":          func(v *behaviour.View) { v.State.Food = nil },
		"no tool":          func(v *behaviour.View) { v.State.Equipped = nil },
		"stale perception": func(v *behaviour.View) { v.PerceptionAt = v.Now.Add(-4 * time.Second) },
		"starred":          func(v *behaviour.View) { v.Perception.Characters[0].Stars = 1 },
		"boar":             func(v *behaviour.View) { v.Perception.Characters[0].Prefab = "Boar" },
		"distant":          func(v *behaviour.View) { v.Perception.Characters[0].Pos[0] = 31 },
		"low health":       func(v *behaviour.View) { v.State.HP = 30 },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			_, r, v, _ := fixture(t)
			mutate(&v)
			o, _ := r.Next(v, nil)
			if o != nil {
				t.Fatal("unsafe action", o)
			}
		})
	}
}
func TestCloseConfirmedAttackerSelectsNativeDirectAlternative(t *testing.T) {
	_, r, v, _ := fixture(t)
	v.Perception.Characters[0].Pos[0] = 3
	v.Perception.Characters[0].TargetingMe = true
	o, e := r.Next(v, nil)
	if e != nil || o == nil {
		t.Fatal(o, e)
	}
	if o.Args.(*protocol.AttackArgs).Kite {
		t.Fatal("did not use direct alternative")
	}
}
func TestBudgetsAndCancellationSurviveResume(t *testing.T) {
	host, r, v, id := fixture(t)
	attack(t, r, v, "task-1")
	_, _ = r.Next(v, &behaviour.Result{Status: "failed", Reason: "unreachable", TaskID: "task-1"})
	raw, _ := json.Marshal(params{id})
	if e := host.End(raw, "new order"); e != nil {
		t.Fatal(e)
	}
	if e := host.Queue(id); e == nil {
		t.Fatal("owner cancellation resumed")
	}
	// A suspended objective may resume, without resetting checkpoint/budgets.
	_ = host.Store.update(id, func(g *Record) error { g.Status = "suspended"; return nil })
	if e := host.Queue(id); e != nil {
		t.Fatal(e)
	}
	g, _ := host.Store.Get(id)
	if g.Attempts != 1 || g.WaitS != 5 {
		t.Fatal(g)
	}
	_ = host.Store.update(id, func(g *Record) error { g.Status = "suspended"; g.Attempts = g.MaxAttempts; return nil })
	if e := host.Queue(id); e == nil {
		t.Fatal("resume reset budget")
	}
}
func TestWaitBudgetEndsUnchangedObservationLoop(t *testing.T) {
	host, r, v, id := fixture(t)
	v.Perception.Characters = nil
	attack(t, r, v, "search-1")
	// A fresh unsupported threat prevents blind discovery, even though the
	// previous search consumed a sector. Advancing timestamps cannot clear it.
	v.Perception.Characters = []protocol.PerceivedCharacter{{Kind: protocol.CharacterCreature, Prefab: "Neck", Hostile: true, HPFrac: 1}}
	_, _ = r.Next(v, &behaviour.Result{Status: "failed", Reason: "invalid_target", TaskID: "search-1"})
	fresh(&v, 5)
	for i := 1; i < 25; i++ {
		o, e := r.Next(v, nil)
		if o != nil {
			t.Fatal(o)
		}
		if i < 24 {
			var wait *behaviour.Delay
			if !errors.As(e, &wait) {
				t.Fatal(e)
			}
		}
		fresh(&v, 5)
	}
	g, _ := host.Store.Get(id)
	if g.Status != "needs_plan" || g.WaitS != 120 || g.TaskCount != 1 {
		t.Fatal(g)
	}
}

func TestDiscoveryBindsOnlyTheNativeFinalReceipt(t *testing.T) {
	for _, entity := range []string{"", "native-grey"} {
		t.Run("receipt="+entity, func(t *testing.T) {
			host, r, v, id := fixture(t)
			v.Perception.Characters = nil
			order, err := r.Next(v, nil)
			if err != nil || order == nil {
				t.Fatal(order, err)
			}
			args := order.Args.(*protocol.AttackArgs)
			if !args.Kite || args.Entity != "" || args.At != nil {
				t.Fatal("discovery must use native acquisition", args)
			}
			if err := r.Dispatched("search-1"); err != nil {
				t.Fatal(err)
			}
			_, err = r.Next(v, &behaviour.Result{TaskID: "search-1", Status: "succeeded", Detail: "observed Greyling death", Entity: entity})
			if err != nil {
				t.Fatal(err)
			}
			g, _ := host.Store.Get(id)
			if entity == "" {
				if g.Status != "needs_plan" || g.Proof != nil {
					t.Fatal("unbound discovery claimed success", g)
				}
			} else if g.Status != "succeeded" || g.Proof == nil || g.Proof.Entity != entity {
				t.Fatal(g)
			}
		})
	}
}

func TestDiscoveryRadiusLeavesRoomForLocalCombatAndEscape(t *testing.T) {
	for _, test := range []struct{ radius, offset, want float64 }{{200, 0, 24}, {100, 0, 12}, {200, 120, 24}, {200, 149, 0}} {
		t.Run(fmt.Sprintf("radius=%g/offset=%g", test.radius, test.offset), func(t *testing.T) {
			host, r, v, id := fixture(t)
			if err := host.Store.update(id, func(g *Record) error { g.Radius = test.radius; return nil }); err != nil {
				t.Fatal(err)
			}
			v.State.Pos[0] = test.offset
			v.Perception.Characters = nil
			order, _ := r.Next(v, nil)
			if test.want == 0 {
				if order != nil {
					t.Fatal("search escaped remaining scope", order)
				}
				return
			}
			if order == nil {
				t.Fatal("no discovery")
			}
			args := order.Args.(*protocol.AttackArgs)
			if args.SearchRadius == nil || args.SearchCenter == nil || *args.SearchRadius != test.want || distance(*args.SearchCenter, protocol.Vec3{})+*args.SearchRadius+52 > test.radius {
				t.Fatal(args)
			}
		})
	}
}

func TestTargetedReceiptCannotSubstituteAnotherEntity(t *testing.T) {
	host, r, v, id := fixture(t)
	attack(t, r, v, "task-1")
	_, err := r.Next(v, &behaviour.Result{TaskID: "task-1", Status: "succeeded", Detail: "observed Greyling death", Entity: "other"})
	if err != nil {
		t.Fatal(err)
	}
	g, _ := host.Store.Get(id)
	if g.Status != "needs_plan" || g.Proof != nil {
		t.Fatal(g)
	}
}
func TestStoreRejectsPolicyTamperingAndCopiesSnapshots(t *testing.T) {
	host, _, _, id := fixture(t)
	g, _ := host.Store.Get(id)
	g.State[0] = 'x'
	again, _ := host.Store.Get(id)
	if !json.Valid(again.State) {
		t.Fatal("snapshot alias")
	}
	raw, e := os.ReadFile(host.Store.path)
	if e != nil {
		t.Fatal(e)
	}
	raw = []byte(strings.Replace(string(raw), g.PolicyHash, strings.Repeat("a", 64), 1))
	if e = os.WriteFile(host.Store.path, raw, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = Open(host.Store.path); e == nil {
		t.Fatal("tampered policy accepted")
	}
}
func TestPolicyLimitsAndImmutableInputs(t *testing.T) {
	if e := ValidatePolicy(combatPolicy); e != nil {
		t.Fatal(e)
	}
	for _, src := range []string{
		"load('secret', 'x')\ndef step(context, state, outcome):\n return {}",
		"def step(context, state, outcome):\n for x in range(100):\n  pass\n return {}",
		"def step(context, state, outcome):\n return {'action': 'wait', 'state': 'a' * 999999999}",
		"def step(context, state, outcome):\n return step(context, state, outcome)",
		"def step(context, state, outcome):\n return context.clear()",
		"def step(context, state, outcome):\n return [x for x in state]",
	} {
		if e := ValidatePolicy(src); e == nil {
			t.Fatalf("accepted %s", src)
		}
	}
	source := "def step(context, state, outcome):\n return {\"action\": \"wait\", \"waitS\": 31}\n"
	if _, e := Evaluate(source, map[string]any{}, json.RawMessage(`{}`), nil); e == nil {
		t.Fatal("unbounded wait")
	}
	source = "def step(context, state, outcome):\n return {\"action\": \"wait\", \"waitS\": 1, \"unknown\": True}\n"
	if _, e := Evaluate(source, map[string]any{}, json.RawMessage(`{}`), nil); e == nil {
		t.Fatal("unknown directive field")
	}
}

func TestLifecycleSuspensionAndRepeatedFailuresRemainBounded(t *testing.T) {
	for _, inflight := range []bool{false, true} {
		host, r, v, id := fixture(t)
		if inflight {
			attack(t, r, v, "task-1")
		}
		raw, _ := json.Marshal(params{id})
		if e := host.End(raw, "controller stopped"); e != nil {
			t.Fatal(e)
		}
		g, _ := host.Store.Get(id)
		want := "suspended"
		if inflight {
			want = "needs_plan"
		}
		if g.Status != want {
			t.Fatal(g)
		}
		reopened, e := Open(host.Store.path)
		if e != nil {
			t.Fatal(e)
		}
		if e = (&Template{Store: reopened}).Queue(id); (e != nil) != inflight {
			t.Fatalf("inflight=%v: %v", inflight, e)
		}
	}
	host, r, v, id := fixture(t)
	attack(t, r, v, "first")
	for i := 0; i < 3; i++ {
		task := []string{"first", "second", "third"}[i]
		v.Perception.Characters[0].ID = task + "-new"
		o, e := r.Next(v, &behaviour.Result{Status: "failed", Reason: "refused", TaskID: task})
		if e != nil {
			t.Fatal(e)
		}
		if i < 2 {
			if o == nil {
				t.Fatal("lost objective before recovery budget")
			}
			if e = r.Dispatched([]string{"second", "third"}[i]); e != nil {
				t.Fatal(e)
			}
		} else if o != nil {
			t.Fatal("fourth attempt escaped repeat limit")
		}
	}
	g, _ := host.Store.Get(id)
	if g.Attempts != 3 || g.Status != "needs_plan" {
		t.Fatal(g)
	}
}
func TestDeadlineBoundsNativeTimeoutAndPolicyCannotChangeScope(t *testing.T) {
	host, r, v, id := fixture(t)
	if e := host.Store.update(id, func(g *Record) error { g.Deadline = v.Now.Add(2 * time.Second); return nil }); e != nil {
		t.Fatal(e)
	}
	o, e := r.Next(v, nil)
	if e != nil || o == nil || o.TimeoutS != 2 {
		t.Fatal(o, e)
	}
	host, r, v, id = fixture(t)
	source := "def step(context, state, outcome):\n return {\"action\": \"run\", \"template\": \"combat\", \"params\": {\"world\": \"other\", \"entity\": \"grey\"}}\n"
	if e = host.Store.update(id, func(g *Record) error { g.Policy = source; g.PolicyHash = hash(source); return nil }); e != nil {
		t.Fatal(e)
	}
	o, e = r.Next(v, nil)
	if e != nil || o != nil {
		t.Fatal(o, e)
	}
	g, _ := host.Store.Get(id)
	if g.Attempts != 0 || g.Status != "needs_plan" {
		t.Fatal(g)
	}
}
