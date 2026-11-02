package goal

import (
	"errors"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

func acquireFixture(t *testing.T) (*Template, *runner, behaviour.View, string) {
	t.Helper()
	h, r, v, id := foodFixture(t)
	if err := h.Store.update(id, func(g *Record) error { g.Food.Phase = "raw"; g.Food.SearchPinned = true; return nil }); err != nil {
		t.Fatal(err)
	}
	v.State.WeightMax = 300
	v.State.Stamina = 80
	return h, r, v, id
}
func forageResource(id, item string, pos protocol.Vec3) protocol.PerceivedResource {
	return protocol.PerceivedResource{ID: id, Kind: protocol.ResourcePickable, Prefab: "Pickable_" + item, Pos: pos, Yield: &protocol.ResourceYield{Prefab: item, Count: 3, StackMax: 50, Weight: .1}}
}
func acquireResult(t *testing.T, r *runner, v *behaviour.View, id string, status, reason string) *behaviour.Result {
	t.Helper()
	if err := r.Dispatched(id); err != nil {
		t.Fatal(err)
	}
	foodFresh(v)
	v.Perception.Seq += 10
	return &behaviour.Result{TaskID: id, Status: status, Reason: reason}
}
func TestFoodForageSpawnPickupThenVerifiedReturnAndCooking(t *testing.T) {
	h, r, v, id := acquireFixture(t)
	source := forageResource("raw-source", "NeckTail", protocol.Vec3{20, 0, 0})
	// Raw meat is admitted only as an observed drop, not an invented animal yield.
	source.Kind = protocol.ResourceDrop
	v.Perception.Resources = []protocol.PerceivedResource{source}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskPickup {
		t.Fatal(o, err)
	}
	last := acquireResult(t, r, &v, "pickup", protocol.StatusSucceeded, "")
	v.State.Pos = source.Pos
	v.Inventory.Items = []protocol.InventoryItem{{Prefab: "NeckTail", Stack: 3}}
	v.Perception.Resources = nil
	o, err = r.Next(v, last)
	if err != nil || o == nil || o.Task != protocol.TaskMoveTo {
		t.Fatal(o, err)
	}
	g, _ := h.Store.Get(id)
	if len(g.Food.Receipts) != 1 || g.Food.Receipts[0].Count != 3 || g.Food.Acquisition.Targets != 1 {
		t.Fatal(g)
	}
	last = acquireResult(t, r, &v, "return", protocol.StatusSucceeded, "")
	v.State.Pos = protocol.Vec3{}
	o, err = r.Next(v, last)
	if err != nil || o == nil || o.Task != protocol.TaskInteract || o.Args.(*protocol.InteractArgs).Cook != "NeckTail" {
		t.Fatal(o, err)
	}
}
func TestFoodPickableWaitsForFreshProductAndMeasuresPartialPickup(t *testing.T) {
	h, r, v, id := acquireFixture(t)
	source := forageResource("bush", "Raspberry", protocol.Vec3{20, 0, 0})
	v.Perception.Resources = []protocol.PerceivedResource{source}
	o, err := r.Next(v, nil)
	if err != nil || o.Task != protocol.TaskInteract {
		t.Fatal(o, err)
	}
	last := acquireResult(t, r, &v, "interact", protocol.StatusSucceeded, "")
	v.State.Pos = source.Pos
	o, err = r.Next(v, last)
	var delay *behaviour.Delay
	if o != nil || !errors.As(err, &delay) {
		t.Fatal(o, err)
	}
	foodFresh(&v)
	drop := source
	drop.ID = "drop"
	drop.Kind = protocol.ResourceDrop
	v.Perception.Resources = []protocol.PerceivedResource{drop}
	o, err = r.Next(v, nil)
	if err != nil || o == nil || o.Args.(*protocol.PickupArgs).Entity != "drop" {
		t.Fatal(o, err)
	}
	last = acquireResult(t, r, &v, "pickup", protocol.StatusSucceeded, "")
	v.Inventory.Items = []protocol.InventoryItem{{Prefab: "Raspberry", Stack: 1}}
	o, err = r.Next(v, last)
	if err != nil || o == nil || o.Task != protocol.TaskMoveTo {
		t.Fatal(o, err)
	}
	g, _ := h.Store.Get(id)
	if g.Food.Receipts[0].Count != 1 || g.Food.Acquisition.Targets != 2 {
		t.Fatal(g)
	}
}
func TestFoodCannotClaimYieldFromSuccessWithoutPackGain(t *testing.T) {
	h, r, v, id := acquireFixture(t)
	v.Perception.Resources = []protocol.PerceivedResource{forageResource("bush", "Raspberry", protocol.Vec3{})}
	o, err := r.Next(v, nil)
	if err != nil || o == nil {
		t.Fatal(o, err)
	}
	last := acquireResult(t, r, &v, "interact", protocol.StatusSucceeded, "")
	o, err = r.Next(v, last)
	var delay *behaviour.Delay
	if o != nil || !errors.As(err, &delay) {
		t.Fatal(o, err)
	}
	v.Now = v.Now.Add(10 * time.Second)
	v.StateAt = v.Now
	v.PerceptionAt = v.Now
	foodFresh(&v)
	o, err = r.Next(v, nil)
	g, _ := h.Store.Get(id)
	if err != nil || o != nil || g.Status != "needs_plan" || len(g.Food.Receipts) != 0 || g.Food.Acquisition.Targets != 1 {
		t.Fatal(o, err, g)
	}
}
func TestFoodAutoPickupRaceRequiresMeasuredGainAndExactTask(t *testing.T) {
	for _, wrong := range []bool{false, true} {
		t.Run(map[bool]string{false: "auto", true: "wrong-task"}[wrong], func(t *testing.T) {
			h, r, v, id := acquireFixture(t)
			p := forageResource("raw", "NeckTail", protocol.Vec3{})
			p.Kind = protocol.ResourceDrop
			v.Perception.Resources = []protocol.PerceivedResource{p}
			o, err := r.Next(v, nil)
			if err != nil || o == nil {
				t.Fatal(o, err)
			}
			last := acquireResult(t, r, &v, "ours", protocol.StatusRejected, protocol.ReasonInvalidTarget)
			v.Inventory.Items = []protocol.InventoryItem{{Prefab: "NeckTail", Stack: 3}}
			if wrong {
				last.TaskID = "other"
			}
			o, err = r.Next(v, last)
			g, _ := h.Store.Get(id)
			if wrong {
				if o != nil || g.Status != "needs_plan" || len(g.Food.Receipts) > 0 {
					t.Fatal(o, err, g)
				}
			} else if err != nil || o == nil || o.Task != protocol.TaskInteract || len(g.Food.Receipts) != 1 {
				t.Fatal(o, err, g)
			}
		})
	}
}
func TestFoodForageNeverFinishesBelowHomeFloor(t *testing.T) {
	h, r, v, id := acquireFixture(t)
	v.Perception.Resources = []protocol.PerceivedResource{forageResource("bush", "Raspberry", protocol.Vec3{20, 0, 0})}
	_, _ = r.Next(v, nil)
	last := acquireResult(t, r, &v, "take", protocol.StatusSucceeded, "")
	v.State.Pos = protocol.Vec3{20, 0, 0}
	v.Inventory.Items = []protocol.InventoryItem{{Prefab: "Raspberry", Stack: 3}}
	o, err := r.Next(v, last)
	if err != nil || o == nil || o.Task != protocol.TaskMoveTo {
		t.Fatal(o, err)
	}
	last = acquireResult(t, r, &v, "home", protocol.StatusSucceeded, "")
	v.State.Pos = protocol.Vec3{0, -2, 0}
	o, err = r.Next(v, last)
	g, _ := h.Store.Get(id)
	if o != nil || err != nil || g.Status != "needs_plan" || g.Pending == nil {
		t.Fatal(o, err, g)
	}
}
func TestFoodForageSitesAreHintsAndFailedVisitsStaySpent(t *testing.T) {
	h, r, v, id := acquireFixture(t)
	sites := []home.Landmark{{Entity: "one", Kind: protocol.ResourcePickable, Prefab: "RaspberryBush", Pos: protocol.Vec3{20, 0, 0}, At: v.Now}, {Entity: "alias", Kind: protocol.ResourcePickable, Prefab: "RaspberryBush", Pos: protocol.Vec3{21, 0, 0}, At: v.Now}, {Entity: "two", Kind: protocol.ResourcePickable, Prefab: "Pickable_Mushroom", Pos: protocol.Vec3{40, 0, 0}, At: v.Now}}
	h.Landmarks = func(string) []home.Landmark { return sites }
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskMoveTo {
		t.Fatal(o, err)
	}
	last := acquireResult(t, r, &v, "visit", protocol.StatusFailed, protocol.ReasonUnreachable)
	o, err = r.Next(v, last)
	if err != nil || o == nil || *o.Args.(*protocol.MoveToArgs).To.Pos != sites[2].Pos {
		t.Fatal(o, err)
	}
	g, _ := h.Store.Get(id)
	if len(g.Food.Acquisition.Visits) != 2 || len(g.Food.Receipts) != 0 {
		t.Fatal(g)
	}
	// An interrupted trip remains uncertain after a real store reload.
	if err = r.Dispatched("uncertain"); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(h.Store.path)
	if err != nil {
		t.Fatal(err)
	}
	host := Template{Store: reopened}
	if err = host.Queue(id); err == nil {
		t.Fatal("replayed uncertain visit")
	}
}
func TestFoodHuntUsesCurrentPermissionAndExactDeathThenSeparateLoot(t *testing.T) {
	for _, mode := range []string{"allowed", "revoked", "tamed", "starred", "multiple", "wrong-death", "missing-entity"} {
		t.Run(mode, func(t *testing.T) {
			h, r, v, id := acquireFixture(t)
			h.Allowed = func(string) bool { return mode != "revoked" }
			c := protocol.PerceivedCharacter{ID: "boar", Kind: protocol.CharacterCreature, Prefab: "Boar", Pos: protocol.Vec3{20, 0, 0}, HPFrac: 1, Hostile: true}
			if mode == "tamed" {
				c.Kind = protocol.CharacterTamed
			}
			if mode == "starred" {
				c.Stars = 1
			}
			v.Perception.Characters = []protocol.PerceivedCharacter{c}
			if mode == "multiple" {
				other := c
				other.ID = "other"
				v.Perception.Characters = append(v.Perception.Characters, other)
			}
			o, err := r.Next(v, nil)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "revoked" || mode == "tamed" || mode == "starred" || mode == "multiple" {
				if o != nil {
					t.Fatal(o)
				}
				return
			}
			if o == nil || o.Task != protocol.TaskAttack || o.Args.(*protocol.AttackArgs).Entity != "boar" {
				t.Fatal(o)
			}
			last := acquireResult(t, r, &v, "hunt", protocol.StatusSucceeded, "")
			last.Entity = "boar"
			last.Detail = "observed Boar death"
			v.Perception.Characters = nil
			if mode == "wrong-death" {
				last.Detail = "target disappeared"
			}
			if mode == "missing-entity" {
				last.Entity = ""
			}
			o, err = r.Next(v, last)
			g, _ := h.Store.Get(id)
			if mode != "allowed" {
				if err != nil || o != nil || g.Status != "needs_plan" || len(g.Food.Receipts) > 0 {
					t.Fatal(o, err, g)
				}
				return
			}
			var delay *behaviour.Delay
			if o != nil || !errors.As(err, &delay) || g.Food.Acquisition.Hunts != 1 || len(g.Food.Receipts) != 1 || g.Food.Receipts[0].Action != "hunt" {
				t.Fatal(o, err, g)
			}
			foodFresh(&v)
			p := forageResource("loot", "RawMeat", c.Pos)
			p.Kind = protocol.ResourceDrop
			v.Perception.Resources = []protocol.PerceivedResource{p}
			o, err = r.Next(v, nil)
			if err != nil || o == nil || o.Task != protocol.TaskPickup {
				t.Fatal(o, err)
			}
		})
	}
}
func TestFoodAcquisitionCannotDispatchWithoutCheckpointOrAdmitUnknownYields(t *testing.T) {
	for _, mode := range []string{"missing", "capacity", "outside", "unsafe", "write"} {
		t.Run(mode, func(t *testing.T) {
			h, r, v, _ := acquireFixture(t)
			p := forageResource("bush", "Raspberry", protocol.Vec3{})
			switch mode {
			case "missing":
				p.Yield = nil
			case "capacity":
				v.State.WeightMax = .1
			case "outside":
				p.Pos[0] = 100
			case "unsafe":
				v.Perception.Characters = []protocol.PerceivedCharacter{{ID: "enemy", Kind: protocol.CharacterCreature, HPFrac: 1, Pos: protocol.Vec3{1, 0, 0}, Hostile: true}}
			case "write":
				h.Store.write = func([]byte) error { return errors.New("disk") }
			}
			v.Perception.Resources = []protocol.PerceivedResource{p}
			o, err := r.Next(v, nil)
			if o != nil {
				t.Fatal(o, err)
			}
			if mode == "write" && err == nil {
				t.Fatal("checkpoint failure swallowed")
			}
		})
	}
}
func TestOldFoodCheckpointDoesNotGainNewAcquisitionScope(t *testing.T) {
	h, r, v, id := acquireFixture(t)
	if err := h.Store.update(id, func(g *Record) error { g.Food.Acquisition = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	v.Perception.Resources = []protocol.PerceivedResource{forageResource("bush", "Raspberry", protocol.Vec3{})}
	o, err := r.Next(v, nil)
	g, _ := h.Store.Get(id)
	if err != nil || o != nil || g.Status != "needs_plan" {
		t.Fatal(o, err, g)
	}
}

func TestFoodFuelShortageCollectsObservedWoodBeforeNativeCooking(t *testing.T) {
	h, r, v, id := acquireFixture(t)
	v.Inventory.Items = []protocol.InventoryItem{{Prefab: "NeckTail", Stack: 1}}
	v.Perception.Structures[2].Cooking.Heated = false
	p := forageResource("branch", "Wood", protocol.Vec3{4, 0, 0})
	p.Yield.Count = 1
	v.Perception.Resources = []protocol.PerceivedResource{p}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Task != protocol.TaskInteract || o.Args.(*protocol.InteractArgs).Cook != "" {
		t.Fatal(o, err)
	}
	last := acquireResult(t, r, &v, "wood", protocol.StatusSucceeded, "")
	v.Inventory.Items = append(v.Inventory.Items, protocol.InventoryItem{Prefab: "Wood", Stack: 1})
	v.Perception.Resources = nil
	o, err = r.Next(v, last)
	g, _ := h.Store.Get(id)
	if err != nil || o == nil || o.Task != protocol.TaskInteract || o.Args.(*protocol.InteractArgs).Cook != "NeckTail" || len(g.Food.Receipts) != 1 || g.Food.Receipts[0].Item != "Wood" {
		t.Fatal(o, err, g)
	}
}
func TestFoodAcquisitionExhaustedBudgetsCannotResetOnResume(t *testing.T) {
	h, r, v, id := acquireFixture(t)
	if err := h.Store.update(id, func(g *Record) error { g.Food.Acquisition.Targets = 12; return nil }); err != nil {
		t.Fatal(err)
	}
	v.Perception.Resources = []protocol.PerceivedResource{forageResource("bush", "Raspberry", protocol.Vec3{})}
	o, err := r.Next(v, nil)
	if err != nil || o != nil {
		t.Fatal(o, err)
	}
	if err = h.Queue(id); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"id":"` + id + `"}`)
	run, err := h.Start(raw)
	if err != nil {
		t.Fatal(err)
	}
	o, err = run.Next(v, nil)
	g, _ := h.Store.Get(id)
	if err != nil || o != nil || g.Food.Acquisition.Targets != 12 || g.Status != "needs_plan" {
		t.Fatal(o, err, g)
	}
}
func TestFoodProductWaitKeepsOriginalRunAndBoundsAmbiguousGain(t *testing.T) {
	for _, extra := range []bool{false, true} {
		t.Run(map[bool]string{false: "stale-pack", true: "excess-gain"}[extra], func(t *testing.T) {
			h, r, v, id := acquireFixture(t)
			p := forageResource("source", "NeckTail", protocol.Vec3{})
			p.Kind = protocol.ResourceDrop
			v.Perception.Resources = []protocol.PerceivedResource{p}
			_, _ = r.Next(v, nil)
			last := acquireResult(t, r, &v, "take", protocol.StatusSucceeded, "")
			g, _ := h.Store.Get(id)
			v.Inventory.Seq = g.Food.BeforePackSeq
			count := 3
			if extra {
				count = 4
				v.Inventory.Seq++
			}
			v.Inventory.Items = []protocol.InventoryItem{{Prefab: "NeckTail", Stack: count}}
			o, err := r.Next(v, last)
			g, _ = h.Store.Get(id)
			if o != nil || len(g.Food.Receipts) > 0 || g.Status == "succeeded" {
				t.Fatal(o, err, g)
			}
			if extra {
				if err != nil || g.Status != "needs_plan" {
					t.Fatal(err, g)
				}
			} else {
				var delay *behaviour.Delay
				if !errors.As(err, &delay) || g.Food.Acquisition.Product.Run != h.Store.runID {
					t.Fatal(err, g)
				}
			}
		})
	}
}

func TestFoodLostHuntCanSelectDifferentPreyButNeverReplayTheFirst(t *testing.T) {
	h, r, v, id := acquireFixture(t)
	h.Allowed = func(string) bool { return true }
	a := protocol.PerceivedCharacter{ID: "a", Kind: protocol.CharacterCreature, Prefab: "Boar", Pos: protocol.Vec3{20, 0, 0}, HPFrac: 1, Hostile: true}
	b := a
	b.ID = "b"
	b.Pos[0] = -20
	v.Perception.Characters = []protocol.PerceivedCharacter{a, b}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Args.(*protocol.AttackArgs).Entity != "a" {
		t.Fatal(o, err)
	}
	last := acquireResult(t, r, &v, "lost", protocol.StatusFailed, protocol.ReasonTargetLost)
	o, err = r.Next(v, last)
	g, _ := h.Store.Get(id)
	if err != nil || o == nil || o.Args.(*protocol.AttackArgs).Entity != "b" || g.Food.Acquisition.Hunts != 2 || len(g.Food.Receipts) > 0 {
		t.Fatal(o, err, g)
	}
}

func TestFoodReadyAfterCookingReturnsBeforeSeekingAnyAdditionalSources(t *testing.T) {
	h, r, v, id := acquireFixture(t)
	if err := h.Store.update(id, func(g *Record) error { g.Food.Reserve = 1; return nil }); err != nil {
		t.Fatal(err)
	}
	v.Inventory.Items = []protocol.InventoryItem{{Prefab: "NeckTail", Stack: 1}}
	// A tempting live source must not cause another trip after readiness.
	v.Perception.Resources = []protocol.PerceivedResource{forageResource("extra", "Raspberry", protocol.Vec3{20, 0, 0})}
	o, err := r.Next(v, nil)
	if err != nil || o == nil || o.Args.(*protocol.InteractArgs).Cook != "NeckTail" {
		t.Fatal(o, err)
	}
	last := acquireResult(t, r, &v, "cook", protocol.StatusSucceeded, "")
	last.Cooking = deliveryRows("cook", "NeckTailGrilled", "station")
	v.State.Pos = protocol.Vec3{5, 0, 0}
	v.Inventory.Items = []protocol.InventoryItem{{Prefab: "NeckTailGrilled", Stack: 1}}
	o, err = r.Next(v, last)
	if err != nil || o == nil || o.Task != protocol.TaskMoveTo || *o.Args.(*protocol.MoveToArgs).To.Pos != *v.Home {
		t.Fatal(o, err)
	}
	last = acquireResult(t, r, &v, "home", protocol.StatusSucceeded, "")
	v.State.Pos = *v.Home
	o, err = r.Next(v, last)
	g, _ := h.Store.Get(id)
	if err != nil || o != nil || g.Status != "succeeded" || g.TaskCount != 2 || g.Food.Batches != 1 {
		t.Fatal(o, err, g)
	}
}
