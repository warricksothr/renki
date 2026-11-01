package goal

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/protocol"
)

func raid() *protocol.WorldEventStatus {
	return &protocol.WorldEventStatus{RandomEvent: &protocol.AreaEvent{Name: "army_greydwarf", Radius: 96, ElapsedS: 121, DurationS: 120, TimerMayPause: true, Inside: true}}
}

func TestHouseholdEventWaitAndFreshClearResume(t *testing.T) {
	host, r, v, id := stockFixture(t)
	v.State.Flags = []string{protocol.FlagInShelter}
	v.State.WorldEvents = raid()
	o, err := r.Next(v, nil)
	var delay *behaviour.Delay
	if o != nil || !errors.As(err, &delay) {
		t.Fatal("elapsed timer became expiry", o, err)
	}
	g, _ := host.Store.Get(id)
	if g.WaitS != 2 || g.Pending != nil {
		t.Fatal(g.WaitS, g.Pending)
	}
	// Repeated evaluations before wake must not charge another wait.
	_, _ = r.Next(v, nil)
	g, _ = host.Store.Get(id)
	if g.WaitS != 2 {
		t.Fatal("double charged wait", g.WaitS)
	}
	foodFresh(&v)
	v.State.WorldEvents = &protocol.WorldEventStatus{}
	o, err = r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskWithdraw {
		t.Fatal("fresh observed clear failed to resume", o, err)
	}
}

func TestHouseholdEventNeverSheltersWithActualThreatsOrWithoutShelter(t *testing.T) {
	for _, mode := range []string{"near", "pursuing", "outside"} {
		t.Run(mode, func(t *testing.T) {
			host, r, v, id := stockFixture(t)
			v.State.Flags = []string{protocol.FlagInShelter}
			v.State.WorldEvents = raid()
			switch mode {
			case "near":
				v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "enemy", HPFrac: 1, Hostile: true, Pos: protocol.Vec3{2, 0, 0}}}
			case "pursuing":
				v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "enemy", HPFrac: 1, Hostile: true, TargetingMe: true, Pos: protocol.Vec3{25, 0, 0}}}
			case "outside":
				v.State.Flags = nil
			}
			o, err := r.Next(v, nil)
			var wait *behaviour.Delay
			g, _ := host.Store.Get(id)
			if o != nil || err != nil || errors.As(err, &wait) || g.Status != "needs_plan" {
				t.Fatal("unsafe event wait admitted", o, err)
			}
		})
	}
}

func TestHouseholdEventUnknownWaitsAndDistantAreaDoesNotBlock(t *testing.T) {
	for _, mode := range []string{"unknown", "local", "distant"} {
		t.Run(mode, func(t *testing.T) {
			_, r, v, _ := stockFixture(t)
			v.State.Flags = []string{protocol.FlagInShelter}
			switch mode {
			case "unknown":
				v.State.WorldEvents = nil
			case "local":
				v.State.WorldEvents = &protocol.WorldEventStatus{LocalEvent: &protocol.EventNotice{Name: "boss", Kind: "local"}}
			case "distant":
				v.State.WorldEvents = raid()
				v.State.WorldEvents.RandomEvent.Center = protocol.Vec3{500, 0, 0}
			}
			o, err := r.Next(v, nil)
			var delay *behaviour.Delay
			if mode == "distant" {
				if err != nil || o == nil {
					t.Fatal(o, err)
				}
			} else if o != nil || !errors.As(err, &delay) {
				t.Fatal(o, err)
			}
		})
	}
}

func TestHouseholdEventSettlesReceiptBeforeWaitingAndBoundsWait(t *testing.T) {
	host, r, v, id := stockFixture(t)
	o, err := r.Next(v, nil)
	if err != nil || o == nil {
		t.Fatal(o, err)
	}
	last := foodTransfer(t, r, &v, "inspect", "a", meals[0], 0, nil)
	v.State.Flags = []string{protocol.FlagInShelter}
	v.State.WorldEvents = raid()
	o, err = r.Next(v, last)
	var delay *behaviour.Delay
	if o != nil || !errors.As(err, &delay) {
		t.Fatal(o, err)
	}
	g, _ := host.Store.Get(id)
	if g.Pending != nil || len(g.Food.Chests) == 0 || !g.Food.Chests[0].Observed {
		t.Fatal("receipt not settled", g.Pending, g.Food.Chests)
	}
	if err := host.Store.update(id, func(g *Record) error { g.WaitS = 240; g.Wake = time.Time{}; return nil }); err != nil {
		t.Fatal(err)
	}
	_, err = r.Next(v, nil)
	g, _ = host.Store.Get(id)
	if err != nil || g.Status != "needs_plan" || !strings.Contains(g.Reason, "wait budget exhausted") {
		t.Fatal(err, g.Status, g.Reason)
	}
}
