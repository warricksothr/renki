package session

import (
	"context"
	"github.com/warricksothr/renki/internal/protocol"
	"testing"
)

func TestOrderedPortalRequiresAdvertisedFeature(t *testing.T) {
	s, _, _ := setup(t, []string{protocol.TaskInteract}, []string{protocol.TaskInteract})
	_, err := s.Dispatch(context.Background(), Order{Task: protocol.TaskInteract, Args: protocol.InteractArgs{Entity: "1:10", PortalTo: "1:20"}, TimeoutS: 120})
	if err == nil {
		t.Fatal("ordered crossing dispatched to an old plugin")
	}
}
