package app

import (
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestPanelAndMapShareLiveEventKnowledge(t *testing.T) {
	a, p, _, _ := goalFixture(t)
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one", WorldEvents: &protocol.WorldEventStatus{RandomEvent: &protocol.AreaEvent{Name: "army_greydwarf", Radius: 96}}})
	deadline := time.Now().Add(time.Second)
	for a.PanelStatus("test").WorldEvents == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	status := a.PanelStatus("test")
	mapped := a.MapSnapshot().(MapSnapshot)
	if status.WorldEvents == nil || mapped.WorldEvents == nil || status.WorldEvents.RandomEvent.Name != mapped.WorldEvents.RandomEvent.Name {
		t.Fatal(status.WorldEvents, mapped.WorldEvents)
	}
	p.Close()
	for a.PanelStatus("test").Connected && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if a.PanelStatus("test").WorldEvents != nil || a.MapSnapshot().(MapSnapshot).WorldEvents != nil {
		t.Fatal("event persisted as live after logout")
	}
}
