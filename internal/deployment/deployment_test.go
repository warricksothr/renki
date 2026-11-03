package deployment

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warricksothr/renki/internal/atlas"
	"github.com/warricksothr/renki/internal/config"
)

const source = "1111111111111111111111111111111111111111"
const parent = "2222222222222222222222222222222222222222"

func write(t *testing.T, path string, v any) {
	t.Helper()
	b, e := json.Marshal(v)
	if e == nil {
		e = os.WriteFile(path, b, 0600)
	}
	if e != nil {
		t.Fatal(e)
	}
}
func TestPreflightReadOnlyAndRuntimeDrift(t *testing.T) {
	for _, full := range []bool{false, true} {
		mode := "controller"
		if full {
			mode = "full"
		}
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			data := filepath.Join(dir, "data")
			os.Mkdir(data, 0700)
			cfg := filepath.Join(dir, "config.json")
			write(t, cfg, map[string]any{"dataDir": data, "allowedTasks": []string{"stop"}})
			manifest := filepath.Join(dir, "deployment-manifest.json")
			receipt := filepath.Join(dir, "deployment.json")
			runtime := filepath.Join(dir, "runtime.json")
			m := Manifest{Format: 1, Source: source, Version: "1111111", Ancestors: []string{source, parent}, AtlasSchema: atlas.SchemaVersion, Tasks: config.KnownTasks, Actor: "agent:test/preflight", Files: map[string]string{}}
			names := []string{"controller.exe"}
			if full {
				os.Mkdir(filepath.Join(dir, "plugin"), 0700)
				names = append(names, "plugin/Renki.Core.dll", "plugin/Renki.Plugin.dll")
			}
			for _, name := range names {
				os.WriteFile(filepath.Join(dir, name), []byte("candidate "+name), 0600)
				m.Files[name], _ = Hash(filepath.Join(dir, name))
			}
			r := Runtime{Version: "2222222", GameRunning: false, ControllerRunning: true, Files: map[string]string{"controller.exe": "installed-hash"}}
			write(t, manifest, m)
			write(t, runtime, r)
			check := func(want string) {
				t.Helper()
				err := Check(manifest, receipt, runtime, cfg, source, m.Version)
				if want == "" && err != nil || want != "" && (err == nil || !strings.Contains(err.Error(), want)) {
					t.Fatalf("want %q, got %v", want, err)
				}
			}
			check("") // safely adopt a recognizable clean legacy ancestor
			write(t, receipt, Receipt{Manifest: Manifest{Format: 1, Source: parent, Version: r.Version}, InstalledFiles: r.Files, Previous: r})
			check("")
			write(t, receipt, Receipt{Manifest: Manifest{Format: 1, Source: parent, Version: r.Version}, InstalledFiles: map[string]string{"controller.exe": "changed"}})
			check("checksums differ")
			write(t, receipt, Receipt{Manifest: Manifest{Format: 1, Source: parent, Version: r.Version}, InstalledFiles: r.Files})
			r.Version = "unknown"
			write(t, runtime, r)
			check("version differs")
			r.Version = "2222222"
			write(t, runtime, r)
			write(t, receipt, Receipt{Manifest: Manifest{Format: 1, Source: strings.Repeat("3", 40), Version: r.Version}, InstalledFiles: r.Files})
			check("ancestry")
			os.Remove(receipt)
			r.Version = "2222222-dirty"
			write(t, runtime, r)
			check("legacy")
			r.Version = "2222222"
			write(t, runtime, r)
			settings := filepath.Join(data, "settings.json")
			write(t, settings, map[string]any{"overrides": map[string]any{"allowedTasks": map[string]any{"value": []string{"future_task"}}}})
			check("unsupported")
			write(t, settings, map[string]any{"overrides": map[string]any{"allowedTasks": map[string]any{"value": []string{"inspect_container"}}}})
			atlasPath := filepath.Join(data, "atlas.sqlite")
			db, e := sql.Open("sqlite", atlasPath)
			if e != nil {
				t.Fatal(e)
			}
			_, e = db.Exec("PRAGMA user_version=99")
			if e != nil {
				t.Fatal(e)
			}
			db.Close()
			before, _ := Hash(atlasPath)
			saved, _ := Hash(settings)
			check("atlas schema 99")
			after, _ := Hash(atlasPath)
			savedAfter, _ := Hash(settings)
			if before != after || saved != savedAfter {
				t.Fatal("read-only preflight changed durable data")
			}
			if _, e = os.Stat(atlasPath + "-wal"); !os.IsNotExist(e) {
				t.Fatal("preflight created a WAL")
			}
			os.Remove(atlasPath)
			os.WriteFile(filepath.Join(dir, "controller.exe"), []byte("corrupt"), 0600)
			check("checksum mismatch")
		})
	}
}
func TestMissingAtlasDoesNotCreateData(t *testing.T) {
	p := filepath.Join(t.TempDir(), "missing", "atlas.sqlite")
	if n, e := AtlasSchema(p); e != nil || n != 0 {
		t.Fatal(n, e)
	}
	if _, e := os.Stat(filepath.Dir(p)); !os.IsNotExist(e) {
		t.Fatal("created missing data directory")
	}
}
