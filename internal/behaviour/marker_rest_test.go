package behaviour

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

func restFixture(t *testing.T, sleep bool) (*markerRestRun, View) {
	t.Helper()
	v := markerView("cooking")
	v.Idle = true
	v.Home = &protocol.Vec3{}
	v.State.Time.Day = 1
	v.State.Time.Night = sleep
	v.State.Flags = []string{protocol.FlagGrounded, protocol.FlagInShelter}
	v.Perception.Structures = []protocol.PerceivedStructure{{ID: "mine", Kind: protocol.StructureBed, Mine: true, Pos: protocol.Vec3{4, 0, 0}}}
	d := MarkerRestDefinition(Marker{World: "one", Name: "work", Kind: "cooking"}, sleep)
	if err := validate(d, map[string]Template{"marker_rest": MarkerRest{}}); err != nil {
		t.Fatal(err)
	}
	r, err := (MarkerRest{}).Start(d.Params)
	if err != nil {
		t.Fatal(err)
	}
	return r.(*markerRestRun), v
}
func TestRestNeedsBedAttachmentBeyondANativeSleepReceipt(t *testing.T) {
	r, v := restFixture(t, true)
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskInteract || !o.Args.(*protocol.InteractArgs).Sleep || o.Args.(*protocol.InteractArgs).Entity != "mine" {
		t.Fatal(o, err)
	}
	result := &Result{Status: protocol.StatusSucceeded}
	v.State.Seq++
	if _, err = r.Next(v, result); err != ErrWait {
		t.Fatal("receipt alone earned sleep", err)
	}
	v.State.Flags = append(v.State.Flags, "in_bed")
	v.State.Pos = protocol.Vec3{4, .4, 0}
	if o, err = r.Next(v, result); err != nil || o != nil {
		t.Fatal(o, err)
	}
}
func TestDayRestWalksOnceAndLeavesTheBodyAvailable(t *testing.T) {
	r, v := restFixture(t, false)
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskMoveTo {
		t.Fatal(o, err)
	}
	a := o.Args.(*protocol.MoveToArgs)
	if !a.Dry || !a.AvoidHostiles || a.To.Entity != "mine" {
		t.Fatal(a)
	}
	result := &Result{Status: protocol.StatusSucceeded}
	if _, err = r.Next(v, result); err != ErrWait {
		t.Fatal("old observation accepted", err)
	}
	v.State.Seq++
	v.State.Pos = protocol.Vec3{3, 0, 0}
	if o, err = r.Next(v, result); err != nil || o != nil {
		t.Fatal(o, err)
	}
	r, v = restFixture(t, false)
	v.State.Pos = protocol.Vec3{3, 0, 0}
	if o, err = r.Next(v, nil); err != nil || o != nil {
		t.Fatal("already beside bed should wait without dispatch", o, err)
	}
}

func TestDayRestUsesReachableFloorBesideACrowdedBed(t *testing.T) {
	r, v := restFixture(t, false)
	v.State.Pos = protocol.Vec3{-32.97, 36.09, 16.62}
	home := protocol.Vec3{-35.9, 37.66, 6.53}
	v.Home = &home
	v.Waypoints[0].Pos = protocol.Vec3{-36.45, 36.09, 6.57}
	v.Waypoints[0].Radius = 14
	v.Perception.Structures[0].Pos = protocol.Vec3{-31.08, 36.11, 15.19}
	o, err := r.Next(v, nil)
	if err != nil || o == nil {
		t.Fatal(o, err)
	}
	a := o.Args.(*protocol.MoveToArgs)
	// The captured search reached this actual floor node, but the former
	// 1.25m arrival radius demanded climbing into the row of raised beds.
	closest := protocol.Vec3{-30.47, 36.09, 16.62}
	if a.StopDistance == nil || distance(closest, v.Perception.Structures[0].Pos) > *a.StopDistance {
		t.Fatal(a)
	}
	v.State.Pos = closest
	v.State.Seq++
	if o, err := r.Next(v, &Result{Status: protocol.StatusSucceeded}); err != nil || o != nil {
		t.Fatal(o, err)
	}
}

func TestDayRestDoesNotTreatALowerFloorAsAlreadyBesideBed(t *testing.T) {
	r, v := restFixture(t, false)
	v.State.Pos = protocol.Vec3{4, -1, 0}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskMoveTo {
		t.Fatal("lower floor bypassed observed arrival", o, err)
	}
	v.State.Seq++
	if _, err = r.Next(v, &Result{Status: protocol.StatusSucceeded}); err == nil {
		t.Fatal("lower floor earned a bedside wait")
	}
}
func TestRestNeverClaimsUnknownFreeTakenOrAnotherFloorBeds(t *testing.T) {
	for _, kind := range []string{"unknown", "free", "taken", "floor", "distant"} {
		t.Run(kind, func(t *testing.T) {
			r, v := restFixture(t, true)
			b := &v.Perception.Structures[0]
			switch kind {
			case "floor":
				b.Pos[1] = 2
			case "distant":
				b.Pos[0] = 15
			default:
				b.Mine = false
			}
			if kind == "taken" {
				owned := true
				b.Owned = &owned
			}
			if kind == "free" {
				owned := false
				b.Owned = &owned
			}
			if o, err := r.Next(v, nil); err == nil || o != nil {
				t.Fatal(o, err)
			}
		})
	}
}
func TestRestGuardsFreshnessHomeModeOwnershipThreatsAndFloor(t *testing.T) {
	for _, mode := range []string{"state", "perception", "clock", "night", "air", "shelter", "wet", "threat", "home", "marker", "bed", "floor", "nan", "injury"} {
		t.Run(mode, func(t *testing.T) {
			r, v := restFixture(t, false)
			if _, err := r.Next(v, nil); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "state":
				v.StateAt = v.Now.Add(-4 * time.Second)
			case "perception":
				v.PerceptionAt = v.Now.Add(-4 * time.Second)
			case "clock":
				v.State.Time.Day = 0
			case "night":
				v.State.Time.Night = true
			case "air":
				v.State.Flags = []string{protocol.FlagInShelter}
			case "shelter":
				v.State.Flags = []string{protocol.FlagGrounded}
			case "wet":
				v.State.Flags = append(v.State.Flags, protocol.FlagSwimming)
			case "threat":
				v.Perception.Characters = []protocol.PerceivedCharacter{{Hostile: true, HPFrac: 1, Pos: v.State.Pos}}
			case "home":
				v.Home = &protocol.Vec3{1, 0, 0}
			case "marker":
				v.Waypoints[0].Pos[0]++
			case "bed":
				v.Perception.Structures[0].Mine = false
			case "floor":
				v.Perception.Structures[0].Pos[1] = 3
			case "nan":
				v.State.Pos[0] = math.NaN()
			case "injury":
				v.State.HP -= 5
			}
			if err := r.Check(v); err == nil {
				t.Fatal("guard admitted", mode)
			}
		})
	}
}
func TestSleepingBodiesNeverQueueIdleChores(t *testing.T) {
	for _, flag := range []string{"in_bed", protocol.FlagSleeping} {
		f, w := idleSetup(t)
		i := f.e.instances["idle"]
		v := View{State: &protocol.State{World: "test", Phase: protocol.PhaseInWorld, HP: 25, Flags: []string{flag}}, Idle: true, Now: f.now, StateAt: f.now, Waypoints: []home.Waypoint{*w}}
		i.idleSince = f.now.Add(-time.Minute)
		f.e.idleTrigger("idle", i, v, v.Now)
		if i.pending || !i.idleSince.IsZero() {
			t.Fatal("sleeping body queued idle work", flag)
		}
	}
}
func TestRestDefinitionRejectsAMismatchedIdleMarker(t *testing.T) {
	d := MarkerRestDefinition(Marker{World: "one", Name: "work", Kind: "cooking"}, true)
	d.Trigger.Marker = &Marker{World: "one", Name: "other", Kind: "cooking"}
	if err := validate(d, map[string]Template{"marker_rest": MarkerRest{}}); err == nil {
		t.Fatal("mismatched marker admitted")
	}
	if err := (MarkerRest{}).Validate(json.RawMessage(`{"marker":{},"typo":true}`)); err == nil {
		t.Fatal("unknown fields admitted")
	}
}
