package session

import (
	"context"
	"github.com/warricksothr/renki/internal/protocol"
)

// Configure changes session options without taking over the body or touching
// its control revision. A reconnect receives the persisted choice in welcome.
func (s *Session) Configure(ctx context.Context, c protocol.Config) error {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return ErrNotConnected
	}
	return conn.Send(ctx, &c)
}
