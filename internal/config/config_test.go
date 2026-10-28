package config

import (
	"os"
	"path/filepath"
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
	if c.Listen != "127.0.0.1:8765" || c.Rates.StateHz != 5 || len(c.AllowedTasks) != 6 {
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
	if _, err := Load(write(t, `{"allowedTasks":["stop","attack"]}`)); err == nil {
		t.Fatal("want an error for a task the controller does not know")
	}
}
