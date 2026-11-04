package presence

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func bedtimeFixture(t *testing.T) (*Manager, *fake, *Live, *time.Time, *home.Store) {
	t.Helper()
	m, f, live, now := fixture(t)
	store, err := home.Open(filepath.Join(t.TempDir(), "memory.json"))
	if err != nil {
		t.Fatal(err)
	}
	m.o.ReserveBedtime = store.ReserveBedtime
	live.Presence.Bedtime = protocol.ChannelShout
	f.state.WorldUID = "123"
	f.state.Time = protocol.GameTime{Day: 213, DayFraction: .8, Night: true}
	others := 1
	f.state.OtherPlayers = &others
	return m, f, live, now, store
}

func TestBedtimeOncePerNightAndCalendarDayAcrossReconnectAndManagerRestart(t *testing.T) {
	m, f, live, now, store := bedtimeFixture(t)
	m.Tick(context.Background())
	if len(f.orders) != 1 || f.orders[0].Task != protocol.TaskSay || f.orders[0].Args.(*protocol.SayArgs).Channel != protocol.ChannelShout {
		t.Fatal(f.orders)
	}
	advance(f, now, 3*time.Second)
	m.Tick(context.Background())
	f.hello = &protocol.Hello{}
	m.Tick(context.Background())
	opts := m.o
	opts.ReserveBedtime = store.ReserveBedtime
	m = New(opts)
	f.state.Time = protocol.GameTime{Day: 214, DayFraction: .1, Night: true}
	m.Tick(context.Background())
	if len(f.orders) != 1 {
		t.Fatal("midnight/reconnect/restart repeated invitation", f.orders)
	}
	advance(f, now, 3*time.Second)
	f.state.Time = protocol.GameTime{Day: 214, DayFraction: .8, Night: true}
	m.Tick(context.Background())
	if len(f.orders) != 2 {
		t.Fatal("next evening not invited", f.orders)
	}
	advance(f, now, 3*time.Second)
	f.state.Time = protocol.GameTime{Day: 213, DayFraction: .8, Night: true}
	m.Tick(context.Background())
	live.Presence.Bedtime = protocol.ChannelNormal
	m.Tick(context.Background())
	if len(f.orders) != 2 {
		t.Fatal("rollback/mode change repeated invitation")
	}
}

func TestBedtimeDoesNotConsumeAnAbsentStaleOrSleepingScene(t *testing.T) {
	for _, name := range []string{"disabled", "day", "unknown day", "bad fraction", "unknown world", "alone", "unknown roster", "stale state", "offline", "sleeping", "in bed", "local absent", "local stale"} {
		t.Run(name, func(t *testing.T) {
			m, f, live, now, _ := bedtimeFixture(t)
			switch name {
			case "disabled":
				live.Presence.Bedtime = ""
			case "day":
				f.state.Time.Night = false
			case "unknown day":
				f.state.Time.Day = 0
			case "bad fraction":
				f.state.Time.DayFraction = math.NaN()
			case "unknown world":
				f.state.WorldUID = ""
			case "alone":
				*f.state.OtherPlayers = 0
			case "unknown roster":
				f.state.OtherPlayers = nil
			case "stale state":
				f.at = now.Add(-4 * time.Second)
			case "offline":
				f.connected = false
			case "sleeping":
				f.state.Flags = []string{protocol.FlagSleeping}
			case "in bed":
				f.state.Flags = []string{protocol.FlagInBed}
			case "local absent":
				live.Presence.Bedtime = protocol.ChannelNormal
			case "local stale":
				live.Presence.Bedtime = protocol.ChannelNormal
				f.seenAt = now.Add(-4 * time.Second)
			}
			m.Tick(context.Background())
			if len(f.orders) != 0 {
				t.Fatal("invalid scene invited", f.orders)
			}
			live.Presence.Bedtime = protocol.ChannelShout
			f.state.Time = protocol.GameTime{Day: 213, DayFraction: .8, Night: true}
			f.state.WorldUID = "123"
			f.state.Flags = nil
			f.connected = true
			f.at = *now
			others := 1
			f.state.OtherPlayers = &others
			m.Tick(context.Background())
			if len(f.orders) != 1 {
				t.Fatal("invalid scene consumed daily invitation", f.orders)
			}
		})
	}
}

func TestBedtimeLocalAudienceUsesFreshNearbyPlayersAndNormalChat(t *testing.T) {
	m, f, live, _, _ := bedtimeFixture(t)
	live.Presence.Bedtime = protocol.ChannelNormal
	f.seen.Characters = []protocol.PerceivedCharacter{{Kind: "player", Name: "Tasleroch", HPFrac: 1}, {Kind: "player", Name: "Far", HPFrac: 1, Pos: protocol.Vec3{16, 0, 0}}, {Kind: "creature", Name: "Deer", HPFrac: 1}}
	m.Tick(context.Background())
	if len(f.orders) != 0 {
		t.Fatal(f.orders)
	}
	f.seen.Characters = append(f.seen.Characters, protocol.PerceivedCharacter{Kind: "player", Name: "Friend", HPFrac: 1, Pos: protocol.Vec3{14, 0, 0}})
	m.Tick(context.Background())
	if len(f.orders) != 1 || f.orders[0].Args.(*protocol.SayArgs).Channel != protocol.ChannelNormal {
		t.Fatal(f.orders)
	}
}

func TestBedtimeUncertainSendCannotReplayAndFailureToReserveDoesNotSend(t *testing.T) {
	m, f, _, now, _ := bedtimeFixture(t)
	reserve := m.o.ReserveBedtime
	m.o.ReserveBedtime = func(string, string, int, int) (bool, error) { return false, errors.New("disk full") }
	m.Tick(context.Background())
	if len(f.orders) != 0 {
		t.Fatal("sent without durable reservation")
	}
	advance(f, now, 11*time.Second)
	m.o.ReserveBedtime = reserve
	f.sendError = session.ErrNotConnected
	m.Tick(context.Background())
	if len(f.orders) != 1 {
		t.Fatal(f.orders)
	}
	advance(f, now, 3*time.Second)
	f.sendError = nil
	m = New(m.o)
	m.Tick(context.Background())
	if len(f.orders) != 1 {
		t.Fatal("uncertain send replayed", f.orders)
	}
}
