// Package inspection projects existing evidence without planning or dispatching.
package inspection

import (
	"fmt"
	"math"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/goal"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

type Input struct {
	Now        time.Time
	Run        string
	Session    session.InspectionSnapshot
	Goal       *goal.Record
	Choices    []Choice
	Behaviours []behaviour.Status
}
type Choice struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
}
type Freshness struct {
	Status string     `json:"status"`
	At     *time.Time `json:"at,omitempty"`
	AgeS   *float64   `json:"ageS,omitempty"`
	Seq    uint64     `json:"seq,omitempty"`
}
type Body struct {
	Owner       string `json:"owner"`
	Instance    string `json:"instance,omitempty"`
	TaskID      string `json:"taskId,omitempty"`
	Task        string `json:"task,omitempty"`
	Status      string `json:"status,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Phase       string `json:"phase,omitempty"`
	Reason      string `json:"reason,omitempty"`
	Entity      string `json:"entity,omitempty"`
	RelatedGoal bool   `json:"relatedGoal"`
	StateMatch  bool   `json:"stateMatch"`
}
type Objective struct {
	ID           string    `json:"id"`
	Kind         string    `json:"kind"`
	Status       string    `json:"status"`
	World        string    `json:"world"`
	PolicyHash   string    `json:"policyHash"`
	Phase        string    `json:"phase,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	Decision     string    `json:"decision,omitempty"`
	AttemptsLeft int       `json:"attemptsLeft"`
	TasksLeft    int       `json:"tasksLeft"`
	Deadline     time.Time `json:"deadline"`
	RemainingS   float64   `json:"remainingS"`
	WaitedS      float64   `json:"waitedS"`
}
type Guard struct {
	Code     string `json:"code"`
	Label    string `json:"label"`
	Layer    string `json:"layer"`
	Result   string `json:"result"`
	Observed string `json:"observed"`
	Required string `json:"required"`
	Source   string `json:"source"`
}
type Event struct {
	Kind     string     `json:"kind"`
	Run      string     `json:"run,omitempty"`
	TaskID   string     `json:"taskId,omitempty"`
	At       *time.Time `json:"at,omitempty"`
	Summary  string     `json:"summary"`
	Entity   string     `json:"entity,omitempty"`
	AuditURL string     `json:"auditUrl,omitempty"`
}
type Snapshot struct {
	SchemaVersion int        `json:"schemaVersion"`
	Contract      string     `json:"contract"`
	At            time.Time  `json:"at"`
	Run           string     `json:"run"`
	Connection    uint64     `json:"connection"`
	World         string     `json:"world,omitempty"`
	WorldUID      string     `json:"worldUid,omitempty"`
	Freshness     Freshness  `json:"freshness"`
	Body          Body       `json:"body"`
	Choices       []Choice   `json:"choices"`
	Goal          *Objective `json:"goal,omitempty"`
	Graph         *Graph     `json:"graph,omitempty"`
	Guards        []Guard    `json:"guards"`
	Evidence      []Event    `json:"evidence"`
	Gaps          []string   `json:"gaps"`
}

func text(s string) string {
	r := []rune(s)
	if len(r) > 384 {
		return string(r[:384]) + "…"
	}
	return s
}
func truth(ok bool) string {
	if ok {
		return "pass"
	}
	return "fail"
}

// Project never upgrades a necessary telemetry predicate into native admission.
// Unexported checks stay unknown, even if the routine previously started.
func Project(in Input) Snapshot {
	s := in.Session
	o := Snapshot{SchemaVersion: 1, Contract: "inspection-v1", At: in.Now, Run: in.Run,
		Connection: s.Connection, Choices: in.Choices, Body: Body{Owner: "unavailable"},
		Guards: []Guard{}, Evidence: []Event{}, Gaps: []string{
			"Decision inputs and causal transitions are not yet archived; evidence below is a bounded checkpoint projection.",
			"Map and camera are live views, not retained evidence synchronized to these decisions.",
		}}
	if o.Choices == nil {
		o.Choices = []Choice{}
	}
	o.Freshness.Status = "offline"
	state := s.State
	if s.Connected {
		o.Freshness.Status = "unknown"
		if state != nil {
			o.World, o.WorldUID = state.World, state.WorldUID
			if !s.At.IsZero() {
				age := in.Now.Sub(s.At).Seconds()
				o.Freshness = Freshness{Status: "stale", At: &s.At, AgeS: &age, Seq: state.Seq}
				if age >= 0 && age <= 3 && state.Phase == protocol.PhaseInWorld && state.World != "" {
					o.Freshness.Status = "fresh"
				}
			}
		}
	}
	fresh := o.Freshness.Status == "fresh"
	if g := in.Goal; g != nil {
		o.Goal = &Objective{ID: g.ID, Kind: g.Kind, Status: g.Status, World: g.World, PolicyHash: g.PolicyHash,
			Reason: text(g.Reason), AttemptsLeft: max(0, g.MaxAttempts-g.Attempts), TasksLeft: max(0, g.MaxTasks-g.TaskCount),
			Deadline: g.Deadline, RemainingS: math.Max(0, g.Deadline.Sub(in.Now).Seconds()), WaitedS: g.WaitS}
		if g.Food != nil {
			o.Goal.Phase = g.Food.Phase
			if g.Food.Stock != nil {
				o.Goal.Decision = text(g.Food.Stock.Decision)
			}
			o.Graph = foodGraph(g.Food.Phase)
		} else if g.PostCombat != nil {
			o.Goal.Phase = g.PostCombat.Phase
		}
		o.Evidence = evidence(*g)
		if g.Pending != nil && g.Pending.Run == "" {
			o.Gaps = append(o.Gaps, "Pending intent has no recorded controller run; its effect and current ownership are uncertain.")
		} else if g.Pending != nil && g.Pending.Run != in.Run {
			o.Gaps = append(o.Gaps, "Pending intent belongs to an earlier controller run; its effect is uncertain and must not be replayed.")
		}
	}
	if fresh {
		projectBody(&o, in)
	}
	if !fresh {
		o.Body.Reason = "No fresh in-world observation; saved goal evidence is historical."
	}
	add := func(code, label, layer, result, observed, required, source string) {
		o.Guards = append(o.Guards, Guard{code, label, layer, result, text(observed), required, source})
	}
	observationResult := "unknown"
	if fresh {
		observationResult = "pass"
	} else if o.Freshness.Status == "stale" {
		observationResult = "stale"
	}
	add("state_freshness", "World observation", "observation", observationResult,
		o.Freshness.Status, "in-world state, age 0–3 s", "session.state")
	if g := in.Goal; g != nil {
		add("adapter_support", "Host adapter", "support", truth(g.Kind == "combat_trial" || g.Kind == "food_resupply"), g.Kind, "implemented goal adapter", "goal.kind")
		ownershipResult := "unknown"
		if fresh {
			ownershipResult = truth(o.Body.RelatedGoal)
		}
		add("goal_ownership", "Selected goal controls current task", "ownership", ownershipResult, o.Body.Owner, "current automation lease and matching task provenance", "session.dispatch + behaviour.episode")
	}
	permissionTask := o.Body.Task
	if permissionTask == "" && in.Goal != nil && in.Goal.Kind == "combat_trial" {
		permissionTask = protocol.TaskAttack
	}
	if permissionTask != "" {
		result, observed := "unknown", "plugin not connected"
		if s.Connected {
			result = truth(slices.Contains(s.Capabilities, permissionTask))
			observed = permissionTask
		}
		add("task_capability", "Plugin task capability", "support", result, observed, "task advertised by plugin", "handshake.capabilities")
		if s.Connected {
			result = truth(slices.Contains(s.Allowed, permissionTask))
		}
		add("controller_permission", "Effective controller permission", "permission", result, observed, "task in allowed set and live controller permit", "session.allowed")
		add("native_permission", "Native action admission", "permission", "unknown", "Not exported as a separate permission snapshot", "native permissions and per-action safety must pass", "native task; unavailable independently")
	}
	if fresh && o.Body.StateMatch && state.Combat != nil && state.Combat.ID == o.Body.TaskID && state.Combat.Peel != nil {
		p := state.Combat.Peel
		o.Graph = peelGraph(p.Phase)
		add("selected_pursuit", "Selected enemy pursuing", "pursuit", truth(slices.Contains(p.Pursuers, state.Combat.Entity)), state.Combat.Entity, "selected target in pursuer set", "state.combat.peel")
		add("single_pursuer", "One pursuer", "pursuit", truth(len(p.Pursuers) == 1), fmt.Sprint(len(p.Pursuers)), "exactly 1; necessary telemetry predicate", "state.combat.peel")
		add("known_group", "Accounted group targeting", "pursuit", truth(len(p.Unknown) == 0), fmt.Sprint(len(p.Unknown)), "0 unknown members", "state.combat.peel")
		add("quiet_interval", "Continuous isolation interval", "pursuit", truth(p.QuietS >= 3), fmt.Sprintf("%.1f s", p.QuietS), "at least 3 continuous s; not an admission receipt", "state.combat.peel")
		add("competitor_clearance", "Competitor clearance", "pursuit", "unknown", "Per-member distances are not exported", "native 24 m separation and fresh non-pursuit evidence", "native pursuit tracker")
		add("escape_validity", "Current escape corridor", "safety", "unknown", map[bool]string{true: "Planned endpoint reported", false: "No planned endpoint reported"}[p.Escape != nil], "native physical recheck must still pass", "state.combat.peel.escape; plan only")
		add("peel_preparation", "Food, gear and movement readiness", "readiness", "unknown", fmt.Sprintf("HP %.1f/%.1f · stamina %.1f", state.HP, state.HPMax, state.Stamina), "native phase-specific readiness, armour and durability checks", "state vitals; full native guard not exported")
	}
	if g := in.Goal; g != nil && g.Food != nil {
		f := g.Food
		add("chest_candidates", "Chest search ledger", "progress", "recorded", fmt.Sprintf("%d candidate chests · %d search ledger entries · %d receipts", len(f.Chests), len(f.Tried), len(f.Receipts)), "candidate-negative result does not prove every chest empty", "goal.food checkpoint")
		add("food_phase", "Food adapter phase", "progress", "recorded", f.Phase, "saved host phase; not live chest contents", "goal.food.phase")
	}
	if !fresh {
		o.Gaps = append(o.Gaps, "Live native phases and pursuit guards are withheld until a fresh in-world observation arrives.")
	}
	if o.Freshness.AgeS != nil && *o.Freshness.AgeS > 1 {
		for i := range o.Guards {
			if o.Guards[i].Layer == "pursuit" {
				o.Guards[i].Result = "stale"
			}
		}
	}
	return o
}

func projectBody(o *Snapshot, in Input) {
	s, state := in.Session, in.Session.State
	if slices.Contains(state.Flags, "recovering") {
		o.Body = Body{Owner: "native_recovery", Phase: "recovery", Reason: "Native startup recovery owns the body; normal tasks are held."}
		return
	}
	o.Body.Owner = "idle"
	t := s.Task
	if t != nil {
		o.Body.TaskID, o.Body.Task, o.Body.Status = t.ID, t.Name, t.Status
		origin, ok := t.Origin.(behaviour.Origin)
		if ok && t.AutomationCurrent {
			o.Body.Owner, o.Body.Instance = "behaviour", text(origin.Instance)
			for _, b := range in.Behaviours {
				if b.Name != origin.Instance || !b.Running {
					continue
				}
				if g := in.Goal; g != nil && g.Status == "running" && (b.EpisodeID == g.ID || b.Name == "goal/"+g.ID) &&
					g.Pending != nil && g.Pending.Run == in.Run && g.Pending.TaskID == t.ID && state.World == g.World {
					o.Body.RelatedGoal = true
				}
			}
		} else if ok {
			o.Body.Owner = "unconfirmed"
			o.Body.Reason = "The old behavior lease was revoked; this task cannot establish current goal ownership."
		} else {
			o.Body.Owner = "direct_or_system"
		}
		if state.Task == nil || state.Task.ID != t.ID || protocol.Final(state.Task.Status) {
			o.Body.Status = "awaiting_state"
			o.Body.Reason = "Controller dispatch is newer than the matching native task observation."
			return
		}
	} else if state.Task != nil && !protocol.Final(state.Task.Status) {
		o.Body.Owner = "native_unattributed"
		o.Body.TaskID, o.Body.Task, o.Body.Status = state.Task.ID, state.Task.Task, state.Task.Status
		o.Body.Reason = "Native task observed; current dispatch provenance is no longer retained."
	} else if s.Automation {
		o.Body.Owner = "automation"
	}
	if o.Body.TaskID == "" {
		return
	}
	o.Body.StateMatch = true
	if c := state.Combat; c != nil && c.ID == o.Body.TaskID {
		o.Body.Mode, o.Body.Reason, o.Body.Entity = c.Mode, text(c.Reason), text(c.Entity)
		if c.Peel != nil {
			o.Body.Phase = c.Peel.Phase
		}
	} else if j := state.Journey; j != nil && j.ID == o.Body.TaskID {
		o.Body.Mode, o.Body.Reason = j.Mode, text(j.Reason)
	} else if e := state.Escort; e != nil && e.ID == o.Body.TaskID {
		o.Body.Mode, o.Body.Reason, o.Body.Entity = e.Mode, text(e.Reason), text(e.Entity)
	}
}

func evidence(g goal.Record) []Event {
	items := []Event{}
	for _, h := range g.History {
		at := h.At
		e := Event{Kind: "outcome", Run: h.Run, TaskID: h.TaskID, At: recordedTime(at), Entity: text(h.Entity), Summary: text(strings.TrimSpace(h.Template + " · " + h.Status + " · " + h.Reason + " " + h.Detail))}
		if h.Run != "" && h.TaskID != "" && !at.IsZero() {
			e.AuditURL = "/api/task?" + url.Values{"id": {h.TaskID}, "at": {at.UTC().Format(time.RFC3339Nano)}}.Encode()
		}
		items = append(items, e)
	}
	if p := g.Pending; p != nil {
		at := p.At
		items = append(items, Event{Kind: "intent", Run: p.Run, TaskID: p.TaskID, At: recordedTime(at), Entity: text(p.Entity), Summary: text(p.Template + " · persisted intent; effect awaits reconciliation")})
	}
	if f := g.Food; f != nil {
		start := max(0, len(f.Receipts)-4)
		for _, r := range f.Receipts[start:] {
			items = append(items, Event{Kind: "inventory_receipt", Run: r.Run, TaskID: r.TaskID, Entity: text(r.Entity), Summary: text(fmt.Sprintf("%s · %d %s · %s", r.Action, r.Count, r.Item, r.Evidence))})
		}
	}
	// Receipt times are not invented. Timestamped evidence is ordered for display,
	// without claiming that wall-clock ordering proves a causal decision history.
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].At == nil {
			return false
		}
		if items[j].At == nil {
			return true
		}
		return items[i].At.Before(*items[j].At)
	})
	if len(items) > 12 {
		items = items[len(items)-12:]
	}
	return items
}

func recordedTime(at time.Time) *time.Time {
	if at.IsZero() {
		return nil
	}
	return &at
}
