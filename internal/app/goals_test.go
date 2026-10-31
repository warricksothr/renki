package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/goal"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func goalFixture(t *testing.T) (*App, *bridgetest.Plugin, func(string, string, bool, bool) *httptest.ResponseRecorder, func(string, float64, bool)) {
	t.Helper()
	a := New(config.Default(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil)
	tasks := []string{"attack", "stop", "move_to"}
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome { return &protocol.Welcome{AllowedTasks: tasks} }}, a.Session)
	p := bridgetest.Connect(t, url, tasks)
	send := func(entity string, x float64, targeting bool) {
		p.Send(&protocol.State{Envelope: protocol.Envelope{T: 100}, World: "one", Phase: protocol.PhaseInWorld, HP: 80, HPMax: 80, Stamina: 60, Equipped: map[string]string{"right": "AxeFlint"}, Food: &protocol.FoodStatus{Active: []protocol.ActiveFood{{Prefab: "Raspberry"}}}})
		p.Send(&protocol.Inventory{})
		p.Send(&protocol.Perception{Envelope: protocol.Envelope{T: 100}, Characters: []protocol.PerceivedCharacter{{ID: entity, Kind: "creature", Prefab: "Greyling", HPFrac: 1, Hostile: true, Pos: protocol.Vec3{x, 0, 0}, TargetingMe: targeting}}})
		until := time.Now().Add(time.Second)
		for time.Now().Before(until) {
			seen := a.Session.Perception()
			if seen != nil && seen.Characters[0].ID == entity && seen.Characters[0].TargetingMe == targeting {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("no current observation")
	}
	send("first", 10, false)
	a.Behaviours.Tick(context.Background())
	h := panel.Handler(panel.Options{Token: "test", Status: func() any { return a.PanelStatus("test") }, Goals: a.GoalHandler()})
	call := func(method, body string, auth, csrf bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/api/goals", strings.NewReader(body))
		if auth {
			req.Header.Set("Authorization", "Bearer test")
		}
		if csrf {
			req.Header.Set(panel.ActHeader, "1")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	return a, p, call, send
}

const startGoal = `{"action":"start","spec":{"kind":"combat_trial","world":"one","center":[0,0,0],"radius":100}}`

func TestCombatGoalReturnsThroughTheEngineBeforeReportingTripSuccess(t *testing.T) {
	a, p, call, _ := goalFixture(t)
	home := protocol.Vec3{5, 0, 0}
	if _, err := a.Home.SetHome(home, "test"); err != nil {
		t.Fatal(err)
	}
	request := strings.Replace(startGoal, `"radius":100`, `"radius":100,"returnHome":true`, 1)
	g := createdGoal(t, call("POST", request, true, true))
	a.Behaviours.Tick(context.Background())
	fight := p.NextTask()
	if fight.Task != "attack" {
		t.Fatal(fight)
	}
	p.Send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: fight.ID}, Status: "succeeded", Detail: "observed Greyling death", Entity: "first"})
	g = tickUntilGoal(t, a, g.ID, func(g goal.Record) bool { return g.Proof != nil })
	if g.Status == "succeeded" {
		t.Fatal("kill alone completed the trip")
	}
	p.Send(&protocol.Perception{Envelope: protocol.Envelope{T: 100}})
	until := time.Now().Add(4 * time.Second)
	for time.Now().Before(until) {
		a.Behaviours.Tick(context.Background())
		g, _ = a.Goals.Get(g.ID)
		if g.Pending != nil && g.Pending.Template == "return_home" && g.Pending.TaskID != "" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if g.Pending == nil || g.Pending.Template != "return_home" {
		t.Fatal(g)
	}
	back := p.NextTask()
	var args protocol.MoveToArgs
	if err := json.Unmarshal(back.Args, &args); err != nil || back.Task != "move_to" || !args.Dry || args.To.Pos == nil || *args.To.Pos != home {
		t.Fatal(back, err)
	}
	p.Send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: back.ID}, Status: "succeeded"})
	g = tickUntilGoal(t, a, g.ID, func(g goal.Record) bool { return g.PostCombat != nil && g.PostCombat.Receipt != nil })
	if g.Status == "succeeded" {
		t.Fatal("old position completed the trip")
	}
	p.Send(&protocol.State{Envelope: protocol.Envelope{T: 101}, World: "one", Phase: protocol.PhaseInWorld, Pos: home, HP: 80, HPMax: 80, Stamina: 60})
	p.Send(&protocol.Perception{Envelope: protocol.Envelope{T: 101}})
	until = time.Now().Add(4 * time.Second)
	for time.Now().Before(until) {
		a.Behaviours.Tick(context.Background())
		g, _ = a.Goals.Get(g.ID)
		if g.Status == "succeeded" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if g.Status != "succeeded" || g.Proof.TaskID != fight.ID || g.PostCombat.Receipt.TaskID != back.ID || g.Attempts != 1 || g.TaskCount != 2 {
		t.Fatal(g)
	}
	p.Quiet(20 * time.Millisecond)
}

func createdGoal(t *testing.T, w *httptest.ResponseRecorder) goal.Record {
	t.Helper()
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var g goal.Record
	if e := json.Unmarshal(w.Body.Bytes(), &g); e != nil {
		t.Fatal(e)
	}
	return g
}
func tickUntilGoal(t *testing.T, a *App, id string, predicate func(goal.Record) bool) goal.Record {
	t.Helper()
	until := time.Now().Add(time.Second)
	for time.Now().Before(until) {
		a.Behaviours.Tick(context.Background())
		g, _ := a.Goals.Get(id)
		if predicate(g) {
			return g
		}
		time.Sleep(time.Millisecond)
	}
	g, _ := a.Goals.Get(id)
	t.Fatalf("goal did not transition: %+v", g)
	return g
}
func TestGoalAPIAuthenticationAndRecoveryAcrossNativeFailure(t *testing.T) {
	a, p, call, send := goalFixture(t)
	if w := call("GET", "", false, false); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := call("POST", startGoal, true, false); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if len(a.Goals.List()) != 0 {
		t.Fatal("unauthorized mutation")
	}
	g := createdGoal(t, call("POST", startGoal, true, true))
	a.Behaviours.Tick(context.Background())
	first := p.NextTask()
	if first.Task != "attack" {
		t.Fatal(first)
	}
	send("second", 3, true)
	p.Send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: first.ID}, Status: "failed", Reason: "unreachable", Detail: "travel limit"})
	g = tickUntilGoal(t, a, g.ID, func(g goal.Record) bool { return g.Attempts == 2 })
	second := p.NextTask()
	if second.ID == first.ID || second.Task != "attack" {
		t.Fatal(second)
	}
	var args protocol.AttackArgs
	_ = json.Unmarshal(second.Args, &args)
	if args.Kite || args.Entity != "second" {
		t.Fatal(args)
	}
	p.Send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: second.ID}, Status: "succeeded", Detail: "observed Greyling death"})
	g = tickUntilGoal(t, a, g.ID, func(g goal.Record) bool { return g.Status == "succeeded" })
	if len(g.History) != 2 || g.Proof.TaskID != second.ID {
		t.Fatal(g)
	}
	p.Quiet(20 * time.Millisecond)
	status := a.PanelStatus("test")
	if len(status.Goals) != 1 || status.Goals[0].Status != "succeeded" {
		t.Fatal(status.Goals)
	}
}
func TestOwnerOrderCancelsRunningAndQueuedGoalsWithoutStoppingNewOrder(t *testing.T) {
	for _, queued := range []bool{false, true} {
		t.Run(map[bool]string{true: "queued", false: "running"}[queued], func(t *testing.T) {
			a, p, call, _ := goalFixture(t)
			g := createdGoal(t, call("POST", startGoal, true, true))
			if !queued {
				a.Behaviours.Tick(context.Background())
				p.NextTask()
			}
			newer, e := a.Session.Dispatch(context.Background(), session.Order{Task: "move_to", Args: json.RawMessage(`{"to":[1,0,0]}`), TimeoutS: 10})
			if e != nil {
				t.Fatal(e)
			}
			if task := p.NextTask(); task.ID != newer.ID {
				t.Fatal(task)
			}
			g = tickUntilGoal(t, a, g.ID, func(g goal.Record) bool { return g.Status == "cancelled" })
			a.Behaviours.Tick(context.Background())
			p.Quiet(20 * time.Millisecond)
			if w := call("POST", `{"action":"resume","id":"`+g.ID+`"}`, true, true); w.Code != 400 {
				t.Fatal(w.Code)
			}
		})
	}
}

func TestDiscoveryGoalCompletesWithNativeBoundEntity(t *testing.T) {
	a, p, call, _ := goalFixture(t)
	p.Send(&protocol.Perception{Envelope: protocol.Envelope{T: 100}})
	until := time.Now().Add(time.Second)
	for len(a.Session.Perception().Characters) != 0 && time.Now().Before(until) {
		time.Sleep(time.Millisecond)
	}
	if len(a.Session.Perception().Characters) != 0 {
		t.Fatal("discovery perception not received")
	}
	g := createdGoal(t, call("POST", startGoal, true, true))
	a.Behaviours.Tick(context.Background())
	request := p.NextTask()
	var args protocol.AttackArgs
	if err := json.Unmarshal(request.Args, &args); err != nil || request.Task != "attack" || !args.Kite || args.Entity != "" {
		t.Fatal(request, err)
	}
	p.Send(&protocol.TaskStatus{Envelope: protocol.Envelope{Re: request.ID}, Status: "succeeded", Detail: "observed Greyling death", Entity: "native-grey"})
	g = tickUntilGoal(t, a, g.ID, func(g goal.Record) bool { return g.Status == "succeeded" })
	if g.Proof == nil || g.Proof.TaskID != request.ID || g.Proof.Entity != "native-grey" {
		t.Fatal(g)
	}
}
func TestGoalPermissionFailureAndOfflineStopRemainSafe(t *testing.T) {
	a, p, call, _ := goalFixture(t)
	g := createdGoal(t, call("POST", startGoal, true, true))
	a.Session.SetPermit(func(task string) bool { return task != "attack" })
	g = tickUntilGoal(t, a, g.ID, func(g goal.Record) bool { return g.Status == "needs_plan" })
	if g.Attempts != 1 {
		t.Fatal(g)
	}
	p.Quiet(20 * time.Millisecond)
	body, _ := json.Marshal(map[string]string{"action": "stop", "id": g.ID})
	w := call(http.MethodPost, string(body), true, true)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"status":"cancelled"`)) {
		t.Fatal(w.Body.String())
	}
}
