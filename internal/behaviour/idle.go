package behaviour

import (
	"errors"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
)

func (m Marker) validate() error {
	name, err := home.WaypointName(m.Name)
	if err != nil || name != m.Name || strings.TrimSpace(m.World) == "" || !slices.Contains(home.WaypointKinds, m.Kind) {
		return errors.New("marker needs world, canonical name and known kind")
	}
	return nil
}
func (m Marker) resolve(v View) (home.Waypoint, bool) {
	if v.State == nil || v.State.World != m.World {
		return home.Waypoint{}, false
	}
	for _, w := range v.Waypoints {
		if w.Name == m.Name && w.Kind == m.Kind {
			return w, true
		}
	}
	return home.Waypoint{}, false
}
func markerRadius(w home.Waypoint) float64 { return math.Min(40, math.Max(3, w.Radius)) }
func withinMarker(pos protocol.Vec3, w home.Waypoint) bool {
	// A large map circle must not admit another floor beneath a raised base.
	return math.Abs(pos[1]-w.Pos[1]) <= 2 && math.Hypot(pos[0]-w.Pos[0], pos[2]-w.Pos[2]) <= markerRadius(w)
}
func idleFresh(v View) bool {
	return v.State != nil && v.State.Phase == protocol.PhaseInWorld && v.State.HP > 0 &&
		!v.StateAt.IsZero() && v.Now.Sub(v.StateAt) >= 0 && v.Now.Sub(v.StateAt) <= 3*time.Second
}
func idleMarkerValid(i *instance, v View) bool {
	if !idleFresh(v) || i.idleMarker == nil {
		return false
	}
	w, ok := i.definition.Trigger.Marker.resolve(v)
	journey, travels := i.runner.(IdleJourney)
	return ok && w == *i.idleMarker && (withinMarker(v.State.Pos, w) || travels && journey.TravelFromIdleMarker())
}
func (e *Engine) idleTrigger(name string, i *instance, v View, now time.Time) {
	if e.active == name {
		return
	}
	w, ok := i.definition.Trigger.Marker.resolve(v)
	if !i.definition.Enabled || !idleFresh(v) || resting(v.State) || !v.Idle || e.active != "" || !ok || !withinMarker(v.State.Pos, w) || !i.canStart(v) {
		if i.pending {
			e.ended(i, "idle marker unavailable")
		}
		i.pending = false
		i.idleSince = time.Time{}
		i.idleMarker = nil
		return
	}
	if i.idleMarker == nil || *i.idleMarker != w {
		if i.pending {
			e.ended(i, "idle marker changed")
		}
		i.pending = false
		i.idleSince = now
		i.idleMarker = &w
	}
	if i.idleSince.IsZero() {
		i.idleSince = now
	}
	if !now.Before(i.next) && now.Sub(i.idleSince) >= duration(i.definition.Trigger.DwellS) {
		i.pending = true
	}
}
