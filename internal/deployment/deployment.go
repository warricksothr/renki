// Package deployment checks a staged build without opening any writable stores.
package deployment

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/atlas"
	"github.com/warricksothr/renki/internal/config"
)

type Manifest struct {
	Branch      string            `json:"branch"`
	Format      int               `json:"format"`
	Source      string            `json:"source"`
	Version     string            `json:"version"`
	Ancestors   []string          `json:"ancestors"`
	AtlasSchema int               `json:"atlasSchema"`
	Tasks       []string          `json:"tasks"`
	Files       map[string]string `json:"files"`
	Actor       string            `json:"actor"`
}
type Runtime struct {
	GameRunning       bool              `json:"gameRunning"`
	ControllerRunning bool              `json:"controllerRunning"`
	Version           string            `json:"version"`
	Files             map[string]string `json:"files"`
}
type Receipt struct {
	Manifest       Manifest          `json:"manifest"`
	InstalledFiles map[string]string `json:"installedFiles"`
	Previous       Runtime           `json:"previous"`
	InstalledAt    time.Time         `json:"installedAt"`
}

func Hash(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
func Read(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}
func ancestor(m Manifest, source string) bool {
	for _, a := range m.Ancestors {
		if a == source {
			return true
		}
	}
	return false
}

// Check validates artifacts, ancestry and durable data before the runtime stops.
// It never calls atlas.Open, settings.Open, or any startup path that can write.
func Check(manifestPath, installedPath, runtimePath, configPath, source, version string) error {
	var m Manifest
	if err := Read(manifestPath, &m); err != nil {
		return errors.New("cannot read candidate deployment manifest")
	}
	if m.Format != 1 || m.Source != source || m.Version != version || len(source) != 40 || !ancestor(m, source) || strings.HasSuffix(version, "-dirty") || m.Actor == "" {
		return errors.New("candidate manifest does not describe a committed, attributed build; rebuild after committing")
	}
	if m.AtlasSchema != atlas.SchemaVersion || !slices.Equal(m.Tasks, config.KnownTasks) {
		return errors.New("candidate capabilities do not match its controller")
	}
	if len(m.Files) != 1 && len(m.Files) != 3 {
		return errors.New("candidate manifest must contain a controller or a complete controller/Core/plugin package")
	}
	for name, want := range m.Files {
		if name != "controller.exe" && name != "plugin/Renki.Core.dll" && name != "plugin/Renki.Plugin.dll" {
			return errors.New("unexpected candidate artifact")
		}
		got, err := Hash(filepath.Join(filepath.Dir(manifestPath), filepath.FromSlash(name)))
		if err != nil || got != want {
			return fmt.Errorf("candidate checksum mismatch: %s", name)
		}
	}
	if m.Files["controller.exe"] == "" || len(m.Files) == 3 && (m.Files["plugin/Renki.Core.dll"] == "" || m.Files["plugin/Renki.Plugin.dll"] == "") {
		return errors.New("incomplete candidate package")
	}
	var r Runtime
	if err := Read(runtimePath, &r); err != nil {
		return errors.New("cannot read runtime snapshot")
	}
	var installed Receipt
	err := Read(installedPath, &installed)
	switch {
	case err == nil:
		if installed.Manifest.Format != 1 || len(installed.Manifest.Source) != 40 || installed.InstalledFiles["controller.exe"] == "" {
			return errors.New("installed deployment receipt is invalid")
		}
		if !ancestor(m, installed.Manifest.Source) {
			return fmt.Errorf("runtime drift: installed source %s (branch %q) is absent from candidate ancestry; integrate that source first", installed.Manifest.Source[:12], installed.Manifest.Branch)
		}
		if r.Version != installed.Manifest.Version {
			return errors.New("runtime drift: installed controller version differs from its receipt")
		}
		if !maps.Equal(r.Files, installed.InstalledFiles) {
			return errors.New("runtime drift: installed checksums differ from the deployment receipt")
		}
	case errors.Is(err, os.ErrNotExist):
		if r.Files["controller.exe"] != "" {
			// Legacy controllers expose only -version. Adopt only an identifiable,
			// clean ancestor; an unknown version must never imply permission to replace.
			matched := false
			if !strings.HasSuffix(r.Version, "-dirty") {
				for _, a := range m.Ancestors {
					legacy := r.Version
					if i := strings.LastIndex(legacy, "-g"); i >= 0 {
						legacy = legacy[i+2:]
					}
					if len(legacy) >= 7 && strings.HasPrefix(a, legacy) {
						matched = true
					}
				}
			}
			if !matched {
				return fmt.Errorf("runtime drift: legacy controller version %q has no recognizable ancestor; build on its source before installing", r.Version)
			}
		}
	default:
		return errors.New("cannot read installed deployment receipt")
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return errors.New("candidate cannot load the installed configuration; inspect configuration compatibility locally")
	}
	var saved struct {
		Overrides map[string]struct {
			Value json.RawMessage `json:"value"`
		} `json:"overrides"`
	}
	err = Read(filepath.Join(cfg.DataDir, "settings.json"), &saved)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.New("cannot inspect saved settings")
	}
	if o, ok := saved.Overrides["allowedTasks"]; ok {
		var tasks []string
		if json.Unmarshal(o.Value, &tasks) != nil {
			return errors.New("invalid saved task permissions")
		}
		for _, task := range tasks {
			if !slices.Contains(config.KnownTasks, task) {
				return fmt.Errorf("saved task permission is unsupported: %s", task)
			}
		}
	}
	schema, err := AtlasSchema(filepath.Join(cfg.DataDir, "atlas.sqlite"))
	if err != nil {
		return fmt.Errorf("cannot inspect atlas read-only: %w", err)
	}
	if schema > m.AtlasSchema {
		return fmt.Errorf("atlas schema %d is newer than candidate support %d", schema, m.AtlasSchema)
	}
	return nil
}

func AtlasSchema(path string) (int, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return 0, err
	}
	p := filepath.ToSlash(path)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	u := url.URL{Scheme: "file", Path: p, RawQuery: "mode=ro"}
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return 0, err
	}
	defer db.Close()
	var schema int
	err = db.QueryRow("PRAGMA user_version").Scan(&schema)
	return schema, err
}
