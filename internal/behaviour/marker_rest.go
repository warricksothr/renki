package behaviour

import (
	"encoding/json"
	"errors"
	"math"
	"slices"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// MarkerRest is optional idle work; native bed use owns all sleep refusals.
// Waiting by a bed completes its move and leaves the body available for work.
type MarkerRest struct{}
type MarkerRestParams struct {
	Marker Marker `json:"marker"`
	Sleep  bool   `json:"sleep"`
}

func (MarkerRest) Validate(raw json.RawMessage) error {
	var p MarkerRestParams
	if err := Decode(raw, &p); err != nil {
		return err
	}
	return p.Marker.validate()
}
func (t MarkerRest) Start(raw json.RawMessage) (Runner, error) {
	if err := t.Validate(raw); err != nil {
		return nil, err
	}
	var p MarkerRestParams
	_ = Decode(raw, &p)
	return &markerRestRun{p: p}, nil
}
func MarkerRestDefinition(marker Marker, sleep bool) Definition {
	raw, _ := json.Marshal(MarkerRestParams{Marker: marker, Sleep: sleep})
	dwell, priority := 120.0, -30
	if sleep {
		dwell, priority = 15, 0
	}
	return Definition{Template: "marker_rest", Params: raw, Trigger: Trigger{Kind: "idle", Marker: &marker, DwellS: dwell},
		When: Conditions{Night: &sleep}, Repeat: Repeat{Mode: "trigger", CooldownS: 300, MaxRuns: 100}, Enabled: true, Priority: priority}
}

type markerRestRun struct {
	p         MarkerRestParams
	marker    *home.Waypoint
	home      *protocol.Vec3
	bed       *protocol.PerceivedStructure
	sent      bool
	beforeSeq uint64
	began     time.Time
	health    healthSample
}

func resting(s *protocol.State) bool {
	return s != nil && (slices.Contains(s.Flags, protocol.FlagInBed) || slices.Contains(s.Flags, protocol.FlagSleeping))
}
func (r *markerRestRun) TravelFromIdleMarker() bool { return true }
func (r *markerRestRun) Check(v View) error {
	if !idleFresh(v) || v.Perception == nil || v.PerceptionAt.IsZero() || v.Now.Sub(v.PerceptionAt) < 0 ||
		v.Now.Sub(v.PerceptionAt) > 3*time.Second || math.Abs(v.State.T-v.Perception.T) > 3 {
		return errors.New("rest needs fresh state and surroundings")
	}
	s := v.State
	if v.Home == nil {
		return errors.New("rest needs known home")
	}
	for _, pos := range []protocol.Vec3{s.Pos, *v.Home} {
		for _, n := range pos {
			if !finite(n) {
				return errors.New("rest needs finite home and position")
			}
		}
	}
	w, ok := r.p.Marker.resolve(v)
	if !ok || r.marker != nil && w != *r.marker || r.home != nil && *v.Home != *r.home ||
		distance(s.Pos, *v.Home) > 20 || !slices.Contains(s.Flags, protocol.FlagInShelter) ||
		!resting(s) && !slices.Contains(s.Flags, protocol.FlagGrounded) ||
		s.Location != nil && s.Location.InWater || slices.Contains(s.Flags, protocol.FlagSwimming) {
		return errors.New("rest needs unchanged home context and dry sheltered footing")
	}
	if s.Time.Day < 1 || s.Time.Night != r.p.Sleep {
		return errors.New("rest day/night mode changed or clock unknown")
	}
	for _, c := range v.Perception.Characters {
		if c.HPFrac > 0 && (c.TargetingMe || c.Hostile && distance(c.Pos, s.Pos) < 12) {
			return errors.New("nearby threat interrupts rest")
		}
	}
	if r.marker != nil {
		now := sampleHealth(v)
		if v.Now.Sub(r.began) > 90*time.Second || r.health.lost(now) {
			return errors.New("rest injured or time budget exhausted")
		}
		// Resting heals, so a hit after healing past the start still counts.
		r.health = r.health.roll(now)
	}
	if r.bed != nil {
		for _, b := range v.Perception.Structures {
			if b.ID == r.bed.ID && b.Kind == protocol.StructureBed && b.Mine && b.Pos == r.bed.Pos {
				return nil
			}
		}
		return errors.New("owned bed changed or is no longer observed")
	}
	return nil
}
func (r *markerRestRun) Next(v View, result *Result) (*session.Order, error) {
	if err := r.Check(v); err != nil {
		return nil, err
	}
	if resting(v.State) {
		if r.sent && (result == nil || result.Status != protocol.StatusSucceeded || v.State.Seq <= r.beforeSeq) {
			return nil, ErrWait
		}
		if r.bed != nil && (distance(v.State.Pos, r.bed.Pos) > 2 || math.Abs(v.State.Pos[1]-r.bed.Pos[1]) > .75) {
			return nil, errors.New("sleep attachment is not beside the selected owned bed")
		}
		return nil, nil
	}
	if r.marker == nil {
		w, _ := r.p.Marker.resolve(v)
		h := *v.Home
		r.marker, r.home, r.began, r.health = &w, &h, v.Now, sampleHealth(v)
		for _, b := range v.Perception.Structures {
			if b.ID != "" && b.Kind == protocol.StructureBed && b.Mine && distance(b.Pos, h) <= 12 &&
				math.Abs(b.Pos[1]-v.State.Pos[1]) <= 1.25 && (r.bed == nil || distance(b.Pos, v.State.Pos) < distance(r.bed.Pos, v.State.Pos)) {
				bed := b
				r.bed = &bed
			}
		}
		if r.bed == nil {
			return nil, errors.New("no observed owned bed near home; no player's bed was taken")
		}
	}
	if r.sent {
		if result == nil || result.Status != protocol.StatusSucceeded {
			return nil, errors.New("rest lacks native success")
		}
		if v.State.Seq <= r.beforeSeq {
			return nil, ErrWait
		}
		if r.p.Sleep {
			return nil, ErrWait
		} // a sleep receipt alone is not bed attachment
		if !besideBedOnFloor(v.State.Pos, r.bed.Pos) {
			return nil, errors.New("bedside move lacks observed arrival on the bed floor")
		}
		return nil, nil
	}
	if !v.Idle {
		return nil, errors.New("another body order owns rest time")
	}
	if !r.p.Sleep && besideBedOnFloor(v.State.Pos, r.bed.Pos) {
		return nil, nil
	}
	r.sent, r.beforeSeq = true, v.State.Seq
	if r.p.Sleep {
		return &session.Order{Task: protocol.TaskInteract, Args: &protocol.InteractArgs{Entity: r.bed.ID, Sleep: true}, TimeoutS: 60}, nil
	}
	within := 2.0
	return &session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Entity: r.bed.ID}, StopDistance: &within, Dry: true, AvoidHostiles: true}, TimeoutS: 60}, nil
}

func besideBedOnFloor(pos, bed protocol.Vec3) bool {
	// Match native movement's horizontal arrival radius, retaining the
	// stricter floor band so a lower storey cannot count as bedside waiting.
	return math.Hypot(pos[0]-bed[0], pos[2]-bed[2]) <= 2 && math.Abs(pos[1]-bed[1]) <= .75
}
