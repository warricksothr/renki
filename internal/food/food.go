// Package food maintains the bot's food slots through ordinary use_item tasks.
package food

import (
	"context"
	"log/slog"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

const Retry = 30 * time.Second
const Notice = 5 * time.Minute

type Session interface {
	State() *protocol.State
	Handshake() (*protocol.Hello, *protocol.Welcome, bool)
	Allowed(string) bool
	Dispatch(context.Context, session.Order) (*session.Task, error)
}
type Keeper struct {
	s                        Session
	owners                   func() []string
	log                      *slog.Logger
	now                      func() time.Time
	hello                    *protocol.Hello
	world                    string
	state                    *protocol.State
	fresh, next, asked, sent time.Time
	pending                  *session.Task
	sentSeq                  uint64
}

func New(s Session, owners func() []string, log *slog.Logger, now func() time.Time) *Keeper {
	if now == nil {
		now = time.Now
	}
	if log == nil {
		log = slog.Default()
	}
	return &Keeper{s: s, owners: owners, log: log, now: now}
}
func (k *Keeper) Run(ctx context.Context) {
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			k.Tick(ctx)
		}
	}
}
func (k *Keeper) Tick(ctx context.Context) {
	hello, _, connected := k.s.Handshake()
	st := k.s.State()
	now := k.now()
	if !connected || st == nil || st.Phase != protocol.PhaseInWorld || st.HP <= 0 || st.Food == nil {
		return
	}
	if hello != k.hello || st.World != k.world {
		if st.World != k.world {
			k.asked = time.Time{}
		}
		k.hello, k.world = hello, st.World
		k.pending = nil
		k.next = time.Time{}
		k.state = nil
	}
	if st != k.state {
		k.state, k.fresh = st, now
	}
	if now.Sub(k.fresh) > 3*time.Second || !k.s.Allowed(protocol.TaskUseItem) {
		return
	}
	if k.pending != nil {
		select {
		case <-k.pending.Done():
			status, reason, _ := k.pending.Status()
			if st.Seq <= k.sentSeq {
				return
			} // Wait for the post-action observation.
			k.pending = nil
			if status != protocol.StatusSucceeded {
				k.next = now.Add(Retry)
				k.log.Debug("food refused", "reason", reason)
			} else {
				k.next = now.Add(2 * time.Second)
			}
		default:
			if now.Sub(k.sent) > 10*time.Second {
				k.pending = nil
				k.next = now.Add(Retry)
			}
		}
		return
	}
	if now.Before(k.next) || slices.Contains(st.Flags, protocol.FlagAttacking) || slices.Contains(st.Flags, protocol.FlagSleeping) ||
		st.Task != nil && (st.Task.Task == protocol.TaskFlee || st.Task.Task == protocol.TaskAttack) {
		return
	}
	if item := Choose(st.Food); item != "" {
		t, err := k.s.Dispatch(ctx, session.Order{Task: protocol.TaskUseItem, Args: &protocol.UseItemArgs{Item: item}, TimeoutS: 5})
		k.next = now.Add(Retry)
		if err != nil {
			k.log.Debug("could not eat", "err", err)
			return
		}
		k.pending, k.sent, k.sentSeq = t, now, st.Seq
		k.log.Info("eating carried food", "prefab", item)
		return
	}
	if len(st.Food.Active) > 0 || len(st.Food.Carried) > 0 || st.HPMax <= 0 || st.HP/st.HPMax >= .8 || len(k.owners()) == 0 || !k.s.Allowed(protocol.TaskSay) ||
		!k.asked.IsZero() && now.Sub(k.asked) < Notice {
		return
	}
	_, err := k.s.Dispatch(ctx, session.Order{Task: protocol.TaskSay, Args: &protocol.SayArgs{Text: strings.Join(k.owners(), ", ") + ", I'm low on health and out of carried food. Please give me food or point me to a stocked chest."}, TimeoutS: 5})
	if err == nil {
		k.asked = now
	}
}

// Choose fills unused slots before refreshing an existing food; within each
// group prefer health and then regeneration. Only game-eligible food is used.
func Choose(status *protocol.FoodStatus) string {
	if status == nil {
		return ""
	}
	active := map[string]bool{}
	for _, f := range status.Active {
		active[f.Prefab] = true
	}
	var best *protocol.CarriedFood
	for i := range status.Carried {
		f := &status.Carried[i]
		if !f.CanEat || f.Prefab == "" || f.Health <= 0 || math.IsNaN(f.Health) || math.IsInf(f.Health, 0) {
			continue
		}
		if best == nil || len(status.Active) < 3 && active[best.Prefab] && !active[f.Prefab] ||
			(len(status.Active) >= 3 || active[f.Prefab] == active[best.Prefab]) && (f.Health > best.Health || f.Health == best.Health && f.Regen > best.Regen) {
			best = f
		}
	}
	if best == nil {
		return ""
	}
	return best.Prefab
}
