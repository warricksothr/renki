// Package burden frees a bot that carries more than it can. Over its carry
// weight the game lets it walk but not run, stops its stamina coming back,
// and once that is gone will not move it at all; players cannot take
// things out of its pack, so only the bot can fix it. It drops the least
// valuable, heaviest things first, never its gear, just enough to be back
// under its carry weight, says what it left, and remembers where.
package burden

import (
	"cmp"
	"math"
	"slices"
	"strings"

	"github.com/warricksothr/renki/internal/protocol"
)

// Drop is one drop task: Count of Prefab.
type Drop struct {
	Prefab string
	Count  int
}

// common are the raw materials found everywhere, the first to go.
var common = map[string]bool{
	"Wood": true, "Stone": true, "Flint": true, "Resin": true, "Feathers": true,
}

// tier ranks how much an item is worth keeping, lowest first: common raw
// materials, then other materials, then food and potions, then trophies,
// odds and ends and any type it does not know (a boss trophy or a key is
// worth more than a meal), then valuables. Gear has no tier; it is never
// dropped.
func tier(it protocol.InventoryItem) (int, bool) {
	switch {
	// Gear is never dropped: it is slow to replace, and the ground is no
	// place for it.
	case it.Equipped || it.Gear():
		return 0, false
	case it.Value > 0:
		return 4, true
	case it.Type == "Consumable" || it.Type == "Fish":
		return 2, true
	case it.Type != "Material":
		return 3, true
	case common[it.Prefab]:
		return 0, true
	}
	return 1, true
}

// Plan chooses what to drop to bring weight down to max: the lowest tier
// first and, within a tier, the heaviest per item, and only as many of
// each as it takes. short is true when everything it may drop is not
// enough.
func Plan(inv *protocol.Inventory, weight, max float64) (drops []Drop, short bool) {
	type kind struct {
		prefab string
		tier   int
		weight float64
		count  int
	}
	var kinds []kind
	for _, it := range inv.Items {
		t, ok := tier(it)
		// An item with no prefab goes by its name token (6.3), which drop
		// cannot name: the game has nothing to put on the ground.
		if !ok || it.Weight <= 0 || strings.HasPrefix(it.Prefab, "$") {
			continue
		}
		i := slices.IndexFunc(kinds, func(k kind) bool { return k.prefab == it.Prefab })
		if i < 0 {
			kinds = append(kinds, kind{prefab: it.Prefab, tier: t, weight: it.Weight})
			i = len(kinds) - 1
		}
		kinds[i].count += it.Stack
	}
	slices.SortFunc(kinds, func(a, b kind) int {
		return cmp.Or(cmp.Compare(a.tier, b.tier), cmp.Compare(b.weight, a.weight),
			cmp.Compare(b.count, a.count), cmp.Compare(a.prefab, b.prefab))
	})
	over := weight - max
	for _, k := range kinds {
		if over <= 0 {
			break
		}
		n := min(k.count, int(math.Ceil(over/k.weight-1e-9)))
		drops = append(drops, Drop{Prefab: k.prefab, Count: n})
		over -= float64(n) * k.weight
	}
	return drops, over > 0
}
