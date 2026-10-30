// Package behaviour schedules named, validated Go templates. Definitions
// contain policy and parameters; templates own the actual task sequence.
package behaviour

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

type Definition struct {
	Template string          `json:"template"`
	Params   json.RawMessage `json:"params"`
	Trigger  Trigger         `json:"trigger"`
	When     Conditions      `json:"when"`
	Repeat   Repeat          `json:"repeat"`
	Priority int             `json:"priority"`
	Enabled  bool            `json:"enabled"`
}
type Trigger struct {
	Kind   string  `json:"kind"` // manual, event, timer
	Event  string  `json:"event,omitempty"`
	EveryS float64 `json:"everyS,omitempty"`
}
type Repeat struct {
	Mode      string      `json:"mode"` // once, trigger, while, until
	CooldownS float64     `json:"cooldownS,omitempty"`
	MaxRuns   int         `json:"maxRuns,omitempty"`
	Condition *Conditions `json:"condition,omitempty"`
}
type Conditions struct {
	Items             map[string]int `json:"items,omitempty"`
	MinFreeSlots      int            `json:"minFreeSlots,omitempty"`
	MaxWeightFraction *float64       `json:"maxWeightFraction,omitempty"`
	Encumbered        *bool          `json:"encumbered,omitempty"`
	Stranded          *bool          `json:"stranded,omitempty"`
	Night             *bool          `json:"night,omitempty"`
	Owner             *OwnerNear     `json:"owner,omitempty"`
	Near              *Place         `json:"near,omitempty"`
	AtHomeRadius      *float64       `json:"atHomeRadius,omitempty"`
	Stock             []Stock        `json:"stock,omitempty"`
}
type OwnerNear struct {
	Name   string  `json:"name,omitempty"`
	Radius float64 `json:"radius"`
}
type Place struct {
	World  string        `json:"world"`
	Pos    protocol.Vec3 `json:"pos"`
	Radius float64       `json:"radius"`
}
type Stock struct {
	Container string  `json:"container"`
	Prefab    string  `json:"prefab"`
	MinCount  int     `json:"minCount"`
	MaxAgeS   float64 `json:"maxAgeS"`
}

// Result is the last completed step, or nil at the start of a run.
type Result struct{ Status, Reason, Detail, TaskID string }

// ErrWait keeps a runner alive for a fresh observation, for at most five
// seconds. It does not send a task or release body ownership.
var ErrWait = errors.New("waiting for an observation")

// FailureAware is opt-in. Ordinary templates still fail on every failed
// task; a bounded multi-chest runner can handle expected partial/refusal results.
type FailureAware interface{ Recover(Result) bool }

// Lifecycle receives every end, including cancellation of queued work.
// Implementations must be idempotent and must never dispatch game actions.
type Lifecycle interface {
	End(json.RawMessage, string) error
}

// TaskStarted checkpoints the assigned receipt identity after dispatch.
type TaskStarted interface{ Dispatched(string) error }

// Delay is an explicit bounded observation wait; it keeps body ownership.
type Delay struct{ Until time.Time }

func (d *Delay) Error() string { return "waiting until " + d.Until.String() }

type Runner interface {
	Next(View, *Result) (*session.Order, error)
}
type Template interface {
	Validate(json.RawMessage) error
	Start(json.RawMessage) (Runner, error)
}

// Decode rejects misspelled fields and trailing JSON before anything saves.
func Decode(raw []byte, to any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(to); err != nil {
		return err
	}
	if err := d.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("expected one JSON value")
	}
	return nil
}
func finite(n float64) bool  { return !math.IsNaN(n) && !math.IsInf(n, 0) }
func seconds(n float64) bool { return finite(n) && n >= 0 && n <= 86400 }
func (c Conditions) validate() error {
	if c.MinFreeSlots < 0 || c.MinFreeSlots > 10000 {
		return errors.New("invalid free-slot count")
	}
	for p, n := range c.Items {
		if strings.TrimSpace(p) == "" || n < 1 {
			return errors.New("items need a prefab and positive count")
		}
	}
	if c.MaxWeightFraction != nil && (!finite(*c.MaxWeightFraction) || *c.MaxWeightFraction < 0 || *c.MaxWeightFraction > 1) {
		return errors.New("weight fraction must be between zero and one")
	}
	radius := func(n float64) bool { return finite(n) && n > 0 && n <= 1000 }
	if c.Owner != nil && !radius(c.Owner.Radius) {
		return errors.New("invalid owner radius")
	}
	if c.AtHomeRadius != nil && !radius(*c.AtHomeRadius) {
		return errors.New("invalid home radius")
	}
	if c.Near != nil {
		if c.Near.World == "" || !radius(c.Near.Radius) {
			return errors.New("place needs world and radius")
		}
		for _, n := range c.Near.Pos {
			if !finite(n) {
				return errors.New("invalid place position")
			}
		}
	}
	for _, s := range c.Stock {
		if s.Container == "" || s.Prefab == "" || s.MinCount < 0 || !seconds(s.MaxAgeS) || s.MaxAgeS == 0 {
			return errors.New("stock needs container, prefab, count and positive maximum age")
		}
	}
	return nil
}

var eventNames = map[string]bool{
	"inventory_full": true, "drop_appeared": true, "idle_at_home": true, "nightfall": true, "stock_low": true,
	protocol.EventDamaged: true, protocol.EventLowHealth: true, protocol.EventItemAdded: true,
	protocol.EventPlayerJoined: true, protocol.EventPlayerLeft: true,
}

func validate(d Definition, templates map[string]Template) error {
	t := templates[d.Template]
	if t == nil {
		return fmt.Errorf("unknown template %q", d.Template)
	}
	if err := t.Validate(d.Params); err != nil {
		return fmt.Errorf("%s parameters: %w", d.Template, err)
	}
	switch d.Trigger.Kind {
	case "manual":
		if d.Trigger.Event != "" || d.Trigger.EveryS != 0 {
			return errors.New("manual trigger has event/timer fields")
		}
	case "event":
		if !eventNames[d.Trigger.Event] || d.Trigger.EveryS != 0 {
			return errors.New("invalid event trigger")
		}
	case "timer":
		if !seconds(d.Trigger.EveryS) || d.Trigger.EveryS < 1 || d.Trigger.Event != "" {
			return errors.New("timer interval must be 1–86400 seconds")
		}
	default:
		return errors.New("trigger kind must be manual, event or timer")
	}
	if err := d.When.validate(); err != nil {
		return err
	}
	switch d.Repeat.Mode {
	case "once", "trigger":
		if d.Repeat.Condition != nil {
			return errors.New("repeat condition only belongs to while/until")
		}
	case "while", "until":
		if d.Repeat.Condition == nil {
			return errors.New("while/until needs a condition")
		}
		if err := d.Repeat.Condition.validate(); err != nil {
			return err
		}
	default:
		return errors.New("repeat mode must be once, trigger, while or until")
	}
	if !seconds(d.Repeat.CooldownS) || d.Repeat.MaxRuns < 0 || d.Repeat.MaxRuns > 1000000 {
		return errors.New("invalid cooldown or run limit")
	}
	return nil
}

// MoveTo is a bounded movement template for the first engine check. Loot
// and stow register their own Go runners rather than scripts in settings.
type MoveTo struct{}

func (MoveTo) Validate(raw json.RawMessage) error {
	var a protocol.MoveToArgs
	if err := Decode(raw, &a); err != nil {
		return err
	}
	r, err := protocol.NewTask("validate", protocol.TaskMoveTo, &a, 60)
	if err != nil {
		return err
	}
	_, err = r.DecodeArgs()
	return err
}
func (MoveTo) Start(raw json.RawMessage) (Runner, error) {
	var a protocol.MoveToArgs
	if err := Decode(raw, &a); err != nil {
		return nil, err
	}
	return &moveRun{args: a}, nil
}

type moveRun struct {
	args protocol.MoveToArgs
	sent bool
}

func (r *moveRun) Next(_ View, _ *Result) (*session.Order, error) {
	if r.sent {
		return nil, nil
	}
	r.sent = true
	return &session.Order{Task: protocol.TaskMoveTo, Args: &r.args, TimeoutS: 60}, nil
}
