package app

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestPanelRetainsNativeEscortIdentityAndClearsItOffline(t *testing.T) {
	a, p, _, _ := goalFixture(t)
	p.Send(&protocol.State{Phase: protocol.PhaseInWorld, World: "one", HP: 80, HPMax: 80,
		Task:   &protocol.TaskRef{ID: "e-1", Task: protocol.TaskEscort, Status: protocol.StatusRunning},
		Escort: &protocol.EscortStatus{ID: "e-1", Leader: "Nimah", Mode: "follow", Reason: "rejoining"}})
	deadline := time.Now().Add(time.Second)
	for a.Session.State().Escort == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	status := a.PanelStatus("test")
	if status.Escort == nil || status.Escort.ID != "e-1" || status.Escort.Leader != "Nimah" {
		t.Fatal(status.Escort)
	}
	raw, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	if len(fields["escort"]) == 0 {
		t.Fatal("escort missing from panel API")
	}
	p.Close()
	for a.PanelStatus("test").Connected && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if a.PanelStatus("test").Escort != nil {
		t.Fatal("offline escort retained")
	}
}
