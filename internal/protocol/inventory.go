package protocol

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
