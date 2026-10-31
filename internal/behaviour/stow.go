package behaviour

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// Stow uses game transfers as the authority for capacity and stack compatibility.
// Remembered chests provide locations only; every destination is attempted once.
type Stow struct{}
type StowParams struct {
	World    string         `json:"world"`
	Items    []string       `json:"items"`
	From     string         `json:"from,omitempty"`
	To       []string       `json:"to"`
	Keep     map[string]int `json:"keep,omitempty"`
	Overflow string         `json:"overflow,omitempty"`
}

type StorageNeed struct {
	World  string         `json:"world"`
	Items  map[string]int `json:"items"`
	Chests []string       `json:"chests"`
	Reason string         `json:"reason"`
	Quiet  bool           `json:"quiet,omitempty"`
}

func (n *StorageNeed) Error() string {
	parts := []string{}
	for _, p := range sortedKeys(n.Items) {
		parts = append(parts, fmt.Sprintf("%d %s", n.Items[p], p))
	}
	return "Storage needed for " + strings.Join(parts, ", ") + ": " + n.Reason
}
func sortedKeys[M ~map[string]V, V any](m M) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func namesOK(xs []string, max int) bool {
	if len(xs) == 0 || len(xs) > max {
		return false
	}
	seen := map[string]bool{}
	for _, x := range xs {
		if strings.TrimSpace(x) != x || x == "" || len(x) > 100 || seen[x] {
			return false
		}
		seen[x] = true
	}
	return true
}
func (p StowParams) validate() error {
	if strings.TrimSpace(p.World) == "" || !namesOK(p.Items, 64) || !namesOK(p.To, 16) {
		return errors.New("stow needs world, 1–64 item selectors and 1–16 unique chest IDs")
	}
	if p.From != "" && p.From != "pack" {
		return errors.New("stow currently takes items from pack only")
	}
	if p.Overflow != "" && p.Overflow != "tell" && p.Overflow != "leave" {
		return errors.New("overflow must be tell or leave")
	}
	if len(p.Keep) > 64 {
		return errors.New("too many keep entries")
	}
	for name, n := range p.Keep {
		if !namesOK([]string{name}, 1) || n < 0 || n > 1000000 {
			return errors.New("keep needs prefab names and nonnegative counts")
		}
	}
	return nil
}
func (Stow) Validate(raw json.RawMessage) error {
	var p StowParams
	if err := Decode(raw, &p); err != nil {
		return err
	}
	return p.validate()
}
func (t Stow) Start(raw json.RawMessage) (Runner, error) {
	if err := t.Validate(raw); err != nil {
		return nil, err
	}
	var p StowParams
	_ = Decode(raw, &p)
	return &stowRun{p: p}, nil
}

func selected(selectors []string, it protocol.InventoryItem) bool {
	for _, s := range selectors {
		if s == "*" || s == it.Prefab {
			return true
		}
		switch s {
		case "wood":
			if slices.Contains([]string{"Wood", "FineWood", "RoundLog", "YggdrasilWood", "Blackwood"}, it.Prefab) {
				return true
			}
		case "ore":
			if slices.Contains([]string{"CopperOre", "TinOre", "IronScrap", "SilverOre", "BlackMetalScrap", "FlametalOre", "FlametalOreNew", "CopperScrap"}, it.Prefab) {
				return true
			}
		case "food":
			if it.Type == "Consumable" {
				return true
			}
		}
	}
	return false
}
func stowCounts(inv *protocol.Inventory, p StowParams) map[string]int {
	out := map[string]int{}
	if inv == nil {
		return out
	}
	for _, it := range inv.Items {
		if !it.Equipped && !it.Gear() && selected(p.Items, it) && it.Stack > 0 {
			out[it.Prefab] += it.Stack
		}
	}
	for name, n := range out {
		n = min(n, max(0, inv.Count(name)-p.Keep[name]))
		if n == 0 {
			delete(out, name)
		} else {
			out[name] = n
		}
	}
	return out
}

type stowRun struct {
	p            StowParams
	remaining    map[string]int
	index        int
	walking      bool
	approached   bool
	homeApproach bool
	pending      bool
	beforeSeq    uint64
	failures     []string
	dry          bool
}

func (*stowRun) Recover(r Result) bool {
	if r.Status == protocol.StatusRejected && r.Reason == protocol.ReasonInvalidTarget {
		// A missing chest is rejected before a TransferTask exists; it
		// cannot have moved items and needs no transfer receipt.
		return true
	}
	if r.Status != protocol.StatusFailed && r.Status != protocol.StatusRejected {
		return false
	}
	return slices.Contains([]string{protocol.ReasonInventoryFull, protocol.ReasonRefused, protocol.ReasonTargetLost, protocol.ReasonUnreachable, protocol.ReasonStuck}, r.Reason)
}
func chestPosition(v View, id string) (protocol.Vec3, bool) {
	if freshPerception(v) {
		for _, c := range v.Perception.Structures {
			if c.ID == id && c.Kind == protocol.StructureContainer {
				return c.Pos, true
			}
		}
	}
	for _, c := range v.Containers {
		if c.Entity == id {
			return c.Pos, true
		}
	}
	return protocol.Vec3{}, false
}
func (r *stowRun) Next(v View, result *Result) (*session.Order, error) {
	if v.State == nil || v.State.World != r.p.World {
		return nil, errors.New("stow world changed")
	}
	if v.Inventory == nil {
		return nil, ErrWait
	}
	if r.remaining == nil {
		r.remaining = stowCounts(v.Inventory, r.p)
	}
	if r.walking {
		if result == nil {
			return nil, errors.New("missing chest approach result")
		}
		r.walking = false
		r.approached = result.Status == protocol.StatusSucceeded
		if !r.approached {
			// A raised chest can be a poor exterior path goal even though
			// the owner's configured home point has a usable entrance route.
			// Try it once, only for a known chest within 10 m of that home.
			if !r.homeApproach && v.Home != nil && (result.Reason == protocol.ReasonUnreachable || result.Reason == protocol.ReasonStuck) {
				if pos, ok := chestPosition(v, r.p.To[r.index]); ok && distance(pos, *v.Home) <= 10 {
					r.walking, r.homeApproach = true, true
					home := *v.Home
					return &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &home}, Dry: r.dry}, TimeoutS: 90}, nil
				}
			}
			r.failures = append(r.failures, r.p.To[r.index]+": "+result.Reason)
			r.index++
			r.homeApproach = false
		}
	}
	if r.pending {
		if result == nil {
			return nil, errors.New("missing deposit result")
		}
		// Transfer reports are sent before the pack and final task result. A
		// receipt from another task or a remembered snapshot cannot prove a move.
		if result.Status == protocol.StatusSucceeded || result.Reason == protocol.ReasonInventoryFull {
			var receipt *protocol.Container
			for _, c := range v.Containers {
				if c.Task == result.TaskID && c.Entity == r.p.To[r.index] && c.Direction == protocol.TaskDeposit {
					receipt = c
					break
				}
			}
			if receipt == nil || v.Inventory.Seq <= r.beforeSeq {
				return nil, ErrWait
			}
			for name, n := range receipt.Moved {
				if n < 0 || n > r.remaining[name] {
					return nil, errors.New("invalid deposit receipt")
				}
				r.remaining[name] -= n
			}
		}
		if result.Status != protocol.StatusSucceeded {
			r.failures = append(r.failures, r.p.To[r.index]+": "+result.Reason)
		}
		r.pending = false
		r.approached = false
		r.homeApproach = false
		r.index++
	}
	current := stowCounts(v.Inventory, r.p)
	for name, n := range r.remaining {
		r.remaining[name] = min(n, current[name])
		if r.remaining[name] <= 0 {
			delete(r.remaining, name)
		}
	}
	if len(r.remaining) == 0 {
		return nil, nil
	}
	if r.index >= len(r.p.To) {
		why := strings.Join(r.failures, "; ")
		if why == "" {
			why = "listed chests have no room"
		}
		return nil, &StorageNeed{World: r.p.World, Items: maps.Clone(r.remaining), Chests: slices.Clone(r.p.To), Reason: why, Quiet: r.p.Overflow == "leave"}
	}
	id := r.p.To[r.index]
	if pos, ok := chestPosition(v, id); ok && !r.approached && distance(v.State.Pos, pos) > 4 {
		r.walking = true
		return &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}, Dry: r.dry}, TimeoutS: 90}, nil
	}
	items := []protocol.TransferItem{}
	for _, name := range sortedKeys(r.remaining) {
		items = append(items, protocol.TransferItem{Prefab: name, Count: protocol.ItemCount(r.remaining[name])})
	}
	r.pending = true
	r.beforeSeq = v.Inventory.Seq
	return &session.Order{Task: protocol.TaskDeposit, Args: &protocol.TransferArgs{Container: id, Items: items}, TimeoutS: 90}, nil
}
