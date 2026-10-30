package session

import (
	"math"
	"time"

	"github.com/warricksothr/renki/internal/bridge"
	"github.com/warricksothr/renki/internal/protocol"
)

// PlayerChatObservation captures scope atomically before asynchronous work.
type PlayerChatObservation struct {
	Hello *protocol.Hello
	World string
	At    time.Time
}

type PlayerChatListener interface {
	PlayerChat(*protocol.Chat, PlayerChatObservation)
}

func (s *Session) ObservePlayerChat(c *bridge.Conn, chat *protocol.Chat, now time.Time) (PlayerChatObservation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.state
	if s.conn == nil || s.conn != c || st == nil || st.Phase != protocol.PhaseInWorld || st.World == "" || st.HP <= 0 || now.Sub(s.stateAt) < 0 || now.Sub(s.stateAt) > 3*time.Second || math.IsNaN(chat.T) || math.IsInf(chat.T, 0) || math.Abs(chat.T-st.T) > 5 || chat.Seq <= st.Seq || chat.Seq <= s.playerChatSeq {
		return PlayerChatObservation{}, false
	}
	switch chat.Channel {
	case protocol.ChannelNormal, protocol.ChannelShout, protocol.ChannelWhisper:
	default:
		return PlayerChatObservation{}, false
	}
	s.playerChatSeq = chat.Seq
	return PlayerChatObservation{Hello: c.Hello, World: st.World, At: now}, true
}
