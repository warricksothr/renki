package home

import (
	"maps"
	"math"
	"time"
)

// Legacy landmarks are rounded to half metres: their maximum 3D error is
// sqrt(3)*.25 = .433 m. A live positive match must fit that error, including
// height. This is never an absence test against truncated perception.
const landmarkMatch = .45

type landmarkCell struct {
	typeKey landmarkKey
	x, y, z float64
}

func observationCell(o Landmark) landmarkCell {
	return landmarkCell{landmarkType{Kind: o.Kind, Prefab: o.Prefab, MinTool: o.MinTool}.key(),
		math.Floor(o.Pos[0]), math.Floor(o.Pos[1]), math.Floor(o.Pos[2])}
}

// supersededLandmarks resolves older records only against unique newer
// positive observations. Simultaneously observed IDs must stay distinct.
func supersededLandmarks(w landmarkWorld, observations []Landmark) map[string]string {
	latest := make(map[string]Landmark, len(observations))
	for _, o := range observations {
		if o.Prefab != "" && !latest[o.Entity].At.After(o.At) {
			latest[o.Entity] = o
		}
	}
	index := map[landmarkCell][]Landmark{}
	for _, o := range latest {
		key := observationCell(o)
		index[key] = append(index[key], o)
	}
	aliases := map[string]string{}
	for id, old := range w.Entities {
		if old.Type < 0 || old.Type >= len(w.Types) || w.Types[old.Type].Prefab == "" {
			continue
		}
		t := w.Types[old.Type]
		key := observationCell(Landmark{Kind: t.Kind, Prefab: t.Prefab, MinTool: t.MinTool, Pos: old.Pos})
		var newest time.Time
		var match string
		ambiguous := false
		for x := -1; x <= 1; x++ {
			for y := -1; y <= 1; y++ {
				for z := -1; z <= 1; z++ {
					near := landmarkCell{key.typeKey, key.x + float64(x), key.y + float64(y), key.z + float64(z)}
					for _, o := range index[near] {
						if o.Entity == id || !o.At.After(old.At) || math.Hypot(old.Pos.FlatDistance(o.Pos), old.Pos[1]-o.Pos[1]) > landmarkMatch ||
							!latest[id].At.Before(o.At) {
							continue
						}
						if o.At.After(newest) {
							match, newest, ambiguous = o.Entity, o.At, false
						} else if o.At.Equal(newest) && match != o.Entity {
							ambiguous = true
						}
					}
				}
			}
		}
		if match != "" && !ambiguous {
			aliases[id] = match
		}
	}
	// A new object cannot settle which of two older nearby objects it
	// replaces. Leave that ambiguous group alone as well.
	counts := map[string]int{}
	for _, id := range aliases {
		counts[id]++
	}
	for oldID, id := range aliases {
		if counts[id] > 1 {
			delete(aliases, oldID)
		}
	}
	return aliases
}

// Contents keep their original source and time. Moving this last-known hint
// to a reconciled place cannot make it a current inventory observation.
func reconcileContainers(current map[string]RememberedContainer, aliases map[string]string) map[string]RememberedContainer {
	next := maps.Clone(current)
	for oldID, id := range aliases {
		old, ok := next[oldID]
		if !ok {
			continue
		}
		delete(next, oldID)
		known, exists := next[id]
		if exists && (known.ObservedEntity == "" || known.ObservedEntity == id || !old.At.After(known.At)) {
			continue
		}
		if old.ObservedEntity == "" {
			old.ObservedEntity = oldID
		}
		old.Entity = id
		next[id] = old
	}
	return next
}
