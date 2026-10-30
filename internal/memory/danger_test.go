package memory

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func TestDangerRecorderRetriesWithOriginalWorldAndTimeAndFlushesOnStop(t *testing.T) {
	s := session.New(listener{}, nil)
	c := &bridge.Conn{Hello: &protocol.Hello{}, Welcome: &protocol.Welcome{}}
	s.Connected(c)
	folder := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(folder, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := home.Open(filepath.Join(folder, "memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	d := &Danger{Store: store}
	st := &protocol.State{Envelope: protocol.Envelope{Seq: 1, T: 10}, Phase: protocol.PhaseInWorld, World: "one", HP: 25}
	s.Message(c, st)
	event := &protocol.Event{Envelope: protocol.Envelope{Seq: 2, T: 10}, Event: protocol.EventDamaged, Amount: 5, By: "enemy", Prefab: "Greyling"}
	record := func(e *protocol.Event) {
		if observation, valid := s.ObserveInjury(c, e, time.Now().UTC()); valid {
			d.Observe(e, observation)
		}
	}
	for _, amount := range []float64{math.NaN(), math.Inf(1), -1, 0} {
		bad := *event
		bad.Amount = amount
		d.Observe(&bad, session.InjuryObservation{World: "one", At: time.Now().UTC()})
	}
	record(event)
	record(event) // duplicate sequence cannot count twice
	if err = os.WriteFile(filepath.Join(folder, "memory.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(folder, folder+"-saved"); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(folder, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	if d.Flush() || len(store.Harms("one")) != 0 || len(d.pending) != 1 {
		t.Fatal("failed save lost/mutated batch")
	}
	at := d.pending[0].At
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 3, T: 11}, Phase: protocol.PhaseInWorld, World: "two", HP: 25})
	event.Seq, event.T, event.Event = 4, 11, protocol.EventDied
	record(event)
	if err = os.Remove(folder); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(folder+"-saved", folder); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d.Run(ctx)
	reopened, err := home.Open(filepath.Join(folder, "memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	one, two := reopened.Harms("one"), reopened.Harms("two")
	if len(one) != 1 || len(two) != 1 || !one[0].At.Equal(at) || one[0].Prefab != "Greyling" || two[0].Event != protocol.EventDied {
		t.Fatal(one, two)
	}
}

func TestDangerQueueStaysBoundedDuringPersistenceOutage(t *testing.T) {
	s := session.New(listener{}, nil)
	c := &bridge.Conn{Hello: &protocol.Hello{}, Welcome: &protocol.Welcome{}}
	s.Connected(c)
	s.Message(c, &protocol.State{Envelope: protocol.Envelope{Seq: 1, T: 1}, Phase: protocol.PhaseInWorld, World: "one", HP: 25})
	d := &Danger{Store: home.Memory()}
	for i := range 600 {
		d.Observe(&protocol.Event{Envelope: protocol.Envelope{Seq: uint64(i + 2), T: 1}, Event: protocol.EventDamaged, Amount: 1}, session.InjuryObservation{World: "one", At: time.Now().UTC()})
	}
	if len(d.queue) != 256 || d.dropped.Load() != 344 {
		t.Fatal(len(d.queue), d.dropped.Load())
	}
	if !d.Flush() || len(d.Store.Harms("one")) != home.MaxHarms {
		t.Fatal("bounded queue did not flush")
	}
}
