package behaviour

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// MarkerCargo sorts surplus materials when an idle loop ends at storage.
// Origins rank inspection candidates; only native matching deposits assign stock.
type MarkerCargo struct{}
type MarkerCargoParams struct {
	Marker Marker         `json:"marker"`
	Keep   map[string]int `json:"keep,omitempty"`
}

func (MarkerCargo) Validate(raw json.RawMessage) error {
	var p MarkerCargoParams
	if err := Decode(raw, &p); err != nil {
		return err
	}
	if err := p.Marker.validate(); err != nil {
		return err
	}
	if p.Marker.Kind != "storage" {
		return errors.New("cargo work needs a storage marker")
	}
	return (StowParams{World: p.Marker.World, Items: []string{"*"}, To: []string{"validation"}, Keep: p.Keep}).validate()
}
func (t MarkerCargo) Start(raw json.RawMessage) (Runner, error) {
	if err := t.Validate(raw); err != nil {
		return nil, err
	}
	var p MarkerCargoParams
	_ = Decode(raw, &p)
	return &markerCargoRun{p: p, guard: markerFoodRun{p: MarkerFoodParams{Marker: p.Marker}}, tried: map[string]bool{}, shortage: map[string]int{}}, nil
}
func MarkerCargoDefinition(marker Marker) Definition {
	raw, _ := json.Marshal(MarkerCargoParams{Marker: marker, Keep: map[string]int{"Wood": 10}})
	return Definition{Template: "marker_cargo", Params: raw, Trigger: Trigger{Kind: "idle", Marker: &marker, DwellS: 30}, Repeat: Repeat{Mode: "trigger", CooldownS: 300, MaxRuns: 100}, Enabled: true, Priority: -20}
}

type markerCargoRun struct {
	p        MarkerCargoParams
	guard    markerFoodRun
	child    *stowRun
	items    []string
	tried    map[string]bool
	shortage map[string]int
	chests   []string
	failures []string
}

func (r *markerCargoRun) Check(v View) error {
	if v.State != nil && slices.Contains(v.State.Flags, "recovering") {
		return errors.New("native recovery owns the body")
	}
	if !slices.Contains(v.Features, protocol.FeatureMatchingDeposit) {
		return errors.New("cargo sorting requires native conditional deposit support")
	}
	if err := r.guard.Check(v); err != nil {
		return fmt.Errorf("cargo sorting: %s", strings.ReplaceAll(err.Error(), "food", "cargo"))
	}
	w, _ := r.p.Marker.resolve(v)
	if !slices.Contains(v.State.Flags, protocol.FlagGrounded) || absHeight(v.State.Pos, w.Pos) > .75 {
		return errors.New("cargo sorting requires grounded feet on the storage floor")
	}
	return nil
}
func (r *markerCargoRun) Recover(result Result) bool {
	return r.child != nil && (r.child.Recover(result) || result.Status == protocol.StatusFailed && result.Reason == protocol.ReasonNotAllowed)
}
func (r *markerCargoRun) surplus(v View) map[string]int {
	items := []string{}
	for _, it := range v.Inventory.Items {
		// Everything edible, equipped, useful as gear, or of unknown type stays.
		if (it.Type == "Material" || it.Type == "Trophy") && !it.Equipped && !it.Gear() && !slices.Contains(items, it.Prefab) {
			items = append(items, it.Prefab)
		}
	}
	return stowCounts(v.Inventory, StowParams{Items: items, Keep: r.p.Keep})
}

type cargoChest struct {
	id                       string
	pos                      protocol.Vec3
	matching                 bool
	rank                     int
	originDistance, distance float64
}

func (r *markerCargoRun) candidates(v View, item string) []cargoChest {
	byID := map[string]cargoChest{}
	live := map[string]protocol.Vec3{}
	for _, c := range v.Perception.Structures {
		if c.Kind == protocol.StructureContainer {
			live[c.ID] = c.Pos
		}
	}
	add := func(id string, pos protocol.Vec3, matching bool) {
		current, exists := live[id]
		if !exists {
			return
		} // Tombstones and old/missing IDs never become storage.
		pos = current
		if id == "" || !withinMarker(pos, *r.guard.marker) || absHeight(pos, r.guard.marker.Pos) > .75 {
			return
		}
		for _, n := range pos {
			if !finite(n) {
				return
			}
		}
		old := byID[id]
		old.id, old.pos = id, pos
		old.matching = old.matching || matching
		byID[id] = old
	}
	for _, c := range v.Containers {
		if v.State.T >= c.T {
			add(c.Entity, c.Pos, c.Inventory().Count(item) > 0)
		}
	}
	for _, c := range v.Perception.Structures {
		if c.Kind == protocol.StructureContainer {
			add(c.ID, c.Pos, false)
		}
	}
	out := []cargoChest{}
	for id, c := range byID {
		if r.tried[item+"\x00"+id] {
			continue
		}
		c.rank = 2
		c.originDistance = 1e9
		c.distance = distance(v.State.Pos, c.pos)
		for _, p := range v.Pickups {
			age := v.Now.Sub(p.At)
			if p.Prefab == item && p.World == v.State.World && p.WorldUID == v.State.WorldUID && age >= 0 && age <= home.KeepPickups && distance(p.Pos, c.pos) <= 12 && absHeight(p.Pos, c.pos) <= 2 {
				c.rank = 1
				c.originDistance = min(c.originDistance, distance(p.Pos, c.pos))
			}
		}
		if c.matching {
			c.rank = 0
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.rank != b.rank {
			return a.rank < b.rank
		}
		if a.originDistance != b.originDistance {
			return a.originDistance < b.originDistance
		}
		if a.distance != b.distance {
			return a.distance < b.distance
		}
		return a.id < b.id
	})
	return out
}
func absHeight(a, b protocol.Vec3) float64 {
	d := a[1] - b[1]
	if d < 0 {
		return -d
	}
	return d
}
func (r *markerCargoRun) Next(v View, result *Result) (*session.Order, error) {
	if err := r.Check(v); err != nil {
		return nil, err
	}
	if r.guard.marker == nil {
		w, _ := r.p.Marker.resolve(v)
		r.guard.marker = &w
		r.guard.began = v.Now
		r.guard.health = sampleHealth(v)
		r.items = sortedKeys(r.surplus(v))
	}
	for {
		if r.child != nil {
			order, err := r.child.Next(v, result)
			result = nil
			if order != nil {
				return order, err
			}
			if err != nil {
				var need *StorageNeed
				if !errors.As(err, &need) {
					return nil, err
				}
				if len(r.failures) < 8 {
					r.failures = append(r.failures, need.Reason)
				}
			}
			r.child = nil
		}
		if len(r.items) == 0 {
			if len(r.shortage) > 0 {
				why := "no reachable matching chest with space; materials kept"
				if len(r.failures) > 0 {
					why += ": " + strings.Join(r.failures, "; ")
				}
				return nil, &StorageNeed{World: r.p.Marker.World, Items: maps.Clone(r.shortage), Chests: slices.Clone(r.chests), Reason: why}
			}
			return nil, nil
		}
		item := r.items[0]
		count := r.surplus(v)[item]
		if count == 0 {
			r.items = r.items[1:]
			continue
		}
		candidates := r.candidates(v, item)
		if len(candidates) == 0 {
			r.shortage[item] = count
			r.items = r.items[1:]
			continue
		}
		id := candidates[0].id
		r.tried[item+"\x00"+id] = true
		if !slices.Contains(r.chests, id) {
			r.chests = append(r.chests, id)
		}
		r.child = &stowRun{p: StowParams{World: r.p.Marker.World, Items: []string{item}, To: []string{id}, Keep: r.p.Keep, MatchExisting: true}, dry: true}
	}
}
