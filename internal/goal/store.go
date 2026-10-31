// Package goal preserves objectives above bounded behavior runners.
package goal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

const MaxRecords = 64
const MaxJSON = 16 << 10
const MaxSource = 16 << 10

type Outcome struct {
	Status string    `json:"status"`
	Reason string    `json:"reason,omitempty"`
	Detail string    `json:"detail,omitempty"`
	TaskID string    `json:"taskId,omitempty"`
	Entity string    `json:"entity,omitempty"`
	At     time.Time `json:"at"`
}
type Intent struct {
	Template    string          `json:"template"`
	Params      json.RawMessage `json:"params"`
	Entity      string          `json:"entity"`
	TaskID      string          `json:"taskId,omitempty"`
	Fingerprint string          `json:"fingerprint"`
	Search      *SearchPlan     `json:"search,omitempty"`
}

// A sector is an attempted bearing, not proof that its entire area was seen.
// The host chooses plans; policy state cannot reset this ledger.
type SearchPlan struct {
	Sector int           `json:"sector"`
	Origin protocol.Vec3 `json:"origin"`
	Radius float64       `json:"radius"`
}
type SearchAttempt struct {
	Plan    SearchPlan `json:"plan"`
	Outcome Outcome    `json:"outcome"`
	Empty   bool       `json:"empty"`
}
type Observation struct {
	At      time.Time     `json:"at"`
	World   string        `json:"world"`
	Pos     protocol.Vec3 `json:"pos"`
	HP      float64       `json:"hp"`
	HPMax   float64       `json:"hpMax"`
	Stamina float64       `json:"stamina"`
	Target  string        `json:"target,omitempty"`
	Ready   bool          `json:"ready"`
}
type Record struct {
	Observation     *Observation    `json:"observation,omitempty"`
	ID              string          `json:"id"`
	Kind            string          `json:"kind"`
	World           string          `json:"world"`
	Center          protocol.Vec3   `json:"center"`
	Radius          float64         `json:"radius"`
	Policy          string          `json:"policy,omitempty"`
	PolicyHash      string          `json:"policyHash"`
	State           json.RawMessage `json:"state,omitempty"`
	Status          string          `json:"status"`
	Reason          string          `json:"reason,omitempty"`
	Created         time.Time       `json:"created"`
	Deadline        time.Time       `json:"deadline"`
	Attempts        int             `json:"attempts"`
	TaskCount       int             `json:"taskCount"`
	MaxAttempts     int             `json:"maxAttempts"`
	MaxTasks        int             `json:"maxTasks"`
	WaitS           float64         `json:"waitS"`
	Wake            time.Time       `json:"wake,omitempty"`
	Pending         *Intent         `json:"pending,omitempty"`
	Last            *Outcome        `json:"last,omitempty"`
	History         []Outcome       `json:"history,omitempty"`
	Proof           *Outcome        `json:"proof,omitempty"`
	LastFingerprint string          `json:"lastFingerprint,omitempty"`
	Repeated        int             `json:"repeated"`
	Searches        []SearchAttempt `json:"searches,omitempty"`
}
type Store struct {
	mu      sync.Mutex
	path    string
	records map[string]Record
	// write is injectable for checkpoint failure tests.
	write func([]byte) error
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, records: map[string]Record{}}
	s.write = s.saveFile
	if path == "" {
		return s, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if len(raw) > 4<<20 {
		return nil, errors.New("goal file too large")
	}
	var disk struct {
		Version int               `json:"version"`
		Records map[string]Record `json:"records"`
	}
	if err = json.Unmarshal(raw, &disk); err != nil {
		return nil, err
	}
	if disk.Version != 1 || len(disk.Records) > MaxRecords {
		return nil, errors.New("invalid goal store version or count")
	}
	for id, r := range disk.Records {
		if id != r.ID || r.PolicyHash != hash(r.Policy) {
			return nil, errors.New("goal identity/policy checksum mismatch")
		}
		if err = validateRecord(r); err != nil {
			return nil, fmt.Errorf("goal %s: %w", id, err)
		}
		if r.Status == "running" || r.Status == "queued" {
			r.Status = "suspended"
			r.Reason = "controller restarted; explicit resume required"
			if r.Pending != nil {
				r.Status = "needs_plan"
				r.Reason = "uncertain action after restart; not replayed"
			}
		}
		s.records[id] = r
	}
	if err = s.persist(s.records); err != nil {
		return nil, err
	}
	return s, nil
}
func hash(source string) string { h := sha256.Sum256([]byte(source)); return hex.EncodeToString(h[:]) }
func clone(r Record) Record {
	b, _ := json.Marshal(r)
	var c Record
	_ = json.Unmarshal(b, &c)
	return c
}
func (s *Store) Get(id string) (Record, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[id]
	return clone(r), ok
}
func (s *Store) List() []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Record, 0, len(s.records))
	for _, r := range s.records {
		out = append(out, clone(r))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created.After(out[j].Created) })
	return out
}
func (s *Store) update(id string, f func(*Record) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.records[id]
	if !ok {
		return errors.New("unknown goal")
	}
	next := clone(old)
	if err := f(&next); err != nil {
		return err
	}
	if err := validateRecord(next); err != nil {
		return err
	}
	records := make(map[string]Record, len(s.records))
	for k, v := range s.records {
		records[k] = v
	}
	records[id] = next
	if err := s.persist(records); err != nil {
		return err
	}
	s.records = records
	return nil
}
func (s *Store) Add(r Record) error {
	if err := validateRecord(r); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.records) >= MaxRecords {
		return errors.New("goal record limit reached")
	}
	if _, ok := s.records[r.ID]; ok {
		return errors.New("goal already exists")
	}
	records := make(map[string]Record, len(s.records)+1)
	for k, v := range s.records {
		records[k] = v
	}
	records[r.ID] = clone(r)
	if err := s.persist(records); err != nil {
		return err
	}
	s.records = records
	return nil
}
func (s *Store) persist(records map[string]Record) error {
	b, err := json.MarshalIndent(struct {
		Version int               `json:"version"`
		Records map[string]Record `json:"records"`
	}{1, records}, "", "  ")
	if err != nil {
		return err
	}
	if len(b) > 4<<20 {
		return errors.New("goal store size limit reached")
	}
	return s.write(b)
}
func (s *Store) saveFile(b []byte) error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".goals-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return replaceFile(f.Name(), s.path)
}
func terminal(status string) bool {
	return status == "succeeded" || status == "cancelled" || status == "needs_plan"
}
func validateRecord(r Record) error {
	if r.ID == "" || len(r.ID) > 64 || r.Kind != "combat_trial" || r.World == "" || len(r.World) > 200 || r.Radius < 60 || r.Radius > 250 || !finite(r.Radius) {
		return errors.New("invalid goal identity/world/area")
	}
	for _, n := range r.Center {
		if !finite(n) {
			return errors.New("invalid goal center")
		}
	}
	if r.MaxAttempts < 1 || r.MaxAttempts > 8 || r.MaxTasks < 1 || r.MaxTasks > 8 || r.Attempts < 0 || r.Attempts > r.MaxAttempts || r.TaskCount < 0 || r.TaskCount > r.MaxTasks || r.WaitS < 0 || r.WaitS > 120 || !finite(r.WaitS) {
		return errors.New("invalid goal budgets")
	}
	if r.Created.IsZero() || !r.Deadline.After(r.Created) || r.Deadline.Sub(r.Created) > 20*time.Minute {
		return errors.New("invalid goal deadline")
	}
	switch r.Status {
	case "ready", "queued", "running", "suspended", "needs_plan", "cancelled", "succeeded":
	default:
		return errors.New("invalid goal status")
	}
	if len(r.Policy) > MaxSource || r.PolicyHash != hash(r.Policy) || len(r.State) > MaxJSON || !json.Valid(r.State) || len(r.History) > 16 {
		return errors.New("invalid policy/checkpoint")
	}
	if len(r.Searches) > r.Attempts || len(r.Searches) > 8 {
		return errors.New("invalid search ledger size")
	}
	seen := map[int]bool{}
	plans := make([]SearchPlan, 0, len(r.Searches)+1)
	for _, search := range r.Searches {
		if search.Outcome.TaskID == "" || (search.Empty && !emptySearch(search.Outcome)) {
			return errors.New("invalid search receipt")
		}
		plans = append(plans, search.Plan)
	}
	if r.Pending != nil && r.Pending.Search != nil {
		plans = append(plans, *r.Pending.Search)
	}
	for _, plan := range plans {
		if plan.Sector < 0 || plan.Sector > 7 || seen[plan.Sector] || !finite(plan.Radius) || plan.Radius < 12 || plan.Radius > 60 {
			return errors.New("invalid or repeated search sector")
		}
		for _, n := range plan.Origin {
			if !finite(n) {
				return errors.New("invalid search origin")
			}
		}
		if distance(plan.Origin, r.Center)+plan.Radius+52 > r.Radius+1e-6 {
			return errors.New("search plan exceeds goal envelope")
		}
		seen[plan.Sector] = true
	}
	return nil
}

// Summaries keep policy source and full checkpoints out of every status frame.
func (s *Store) Summaries() []Record {
	out := s.List()
	for i := range out {
		out[i].Policy = ""
		out[i].State = nil
		if len(out[i].History) > 3 {
			out[i].History = out[i].History[len(out[i].History)-3:]
		}
	}
	return out
}
