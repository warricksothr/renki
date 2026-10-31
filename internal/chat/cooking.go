package chat

import (
	"math"
	"slices"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// A cooking request makes one native batch. It does not guess chest contents,
// touch someone else's occupied station, or promise a result before a receipt.
func (b *Brain) cook(from origin) (session.Order, string, bool) {
	state, seen, pack := b.d.State(), b.d.Perception(), b.d.Inventory()
	if state == nil || state.Phase != protocol.PhaseInWorld || seen == nil || pack == nil ||
		math.Abs(state.T-seen.T) > panelSightFor || math.Abs(state.T-pack.T) > panelSightFor {
		b.say(from, "I need a fresh view of my pack and nearby cooking stations first.")
		return session.Order{}, "", false
	}
	if !slices.Contains(b.d.AllowedTasks(), protocol.TaskPickup) {
		b.say(from, "I need pickup permission to collect the cooked food.")
		return session.Order{}, "", false
	}
	raw := ""
	for _, item := range []string{"RawMeat", "DeerMeat", "NeckTail"} {
		if pack.Count(item) > 0 {
			raw = item
			break
		}
	}
	if raw == "" {
		b.say(from, "I need carried raw boar meat, deer meat or a neck tail to cook. I can already eat ready food.")
		return session.Order{}, "", false
	}
	best, distance := "", 12.0
	for _, station := range seen.Structures {
		c := station.Cooking
		if station.ID == "" || station.Kind != protocol.StructureStation || c == nil || !c.Supported || !c.Heated || len(c.Slots) == 0 || len(c.Slots) > 16 {
			continue
		}
		if slices.ContainsFunc(c.Slots, func(s protocol.CookingSlotStatus) bool { return s.Item != "" || s.State != "empty" }) {
			continue
		}
		d := math.Hypot(state.Pos.FlatDistance(station.Pos), state.Pos[1]-station.Pos[1])
		if d < distance || d == distance && (best == "" || station.ID < best) {
			best, distance = station.ID, d
		}
	}
	if best == "" {
		b.say(from, "I need an empty cooking station over a working fire within 12 metres. I won't take an existing batch.")
		return session.Order{}, "", false
	}
	return session.Order{Task: protocol.TaskInteract, Args: &protocol.InteractArgs{Entity: best, Cook: raw}, TimeoutS: 180},
		"I'll cook one piece, watch it and collect the result.", true
}
