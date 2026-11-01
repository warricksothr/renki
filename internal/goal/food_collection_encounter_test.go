package goal

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func TestCollectionRecoveryRequiresTheExactSourceTaskAndJourney(t *testing.T) {
	for _, drop := range []bool{false, true} {
		for _, changed := range []string{"matching", "task", "journey_task", "journey_id", "old_run", "source", "opt_out"} {
			h, r, v, id := acquireFixture(t)
			h.Allowed = func(string) bool { return true }
			source := forageResource("source", "NeckTail", protocol.Vec3{20, 0, 0})
			if drop {
				source.Kind = protocol.ResourceDrop
			}
			v.Perception.Resources = []protocol.PerceivedResource{source}
			o, err := r.Next(v, nil)
			if err != nil || o == nil {
				t.Fatal(o, err)
			}
			if err = r.Dispatched("collection"); err != nil {
				t.Fatal(err)
			}
			v.State.Task = &protocol.TaskRef{ID: "collection", Task: o.Task, Status: protocol.StatusRunning}
			v.State.Journey = &protocol.JourneyStatus{ID: "collection", Task: o.Task, Mode: "defend", Attempts: 1}
			v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "enemy", Prefab: "Greyling", HPFrac: 1, Hostile: true, Pos: protocol.Vec3{10, 0, 0}}}
			n := uint64(111)
			v.InjurySeq, v.State.HP = &n, 45
			v.Injury = &session.InjuryStatus{Sequence: n, Damage: 5, Complete: true}
			if err = h.Store.update(id, func(g *Record) error {
				switch changed {
				case "task":
					v.State.Task.Task = protocol.TaskHarvest
				case "journey_task":
					v.State.Journey.Task = protocol.TaskMoveTo
				case "journey_id":
					v.State.Journey.ID = "previous"
				case "old_run":
					g.Pending.Run = "previous-controller"
				case "source":
					g.Pending.Entity = "another-source"
				case "opt_out":
					var args map[string]any
					_ = json.Unmarshal(g.Pending.Params, &args)
					args["recoverThreats"] = false
					g.Pending.Params, _ = json.Marshal(args)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if err = r.Check(v); (err == nil) != (changed == "matching") {
				t.Fatal(drop, changed, err)
			}
			g, _ := h.Store.Get(id)
			if g.TaskCount != 1 || g.Pending.TaskID != "collection" || len(g.Food.Receipts) != 0 {
				t.Fatal("guard changed progress", g)
			}
		}
	}
}

func TestCollectionRecoveryPreservesItsDamageBudgetThroughVerifiedReturn(t *testing.T) {
	for _, drop := range []bool{false, true} {
		h, r, v, id := acquireFixture(t)
		source := forageResource("source", "NeckTail", protocol.Vec3{20, 0, 0})
		if drop {
			source.Kind = protocol.ResourceDrop
		}
		v.Perception.Resources = []protocol.PerceivedResource{source}
		_, err := r.Next(v, nil)
		if err != nil {
			t.Fatal(err)
		}
		last := acquireResult(t, r, &v, "collected", protocol.StatusSucceeded, "")
		n := uint64(111)
		v.InjurySeq, v.State.HP, v.State.Pos = &n, 45, source.Pos
		v.Injury = &session.InjuryStatus{Sequence: n, Damage: 5, Complete: true}
		v.Inventory.Items = []protocol.InventoryItem{{Prefab: "NeckTail", Stack: 3}}
		o, err := r.Next(v, last)
		g, _ := h.Store.Get(id)
		if err != nil || o == nil || o.Task != protocol.TaskMoveTo || g.Pending.Template != "food_return" || len(g.Food.Receipts) != 1 {
			t.Fatal(o, g, err)
		}
		if g.Food.Receipts[0].TaskID != "collected" || g.Food.Receipts[0].Entity != source.ID || g.Food.Receipts[0].Count != 3 ||
			g.Food.TravelBudget.Injury != 0 || g.Food.TravelBudget.Damage != 0 || g.Food.TravelBudget.HP != 50 || *g.Food.Injury != 111 {
			t.Fatal("collection lost provenance or reset its damage allowance", g)
		}
	}
}

func TestDelayedCollectionReceiptSurvivesANewThreatOrIdleInjury(t *testing.T) {
	for _, incident := range []string{"threat", "injury", "excessive_gain", "stale_pack", "old_run"} {
		h, r, v, id := acquireFixture(t)
		v.Perception.Resources = []protocol.PerceivedResource{forageResource("bush", "Raspberry", protocol.Vec3{20, 0, 0})}
		if _, err := r.Next(v, nil); err != nil {
			t.Fatal(err)
		}
		last := acquireResult(t, r, &v, "forage", protocol.StatusSucceeded, "")
		_, err := r.Next(v, last)
		var delay *behaviour.Delay
		if !errors.As(err, &delay) {
			t.Fatal("did not wait for delayed RPC product", err)
		}
		foodFresh(&v)
		v.Inventory.Items = []protocol.InventoryItem{{Prefab: "Raspberry", Stack: 3}}
		v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "enemy", HPFrac: 1, Hostile: true, Pos: protocol.Vec3{10, 0, 0}}}
		switch incident {
		case "injury":
			n := uint64(111)
			v.InjurySeq, v.State.HP = &n, 48
			v.Injury = &session.InjuryStatus{Sequence: n, Damage: 2, Complete: true}
			v.Perception.Characters = nil
		case "excessive_gain":
			v.Inventory.Items[0].Stack = 4
		case "stale_pack":
			g, _ := h.Store.Get(id)
			v.Inventory.Seq = g.Food.Acquisition.Product.BeforePack
		case "old_run":
			if err = h.Store.update(id, func(g *Record) error { g.Food.Acquisition.Product.Run = "old-controller"; return nil }); err != nil {
				t.Fatal(err)
			}
		}
		if incident != "old_run" {
			if err = r.Check(v); err != nil {
				t.Fatal("cancelled product accounting before decision", incident, err)
			}
		}
		o, err := r.Next(v, nil)
		g, _ := h.Store.Get(id)
		if err != nil || o != nil || g.Status != "needs_plan" || g.TaskCount != 1 || g.Pending != nil {
			t.Fatal(incident, o, g, err)
		}
		valid := incident == "threat" || incident == "injury"
		if (len(g.Food.Receipts) == 1) != valid || (g.Food.Acquisition.Product == nil) != valid {
			t.Fatal(incident, g)
		}
		if valid && (g.Food.Receipts[0].TaskID != "forage" || g.Food.Receipts[0].Entity != "bush") {
			t.Fatal(g)
		}
	}
}

func TestUncertainNativeCollectionDoesNotConvertIncidentalGainIntoAReceipt(t *testing.T) {
	h, r, v, id := acquireFixture(t)
	v.Perception.Resources = []protocol.PerceivedResource{forageResource("bush", "Raspberry", protocol.Vec3{20, 0, 0})}
	if _, err := r.Next(v, nil); err != nil {
		t.Fatal(err)
	}
	last := acquireResult(t, r, &v, "uncertain-use", protocol.StatusFailed, protocol.ReasonStopped)
	v.Inventory.Items = []protocol.InventoryItem{{Prefab: "Raspberry", Stack: 3}}
	o, err := r.Next(v, last)
	g, _ := h.Store.Get(id)
	if err != nil || o != nil || g.Status != "needs_plan" || g.Pending == nil || g.Pending.TaskID != "uncertain-use" || len(g.Food.Receipts) != 0 {
		t.Fatal("uncertain game use became a collection receipt or replayable action", o, g, err)
	}
}

func TestFailedCollectionDoesNotAcknowledgeEncounterDamage(t *testing.T) {
	h, r, v, id := acquireFixture(t)
	v.Perception.Resources = []protocol.PerceivedResource{forageResource("bush", "Raspberry", protocol.Vec3{20, 0, 0})}
	if _, err := r.Next(v, nil); err != nil {
		t.Fatal(err)
	}
	last := acquireResult(t, r, &v, "failed-source", protocol.StatusFailed, protocol.ReasonTargetLost)
	n := uint64(111)
	v.InjurySeq, v.State.HP = &n, 45
	v.Injury = &session.InjuryStatus{Sequence: n, Damage: 5, Complete: true}
	o, err := r.Next(v, last)
	g, _ := h.Store.Get(id)
	if err != nil || o != nil || g.Status != "needs_plan" || *g.Food.Injury != 0 || len(g.Food.Receipts) != 0 {
		t.Fatal("failed collection renewed injury authority or invented progress", o, g, err)
	}
}
