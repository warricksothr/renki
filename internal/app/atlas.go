package app

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/warricksothr/renki/internal/atlas"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

// AtlasHandler is read-only and registered behind panel authentication.
func (a *App) AtlasHandler() http.Handler { return a.atlasHandler(true) }
func (a *App) atlasHandler(contents bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/atlas/worlds", func(w http.ResponseWriter, r *http.Request) {
		if a.Atlas == nil {
			http.Error(w, "atlas unavailable", http.StatusServiceUnavailable)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		worlds, err := a.Atlas.Worlds(ctx)
		if err != nil {
			http.Error(w, "atlas read failed", http.StatusServiceUnavailable)
			return
		}
		writeAtlas(w, worlds)
	})
	mux.HandleFunc("GET /api/atlas/map", func(w http.ResponseWriter, r *http.Request) {
		if a.Atlas == nil {
			http.Error(w, "atlas unavailable", http.StatusServiceUnavailable)
			return
		}
		key := r.URL.Query().Get("world")
		live := key == "live"
		var connection uint64
		var world string
		var scope *protocol.TerrainScope
		if live {
			world, connection, scope, _ = a.Session.TerrainView()
			if scope == nil {
				writeAtlas(w, map[string]any{"world": world, "connection": connection, "tiles": []atlas.Tile{}})
				return
			}
			key = atlas.Key(*scope)
		}
		if len(key) != 64 {
			http.Error(w, "invalid atlas world key", http.StatusBadRequest)
			return
		}
		bounds := atlas.Bounds{}
		for name, target := range map[string]*float64{"minX": &bounds.MinX, "maxX": &bounds.MaxX, "minZ": &bounds.MinZ, "maxZ": &bounds.MaxZ} {
			value, err := strconv.ParseFloat(r.URL.Query().Get(name), 64)
			if err != nil {
				http.Error(w, "invalid viewport", http.StatusBadRequest)
				return
			}
			*target = value
		}
		if !bounds.Valid() {
			http.Error(w, "viewport must be at most 512 m per side", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		worlds, err := a.Atlas.Worlds(ctx)
		if err != nil {
			http.Error(w, "atlas read failed", http.StatusServiceUnavailable)
			return
		}
		var selected *atlas.World
		for _, candidate := range worlds {
			if candidate.Key == key {
				selected = &candidate
				break
			}
		}
		if selected == nil && !live {
			http.Error(w, "saved atlas not found", http.StatusNotFound)
			return
		}
		tiles, err := a.Atlas.Tiles(ctx, key, bounds, time.Now())
		if err != nil {
			http.Error(w, "atlas read failed", http.StatusServiceUnavailable)
			return
		}
		if live {
			currentWorld, currentConn, currentScope, _ := a.Session.TerrainView()
			if currentWorld != world || currentConn != connection || currentScope == nil || atlas.Key(*currentScope) != key {
				http.Error(w, "live world changed", http.StatusConflict)
				return
			}
		}
		result := map[string]any{"worldKey": key, "world": world, "connection": connection, "saved": !live, "manifest": selected, "tiles": tiles}
		if !live && selected != nil {
			// Saved terrain and landmarks share the explicitly selected world.
			// Never substitute another live session's objects or imply freshness.
			landmarks := []home.Landmark{}
			for _, l := range a.Landmarks.Store.Landmarks(selected.Name) {
				if l.Pos[0] >= bounds.MinX && l.Pos[0] <= bounds.MaxX && l.Pos[2] >= bounds.MinZ && l.Pos[2] <= bounds.MaxZ {
					landmarks = append(landmarks, l)
					if len(landmarks) == 512 {
						break
					}
				}
			}
			result["landmarks"] = landmarks
			mapped := make(map[string]bool, len(landmarks))
			for _, l := range landmarks {
				mapped[l.Entity] = true
			}
			containers := []home.RememberedContainer{}
			for _, c := range a.Memory.Store.RememberedContainers(selected.Name) {
				if mapped[c.Entity] {
					containers = append(containers, c)
				}
			}
			if contents {
				result["containers"] = containers
			}
		}
		writeAtlas(w, result)
	})
	return mux
}
func writeAtlas(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(value)
}
