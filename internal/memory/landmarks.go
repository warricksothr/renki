package memory

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

const landmarkBatchEvery = 30 * time.Second

// Landmarks batches observations without issuing tasks. Call Tick/Flush from
// one goroutine. Pending observations retain their own world through outages.
type Landmarks struct {
	Session *session.Session
	Store   *home.Store
	Logger  *slog.Logger
	conn    *protocol.Hello
	frame   *protocol.Perception
	world   string
	pending map[string]home.Landmark
	since   time.Time
}

func (l *Landmarks) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			l.Tick()
			l.Flush()
			return
		case <-t.C:
			l.Tick()
		}
	}
}

func (l *Landmarks) Tick() { l.tick(time.Now()) }

func (l *Landmarks) tick(now time.Time) {
	conn, world, frame, at := l.Session.PerceptionObservation()
	if world != "" && world != l.world {
		// Do not discard a failed batch or give it the new world's name.
		if !l.flush(now) {
			return
		}
		l.world, l.conn, l.frame = world, nil, nil
	}
	if conn != nil && (conn != l.conn || frame != l.frame) {
		if len(l.pending) == 0 {
			l.since = now
		}
		if l.pending == nil {
			l.pending = map[string]home.Landmark{}
		}
		for _, o := range landmarks(frame, at) {
			if _, exists := l.pending[o.Entity]; !exists && len(l.pending) >= home.MaxLandmarks {
				continue
			}
			if !l.pending[o.Entity].At.After(o.At) {
				l.pending[o.Entity] = o
			}
		}
		l.conn, l.frame = conn, frame
	}
	if len(l.pending) > 0 && now.Sub(l.since) >= landmarkBatchEvery {
		l.flush(now)
	}
}

// Flush persists the batch with original arrival times; failure retains it.
func (l *Landmarks) Flush() bool { return l.flush(time.Now()) }

func (l *Landmarks) flush(now time.Time) bool {
	if len(l.pending) == 0 {
		return true
	}
	updates := make([]home.Landmark, 0, len(l.pending))
	for _, o := range l.pending {
		updates = append(updates, o)
	}
	sort.Slice(updates, func(i, j int) bool { return updates[i].Entity < updates[j].Entity })
	if err := l.Store.RememberLandmarks(l.world, updates, now); err != nil {
		if l.Logger != nil {
			l.Logger.Warn("could not remember landmarks", "world", l.world, "err", err)
		}
		return false
	}
	l.pending = nil
	return true
}

func landmarks(frame *protocol.Perception, at time.Time) []home.Landmark {
	out := []home.Landmark{}
	for _, s := range frame.Structures {
		if s.ID == "" || s.Kind == protocol.StructureTombstone || s.Kind == protocol.StructureOther || s.Kind == "" {
			continue
		}
		out = append(out, home.Landmark{Entity: s.ID, Kind: s.Kind, Prefab: s.Prefab, Pos: s.Pos, Tag: s.Tag, At: at})
	}
	for _, r := range frame.Resources {
		if r.ID == "" || !notableResource(r) {
			continue
		}
		var tool *int
		if r.MinTool != nil {
			v := *r.MinTool
			tool = &v
		}
		out = append(out, home.Landmark{Entity: r.ID, Kind: r.Kind, Prefab: r.Prefab, Pos: r.Pos, MinTool: tool, At: at})
	}
	return out
}

func notableResource(r protocol.PerceivedResource) bool {
	if r.Kind == protocol.ResourcePickable {
		return true
	}
	if r.Kind != protocol.ResourceRock {
		return false
	}
	// Ordinary stone remains scenery. This conservative list names Valheim's
	// persistent mining deposits; unknown/modded prefabs wait for a catalogue.
	name := strings.ToLower(r.Prefab)
	for _, prefix := range []string{"copper", "tin", "silver", "obsidian", "mine_rock_meteorite", "giant_brain", "giant_ribs", "giant_skull", "flametal"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
