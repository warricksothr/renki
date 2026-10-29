package settings

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/config"
)

func base() config.Config {
	c := config.Default()
	c.Owners = []string{"Sothr"}
	c.LLM = &config.LLM{}
	return c
}

func open(t *testing.T, path string) *Store {
	t.Helper()
	at := time.Date(2026, 10, 29, 1, 0, 0, 0, time.UTC)
	s, err := Open(path, base(), slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return at })
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestValuesStartFromTheConfig(t *testing.T) {
	v := open(t, "").Values()
	if !slices.Equal(v.Owners, []string{"Sothr"}) || !v.Voice || v.Home.IdleMinutes != 10 || !v.Overload.Drop {
		t.Errorf("values %+v", v)
	}
}

func TestChangesSurviveARestartAndResetReturnsToTheConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "settings.json")
	s := open(t, path)
	var seen []Values
	s.OnChange(func(v Values) { seen = append(seen, v) })
	if err := s.Set("home.idleMinutes", json.RawMessage(`25`), "owner:Sothr"); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("owners", json.RawMessage(`["Sothr", "Nimah"]`), ByPanel); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 2 || seen[1].Home.IdleMinutes != 25 {
		t.Fatalf("OnChange saw %+v", seen)
	}

	s = open(t, path)
	v := s.Values()
	if v.Home.IdleMinutes != 25 || !slices.Equal(v.Owners, []string{"Sothr", "Nimah"}) || !v.Home.WhenLost {
		t.Fatalf("after a restart %+v", v)
	}
	e := s.Entries()[slices.IndexFunc(s.Entries(), func(e Entry) bool { return e.Key == "home.idleMinutes" })]
	if string(e.Value) != "25" || string(e.Config) != "10" || e.Set == nil || e.Set.By != "owner:Sothr" {
		t.Errorf("entry %+v", e)
	}
	if err := s.Reset("home.idleMinutes", ByPanel); err != nil {
		t.Fatal(err)
	}
	if s.Values().Home.IdleMinutes != 10 {
		t.Error("reset did not return to the config")
	}
}

func TestBadChangesAreRefused(t *testing.T) {
	s := open(t, "")
	for _, c := range []struct {
		key, value string
		want       error
	}{
		{"listen", `"0.0.0.0:1"`, ErrUnknownKey},
		{"home.idleMinutes", `-1`, ErrBadValue},
		{"home.afterDeath", `"dance"`, ErrBadValue},
		{"allowedTasks", `["stop", "fly"]`, ErrBadValue},
		{"voice", `"yes"`, ErrBadValue},
		{"home.idleMinutes", `{`, ErrBadValue},
		{"home.idleMinutes", `null`, ErrBadValue},
		{"overload.drop", ` null `, ErrBadValue},
	} {
		if err := s.Set(c.key, json.RawMessage(c.value), ByPanel); !errors.Is(err, c.want) {
			t.Errorf("%s = %s: %v, want %v", c.key, c.value, err, c.want)
		}
	}
	if v := s.Values(); v.Home.IdleMinutes != 10 || v.Home.AfterDeath != config.AfterDeathHome {
		t.Errorf("a refused change stuck: %+v", v)
	}
}

func TestPinsHoldOffTheAgentOnly(t *testing.T) {
	s := open(t, "")
	if err := s.Pin("overload.drop", ByAgent); !errors.Is(err, ErrNotAllowed) {
		t.Errorf("the agent pinned: %v", err)
	}
	if err := s.Pin("overload.drop", "owner:Sothr"); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("overload.drop", json.RawMessage(`false`), ByAgent); !errors.Is(err, ErrPinned) {
		t.Errorf("the agent changed a pinned setting: %v", err)
	}
	if err := s.Reset("overload.drop", ByAgent); !errors.Is(err, ErrPinned) {
		t.Errorf("the agent reset a pinned setting: %v", err)
	}
	if err := s.Set("overload.drop", json.RawMessage(`false`), ByPanel); err != nil || s.Values().Overload.Drop {
		t.Errorf("the panel could not change it: %v", err)
	}
	if err := s.Unpin("overload.drop", ByPanel); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("overload.drop", json.RawMessage(`true`), ByAgent); err != nil {
		t.Errorf("unpinned: %v", err)
	}
}

func TestTheAgentNeverChangesWhoGivesOrders(t *testing.T) {
	s := open(t, "")
	for _, k := range []string{"owners", "allowedTasks"} {
		if err := s.Set(k, json.RawMessage(`[]`), ByAgent); !errors.Is(err, ErrNotAllowed) {
			t.Errorf("the agent set %s: %v", k, err)
		}
		if err := s.Reset(k, ByAgent); !errors.Is(err, ErrNotAllowed) {
			t.Errorf("the agent reset %s: %v", k, err)
		}
	}
}

func TestWarningsReadTheSettingsInForce(t *testing.T) {
	s := open(t, "")
	if w := s.Warnings(); len(w) != 0 {
		t.Fatalf("warnings %q", w)
	}
	s.Set("allowedTasks", json.RawMessage(`["stop"]`), ByPanel)
	if w := s.Warnings(); len(w) != 1 {
		t.Errorf("without drop: %q", w)
	}
}

func TestSavedChangesTheConfigNoLongerAcceptsAreDropped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte(`{"overrides": {
		"home.idleMinutes": {"value": 5, "by": "panel"},
		"home.afterDeath": {"value": "dance", "by": "panel"},
		"gone": {"value": 1, "by": "panel"}}}`), 0o644)
	v := open(t, path).Values()
	if v.Home.IdleMinutes != 5 || v.Home.AfterDeath != config.AfterDeathHome {
		t.Errorf("values %+v", v)
	}
}
