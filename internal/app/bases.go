package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/atlas"
	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/goal"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
)

func (a *App) baseSnapshot(world string) any {
	bases, active := a.Memory.Store.Bases(world)
	var surveys []goal.Record
	for _, g := range a.Goals.Summaries() {
		if g.World == world && g.Survey != nil && (active == "" || g.Survey.Pinned.Base.Name == active) {
			surveys = append(surveys, g)
		}
	}
	return struct {
		World   string        `json:"world"`
		Bases   []home.Base   `json:"bases"`
		Active  string        `json:"active"`
		Surveys []goal.Record `json:"surveys"`
	}{world, bases, active, surveys}
}
func (a *App) BaseHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		world := r.URL.Query().Get("world")
		if world == "" {
			world = a.Home.World()
		}
		if r.Method == http.MethodGet {
			writeJSON(w, 200, a.baseSnapshot(world))
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		if r.Header.Get(panel.ActHeader) == "" {
			http.Error(w, "missing "+panel.ActHeader, 403)
			return
		}
		var req struct {
			Action     string     `json:"action"`
			World      string     `json:"world"`
			Name       string     `json:"name"`
			Base       *home.Base `json:"base,omitempty"`
			ExpectedAt *time.Time `json:"expectedAt,omitempty"`
			ID         string     `json:"id,omitempty"`
		}
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 32<<10))
		if err == nil {
			err = behaviour.Decode(raw, &req)
		}
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if req.World == "" {
			req.World = world
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		switch req.Action {
		case "save":
			if req.Base == nil {
				err = errors.New("base settings required")
				break
			}
			req.Base.At = time.Now().UTC()
			req.Base.SetBy = "panel"
			err = a.Memory.Store.EditBase(req.World, req.Name, req.Base, req.ExpectedAt, false)
		case "select", "delete":
			err = a.Memory.Store.EditBase(req.World, req.Name, nil, req.ExpectedAt, req.Action == "delete")
		case "start":
			if req.World != a.Home.World() {
				err = errors.New("survey requires the current world")
				break
			}
			_, err = a.startSurvey(ctx, "panel")
		case "stop":
			a.goalMu.Lock()
			err = a.goalAction(ctx, req.ID, "stop")
			a.goalMu.Unlock()
		default:
			err = errors.New("base action must be save, select, delete, start or stop")
		}
		if err != nil {
			code := 400
			if errors.Is(err, home.ErrWaypointConflict) {
				code = 409
			}
			http.Error(w, err.Error(), code)
			return
		}
		writeJSON(w, 200, a.baseSnapshot(req.World))
	})
}

func (a *App) startSurvey(ctx context.Context, by string) (string, error) {
	a.goalMu.Lock()
	defer a.goalMu.Unlock()
	values := a.Settings.Values()
	if by != "panel" && !slices.Contains(values.Players.Commanders(values.Owners), by) {
		return "", errors.New("survey requires an owner or operator")
	}
	state, at := a.Session.StateObservation()
	revision := a.Session.ControlRevision()
	if state == nil || state.Phase != protocol.PhaseInWorld || time.Since(at) < 0 || time.Since(at) > 3*time.Second || state.Task != nil && !protocol.Final(state.Task.Status) {
		return "", errors.New("fresh idle body required; stop current work first")
	}
	for _, i := range a.Behaviours.List() {
		if i.Pending || i.Running {
			return "", errors.New("another behavior has queued or owns the body")
		}
	}
	for _, task := range []string{protocol.TaskMoveTo, protocol.TaskStop} {
		if !a.Session.Allowed(task) {
			return "", fmt.Errorf("%s permission required", task)
		}
	}
	b := a.Memory.Store.ActiveBase(state.World)
	if b == nil {
		return "", errors.New("configure and select a base area first in Places & storage")
	}
	var h *protocol.Vec3
	if saved, ok := a.Memory.Store.Home(state.World); ok {
		p := saved.Pos
		h = &p
	}
	pinned, err := home.ResolveBase(*b, a.Memory.Store.Waypoints(state.World), h)
	if err != nil {
		return "", err
	}
	world, conn, scope, _ := a.Session.TerrainView()
	if scope == nil || world != state.World {
		return "", errors.New("fresh identified terrain observations required")
	}
	var injury *uint64
	if hit := a.Session.InjuryStatus(); hit != nil {
		n := hit.Sequence
		injury = &n
	}
	g, err := goal.NewSurvey(world, atlas.Key(*scope), scope.Source, conn, pinned, state.Pos, state.HP, injury, time.Now().UTC())
	if err != nil {
		return "", err
	}
	if err = a.goalTemplate.SurveyReady(g); err != nil {
		return "", err
	}
	if a.Session.ControlRevision() != revision {
		return "", errors.New("a newer order replaced the survey request")
	}
	if err = a.Goals.Add(g); err != nil {
		return "", err
	}
	if err = a.goalAction(ctx, g.ID, "start"); err != nil {
		_ = a.goalTemplate.Cancel(g.ID)
		return "", err
	}
	if a.Session.ControlRevision() != revision {
		_ = a.goalAction(ctx, g.ID, "stop")
		return "", errors.New("a newer order replaced the survey request")
	}
	return fmt.Sprintf("I'll familiarize myself with %s within its %.0f m radius (%.0f m across), for at most %.0f minutes and %.0f m of travel, including return. Say stop to cancel.", b.Name, b.Radius, b.Radius*2, b.DurationS/60, b.TravelM), nil
}

func (a *App) baseChat(ctx context.Context, by, text string, pos *protocol.Vec3) (string, bool, error) {
	text = strings.TrimSpace(text)
	if text == "survey status" {
		base := a.Memory.Store.ActiveBase(a.Home.World())
		for _, g := range a.Goals.List() {
			if g.Kind == "familiarize" && g.World == a.Home.World() && (base == nil || g.Survey != nil && g.Survey.Pinned.Base.Name == base.Name) {
				return goal.SurveySummary(g), true, nil
			}
		}
		return "I haven't surveyed this base yet.", true, nil
	}
	if text == "familiarize yourself with home" || text == "familiarize yourself with this area" || text == "familiarize yourself with the area" || text == "survey home" {
		line, err := a.startSurvey(ctx, by)
		return line, true, err
	}
	if strings.HasPrefix(text, "base radius ") {
		radius, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(text, "base radius "), "m")), 64)
		if err != nil {
			return "", true, errors.New("use base radius followed by metres")
		}
		b := a.Memory.Store.ActiveBase(a.Home.World())
		if b == nil {
			return "", true, errors.New("configure a base first")
		}
		expected := b.At
		b.Radius = radius
		b.At = time.Now().UTC()
		b.SetBy = by
		err = a.Memory.Store.EditBase(a.Home.World(), b.Name, b, &expected, false)
		return fmt.Sprintf("Base radius is %.0f m (%.0f m across).", radius, 2*radius), true, err
	}
	if strings.HasPrefix(text, "base sleeping ") || strings.HasPrefix(text, "base storage ") || strings.HasPrefix(text, "base return ") {
		parts := strings.SplitN(text, " ", 3)
		name, err := home.WaypointName(parts[2])
		if parts[2] == "home" && parts[1] == "return" {
			name = "home"
			err = nil
		}
		if err != nil {
			return "", true, err
		}
		b := a.Memory.Store.ActiveBase(a.Home.World())
		if b == nil {
			return "", true, errors.New("configure a base first")
		}
		expected := b.At
		switch parts[1] {
		case "sleeping":
			b.Sleeping = name
		case "storage":
			b.Storage = []string{name}
		case "return":
			b.Return = name
		}
		b.At = time.Now().UTC()
		b.SetBy = by
		err = a.Memory.Store.EditBase(a.Home.World(), b.Name, b, &expected, false)
		return "Bound " + parts[1] + " to " + name + ".", true, err
	}
	return "", false, nil
}
