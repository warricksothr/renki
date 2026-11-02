package app

import (
	"encoding/json"
	"github.com/warricksothr/renki/internal/home"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/goal"
	"github.com/warricksothr/renki/internal/panel"
)

func TestIdlePersonalResupplyPresetPinsHomeAndHasChorePriority(t *testing.T) {
	a := New(config.Default(), nil, nil, nil, nil, nil)
	marker := home.Waypoint{Name: "house", Kind: "place", At: time.Now(), Radius: 12}
	if err := a.Memory.Store.EditWaypoint("one", "", &marker, nil); err != nil {
		t.Fatal(err)
	}
	call := func(enabled bool) int {
		body := `{"action":"idle-resupply","marker":{"world":"one","name":"house","kind":"place"},"enabled":false}`
		if enabled {
			body = strings.Replace(body, "false", "true", 1)
		}
		r := httptest.NewRequest("POST", "/api/behaviours", strings.NewReader(body))
		r.Header.Set(panel.ActHeader, "1")
		w := httptest.NewRecorder()
		a.BehaviourHandler().ServeHTTP(w, r)
		return w.Code
	}
	if call(true) != 400 {
		t.Fatal("resupply accepted unknown home")
	}
	if err := a.Memory.Store.SetHome("one", home.Home{}); err != nil {
		t.Fatal(err)
	}
	for _, enabled := range []bool{true, false, true} {
		if code := call(enabled); code != 200 {
			t.Fatal(code)
		}
		rows := a.Behaviours.List()
		if len(rows) != 1 {
			t.Fatal(rows)
		}
		d := rows[0].Definition
		var spec goal.Spec
		if err := json.Unmarshal(d.Params, &spec); err != nil || spec.World != "one" || spec.Reserve != 2 || spec.Stock != nil || spec.Radius != 150 ||
			d.Template != "personal_food" || d.Enabled != enabled || d.Priority <= 0 || d.When.Night != nil || d.Trigger.DwellS != 15 || d.Repeat.CooldownS != 300 {
			t.Fatal(d, spec, err)
		}
	}
}

func TestBehaviourControlsRequireAuthenticationAndValidateBeforeSaving(t *testing.T) {
	a := New(config.Default(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil)
	handler := panel.Handler(panel.Options{Token: "test", Status: func() any { return nil }, Behaviours: a.BehaviourHandler()})
	call := func(method, body string, auth, csrf bool) int {
		r := httptest.NewRequest(method, "/api/behaviours", strings.NewReader(body))
		if auth {
			r.Header.Set("Authorization", "Bearer test")
		}
		if csrf {
			r.Header.Set(panel.ActHeader, "1")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code
	}
	valid := `{"name":"trip","action":"save","definition":{"template":"move_to","params":{"to":[1,2,3]},"trigger":{"kind":"manual"},"repeat":{"mode":"once"},"enabled":false}}`
	if status := call(http.MethodPost, valid, false, true); status != http.StatusUnauthorized {
		t.Fatal(status)
	}
	if status := call(http.MethodPost, valid, true, false); status != http.StatusForbidden {
		t.Fatal(status)
	}
	if len(a.Settings.Instances()) != 0 {
		t.Fatal("unauthorized mutation")
	}
	if status := call(http.MethodPost, valid, true, true); status != http.StatusOK {
		t.Fatal(status)
	}
	if status := call(http.MethodGet, "", true, false); status != http.StatusOK {
		t.Fatal(status)
	}
	for _, bad := range []string{strings.Replace(valid, `"move_to"`, `"unknown"`, 1), strings.Replace(valid, `"params":`, `"typo":`, 1), valid + `{}`} {
		if status := call(http.MethodPost, bad, true, true); status != http.StatusBadRequest {
			t.Fatal(status)
		}
	}
	if len(a.Settings.Instances()) != 1 {
		t.Fatal(a.Settings.Instances())
	}
	for _, body := range []string{`{"name":"trip","action":"clone","target":"copy"}`, `{"name":"copy","action":"enable"}`, `{"name":"copy","action":"disable"}`, `{"name":"copy","action":"delete"}`} {
		if status := call(http.MethodPost, body, true, true); status != http.StatusOK {
			t.Fatal(status, body)
		}
	}
}

func TestIdleFoodPresetUsesCurrentMarkerAndReplacesOneDefinition(t *testing.T) {
	a := New(config.Default(), nil, nil, nil, nil, nil)
	marker := home.Waypoint{Name: "pantry", Kind: "storage", At: time.Now(), Radius: 6}
	if err := a.Memory.Store.EditWaypoint("one", "", &marker, nil); err != nil {
		t.Fatal(err)
	}
	call := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/behaviours", strings.NewReader(body))
		r.Header.Set(panel.ActHeader, "1")
		w := httptest.NewRecorder()
		a.BehaviourHandler().ServeHTTP(w, r)
		return w
	}
	for _, enabled := range []string{"true", "false", "true"} {
		w := call(`{"action":"idle-food","marker":{"world":"one","name":"pantry","kind":"storage"},"enabled":` + enabled + `}`)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		rows := a.Behaviours.List()
		if len(rows) != 1 || rows[0].Definition.Enabled != (enabled == "true") || rows[0].Definition.Trigger.DwellS != 30 || rows[0].Definition.Repeat.CooldownS != 300 {
			t.Fatal(rows)
		}
	}
	for _, bad := range []string{
		`{"action":"idle-food","marker":{"world":"two","name":"pantry","kind":"storage"},"enabled":true}`,
		`{"action":"idle-food","marker":{"world":"one","name":"pantry","kind":"cooking"},"enabled":true}`,
		`{"action":"idle-food","marker":{"world":"one","name":"pantry","kind":"storage"}}`,
	} {
		if w := call(bad); w.Code != 400 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}

func TestIdleRestPresetsKeepNightAndDaySeparate(t *testing.T) {
	a := New(config.Default(), nil, nil, nil, nil, nil)
	marker := home.Waypoint{Name: "house", Kind: "place", At: time.Now(), Radius: 10}
	if err := a.Memory.Store.EditWaypoint("one", "", &marker, nil); err != nil {
		t.Fatal(err)
	}
	call := func(body string) int {
		r := httptest.NewRequest("POST", "/api/behaviours", strings.NewReader(body))
		r.Header.Set(panel.ActHeader, "1")
		w := httptest.NewRecorder()
		a.BehaviourHandler().ServeHTTP(w, r)
		return w.Code
	}
	body := `{"action":"idle-rest","marker":{"world":"one","name":"house","kind":"place"},"enabled":true}`
	if call(body) != 400 {
		t.Fatal("rest mode must be explicit")
	}
	for _, mode := range []string{"true", "false", "true"} {
		if got := call(strings.TrimSuffix(body, "}") + `,"sleep":` + mode + `}`); got != 200 {
			t.Fatal(got, mode)
		}
	}
	rows := a.Behaviours.List()
	if len(rows) != 2 {
		t.Fatal(rows)
	}
	for _, row := range rows {
		d := row.Definition
		if d.Template != "marker_rest" || d.When.Night == nil || d.Repeat.CooldownS != 300 {
			t.Fatal(d)
		}
		if *d.When.Night && d.Trigger.DwellS != 15 || !*d.When.Night && d.Trigger.DwellS != 120 {
			t.Fatal(d)
		}
	}
	if call(strings.Replace(strings.TrimSuffix(body, "}")+`,"sleep":true}`, `"name":"house"`, `"name":"removed"`, 1)) != 400 {
		t.Fatal("missing marker admitted")
	}
}
