package home

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
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

// memory is the file's content.
type memory struct {
	// Homes by world name.
	Homes map[string]Home `json:"homes"`
	// Interrupted is set while a body task runs, and cleared when it ends
	// in the same session. Still set at the next session means an outage
	// cut it short.
	Interrupted bool `json:"interrupted"`
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
