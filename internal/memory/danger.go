package memory

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// Danger records events without blocking the bridge on persistence. Flush is
// owned by the worker (or a test); Observe may run concurrently with it.
type Danger struct {
	Store   *home.Store
	Logger  *slog.Logger
	mu      sync.Mutex
	queue   chan home.Harm
	dropped atomic.Uint64
	pending []home.Harm
}

func (d *Danger) Observe(e *protocol.Event, observation session.InjuryObservation) {
	if e.Event != protocol.EventDamaged && e.Event != protocol.EventDied {
		return
	}
	if math.IsNaN(e.Amount) || math.IsInf(e.Amount, 0) || e.Amount < 0 || (e.Event == protocol.EventDamaged && e.Amount == 0) {
		return
	}
	if observation.World == "" || observation.At.IsZero() {
		return
	}
	pos := observation.Pos
	for _, v := range pos {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return
		}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.queue == nil {
		d.queue = make(chan home.Harm, 256)
	}
	harm := home.Harm{World: observation.World, Pos: pos, At: observation.At, Event: e.Event, Amount: e.Amount, By: e.By, Prefab: e.Prefab}
	select {
	case d.queue <- harm:
	default:
		d.dropped.Add(1)
	}
}

func (d *Danger) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			if d.Flush() {
				d.Flush()
			}
			return
		case <-t.C:
			d.Flush()
		}
	}
}

// Flush retains the original event timestamps across failed saves. Queue and
// pending batch are bounded independently, even during a prolonged disk fault.
func (d *Danger) Flush() bool {
	d.mu.Lock()
	q := d.queue
	d.mu.Unlock()
	for len(d.pending) < 256 {
		select {
		case h := <-q:
			d.pending = append(d.pending, h)
		default:
			goto save
		}
	}
save:
	if n := d.dropped.Swap(0); n > 0 && d.Logger != nil {
		d.Logger.Warn("danger events dropped", "count", n)
	}
	if len(d.pending) == 0 {
		return true
	}
	if err := d.Store.RememberHarms(d.pending, time.Now().UTC()); err != nil {
		if d.Logger != nil {
			d.Logger.Warn("could not remember injuries", "err", err)
		}
		return false
	}
	d.pending = nil
	// A full failed batch can leave another bounded batch queued. Drain it on
	// the next tick, or in the second final flush at shutdown.
	return true
}
