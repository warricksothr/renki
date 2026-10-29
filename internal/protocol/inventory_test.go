package protocol

import (
	"slices"
	"testing"
)

func TestInventoryCountsAndRoom(t *testing.T) {
	inv := &Inventory{Size: [2]int{2, 2}, Items: []InventoryItem{
		{Slot: [2]int{0, 0}, Prefab: "AxeBronze", Stack: 1},
		{Slot: [2]int{1, 0}, Prefab: "Wood", Stack: 42, StackMax: 50},
		{Slot: [2]int{0, 1}, Prefab: "Wood", Stack: 50, StackMax: 50},
	}}
	if got := inv.Count("Wood"); got != 92 {
		t.Errorf("Count(Wood) = %d; want 92", got)
	}
	if got := inv.Free(); got != 1 {
		t.Errorf("Free = %d; want 1", got)
	}
	// 8 more on the partial stack, and a full 50 in the empty slot.
	if got := inv.Room("Wood", 0); got != 58 {
		t.Errorf("Room(Wood) = %d; want 58", got)
	}
	if got := inv.Room("Stone", 50); got != 50 {
		t.Errorf("Room(Stone, 50) = %d; want 50", got)
	}
	if got := inv.Room("Stone", 0); got != 1 {
		t.Errorf("Room(Stone, unknown) = %d; want 1", got)
	}
}

func TestEquipableIsWhatEquipCanActOn(t *testing.T) {
	inv := &Inventory{Items: []InventoryItem{
		{Prefab: "AxeBronze", Type: "OneHandedWeapon", Equipped: true, Durability: 100, DurabilityMax: 200},
		{Prefab: "AxeBronze", Type: "OneHandedWeapon", Durability: 200, DurabilityMax: 200}, // a spare of what it holds
		{Prefab: "Club", Type: "OneHandedWeapon", Durability: 0, DurabilityMax: 100},        // broken
		{Prefab: "ShieldWood", Type: "Shield", Durability: 50, DurabilityMax: 100},
		{Prefab: "ShieldWood", Type: "Shield", Durability: 80, DurabilityMax: 100},
		{Prefab: "ArrowWood", Type: "Ammo"},
		{Prefab: "Wood", Type: "Material", Stack: 20},
		{Prefab: "AtgeirAttach", Type: "Attach_Atgeir"},
	}}
	if got := inv.Equipable(); !slices.Equal(got, []string{"ShieldWood", "ArrowWood"}) {
		t.Errorf("equipable %v", got)
	}
}
