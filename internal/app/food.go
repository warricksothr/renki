package app

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"slices"
	"time"

	"github.com/warricksothr/renki/internal/goal"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

type escortSupplyRequest struct {
	args     protocol.EscortArgs
	hello    *protocol.Hello
	world    string
	revision uint64
}

func (a *App) startFood(ctx context.Context, by string, escort *protocol.EscortArgs) (string, error) {
	return a.startFoodFor(ctx, by, escort, nil)
}
func (a *App) startFoodFor(ctx context.Context, by string, escort *protocol.EscortArgs, expected *escortSupplyRequest) (string, error) {
	a.goalMu.Lock()
	defer a.goalMu.Unlock()
	v := a.Settings.Values()
	if !slices.Contains(v.Players.Commanders(v.Owners), by) {
		return "", errors.New("food resupply requires a current owner or operator")
	}
	s, at := a.Session.StateObservation()
	revision := a.Session.ControlRevision()
	if expected != nil {
		hello, _, connected := a.Session.Handshake()
		if !connected || hello != expected.hello || s == nil || s.World != expected.world || revision != expected.revision {
			return "", errors.New("escort supply request superseded")
		}
	}
	if s == nil || s.Phase != protocol.PhaseInWorld || time.Since(at) > 3*time.Second || s.Task != nil && !protocol.Final(s.Task.Status) {
		return "", errors.New("fresh idle body required; stop current work first")
	}
	world, h, ok := a.Home.Current()
	if !ok || world != s.World || math.Hypot(s.Pos.FlatDistance(h.Pos), s.Pos[1]-h.Pos[1]) > 150 {
		return "", errors.New("known home within 150 m required")
	}
	for _, i := range a.Behaviours.List() {
		if i.Pending || i.Running {
			return "", errors.New("another behavior owns or has queued the body")
		}
	}
	for _, task := range []string{protocol.TaskMoveTo, protocol.TaskWithdraw, protocol.TaskInteract, protocol.TaskPickup, protocol.TaskUseItem, protocol.TaskStop} {
		if !a.Session.Allowed(task) {
			return "", errors.New(task + " permission required for food resupply")
		}
	}
	g, err := goal.NewRecord(goal.Spec{Kind: "food_resupply", World: s.World, Center: h.Pos, Radius: 150, Escort: escort}, time.Now().UTC())
	if err != nil {
		return "", err
	}
	if err = a.Goals.Add(g); err != nil {
		return "", err
	}
	if a.Session.ControlRevision() != revision {
		_ = a.goalTemplate.Cancel(g.ID)
		return "", errors.New("newer order replaced food resupply")
	}
	if err = a.goalAction(ctx, g.ID, "start"); err != nil {
		return "", err
	}
	if a.Session.ControlRevision() != revision {
		_ = a.goalAction(ctx, g.ID, "stop")
		return "", errors.New("newer order replaced food resupply")
	}
	return "I'll check home storage, forage for missing food, hunt only if permitted and safe, then return to cook and recover with a small food reserve. Tell me stop to cancel.", nil
}

func (a *App) queueEscortSupply(t *session.Task) bool {
	status, reason, detail := t.Status()
	if t.Name != protocol.TaskEscort || status != protocol.StatusFailed || reason != protocol.ReasonRefused || detail != "escort needs food resupply" {
		return false
	}
	var args protocol.EscortArgs
	if json.Unmarshal(t.Args, &args) != nil || !args.Resupply {
		return false
	}
	hello, _, connected := a.Session.Handshake()
	s := a.Session.State()
	if !connected || s == nil {
		return true
	}
	q := escortSupplyRequest{args, hello, s.World, a.Session.ControlRevision()}
	select {
	case a.escortSupply <- q:
	default:
	}
	return true
}
func (a *App) runEscortSupply(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case q := <-a.escortSupply:
			until := time.Now().Add(5 * time.Second)
			for time.Now().Before(until) {
				hello, _, connected := a.Session.Handshake()
				s := a.Session.State()
				if !connected || hello != q.hello || s == nil || s.World != q.world || a.Session.ControlRevision() != q.revision {
					break
				}
				if s.Task == nil || protocol.Final(s.Task.Status) {
					_, err := a.startFoodFor(ctx, q.args.Player, &q.args, &q)
					if err != nil {
						a.supplyLog.Warn("escort food resupply stopped", "reason", err)
					}
					break
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(100 * time.Millisecond):
				}
			}
		}
	}
}
