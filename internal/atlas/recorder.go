package atlas

import (
	"context"
	"log/slog"
	"time"

	"github.com/warricksothr/renki/internal/session"
)

type Recorder struct {
	Store   *Store
	Session *session.Session
	Logger  *slog.Logger
}

func (r *Recorder) Run(ctx context.Context) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	log := r.Logger
	if log == nil {
		log = slog.Default()
	}
	pending := []session.TerrainObservation{}
	nextMaintenance := time.Time{}
	flush := func() {
		// Do not drain the bounded session queue while a failed batch is pending.
		if len(pending) == 0 {
			var dropped uint64
			pending, dropped = r.Session.DrainTerrain()
			if dropped > 0 {
				log.Warn("atlas queue overflow", "tiles", dropped)
			}
		}
		writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		failed := false
		for len(pending) > 0 {
			if err := r.Store.Ingest(writeCtx, pending[0]); err != nil {
				log.Warn("atlas batch retained for retry", "err", err)
				failed = true
				break
			}
			pending = pending[1:]
			if len(pending) == 0 {
				var dropped uint64
				pending, dropped = r.Session.DrainTerrain()
				if dropped > 0 {
					log.Warn("atlas queue overflow", "tiles", dropped)
				}
			}
		}
		if failed || time.Now().After(nextMaintenance) {
			maintenanceCtx, maintenanceCancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer maintenanceCancel()
			if err := r.Store.Maintain(maintenanceCtx, time.Now()); err != nil {
				log.Warn("atlas maintenance", "err", err)
			}
			nextMaintenance = time.Now().Add(30 * time.Second)
		}
	}
	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-tick.C:
			flush()
		}
	}
}
