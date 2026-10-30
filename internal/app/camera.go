package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// CameraHandler serves the latest image separately from metadata/audit streams.
func (a *App) CameraHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/camera", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		cfg, base, override := a.cameraSettings()
		frame := a.Session.Screenshot()
		if !cfg.Enabled || frame == nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"available": false, "config": cfg, "baseConfig": base, "override": override})
			return
		}
		frame.JPEG = nil
		_ = json.NewEncoder(w).Encode(map[string]any{"available": true, "config": cfg, "baseConfig": base, "override": override, "frame": frame, "ageS": time.Since(frame.ReceivedAt).Seconds(), "imageUrl": fmt.Sprintf("/api/camera.jpg?connection=%d&seq=%d", frame.Connection, frame.Seq)})
	})
	mux.HandleFunc("GET /api/camera.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		image := a.Session.Screenshot()
		cfg, _, _ := a.cameraSettings()
		if !cfg.Enabled || image == nil {
			http.Error(w, "No current camera image", http.StatusNotFound)
			return
		}
		// A URL for an older frame must never silently return a different scene.
		for key, want := range map[string]uint64{"connection": image.Connection, "seq": image.Seq} {
			if v := r.URL.Query().Get(key); v != "" && v != strconv.FormatUint(want, 10) {
				http.Error(w, "Camera frame superseded; refresh metadata", http.StatusConflict)
				return
			}
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("X-Renki-Captured-At", image.CapturedAt.Format(time.RFC3339Nano))
		_, _ = w.Write(image.JPEG)
	})
	mux.HandleFunc("POST /api/camera", a.cameraCommand)
	return mux
}
