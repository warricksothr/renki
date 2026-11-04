package presence

import (
	"context"
	"math"
	"slices"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func (m *Manager) inviteBedtime(ctx context.Context, live Live, current request, st *protocol.State, now time.Time) {
	mode := live.Presence.Bedtime
	if (mode != protocol.ChannelNormal && mode != protocol.ChannelShout) || m.o.ReserveBedtime == nil || now.Before(m.nextSay) || now.Before(m.nextBedtime) ||
		!st.Time.Night || st.Time.Day < 1 || math.IsNaN(st.Time.DayFraction) || math.IsInf(st.Time.DayFraction, 0) || st.Time.DayFraction < 0 || st.Time.DayFraction >= 1 ||
		st.WorldUID == "" || st.Character == "" || slices.Contains(st.Flags, protocol.FlagInBed) || slices.Contains(st.Flags, protocol.FlagSleeping) {
		return
	}
	// The evening and following pre-dawn belong to one night, despite midnight.
	night := st.Time.Day
	if st.Time.DayFraction < .5 {
		night--
	}
	if mode == protocol.ChannelShout {
		if st.OtherPlayers == nil || *st.OtherPlayers < 1 {
			return
		}
	} else {
		hello, world, seen, at := m.o.Session.PerceptionObservation()
		if hello != current.hello || world != current.world || seen == nil || now.Sub(at) < 0 || now.Sub(at) > 3*time.Second || math.Abs(st.T-seen.T) > 5 {
			return
		}
		audience := false
		for _, p := range seen.Characters {
			if p.Kind == protocol.CharacterPlayer && p.Name != "" && p.Name != st.Character && p.HPFrac > 0 && math.Hypot(p.Pos.FlatDistance(st.Pos), p.Pos[1]-st.Pos[1]) <= 15 {
				audience = true
				break
			}
		}
		if !audience {
			return
		}
	}
	reserved, err := m.o.ReserveBedtime(st.WorldUID, st.Character, st.Time.Day, night)
	if err != nil {
		m.nextBedtime = now.Add(10 * time.Second)
		m.o.Logger.Warn("bedtime invitation not reserved", "err", err)
		return
	}
	if !reserved {
		return
	}
	// A crash or uncertain send after this durable reservation cannot replay it.
	_, err = m.o.Session.DispatchFor(ctx, session.Order{Task: protocol.TaskSay, Args: &protocol.SayArgs{Text: "It's night. Does anyone want to go to bed?", Channel: mode}, TimeoutS: 5, Origin: "nightly bedtime invitation"}, current.hello, current.world)
	m.nextSay = now.Add(2 * time.Second)
	if err != nil {
		m.o.Logger.Warn("bedtime invitation attempt ended without a confirmed send", "err", err)
	}
}
