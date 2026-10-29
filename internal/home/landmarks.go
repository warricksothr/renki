package home

import (
	"errors"
	"maps"
	"math"
	"sort"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

const MaxLandmarks = 16384
const KeepLandmarks = 30 * 24 * time.Hour

// Landmark is a last-seen place, not proof that an object is still there.
// Static metadata is expanded on reads; the file shares it per world.
type Landmark struct {
	Entity  string        `json:"entity"`
	Kind    string        `json:"kind"`
	Prefab  string        `json:"prefab,omitempty"`
	MinTool *int          `json:"minTool,omitempty"`
	Pos     protocol.Vec3 `json:"pos"`
	At      time.Time     `json:"at"`
	Tag     string        `json:"tag,omitempty"`
}

type landmarkType struct {
	Kind    string `json:"kind"`
	Prefab  string `json:"prefab,omitempty"`
	MinTool *int   `json:"minTool,omitempty"`
}

type landmarkRecord struct {
	Type int           `json:"type"`
	Pos  protocol.Vec3 `json:"pos"`
	At   time.Time     `json:"at"`
	Tag  string        `json:"tag,omitempty"`
}

type landmarkWorld struct {
	Types    []landmarkType            `json:"types"`
	Entities map[string]landmarkRecord `json:"entities"`
}

type landmarkKey struct {
	kind, prefab string
	tool         int
	hasTool      bool
}

func (t landmarkType) key() landmarkKey {
	k := landmarkKey{kind: t.Kind, prefab: t.Prefab}
	if t.MinTool != nil {
		k.tool, k.hasTool = *t.MinTool, true
	}
	return k
}

func copyTool(tool *int) *int {
	if tool == nil {
		return nil
	}
	v := *tool
	return &v
}

// Landmarks returns independent expanded observations, newest first.
func (s *Store) Landmarks(world string) []Landmark {
	s.mu.Lock()
	defer s.mu.Unlock()
	w := s.mem.Landmarks[world]
	out := make([]Landmark, 0, len(w.Entities))
	for id, r := range w.Entities {
		if r.Type < 0 || r.Type >= len(w.Types) {
			continue
		}
		t := w.Types[r.Type]
		out = append(out, Landmark{Entity: id, Kind: t.Kind, Prefab: t.Prefab, MinTool: copyTool(t.MinTool), Pos: r.Pos, At: r.At, Tag: r.Tag})
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].At.Equal(out[j].At) {
			return out[i].At.After(out[j].At)
		}
		return out[i].Entity < out[j].Entity
	})
	return out
}

// RememberLandmarks rounds locations to half metres and replaces observations
// by entity. Older frames cannot overwrite newer knowledge. Pruning also
// compacts the catalogue, so unused prefab metadata cannot accumulate.
func (s *Store) RememberLandmarks(world string, observations []Landmark, now time.Time) error {
	if world == "" {
		return errors.New("landmark observation needs a world")
	}
	for _, o := range observations {
		if o.Entity == "" || o.Kind == "" || o.At.IsZero() {
			return errors.New("landmark observation needs entity, kind and time")
		}
		for _, v := range o.Pos {
			if math.IsNaN(v) || math.IsInf(v, 0) || math.IsInf(v*2, 0) {
				return errors.New("landmark position must be finite")
			}
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.mem.Landmarks[world]
	next := landmarkWorld{Types: append([]landmarkType(nil), old.Types...), Entities: maps.Clone(old.Entities)}
	if next.Entities == nil {
		next.Entities = map[string]landmarkRecord{}
	}
	index := map[landmarkKey]int{}
	for i, t := range next.Types {
		index[t.key()] = i
	}
	for _, o := range observations {
		if next.Entities[o.Entity].At.After(o.At) {
			continue
		}
		t := landmarkType{Kind: o.Kind, Prefab: o.Prefab, MinTool: copyTool(o.MinTool)}
		i, ok := index[t.key()]
		if !ok {
			i = len(next.Types)
			index[t.key()] = i
			next.Types = append(next.Types, t)
		}
		for i := range o.Pos {
			o.Pos[i] = math.Round(o.Pos[i]*2) / 2
		}
		next.Entities[o.Entity] = landmarkRecord{Type: i, Pos: o.Pos, At: o.At, Tag: o.Tag}
	}
	for id, r := range next.Entities {
		if now.Sub(r.At) > KeepLandmarks || r.Type < 0 || r.Type >= len(next.Types) {
			delete(next.Entities, id)
		}
	}
	ids := make([]string, 0, len(next.Entities))
	for id := range next.Entities {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := next.Entities[ids[i]], next.Entities[ids[j]]
		if !a.At.Equal(b.At) {
			return a.At.After(b.At)
		}
		return ids[i] < ids[j]
	})
	if len(ids) > MaxLandmarks {
		for _, id := range ids[MaxLandmarks:] {
			delete(next.Entities, id)
		}
		ids = ids[:MaxLandmarks]
	}
	// Stable entity order also makes catalogue numbering deterministic.
	sort.Strings(ids)
	types := []landmarkType{}
	remap := map[int]int{}
	for _, id := range ids {
		r := next.Entities[id]
		i, ok := remap[r.Type]
		if !ok {
			i = len(types)
			remap[r.Type] = i
			types = append(types, next.Types[r.Type])
		}
		r.Type = i
		next.Entities[id] = r
	}
	next.Types = types
	previous := s.mem.Landmarks
	worlds := maps.Clone(previous)
	if worlds == nil {
		worlds = map[string]landmarkWorld{}
	}
	worlds[world] = next
	s.mem.Landmarks = worlds
	if err := s.save(); err != nil {
		s.mem.Landmarks = previous
		return err
	}
	return nil
}
