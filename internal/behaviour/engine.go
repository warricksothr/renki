package behaviour

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
	"github.com/warricksothr/renki/internal/settings"
)

const MaxInstances = 256

type Options struct {
	Session    *session.Session
	Store      *settings.Store
	Templates  map[string]Template
	Owners     func() []string
	Home       func() (protocol.Vec3, bool)
	Containers func(world string) []home.RememberedContainer
	Landmarks  func(world string) []home.Landmark
	Danger     func(world string, now time.Time) []home.DangerZone
	Logger     *slog.Logger
	Now        func() time.Time
	Notify     func(context.Context, string)
}
type Engine struct {
	mu                  sync.Mutex
	o                   Options
	instances           map[string]*instance
	active              string
	lease               *session.Automation
	events              chan string
	dropped             atomic.Uint64
	conn                *protocol.Hello
	world               string
	orders              uint64
	full, night, atHome bool
	drops               map[string]bool
}
type instance struct {
	definition            Definition
	pending               bool
	completed             int
	next, timer, deadline time.Time
	runner                Runner
	task                  *session.Task
	result                *Result
	step                  int
	last                  string
	waitingSince          time.Time
	storage               *StorageNeed
	notice                string
	noticeAt              time.Time
	requestConn           *protocol.Hello
	requestWorld          string
}
type Status struct {
	Name          string       `json:"name"`
	Definition    Definition   `json:"definition"`
	Runs          int          `json:"runs"`
	Pending       bool         `json:"pending"`
	Running       bool         `json:"running"`
	Last          string       `json:"last,omitempty"`
	StorageNeeded *StorageNeed `json:"storageNeeded,omitempty"`
}
type Origin struct {
	Instance string
	Step     int
}

func New(o Options) *Engine {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.Owners == nil {
		o.Owners = func() []string { return nil }
	}
	templates := map[string]Template{"move_to": MoveTo{}, "stow": Stow{}, "collect": Collect{}, "gather": Gather{}, "combat": Combat{}}
	for name, t := range o.Templates {
		templates[name] = t
	}
	o.Templates = templates
	e := &Engine{o: o, instances: map[string]*instance{}, events: make(chan string, 64), orders: o.Session.ControlRevision()}
	// Settings are untrusted on disk too: invalid/removed templates do not run.
	savedInstances := o.Store.Instances()
	names := make([]string, 0, len(savedInstances))
	for name := range savedInstances {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		saved := savedInstances[name]
		if name == "" || len(name) > 80 || strings.TrimSpace(name) != name {
			o.Logger.Warn("invalid instance name", "instance", name)
			continue
		}
		var d Definition
		if err := Decode(saved.Value, &d); err != nil {
			o.Logger.Warn("invalid instance", "instance", name, "err", err)
			continue
		}
		if err := validate(d, templates); err != nil {
			o.Logger.Warn("invalid instance", "instance", name, "err", err)
			continue
		}
		if len(e.instances) >= MaxInstances {
			o.Logger.Warn("instance limit reached")
			break
		}
		e.instances[name] = e.makeInstance(d)
	}
	return e
}
func duration(s float64) time.Duration { return time.Duration(s * float64(time.Second)) }
func (e *Engine) makeInstance(d Definition) *instance {
	return &instance{definition: d, timer: e.o.Now().Add(duration(d.Trigger.EveryS))}
}
func (e *Engine) List() []Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]Status, 0, len(e.instances))
	for name, i := range e.instances {
		var storage *StorageNeed
		if i.storage != nil {
			copy := *i.storage
			copy.Items = maps.Clone(copy.Items)
			copy.Chests = slices.Clone(copy.Chests)
			storage = &copy
		}
		out = append(out, Status{Name: name, Definition: i.definition, Runs: i.completed, Pending: i.pending, Running: e.active == name, Last: i.last, StorageNeeded: storage})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	// Neither callers nor templates can mutate the engine's definitions.
	raw, _ := json.Marshal(out)
	var cloned []Status
	_ = json.Unmarshal(raw, &cloned)
	return cloned
}
func (e *Engine) Put(ctx context.Context, name string, raw json.RawMessage, by string) error {
	var d Definition
	if err := Decode(raw, &d); err != nil {
		return err
	}
	if err := validate(d, e.o.Templates); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.put(ctx, name, d, by)
}
func (e *Engine) put(ctx context.Context, name string, d Definition, by string) error {
	if name == "" || len(name) > 80 || strings.TrimSpace(name) != name || by == "" {
		return errors.New("instance needs a name (1–80 bytes) and source")
	}
	if _, ok := e.instances[name]; !ok && len(e.instances) >= MaxInstances {
		return errors.New("instance limit reached")
	}
	if e.instances[name] != nil && e.instances[name].pending {
		e.ended(e.instances[name], "edited")
	}
	if e.active == name {
		if err := e.cancel(ctx, "edited", false); err != nil {
			return err
		}
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return err
	}
	if err = e.o.Store.SetInstance(name, raw, by); err != nil {
		return err
	}
	e.instances[name] = e.makeInstance(d)
	return nil
}
func (e *Engine) Action(ctx context.Context, name, action, by string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.action(ctx, name, action, by)
}
func (e *Engine) action(ctx context.Context, name, action, by string) error {
	i := e.instances[name]
	if i == nil {
		return fmt.Errorf("unknown instance %q", name)
	}
	if by == "" {
		return errors.New("instance change needs a source")
	}
	switch action {
	case "start":
		if e.active == name && e.lease.Valid() {
			return nil
		}
		if current := e.o.Session.ControlRevision(); current != e.orders {
			if err := e.cancel(ctx, "new order", false); err != nil {
				return err
			}
			e.clearPending("new order")
			e.orders = current
		}
		hello, _, connected := e.o.Session.Handshake()
		state := e.o.Session.State()
		if !connected || state == nil || state.Phase != protocol.PhaseInWorld || state.HP <= 0 {
			return session.ErrBodyBusy
		}
		if !i.definition.Enabled {
			d := i.definition
			d.Enabled = true
			if err := e.put(ctx, name, d, by); err != nil {
				return err
			}
			i = e.instances[name]
		}
		i.requestConn, i.requestWorld = hello, state.World
		i.pending = true
		i.completed = 0
		i.next = time.Time{}
		e.o.Logger.Info("behaviour triggered", "instance", name, "trigger", "manual", "by", by)
	case "stop":
		if e.active == name {
			if err := e.cancel(ctx, "stopped", false); err != nil {
				return err
			}
		}
		if i.pending {
			e.ended(i, "stopped")
		}
		i.pending = false
		e.o.Logger.Info("behaviour stopped", "instance", name, "by", by)
	case "enable", "disable":
		d := i.definition
		d.Enabled = action == "enable"
		if d.Enabled == i.definition.Enabled {
			return nil
		}
		return e.put(ctx, name, d, by)
	case "delete":
		if i.pending {
			e.ended(i, "deleted")
		}
		if e.active == name {
			if err := e.cancel(ctx, "deleted", false); err != nil {
				return err
			}
		}
		if err := e.o.Store.DeleteInstance(name, by); err != nil {
			return err
		}
		delete(e.instances, name)
	default:
		return fmt.Errorf("unknown behaviour action %q", action)
	}
	return nil
}
func (e *Engine) Clone(ctx context.Context, from, to, by string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	i := e.instances[from]
	if i == nil {
		return fmt.Errorf("unknown instance %q", from)
	}
	if _, ok := e.instances[to]; ok {
		return errors.New("clone destination already exists")
	}
	d := i.definition
	d.Enabled = false
	return e.put(ctx, to, d, by)
}

// Event is bounded and never blocks the bridge's read goroutine.
func (e *Engine) Event(event string) {
	if !eventNames[event] {
		return
	}
	select {
	case e.events <- event:
	default:
		e.dropped.Add(1)
	}
}

// Limit work even if producers continuously refill the bounded buffer.
func (e *Engine) drainEvents() map[string]bool {
	events := map[string]bool{}
	for range cap(e.events) {
		select {
		case event := <-e.events:
			events[event] = true
		default:
			return events
		}
	}
	return events
}

func (e *Engine) Run(ctx context.Context) {
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			stop, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			e.mu.Lock()
			_ = e.cancel(stop, "controller stopped", false)
			e.mu.Unlock()
			cancel()
			return
		case <-t.C:
			step, cancel := context.WithTimeout(ctx, 3*time.Second)
			e.Tick(step)
			cancel()
		}
	}
}
func (e *Engine) view() View {
	state, stateAt := e.o.Session.StateObservation()
	hello, world, perception, perceptionAt := e.o.Session.PerceptionObservation()
	currentHello, _, _ := e.o.Session.Handshake()
	if state == nil || hello != currentHello || world != state.World {
		perception = nil
		perceptionAt = time.Time{}
	}
	v := View{StateAt: stateAt, PerceptionAt: perceptionAt, Now: e.o.Now(), State: state, Inventory: e.o.Session.Inventory(), Perception: perception, Containers: e.o.Session.Containers(), Owners: e.o.Owners()}
	v.InjurySeq = e.o.Session.InjurySequence()
	if e.o.Danger != nil && v.State != nil {
		v.Danger = e.o.Danger(v.State.World, v.Now)
	}
	if e.o.Containers != nil && v.State != nil {
		// Live contents supersede the entire old observation, including an
		// empty chest. Never satisfy a condition from an older duplicate.
		live := make(map[string]bool, len(v.Containers))
		for _, c := range v.Containers {
			live[c.Entity] = true
		}
		for _, c := range e.o.Containers(v.State.World) {
			age := e.o.Now().Sub(c.At)
			if live[c.Entity] || c.At.IsZero() || age < 0 || c.ObservedEntity != "" && c.ObservedEntity != c.Entity {
				continue
			}
			v.Containers = append(v.Containers, &protocol.Container{Envelope: protocol.Envelope{T: v.State.T - age.Seconds()}, Entity: c.Entity, Pos: c.Pos, Size: c.Size, Items: c.Items})
		}
	}
	if e.o.Home != nil {
		if home, ok := e.o.Home(); ok {
			v.Home = &home
		}
	}
	v.Idle = v.State != nil && (v.State.Task == nil || protocol.Final(v.State.Task.Status))
	return v
}
func (e *Engine) triggers(v View, now time.Time) {
	events := e.drainEvents()
	if dropped := e.dropped.Swap(0); dropped > 0 {
		e.o.Logger.Warn("behaviour events dropped", "count", dropped)
	}
	full := v.Inventory != nil && (v.Inventory.Free() == 0 || v.State.Encumbered() || (v.State.WeightMax > 0 && v.State.Weight >= v.State.WeightMax))
	if full && !e.full {
		events["inventory_full"] = true
	}
	e.full = full
	if v.State.Time.Night && !e.night {
		events["nightfall"] = true
	}
	e.night = v.State.Time.Night
	atHome := v.Idle && v.Home != nil && distance(v.State.Pos, *v.Home) <= 3
	if atHome && !e.atHome {
		events["idle_at_home"] = true
	}
	e.atHome = atHome
	if v.Perception != nil && v.State.T-v.Perception.T >= 0 && v.State.T-v.Perception.T <= 5 {
		drops := map[string]bool{}
		for _, r := range v.Perception.Resources {
			if r.Kind == protocol.ResourceDrop {
				drops[r.ID] = true
				if !e.drops[r.ID] {
					events["drop_appeared"] = true
				}
			}
		}
		e.drops = drops
	}
	for name, i := range e.instances {
		d := i.definition
		if !d.Enabled {
			continue
		}
		triggered := d.Trigger.Kind == "event" && events[d.Trigger.Event]
		if d.Trigger.Kind == "timer" && !now.Before(i.timer) {
			triggered = true
			i.timer = now.Add(duration(d.Trigger.EveryS))
		}
		if triggered && !i.pending && e.active != name {
			i.pending = true
			e.o.Logger.Info("behaviour triggered", "instance", name, "trigger", d.Trigger.Kind, "event", d.Trigger.Event)
		}
	}
}
func (i *instance) allowed(v View) bool {
	d := i.definition
	if !d.Enabled || (d.Repeat.Mode == "once" && i.completed > 0) || (d.Repeat.MaxRuns > 0 && i.completed >= d.Repeat.MaxRuns) || !d.When.Match(v) {
		return false
	}
	if d.Repeat.Mode == "while" && !d.Repeat.Condition.Match(v) {
		return false
	}
	if d.Repeat.Mode == "until" && d.Repeat.Condition.Match(v) {
		return false
	}
	return true
}
func (e *Engine) candidate(v View, now time.Time) string {
	best := ""
	for name, i := range e.instances {
		if i.pending && !now.Before(i.next) && i.allowed(v) && (best == "" || i.definition.Priority > e.instances[best].definition.Priority || (i.definition.Priority == e.instances[best].definition.Priority && name < best)) {
			best = name
		}
	}
	return best
}

func (e *Engine) ended(i *instance, why string) {
	if life, ok := e.o.Templates[i.definition.Template].(Lifecycle); ok {
		if err := life.End(i.definition.Params, why); err != nil {
			e.o.Logger.Error("behaviour checkpoint failed", "err", err)
		}
	}
}
func (e *Engine) clearPending(why string) {
	for _, i := range e.instances {
		if i.pending {
			e.ended(i, why)
		}
		i.pending = false
	}
}

// cancel holds e.mu. A revoked lease must never send stop to the new owner.
func (e *Engine) cancel(ctx context.Context, why string, requeue bool) error {
	if e.active == "" {
		return nil
	}
	i := e.instances[e.active]
	if e.lease.Valid() && i.task != nil {
		select {
		case <-i.task.Done():
		default:
			if _, err := e.lease.Dispatch(ctx, session.Order{Task: protocol.TaskStop, TimeoutS: 5}); err != nil && !errors.Is(err, session.ErrBodyBusy) && !errors.Is(err, session.ErrNotConnected) {
				return err
			}
		}
	}
	e.o.Logger.Info("behaviour interrupted", "instance", e.active, "reason", why)
	e.ended(i, why)
	i.pending = requeue
	i.runner = nil
	i.task = nil
	i.last = why
	e.lease.Release()
	e.lease = nil
	e.active = ""
	return nil
}
func (e *Engine) finish(success bool, detail string, v View) {
	i := e.instances[e.active]
	e.ended(i, detail)
	i.completed++
	i.last = detail
	if success {
		i.storage = nil
		i.notice = ""
	}
	cooldown := duration(i.definition.Repeat.CooldownS)
	if cooldown < time.Second {
		cooldown = time.Second
	}
	i.next = e.o.Now().Add(cooldown)
	i.pending = success && (i.definition.Repeat.Mode == "while" || i.definition.Repeat.Mode == "until") && i.allowed(v)
	msg := "behaviour finished"
	if !success {
		msg = "behaviour failed"
	}
	e.o.Logger.Info(msg, "instance", e.active, "runs", i.completed, "detail", detail)
	i.runner = nil
	i.task = nil
	e.lease.Release()
	e.lease = nil
	e.active = ""
}

func (e *Engine) fail(ctx context.Context, err error, v View) {
	i := e.instances[e.active]
	var need *StorageNeed
	if errors.As(err, &need) {
		i.storage = need
		line := need.Error()
		if e.o.Notify != nil && !need.Quiet && (line != i.notice || e.o.Now().Sub(i.noticeAt) >= 5*time.Minute) {
			i.notice, i.noticeAt = line, e.o.Now()
			e.o.Notify(ctx, line)
		}
	}
	e.finish(false, err.Error(), v)
}
func (e *Engine) Tick(ctx context.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()
	hello, _, connected := e.o.Session.Handshake()
	v := e.view()
	now := e.o.Now()
	if !connected || v.State == nil || v.State.Phase != protocol.PhaseInWorld || v.State.HP <= 0 {
		_ = e.cancel(ctx, "unavailable", false)
		e.clearPending("unavailable")
		_ = e.drainEvents()
		return
	}
	if hello != e.conn || v.State.World != e.world {
		_ = e.cancel(ctx, "session/world changed", false)
		e.conn, e.world = hello, v.State.World
		e.full, e.night, e.atHome = false, false, false
		e.drops = nil
		for _, i := range e.instances {
			if i.requestConn != hello || i.requestWorld != e.world {
				if i.pending {
					e.ended(i, "session/world changed")
				}
				i.pending = false
			}
			i.timer = now.Add(duration(i.definition.Trigger.EveryS))
		}
		_ = e.drainEvents()
	}
	if e.active != "" && !e.lease.Valid() {
		e.orders = e.o.Session.ControlRevision()
		_ = e.cancel(ctx, "new order", false)
		e.clearPending("new order")
		return
	}
	e.triggers(v, now)
	if revision := e.o.Session.ControlRevision(); revision != e.orders {
		e.orders = revision
		_ = e.cancel(ctx, "new order", false)
		e.clearPending("new order")
		return
	}
	if e.active != "" && !e.instances[e.active].allowed(v) {
		if err := e.cancel(ctx, "condition changed", false); err != nil {
			e.o.Logger.Warn("could not stop behaviour", "err", err)
			return
		}
	}
	best := e.candidate(v, now)
	if e.active != "" && best != "" && e.instances[best].definition.Priority > e.instances[e.active].definition.Priority {
		if err := e.cancel(ctx, "higher priority: "+best, true); err != nil {
			e.o.Logger.Warn("could not preempt behaviour", "err", err)
			return
		}
	}
	if e.active == "" {
		best = e.candidate(v, now)
		if best == "" {
			return
		}
		lease, err := e.o.Session.TryAutomationFor(hello, v.State.World)
		if err != nil {
			return
		}
		i := e.instances[best]
		e.active, e.lease = best, lease
		i.pending = false
		i.step = 0
		i.result = nil
		i.waitingSince = time.Time{}
		raw := append(json.RawMessage(nil), i.definition.Params...)
		i.runner, err = e.o.Templates[i.definition.Template].Start(raw)
		e.o.Logger.Info("behaviour started", "instance", best, "template", i.definition.Template)
		if err != nil || i.runner == nil {
			e.finish(false, fmt.Sprintf("start: %v", err), v)
			return
		}
	}
	i := e.instances[e.active]
	if i.task != nil {
		select {
		case <-i.task.Done():
			status, reason, detail := i.task.Status()
			i.result = &Result{Status: status, Reason: reason, Detail: detail, TaskID: i.task.ID, Entity: i.task.Entity()}
			i.task = nil
			i.waitingSince = time.Time{}
			if status != protocol.StatusSucceeded {
				recovery, ok := i.runner.(FailureAware)
				if !ok || !recovery.Recover(*i.result) {
					e.finish(false, reason+": "+detail, v)
					return
				}
			}
			// Completion can arrive while view() is reading. Refresh after
			// Done closes so the preceding inventory/receipt frames are visible.
			v = e.view()
		default:
			if now.After(i.deadline) {
				name := e.active
				if err := e.cancel(ctx, "step timed out", false); err != nil {
					e.o.Logger.Warn("could not stop timed-out behaviour", "err", err)
				} else {
					i.completed++
					e.o.Logger.Info("behaviour failed", "instance", name, "runs", i.completed, "detail", "step timed out")
				}
			}
			return
		}
	}
	order, err := i.runner.Next(v, i.result)
	var delay *Delay
	if errors.As(err, &delay) {
		if i.waitingSince.IsZero() {
			i.waitingSince = now
		}
		if delay.Until.After(now.Add(30*time.Second)) || !delay.Until.After(now) {
			e.finish(false, "invalid or expired bounded wait", v)
		}
		return
	}
	if errors.Is(err, ErrWait) {
		if i.waitingSince.IsZero() {
			i.waitingSince = now
		}
		if now.Sub(i.waitingSince) >= 5*time.Second {
			e.finish(false, "fresh observation did not arrive within five seconds", v)
		}
		return
	}
	if err != nil {
		e.fail(ctx, err, v)
		return
	}
	i.waitingSince = time.Time{}
	if order == nil {
		e.finish(true, "completed", v)
		return
	}
	if order.Task == protocol.TaskStop || order.TimeoutS <= 0 || order.TimeoutS > 300 {
		e.finish(false, "template step needs a bounded timeout and cannot be stop", v)
		return
	}
	i.step++
	i.result = nil
	order.Origin = Origin{e.active, i.step}
	i.task, err = e.lease.Dispatch(ctx, *order)
	if err != nil {
		if !e.lease.Valid() {
			_ = e.cancel(ctx, "new order", false)
			e.clearPending("new order")
			return
		}
		e.finish(false, err.Error(), v)
		return
	}
	if started, ok := i.runner.(TaskStarted); ok {
		if err := started.Dispatched(i.task.ID); err != nil {
			_ = e.cancel(ctx, "dispatch checkpoint failed: "+err.Error(), false)
			return
		}
	}
	i.deadline = now.Add(duration(order.TimeoutS) + 5*time.Second)
	e.o.Logger.Info("behaviour step", "instance", e.active, "step", i.step, "task", order.Task, "id", i.task.ID)
}
