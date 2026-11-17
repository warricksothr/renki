// Package food maintains the bot's food slots through ordinary use_item tasks.
package food

import (
	"context"
	"encoding/json"
	"log/slog"
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
	StateObservation() (*protocol.State, time.Time)
	Handshake() (*protocol.Hello, *protocol.Welcome, bool)
	Allowed(string) bool
	Dispatch(context.Context, session.Order) (*session.Task, error)
	DispatchFor(context.Context, session.Order, *protocol.Hello, string) (*session.Task, error)
}
type Keeper struct {
	policy            func() Policy
	work              func() string
	lastDecision      string
	reserved          func() bool
	s                 Session
	owners            func() []string
	log               *slog.Logger
	now               func() time.Time
	hello             *protocol.Hello
	world             string
	next, asked, sent time.Time
	pending           *session.Task
	sentSeq           uint64
}

// SetPolicy is configured before Run; callbacks return immutable snapshots.
func (k *Keeper) SetPolicy(policy func() Policy, work func() string) { k.policy, k.work = policy, work }

func (k *Keeper) decision(st *protocol.State) Decision {
	p, work := Policy{}, ""
	if k.policy != nil {
		p = k.policy()
	}
	if k.work != nil {
		work = k.work()
	}
	return Decide(st, p, work)
}

// SetReserved is configured before Run. A food goal then owns its own eating
// steps and receipts rather than racing this background keeper.
func (k *Keeper) SetReserved(check func() bool) { k.reserved = check }

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
	if k.reserved != nil && k.reserved() {
		return
	}
	hello, _, connected := k.s.Handshake()
	st, at := k.s.StateObservation()
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
	}
	if at.IsZero() || now.Sub(at) < 0 || now.Sub(at) > 3*time.Second || !k.s.Allowed(protocol.TaskUseItem) {
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
	if now.Before(k.next) || slices.Contains(st.Flags, protocol.FlagAttacking) || slices.Contains(st.Flags, protocol.FlagSleeping) || slices.Contains(st.Flags, protocol.FlagInBed) ||
		st.Task != nil && (st.Task.Task == protocol.TaskFlee || st.Task.Task == protocol.TaskAttack) {
		return
	}
	d := k.decision(st)
	raw, _ := json.Marshal(d)
	if string(raw) != k.lastDecision {
		k.log.Info("food decision", "decision", d, "world", st.World, "worldUid", st.WorldUID, "stateSeq", st.Seq, "observationAt", at)
		k.lastDecision = string(raw)
	}
	if item := d.Item; item != "" {
		t, err := k.s.DispatchFor(ctx, session.Order{Task: protocol.TaskUseItem, Args: &protocol.UseItemArgs{Item: item}, TimeoutS: 5, Origin: "food:" + d.Mode + ":" + d.Reason}, hello, st.World)
		k.next = now.Add(Retry)
		if err != nil {
			k.log.Debug("could not eat", "err", err)
			return
		}
		k.pending, k.sent, k.sentSeq = t, now, st.Seq
		k.log.Info("eating carried food", "prefab", item, "taskId", t.ID, "decision", d, "stateSeq", st.Seq, "worldUid", st.WorldUID)
		return
	}
	if len(st.Food.Active) > 0 || len(st.Food.Carried) > 0 || st.HPMax <= 0 || st.HP/st.HPMax >= .8 || len(k.owners()) == 0 || !k.s.Allowed(protocol.TaskSay) ||
		!k.asked.IsZero() && now.Sub(k.asked) < Notice {
		return
	}
	_, err := k.s.DispatchFor(ctx, session.Order{Task: protocol.TaskSay, Args: &protocol.SayArgs{Text: strings.Join(k.owners(), ", ") + ", I'm low on health and out of carried food. Please give me food or point me to a stocked chest."}, TimeoutS: 5}, hello, st.World)
	if err == nil {
		k.asked = now
	}
}

// Choose fills unused slots before refreshing an existing food; within each
// group prefer health and then regeneration. Only game-eligible food is used.
func Choose(status *protocol.FoodStatus) string {
	return Decide(&protocol.State{Phase: protocol.PhaseInWorld, HP: 1, Food: status}, Policy{}, "").Item
}
