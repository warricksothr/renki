package behaviour

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestEngineDefendsAnAttributedNativeHarvestRefusalWithoutAnEntityField(t *testing.T) {
	f := setup(t, protocol.TaskHarvest, protocol.TaskAttack)
	_, v := gathering(t)
	f.p.Send(v.State)
	f.p.Send(v.Inventory)
	f.p.Send(v.Perception)
	wait(t, func() bool { return f.s.Inventory() != nil && f.s.Perception() != nil && f.s.State().Food != nil })
	d := def()
	d.Template = "gather"
	d.Params = json.RawMessage(`{"site":{"world":"test","pos":[0,0,0],"radius":30},"to":["a"],"defend":true}`)
	f.put(t, "gather", d)
	f.act(t, "gather", "start")
	f.tick()
	cut := f.p.NextTask()
	if cut.Task != protocol.TaskHarvest {
		t.Fatal(cut)
	}
	gatherPursuer(&v)
	f.p.Send(v.Perception)
	wait(t, func() bool { return len(f.s.Perception().Characters) == 1 })
	// Captured native failure: re/status/reason/detail, no entity. The engine
	// must use its outstanding task correlation, not a made-up native binding.
	f.done(t, cut, protocol.StatusFailed, protocol.ReasonRefused)
	f.tick()
	attack := f.p.NextTask()
	var args protocol.AttackArgs
	if err := json.Unmarshal(attack.Args, &args); err != nil || attack.Task != protocol.TaskAttack || args.Entity != "pursuer" || !args.Kite {
		t.Fatal(attack, err)
	}
	r := f.e.instances["gather"].runner.(*gatherRun)
	if r.resume == nil || r.resume.ID != "tree" || r.pending != nil || r.completed["tree"] || r.trees != 1 || r.logs != 0 {
		t.Fatal(r)
	}
}

func gatherPursuer(v *View) {
	v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "pursuer", Kind: protocol.CharacterCreature,
		Prefab: "Greyling", Hostile: true, TargetingMe: true, HPFrac: 1, Pos: protocol.Vec3{12, 0, 0}}}
}

func defenseSuccess() *Result {
	return &Result{Status: protocol.StatusSucceeded, Entity: "pursuer", Detail: "observed Greyling death"}
}

func TestGatherAtItsSiteCanDefendBeforeStartingPreparationOrCutting(t *testing.T) {
	for _, nearSite := range []bool{true, false} {
		r, v := gathering(t)
		r.p.Defend = true
		gatherPursuer(&v)
		if !nearSite {
			r.p.Site.Pos[0] = 100
		}
		order, err := r.Next(v, nil)
		if !nearSite {
			if order != nil || err == nil || r.defenses != 0 {
				t.Fatal("work-site defense admitted a separate distant fight", order, err)
			}
			continue
		}
		if err != nil || order == nil || order.Task != protocol.TaskAttack || r.trees != 0 || r.prepared || r.arrived || r.keep["Wood"] != 12 {
			t.Fatal(order, err, r)
		}
		finishDefense(t, r, &v)
		nextTask(t, r, v, defenseSuccess(), protocol.TaskHarvest)
		if r.trees != 1 || !r.prepared || r.keep["Wood"] != 12 {
			t.Fatal(r)
		}
	}
}

func TestGatherKeepsItsUnfinishedChainWhileAConfirmedPursuerApproaches(t *testing.T) {
	r, v := gathering(t)
	r.p.Defend = true
	nextTask(t, r, v, nil, protocol.TaskHarvest)
	gatherPursuer(&v)
	v.Perception.Characters[0].Pos[0] = 25
	log := gatherResource("unfinished", protocol.ResourceLog, 6)
	order, err := gatherFinishedTarget(t, r, &v, 0, log)
	if order != nil || !errors.Is(err, ErrWait) || !r.completed["tree"] || !r.chain[log.ID] || r.defenses != 0 {
		t.Fatal(order, err, r)
	}
	v.Now = v.Now.Add(5 * time.Second)
	v.Perception.Seq++
	v.Perception.Characters[0].Pos[0] = 19
	nextTask(t, r, v, nil, protocol.TaskAttack)
	finishDefense(t, r, &v)
	order = nextTask(t, r, v, defenseSuccess(), protocol.TaskHarvest)
	if order.Args.(*protocol.HarvestArgs).Entity != log.ID || r.trees != 1 || r.keep["Wood"] != 12 {
		t.Fatal(order, r)
	}
}

func TestGatherPursuerWaitCannotExtendOrSubstituteItsTarget(t *testing.T) {
	for _, mode := range []string{"timeout", "changed", "multiple", "injured", "unconfirmed"} {
		t.Run(mode, func(t *testing.T) {
			r, v := gathering(t)
			r.p.Defend = true
			r.arrived = true
			r.began = v.Now
			gatherPursuer(&v)
			v.Perception.Characters[0].Pos[0] = 25
			if order, err := r.defend(v); order != nil || !errors.Is(err, ErrWait) {
				t.Fatal(order, err)
			}
			v.Now = v.Now.Add(16 * time.Second)
			switch mode {
			case "changed":
				v.Perception.Characters[0].ID = "substitute"
			case "multiple":
				other := v.Perception.Characters[0]
				other.ID = "second"
				v.Perception.Characters = append(v.Perception.Characters, other)
			case "injured":
				v.State.HP = 10
			case "unconfirmed":
				v.Perception.Characters[0].TargetingMe = false
			}
			order, err := r.defend(v)
			if order != nil || errors.Is(err, ErrWait) || mode != "unconfirmed" && err == nil || r.defenses != 0 {
				t.Fatal(order, err, r)
			}
		})
	}
}

func finishDefense(t *testing.T, r *gatherRun, v *View) {
	t.Helper()
	if _, err := r.Next(*v, defenseSuccess()); !errors.Is(err, ErrWait) {
		t.Fatal(err)
	}
	v.State.Seq++
	v.Perception.Seq++
	v.Perception.Characters = nil
}

func TestGatherDefensePreservesFelledTreeChainAndGains(t *testing.T) {
	r, v := gathering(t)
	r.p.Defend = true
	nextTask(t, r, v, nil, protocol.TaskHarvest)
	gatherPursuer(&v)
	log := gatherResource("new log", protocol.ResourceLog, 6)
	order, err := gatherFinishedTarget(t, r, &v, 0, log)
	if err != nil || order == nil || order.Task != protocol.TaskAttack {
		t.Fatal(order, err)
	}
	args := order.Args.(*protocol.AttackArgs)
	if args.Entity != "pursuer" || !args.Kite || args.Style != "melee" || order.TimeoutS != 180 {
		t.Fatal(args, order)
	}
	finishDefense(t, r, &v)
	order = nextTask(t, r, v, defenseSuccess(), protocol.TaskHarvest)
	if order.Args.(*protocol.HarvestArgs).Entity != log.ID || !r.completed["tree"] || r.trees != 1 || r.logs != 0 || r.keep["Wood"] != 12 {
		t.Fatal(r, order)
	}
	a, b := gatherResource("a", protocol.ResourceLog, 7), gatherResource("b", protocol.ResourceLog, 8)
	if _, err := gatherFinishedTarget(t, r, &v, 0, a, b); err != nil {
		t.Fatal(err)
	}
	if _, err := gatherFinishedTarget(t, r, &v, 10, b); err != nil {
		t.Fatal(err)
	}
	order, err = gatherFinishedTarget(t, r, &v, 10)
	if err != nil || order.Task != protocol.TaskDeposit || order.Args.(*protocol.TransferArgs).Items[0].Count != 20 || r.logs != 3 {
		t.Fatal(order, err, r.logs)
	}
}

func TestGatherRefusedCutDefendsThenRetriesExactResourceWithoutCompletion(t *testing.T) {
	r, v := gathering(t)
	r.p.Defend = true
	nextTask(t, r, v, nil, protocol.TaskHarvest)
	gatherPursuer(&v)
	refused := &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonRefused, Entity: "tree"}
	if !r.Recover(*refused) {
		t.Fatal("refusal was not recoverable")
	}
	nextTask(t, r, v, refused, protocol.TaskAttack)
	if r.completed["tree"] || r.pending != nil || r.resume.ID != "tree" || r.trees != 1 {
		t.Fatal(r)
	}
	finishDefense(t, r, &v)
	order := nextTask(t, r, v, defenseSuccess(), protocol.TaskHarvest)
	if order.Args.(*protocol.HarvestArgs).Entity != "tree" || r.targets != 2 || r.trees != 1 || r.completed["tree"] {
		t.Fatal(order, r)
	}
	order = finishGatherTree(t, r, &v, "one")
	if order.Task != protocol.TaskDeposit || r.keep["Wood"] != 12 {
		t.Fatal(order, r.keep)
	}
}

func TestGatherDefenseRequiresExactDeathAndSubsequentFreshObservations(t *testing.T) {
	for _, mode := range []string{"failed", "wrong entity", "disappeared", "stale", "same pose", "same perception", "alive", "injured", "missing resource"} {
		t.Run(mode, func(t *testing.T) {
			r, v := gathering(t)
			r.p.Defend = true
			nextTask(t, r, v, nil, protocol.TaskHarvest)
			gatherPursuer(&v)
			nextTask(t, r, v, &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonRefused, Entity: "tree"}, protocol.TaskAttack)
			result := defenseSuccess()
			switch mode {
			case "failed":
				result.Status = protocol.StatusFailed
			case "wrong entity":
				result.Entity = "other"
			case "disappeared":
				result.Detail = "enemy disappeared"
			}
			if mode == "failed" || mode == "wrong entity" || mode == "disappeared" {
				if order, err := r.Next(v, result); order != nil || err == nil || errors.Is(err, ErrWait) {
					t.Fatal(order, err)
				}
				return
			}
			if _, err := r.Next(v, result); !errors.Is(err, ErrWait) {
				t.Fatal(err)
			}
			v.State.Seq++
			v.Perception.Seq++
			if mode != "alive" {
				v.Perception.Characters = nil
			}
			switch mode {
			case "stale":
				v.State.T += 6
			case "same pose":
				v.State.Seq--
			case "same perception":
				v.Perception.Seq--
			case "injured":
				v.State.HP--
			case "missing resource":
				v.Perception.Resources = nil
			}
			order, err := r.Next(v, result)
			if order != nil || err == nil {
				t.Fatal(order, err)
			}
			if mode != "injured" && mode != "missing resource" && !errors.Is(err, ErrWait) {
				t.Fatal(err)
			}
		})
	}
}

func TestGatherRefusalFailsClosedWithoutSafeOptedInPursuer(t *testing.T) {
	for _, mode := range []string{"default", "unconfirmed", "starred", "two", "stamina", "food", "axe", "bystander", "budget", "count", "wrong receipt", "uncorrelated receipt", "unreachable"} {
		t.Run(mode, func(t *testing.T) {
			r, v := gathering(t)
			r.p.Defend = true
			nextTask(t, r, v, nil, protocol.TaskHarvest)
			gatherPursuer(&v)
			result := &Result{Status: protocol.StatusFailed, Reason: protocol.ReasonRefused, Entity: "tree"}
			switch mode {
			case "default":
				r.p.Defend = false
			case "unconfirmed":
				v.Perception.Characters[0].TargetingMe = false
			case "starred":
				v.Perception.Characters[0].Stars = 1
			case "two":
				c := v.Perception.Characters[0]
				c.ID = "other"
				v.Perception.Characters = append(v.Perception.Characters, c)
			case "stamina":
				v.State.Stamina = 39
			case "food":
				v.State.Food = nil
			case "axe":
				v.Inventory.Items[0].Durability = 24
			case "bystander":
				v.Perception.Characters = append(v.Perception.Characters, protocol.PerceivedCharacter{ID: "owner", HPFrac: 1})
			case "budget":
				v.Now = v.Now.Add(5 * time.Minute)
			case "count":
				r.defenses = 2
			case "wrong receipt":
				result.Entity = "other"
			case "uncorrelated receipt":
				result.Entity = ""
			case "unreachable":
				result.Reason = protocol.ReasonUnreachable
			}
			if order, err := r.Next(v, result); order != nil || err == nil || errors.Is(err, ErrWait) {
				t.Fatal(order, err)
			}
			if r.completed["tree"] || r.trees != 1 || r.logs != 0 {
				t.Fatal(r)
			}
		})
	}
}
