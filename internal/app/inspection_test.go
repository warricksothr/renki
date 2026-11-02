package app

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/warricksothr/renki/internal/inspection"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func TestInspectionHTTPAuthenticationReadOnlyAndImmediateOwnerTakeover(t *testing.T) {
	a, p, goals, _ := goalFixture(t)
	g := createdGoal(t, goals("POST", startGoal, true, true))
	a.Behaviours.Tick(context.Background())
	task := p.NextTask()
	h := panel.Handler(panel.Options{Token: "test", Status: func() any { return nil }, Inspection: a.InspectionHandler()})
	request := func(method, path string, auth bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		if auth {
			r.Header.Set("Authorization", "Bearer test")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if request("GET", "/api/inspection", false).Code != 401 {
		t.Fatal("inspection without authentication")
	}
	if request("POST", "/api/inspection", true).Code != 405 {
		t.Fatal("inspection accepted write")
	}
	if request("GET", "/api/inspection?goal=missing", true).Code != 404 {
		t.Fatal("unknown goal accepted")
	}
	if request("GET", "/api/inspection?goal="+strings.Repeat("x", 97), true).Code != 400 {
		t.Fatal("unbounded goal selector accepted")
	}
	before, _ := a.Goals.Get(g.ID)
	beforeJSON, _ := json.Marshal(before)
	revision := a.Session.ControlRevision()
	w := request("GET", "/api/inspection", true)
	var out inspection.Snapshot
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || out.Goal.ID != g.ID || out.Body.TaskID != task.ID || !out.Body.RelatedGoal {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	after, _ := a.Goals.Get(g.ID)
	afterJSON, _ := json.Marshal(after)
	if string(beforeJSON) != string(afterJSON) || revision != a.Session.ControlRevision() {
		t.Fatal("inspection changed control/checkpoint")
	}
	_, err := a.Session.Dispatch(context.Background(), session.Order{Task: "move_to", Origin: "PRIVATE CHAT", Args: protocol.MoveToArgs{To: protocol.Target{Pos: &protocol.Vec3{5, 0, 0}}}})
	if err != nil {
		t.Fatal(err)
	}
	next := p.NextTask()
	w = request("GET", "/api/inspection?goal="+g.ID, true)
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || out.Body.RelatedGoal || out.Body.TaskID != next.ID || out.Body.Status != "awaiting_state" || strings.Contains(w.Body.String(), "PRIVATE CHAT") {
		t.Fatal(w.Body.String(), err)
	}
}
