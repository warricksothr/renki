package behaviour

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// Travel takes the body to a position over a navigator's plan: the trip an
// owner asks for with "go to" when the way is known. Each leg is an ordinary
// move (route_walk.go). It is for known ground only; StartTravel refuses a
// trip with no plan, and the caller then sends the one direct move it always
// did, which is how new ground is learnt.
type Travel struct {
	Routes Navigator
}

type TravelParams struct {
	World string        `json:"world"`
	To    protocol.Vec3 `json:"to"`
	// Place names the destination in reports.
	Place string `json:"place,omitempty"`
	// StopM is how near to get. Zero means 1 m.
	StopM float64 `json:"stopM,omitempty"`
	Run   bool    `json:"run,omitempty"`
}

// A trip is given this long for each metre planned, and never less than a
// minute: a slow walk with a pause at every leg end.
const travelSecondsPerM = 3

func (p *TravelParams) defaults() {
	if p.StopM == 0 {
		p.StopM = 1
	}
}

func (Travel) Validate(raw json.RawMessage) error {
	var p TravelParams
	if err := Decode(raw, &p); err != nil {
		return err
	}
	p.defaults()
	if strings.TrimSpace(p.World) == "" || !finite(p.StopM) || p.StopM < .2 || p.StopM > 10 || len([]rune(p.Place)) > 64 {
		return errors.New("travel needs a world, a stop distance of 0.2–10m and a place name of at most 64 characters")
	}
	for _, n := range p.To {
		if !finite(n) || math.Abs(n) > 1e6 {
			return errors.New("invalid travel destination")
		}
	}
	return nil
}

func (t Travel) Start(raw json.RawMessage) (Runner, error) {
	if err := t.Validate(raw); err != nil {
		return nil, err
	}
	var p TravelParams
	_ = Decode(raw, &p)
	p.defaults()
	return &travelRun{p: p, routes: t.Routes}, nil
}

type travelRun struct {
	p        TravelParams
	routes   Navigator
	walk     *routeWalk
	deadline time.Time
	taskID   string
	told     bool
}

func (r *travelRun) name() string {
	if r.p.Place != "" {
		return r.p.Place
	}
	return fmt.Sprintf("[%.0f, %.0f, %.0f]", r.p.To[0], r.p.To[1], r.p.To[2])
}

// Recover keeps the trip going after a leg fails: the walk plans again.
func (r *travelRun) Recover(result Result) bool { return r.walk.answers(result) }
func (r *travelRun) Dispatched(id string) error { r.taskID = id; return nil }

func (r *travelRun) check(v View) error {
	if !idleFresh(v) || v.State.World != r.p.World {
		return errors.New("travel needs fresh state in the requested world")
	}
	for _, n := range v.State.Pos {
		if !finite(n) || math.Abs(n) > 1e6 {
			return errors.New("invalid body position")
		}
	}
	return nil
}

func (r *travelRun) Next(v View, result *Result) (*session.Order, error) {
	if err := r.check(v); err != nil {
		return nil, err
	}
	if r.told {
		return nil, nil
	}
	if r.walk == nil {
		style := MoveStyle{}
		if r.p.Run {
			run := true
			style.Run = &run
		}
		r.walk = newRouteWalk(r.routes, r.p.World, v.State.Pos, r.p.To, r.p.StopM, style)
		r.deadline = v.Now.Add(time.Duration(math.Max(60, travelSecondsPerM*r.walk.lengthM(v.State.Pos)) * float64(time.Second)))
		return r.walk.first(), nil
	}
	if result == nil || result.TaskID != r.taskID {
		return nil, errors.New("missing matching travel outcome")
	}
	next := r.walk.step(v, result)
	if next != nil && v.Now.Before(r.deadline) {
		return next, nil
	}
	if next != nil {
		r.walk.abandon("out of time")
	}
	if r.walk.arrived {
		return nil, nil
	}
	// Say why once, in the game, where the owner who asked is.
	r.told = true
	line := "I couldn't get to " + r.name() + ": " + r.walk.story() + "."
	if len([]rune(line)) > 200 {
		line = string([]rune(line)[:197]) + "..."
	}
	return &session.Order{Task: protocol.TaskSay, Args: &protocol.SayArgs{Text: line}, TimeoutS: 5}, nil
}

func (r *travelRun) InterruptedSummary() string { return r.Summary() }

func (r *travelRun) Summary() string {
	if r.walk == nil {
		return "Travel to " + r.name() + ": not started."
	}
	return "Travel to " + r.name() + ": " + r.walk.story() + "."
}

const travelInstance = "Owner travel"
const travelSource = "owner:travel:"

// ErrNoKnownRoute is StartTravel's answer when the navigator has no plan
// worth walking: none at all, or a single leg an ordinary move makes as well.
var ErrNoKnownRoute = errors.New("no known route")

// StartTravel starts a trip for an owner over known ground. It returns
// ErrNoKnownRoute when there is no plan of at least two legs, and another
// error when the body is busy; nothing was started and the caller may send
// its direct move instead. ErrReplaced is the exception: a newer order came
// in while the trip was starting, and nothing may be sent over it.
func (e *Engine) StartTravel(ctx context.Context, by string, request TravelParams) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.o.Routes == nil {
		return "", ErrNoKnownRoute
	}
	var story string
	err := e.startOwnerRun(ctx, by, travelInstance, travelSource, "travel", "trip", func(v View) (json.RawMessage, error) {
		if request.World == "" {
			request.World = v.State.World
		}
		request.defaults()
		raw, _ := json.Marshal(request)
		started, err := e.o.Templates["travel"].Start(raw)
		if err != nil {
			return nil, err
		}
		if err = started.(*travelRun).check(v); err != nil {
			return nil, err
		}
		plan := e.o.Routes.Plan(request.World, v.State.Pos, request.To)
		if !plan.Found || len(plan.Legs) < 2 {
			return nil, ErrNoKnownRoute
		}
		story = fmt.Sprintf("%d legs, %.0f m", len(plan.Legs), plan.LengthM)
		return raw, nil
	})
	if err != nil {
		return "", err
	}
	return story, nil
}
