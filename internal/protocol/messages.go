package protocol

import "slices"

// Message types, as they appear in `type`.
const (
	TypeHello      = "hello"
	TypeWelcome    = "welcome"
	TypeState      = "state"
	TypePerception = "perception"
	TypeInventory  = "inventory"
	TypeContainer  = "container"
	TypeLayout     = "layout"
	TypeChat       = "chat"
	TypeEvent      = "event"
	TypeTask       = "task"
	TypePing       = "ping"
	TypePong       = "pong"
	TypeConfig     = "config"
	TypeLogout     = "logout"
	TypeBye        = "bye"
	TypeError      = "error"
	TypeTrace      = "trace"
)

// Phase values the plugin reports (section 5).
const (
	PhaseMenu       = "menu"
	PhaseConnecting = "connecting"
	PhaseInWorld    = "in_world"
	PhaseDead       = "dead"
	PhaseLoggingOut = "logging_out"
)

// CodeBadMessage is the `error` code for a frame the receiver cannot use
// (section 6.8).
const CodeBadMessage = "bad_message"

// ---- Session (section 5) ----

// Hello opens a session: plugin to controller.
type Hello struct {
	Envelope
	Token        string   `json:"token"`
	Plugin       string   `json:"plugin"`
	Game         string   `json:"game"`
	Phase        string   `json:"phase"`
	Character    string   `json:"character"`
	World        string   `json:"world"`
	Capabilities []string `json:"capabilities"`
}

func (*Hello) MessageType() string { return TypeHello }

// Welcome answers Hello: controller to plugin. AllowedTasks can only narrow
// what the plugin's own safety config allows.
type Welcome struct {
	Envelope
	AgentName    string           `json:"agentName"`
	Owners       []string         `json:"owners"`
	Rates        Rates            `json:"rates"`
	Perception   PerceptionConfig `json:"perception"`
	AllowedTasks []string         `json:"allowedTasks"`
}

func (*Welcome) MessageType() string { return TypeWelcome }

// Rates are the stream frequencies in Hz.
type Rates struct {
	StateHz      float64 `json:"stateHz"`
	PerceptionHz float64 `json:"perceptionHz"`
}

// PerceptionConfig bounds the perception snapshot.
type PerceptionConfig struct {
	Radius     float64 `json:"radius"`
	MaxPerKind int     `json:"maxPerKind"`
}

// Logout asks the plugin to save and return to the main menu. It carries an
// `id`, so the plugin can answer it.
type Logout struct{ Envelope }

func (*Logout) MessageType() string { return TypeLogout }

// Bye is sent by either side before it closes the socket.
type Bye struct {
	Envelope
	Reason string `json:"reason"`
}

func (*Bye) MessageType() string { return TypeBye }

// ---- Housekeeping (sections 6.8 and 7.4) ----

// Ping is sent by the controller; the plugin answers with Pong.
type Ping struct{ Envelope }

func (*Ping) MessageType() string { return TypePing }

// Pong answers Ping, with `re` set to the ping's `id`.
type Pong struct{ Envelope }

func (*Pong) MessageType() string { return TypePong }

// Error reports a message the sender could not use. `re` names it when it
// had an `id`.
type Error struct {
	Envelope
	Code   string `json:"code"`
	Detail string `json:"detail,omitempty"`
}

func (*Error) MessageType() string { return TypeError }

// Config changes rates and perception mid-session. A nil field is left as
// it was.
type Config struct {
	Envelope
	Rates      *Rates            `json:"rates,omitempty"`
	Perception *PerceptionConfig `json:"perception,omitempty"`
}

func (*Config) MessageType() string { return TypeConfig }

// ---- Plugin to controller (section 6) ----

// State is the bot's own body, streamed at stateHz (6.1).
type State struct {
	Envelope
	Phase      string            `json:"phase"`
	Character  string            `json:"character"`
	World      string            `json:"world"`
	Pos        Vec3              `json:"pos"`
	Yaw        float64           `json:"yaw"`
	Vel        Vec3              `json:"vel"`
	HP         float64           `json:"hp"`
	HPMax      float64           `json:"hpMax"`
	Stamina    float64           `json:"stamina"`
	StaminaMax float64           `json:"staminaMax"`
	Eitr       float64           `json:"eitr"`
	EitrMax    float64           `json:"eitrMax"`
	Flags      []string          `json:"flags"`
	Biome      string            `json:"biome"`
	Location   *Location         `json:"location,omitempty"`
	Env        string            `json:"env"`
	Time       GameTime          `json:"time"`
	Equipped   map[string]string `json:"equipped"`
	Effects    []string          `json:"effects"`
	Weight     float64           `json:"weight"`
	WeightMax  float64           `json:"weightMax"`
	Task       *TaskRef          `json:"task,omitempty"`
}

func (*State) MessageType() string { return TypeState }

// State flags (6.1).
const (
	FlagGrounded   = "grounded"
	FlagRunning    = "running"
	FlagCrouching  = "crouching"
	FlagSwimming   = "swimming"
	FlagSitting    = "sitting"
	FlagSleeping   = "sleeping"
	FlagAttacking  = "attacking"
	FlagBlocking   = "blocking"
	FlagEncumbered = "encumbered"
	FlagStranded   = "stranded"
	FlagInShelter  = "in_shelter"
	FlagNearFire   = "near_fire"
)

// Encumbered reports that the bot carries more than weightMax (6.1).
func (s *State) Encumbered() bool { return slices.Contains(s.Flags, FlagEncumbered) }

// Stranded reports that the game will not move the bot: encumbered with no
// stamina left, which does not come back until it carries less. The plugin
// decides it, as it decides when a body task fails for it (6.1).
func (s *State) Stranded() bool { return slices.Contains(s.Flags, FlagStranded) }

// GameTime is the in-game calendar.
type GameTime struct {
	Day         int     `json:"day"`
	DayFraction float64 `json:"dayFraction"`
	Night       bool    `json:"night"`
}

// TaskRef is the body task the plugin is running, as reported in State.
type TaskRef struct {
	ID     string `json:"id"`
	Task   string `json:"task"`
	Status string `json:"status"`
}

// Perception is a snapshot of what is near the bot, nearest first (6.2).
// It is streamed from M2.
type Perception struct {
	Envelope
	Characters []PerceivedCharacter `json:"characters"`
	Resources  []PerceivedResource  `json:"resources"`
	Structures []PerceivedStructure `json:"structures"`
}

func (*Perception) MessageType() string { return TypePerception }

// PerceivedCharacter is a player, creature or tamed creature in range.
type PerceivedCharacter struct {
	ID          string  `json:"id"`
	Kind        string  `json:"kind"`
	Name        string  `json:"name,omitempty"`
	Prefab      string  `json:"prefab,omitempty"`
	Pos         Vec3    `json:"pos"`
	HPFrac      float64 `json:"hpFrac"`
	Stars       int     `json:"stars,omitempty"`
	Hostile     bool    `json:"hostile,omitempty"`
	TargetingMe bool    `json:"targetingMe,omitempty"`
}

// Perceived kinds (6.2).
const (
	CharacterPlayer   = "player"
	CharacterCreature = "creature"
	CharacterTamed    = "tamed"

	ResourceTree     = "tree"
	ResourceLog      = "log"
	ResourceStump    = "stump"
	ResourceRock     = "rock"
	ResourcePickable = "pickable"
	ResourceDrop     = "drop"

	StructureContainer = "container"
	StructureStation   = "station"
	StructureBed       = "bed"
	StructurePortal    = "portal"
	StructureDoor      = "door"
	StructureFire      = "fire"
	StructureSmelter   = "smelter"
	StructureTombstone = "tombstone"
	StructureOther     = "other"
)

// PerceivedResource is something the bot could harvest or pick up. MinTool
// is a pointer because tier 0 is a real value and absent means "no tool".
type PerceivedResource struct {
	ID      string         `json:"id"`
	Kind    string         `json:"kind"`
	Prefab  string         `json:"prefab"`
	Pos     Vec3           `json:"pos"`
	MinTool *int           `json:"minTool,omitempty"`
	Stack   int            `json:"stack,omitempty"`
	Yield   *ResourceYield `json:"yield,omitempty"`
}

// ResourceYield describes the primary item from a drop or simple pickable.
// Count is an upper bound including possible skill bonuses, not a receipt.
// Missing yields are ineligible for automatic collection.
type ResourceYield struct {
	Prefab   string  `json:"prefab"`
	Count    int     `json:"count"`
	StackMax int     `json:"stackMax"`
	Weight   float64 `json:"weight"`
}

// PerceivedStructure is a built piece the bot might use.
type PerceivedStructure struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Prefab string `json:"prefab,omitempty"`
	Tag    string `json:"tag,omitempty"`
	Pos    Vec3   `json:"pos"`
	Mine   bool   `json:"mine,omitempty"`
	// Open is set for doors only.
	Open *bool `json:"open,omitempty"`
}

// Inventory is sent on change and once after Welcome (6.3). It is streamed
// from M2.
type Inventory struct {
	Envelope
	Size  [2]int          `json:"size"`
	Items []InventoryItem `json:"items"`
}

func (*Inventory) MessageType() string { return TypeInventory }

// Container is an observed chest, with exact transfer results when Task is set.
// The snapshot is sent after normal access is granted and again after a transfer.
type Container struct {
	Envelope
	Entity    string          `json:"entity"`
	Pos       Vec3            `json:"pos"`
	Size      [2]int          `json:"size"`
	Items     []InventoryItem `json:"items"`
	Task      string          `json:"task,omitempty"`
	Direction string          `json:"direction,omitempty"`
	Moved     map[string]int  `json:"moved,omitempty"`
	Remaining map[string]int  `json:"remaining,omitempty"`
}

func (*Container) MessageType() string { return TypeContainer }

func (c *Container) Inventory() *Inventory { return &Inventory{Size: c.Size, Items: c.Items} }

// Layout is the walls near the bot, sent when they change (6.9).
type Layout struct {
	Envelope
	Walls []Wall `json:"walls"`
}

func (*Layout) MessageType() string { return TypeLayout }

// Wall is the line along a wall's foot, from A to B, and its height.
type Wall struct {
	A Vec3    `json:"a"`
	B Vec3    `json:"b"`
	H float64 `json:"h"`
}

// InventoryItem is one occupied slot.
type InventoryItem struct {
	Slot          [2]int  `json:"slot"`
	Prefab        string  `json:"prefab"`
	Stack         int     `json:"stack"`
	StackMax      int     `json:"stackMax,omitempty"`
	Quality       int     `json:"quality,omitempty"`
	Durability    float64 `json:"durability,omitempty"`
	DurabilityMax float64 `json:"durabilityMax,omitempty"`
	Equipped      bool    `json:"equipped,omitempty"`
	// Weight is one of the stack's, Type the game's item type (Material,
	// Consumable, OneHandedWeapon...), and Value what a trader gives, set
	// only on valuables.
	Weight float64 `json:"weight"`
	Type   string  `json:"type"`
	Value  int     `json:"value,omitempty"`
}

// Chat is every chat line except the bot's own (6.4).
type Chat struct {
	Envelope
	From    ChatSender `json:"from"`
	Channel string     `json:"channel"`
	Text    string     `json:"text"`
	Pos     Vec3       `json:"pos"`
	Dist    float64    `json:"dist"`
}

func (*Chat) MessageType() string { return TypeChat }

// ChatSender is who spoke. Owner is the plugin's view of welcome.owners.
type ChatSender struct {
	Name     string `json:"name"`
	PlayerID string `json:"playerId"`
	Owner    bool   `json:"owner"`
}

// Chat channels (6.4). Say accepts the first three.
const (
	ChannelNormal  = "normal"
	ChannelShout   = "shout"
	ChannelWhisper = "whisper"
	ChannelPing    = "ping"
)

// Event is something that happened to or around the bot (6.5). Which of the
// optional fields are set depends on Event.
type Event struct {
	Envelope
	Event      string  `json:"event"`
	By         string  `json:"by,omitempty"`
	Amount     float64 `json:"amount,omitempty"`
	DamageType string  `json:"damageType,omitempty"`
	Pos        *Vec3   `json:"pos,omitempty"`
	Prefab     string  `json:"prefab,omitempty"`
	Count      int     `json:"count,omitempty"`
	Skill      string  `json:"skill,omitempty"`
	Level      int     `json:"level,omitempty"`
	Name       string  `json:"name,omitempty"`
	PlayerID   string  `json:"playerId,omitempty"`
	Biome      string  `json:"biome,omitempty"`
	HPFrac     float64 `json:"hpFrac,omitempty"`
}

func (*Event) MessageType() string { return TypeEvent }

// Trace is a decision the plugin made on its own, such as a change of
// gait or a stuck recovery: plugin to controller (6.7). The controller
// writes it to the audit log and acts on none of it.
type Trace struct {
	Envelope
	// Task is the running task's id, when there is one.
	Task string `json:"task,omitempty"`
	// Source names what decided: "follow", "walker", "runner".
	Source string `json:"source"`
	Msg    string `json:"msg"`
	// Data holds the numbers behind the decision.
	Data map[string]any `json:"data,omitempty"`
}

func (*Trace) MessageType() string { return TypeTrace }

// Event names (6.5).
const (
	EventDamaged      = "damaged"
	EventDied         = "died"
	EventRespawned    = "respawned"
	EventItemAdded    = "item_added"
	EventSkillRaised  = "skill_raised"
	EventPlayerJoined = "player_joined"
	EventPlayerLeft   = "player_left"
	EventEnteredBiome = "entered_biome"
	EventLowHealth    = "low_health"
)

// TaskStatus reports on a task the controller sent, with `re` set to its id
// (6.6).
type TaskStatus struct {
	Envelope
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	Detail string `json:"detail,omitempty"`
}

func (*TaskStatus) MessageType() string { return TypeTask }

// Task statuses (6.6). Rejected is the only reply to a task that never
// starts; the others run accepted, running, then one of the last three.
const (
	StatusAccepted  = "accepted"
	StatusRejected  = "rejected"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// Final reports whether status ends a task's life.
func Final(status string) bool {
	switch status {
	case StatusRejected, StatusSucceeded, StatusFailed, StatusCancelled:
		return true
	}
	return false
}

// Failure reasons (6.6).
const (
	ReasonUnsupported      = "unsupported"
	ReasonNotAllowed       = "not_allowed"
	ReasonBadArgs          = "bad_args"
	ReasonInvalidTarget    = "invalid_target"
	ReasonTargetLost       = "target_lost"
	ReasonUnreachable      = "unreachable"
	ReasonDeepWater        = "deep_water"
	ReasonStuck            = "stuck"
	ReasonTimeout          = "timeout"
	ReasonNoTool           = "no_tool"
	ReasonInventoryFull    = "inventory_full"
	ReasonTooHeavy         = "too_heavy"
	ReasonEncumbered       = "encumbered"
	ReasonMissingMaterials = "missing_materials"
	ReasonSuperseded       = "superseded"
	ReasonStopped          = "stopped"
	ReasonDied             = "died"
	ReasonRefused          = "refused"
)
