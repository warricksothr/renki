package behaviour

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

const woodInstance = "Owner wood gathering"
const woodSource = "chat:wood:"

// StartWood queues one manual cycle. The saved definition remembers the last
// requested site, but never resumes work merely because the controller restarts.
// The caller must authorize the player before calling this entry point.
func (e *Engine) StartWood(ctx context.Context, by string) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if strings.TrimSpace(by) == "" {
		return "", errors.New("a player must request the work")
	}
	revision := e.o.Session.ControlRevision()
	v := e.view()
	pack := e.o.Session.PackSnapshot()
	if v.State != nil && pack.World == v.State.World {
		v.Inventory = pack.Inventory
	} else {
		v.Inventory = nil
	}
	if v.State == nil || v.State.Phase != protocol.PhaseInWorld {
		return "", errors.New("I need to be in the world before gathering")
	}
	if !v.Idle || e.active != "" {
		return "", errors.New("stop my current task first")
	}
	for _, i := range e.instances {
		if i.pending {
			return "", errors.New("stop the queued behavior first")
		}
	}
	for _, task := range []string{protocol.TaskMoveTo, protocol.TaskHarvest, protocol.TaskDeposit, protocol.TaskInteract, protocol.TaskStop} {
		if !slices.Contains(e.o.Session.AllowedTasks(), task) {
			return "", fmt.Errorf("%s is not allowed", task)
		}
	}
	var last *GatherParams
	if i := e.instances[woodInstance]; i != nil {
		saved := e.o.Store.Instances()[woodInstance]
		if i.definition.Template != "gather" || !strings.HasPrefix(saved.By, woodSource) {
			return "", errors.New("rename the existing Owner wood gathering behavior before using this command")
		}
		var p GatherParams
		if err := Decode(i.definition.Params, &p); err == nil {
			last = &p
		}
	}
	var hints []home.RememberedContainer
	if e.o.Containers != nil && v.State != nil {
		hints = e.o.Containers(v.State.World)
	}
	var landmarks []home.Landmark
	if e.o.Landmarks != nil {
		landmarks = e.o.Landmarks(v.State.World)
	}
	p, remembered, err := planWood(v, last, hints, landmarks...)
	if err != nil {
		return "", err
	}
	params, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	d := Definition{Template: "gather", Params: params, Trigger: Trigger{Kind: "manual"}, Repeat: Repeat{Mode: "once"}, Priority: 10}
	if err := validate(d, e.o.Templates); err != nil {
		return "", err
	}
	if err := e.put(ctx, woodInstance, d, woodSource+by); err != nil {
		return "", err
	}
	if err := e.action(ctx, woodInstance, "start", woodSource+by); err != nil {
		return "", err
	}
	// Planning/settings writes must not resurrect work after a newer order.
	// Tick cannot dispatch while this lock is held; a later revision also
	// invalidates the pending instance through its normal owner-order check.
	if e.o.Session.ControlRevision() != revision {
		_ = e.action(ctx, woodInstance, "stop", woodSource+by)
		return "", errors.New("a newer order replaced this request")
	}
	where := "a nearby safe beech"
	if remembered {
		where = "my last wood work site"
	}
	return "I'll gather up to two beeches at " + where + ", then return home and store the new wood. I'll return sooner if my pack or axe needs it. Tell me stop to cancel.", nil
}

func planWood(v View, last *GatherParams, hints []home.RememberedContainer, landmarks ...home.Landmark) (GatherParams, bool, error) {
	p := GatherParams{MaxTrees: 2, ReturnHome: true}
	fail := func(message string) (GatherParams, bool, error) { return p, false, errors.New(message) }
	if v.State == nil || v.State.Phase != protocol.PhaseInWorld || v.State.World == "" || !freshPerception(v) ||
		v.Inventory == nil || v.StateAt.IsZero() || v.PerceptionAt.IsZero() ||
		v.Now.Sub(v.StateAt) < 0 || v.Now.Sub(v.StateAt) > 3*time.Second ||
		v.Now.Sub(v.PerceptionAt) < 0 || v.Now.Sub(v.PerceptionAt) > 5*time.Second {
		return fail("I need fresh observations and inventory in this world")
	}
	if err := gatherReady(v); err != nil {
		return p, false, err
	}
	if !finite(v.State.Stamina) || v.State.Stamina < 40 {
		return fail("I need at least 40 stamina before gathering")
	}
	if v.Home == nil || distance(v.State.Pos, *v.Home) > 150 {
		return fail("I need a known home within 150 m")
	}
	p.Site = Place{World: v.State.World, Radius: 20}
	var best *protocol.PerceivedResource
	for _, r := range v.Perception.Resources {
		if r.ID == "" || r.Kind != protocol.ResourceTree || r.Prefab != "Beech1" || r.MinTool == nil || *r.MinTool != 0 ||
			distance(v.State.Pos, r.Pos) > 60 || distance(r.Pos, *v.Home) > 150 || !gatherClear(v, r) {
			continue
		}
		if best == nil || targetCost(v, r.Pos) < targetCost(v, best.Pos) ||
			targetCost(v, r.Pos) == targetCost(v, best.Pos) && r.ID < best.ID {
			copy := r
			best = &copy
		}
	}
	remembered := last != nil && last.Site.World == v.State.World && distance(last.Site.Pos, *v.Home) <= 150
	if remembered {
		// Fresh visible safe trees near the old site confirm it. A distant
		// remembered site is only a destination hint; gather rechecks on arrival.
		var confirmed *protocol.PerceivedResource
		for _, r := range v.Perception.Resources {
			if r.ID != "" && r.Kind == protocol.ResourceTree && r.Prefab == "Beech1" && r.MinTool != nil && *r.MinTool == 0 &&
				distance(r.Pos, last.Site.Pos) <= 20 && distance(r.Pos, *v.Home) <= 150 && gatherClear(v, r) &&
				(confirmed == nil || targetCost(v, r.Pos) < targetCost(v, confirmed.Pos) ||
					targetCost(v, r.Pos) == targetCost(v, confirmed.Pos) && r.ID < confirmed.ID) {
				copy := r
				confirmed = &copy
			}
		}
		remembered = confirmed != nil || distance(v.State.Pos, last.Site.Pos) > 60
		if confirmed != nil {
			// Center travel on an observed tree so it falls within native
			// acquisition even when it is at the old site's outer edge.
			copy := *last
			copy.Site.Pos = confirmed.Pos
			last = &copy
		}
	}
	if remembered {
		p.Site.Pos = last.Site.Pos
	} else if best != nil {
		p.Site.Pos = best.Pos
	} else {
		var hint *home.Landmark
		for _, l := range landmarks {
			// A locally visible absence/unsafe tree supersedes memory. Distant
			// hints are travel destinations only, with full revalidation there.
			if l.Kind != protocol.ResourceTree || l.Prefab != "Beech1" || l.MinTool == nil || *l.MinTool != 0 ||
				l.At.After(v.Now) || v.Now.Sub(l.At) > 7*24*time.Hour || distance(l.Pos, *v.Home) > 150 || distance(l.Pos, v.State.Pos) <= 60 ||
				!gatherClear(v, protocol.PerceivedResource{Kind: protocol.ResourceTree, Pos: l.Pos}) {
				continue
			}
			if hint == nil || targetCost(v, l.Pos) < targetCost(v, hint.Pos) || targetCost(v, l.Pos) == targetCost(v, hint.Pos) && l.Entity < hint.Entity {
				copy := l
				hint = &copy
			}
		}
		if hint == nil {
			return fail("I don't know a safe beech work site nearby; trees near characters or buildings must stay standing")
		}
		p.Site.Pos = hint.Pos
	}
	positions := map[string]protocol.Vec3{}
	for _, c := range hints {
		positions[c.Entity] = c.Pos
	}
	for _, c := range v.Containers {
		positions[c.Entity] = c.Pos
	}
	for _, c := range v.Perception.Structures {
		if c.Kind == protocol.StructureContainer {
			positions[c.ID] = c.Pos
		}
	}
	for id, pos := range positions {
		if id != "" && distance(pos, *v.Home) <= 10 {
			p.To = append(p.To, id)
		}
	}
	sort.Slice(p.To, func(i, j int) bool {
		a, b := distance(positions[p.To[i]], *v.Home), distance(positions[p.To[j]], *v.Home)
		return a < b || a == b && p.To[i] < p.To[j]
	})
	if len(p.To) == 0 {
		return fail("I need known storage within 10 m of home")
	}
	if len(p.To) > 16 {
		p.To = p.To[:16]
	}
	return p, remembered, nil
}
