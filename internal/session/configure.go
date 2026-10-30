package session

import (
	"context"
	"github.com/warricksothr/renki/internal/protocol"
)

// Configure changes session options without taking over the body or touching
// its control revision. A reconnect receives the persisted choice in welcome.
func (s *Session) Configure(ctx context.Context, c protocol.Config) error {
	return s.ConfigureConnection(ctx, 0, c)
}

// ConfigureConnection refuses to send a delayed configuration to a replacement
// connection. Zero selects the current connection for ordinary settings.
func (s *Session) ConfigureConnection(ctx context.Context, connection uint64, c protocol.Config) error {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil || connection != 0 && conn.ID() != connection {
		return ErrNotConnected
	}
	return conn.Send(ctx, &c)
}
