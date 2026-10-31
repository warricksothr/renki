package chat

import (
	"testing"

	"github.com/warricksothr/renki/internal/protocol"
)

func cookingBrain() (*Brain, *fakeSession) {
	b, f, _ := newBrain(nil)
	f.allowed = append(f.allowed, protocol.TaskInteract, protocol.TaskPickup)
	f.state = &protocol.State{Phase: protocol.PhaseInWorld, Pos: protocol.Vec3{1, 30, 1}}
	f.inventory = &protocol.Inventory{Items: []protocol.InventoryItem{{Prefab: "NeckTail", Stack: 2}}}
	f.perception = &protocol.Perception{Structures: []protocol.PerceivedStructure{{ID: "1:1", Kind: protocol.StructureStation,
		Pos: protocol.Vec3{2, 30, 1}, Cooking: &protocol.CookingStatus{Supported: true, Heated: true,
			Slots: []protocol.CookingSlotStatus{{State: "empty"}, {Index: 1, State: "empty"}}}}}}
	return b, f
}

func TestPanelCookSelectsACarriedIngredientAndFreshEmptyStation(t *testing.T) {
	b, f := cookingBrain()
	// The horizontally closer station on another floor is outside 3D reach.
	other := f.perception.Structures[0]
	other.ID = "upstairs"
	other.Pos[1] = 60
	f.perception.Structures = append([]protocol.PerceivedStructure{other}, f.perception.Structures...)
	if err := b.FromPanel("Sothr", "cook food"); err != nil {
		t.Fatal(err)
	}
	orders := f.sent()
	if len(orders) != 1 {
		t.Fatal(orders)
	}
	args, ok := orders[0].Args.(*protocol.InteractArgs)
	if !ok || args.Entity != "1:1" || args.Cook != "NeckTail" || orders[0].TimeoutS != 180 {
		t.Fatal(orders[0])
	}
}

func TestCookMissingPrerequisitesDoesNotDisplaceTheCurrentBody(t *testing.T) {
	for _, cause := range []string{"ingredient", "busy", "cold", "unknown", "stale pack", "stale station", "pickup permission"} {
		t.Run(cause, func(t *testing.T) {
			b, f := cookingBrain()
			switch cause {
			case "ingredient":
				f.inventory.Items = nil
			case "busy":
				f.perception.Structures[0].Cooking.Slots[0] = protocol.CookingSlotStatus{Item: "RawMeat", State: "cooking"}
			case "cold":
				f.perception.Structures[0].Cooking.Heated = false
			case "unknown":
				f.perception.Structures[0].Cooking = nil
			case "stale pack":
				f.inventory.T = -100
			case "stale station":
				f.perception.T = -100
			case "pickup permission":
				f.allowed = []string{protocol.TaskInteract, protocol.TaskSay}
			}
			if err := b.FromPanel("Sothr", "cook food"); err != nil {
				t.Fatal(err)
			}
			if len(f.tasks()) != 0 {
				t.Fatal(f.sent())
			}
		})
	}
}
