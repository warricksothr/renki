// Command deploymanifest records the candidate build's compatibility and provenance.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/warricksothr/renki/internal/atlas"
	"github.com/warricksothr/renki/internal/config"
	"github.com/warricksothr/renki/internal/deployment"
)

func main() {
	dir := flag.String("dir", "", "staged package directory")
	version := flag.String("version", "", "build version")
	actor := flag.String("actor", "", "deployment actor")
	flag.Parse()
	status, err := exec.Command("git", "status", "--porcelain", "--untracked-files=normal").Output()
	if err != nil || len(strings.TrimSpace(string(status))) != 0 {
		log.Fatal("deployment requires a clean committed source tree")
	}
	branch, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		log.Fatal(err)
	}
	history, err := exec.Command("git", "rev-list", "HEAD").Output()
	if err != nil {
		log.Fatal(err)
	}
	ancestors := strings.Fields(string(history))
	if len(ancestors) == 0 {
		log.Fatal("empty source history")
	}
	if *actor == "" {
		var lease struct {
			Actor string `json:"actor"`
		}
		_ = deployment.Read(".vm-lease", &lease)
		*actor = lease.Actor
		if *actor == "" {
			host, _ := os.Hostname()
			*actor = os.Getenv("USER") + "@" + host
		}
	}
	m := deployment.Manifest{Format: 1, Branch: strings.TrimSpace(string(branch)), Source: ancestors[0], Version: *version, Ancestors: ancestors, AtlasSchema: atlas.SchemaVersion, Tasks: config.KnownTasks, Actor: *actor, Files: map[string]string{}}
	for _, name := range []string{"controller.exe", "plugin/Renki.Core.dll", "plugin/Renki.Plugin.dll"} {
		hash, err := deployment.Hash(filepath.Join(*dir, filepath.FromSlash(name)))
		if os.IsNotExist(err) && name != "controller.exe" {
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
		m.Files[name] = hash
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err == nil {
		err = os.WriteFile(filepath.Join(*dir, "deployment-manifest.json"), append(b, '\n'), 0600)
	}
	if err != nil {
		log.Fatal(err)
	}
}
