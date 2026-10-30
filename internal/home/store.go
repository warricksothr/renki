package home

import (
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

// Home is where the bot lives in one world.
type Home struct {
	Pos   protocol.Vec3 `json:"pos"`
	SetBy string        `json:"setBy"`
	At    time.Time     `json:"at"`
}

// Cache is what the bot had to leave on the ground in one place, to be
// picked up later.
type Cache struct {
	Pos protocol.Vec3 `json:"pos"`
	// Items counts what was left, by prefab.
	Items map[string]int `json:"items"`
	At    time.Time      `json:"at"`
	Why   string         `json:"why"`
}

// Grave is where the bot died, and so where its things are.
type Grave struct {
	Pos protocol.Vec3 `json:"pos"`
	At  time.Time     `json:"at"`
}

// keepGraves is how many graves are remembered per world; the game keeps
// each until it is emptied, but the oldest are the least likely to be there.
const keepGraves = 5

// memory is the file's content.
type memory struct {
	// Homes by world name.
	Homes map[string]Home `json:"homes"`
	// Interrupted is set while a body task runs, and cleared when it ends
	// in the same session. Still set at the next session means an outage
	// cut it short.
	Interrupted bool `json:"interrupted"`
	// LeftBehind is what the bot dropped and has not seen picked up, by
	// world name.
	LeftBehind map[string][]Cache `json:"leftBehind,omitempty"`
	// Graves are where the bot died and has not yet fetched its things
	// from, by world name, oldest first.
	Graves map[string][]Grave `json:"graves,omitempty"`
	// Shunned are the worlds where the bot died near home and has had no
	// order since: nothing sends it home there by itself.
	Shunned    []string                                  `json:"shunned,omitempty"`
	Containers map[string]map[string]RememberedContainer `json:"containers,omitempty"`
	Landmarks  map[string]landmarkWorld                  `json:"landmarks,omitempty"`
	Harms      map[string][]Harm                         `json:"harms,omitempty"`
	Waypoints  map[string]waypointWorld                  `json:"waypoints,omitempty"`
	Players    map[string][]PlayerEncounter              `json:"players,omitempty"`
}

// Store is what the controller remembers between runs, in one JSON file
// that is replaced whole on every change.
type Store struct {
	path string

	mu  sync.Mutex
	mem memory
}

// Memory returns a store that keeps everything in memory, for tests and
// for running with nowhere to write.
func Memory() *Store { return &Store{mem: memory{Homes: map[string]Home{}}} }

// Open reads the store at path, or starts an empty one if there is none.
func Open(path string) (*Store, error) {
	s := &Store{path: path, mem: memory{Homes: map[string]Home{}}}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &s.mem); err != nil {
		return nil, err
	}
	if s.mem.Homes == nil {
		s.mem.Homes = map[string]Home{}
	}
	return s, nil
}

// Home returns the home in world.
func (s *Store) Home(world string) (Home, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.mem.Homes[world]
	return h, ok
}

// SetHome stores the home in world.
func (s *Store) SetHome(world string, h Home) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mem.Homes[world] = h
	return s.save()
}

// Interrupted reports the flag Keeper uses to notice outages.
func (s *Store) Interrupted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mem.Interrupted
}

// SetInterrupted sets the flag, writing only when it changes.
func (s *Store) SetInterrupted(v bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mem.Interrupted == v {
		return nil
	}
	s.mem.Interrupted = v
	return s.save()
}

// LeftBehind returns a copy of what the bot left on the ground in world.
func (s *Store) LeftBehind(world string) []Cache {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Cache, 0, len(s.mem.LeftBehind[world]))
	for _, c := range s.mem.LeftBehind[world] {
		c.Items = maps.Clone(c.Items)
		out = append(out, c)
	}
	return out
}

// UpdateLeftBehind replaces what is remembered in world with what f makes
// of it, and writes the file. Caches f leaves with no items are dropped.
func (s *Store) UpdateLeftBehind(world string, f func([]Cache) []Cache) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	caches := f(s.mem.LeftBehind[world])
	caches = slices.DeleteFunc(caches, func(c Cache) bool { return len(c.Items) == 0 })
	if s.mem.LeftBehind == nil {
		s.mem.LeftBehind = map[string][]Cache{}
	}
	if len(caches) == 0 {
		delete(s.mem.LeftBehind, world)
	} else {
		s.mem.LeftBehind[world] = caches
	}
	return s.save()
}

// Shunned reports whether the bot keeps away from home in world.
func (s *Store) Shunned(world string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Contains(s.mem.Shunned, world)
}

// Shun adds world to those shunned, writing only when it changes.
func (s *Store) Shun(world string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.Contains(s.mem.Shunned, world) {
		return nil
	}
	s.mem.Shunned = append(s.mem.Shunned, world)
	return s.save()
}

// Unshun ends the shun in world, writing only when there was one.
func (s *Store) Unshun(world string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !slices.Contains(s.mem.Shunned, world) {
		return nil
	}
	s.mem.Shunned = slices.DeleteFunc(s.mem.Shunned, func(w string) bool { return w == world })
	return s.save()
}

// Graves returns where the bot died in world, oldest first.
func (s *Store) Graves(world string) []Grave {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.mem.Graves[world])
}

// AddGrave remembers a death in world, forgetting the oldest past
// keepGraves.
func (s *Store) AddGrave(world string, g Grave) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mem.Graves == nil {
		s.mem.Graves = map[string][]Grave{}
	}
	gs := append(s.mem.Graves[world], g)
	if len(gs) > keepGraves {
		gs = gs[len(gs)-keepGraves:]
	}
	s.mem.Graves[world] = gs
	return s.save()
}

// ForgetGrave forgets the grave in world nearest pos, if within r: one
// emptied, while others beside it may still be full.
func (s *Store) ForgetGrave(world string, pos protocol.Vec3, r float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	gs := slices.Clone(s.mem.Graves[world])
	near := -1
	for i, g := range gs {
		if d := g.Pos.FlatDistance(pos); d <= r && (near < 0 || d < gs[near].Pos.FlatDistance(pos)) {
			near = i
		}
	}
	if near < 0 {
		return nil
	}
	gs = slices.Delete(gs, near, near+1)
	if len(gs) == 0 {
		delete(s.mem.Graves, world)
	} else {
		s.mem.Graves[world] = gs
	}
	return s.save()
}

// save writes the file beside itself and renames it into place, so a crash
// leaves the old file or the new one, never half of one. s.mu is held.
func (s *Store) save() error {
	if s.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(s.mem, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
