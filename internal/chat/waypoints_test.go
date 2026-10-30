package chat

import (
	"testing"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

func TestOwnerWaypointCommandsMarkMoveRenameGoForgetAndWorlds(t *testing.T) {
	b, f, clock := newBrain(nil)
	store := home.Memory()
	b.opts.Places = store
	f.state = &protocol.State{Phase: protocol.PhaseInWorld, World: "one"}
	b.Chat(line("Mallory", "Bjorn mark this as the mine"))
	if len(store.Waypoints("one")) != 0 {
		t.Fatal("non-owner wrote a place")
	}
	b.Chat(line("Sothr", "Bjorn mark this as the mine"))
	w, ok := store.Waypoint("one", "mine")
	if !ok || w.Pos != (protocol.Vec3{10, 0, 10}) {
		t.Fatal(w)
	}
	clock.advance(1)
	m := line("Sothr", "Bjorn move the mine here")
	m.Pos = protocol.Vec3{20, 4, 30}
	b.Chat(m)
	w, _ = store.Waypoint("one", "mine")
	if w.Pos != m.Pos {
		t.Fatal(w)
	}
	b.Chat(line("Sothr", "Bjorn rename the mine to east quarry"))
	if _, ok := store.Waypoint("one", "mine"); ok {
		t.Fatal("old name remains")
	}
	b.Chat(line("Sothr", "Bjorn go to the east quarry"))
	orders := f.sent()
	found := false
	for _, o := range orders {
		if a, ok := o.Args.(*protocol.MoveToArgs); ok {
			found = a.To.Entity == "east quarry"
		}
	}
	if !found {
		t.Fatal(orders)
	}
	f.state = &protocol.State{Phase: protocol.PhaseInWorld, World: "two"}
	b.Chat(line("Sothr", "Bjorn forget east quarry"))
	if _, ok := store.Waypoint("one", "east quarry"); !ok {
		t.Fatal("other world deletion leaked")
	}
	f.state = &protocol.State{Phase: protocol.PhaseInWorld, World: "one"}
	b.Chat(line("Sothr", "Bjorn forget the east quarry"))
	if len(store.Waypoints("one")) != 0 {
		t.Fatal("not forgotten")
	}
}

func TestPanelCannotMarkAnUnseenOwnersPosition(t *testing.T) {
	b, f, _ := newBrain(nil)
	b.opts.Places = home.Memory()
	f.state = &protocol.State{Phase: protocol.PhaseInWorld, World: "one"}
	c := line("Sothr", "mark this as mine")
	c.Channel = channelPanel
	c.Dist = -1
	b.Chat(c)
	if len(b.opts.Places.Waypoints("one")) != 0 {
		t.Fatal("invented speaker position")
	}
}
