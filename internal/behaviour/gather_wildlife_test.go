package behaviour

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func worksiteDeer(v *View) {
	v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "deer", Kind: protocol.CharacterCreature,
		Prefab: "Deer", Hostile: true, HPFrac: 1, Pos: protocol.Vec3{12, 0, 0}}}
}

func TestGatherMovesOnceThenRequiresFreshActualWildlifeClearance(t *testing.T) {
	r, v := gathering(t)
	worksiteDeer(&v)
	order := nextTask(t, r, v, nil, protocol.TaskMoveTo)
	args := order.Args.(*protocol.MoveToArgs)
	if args.To.Entity != "" || args.To.Pos == nil || *args.To.Pos != v.Perception.Characters[0].Pos ||
		args.StopDistance == nil || *args.StopDistance != 8 || args.Run == nil || !*args.Run ||
		!args.Dry || !args.AvoidHostiles || order.TimeoutS != 30 || r.targets != 0 || r.trees != 0 {
		t.Fatal(order, r)
	}
	if _, err := r.Next(v, success("clear")); !errors.Is(err, ErrWait) {
		t.Fatal("receipt alone authorized harvesting", err)
	}
	v.Perception.Characters = nil
	if _, err := r.Next(v, success("clear")); !errors.Is(err, ErrWait) {
		t.Fatal("stale observation authorized harvesting", err)
	}
	v.Perception.Seq++
	worksiteDeer(&v)
	if _, err := r.Next(v, success("clear")); !errors.Is(err, ErrWait) || r.wildlifeAttempts != 1 {
		t.Fatal("chased the deer instead of waiting", err, r.wildlifeAttempts)
	}
	v.Perception.Seq++
	v.Perception.Characters[0].Pos[0] = 60
	order = nextTask(t, r, v, success("clear"), protocol.TaskHarvest)
	if order.Args.(*protocol.HarvestArgs).Entity != "tree" || r.keep["Wood"] != 12 || r.wildlifeAttempts != 1 {
		t.Fatal(order, r)
	}
}

func TestGatherWildlifeClearancePreservesAllOtherExclusions(t *testing.T) {
	for _, mode := range []string{"player", "boar", "greyling", "targeting", "building", "wall", "far deer", "low stamina", "unknown stamina", "budget"} {
		t.Run(mode, func(t *testing.T) {
			r, v := gathering(t)
			worksiteDeer(&v)
			switch mode {
			case "player":
				v.Perception.Characters[0].Kind = protocol.CharacterPlayer
			case "boar":
				v.Perception.Characters[0].Prefab = "Boar"
			case "greyling":
				v.Perception.Characters[0].Prefab = "Greyling"
			case "targeting":
				v.Perception.Characters[0].TargetingMe = true
			case "building":
				v.Perception.Structures = []protocol.PerceivedStructure{{Kind: protocol.StructureContainer, Pos: protocol.Vec3{5, 0, 0}}}
			case "wall":
				v.Walls = []protocol.Wall{{A: protocol.Vec3{5, 0, -5}, B: protocol.Vec3{5, 0, 5}, H: 2}}
			case "far deer":
				v.Perception.Characters[0].Pos[0] = 26
			case "low stamina":
				v.State.Stamina = 39
			case "unknown stamina":
				v.State.Stamina = math.NaN()
			case "budget":
				r.wildlifeAttempts = r.p.treeLimit()
			}
			if order, err := r.Next(v, nil); order != nil || err == nil || r.targets != 0 {
				t.Fatal("unsafe clearance/harvest", order, err, r)
			}
		})
	}
}

func TestGatherWildlifeClearanceHasReceiptAndObservationDeadlines(t *testing.T) {
	for _, mode := range []string{"failed", "still present", "stale"} {
		t.Run(mode, func(t *testing.T) {
			r, v := gathering(t)
			worksiteDeer(&v)
			nextTask(t, r, v, nil, protocol.TaskMoveTo)
			result := success("clear")
			if mode == "failed" {
				result.Status = protocol.StatusFailed
			} else {
				if _, err := r.Next(v, result); !errors.Is(err, ErrWait) {
					t.Fatal(err)
				}
				v.Now = v.Now.Add(21 * time.Second)
				if mode == "still present" {
					v.Perception.Seq++
				}
			}
			if order, err := r.Next(v, result); order != nil || err == nil || errors.Is(err, ErrWait) || r.targets != 0 {
				t.Fatal(order, err, r)
			}
		})
	}
}

func TestCapturedBeechStandWithOnePassiveDeerGetsABoundedApproach(t *testing.T) {
	r, v := gathering(t)
	r.p.Return, r.p.ReturnHome, r.p.MaxTrees = false, true, 2
	r.p.Site.Pos = protocol.Vec3{94.03, 41.81, 44.71}
	v.State.Pos = protocol.Vec3{92.69, 41.64, 44.27}
	home := protocol.Vec3{-35.9, 37.66, 6.53}
	v.Home = &home
	worksiteDeer(&v)
	v.Perception.Characters[0].Pos = protocol.Vec3{105.21, 43.77, 47.4}
	v.Perception.Resources[0].Pos = r.p.Site.Pos
	order := nextTask(t, r, v, nil, protocol.TaskMoveTo)
	if *order.Args.(*protocol.MoveToArgs).To.Pos != v.Perception.Characters[0].Pos || r.targets != 0 {
		t.Fatal(order, r)
	}
}
