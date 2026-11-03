package behaviour

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// SearchChests inspects ordinary storage without taking anything. Definitions
// may use manual, timer or event triggers; a restart never resumes a manual run.
type SearchChests struct{}

type SearchChestsParams struct {
	World     string         `json:"world"`
	Item      string         `json:"item"`
	Chests    []string       `json:"chests,omitempty"`
	Center    *protocol.Vec3 `json:"center,omitempty"`
	Radius    float64        `json:"radius,omitempty"`
	MaxChests int            `json:"maxChests,omitempty"`
	DurationS float64        `json:"durationS,omitempty"`
	TravelM   float64        `json:"travelM,omitempty"`
}

func (p *SearchChestsParams) defaults() {
	if p.Radius == 0 {
		p.Radius = 25
	}
	if p.MaxChests == 0 {
		p.MaxChests = 16
	}
	if p.DurationS == 0 {
		p.DurationS = 180
	}
	if p.TravelM == 0 {
		p.TravelM = 300
	}
}

func (SearchChests) Validate(raw json.RawMessage) error {
	var p SearchChestsParams
	if err := Decode(raw, &p); err != nil {
		return err
	}
	p.defaults()
	if strings.TrimSpace(p.World) == "" || strings.TrimSpace(p.Item) == "" || len(p.Item) > 80 ||
		!finite(p.Radius) || p.Radius < 10 || p.Radius > 100 || p.MaxChests < 1 || p.MaxChests > 32 ||
		!finite(p.DurationS) || p.DurationS < 10 || p.DurationS > 300 || !finite(p.TravelM) || p.TravelM < 10 || p.TravelM > 1000 ||
		len(p.Chests) > 0 && !namesOK(p.Chests, 32) {
		return errors.New("search needs world/item, radius 10–100m, 1–32 chests, 10–300s and 10–1000m travel")
	}
	if p.Center != nil {
		for _, n := range *p.Center {
			if !finite(n) || math.Abs(n) > 1e6 {
				return errors.New("invalid search center")
			}
		}
	}
	return nil
}

func (t SearchChests) Start(raw json.RawMessage) (Runner, error) {
	if err := t.Validate(raw); err != nil {
		return nil, err
	}
	var p SearchChestsParams
	_ = Decode(raw, &p)
	p.defaults()
	return &chestSearchRun{p: p, seen: map[string]bool{}}, nil
}

func (t SearchChests) Ready(raw json.RawMessage, v View) bool {
	r, err := t.Start(raw)
	return err == nil && r.(*chestSearchRun).Check(v) == nil
}

type chestCandidate struct {
	id  string
	pos protocol.Vec3
}
type chestFinding struct {
	chestCandidate
	count int
}
type chestSearchRun struct {
	p                           SearchChestsParams
	center, lastPos             protocol.Vec3
	began                       time.Time
	health                      healthSample
	travel                      float64
	seen                        map[string]bool
	queue                       []chestCandidate
	found                       []chestFinding
	checked, unknown, attempted int
	problems                    []string
	pending                     *chestCandidate
	taskID                      string
	beforeSeq                   uint64
	reported                    bool
	legStartTravel              float64
	stop                        bool
}

func (r *chestSearchRun) Check(v View) error {
	if !idleFresh(v) || v.State.World != r.p.World || v.Perception == nil ||
		v.PerceptionAt.IsZero() || v.Now.Sub(v.PerceptionAt) < 0 || v.Now.Sub(v.PerceptionAt) > 3*time.Second || math.Abs(v.State.T-v.Perception.T) > 3 {
		return errors.New("chest search needs fresh state and surroundings in the requested world")
	}
	if !slices.Contains(v.Features, protocol.FeatureInspectContainer) {
		return errors.New("plugin does not support read-only chest inspection")
	}
	s := v.State
	for _, n := range s.Pos {
		if !finite(n) || math.Abs(n) > 1e6 {
			return errors.New("invalid body position")
		}
	}
	if !finite(s.HP) || !finite(s.HPMax) || s.HPMax <= 0 || s.HP < math.Max(15, .4*s.HPMax) ||
		!slices.Contains(s.Flags, protocol.FlagGrounded) || slices.Contains(s.Flags, protocol.FlagSwimming) || s.Location != nil && s.Location.InWater {
		return errors.New("chest search needs healthy dry footing")
	}
	for _, c := range v.Perception.Characters {
		if c.HPFrac > 0 && (c.TargetingMe || c.Hostile && distance(c.Pos, s.Pos) < 20) {
			return errors.New("nearby threat interrupts chest search")
		}
	}
	if !r.began.IsZero() {
		r.travel += distance(s.Pos, r.lastPos)
		r.lastPos = s.Pos
		health := sampleHealth(v)
		if r.health.lost(health) {
			return errors.New("injury interrupts chest search")
		}
		r.health = r.health.roll(health)
		if s.Pos.FlatDistance(r.center) > r.p.Radius || r.travel >= r.p.TravelM || v.Now.Sub(r.began).Seconds() >= r.p.DurationS {
			return errors.New("chest search budget/scope ended")
		}
	}
	return nil
}

func (r *chestSearchRun) discover(v View) {
	known := map[string]protocol.Vec3{}
	for _, c := range v.Containers {
		known[c.Entity] = c.Pos
	}
	for _, c := range v.Perception.Structures {
		if c.Kind == protocol.StructureContainer {
			known[c.ID] = c.Pos
		}
	}
	var add []chestCandidate
	if len(r.p.Chests) > 0 {
		for _, id := range r.p.Chests {
			if r.seen[id] {
				continue
			}
			pos, ok := known[id]
			if !ok {
				r.seen[id] = true
				r.unknown++
				r.problems = append(r.problems, id+": location unknown")
				continue
			}
			add = append(add, chestCandidate{id, pos})
		}
	} else {
		for id, pos := range known {
			if !r.seen[id] && pos.FlatDistance(r.center) <= r.p.Radius {
				add = append(add, chestCandidate{id, pos})
			}
		}
	}
	sort.Slice(add, func(i, j int) bool {
		a, b := distance(v.State.Pos, add[i].pos), distance(v.State.Pos, add[j].pos)
		if a == b {
			return add[i].id < add[j].id
		}
		return a < b
	})
	for _, c := range add {
		r.seen[c.id] = true
		if c.pos.FlatDistance(r.center) > r.p.Radius {
			r.unknown++
			r.problems = append(r.problems, c.id+": outside radius")
			continue
		}
		r.queue = append(r.queue, c)
	}
}

func (r *chestSearchRun) Recover(result Result) bool {
	return slices.Contains([]string{protocol.ReasonInvalidTarget, protocol.ReasonNotAllowed, protocol.ReasonRefused, protocol.ReasonTargetLost, protocol.ReasonUnreachable, protocol.ReasonStuck, protocol.ReasonTimeout}, result.Reason)
}

func (r *chestSearchRun) Dispatched(id string) error { r.taskID = id; return nil }

func searchItem(item string, it protocol.InventoryItem) bool {
	normal := func(s string) string {
		return strings.ToLower(strings.NewReplacer(" ", "", "_", "", "-", "").Replace(s))
	}
	return normal(item) == normal(it.Prefab) || strings.EqualFold(item, "food") && it.Type == "Consumable"
}

func (r *chestSearchRun) Next(v View, result *Result) (*session.Order, error) {
	if err := r.Check(v); err != nil {
		return nil, err
	}
	if r.began.IsZero() {
		r.center = v.State.Pos
		if r.p.Center != nil {
			r.center = *r.p.Center
		}
		if v.State.Pos.FlatDistance(r.center) > r.p.Radius {
			return nil, errors.New("body is outside search radius")
		}
		r.began = v.Now
		r.lastPos = v.State.Pos
		r.health = sampleHealth(v)
	}
	if r.reported {
		return nil, nil
	}
	if r.pending != nil {
		if result == nil || result.TaskID != r.taskID {
			return nil, errors.New("missing matching inspection outcome")
		}
		var measured struct {
			Travel *float64 `json:"travel"`
		}
		if strings.HasPrefix(result.Detail, "inspection:") && json.Unmarshal([]byte(strings.TrimPrefix(result.Detail, "inspection:")), &measured) == nil &&
			measured.Travel != nil && finite(*measured.Travel) && *measured.Travel >= 0 {
			r.travel = math.Max(r.travel, r.legStartTravel+*measured.Travel)
		} else if result.Status != protocol.StatusRejected {
			// Cancellation, an outer timeout or a last-moment driver refusal
			// may lack native travel accounting. Do not replenish that budget
			// by continuing with controller-only displacement estimates.
			r.stop = true
			r.problems = append(r.problems, "native travel receipt missing; search stopped")
		}
		if result.Status == protocol.StatusSucceeded {
			var receipt *protocol.Container
			for _, c := range v.Containers {
				if c.Entity == r.pending.id && c.Task == r.taskID && c.Direction == protocol.TaskInspectContainer && c.Seq > r.beforeSeq && math.Abs(v.State.T-c.T) <= 3 {
					receipt = c
					break
				}
			}
			if receipt == nil {
				return nil, ErrWait
			}
			count := 0
			for _, it := range receipt.Items {
				if searchItem(r.p.Item, it) && it.Stack > 0 {
					count += it.Stack
				}
			}
			r.checked++
			if count > 0 {
				r.found = append(r.found, chestFinding{chestCandidate{receipt.Entity, receipt.Pos}, count})
			}
		} else {
			r.unknown++
			r.problems = append(r.problems, r.pending.id+": "+result.Reason)
		}
		r.pending = nil
	}
	r.discover(v)
	if r.stop || len(r.queue) == 0 || r.attempted >= r.p.MaxChests || r.p.DurationS-v.Now.Sub(r.began).Seconds() < 1 || r.p.TravelM-r.travel < 1 {
		r.reported = true
		line := fmt.Sprintf("%s: checked %d, unchecked %d, matches in %d chests. Unobserved chests remain unknown.", r.p.Item, r.checked, r.unknown+len(r.queue), len(r.found))
		if len(r.found) > 0 {
			f := r.found[0]
			line += fmt.Sprintf(" %d at %s [%.0f,%.0f,%.0f].", f.count, f.id, f.pos[0], f.pos[1], f.pos[2])
		}
		if len([]rune(line)) > 200 {
			line = string([]rune(line)[:197]) + "..."
		}
		return &session.Order{Task: protocol.TaskSay, Args: &protocol.SayArgs{Text: line}, TimeoutS: 5}, nil
	}
	c := r.queue[0]
	r.queue = r.queue[1:]
	r.pending = &c
	r.attempted++
	r.legStartTravel = r.travel
	r.beforeSeq = 0
	for _, old := range v.Containers {
		if old.Entity == c.id && old.Seq > r.beforeSeq {
			r.beforeSeq = old.Seq
		}
	}
	seconds := math.Min(45, r.p.DurationS-v.Now.Sub(r.began).Seconds())
	return &session.Order{Task: protocol.TaskInspectContainer, Args: &protocol.InspectContainerArgs{Container: c.id, Scope: &protocol.MoveScope{Center: r.center, Radius: r.p.Radius, TravelM: math.Min(500, r.p.TravelM-r.travel), DurationS: seconds}}, TimeoutS: seconds}, nil
}

func (r *chestSearchRun) InterruptedSummary() string { return r.Summary() }

func (r *chestSearchRun) Summary() string {
	parts := []string{}
	for _, f := range r.found {
		parts = append(parts, fmt.Sprintf("%d at %s [%.0f,%.0f,%.0f]", f.count, f.id, f.pos[0], f.pos[1], f.pos[2]))
	}
	if len(parts) == 0 {
		parts = append(parts, "no matches confirmed")
	}
	unknown := r.unknown + len(r.queue)
	if r.pending != nil {
		unknown++
	}
	return fmt.Sprintf("%s: checked %d; unchecked %d; matches in %d chests. Unobserved chests may remain. %s. %s", r.p.Item, r.checked, unknown, len(r.found), strings.Join(parts, "; "), strings.Join(r.problems, "; "))
}

const chestSearchInstance = "Owner chest search"
const chestSearchSource = "chat:chest-search:"

// StartChestSearch admits only a fresh idle request, without replacing retreat.
func (e *Engine) StartChestSearch(ctx context.Context, by string, request SearchChestsParams) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if strings.TrimSpace(by) == "" {
		return "", errors.New("a player must request the search")
	}
	hello, _, _ := e.o.Session.Handshake()
	v := e.view()
	revision := e.o.Session.ControlRevision()
	if !v.Idle || e.active != "" {
		return "", errors.New("stop my current task first")
	}
	for _, i := range e.instances {
		if i.pending {
			return "", errors.New("stop queued behavior first")
		}
	}
	if v.State == nil {
		return "", errors.New("fresh world required")
	}
	request.World = v.State.World
	request.defaults()
	if request.Center == nil {
		pos := v.State.Pos
		request.Center = &pos
	}
	raw, _ := json.Marshal(request)
	t := SearchChests{}
	r, err := t.Start(raw)
	if err != nil {
		return "", err
	}
	if err = r.(*chestSearchRun).Check(v); err != nil {
		return "", err
	}
	admission, err := e.o.Session.TryAutomationFor(hello, v.State.World)
	if err != nil {
		return "", errors.New("stop my current task first")
	}
	defer admission.Release()
	for _, task := range []string{protocol.TaskInspectContainer, protocol.TaskSay, protocol.TaskStop} {
		if !e.o.Session.Allowed(task) {
			return "", fmt.Errorf("%s is not allowed", task)
		}
	}
	if old := e.instances[chestSearchInstance]; old != nil && (old.definition.Template != "search_chests" || !strings.HasPrefix(e.o.Store.Instances()[chestSearchInstance].By, chestSearchSource)) {
		return "", errors.New("rename the existing Owner chest search behavior first")
	}
	d := Definition{Template: "search_chests", Params: raw, Trigger: Trigger{Kind: "manual"}, Repeat: Repeat{Mode: "once"}, Priority: 10}
	if err = e.put(ctx, chestSearchInstance, d, chestSearchSource+by); err != nil {
		return "", err
	}
	if err = e.action(ctx, chestSearchInstance, "start", chestSearchSource+by); err != nil {
		return "", err
	}
	if e.o.Session.ControlRevision() != revision {
		_ = e.action(ctx, chestSearchInstance, "stop", chestSearchSource+by)
		return "", errors.New("a newer order replaced the search")
	}
	return fmt.Sprintf("I'll look for %s in up to %d chests within %.0fm, report what I checked, and leave the items there. Say stop to cancel.", request.Item, request.MaxChests, request.Radius), nil
}
