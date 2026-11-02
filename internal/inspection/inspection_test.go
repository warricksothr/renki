package inspection

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/goal"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func fixture() Input {
	now := time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC)
	g := goal.Record{ID: "g1", Kind: "combat_trial", Status: "running", World: "one", PolicyHash: "hash", MaxTasks: 6, MaxAttempts: 6,
		Deadline: now.Add(time.Minute), Pending: &goal.Intent{Run: "run1", TaskID: "t-1", Template: "combat", At: now.Add(-time.Second)}}
	s := &protocol.State{Envelope: protocol.Envelope{Seq: 42}, Phase: "in_world", World: "one", WorldUID: "uid", HP: 60, HPMax: 60, Stamina: 60,
		Task: &protocol.TaskRef{ID: "t-1", Task: "attack", Status: "running"}, Combat: &protocol.CombatStatus{ID: "t-1", Entity: "grey", Mode: "kite", Peel: &protocol.PeelStatus{Phase: "lead", Members: 2, Pursuers: []string{"grey"}, Unknown: []string{"other"}, QuietS: 0}}}
	return Input{Now: now, Run: "run1", Goal: &g, Behaviours: []behaviour.Status{{Name: "goal/g1", EpisodeID: "g1", Running: true}},
		Session: session.InspectionSnapshot{Connected: true, Connection: 7, State: s, At: now, Automation: true, Capabilities: []string{"attack"}, Allowed: []string{"attack"},
			Task: &session.InspectionTask{ID: "t-1", Name: "attack", Status: "running", AutomationCurrent: true, Origin: behaviour.Origin{Instance: "goal/g1"}}}}
}
func guard(o Snapshot, code string) Guard {
	for _, g := range o.Guards {
		if g.Code == code {
			return g
		}
	}
	return Guard{}
}

func TestPeelInspectionUsesNativeEvidenceWithoutInventingAdmission(t *testing.T) {
	in := fixture()
	o := Project(in)
	if !o.Body.RelatedGoal || o.Body.Owner != "behaviour" || o.Graph.Active != "lead" {
		t.Fatal(o)
	}
	if guard(o, "known_group").Result != "fail" || guard(o, "quiet_interval").Result != "fail" {
		t.Fatal(o.Guards)
	}
	for _, code := range []string{"native_permission", "competitor_clearance", "escape_validity", "peel_preparation"} {
		if guard(o, code).Result != "unknown" {
			t.Fatal("invented native admission", code, o.Guards)
		}
	}
	// A delayed second pursuer invalidates single-pursuer evidence and resets
	// the observed quiet interval. The graph never invents a completed fight.
	in.Session.State.Combat.Peel.Pursuers = []string{"grey", "late"}
	o = Project(in)
	if guard(o, "single_pursuer").Result != "fail" || o.Body.Phase != "lead" {
		t.Fatal(o)
	}
	in.Session.State.Combat.Peel.Unknown = nil
	in.Session.State.Combat.Peel.Pursuers = []string{"grey"}
	in.Session.State.Combat.Peel.QuietS = 3
	o = Project(in)
	if guard(o, "quiet_interval").Result != "pass" || guard(o, "escape_validity").Result != "unknown" {
		t.Fatal(o.Guards)
	}
	in.Session.At = in.Now.Add(-1500 * time.Millisecond)
	o = Project(in)
	if o.Freshness.Status != "fresh" || guard(o, "quiet_interval").Result != "stale" {
		t.Fatal("pursuit evidence has a stricter freshness limit", o.Guards)
	}
}

func TestLegacyEvidenceDoesNotInventRunOrTimestamp(t *testing.T) {
	in := fixture()
	in.Goal.Pending.Run, in.Goal.Pending.At = "", time.Time{}
	in.Goal.History = []goal.Outcome{{TaskID: "legacy", Template: "combat"}}
	o := Project(in)
	if o.Body.RelatedGoal || !strings.Contains(strings.Join(o.Gaps, " "), "no recorded controller run") {
		t.Fatal(o)
	}
	for _, e := range o.Evidence {
		if e.At != nil || e.AuditURL != "" {
			t.Fatal("invented scope or time", e)
		}
	}
}

func TestOldGoalNeverOwnsNewOrderEvenBeforeStateCatchesUp(t *testing.T) {
	in := fixture()
	in.Session.Task = &session.InspectionTask{ID: "t-2", Name: "move_to", Origin: "private chat: SECRET"}
	o := Project(in)
	if o.Body.Owner != "direct_or_system" || o.Body.RelatedGoal || o.Body.Status != "awaiting_state" || o.Body.Phase != "" || o.Graph != nil {
		t.Fatal(o)
	}
	raw, _ := json.Marshal(o)
	if strings.Contains(string(raw), "SECRET") {
		t.Fatal("raw origin leaked")
	}
	in = fixture()
	in.Session.Task.AutomationCurrent = false
	o = Project(in)
	if o.Body.RelatedGoal || o.Body.Owner != "unconfirmed" {
		t.Fatal(o.Body)
	}
}

func TestOfflineStaleWrongScopeAndNativeRecoveryDoNotInheritGoalOwnership(t *testing.T) {
	for _, change := range []func(*Input){
		func(i *Input) { i.Session.Connected = false }, func(i *Input) { i.Session.At = i.Now.Add(-4 * time.Second) },
		func(i *Input) { i.Session.At = i.Now.Add(time.Second) }, func(i *Input) { i.Session.State.Phase = "menu" },
	} {
		in := fixture()
		change(&in)
		o := Project(in)
		if o.Body.RelatedGoal || o.Body.Phase != "" || guard(o, "known_group").Code != "" {
			t.Fatal(o)
		}
		if len(o.Evidence) != 1 || o.Evidence[0].Kind != "intent" {
			t.Fatal("saved intent disappeared", o)
		}
	}
	in := fixture()
	in.Session.State.World = "another"
	if Project(in).Body.RelatedGoal {
		t.Fatal("cross-world goal owner")
	}
	in = fixture()
	in.Goal.Pending.Run = "old-run"
	if Project(in).Body.RelatedGoal {
		t.Fatal("cross-run goal owner")
	}
	in = fixture()
	in.Session.State.Flags = []string{"recovering"}
	o := Project(in)
	if o.Body.Owner != "native_recovery" || o.Body.RelatedGoal || o.Graph != nil {
		t.Fatal(o)
	}
	in = fixture()
	in.Session.State.Combat.ID = "previous-task"
	if Project(in).Graph != nil {
		t.Fatal("mismatched combat phase attached")
	}
}

func TestPantryNegativeCandidateRetainsRemainingSearchAndVerifiedReceipt(t *testing.T) {
	in := fixture()
	g := in.Goal
	g.Kind = "food_resupply"
	g.Food = &goal.FoodProgress{Phase: "ready", Reserve: 2, Chests: []goal.FoodChest{{Entity: "A"}, {Entity: "B"}}, Tried: []string{"A"},
		Stock: &goal.FoodStock{Decision: "inspect the next allowed chest"}, Receipts: []goal.FoodReceipt{{Run: "run1", TaskID: "t-0", Action: "ready", Item: "Mushroom", Count: 1, Evidence: "matched_transfer_inventory"}}}
	g.History = []goal.Outcome{{Run: "run1", TaskID: "t-0", Template: "stock_probe", Status: "failed", Reason: "invalid_target", Detail: "requested item absent", At: in.Now.Add(-time.Second)}}
	g.Pending = &goal.Intent{Run: "run1", TaskID: "t-1", Template: "ready", Entity: "B", At: in.Now}
	in.Session.State.Combat = nil
	o := Project(in)
	if o.Graph.Active != "ready" || o.Goal.Decision != "inspect the next allowed chest" || !strings.Contains(guard(o, "chest_candidates").Observed, "2 candidate chests · 1 search ledger entries") {
		t.Fatal(o)
	}
	if len(o.Evidence) != 3 || o.Evidence[0].Kind != "outcome" || !strings.Contains(o.Evidence[0].AuditURL, "id=t-0") {
		t.Fatal(o.Evidence)
	}
	if o.Evidence[2].At != nil {
		t.Fatal("invented inventory receipt timestamp")
	}
	in.Session.Connected = false
	o = Project(in)
	if o.Body.RelatedGoal || o.Graph.Active != "ready" || guard(o, "food_phase").Result != "recorded" {
		t.Fatal(o)
	}
}

func TestInspectionIsCompactAndDoesNotIncludePolicyOrFullCheckpoints(t *testing.T) {
	in := fixture()
	in.Goal.Policy = "PRIVATE POLICY"
	in.Goal.Reason = strings.Repeat("x", 10000)
	for n := 0; n < 64; n++ {
		in.Choices = append(in.Choices, Choice{strings.Repeat("a", 24), "food_resupply", "needs_plan"})
	}
	for n := 0; n < 16; n++ {
		in.Goal.History = append(in.Goal.History, goal.Outcome{Run: "run1", TaskID: "t-1", Detail: strings.Repeat("y", 1000), At: in.Now})
	}
	o := Project(in)
	raw, err := json.Marshal(o)
	if err != nil || len(raw) > 32768 || strings.Contains(string(raw), "PRIVATE POLICY") || len(o.Evidence) > 12 {
		t.Fatal(len(raw), err)
	}
}
