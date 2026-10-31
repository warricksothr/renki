package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
)

// TaskRequest asks the plugin to do something (7.1). Args is kept raw until
// DecodeArgs, because its shape depends on Task.
type TaskRequest struct {
	Envelope
	Task     string          `json:"task"`
	Args     json.RawMessage `json:"args,omitempty"`
	TimeoutS float64         `json:"timeoutS"`
}

func (*TaskRequest) MessageType() string { return TypeTask }

// Task names (7.2): M1's, and what M2 has added so far.
const (
	TaskStop     = "stop"
	TaskMoveTo   = "move_to"
	TaskFollow   = "follow"
	TaskLookAt   = "look_at"
	TaskSay      = "say"
	TaskEmote    = "emote"
	TaskPickup   = "pickup"
	TaskDrop     = "drop"
	TaskInteract = "interact"
	TaskFlee     = "flee"
	TaskEquip    = "equip"
	TaskUnequip  = "unequip"
	TaskDeposit  = "deposit"
	TaskWithdraw = "withdraw"
	TaskHarvest  = "harvest"
	TaskAttack   = "attack"
	TaskUseItem  = "use_item"
)

// Slots are the equipment slots `unequip` empties; "hands" is both.
var Slots = []string{"right", "left", "hands", "head", "chest", "legs", "shoulder", "utility", "trinket", "ammo"}

// Instant reports whether a task runs alongside the body task instead of
// replacing it (7.1).
func Instant(task string) bool {
	switch task {
	case TaskLookAt, TaskSay, TaskEmote, TaskDrop, TaskEquip, TaskUnequip, TaskUseItem:
		return true
	}
	return false
}

// ErrUnknownTask is returned by DecodeArgs for a task this controller does
// not know. The plugin answers it with ReasonUnsupported.
var ErrUnknownTask = errors.New("unknown task")

// ErrBadArgs is returned by DecodeArgs for missing or malformed arguments.
// The plugin answers it with ReasonBadArgs.
var ErrBadArgs = errors.New("bad task arguments")

// NewTask builds a request for task with args, which may be nil.
func NewTask(id, task string, args any, timeoutS float64) (*TaskRequest, error) {
	req := &TaskRequest{Envelope: Envelope{ID: id}, Task: task, TimeoutS: timeoutS}
	if args != nil {
		raw, err := json.Marshal(args)
		if err != nil {
			return nil, fmt.Errorf("encode %s args: %w", task, err)
		}
		req.Args = raw
	}
	return req, nil
}

// DecodeArgs returns the typed arguments for a task: a *StopArgs,
// *MoveToArgs, *FollowArgs, *LookAtArgs, *SayArgs, *EmoteArgs,
// *PickupArgs, *DropArgs, *InteractArgs, *FleeArgs, *EquipArgs or
// *UnequipArgs. It checks
// that the required ones are there and that enumerated values are known.
func (r *TaskRequest) DecodeArgs() (any, error) {
	var args interface{ validate() error }
	switch r.Task {
	case TaskStop:
		args = &StopArgs{}
	case TaskMoveTo:
		args = &MoveToArgs{}
	case TaskFollow:
		args = &FollowArgs{}
	case TaskLookAt:
		args = &LookAtArgs{}
	case TaskSay:
		args = &SayArgs{}
	case TaskEmote:
		args = &EmoteArgs{}
	case TaskPickup:
		args = &PickupArgs{}
	case TaskDrop:
		args = &DropArgs{}
	case TaskInteract:
		args = &InteractArgs{}
	case TaskFlee:
		args = &FleeArgs{}
	case TaskEquip:
		args = &EquipArgs{}
	case TaskUseItem:
		args = &UseItemArgs{}
	case TaskUnequip:
		args = &UnequipArgs{}
	case TaskDeposit, TaskWithdraw:
		args = &TransferArgs{}
	case TaskHarvest:
		args = &HarvestArgs{}
	case TaskAttack:
		args = &AttackArgs{}
	default:
		return nil, fmt.Errorf("%w %q", ErrUnknownTask, r.Task)
	}
	if len(r.Args) > 0 && !bytes.Equal(r.Args, []byte("null")) {
		if err := json.Unmarshal(r.Args, args); err != nil {
			return nil, fmt.Errorf("%w for %s: %v", ErrBadArgs, r.Task, err)
		}
	}
	if err := args.validate(); err != nil {
		return nil, fmt.Errorf("%w for %s: %v", ErrBadArgs, r.Task, err)
	}
	return args, nil
}

// ItemCount is a positive count, or AllItems, encoded as "all".
type ItemCount int

const AllItems ItemCount = -1

func (n ItemCount) MarshalJSON() ([]byte, error) {
	if n == AllItems {
		return []byte(`"all"`), nil
	}
	if n <= 0 {
		return nil, errors.New("count must be positive or all")
	}
	return json.Marshal(int(n))
}

func (n *ItemCount) UnmarshalJSON(raw []byte) error {
	if bytes.Equal(raw, []byte(`"all"`)) {
		*n = AllItems
		return nil
	}
	var v int32
	if err := json.Unmarshal(raw, &v); err != nil || v <= 0 {
		return errors.New("count must be a positive integer or all")
	}
	*n = ItemCount(v)
	return nil
}

type TransferItem struct {
	Prefab string    `json:"prefab"`
	Count  ItemCount `json:"count"`
}

// TransferArgs names exactly what may move between the pack and one chest.
type TransferArgs struct {
	Container string         `json:"container"`
	Items     []TransferItem `json:"items"`
}

func (a *TransferArgs) validate() error {
	if a.Container == "" || len(a.Items) == 0 || len(a.Items) > 256 {
		return errors.New("container and 1–256 items are required")
	}
	seen := map[string]bool{}
	for _, it := range a.Items {
		if it.Prefab == "" || it.Count == 0 || it.Count < AllItems || seen[it.Prefab] {
			return errors.New("items need unique prefabs and positive counts or all")
		}
		seen[it.Prefab] = true
	}
	return nil
}

// StopArgs: `stop` takes no arguments.
type StopArgs struct{}

func (*StopArgs) validate() error { return nil }

// AttackArgs starts the bounded melee encounter; auto currently selects melee.
type AttackArgs struct {
	Entity       string   `json:"entity,omitempty"`
	Style        string   `json:"style,omitempty"`
	Kite         bool     `json:"kite,omitempty"`
	At           *Vec3    `json:"at,omitempty"`
	SearchRadius *float64 `json:"searchRadius,omitempty"`
	SearchSector *int     `json:"searchSector,omitempty"`
	SearchOffset *float64 `json:"searchOffset,omitempty"`
	SearchCenter *Vec3    `json:"searchCenter,omitempty"`
	SearchAnchor *Vec3    `json:"searchAnchor,omitempty"`
	SearchLimit  *float64 `json:"searchLimit,omitempty"`
}

func (a *AttackArgs) UnmarshalJSON(raw []byte) error {
	var wire struct {
		Entity       string          `json:"entity"`
		Style        string          `json:"style"`
		Kite         bool            `json:"kite"`
		At           json.RawMessage `json:"at"`
		SearchRadius *float64        `json:"searchRadius"`
		SearchSector json.RawMessage `json:"searchSector"`
		SearchOffset *float64        `json:"searchOffset"`
		SearchCenter json.RawMessage `json:"searchCenter"`
		SearchAnchor json.RawMessage `json:"searchAnchor"`
		SearchLimit  json.RawMessage `json:"searchLimit"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	*a = AttackArgs{Entity: wire.Entity, Style: wire.Style, Kite: wire.Kite, SearchRadius: wire.SearchRadius, SearchOffset: wire.SearchOffset}
	if wire.SearchSector != nil {
		var sector int
		if bytes.Equal(bytes.TrimSpace(wire.SearchSector), []byte("null")) {
			return errors.New("`searchSector` requires an integer from 0 to 7")
		}
		if err := json.Unmarshal(wire.SearchSector, &sector); err != nil {
			return err
		}
		a.SearchSector = &sector
	}
	if wire.At != nil {
		var values []*float64
		if err := json.Unmarshal(wire.At, &values); err != nil {
			return err
		}
		if len(values) != 3 || values[0] == nil || values[1] == nil || values[2] == nil {
			return errors.New("`at` requires an [x,y,z] position")
		}
		a.At = &Vec3{*values[0], *values[1], *values[2]}
	}
	if wire.SearchCenter != nil {
		var values []*float64
		if err := json.Unmarshal(wire.SearchCenter, &values); err != nil {
			return err
		}
		if len(values) != 3 || values[0] == nil || values[1] == nil || values[2] == nil {
			return errors.New("`searchCenter` requires a finite [x,y,z] position")
		}
		a.SearchCenter = &Vec3{*values[0], *values[1], *values[2]}
	}
	if wire.SearchAnchor != nil {
		var values []*float64
		if err := json.Unmarshal(wire.SearchAnchor, &values); err != nil {
			return err
		}
		if len(values) != 3 || values[0] == nil || values[1] == nil || values[2] == nil {
			return errors.New("`searchAnchor` requires a finite [x,y,z] position")
		}
		a.SearchAnchor = &Vec3{*values[0], *values[1], *values[2]}
	}
	if wire.SearchLimit != nil {
		var n float64
		if bytes.Equal(bytes.TrimSpace(wire.SearchLimit), []byte("null")) {
			return errors.New("searchLimit requires a positive number")
		}
		if err := json.Unmarshal(wire.SearchLimit, &n); err != nil {
			return err
		}
		a.SearchLimit = &n
	}
	return nil
}

func (a *AttackArgs) validate() error {
	if a.SearchAnchor != nil || a.SearchLimit != nil {
		if a.SearchCenter == nil || a.SearchAnchor == nil || a.SearchLimit == nil || math.IsNaN(*a.SearchLimit) || math.IsInf(*a.SearchLimit, 0) || *a.SearchLimit <= 0 || *a.SearchLimit > 500 {
			return errors.New("regional bounds require searchCenter, finite searchAnchor and searchLimit in (0,500]")
		}
		for _, n := range *a.SearchAnchor {
			if math.IsNaN(n) || math.IsInf(n, 0) {
				return errors.New("searchAnchor requires finite coordinates")
			}
		}
		radius := 24.0
		if a.SearchRadius != nil {
			radius = *a.SearchRadius
		}
		delta := Vec3{(*a.SearchCenter)[0] - (*a.SearchAnchor)[0], (*a.SearchCenter)[1] - (*a.SearchAnchor)[1], (*a.SearchCenter)[2] - (*a.SearchAnchor)[2]}
		if math.Sqrt(delta[0]*delta[0]+delta[1]*delta[1]+delta[2]*delta[2])+radius > *a.SearchLimit+1e-6 {
			return errors.New("search area exceeds native search bounds")
		}
	}
	if a.SearchCenter != nil {
		if !a.Kite || strings.TrimSpace(a.Entity) != "" || a.At != nil || a.SearchSector != nil || a.SearchOffset != nil {
			return errors.New("`searchCenter` requires untargeted kite discovery without sector/offset")
		}
		for _, n := range *a.SearchCenter {
			if math.IsNaN(n) || math.IsInf(n, 0) {
				return errors.New("`searchCenter` requires finite coordinates")
			}
		}
		if a.SearchRadius != nil && *a.SearchRadius > 24 {
			return errors.New("area discovery limits local `searchRadius` to 24 m")
		}
	}
	if a.SearchOffset != nil {
		radius := 60.0
		if a.SearchRadius != nil {
			radius = *a.SearchRadius
		}
		if a.SearchSector == nil || math.IsNaN(*a.SearchOffset) || math.IsInf(*a.SearchOffset, 0) || *a.SearchOffset < 0 || *a.SearchOffset > 60 || radius+*a.SearchOffset > 120 {
			return errors.New("`searchOffset` requires a search sector, 0–60 m offset and at most 120 m total reach")
		}
	}
	if a.SearchSector != nil && (!a.Kite || strings.TrimSpace(a.Entity) != "" || a.At != nil || *a.SearchSector < 0 || *a.SearchSector > 7) {
		return errors.New("`searchSector` requires untargeted kite discovery and an integer from 0 to 7")
	}
	if a.SearchRadius != nil && (!a.Kite || math.IsNaN(*a.SearchRadius) || math.IsInf(*a.SearchRadius, 0) || *a.SearchRadius < 12 || *a.SearchRadius > 80) {
		return errors.New("`searchRadius` requires kite=true and a radius between 12 and 80 m")
	}
	if a.At != nil && a.SearchRadius != nil && *a.SearchRadius > 12 {
		return errors.New("a fixed fighting area limits `searchRadius` to 12 m")
	}
	if strings.TrimSpace(a.Entity) == "" && !a.Kite {
		return errors.New("`entity` is required")
	}
	if a.At != nil && !a.Kite {
		return errors.New("`at` requires kite=true")
	}
	if a.Style != "" && a.Style != "melee" && a.Style != "auto" {
		return errors.New("this build supports melee or auto attack style")
	}
	return nil
}

// HarvestArgs operates on one resource; descendants remain separate targets.
// Nil Collect means true, matching the plugin default.
type HarvestArgs struct {
	Entity  string `json:"entity"`
	Collect *bool  `json:"collect,omitempty"`
}

func (a *HarvestArgs) validate() error {
	if a.Entity == "" {
		return errors.New("`entity` is required")
	}
	return nil
}

// MoveToArgs are the arguments to `move_to`. A nil pointer means the
// plugin's default: 1.5 m, and walking.
type MoveToArgs struct {
	Dry          bool     `json:"dry,omitempty"`
	To           Target   `json:"to"`
	StopDistance *float64 `json:"stopDistance,omitempty"`
	Run          *bool    `json:"run,omitempty"`
}

func (a *MoveToArgs) validate() error {
	if a.To.IsZero() {
		return errors.New("`to` is required")
	}
	return nil
}

// FollowArgs are the arguments to `follow`. A nil Distance means 3.5 m.
type FollowArgs struct {
	Player   string   `json:"player"`
	Distance *float64 `json:"distance,omitempty"`
	Run      RunMode  `json:"run,omitempty"`
}

func (a *FollowArgs) validate() error {
	if a.Player == "" {
		return errors.New("`player` is required")
	}
	return nil
}

// LookAtArgs are the arguments to `look_at`.
type LookAtArgs struct {
	To Target `json:"to"`
}

func (a *LookAtArgs) validate() error {
	if a.To.IsZero() {
		return errors.New("`to` is required")
	}
	return nil
}

// SayArgs are the arguments to `say`. An empty Channel means normal, and To
// names the player a whisper goes to.
type SayArgs struct {
	Text    string `json:"text"`
	Channel string `json:"channel,omitempty"`
	To      string `json:"to,omitempty"`
}

func (a *SayArgs) validate() error {
	if a.Text == "" {
		return errors.New("`text` is required")
	}
	switch a.Channel {
	case "", ChannelNormal, ChannelShout:
	case ChannelWhisper:
		if a.To == "" {
			return errors.New("a whisper needs `to`")
		}
	default:
		return fmt.Errorf("unknown channel %q", a.Channel)
	}
	return nil
}

// EmoteArgs are the arguments to `emote`.
type EmoteArgs struct {
	Name string `json:"name"`
}

func (a *EmoteArgs) validate() error {
	if a.Name == "" {
		return errors.New("`name` is required")
	}
	return nil
}

// PickupArgs are the arguments to `pickup`: the drop, by entity id.
type PickupArgs struct {
	Entity string `json:"entity"`
}

func (a *PickupArgs) validate() error {
	if a.Entity == "" {
		return errors.New("`entity` is required")
	}
	return nil
}

// DropArgs are the arguments to `drop`: how many of a prefab to put down
// at the bot's feet.
type DropArgs struct {
	Prefab string `json:"prefab"`
	Count  int    `json:"count"`
}

func (a *DropArgs) validate() error {
	if a.Prefab == "" {
		return errors.New("`prefab` is required")
	}
	if a.Count < 1 {
		return errors.New("`count` must be at least 1")
	}
	return nil
}

// InteractArgs are the arguments to `interact`: the entity to use, with
// the alternative use (Alt) or the use key held (Hold).
type InteractArgs struct {
	Entity string `json:"entity"`
	Alt    bool   `json:"alt,omitempty"`
	Hold   bool   `json:"hold,omitempty"`
	Sleep  bool   `json:"sleep,omitempty"`
	Repair bool   `json:"repair,omitempty"`
}

func (a *InteractArgs) validate() error {
	if a.Entity == "" {
		return errors.New("`entity` is required")
	}
	if a.Sleep && (a.Alt || a.Hold) {
		return errors.New("`sleep` cannot be combined with `alt` or `hold`")
	}
	if a.Repair && (a.Sleep || a.Alt || a.Hold) {
		return errors.New("`repair` cannot combine with sleep, alt or hold")
	}
	return nil
}

// FleeArgs are the arguments to `flee`: what to get away from, how far (nil
// means the plugin's 20 m), and where safety lies, if anywhere: the bot runs
// there unless it lies past the threat.
type FleeArgs struct {
	From     Target   `json:"from"`
	Distance *float64 `json:"distance,omitempty"`
	Toward   *Target  `json:"toward,omitempty"`
}

func (a *FleeArgs) validate() error {
	if a.From.IsZero() {
		return errors.New("`from` is required")
	}
	if a.Distance != nil && *a.Distance <= 0 {
		return errors.New("`distance` must be more than 0")
	}
	if a.Toward != nil && a.Toward.IsZero() {
		return errors.New("`toward` must be a position or an entity id")
	}
	return nil
}

// EquipArgs are the arguments to `equip`: the item, by prefab.
type EquipArgs struct {
	Item string `json:"item"`
}

func (a *EquipArgs) validate() error {
	if a.Item == "" {
		return errors.New("`item` is required")
	}
	return nil
}

// UseItemArgs eats one carried food by prefab through normal game consumption.
type UseItemArgs struct {
	Item string `json:"item"`
}

func (a *UseItemArgs) validate() error {
	if a.Item == "" {
		return errors.New("`item` is required")
	}
	return nil
}

// UnequipArgs are the arguments to `unequip`: one of Slots.
type UnequipArgs struct {
	Slot string `json:"slot"`
}

func (a *UnequipArgs) validate() error {
	if !slices.Contains(Slots, a.Slot) {
		return fmt.Errorf("unknown slot %q; want one of %s", a.Slot, strings.Join(Slots, ", "))
	}
	return nil
}

// Target is a place to go or look at: either a world position or an entity
// id, which on the wire is an `[x,y,z]` array or a string.
type Target struct {
	Pos    *Vec3
	Entity string
}

// IsZero reports whether the target was never set.
func (t Target) IsZero() bool { return t.Pos == nil && t.Entity == "" }

func (t Target) MarshalJSON() ([]byte, error) {
	switch {
	case t.Pos != nil && t.Entity != "":
		return nil, errors.New("target has both a position and an entity")
	case t.Pos != nil:
		return json.Marshal(t.Pos)
	case t.Entity != "":
		return json.Marshal(t.Entity)
	}
	return []byte("null"), nil
}

func (t *Target) UnmarshalJSON(data []byte) error {
	*t = Target{}
	switch {
	case bytes.Equal(data, []byte("null")):
		return nil
	case len(data) > 0 && data[0] == '"':
		return json.Unmarshal(data, &t.Entity)
	case len(data) > 0 && data[0] == '[':
		var p Vec3
		if err := json.Unmarshal(data, &p); err != nil {
			return err
		}
		t.Pos = &p
		return nil
	}
	return fmt.Errorf("target must be an [x,y,z] position or an entity id, got %s", data)
}

// RunMode is follow's `run` argument: false walks, true runs, and "auto"
// matches the leader. Leaving it out means auto.
type RunMode int

const (
	RunUnset RunMode = iota
	RunWalk
	RunRun
	RunAuto
)

func (m RunMode) MarshalJSON() ([]byte, error) {
	switch m {
	case RunWalk:
		return []byte("false"), nil
	case RunRun:
		return []byte("true"), nil
	case RunAuto:
		return []byte(`"auto"`), nil
	case RunUnset:
		return []byte("null"), nil
	}
	return nil, fmt.Errorf("unknown RunMode %d", int(m))
}

func (m *RunMode) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case "false":
		*m = RunWalk
	case "true":
		*m = RunRun
	case `"auto"`:
		*m = RunAuto
	case "null":
		*m = RunUnset
	default:
		return fmt.Errorf(`run must be true, false or "auto", got %s`, data)
	}
	return nil
}
