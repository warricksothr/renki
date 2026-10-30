// videoproof serves an isolated, read-only synthetic panel for local video tests.
// It cannot connect to the game bridge or run body commands.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/warricksothr/renki/internal/audit"
	"github.com/warricksothr/renki/internal/panel"
	"github.com/warricksothr/renki/internal/video"
)

func main() {
	data, err := os.ReadFile(".checks/streaming/panel.local.json")
	if err != nil {
		log.Fatal(err)
	}
	var cfg video.Config
	if err = json.Unmarshal(data, &cfg); err != nil {
		log.Fatal("Invalid proof config")
	}
	const world = "synthetic-video-proof"
	var sourceFresh atomic.Bool
	var online atomic.Bool
	online.Store(true)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	// This is a synthetic assertion only; production needs a real capture-health
	// adapter. Never enable this assertion writer in the controller.
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			if online.Load() {
				b, _ := json.Marshal(video.Health{Gameplay: sourceFresh.Load(), Connection: 1, WorldUID: world, ObservedAt: time.Now()})
				_ = os.WriteFile(cfg.HealthFile+".tmp", b, 0600)
				_ = os.Rename(cfg.HealthFile+".tmp", cfg.HealthFile)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	scope := func() video.Scope {
		return video.Scope{Online: online.Load(), Connection: 1, World: "SYNTHETIC — NOT GAME CAPTURE", WorldUID: world}
	}
	media := video.New(cfg, scope)
	trail, _ := audit.Open("", 1, nil)
	defer trail.Close()
	h := panel.Handler(panel.Options{Log: trail, Token: "local-synthetic-proof-token", Status: func() any {
		s := scope()
		phase := "menu"
		if s.Online {
			phase = "in_world"
		}
		return map[string]any{"connected": s.Online, "phase": phase, "connection": s.Connection, "world": s.World, "worldUid": s.WorldUID, "character": "SYNTHETIC VIDEO PROOF", "version": "local proof"}
	}, Video: media})
	mux := http.NewServeMux()
	mux.HandleFunc("POST /proof/offline", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Renki-Proof") != "1" {
			http.Error(w, "proof only", 403)
			return
		}
		online.Store(false)
	})
	mux.HandleFunc("POST /proof/online", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Renki-Proof") != "1" {
			http.Error(w, "proof only", 403)
			return
		}
		online.Store(true)
	})
	mux.HandleFunc("POST /proof/source/on", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Renki-Proof") != "1" {
			http.Error(w, "proof only", 403)
			return
		}
		sourceFresh.Store(true)
	})
	mux.HandleFunc("POST /proof/source/off", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Renki-Proof") != "1" {
			http.Error(w, "proof only", 403)
			return
		}
		sourceFresh.Store(false)
	})
	mux.Handle("/", h)
	srv := &http.Server{Addr: "127.0.0.1:18793", Handler: mux, ReadHeaderTimeout: 5 * time.Second, BaseContext: func(net.Listener) context.Context { return ctx }}
	go func() { <-ctx.Done(); srv.Close() }()
	log.Print("SYNTHETIC ONLY: panel http://127.0.0.1:18793 (local-synthetic-proof-token)")
	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
