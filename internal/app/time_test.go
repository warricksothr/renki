package app

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestStatusPublishesNativeDayNightOnlyForALiveWorld(t *testing.T) {
	a, p, _, _ := goalFixture(t)
	day := protocol.GameTime{Day: 100, DayFraction: .35, Night: false}
	night := protocol.GameTime{Day: 100, DayFraction: .9, Night: true}
	for _, clock := range []protocol.GameTime{day, night} {
		p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one", Time: clock})
		deadline := time.Now().Add(time.Second)
		for a.Session.State().Time != clock && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		status := a.PanelStatus("test")
		raw, err := json.Marshal(status)
		if err != nil {
			t.Fatal(err)
		}
		var wire struct {
			Time *protocol.GameTime `json:"time"`
		}
		if err := json.Unmarshal(raw, &wire); err != nil || wire.Time == nil || *wire.Time != clock {
			t.Fatalf("native clock missing or altered: %s (%v)", raw, err)
		}
	}
	p.Send(&protocol.State{Phase: protocol.PhaseMenu, Time: night})
	deadline := time.Now().Add(time.Second)
	for a.Session.State().Phase != protocol.PhaseMenu && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if a.PanelStatus("test").Time != nil {
		t.Fatal("menu retained a stale world clock")
	}
	p.Close()
	for a.PanelStatus("test").Connected && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if a.PanelStatus("test").Time != nil {
		t.Fatal("offline status retained a world clock")
	}
}
