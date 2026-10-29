package protocol

import "slices"

// Count is how many of a prefab the bot carries, over every stack.
func (inv *Inventory) Count(prefab string) int {
	n := 0
	for _, it := range inv.Items {
		if it.Prefab == prefab {
			n += it.Stack
		}
	}
	return n
}

// Free is how many slots are empty.
func (inv *Inventory) Free() int {
	if n := inv.Size[0]*inv.Size[1] - len(inv.Items); n > 0 {
		return n
	}
	return 0
}

// Room is how many more of a prefab fit: what its partial stacks still
// take, and a full stack in each empty slot. The stack size comes from a
// stack the bot carries; for a prefab it carries none of, pass stackMax
// (from perception or the catalogue), or 0 to count one per empty slot.
// Weight is not counted: the bot can carry more than it can walk with.
// Nor is quality: the game stacks only items of the same quality, so for
// anything with levels this is an upper bound. Materials (wood, stone,
// ore) have one quality, and for them it is exact.
func (inv *Inventory) Room(prefab string, stackMax int) int {
	n := 0
	for _, it := range inv.Items {
		if it.Prefab != prefab {
			continue
		}
		if it.StackMax > 0 {
			stackMax = it.StackMax
		}
		if it.StackMax > it.Stack {
			n += it.StackMax - it.Stack
		}
	}
	if stackMax < 1 {
		stackMax = 1
	}
	return n + inv.Free()*stackMax
}

// gear are the item types the bot fights, works and dresses with (6.3).
var gear = map[string]bool{
	"OneHandedWeapon": true, "TwoHandedWeapon": true, "TwoHandedWeaponLeft": true, "Bow": true,
	"Shield": true, "Helmet": true, "Chest": true, "Legs": true, "Hands": true, "Shoulder": true,
	"Utility": true, "Trinket": true, "Tool": true, "Torch": true, "Ammo": true, "AmmoNonEquipable": true,
	"Attach_Atgeir": true, "Customization": true,
}

// Gear reports whether the item is something the bot fights, works or
// dresses with, as its type says: slow to replace, never dropped, and what
// `equip` takes.
func (it InventoryItem) Gear() bool { return gear[it.Type] }

// Equipable lists the prefabs `equip` could act on: gear a slot takes, not
// broken, and not the same prefab as something already equipped, which
// `equip` would leave as it is.
func (inv *Inventory) Equipable() []string {
	var held, out []string
	for _, it := range inv.Items {
		if it.Equipped {
			held = append(held, it.Prefab)
		}
	}
	for _, it := range inv.Items {
		broken := it.DurabilityMax > 0 && it.Durability <= 0
		if !it.Gear() || broken || unslotted[it.Type] || slices.Contains(held, it.Prefab) || slices.Contains(out, it.Prefab) {
			continue
		}
		out = append(out, it.Prefab)
	}
	return out
}

// unslotted are the gear types no equipment slot takes.
var unslotted = map[string]bool{"AmmoNonEquipable": true, "Customization": true, "Attach_Atgeir": true}
