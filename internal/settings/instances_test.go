package settings

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstancesSurviveRestartWithSettingsAndCannotBeMutatedByReaders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := open(t, path)
	var audit bytes.Buffer
	s.log = slog.New(slog.NewJSONHandler(&audit, nil))
	definition := json.RawMessage(`{"template":"stow","enabled":true,"params":{"to":["1:2"]}}`)
	if err := s.SetInstance("deliver when full", definition, "owner:Nimah"); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("voice", json.RawMessage(`false`), ByPanel); err != nil {
		t.Fatal(err)
	}
	definition[0] = '['
	entries := s.Instances()
	entry := entries["deliver when full"]
	entry.Value[0] = '['
	delete(entries, "deliver when full")
	reopened := open(t, path)
	saved := reopened.Instances()["deliver when full"]
	if !json.Valid(saved.Value) || saved.Value[0] != '{' || saved.By != "owner:Nimah" || saved.At.IsZero() || reopened.Values().Voice {
		t.Fatalf("restart lost settings or instance provenance: %+v", saved)
	}
	if s.Instances()["deliver when full"].Value[0] != '{' {
		t.Fatal("reader mutated stored definition")
	}
	if !strings.Contains(audit.String(), `"msg":"instance changed"`) || !strings.Contains(audit.String(), `"by":"owner:Nimah"`) {
		t.Fatalf("missing audit: %s", audit.String())
	}
	if err := reopened.DeleteInstance("deliver when full", ByPanel); err != nil {
		t.Fatal(err)
	}
	if len(open(t, path).Instances()) != 0 {
		t.Fatal("deleted instance came back after restart")
	}
}

func TestInvalidInstanceAndFailedSaveLeavePreviousDefinition(t *testing.T) {
	s := open(t, "")
	good := json.RawMessage(`{"template":"loot"}`)
	if err := s.SetInstance("loot", good, ByPanel); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`null`, `[]`, `"value"`, `{`} {
		if err := s.SetInstance("loot", json.RawMessage(raw), ByPanel); !errors.Is(err, ErrBadValue) {
			t.Fatalf("accepted %s: %v", raw, err)
		}
	}
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	s.path = filepath.Join(blocker, "settings.json")
	if err := s.SetInstance("loot", json.RawMessage(`{"template":"stow"}`), ByPanel); err == nil {
		t.Fatal("save unexpectedly succeeded")
	}
	if err := s.DeleteInstance("loot", ByPanel); err == nil {
		t.Fatal("delete unexpectedly succeeded")
	}
	if string(s.Instances()["loot"].Value) != string(good) {
		t.Fatal("failed persistence changed live definition")
	}
}
