package protocol

import "math"

// WorldEventStatus is one observed in-world sample. Missing means unavailable;
// an empty object means the game reported no random or locally active event.
type WorldEventStatus struct {
	RandomEvent *AreaEvent   `json:"randomEvent,omitempty"`
	LocalEvent  *EventNotice `json:"localEvent,omitempty"`
}

type EventNotice struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Kind    string `json:"kind"` // random, or local (forced/boss/zone; bounds unknown)
}

type AreaEvent struct {
	Name          string  `json:"name"`
	Message       string  `json:"message"`
	Center        Vec3    `json:"center"`
	Radius        float64 `json:"radius"`
	ElapsedS      float64 `json:"elapsedS"`
	DurationS     float64 `json:"durationS"`
	TimerMayPause bool    `json:"timerMayPause"`
	Inside        bool    `json:"inside"`
}

func eventFinite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

func (e *AreaEvent) Valid() bool {
	return e != nil && e.Name != "" && eventFinite(e.Radius) && e.Radius > 0 &&
		eventFinite(e.Center[0]) && eventFinite(e.Center[1]) && eventFinite(e.Center[2]) &&
		eventFinite(e.ElapsedS) && e.ElapsedS >= 0 && eventFinite(e.DurationS)
}

// Contains follows Valheim's horizontal, strict-radius area test, including its
// dungeon-height exclusion. Timer estimates never establish that an event ended.
func (e *AreaEvent) Contains(p Vec3) bool {
	return e.Valid() && eventFinite(p[0]) && eventFinite(p[1]) && eventFinite(p[2]) &&
		p[1] <= 3000 && math.Hypot(p[0]-e.Center[0], p[2]-e.Center[2]) < e.Radius
}

func (s *WorldEventStatus) Valid() bool {
	return s != nil && (s.RandomEvent == nil || s.RandomEvent.Valid()) &&
		(s.LocalEvent == nil || s.LocalEvent.Name != "" &&
			(s.LocalEvent.Kind == "random" || s.LocalEvent.Kind == "local"))
}
