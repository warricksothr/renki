package goal

import (
	"bytes"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// See policy.go for the enforced decision-tree subset.
//
//go:embed combat.star
var combatPolicy string

type Spec struct {
	Kind   string        `json:"kind"`
	World  string        `json:"world"`
	Center protocol.Vec3 `json:"center"`
	Radius float64       `json:"radius"`
}

func NewRecord(spec Spec, now time.Time) (Record, error) {
	var token [12]byte
	if _, err := rand.Read(token[:]); err != nil {
		return Record{}, err
	}
	r := Record{ID: hex.EncodeToString(token[:]), Kind: spec.Kind, World: spec.World, Center: spec.Center, Radius: spec.Radius, Policy: combatPolicy, PolicyHash: hash(combatPolicy), State: json.RawMessage(`{}`), Status: "ready", Created: now, Deadline: now.Add(10 * time.Minute), MaxAttempts: 6, MaxTasks: 6}
	return r, validateRecord(r)
}
func decode(raw []byte, to any) error { return behaviour.Decode(raw, to) }

type Template struct{ Store *Store }
type params struct {
	ID string `json:"id"`
}

func (t *Template) Validate(raw json.RawMessage) error {
	var p params
	if err := decode(raw, &p); err != nil {
		return err
	}
	r, ok := t.Store.Get(p.ID)
	if !ok {
		return errors.New("unknown goal")
	}
	if err := validateRecord(r); err != nil {
		return err
	}
	return ValidatePolicy(r.Policy)
}
func (t *Template) Start(raw json.RawMessage) (behaviour.Runner, error) {
	if err := t.Validate(raw); err != nil {
		return nil, err
	}
	var p params
	_ = decode(raw, &p)
	if err := t.Store.update(p.ID, func(r *Record) error {
		if r.Status != "queued" || r.Pending != nil {
			return errors.New("goal is not queued or has an uncertain action")
		}
		r.Status = "running"
		return nil
	}); err != nil {
		return nil, err
	}
	return &runner{t: t, id: p.ID}, nil
}
func (t *Template) End(raw json.RawMessage, why string) error {
	var p params
	if err := decode(raw, &p); err != nil {
		return err
	}
	return t.Store.update(p.ID, func(r *Record) error {
		if terminal(r.Status) || r.Status == "suspended" || r.Status == "ready" {
			return nil
		}
		r.Reason = why
		switch why {
		case "controller stopped", "unavailable", "session/world changed":
			r.Status = "suspended"
		case "new order", "stopped", "edited", "deleted":
			r.Status = "cancelled"
		default:
			r.Status = "needs_plan"
		}
		// Pending actions stay uncertain, even if stop was sent; only an observed
		// receipt consumed by Next can clear an intent.
		if r.Pending != nil && r.Status == "suspended" {
			r.Status = "needs_plan"
			r.Reason = "uncertain interrupted action: " + why
		}
		return nil
	})
}
func (t *Template) Queue(id string) error {
	return t.Store.update(id, func(r *Record) error {
		if r.Status != "ready" && r.Status != "suspended" && r.Status != "needs_plan" {
			return errors.New("goal cannot be resumed in this status")
		}
		if r.Pending != nil {
			return errors.New("uncertain action requires a new goal/plan; never replayed")
		}
		if time.Now().After(r.Deadline) || r.Attempts >= r.MaxAttempts || r.TaskCount >= r.MaxTasks || r.WaitS >= 120 {
			return errors.New("goal budget exhausted; cannot reset by resuming")
		}
		r.Status = "queued"
		r.Reason = ""
		return nil
	})
}
func (t *Template) Cancel(id string) error {
	return t.Store.update(id, func(r *Record) error {
		if r.Status == "succeeded" {
			return nil
		}
		r.Status = "cancelled"
		r.Reason = "stopped"
		return nil
	})
}

type runner struct {
	t           *Template
	id          string
	observation *Observation
}

func (r *runner) Recover(behaviour.Result) bool { return true }
func (r *runner) Dispatched(id string) error {
	return r.t.Store.update(r.id, func(g *Record) error {
		if g.Pending == nil {
			return errors.New("missing dispatch intent")
		}
		g.Pending.TaskID = id
		return nil
	})
}
func (r *runner) need(reason string) (*session.Order, error) {
	err := r.t.Store.update(r.id, func(g *Record) error { g.Status = "needs_plan"; g.Reason = reason; return nil })
	if err != nil {
		return nil, err
	}
	return nil, nil
}
func (r *runner) Next(v behaviour.View, last *behaviour.Result) (*session.Order, error) {
	if v.State != nil {
		r.observation = &Observation{At: v.Now, World: v.State.World, Pos: v.State.Pos, HP: v.State.HP, HPMax: v.State.HPMax, Stamina: v.State.Stamina}
	}
	g, ok := r.t.Store.Get(r.id)
	if !ok {
		return nil, errors.New("goal disappeared")
	}
	if terminal(g.Status) {
		return nil, nil
	}
	if v.Now.IsZero() {
		return nil, errors.New("goal requires host time")
	}
	if last != nil && g.Pending != nil {
		if last.TaskID == "" || last.TaskID != g.Pending.TaskID {
			return r.need("native receipt does not match pending action")
		}
		outcome := Outcome{Status: last.Status, Reason: last.Reason, Detail: truncate(last.Detail, 512), TaskID: last.TaskID, Entity: g.Pending.Entity, At: v.Now}
		verified := last.Status == protocol.StatusSucceeded && last.Detail == "observed Greyling death" && g.Pending.Template == "combat" && g.Pending.Entity != ""
		if last.Status == protocol.StatusSucceeded && !verified {
			outcome.Status = "failed"
			outcome.Reason = "unverified_success"
		}
		if err := r.t.Store.update(r.id, func(g *Record) error {
			if g.Last != nil && g.Last.Reason == outcome.Reason {
				g.Repeated++
			} else {
				g.Repeated = 1
			}
			g.Last = &outcome
			g.Pending = nil
			g.History = append(g.History, outcome)
			if len(g.History) > 16 {
				g.History = g.History[len(g.History)-16:]
			}
			if verified {
				g.Proof = &outcome
			}
			return nil
		}); err != nil {
			return nil, err
		}
		g, _ = r.t.Store.Get(r.id)
	}
	if g.Pending != nil {
		return r.need("uncertain action has no receipt; not replayed")
	}
	if v.State == nil || v.State.World != g.World || v.State.Phase != protocol.PhaseInWorld || v.State.HP <= 0 {
		return r.need("original world and living character required")
	}
	if v.StateAt.IsZero() || v.Now.Sub(v.StateAt) < 0 || v.Now.Sub(v.StateAt) > 3*time.Second {
		return r.need("fresh state observation required")
	}
	if !v.Now.Before(g.Deadline) {
		return r.need("overall goal deadline exhausted")
	}
	if distance(v.State.Pos, g.Center) > g.Radius-52 {
		return r.need("outside goal's conservative native travel envelope")
	}
	if g.Proof != nil {
		// Completion belongs to the host, even if the policy would keep running.
		return nil, r.t.Store.update(r.id, func(g *Record) error { g.Status = "succeeded"; g.Reason = "observed Greyling death"; return nil })
	}
	if g.Attempts >= g.MaxAttempts || g.TaskCount >= g.MaxTasks {
		return r.need("attempt/native-task budget exhausted")
	}
	if g.Repeated >= 3 {
		return r.need("same native failure repeated three times")
	}
	if g.Last != nil && (g.Last.Reason == "unverified_success" || g.Last.Reason == protocol.ReasonNotAllowed) {
		return r.need("permission or unverified outcome requires a plan")
	}
	if v.State.Food == nil || len(v.State.Food.Active) == 0 {
		return r.need("no active food; replenish before combat")
	}
	if v.Inventory == nil || v.State.Equipped["right"] == "" { // Native checks remain final; inventory/tool readiness is also shown to policy.
		// The native routine can equip a carried axe; require evidence of one.
		found := false
		if v.Inventory != nil {
			for _, item := range v.Inventory.Items {
				if strings.Contains(item.Prefab, "Axe") && item.Stack > 0 {
					found = true
				}
			}
		}
		if !found {
			return r.need("no known carried/equipped combat tool")
		}
	}
	if g.Wake.After(v.Now) {
		return nil, &behaviour.Delay{Until: g.Wake}
	}
	context, params, fingerprint := combatContext(g, v)
	r.observation.Ready, _ = context["ready"].(bool)
	if p, ok := context["params"].(map[string]any); ok {
		r.observation.Target, _ = p["entity"].(string)
	}
	directive, err := Evaluate(g.Policy, context, g.State, g.Last)
	if err != nil {
		return r.need("policy rejected: " + truncate(err.Error(), 400))
	}
	switch directive.Action {
	case "done":
		return r.need("policy requested completion without native proof")
	case "needs_plan":
		return r.need(directive.Reason)
	case "wait":
		if g.WaitS+directive.WaitS > 120 {
			return r.need("observation/recovery wait budget exhausted")
		}
		until := v.Now.Add(time.Duration(directive.WaitS * float64(time.Second)))
		if err = r.t.Store.update(r.id, func(g *Record) error {
			g.Observation = r.observation
			g.State = directive.State
			g.WaitS += directive.WaitS
			g.Wake = until
			g.Reason = directive.Reason
			return nil
		}); err != nil {
			return nil, err
		}
		return nil, &behaviour.Delay{Until: until}
	case "run":
		if directive.Template != "combat" || params == nil || fingerprint == g.LastFingerprint {
			return r.need("policy requested unavailable or unchanged recovery action")
		}
		if !sameJSON(params, directive.Params) {
			return r.need("policy attempted to alter host combat scope/target")
		}
		template := behaviour.Combat{}
		if err = template.Validate(directive.Params); err != nil {
			return r.need(err.Error())
		}
		child, err := template.Start(directive.Params)
		if err != nil {
			return r.need(err.Error())
		}
		order, err := child.Next(v, nil)
		if err != nil {
			return r.need(err.Error())
		}
		if order == nil || order.Task != protocol.TaskAttack {
			return r.need("combat runner did not produce an attack")
		}
		remaining := g.Deadline.Sub(v.Now).Seconds()
		if remaining < 1 {
			return r.need("overall goal deadline exhausted")
		}
		order.TimeoutS = math.Min(order.TimeoutS, remaining)
		args := order.Args.(*protocol.AttackArgs)
		if err = r.t.Store.update(r.id, func(g *Record) error {
			g.Observation = r.observation
			g.State = directive.State
			g.Attempts++
			g.TaskCount++
			g.Wake = time.Time{}
			g.Reason = directive.Reason
			g.LastFingerprint = fingerprint
			g.Pending = &Intent{Template: "combat", Params: directive.Params, Entity: args.Entity, Fingerprint: fingerprint}
			return nil
		}); err != nil {
			return nil, err
		}
		return order, nil
	}
	return nil, errors.New("unhandled policy directive")
}
func combatContext(g Record, v behaviour.View) (map[string]any, json.RawMessage, string) {
	ctx := map[string]any{"ready": v.State.HPMax > 0 && v.State.HP/v.State.HPMax >= 0.8 && v.State.Stamina >= 40, "target": false, "changed": false, "params": nil}
	if v.PerceptionAt.IsZero() || v.Now.Sub(v.PerceptionAt) < 0 || v.Now.Sub(v.PerceptionAt) > 3*time.Second || v.Perception == nil || v.State.T-v.Perception.T < 0 || v.State.T-v.Perception.T > 3 {
		return ctx, nil, ""
	}
	targets := append([]protocol.PerceivedCharacter(nil), v.Perception.Characters...)
	sort.Slice(targets, func(i, j int) bool {
		return distance(v.State.Pos, targets[i].Pos) < distance(v.State.Pos, targets[j].Pos)
	})
	for _, c := range targets {
		if c.Kind != protocol.CharacterCreature || c.Prefab != "Greyling" || c.Stars != 0 || !c.Hostile || c.HPFrac <= 0 || distance(c.Pos, g.Center) > g.Radius-52 || distance(c.Pos, v.State.Pos) > 30 {
			continue
		}
		direct := c.TargetingMe && distance(c.Pos, v.State.Pos) <= 4
		p := map[string]any{"world": g.World, "entity": c.ID}
		if direct {
			p["direct"] = true
		}
		raw, _ := json.Marshal(p)
		fp := fmt.Sprintf("%s:%t:%t:%d:%d:%d:%d:%d:%d", c.ID, direct, c.TargetingMe, int(math.Floor(c.Pos[0]/2)), int(math.Floor(c.Pos[2]/2)), int(math.Floor(v.State.Pos[0]/2)), int(math.Floor(v.State.Pos[2]/2)), int(v.State.HP/10), int(v.State.Stamina/20))
		if fp == g.LastFingerprint {
			continue
		} // Prefer a changed/alternate target over repeating a refusal.
		ctx["target"], ctx["changed"], ctx["params"] = true, true, p
		return ctx, raw, fp
	}
	return ctx, nil, ""
}
func sameJSON(a, b []byte) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	aa, _ := json.Marshal(x)
	bb, _ := json.Marshal(y)
	return bytes.Equal(aa, bb)
}
func distance(a, b protocol.Vec3) float64 {
	return math.Sqrt(math.Pow(a[0]-b[0], 2) + math.Pow(a[1]-b[1], 2) + math.Pow(a[2]-b[2], 2))
}
func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
