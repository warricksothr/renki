package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
)

// Instances returns independent copies of the named behaviour definitions,
// with each definition's author and time. The engine validates templates
// and their parameter schemas; this store only owns durable JSON objects.
func (s *Store) Instances() map[string]Override {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]Override, len(s.f.Instances))
	for name, entry := range s.f.Instances {
		entry.Value = bytes.Clone(entry.Value)
		out[name] = entry
	}
	return out
}

// SetInstance saves an engine-validated definition in settings.json. Names
// are display names, not paths. A failed save leaves the old definition.
func (s *Store) SetInstance(name string, definition json.RawMessage, by string) error {
	if name == "" || len(name) > 80 || strings.TrimSpace(name) != name || by == "" {
		return fmt.Errorf("%w: instance needs a name (1–80 characters) and source", ErrBadValue)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, definition); err != nil {
		return fmt.Errorf("%w: %v", ErrBadValue, err)
	}
	raw := compact.Bytes()
	if len(raw) == 0 || len(raw) > 64<<10 || raw[0] != '{' {
		return fmt.Errorf("%w: instance must be a JSON object, at most 64 KiB", ErrBadValue)
	}
	s.mu.Lock()
	old := s.f.Instances
	next := maps.Clone(old)
	if next == nil {
		next = map[string]Override{}
	}
	next[name] = Override{Value: raw, By: by, At: s.now()}
	s.f.Instances = next
	if err := s.save(); err != nil {
		s.f.Instances = old
		s.mu.Unlock()
		return err
	}
	previous := old[name]
	s.mu.Unlock()
	s.log.Info("instance changed", "instance", name, "from", string(previous.Value), "to", string(raw), "by", by)
	return nil
}

// DeleteInstance removes a named definition. Runtime Stop is separate: a
// stopped instance remains saved so it can be started again.
func (s *Store) DeleteInstance(name, by string) error {
	if by == "" {
		return fmt.Errorf("%w: instance change needs a source", ErrBadValue)
	}
	s.mu.Lock()
	old := s.f.Instances
	previous, ok := old[name]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("%w: instance %q", ErrUnknownKey, name)
	}
	next := maps.Clone(old)
	delete(next, name)
	s.f.Instances = next
	if err := s.save(); err != nil {
		s.f.Instances = old
		s.mu.Unlock()
		return err
	}
	s.mu.Unlock()
	s.log.Info("instance deleted", "instance", name, "from", string(previous.Value), "by", by)
	return nil
}
