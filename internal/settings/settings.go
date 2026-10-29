// Package settings keeps the controller's runtime settings: what the
// config file says, with what owners, the panel or the agent have changed
// since, each change recorded with who made it and when. Readers take a
// fresh snapshot each time, so a change applies at once.
package settings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/warricksothr/renki/internal/config"
)

// Values are the settings that can change while the bot runs.
type Values struct {
	Owners       []string `json:"owners"`
	Nicknames    []string `json:"nicknames"`
	AllowedTasks []string `json:"allowedTasks"`
	// Voice turns the LLM's chat replies on or off; off, the bot answers
	// the command grammar only. It has no effect with no LLM configured.
	Voice    bool             `json:"voice"`
	Home     config.HomeRules `json:"home"`
	Overload config.Overload  `json:"overload"`
}

// Keys are the settings that can be changed, as dotted paths into Values.
var Keys = []string{
	"owners", "nicknames", "allowedTasks", "voice",
	"home.whenLost", "home.whenStuck", "home.afterOutage", "home.idleMinutes", "home.afterDeath",
	"overload.drop",
}

// ownersOnly are the settings the agent may never change, pinned or not:
// who gives the orders, and what the bot may be told to do.
var ownersOnly = []string{"owners", "allowedTasks"}

// Sources of a change. An owner's is "owner:<name>".
const (
	ByPanel  = "panel"
	ByAgent  = "agent"
	ByConfig = "config"
)

// Errors from Store.
var (
	ErrUnknownKey = errors.New("no such setting")
	ErrPinned     = errors.New("an owner has pinned this setting")
	ErrBadValue   = errors.New("bad value")
	ErrNotAllowed = errors.New("only an owner or the panel may do that")
)

// Override is a value set at runtime in place of the config's.
type Override struct {
	Value json.RawMessage `json:"value"`
	By    string          `json:"by"`
	At    time.Time       `json:"at"`
}

// Pin keeps the agent from changing a setting.
type Pin struct {
	By string    `json:"by"`
	At time.Time `json:"at"`
}

// file is settings.json.
type file struct {
	Overrides map[string]Override `json:"overrides"`
	Pins      map[string]Pin      `json:"pins"`
}

// Store holds the settings.
type Store struct {
	path string
	base config.Config
	log  *slog.Logger
	now  func() time.Time

	mu   sync.Mutex
	f    file
	cur  Values
	subs []func(Values)
}

// Open reads the overrides at path over cfg. An empty path keeps them in
// memory only. Overrides the config no longer accepts are dropped, with a
// warning, rather than stopping the controller.
func Open(path string, cfg config.Config, log *slog.Logger, now func() time.Time) (*Store, error) {
	if log == nil {
		log = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	s := &Store{path: path, base: cfg, log: log, now: now,
		f: file{Overrides: map[string]Override{}, Pins: map[string]Pin{}}}
	if path != "" {
		data, err := os.ReadFile(path)
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			return nil, err
		default:
			if err := json.Unmarshal(data, &s.f); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
		}
	}
	if s.f.Overrides == nil {
		s.f.Overrides = map[string]Override{}
	}
	if s.f.Pins == nil {
		s.f.Pins = map[string]Pin{}
	}
	// A saved change the config no longer accepts, or a key this version
	// does not know, is dropped rather than stopping the controller.
	for _, k := range slices.Sorted(maps.Keys(s.f.Overrides)) {
		err := fmt.Errorf("%w: %q", ErrUnknownKey, k)
		if slices.Contains(Keys, k) {
			_, err = s.build(map[string]Override{k: s.f.Overrides[k]})
		}
		if err != nil {
			s.log.Warn("dropping a saved setting", "key", k, "err", err)
			delete(s.f.Overrides, k)
		}
	}
	cur, err := s.build(s.f.Overrides)
	if err != nil {
		s.log.Warn("dropping all saved settings", "err", err)
		s.f.Overrides = map[string]Override{}
		if cur, err = s.build(nil); err != nil {
			return nil, err
		}
	}
	s.cur = cur
	return s, nil
}

// Values returns the settings now. The slices are copies.
func (s *Store) Values() Values {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.cur
	v.Owners, v.Nicknames, v.AllowedTasks = slices.Clone(v.Owners), slices.Clone(v.Nicknames), slices.Clone(v.AllowedTasks)
	return v
}

// OnChange calls f with the new values after every change.
func (s *Store) OnChange(f func(Values)) {
	s.mu.Lock()
	s.subs = append(s.subs, f)
	s.mu.Unlock()
}

// Set changes key to value (JSON) for by.
func (s *Store) Set(key string, value json.RawMessage, by string) error {
	if !slices.Contains(Keys, key) {
		return fmt.Errorf("%w: %q", ErrUnknownKey, key)
	}
	if by == ByAgent && slices.Contains(ownersOnly, key) {
		return fmt.Errorf("%w: %s", ErrNotAllowed, key)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, value); err != nil {
		return fmt.Errorf("%w for %s: %v", ErrBadValue, key, err)
	}
	if bytes.Equal(compact.Bytes(), []byte("null")) {
		return fmt.Errorf("%w for %s: null (Reset returns to the config's value)", ErrBadValue, key)
	}
	return s.change(key, by, func(o map[string]Override) {
		o[key] = Override{Value: compact.Bytes(), By: by, At: s.now()}
	})
}

// Reset returns key to the config's value.
func (s *Store) Reset(key, by string) error {
	if !slices.Contains(Keys, key) {
		return fmt.Errorf("%w: %q", ErrUnknownKey, key)
	}
	if by == ByAgent && slices.Contains(ownersOnly, key) {
		return fmt.Errorf("%w: %s", ErrNotAllowed, key)
	}
	return s.change(key, by, func(o map[string]Override) { delete(o, key) })
}

func (s *Store) change(key, by string, f func(map[string]Override)) error {
	s.mu.Lock()
	if p, ok := s.f.Pins[key]; ok && by == ByAgent {
		s.mu.Unlock()
		return fmt.Errorf("%w: %s, by %s", ErrPinned, key, p.By)
	}
	next := maps.Clone(s.f.Overrides)
	f(next)
	cur, err := s.build(next)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	old := s.f.Overrides
	s.f.Overrides = next
	if err := s.save(); err != nil {
		s.f.Overrides = old
		s.mu.Unlock()
		return err
	}
	before := lookup(s.cur, key)
	s.cur = cur
	subs := slices.Clone(s.subs)
	s.mu.Unlock()
	s.log.Info("setting changed", "key", key, "from", before, "to", lookup(cur, key), "by", by)
	for _, sub := range subs {
		sub(s.Values())
	}
	return nil
}

// Pin keeps the agent from changing key. Only an owner or the panel may.
func (s *Store) Pin(key, by string) error {
	return s.pin(key, by, true)
}

// Unpin lets the agent change key again.
func (s *Store) Unpin(key, by string) error {
	return s.pin(key, by, false)
}

func (s *Store) pin(key, by string, on bool) error {
	if !slices.Contains(Keys, key) {
		return fmt.Errorf("%w: %q", ErrUnknownKey, key)
	}
	if by != ByPanel && !strings.HasPrefix(by, "owner:") {
		return ErrNotAllowed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old := maps.Clone(s.f.Pins)
	if on {
		s.f.Pins[key] = Pin{By: by, At: s.now()}
	} else {
		delete(s.f.Pins, key)
	}
	if err := s.save(); err != nil {
		s.f.Pins = old
		return err
	}
	s.log.Info("setting pinned", "key", key, "pinned", on, "by", by)
	return nil
}

// Warnings are the config's warnings (config.Warnings) for the settings
// in force.
func (s *Store) Warnings() []string {
	v := s.Values()
	c := s.base
	c.Owners, c.Nicknames, c.AllowedTasks, c.Home, c.Overload = v.Owners, v.Nicknames, v.AllowedTasks, v.Home, v.Overload
	return c.Warnings()
}

// Entry describes one setting: its value now, whether that is the
// config's or a change (with who made it and when), and any pin.
type Entry struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
	// Config is the config file's value, what Reset returns to.
	Config json.RawMessage `json:"config"`
	Set    *Override       `json:"set,omitempty"`
	Pin    *Pin            `json:"pin,omitempty"`
}

// Entries lists every setting, in the order of Keys.
func (s *Store) Entries() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	base, _ := s.build(nil)
	var out []Entry
	for _, k := range Keys {
		e := Entry{Key: k, Value: lookup(s.cur, k), Config: lookup(base, k)}
		if o, ok := s.f.Overrides[k]; ok {
			e.Set = &o
		}
		if p, ok := s.f.Pins[k]; ok {
			e.Pin = &p
		}
		out = append(out, e)
	}
	return out
}

// build applies overrides to the config's values and checks the result.
func (s *Store) build(overrides map[string]Override) (Values, error) {
	b := s.base
	v := Values{Owners: b.Owners, Nicknames: b.Nicknames, AllowedTasks: b.AllowedTasks,
		Voice: b.LLM != nil, Home: b.Home, Overload: b.Overload}
	tree := map[string]any{}
	raw, err := json.Marshal(v)
	if err != nil {
		return Values{}, err
	}
	if err := json.Unmarshal(raw, &tree); err != nil {
		return Values{}, err
	}
	for _, k := range slices.Sorted(maps.Keys(overrides)) {
		var val any
		if err := json.Unmarshal(overrides[k].Value, &val); err != nil {
			return Values{}, fmt.Errorf("%w for %s: %v", ErrBadValue, k, err)
		}
		// Null would decode to the zero value and quietly turn a rule off.
		if val == nil {
			return Values{}, fmt.Errorf("%w for %s: null", ErrBadValue, k)
		}
		put(tree, k, val)
	}
	raw, err = json.Marshal(tree)
	if err != nil {
		return Values{}, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var out Values
	if err := dec.Decode(&out); err != nil {
		return Values{}, fmt.Errorf("%w: %v", ErrBadValue, err)
	}
	// The same rules as the config file.
	c := b
	c.Owners, c.Nicknames, c.AllowedTasks, c.Home, c.Overload = out.Owners, out.Nicknames, out.AllowedTasks, out.Home, out.Overload
	if err := c.Validate(); err != nil {
		return Values{}, fmt.Errorf("%w: %v", ErrBadValue, err)
	}
	return out, nil
}

// put sets the dotted path k in tree.
func put(tree map[string]any, k string, val any) {
	parts := strings.Split(k, ".")
	for _, p := range parts[:len(parts)-1] {
		next, ok := tree[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			tree[p] = next
		}
		tree = next
	}
	tree[parts[len(parts)-1]] = val
}

// lookup returns the dotted path k in v, as JSON.
func lookup(v Values, k string) json.RawMessage {
	raw, _ := json.Marshal(v)
	var tree any
	_ = json.Unmarshal(raw, &tree)
	for _, p := range strings.Split(k, ".") {
		m, ok := tree.(map[string]any)
		if !ok {
			return nil
		}
		tree = m[p]
	}
	out, _ := json.Marshal(tree)
	return out
}

// save writes the file beside itself and renames it into place. s.mu is held.
func (s *Store) save() error {
	if s.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(s.f, "", "  ")
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
