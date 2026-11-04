package session

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/bridge/bridgetest"
	"github.com/warricksothr/renki/internal/protocol"
)

func harvestSession(t *testing.T, features []string) (*Session, *bridgetest.Plugin) {
	t.Helper()
	s := New(&listener{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	url := bridgetest.Serve(t, bridge.Options{Welcome: func(*protocol.Hello) *protocol.Welcome {
		return &protocol.Welcome{AllowedTasks: []string{protocol.TaskHarvest}}
	}}, s)
	p := bridgetest.ConnectWithFeatures(t, url, "", []string{protocol.TaskHarvest}, features)
	waitFor(t, func() bool { _, _, ok := s.Handshake(); return ok })
	s.SetHarvestAreas(func(world string) []protocol.HarvestExclusion {
		if world != "one" {
			t.Errorf("wrong policy world %q", world)
		}
		return []protocol.HarvestExclusion{{Center: protocol.Vec3{-204.39, 32.97, -467.43}, Radius: 60}}
	})
	p.Send(&protocol.State{World: "one", Phase: protocol.PhaseInWorld})
	waitFor(t, func() bool { return s.State() != nil })
	return s, p
}

func observeHarvest(t *testing.T, s *Session, p *bridgetest.Plugin, r protocol.PerceivedResource) {
	t.Helper()
	p.Send(&protocol.Perception{Resources: []protocol.PerceivedResource{r}})
	waitFor(t, func() bool {
		_, _, seen, _ := s.PerceptionObservation()
		return seen != nil && len(seen.Resources) > 0 && seen.Resources[0].ID == r.ID
	})
}

func TestHarvestDispatchRefusesCampEdgesWetUnknownStaleAndMissingTargets(t *testing.T) {
	s, p := harvestSession(t, []string{protocol.FeatureHarvestExclusions})
	water := 30.0
	for _, tc := range []struct {
		name string
		r    protocol.PerceivedResource
		want string
	}{
		{"captured decorative", protocol.PerceivedResource{Kind: "tree", Pos: protocol.Vec3{-192.44, 30.31, -468.84}, WaterLevel: &water}, "protected saved base"},
		{"exact edge any height", protocol.PerceivedResource{Kind: "rock", Pos: protocol.Vec3{-204.39, 200, -407.43}, WaterLevel: &water}, "protected saved base"},
		{"underwater outside", protocol.PerceivedResource{Kind: "tree", Pos: protocol.Vec3{-100, 25, -467.43}, WaterLevel: &water}, "above known water"},
		{"unknown outside", protocol.PerceivedResource{Kind: "tree", Pos: protocol.Vec3{-100, 35, -467.43}}, "above known water"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.r.ID = tc.name
			observeHarvest(t, s, p, tc.r)
			// Raw callers cannot bypass base policy by supplying an empty list.
			_, err := s.Dispatch(context.Background(), Order{Task: protocol.TaskHarvest, Args: map[string]any{"entity": tc.r.ID, "exclude": []any{}}})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatal(err)
			}
		})
	}
	r := protocol.PerceivedResource{ID: "outside", Kind: "tree", Pos: protocol.Vec3{-100, 35, -467.43}, WaterLevel: &water}
	observeHarvest(t, s, p, r)
	if _, err := s.Dispatch(context.Background(), Order{Task: protocol.TaskHarvest, Args: protocol.HarvestArgs{Entity: "missing"}}); err == nil {
		t.Fatal("unobserved target sent")
	}
	s.mu.Lock()
	s.seenAt = time.Now().Add(-6 * time.Second)
	s.mu.Unlock()
	if _, err := s.Dispatch(context.Background(), Order{Task: protocol.TaskHarvest, Args: protocol.HarvestArgs{Entity: r.ID}}); err == nil {
		t.Fatal("stale target sent")
	}
}

func TestHarvestDispatchAddsNativeExclusionsAndRejectsOldPlugin(t *testing.T) {
	for _, supported := range []bool{false, true} {
		t.Run(map[bool]string{false: "old", true: "supported"}[supported], func(t *testing.T) {
			var features []string
			if supported {
				features = []string{protocol.FeatureHarvestExclusions}
			}
			s, p := harvestSession(t, features)
			water := 30.0
			r := protocol.PerceivedResource{ID: "outside", Kind: "tree", Pos: protocol.Vec3{-100, 35, -467.43}, WaterLevel: &water}
			observeHarvest(t, s, p, r)
			_, err := s.Dispatch(context.Background(), Order{Task: protocol.TaskHarvest, Args: protocol.HarvestArgs{Entity: r.ID}})
			if !supported {
				if err == nil || !strings.Contains(err.Error(), "protected dry harvesting") {
					t.Fatal(err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			req := p.NextTask()
			decoded, err := req.DecodeArgs()
			if err != nil {
				t.Fatal(err)
			}
			areas := decoded.(*protocol.HarvestArgs).Exclude
			if len(areas) != 1 || areas[0].Radius != 60 || areas[0].Center[0] != -204.39 {
				t.Fatal(areas)
			}
		})
	}
}
