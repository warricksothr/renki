package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "c.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDefaults(t *testing.T) {
	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen != "127.0.0.1:8765" || c.Rates.StateHz != 5 || !slices.Equal(c.AllowedTasks, KnownTasks) {
		t.Errorf("defaults %+v", c)
	}
}

func TestFileOverridesDefaults(t *testing.T) {
	c, err := Load(write(t, `{"owners":["Sothr"],"agentName":"Ylva","allowedTasks":["stop","say"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if c.AgentName != "Ylva" || !c.IsOwner("Sothr") || c.IsOwner("sothr") || len(c.AllowedTasks) != 2 || c.Listen == "" {
		t.Errorf("loaded %+v", c)
	}
}

func TestLANNeedsToken(t *testing.T) {
	_, err := Load(write(t, `{"listen":"0.0.0.0:8765"}`))
	if err == nil || !strings.Contains(err.Error(), "token") {
		t.Fatalf("err %v; want a token error", err)
	}
	t.Setenv("RENKI_TOKEN", "from-env")
	c, err := Load(write(t, `{"listen":"0.0.0.0:8765"}`))
	if err != nil || c.Token != "from-env" {
		t.Fatalf("with RENKI_TOKEN: %+v, %v", c, err)
	}
}

func TestUnknownTaskRefused(t *testing.T) {
	if _, err := Load(write(t, `{"allowedTasks":["stop","teleport"]}`)); err == nil {
		t.Fatal("want an error for a task the controller does not know")
	}
}

func TestPanelNeedsALongToken(t *testing.T) {
	c := Default()
	c.Panel.Listen = "100.64.0.1:8780"
	if c.Validate() == nil {
		t.Error("a panel with no token was accepted")
	}
	c.Panel.Token = "short"
	if c.Validate() == nil {
		t.Error("a short panel token was accepted")
	}
	c.Panel.Token = "0123456789abcdef"
	if err := c.Validate(); err != nil {
		t.Error(err)
	}
}

func TestAfterDeathMustBeKnown(t *testing.T) {
	c := Default()
	if c.Home.AfterDeath != AfterDeathHome {
		t.Errorf("default %q", c.Home.AfterDeath)
	}
	c.Home.AfterDeath = "dance"
	if c.Validate() == nil {
		t.Error("afterDeath dance was accepted")
	}
}

func TestWarnsWhenOverloadCannotDrop(t *testing.T) {
	c := Default()
	if w := c.Warnings(); len(w) != 0 {
		t.Errorf("defaults: %q", w)
	}
	c.AllowedTasks = []string{"stop", "follow"}
	if w := c.Warnings(); len(w) != 1 {
		t.Errorf("without drop: %q", w)
	}
	c.Overload.Drop = false
	if w := c.Warnings(); len(w) != 0 {
		t.Errorf("with dropping off: %q", w)
	}
}
