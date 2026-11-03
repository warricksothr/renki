package goal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/atlas"
	"github.com/warricksothr/renki/internal/behaviour"
	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

type SurveyLeg struct {
	From   protocol.Vec3   `json:"from"`
	To     protocol.Vec3   `json:"to"`
	Final  protocol.Vec3   `json:"final"`
	TaskID string          `json:"taskId"`
	Place  string          `json:"place,omitempty"`
	Status string          `json:"status"`
	Reason string          `json:"reason,omitempty"`
	Travel float64         `json:"travel"`
	Trail  []protocol.Vec3 `json:"trail"`
	At     time.Time       `json:"at"`
}
type SurveyProgress struct {
	Pinned       home.BasePlaces   `json:"pinned"`
	WorldKey     string            `json:"worldKey"`
	Source       string            `json:"source"`
	Connection   uint64            `json:"connection"`
	Phase        string            `json:"phase"`
	Travel       float64           `json:"travel"`
	Legs         []SurveyLeg       `json:"legs"`
	Coverage     atlas.Coverage    `json:"coverage"`
	Places       home.PlaceChanges `json:"places"`
	Targets      []home.Waypoint   `json:"targets"`
	Unresolved   []string          `json:"unresolved"`
	Target       int               `json:"target"`
	Breadcrumbs  []protocol.Vec3   `json:"breadcrumbs"`
	ReturnIndex  int               `json:"returnIndex"`
	Returned     bool              `json:"returned"`
	Health       float64           `json:"health"`
	Injury       *uint64           `json:"injury,omitempty"`
	LastScan     uint64            `json:"lastScan"`
	ObserveUntil time.Time         `json:"observeUntil"`
}

func NewSurvey(world, key, source string, connection uint64, p home.BasePlaces, pos protocol.Vec3, hp float64, injury *uint64, now time.Time) (Record, error) {
	var token [12]byte
	if _, err := rand.Read(token[:]); err != nil {
		return Record{}, err
	}
	s := &SurveyProgress{Pinned: p, WorldKey: key, Source: source, Connection: connection, Phase: "observe", ReturnIndex: -1, Health: hp, Injury: injury, Breadcrumbs: []protocol.Vec3{pos}, ObserveUntil: now.Add(2 * time.Second)}
	// Verify useful connections in both directions by deliberately revisiting
	// sleeping between storage sites, then the independent return place.
	s.Targets = append(s.Targets, p.Return)
	if p.Sleeping != nil {
		s.Targets = append(s.Targets, *p.Sleeping)
	}
	for _, storage := range p.Storage {
		s.Targets = append(s.Targets, storage)
		if p.Sleeping != nil {
			s.Targets = append(s.Targets, *p.Sleeping)
		}
	}
	s.Targets = append(s.Targets, p.Return)
	g := Record{ID: hex.EncodeToString(token[:]), Kind: "familiarize", World: world, Center: p.Base.Center, Radius: p.Base.Radius, Policy: combatPolicy, PolicyHash: hash(combatPolicy), State: json.RawMessage(`{}`), Status: "ready", Created: now, Deadline: now.Add(time.Duration(p.Base.DurationS * float64(time.Second))), MaxAttempts: 64, MaxTasks: 64, Survey: s}
	return g, validateSurvey(g)
}
func validateSurvey(g Record) error {
	s := g.Survey
	if s == nil || g.ID == "" || len(g.ID) > 64 || g.World == "" || len(g.World) > 128 || s.WorldKey == "" || len(s.WorldKey) != 64 || s.Source == "" || s.Connection == 0 || g.Center != s.Pinned.Base.Center || g.Radius != s.Pinned.Base.Radius {
		return errors.New("invalid familiarization identity/scope")
	}
	if err := s.Pinned.Base.Validate(); err != nil {
		return err
	}
	if g.Created.IsZero() || !g.Deadline.After(g.Created) || g.Deadline.Sub(g.Created) > 20*time.Minute || g.MaxTasks != 64 || g.MaxAttempts != 64 || g.TaskCount < 0 || g.TaskCount > 64 || g.Attempts < 0 || g.Attempts > 64 || s.Travel < 0 || !finite(s.Travel) || len(s.Legs) > 64 || len(s.Breadcrumbs) > 33 || len(s.Targets) > 35 || s.Target < 0 || s.Target > len(s.Targets) || s.ReturnIndex < -1 || s.ReturnIndex >= len(s.Breadcrumbs) {
		return errors.New("invalid familiarization budgets")
	}
	if g.Status != "ready" && g.Status != "queued" && g.Status != "running" && g.Status != "suspended" && !terminal(g.Status) {
		return errors.New("invalid familiarization status")
	}
	if s.Phase != "observe" && s.Phase != "survey" && s.Phase != "return" && s.Phase != "complete" {
		return errors.New("invalid familiarization phase")
	}
	if g.PolicyHash != hash(g.Policy) || len(g.Policy) > MaxSource || len(g.State) > MaxJSON || !json.Valid(g.State) {
		return errors.New("invalid familiarization checkpoint")
	}
	for _, p := range append(append([]protocol.Vec3{}, s.Breadcrumbs...), s.Pinned.Return.Pos) {
		for _, n := range p {
			if !finite(n) {
				return errors.New("nonfinite survey position")
			}
		}
		if !s.Pinned.Base.Contains(p) {
			return errors.New("survey position outside scope")
		}
	}
	for _, leg := range s.Legs {
		if len(leg.Trail) > 128 || !finite(leg.Travel) || leg.Travel < 0 || leg.TaskID == "" {
			return errors.New("invalid survey leg")
		}
	}
	if s.Health <= 0 || !finite(s.Health) {
		return errors.New("invalid survey health")
	}
	return nil
}

func (r *runner) surveyCheck(g Record, v behaviour.View) error {
	s := g.Survey
	if terminal(g.Status) {
		return nil
	}
	if v.State == nil || v.State.World != g.World || v.State.Phase != protocol.PhaseInWorld {
		return errors.New("survey needs its original live world")
	}
	if v.StateAt.IsZero() || v.Now.Sub(v.StateAt) < 0 || v.Now.Sub(v.StateAt) > 3*time.Second {
		return fmt.Errorf("survey needs fresh state (age %.3f seconds)", v.Now.Sub(v.StateAt).Seconds())
	}
	if v.PerceptionAt.IsZero() || v.Now.Sub(v.PerceptionAt) < 0 || v.Now.Sub(v.PerceptionAt) > 5*time.Second {
		return fmt.Errorf("survey needs fresh surroundings (age %.3f seconds)", v.Now.Sub(v.PerceptionAt).Seconds())
	}
	if !behaviour.PerceptionBeside(v, 3) {
		return errors.New("survey state and surroundings are not from compatible game times")
	}
	if !v.Now.Before(g.Deadline) {
		return errors.New("survey deadline exhausted")
	}
	if r.t.Memory == nil || r.t.Session == nil || r.t.Atlas == nil || r.t.Atlas() == nil {
		return errors.New("survey memory or atlas unavailable")
	}
	world, conn, scope, _ := r.t.Session.TerrainView()
	if scope == nil || conn != s.Connection || world != g.World || atlas.Key(*scope) != s.WorldKey || scope.Source != s.Source {
		return errors.New("survey observer/session changed")
	}
	base := r.t.Memory.ActiveBase(g.World)
	if base == nil || !reflect.DeepEqual(*base, s.Pinned.Base) {
		return errors.New("base settings changed; start a new survey")
	}
	var rawHome *protocol.Vec3
	if h, ok := r.t.Memory.Home(g.World); ok {
		p := h.Pos
		rawHome = &p
	}
	current, err := home.ResolveBase(*base, r.t.Memory.Waypoints(g.World), rawHome)
	if err != nil || !reflect.DeepEqual(current, s.Pinned) {
		return errors.New("a pinned base place changed; start a new survey")
	}
	p := v.State
	if !base.Contains(p.Pos) || !finite(p.HP) || !finite(p.HPMax) || p.HPMax <= 0 || p.HP < math.Max(20, p.HPMax*.6) || p.HP < s.Health-.01 || p.Stranded() || p.Encumbered() || slices.Contains(p.Flags, protocol.FlagSwimming) || p.Location != nil && p.Location.InWater {
		return errors.New("survey interrupted by health, footing, capacity or boundary")
	}
	if (s.Injury == nil) != (v.InjurySeq == nil) || s.Injury != nil && *s.Injury != *v.InjurySeq {
		return errors.New("injury interrupts survey")
	}
	for _, c := range v.Perception.Characters {
		if c.HPFrac > 0 && (c.TargetingMe || c.Hostile) && c.Pos.FlatDistance(p.Pos) < 15 {
			return errors.New("nearby threat interrupts survey")
		}
	}
	if !slices.Contains(v.Features, "bounded_move_v1") {
		return errors.New("plugin needs bounded movement support")
	}
	return nil
}

func surveyArrived(pos, target protocol.Vec3) bool {
	return pos.FlatDistance(target) <= 1.5 && math.Abs(pos[1]-target[1]) <= 1.25
}
func mergeNames(a, b []string) []string {
	for _, n := range b {
		if !slices.Contains(a, n) && len(a) < 128 {
			a = append(a, n)
		}
	}
	return a
}
func (r *runner) surveyObserve(g Record, v behaviour.View) (Record, []atlas.Sample, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	coverage, points, err := r.t.Atlas().SurveyCoverage(ctx, g.Survey.WorldKey, g.Survey.Source, g.Center, g.Radius, g.Created, v.Now)
	if err != nil {
		return g, nil, err
	}
	SortedSurveySamples(points)
	changes := home.PlaceChanges{}
	if v.Perception.Seq != g.Survey.LastScan {
		changes, err = r.t.Memory.RefreshBasePlaces(g.World, g.Survey.Pinned.Base, home.ObservedPlaces(v.Perception, v.PerceptionAt))
		if err != nil {
			return g, nil, err
		}
	}
	err = r.t.Store.update(g.ID, func(g *Record) error {
		s := g.Survey
		s.Coverage = coverage
		s.LastScan = v.Perception.Seq
		s.Places.Added = mergeNames(s.Places.Added, changes.Added)
		s.Places.Updated = mergeNames(s.Places.Updated, changes.Updated)
		s.Places.Protected = mergeNames(s.Places.Protected, changes.Protected)
		s.Places.Skipped += changes.Skipped
		s.Places.Unchanged += changes.Unchanged
		return nil
	})
	g, _ = r.t.Store.Get(g.ID)
	return g, points, err
}

func (r *runner) surveyNext(g Record, v behaviour.View, last *behaviour.Result) (*session.Order, error) {
	if terminal(g.Status) {
		return nil, nil
	}
	if err := r.surveyCheck(g, v); err != nil {
		return r.need(err.Error())
	}
	s := g.Survey
	if last != nil && g.Pending != nil {
		if last.TaskID == "" || last.TaskID != g.Pending.TaskID || g.Pending.Run != r.t.Store.RunID() {
			return r.need("survey receipt does not match pending leg")
		}
		var receipt struct {
			Travel float64         `json:"travel"`
			Final  protocol.Vec3   `json:"final"`
			Trail  []protocol.Vec3 `json:"trail"`
			Detail string          `json:"detail"`
		}
		var args protocol.MoveToArgs
		if !strings.HasPrefix(last.Detail, "survey:") || json.Unmarshal([]byte(strings.TrimPrefix(last.Detail, "survey:")), &receipt) != nil || json.Unmarshal(g.Pending.Params, &args) != nil || args.To.Pos == nil || args.Scope == nil || !finite(receipt.Travel) || receipt.Travel < 0 || receipt.Travel > args.Scope.TravelM+3 || len(receipt.Trail) < 1 || len(receipt.Trail) > 128 {
			return r.need("missing or invalid measured native survey receipt; not replayed")
		}
		for _, p := range append(slices.Clone(receipt.Trail), receipt.Final) {
			for _, n := range p {
				if !finite(n) {
					return r.need("invalid survey trail")
				}
			}
			if !s.Pinned.Base.Contains(p) {
				return r.need("native survey left its authorized boundary")
			}
		}
		if v.State.Seq <= last.ReceiptSeq {
			return nil, behaviour.ErrWait
		}
		verified := last.Status == protocol.StatusSucceeded && surveyArrived(v.State.Pos, *args.To.Pos) && surveyArrived(receipt.Final, *args.To.Pos) && slices.Contains(v.State.Flags, protocol.FlagGrounded)
		status := last.Status
		reason := last.Reason
		if last.Status == protocol.StatusSucceeded && !verified {
			status = protocol.StatusFailed
			reason = "arrival not confirmed on target floor"
		}
		from := receipt.Trail[0]
		if err := r.t.Store.update(g.ID, func(g *Record) error {
			s := g.Survey
			s.Travel += receipt.Travel
			s.Legs = append(s.Legs, SurveyLeg{From: from, To: *args.To.Pos, Final: receipt.Final, TaskID: last.TaskID, Place: g.Pending.Entity, Status: status, Reason: reason + " " + receipt.Detail, Travel: receipt.Travel, Trail: receipt.Trail, At: v.Now})
			if s.Phase == "return" {
				if !verified {
					g.Status = "needs_plan"
					g.Reason = "return leg blocked: " + reason
				} else {
					s.ReturnIndex--
				}
			} else {
				if verified && len(s.Breadcrumbs) < 33 {
					s.Breadcrumbs = append(s.Breadcrumbs, receipt.Final)
				}
				s.ObserveUntil = v.Now.Add(2 * time.Second)
			}
			g.Pending = nil
			return nil
		}); err != nil {
			return nil, err
		}
		g, _ = r.t.Store.Get(g.ID)
		s = g.Survey
		if terminal(g.Status) {
			return nil, nil
		}
	}
	if g.Pending != nil {
		return r.need("survey has an uncertain leg; never replayed")
	}
	if v.Now.Before(s.ObserveUntil) {
		return nil, &behaviour.Delay{Until: s.ObserveUntil}
	}
	var points []atlas.Sample
	var err error
	g, points, err = r.surveyObserve(g, v)
	if err != nil {
		return r.need(err.Error())
	}
	s = g.Survey
	// Reserve a reverse walk over every outbound stop and its measured cost,
	// plus a contingency. Neither retries nor return can reset either ledger.
	reserveTime := math.Max(60, math.Max(s.Travel*.8, float64(len(s.Breadcrumbs))*8))
	reserveTravel := math.Max(30, s.Travel*1.5+20)
	mealRemaining := 0.0
	if v.State.Food != nil {
		for _, meal := range v.State.Food.Active {
			mealRemaining = math.Max(mealRemaining, meal.RemainingS)
		}
	}
	if s.Phase != "return" && (mealRemaining <= reserveTime+60 || v.State.Stamina < 20 || g.Deadline.Sub(v.Now).Seconds() <= reserveTime || s.Travel+reserveTravel >= s.Pinned.Base.TravelM || g.TaskCount >= 28) {
		if err = r.beginSurveyReturn(g); err != nil {
			return nil, err
		}
		g, _ = r.t.Store.Get(g.ID)
		s = g.Survey
	}
	if s.Phase == "return" {
		for s.ReturnIndex >= 0 && slices.Contains(v.State.Flags, protocol.FlagGrounded) && surveyArrived(v.State.Pos, s.Breadcrumbs[s.ReturnIndex]) {
			if err = r.t.Store.update(g.ID, func(g *Record) error { g.Survey.ReturnIndex--; return nil }); err != nil {
				return nil, err
			}
			g, _ = r.t.Store.Get(g.ID)
			s = g.Survey
		}
		if s.ReturnIndex >= 0 {
			return r.surveyOrder(g, v, s.Breadcrumbs[s.ReturnIndex], "return")
		}
		if !surveyArrived(v.State.Pos, s.Pinned.Return.Pos) {
			return r.surveyOrder(g, v, s.Pinned.Return.Pos, "return anchor")
		}
		if !slices.Contains(v.State.Flags, protocol.FlagGrounded) {
			return r.need("return lacks supported footing")
		}
		complete := s.Coverage.Unknown == 0 && s.Coverage.Refreshed == s.Coverage.Total && s.Target == len(s.Targets) && len(s.Unresolved) == 0 && s.Pinned.Sleeping != nil && len(s.Pinned.Storage) > 0
		for _, leg := range s.Legs {
			if leg.Status != protocol.StatusSucceeded {
				complete = false
			}
		}
		err = r.t.Store.update(g.ID, func(g *Record) error {
			g.Survey.Returned = true
			g.Survey.Phase = "complete"
			g.Status = "partial"
			if complete {
				g.Status = "succeeded"
			}
			g.Reason = surveySummary(*g)
			return nil
		})
		return nil, err
	}
	for s.Target < len(s.Targets) && slices.Contains(v.State.Flags, protocol.FlagGrounded) && surveyArrived(v.State.Pos, s.Targets[s.Target].Pos) {
		if err = r.t.Store.update(g.ID, func(g *Record) error { g.Survey.Target++; return nil }); err != nil {
			return nil, err
		}
		g, _ = r.t.Store.Get(g.ID)
		s = g.Survey
	}
	if s.Target == len(s.Targets) && s.Pinned.Sleeping != nil && len(s.Pinned.Storage) > 0 && s.Coverage.Unknown == 0 && s.Coverage.Refreshed == s.Coverage.Total {
		if err = r.beginSurveyReturn(g); err != nil {
			return nil, err
		}
		return nil, &behaviour.Delay{Until: v.Now.Add(time.Second)}
	}
	var target *protocol.Vec3
	name := "terrain approach"
	if s.Target < len(s.Targets) {
		w := s.Targets[s.Target]
		p := w.Pos
		name = w.Name
		if v.State.Pos.FlatDistance(p) <= 20 {
			target = &p
		} else {
			target = surveyApproach(g, v, points, &p)
		}
		if target == nil || surveyFailed(s, *target) {
			// Keep an honest unresolved role in the ledger; skip only this
			// attempt, not the need to report the missing connection.
			if err = r.t.Store.update(g.ID, func(g *Record) error {
				g.Survey.Unresolved = mergeNames(g.Survey.Unresolved, []string{w.Name})
				g.Survey.Target++
				g.Reason = "unreachable approach to " + w.Name
				return nil
			}); err != nil {
				return nil, err
			}
			return nil, &behaviour.Delay{Until: v.Now.Add(time.Second)}
		}
	} else {
		target = surveyApproach(g, v, points, nil)
	}
	if target == nil {
		if err = r.beginSurveyReturn(g); err != nil {
			return nil, err
		}
		return nil, &behaviour.Delay{Until: v.Now.Add(time.Second)}
	}
	return r.surveyOrder(g, v, *target, name)
}
func surveyTried(s *SurveyProgress, p protocol.Vec3) bool {
	for _, leg := range s.Legs {
		if leg.Place != "return" && leg.Place != "return anchor" && leg.To.FlatDistance(p) < 4 {
			return true
		}
	}
	return false
}
func surveyFailed(s *SurveyProgress, p protocol.Vec3) bool {
	for _, leg := range s.Legs {
		if leg.Status != protocol.StatusSucceeded && leg.To.FlatDistance(p) < 4 {
			return true
		}
	}
	return false
}
func surveyApproach(g Record, v behaviour.View, points []atlas.Sample, toward *protocol.Vec3) *protocol.Vec3 {
	best := -math.MaxFloat64
	var result *protocol.Vec3
	for _, sample := range points {
		p := sample.Pos
		d := v.State.Pos.FlatDistance(p)
		if d < 6 || d > 16 || p[1] < sample.Water+.5 || math.Abs(p[1]-v.State.Pos[1]) > 8 || v.Now.Sub(sample.Observed) > time.Minute || toward == nil && surveyTried(g.Survey, p) || toward != nil && surveyFailed(g.Survey, p) {
			continue
		}
		score := d
		if toward != nil {
			if p.FlatDistance(*toward) >= v.State.Pos.FlatDistance(*toward)-2 {
				continue
			}
			score = -p.FlatDistance(*toward)
		} else {
			nearest := g.Radius
			for _, leg := range g.Survey.Legs {
				if leg.Status == protocol.StatusSucceeded {
					nearest = math.Min(nearest, p.FlatDistance(leg.Final))
				}
			}
			score += nearest * 3
		}
		if score > best {
			copy := p
			result = &copy
			best = score
		}
	}
	return result
}
func (r *runner) beginSurveyReturn(g Record) error {
	return r.t.Store.update(g.ID, func(g *Record) error {
		g.Survey.Phase = "return"
		g.Survey.ReturnIndex = len(g.Survey.Breadcrumbs) - 1
		return nil
	})
}
func (r *runner) surveyOrder(g Record, v behaviour.View, to protocol.Vec3, name string) (*session.Order, error) {
	if g.TaskCount >= 64 || g.Survey.Travel >= g.Survey.Pinned.Base.TravelM || !g.Survey.Pinned.Base.Contains(to) {
		return r.need("survey task or travel budget exhausted")
	}
	remaining := g.Deadline.Sub(v.Now).Seconds()
	if remaining < 3 {
		return r.need("not enough time for a bounded survey leg")
	}
	duration := math.Min(40, remaining-1)
	travel := math.Min(100, g.Survey.Pinned.Base.TravelM-g.Survey.Travel)
	if g.Survey.Phase != "return" {
		reserveTime := math.Max(60, math.Max(g.Survey.Travel*.8, float64(len(g.Survey.Breadcrumbs))*8))
		reserveTravel := math.Max(30, g.Survey.Travel*1.5+20)
		duration = math.Min(duration, remaining-reserveTime-1)
		travel = math.Min(travel, (g.Survey.Pinned.Base.TravelM-g.Survey.Travel-reserveTravel)/2.5)
		required := math.Sqrt(math.Pow(to[0]-v.State.Pos[0], 2) + math.Pow(to[1]-v.State.Pos[1], 2) + math.Pow(to[2]-v.State.Pos[2], 2))
		if duration < 2 || travel < math.Max(2, required+2) {
			if err := r.beginSurveyReturn(g); err != nil {
				return nil, err
			}
			return nil, &behaviour.Delay{Until: v.Now.Add(time.Second)}
		}
	}
	if travel < 1 {
		return r.need("survey travel budget exhausted")
	}
	stop := 1.0
	args := &protocol.MoveToArgs{To: protocol.Target{Pos: &to}, StopDistance: &stop, Dry: true, AvoidHostiles: true, Scope: &protocol.MoveScope{Center: g.Center, Radius: g.Radius, DurationS: duration, TravelM: travel}}
	raw, _ := json.Marshal(args)
	if err := r.t.Store.update(g.ID, func(g *Record) error {
		g.TaskCount++
		g.Attempts++
		if g.Survey.Phase != "return" {
			g.Survey.Phase = "survey"
		}
		g.Pending = &Intent{Run: r.t.Store.RunID(), At: v.Now, Template: "survey_leg", Params: raw, Entity: name}
		return nil
	}); err != nil {
		return nil, err
	}
	return &session.Order{Task: protocol.TaskMoveTo, Args: args, TimeoutS: duration + 1}, nil
}
func surveySummary(g Record) string {
	s := g.Survey
	blocked := 0
	for _, leg := range s.Legs {
		if leg.Status != protocol.StatusSucceeded {
			blocked++
		}
	}
	roles := ""
	if s.Pinned.Sleeping == nil || len(s.Pinned.Storage) == 0 {
		roles = " Sleeping/storage roles need binding."
	}
	return fmt.Sprintf("Survey %s: %d/%d terrain samples measured, %d new, %d refreshed; %d unknown, %d blocked legs. %.0f m travelled across %d recorded legs; %d places added, %d refreshed, %d protected. Returned: %t.%s", g.Status, s.Coverage.Measured, s.Coverage.Total, s.Coverage.New, s.Coverage.Refreshed, s.Coverage.Unknown, blocked, s.Travel, len(s.Legs), len(s.Places.Added), len(s.Places.Updated), len(s.Places.Protected), s.Returned, roles)
}
func (r *runner) Summary() string {
	g, ok := r.t.Store.Get(r.id)
	if !ok {
		return "goal unavailable"
	}
	if g.Survey != nil {
		return surveySummary(g)
	}
	return g.Reason
}

func SurveySummary(g Record) string {
	if g.Survey == nil {
		return "not a survey"
	}
	return surveySummary(g)
}

// SurveyReady validates admission before a durable goal or engine instance is
// written. The runner repeats these checks during every native leg.
func (t *Template) SurveyReady(g Record) error {
	if t.Session == nil || t.Memory == nil {
		return errors.New("survey host unavailable")
	}
	state, at := t.Session.StateObservation()
	_, world, seen, seenAt := t.Session.PerceptionObservation()
	hello, _, connected := t.Session.Handshake()
	if !connected || hello == nil || state == nil || world != state.World {
		return errors.New("fresh connected survey observations required")
	}
	var h *protocol.Vec3
	if saved, ok := t.Memory.Home(world); ok {
		p := saved.Pos
		h = &p
	}
	v := behaviour.View{Now: time.Now().UTC(), State: state, StateAt: at, Perception: seen, PerceptionAt: seenAt, Features: hello.Features, Home: h, Waypoints: t.Memory.Waypoints(world)}
	if hit := t.Session.InjuryStatus(); hit != nil {
		n := hit.Sequence
		v.InjurySeq = &n
	}
	if err := (&runner{t: t, id: g.ID}).surveyCheck(g, v); err != nil {
		return err
	}
	if state.Stamina < 40 || !slices.Contains(state.Flags, protocol.FlagGrounded) {
		return errors.New("survey needs dry supported footing and at least 40 stamina")
	}
	food := false
	if state.Food != nil {
		for _, meal := range state.Food.Active {
			if meal.RemainingS >= 300 {
				food = true
			}
		}
	}
	if !food {
		return errors.New("survey needs an active meal with at least five minutes remaining")
	}
	return nil
}

// SortedSurveySamples removes dependence on SQLite iteration order when two
// approach candidates have the same score.
func SortedSurveySamples(points []atlas.Sample) {
	sort.Slice(points, func(i, j int) bool {
		if points[i].Pos[0] != points[j].Pos[0] {
			return points[i].Pos[0] < points[j].Pos[0]
		}
		return points[i].Pos[2] < points[j].Pos[2]
	})
}
