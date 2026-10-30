package session

import (
	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/protocol"
	"time"
)

type CameraFrame struct {
	protocol.Screenshot
	Connection uint64    `json:"connection"`
	ReceivedAt time.Time `json:"receivedAt"`
}

func (s *Session) ClearScreenshot() { s.mu.Lock(); s.screenshot = nil; s.mu.Unlock() }
func (s *Session) rememberScreenshot(c *bridge.Conn, image *protocol.Screenshot) {
	if image.Validate() != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != c || s.state == nil || s.state.Phase != protocol.PhaseInWorld || image.World != s.state.World || image.WorldUID != s.state.WorldUID {
		return
	}
	now := time.Now().UTC()
	if s.screenshot != nil && (image.Seq <= s.screenshot.Seq || now.Sub(s.screenshot.ReceivedAt) < time.Second) {
		return
	}
	frame := *image
	frame.JPEG = append([]byte(nil), image.JPEG...)
	s.screenshot = &CameraFrame{Screenshot: frame, Connection: c.ID(), ReceivedAt: now}
}
func (s *Session) Screenshot() *CameraFrame {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil || s.screenshot == nil || s.state == nil || s.state.Phase != protocol.PhaseInWorld {
		return nil
	}
	frame := *s.screenshot
	frame.JPEG = append([]byte(nil), frame.JPEG...)
	return &frame
}
