package behaviour

import (
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

func TestWoodStorageCanBeFiftyMetresFromSleeping(t *testing.T) {
	v := woodView(t)
	b := home.Base{Name: "camp", Center: protocol.Vec3{}, Radius: 100, DurationS: 600, TravelM: 800, Return: "home", Sleeping: "bedroom", Storage: []string{"warehouse"}, At: time.Now()}
	v.Base = &b
	v.Waypoints = []home.Waypoint{{Name: "bedroom", Kind: "rest", Pos: protocol.Vec3{-25, 0, 0}}, {Name: "warehouse", Kind: "storage", Pos: protocol.Vec3{25, 0, 0}, Radius: 8}}
	v.Containers = []*protocol.Container{{Entity: "new chest", Pos: protocol.Vec3{25, 0, 0}}, {Entity: "old home chest", Pos: *v.Home}}
	p, _, err := planWood(v, nil, nil)
	if err != nil || len(p.To) != 1 || p.To[0] != "new chest" {
		t.Fatal(p, err)
	}
	v.Base.Storage = nil
	if _, _, err = planWood(v, nil, nil); err == nil {
		t.Fatal("unassigned storage silently fell back")
	}
}
func TestRestUsesSleepingRoleAwayFromStorageAndReturn(t *testing.T) {
	r, v := restFixture(t, false)
	sleep := home.Waypoint{Name: "bedroom", Kind: "rest", Pos: protocol.Vec3{50, 0, 0}}
	v.Base = &home.Base{Name: "camp", Center: protocol.Vec3{25, 0, 0}, Radius: 80, DurationS: 600, TravelM: 800, Return: "home", Sleeping: "bedroom", Storage: []string{"warehouse"}, At: time.Now()}
	v.Waypoints = append(v.Waypoints, sleep, home.Waypoint{Name: "warehouse", Kind: "storage", Pos: protocol.Vec3{0, 0, 0}})
	v.State.Pos = protocol.Vec3{50, 0, 0}
	v.Perception.Structures[0].Pos = protocol.Vec3{54, 0, 0}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskMoveTo {
		t.Fatal(o, err)
	}
	v.Base.Sleeping = "missing"
	if err = r.Check(v); err == nil {
		t.Fatal("missing sleeping binding accepted")
	}
}
