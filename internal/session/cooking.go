package session

import "github.com/warricksothr/renki/internal/protocol"

// CookingTraces is bounded evidence from this task's current connection.
func (t *Task) CookingTraces() []*protocol.Trace {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]*protocol.Trace(nil), t.cooking...)
}
