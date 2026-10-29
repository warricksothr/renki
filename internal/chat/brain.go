// Package chat decides what the bot does with game chat
// (docs/BRIDGE-PROTOCOL.md section 9). Addressed lines that match the
// command grammar become tasks with no LLM involved; the rest go to a Voice,
// whose one proposed task is checked here like any command.
package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// Dispatcher is what the brain needs from the session.
type Dispatcher interface {
	Dispatch(ctx context.Context, o session.Order) (*session.Task, error)
	State() *protocol.State
	Perception() *protocol.Perception
	Inventory() *protocol.Inventory
	AllowedTasks() []string
}

// Voice turns a line the grammar does not know into words, and at most one
// proposed task.
type Voice interface {
	Reply(ctx context.Context, req Request) (Reply, error)
}

// Request is everything the Voice sees: a summary, never the raw streams.
type Request struct {
	AgentName string
	Nicknames []string
	Speaker   string
	Owner     bool
	Text      string
	Summary   string
	Recent    []Line
	// Tasks are the tasks the bot may be asked to do. Empty for a
	// non-owner, who gets conversation but no tasks.
	Tasks []string
}

// Reply is what the Voice says, and the task it proposes, if any.
type Reply struct {
	Say  string
	Task *Proposal
}

// Proposal is a task the Voice suggests. It is checked like a command
// before it goes anywhere.
type Proposal struct {
	Task string
	Args json.RawMessage
}

// Line is one line of chat history.
type Line struct {
	From string
	Text string
	Bot  bool
}

// Home is the bot's home, kept by internal/home.
type Home interface {
	// SetHome makes pos home in the current world and returns the world.
	SetHome(pos protocol.Vec3, by string) (world string, err error)
	// GoHome sends the bot home; why is for the log.
	GoHome(ctx context.Context, origin any, why string) (*session.Task, error)
	// Homeward reports whether a task was sent to take the bot home.
	Homeward(id string) bool
	// ReturnOn reports whether a body task failing for reason sends the bot home.
	ReturnOn(reason string) bool
	// HomeDistance is how far the bot is from home.
	HomeDistance() (float64, bool)
	// LeftBehind is what the bot had to leave on the ground in this world.
	LeftBehind() []home.Cache
	// Touch counts as activity, for the idle rule.
	Touch()
	// ResumeAfterDeath leaves what to start again after a respawn, when
	// the rules say resume; nil clears it.
	ResumeAfterDeath(func(context.Context) bool)
	// Ordered drops whatever a death left pending, the resume and the
	// trip after the respawn: a newer order wins.
	Ordered()
}

// Options configure a Brain.
type Options struct {
	// AgentName is the name the bot answers to. Empty means the in-game
	// character's name, from the latest state.
	AgentName string
	// Nicknames are other names the bot answers to, beside its own.
	Nicknames []string
	Owners    []string
	// Voice answers lines the grammar does not match. Nil means the bot
	// only knows the grammar.
	Voice Voice
	// Live, when set, gives the owners, the nicknames and whether the
	// voice is on, read fresh each time, in place of the fields above.
	Live func() Live
	// Home keeps the bot's home. Nil means it has none.
	Home Home
	// ReplyWindow is how long after the bot speaks to a player that
	// player's lines count as addressed without the bot's name.
	ReplyWindow time.Duration
	// Cooldown is the minimum time between Voice calls for one player.
	Cooldown time.Duration
	// VoiceTimeout bounds one Voice call.
	VoiceTimeout time.Duration
	Logger       *slog.Logger
	Now          func() time.Time
}

// Live are the brain's settings that can change while it runs.
type Live struct {
	Owners    []string
	Nicknames []string
	// Voice off leaves the bot to the command grammar.
	Voice bool
}

func (b *Brain) live() Live {
	if b.opts.Live != nil {
		return b.opts.Live()
	}
	return Live{Owners: b.opts.Owners, Nicknames: b.opts.Nicknames, Voice: true}
}

// voice is the Voice, or nil while there is none or it is switched off.
func (b *Brain) voice() Voice {
	if b.opts.Voice == nil || !b.live().Voice {
		return nil
	}
	return b.opts.Voice
}

// Brain is the session's listener for chat, events and finished tasks.
type Brain struct {
	opts Options
	d    Dispatcher

	mu        sync.Mutex
	name      *regexp.Regexp
	nameFor   string
	spokeTo   map[string]time.Time
	lastVoice map[string]time.Time
	recent    []Line
}

// historyLen is how many lines of chat the Voice sees.
const historyLen = 20

// New returns a brain that sends tasks through d.
func New(opts Options, d Dispatcher) *Brain {
	if opts.ReplyWindow == 0 {
		opts.ReplyWindow = 20 * time.Second
	}
	if opts.Cooldown == 0 {
		opts.Cooldown = 5 * time.Second
	}
	if opts.VoiceTimeout == 0 {
		opts.VoiceTimeout = 30 * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Brain{
		opts:      opts,
		d:         d,
		spokeTo:   map[string]time.Time{},
		lastVoice: map[string]time.Time{},
	}
}

// agentName is the name the bot answers to: the configured one, or else the
// character's name from the latest state. Empty before the bot has a body.
func (b *Brain) agentName() string {
	if b.opts.AgentName != "" {
		return b.opts.AgentName
	}
	if st := b.d.State(); st != nil {
		return st.Character
	}
	return ""
}

// nothing never matches.
var nothing = regexp.MustCompile(`[^\s\S]`)

// nameRE matches the bot's name or a nickname as a word, or nothing while
// it has none. It is rebuilt only when the name or the nicknames change.
func (b *Brain) nameRE() *regexp.Regexp {
	name := b.agentName()
	nicknames := b.live().Nicknames
	key := name + "\x00" + strings.Join(nicknames, "\x00")
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.name == nil || b.nameFor != key {
		b.name, b.nameFor = nothing, key
		var alts []string
		for _, n := range append([]string{name}, nicknames...) {
			if n = strings.TrimSpace(n); n != "" {
				alts = append(alts, regexp.QuoteMeta(n))
			}
		}
		if len(alts) > 0 {
			b.name = regexp.MustCompile(`(?i)\b(?:` + strings.Join(alts, "|") + `)\b`)
		}
	}
	return b.name
}

// origin rides on a task so a failure can be explained to whoever asked.
type origin struct {
	player  string
	channel string
	// follow is the player a follow command was for, so a death can
	// resume it.
	follow string
}

// channelPanel is the channel of words typed into the control panel: always
// addressed to the bot, and answered on the panel, not in the game.
const channelPanel = "panel"

// FromPanel's errors.
var (
	ErrNotOwner = errors.New("not an owner")
	ErrNoBot    = errors.New("the bot is not connected")
)

// FromPanel takes text typed into the control panel as if speaker, an
// owner, had said it to the bot. Where the speaker stands comes from
// perception; when their character is not in sight, commands that need
// it ("come here", "set home", "pick that up") say so.
func (b *Brain) FromPanel(speaker, text string) error {
	if !slices.Contains(b.live().Owners, speaker) {
		return fmt.Errorf("%w: %s", ErrNotOwner, speaker)
	}
	// Dead or loading counts: an order given then is meant to cancel what a
	// death left pending.
	st := b.d.State()
	if st == nil {
		return ErrNoBot
	}
	c := &protocol.Chat{From: protocol.ChatSender{Name: speaker, Owner: true}, Channel: channelPanel, Text: text, Dist: -1}
	// A sighting counts only from a perception as new as the state, give
	// or take a few scans: none comes while the bot is dead or loading.
	if seen := b.d.Perception(); seen != nil && math.Abs(st.T-seen.T) <= panelSightFor {
		for _, p := range seen.Characters {
			if p.Kind == protocol.CharacterPlayer && p.Name == speaker {
				c.Pos, c.Dist = p.Pos, st.Pos.FlatDistance(p.Pos)
			}
		}
	}
	b.Chat(c)
	return nil
}

// panelSightFor is how old, in game seconds, a perception may be for the
// panel's owner seen in it to count as seen now.
const panelSightFor = 5.0

// located reports whether the bot knows where the speaker of c stands.
func located(c *protocol.Chat) bool { return c.Channel != channelPanel || c.Dist >= 0 }

// Chat implements session.Listener.
func (b *Brain) Chat(c *protocol.Chat) {
	b.remember(Line{From: c.From.Name, Text: c.Text})
	if c.Channel != channelPanel && !b.addressed(c) {
		b.opts.Logger.Debug("not addressed", "from", c.From.Name, "name", b.agentName())
		return
	}
	from := origin{player: c.From.Name, channel: replyChannel(c.Channel)}
	owners := b.live().Owners
	owner := slices.Contains(owners, c.From.Name)
	cmd, ok := parse(b.strip(c.Text))
	if !ok {
		b.opts.Logger.Debug("not a command", "from", c.From.Name, "voice", b.voice() != nil)
		b.converse(c, from, owner)
		return
	}
	if !owner {
		b.opts.Logger.Info("refused: not an owner", "from", c.From.Name, "command", cmd, "owners", owners)
		b.say(from, fmt.Sprintf("Sorry, %s, I only take orders from %s.", c.From.Name, joinNames(owners)))
		return
	}
	switch cmd {
	// Follow too: from the panel the owner may not be in the world at all,
	// and a follow that loses its leader sends the bot home. Refused
	// before it counts as an order, so it cancels nothing.
	case cmdCome, cmdSetHome, cmdPickup, cmdFollow:
		if !located(c) {
			b.say(from, fmt.Sprintf("I can't see you, %s, so I don't know where you mean.", c.From.Name))
			return
		}
	}
	b.opts.Logger.Info("command", "from", c.From.Name, "command", cmd)
	if b.opts.Home != nil {
		b.opts.Home.Touch()
		if cmd != cmdStatus && cmd != cmdSetHome {
			// A newer order wins over whatever a death left pending.
			b.opts.Home.Ordered()
		}
	}
	b.run(cmd, c, from)
}

// Event implements session.Listener.
func (b *Brain) Event(e *protocol.Event) {
	b.opts.Logger.Info("event", "event", e.Event, "by", e.By, "name", e.Name)
}

// TaskDone implements session.Listener: a task someone asked for in chat
// that failed is explained to them.
func (b *Brain) TaskDone(t *session.Task) {
	o, ok := t.Origin.(origin)
	if !ok {
		return
	}
	status, reason, detail := t.Status()
	if status == protocol.StatusSucceeded && t.Name == protocol.TaskPickup {
		// The plugin says "the other N" when some of the stack stayed behind:
		// "no room for the other N" when it filled up first, "the other N
		// is too heavy" when it would have overloaded the bot.
		switch {
		case strings.Contains(detail, "no room"):
			b.say(o, "I got some of it, but I'm out of room.")
		case strings.Contains(detail, "too heavy"):
			b.say(o, "I got some of it, but the rest is too heavy.")
		case strings.Contains(detail, "the other"):
			b.say(o, "I got some of it.")
		default:
			b.say(o, "Got it.")
		}
		return
	}
	if t.Name == protocol.TaskInteract {
		// The plugin passes on what the game said: "used the Bed: Spawn
		// point set", or, refused, the game's words alone.
		switch {
		case status == protocol.StatusSucceeded:
			if _, said, ok := strings.Cut(detail, ": "); ok {
				b.say(o, said)
			} else {
				b.say(o, "Done.")
			}
			return
		case reason == protocol.ReasonRefused && detail != "":
			b.say(o, "It won't: "+detail)
			return
		}
	}
	if status != protocol.StatusFailed && status != protocol.StatusRejected {
		return
	}
	msg := explain(t.Name, reason, o.player)
	if reason == protocol.ReasonDied && b.opts.Home != nil {
		b.leaveResume(t, o)
	}
	if b.returnHome(t, reason, o) {
		switch reason {
		case protocol.ReasonStuck:
			msg = "I'm stuck. I'll try to make it home."
		default:
			msg += " Heading home."
		}
	}
	if msg != "" {
		b.say(o, msg)
	}
}

// leaveResume tells the home keeper what a death cut short: a follow
// starts again for the same player, if they are still about (if not, the
// follow fails with target_lost and the bot goes home on that). Anything
// else leaves nothing to resume.
func (b *Brain) leaveResume(t *session.Task, o origin) {
	if t.Name != protocol.TaskFollow || o.follow == "" {
		b.opts.Home.ResumeAfterDeath(nil)
		return
	}
	b.opts.Home.ResumeAfterDeath(func(ctx context.Context) bool {
		order := session.Order{Task: protocol.TaskFollow, Origin: o,
			Args: &protocol.FollowArgs{Player: o.follow, Run: protocol.RunAuto}}
		if _, err := b.d.Dispatch(ctx, order); err != nil {
			b.opts.Logger.Warn("could not resume the follow", "player", o.follow, "err", err)
			return false
		}
		b.say(o, fmt.Sprintf("I'm back. Coming to find you, %s.", o.follow))
		return true
	})
}

// returnHome sends the bot home after a body task failed, when the home
// rules say so, and reports whether it did. A failed trip home does not
// start another.
func (b *Brain) returnHome(t *session.Task, reason string, o origin) bool {
	h := b.opts.Home
	if h == nil || (t.Name != protocol.TaskFollow && t.Name != protocol.TaskMoveTo) || h.Homeward(t.ID) ||
		!h.ReturnOn(reason) {
		return false
	}
	if _, err := h.GoHome(context.Background(), o, "after "+t.Name+" failed: "+reason); err != nil {
		if !errors.Is(err, home.ErrNoHome) && !errors.Is(err, home.ErrStranded) {
			b.opts.Logger.Warn("could not go home", "err", err)
		}
		return false
	}
	return true
}

func (b *Brain) addressed(c *protocol.Chat) bool {
	if c.Channel == protocol.ChannelWhisper || b.nameRE().MatchString(c.Text) {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	at, ok := b.spokeTo[c.From.Name]
	return ok && b.opts.Now().Sub(at) <= b.opts.ReplyWindow
}

// strip lower-cases text and drops punctuation, the bot's name and "please",
// leaving words separated by single spaces.
func (b *Brain) strip(text string) string {
	text = b.nameRE().ReplaceAllString(text, " ")
	text = strings.Map(func(r rune) rune {
		if r == '\'' {
			return -1
		}
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return ' '
	}, strings.ToLower(text))
	words := slices.DeleteFunc(strings.Fields(text), func(w string) bool { return w == "please" || w == "pls" })
	return strings.Join(words, " ")
}

type command int

const (
	cmdFollow command = iota + 1
	cmdStop
	cmdStatus
	cmdCome
	cmdWait
	cmdSetHome
	cmdGoHome
	cmdPickup
	cmdBed
	cmdDoor
)

func (c command) String() string {
	return [...]string{"", "follow", "stop", "status", "come here", "wait here", "set home", "go home", "pick up",
		"use the bed", "get the door"}[c]
}

var grammar = map[string]command{
	"follow": cmdFollow, "follow me": cmdFollow,
	"stop": cmdStop, "halt": cmdStop,
	"status": cmdStatus, "report": cmdStatus,
	"come": cmdCome, "come here": cmdCome, "come to me": cmdCome,
	"wait": cmdWait, "wait here": cmdWait, "stay": cmdWait, "stay here": cmdWait,
	"set home": cmdSetHome, "set home here": cmdSetHome, "home is here": cmdSetHome, "this is home": cmdSetHome,
	"make this home": cmdSetHome,
	"go home":        cmdGoHome, "home": cmdGoHome, "head home": cmdGoHome, "return home": cmdGoHome,
	"go back home": cmdGoHome,
	"pick that up": cmdPickup, "pick it up": cmdPickup, "pick up": cmdPickup, "pick up that": cmdPickup,
	"grab that": cmdPickup, "grab it": cmdPickup, "get that": cmdPickup,
	"use the bed": cmdBed, "use your bed": cmdBed, "claim the bed": cmdBed, "claim a bed": cmdBed,
	"claim that bed": cmdBed, "go to bed": cmdBed,
	"get the door": cmdDoor, "open the door": cmdDoor, "close the door": cmdDoor, "use the door": cmdDoor,
}

func parse(stripped string) (command, bool) {
	c, ok := grammar[stripped]
	return c, ok
}

func (b *Brain) run(cmd command, c *protocol.Chat, from origin) {
	var (
		order session.Order
		ack   string
	)
	switch cmd {
	case cmdStatus:
		b.say(from, b.status())
		return
	case cmdSetHome:
		b.setHome(c, from)
		return
	case cmdGoHome:
		b.goHome(from)
		return
	case cmdPickup:
		var ok bool
		if order, ack, ok = b.pickup(c, from); !ok {
			return
		}
	case cmdBed, cmdDoor:
		var ok bool
		if order, ack, ok = b.use(cmd, c, from); !ok {
			return
		}
	case cmdFollow:
		order = session.Order{Task: protocol.TaskFollow, Args: &protocol.FollowArgs{Player: c.From.Name, Run: protocol.RunAuto}}
		ack = "Right behind you."
		from.follow = c.From.Name
	case cmdStop:
		order = session.Order{Task: protocol.TaskStop}
		ack = "Stopping."
	case cmdWait:
		order = session.Order{Task: protocol.TaskStop}
		ack = "I'll wait here."
	case cmdCome:
		pos := c.Pos
		stop := 2.0
		run := c.Dist > 15
		order = session.Order{Task: protocol.TaskMoveTo,
			Args: &protocol.MoveToArgs{To: protocol.Target{Pos: &pos}, StopDistance: &stop, Run: &run}}
		ack = "On my way."
	}
	order.Origin = from
	if _, err := b.d.Dispatch(context.Background(), order); err != nil {
		b.opts.Logger.Warn("command not sent", "command", cmd, "err", err)
		if errors.Is(err, session.ErrNotAllowed) {
			b.say(from, "I'm not allowed to do that.")
		}
		return
	}
	b.say(from, ack)
}

// useReach is how far from the speaker, or the bot when the speaker is out
// of sight, a bed or door may be for "use the bed" or "get the door".
const useReach = 10.0

// use chooses the bed or door to interact with: for a bed the bot's own,
// wherever it is in sight; otherwise the nearest the speaker, or the bot
// when it cannot see the speaker. It says why when there is none.
func (b *Brain) use(cmd command, c *protocol.Chat, from origin) (session.Order, string, bool) {
	kind, what, ack := protocol.StructureDoor, "door", "Getting the door."
	if cmd == cmdBed {
		kind, what, ack = protocol.StructureBed, "bed", "Heading to bed."
	}
	near := c.Pos
	if !located(c) {
		if st := b.d.State(); st != nil {
			near = st.Pos
		}
	}
	var (
		best *protocol.PerceivedStructure
		dist = useReach
	)
	if seen := b.d.Perception(); seen != nil {
		for i, s := range seen.Structures {
			if s.Kind != kind {
				continue
			}
			if s.Mine {
				best = &seen.Structures[i]
				break
			}
			if d := near.FlatDistance(s.Pos); d <= dist {
				best, dist = &seen.Structures[i], d
			}
		}
	}
	if best == nil {
		b.say(from, fmt.Sprintf("I don't see a %s nearby.", what))
		return session.Order{}, "", false
	}
	return session.Order{Task: protocol.TaskInteract, Args: &protocol.InteractArgs{Entity: best.ID}}, ack, true
}

// pickupReach is how far from the speaker a drop may lie for "pick that up".
const pickupReach = 10.0

// pickup chooses the drop nearest the speaker, from perception, and checks
// there is room for it. It says why when there is nothing to send.
func (b *Brain) pickup(c *protocol.Chat, from origin) (session.Order, string, bool) {
	var (
		best *protocol.PerceivedResource
		dist = pickupReach
	)
	if seen := b.d.Perception(); seen != nil {
		for i, r := range seen.Resources {
			if r.Kind != protocol.ResourceDrop {
				continue
			}
			if d := c.Pos.FlatDistance(r.Pos); d <= dist {
				best, dist = &seen.Resources[i], d
			}
		}
	}
	if best == nil {
		b.say(from, "I don't see anything to pick up near you.")
		return session.Order{}, "", false
	}
	if inv := b.d.Inventory(); inv != nil && inv.Room(best.Prefab, 0) < 1 {
		b.say(from, "I've no room for that.")
		return session.Order{}, "", false
	}
	order := session.Order{Task: protocol.TaskPickup, Args: &protocol.PickupArgs{Entity: best.ID}}
	return order, fmt.Sprintf("Getting the %s.", strings.ToLower(words(best.Prefab))), true
}

func (b *Brain) setHome(c *protocol.Chat, from origin) {
	if b.opts.Home == nil {
		b.say(from, "I can't keep a home.")
		return
	}
	if _, err := b.opts.Home.SetHome(c.Pos, c.From.Name); err != nil {
		b.opts.Logger.Warn("home not set", "err", err)
		if errors.Is(err, home.ErrNoWorld) {
			b.say(from, "I don't know which world this is yet.")
		} else {
			b.say(from, "I couldn't remember that.")
		}
		return
	}
	b.say(from, "This is home now.")
}

func (b *Brain) goHome(from origin) {
	if b.opts.Home == nil {
		b.say(from, "I don't have a home.")
		return
	}
	if _, err := b.opts.Home.GoHome(context.Background(), from, "told to"); err != nil {
		b.opts.Logger.Warn("not going home", "err", err)
		switch {
		case errors.Is(err, home.ErrNoHome):
			b.say(from, `I don't have a home here yet. Say "set home" where you want it.`)
		case errors.Is(err, home.ErrStranded):
			b.say(from, explain(protocol.TaskMoveTo, protocol.ReasonEncumbered, from.player))
		case errors.Is(err, session.ErrNotAllowed):
			b.say(from, "I'm not allowed to do that.")
		default:
			b.say(from, "I can't go home right now.")
		}
		return
	}
	b.say(from, "Heading home.")
}

func (b *Brain) converse(c *protocol.Chat, from origin, owner bool) {
	voice := b.voice()
	if voice == nil {
		b.say(from, "Hm?")
		return
	}
	now := b.opts.Now()
	b.mu.Lock()
	if last, ok := b.lastVoice[c.From.Name]; ok && now.Sub(last) < b.opts.Cooldown {
		b.mu.Unlock()
		b.opts.Logger.Info("voice cooling down", "player", c.From.Name)
		if c.Channel == channelPanel {
			// In the game the silence is the answer; on the panel it would
			// look like the words were lost.
			b.say(from, "Give me a moment; I spoke just now.")
		}
		return
	}
	b.lastVoice[c.From.Name] = now
	recent := slices.Clone(b.recent)
	b.mu.Unlock()

	req := Request{
		AgentName: b.agentName(),
		Nicknames: b.live().Nicknames,
		Speaker:   c.From.Name,
		Owner:     owner,
		Text:      c.Text,
		Summary:   b.status(),
		Recent:    recent,
	}
	if owner {
		req.Tasks = b.d.AllowedTasks()
		if !located(c) {
			// As the grammar does: no following an owner out of sight.
			req.Tasks = slices.DeleteFunc(slices.Clone(req.Tasks), func(t string) bool { return t == protocol.TaskFollow })
		}
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), b.opts.VoiceTimeout)
		defer cancel()
		reply, err := voice.Reply(ctx, req)
		if err != nil || strings.TrimSpace(reply.Say) == "" && reply.Task == nil {
			b.opts.Logger.Warn("voice failed", "player", c.From.Name, "err", err)
			b.say(from, "Hm?")
			return
		}
		if reply.Task != nil {
			b.propose(reply.Task, from, owner, req.Tasks)
		}
		if s := strings.TrimSpace(reply.Say); s != "" {
			b.say(from, s)
		}
	}()
}

// propose checks a task the Voice suggested exactly as a command is checked:
// owners only, then Dispatch's allowedTasks, capability and argument checks.
func (b *Brain) propose(p *Proposal, from origin, owner bool, offered []string) {
	if !owner {
		b.opts.Logger.Warn("voice proposed a task for a non-owner; dropped", "player", from.player, "task", p.Task)
		return
	}
	if !slices.Contains(offered, p.Task) {
		b.opts.Logger.Warn("voice proposed a task it was not offered; dropped", "player", from.player, "task", p.Task)
		return
	}
	if p.Task == protocol.TaskSay {
		// The reply's own words are the bot's speech; a proposed say would
		// only speak twice.
		return
	}
	var args any
	if len(p.Args) > 0 {
		args = p.Args
	}
	if p.Task == protocol.TaskFollow {
		// Who to find again if a death cuts the follow short.
		var fa protocol.FollowArgs
		if json.Unmarshal(p.Args, &fa) == nil {
			from.follow = fa.Player
		}
	}
	if _, err := b.d.Dispatch(context.Background(), session.Order{Task: p.Task, Args: args, Origin: from}); err != nil {
		b.opts.Logger.Warn("voice proposal refused", "task", p.Task, "args", string(p.Args), "err", err)
		return
	}
	if b.opts.Home != nil {
		// An order in the owner's own words is an order like any other.
		b.opts.Home.Touch()
		b.opts.Home.Ordered()
	}
}

func (b *Brain) say(to origin, text string) {
	if to.channel == channelPanel {
		// For the panel's feed, not the game's chat.
		b.opts.Logger.Info("said to the panel", "to", to.player, "text", text)
		b.remember(Line{From: b.agentName(), Text: text, Bot: true})
		return
	}
	args := &protocol.SayArgs{Text: text}
	switch to.channel {
	case protocol.ChannelWhisper:
		args.Channel, args.To = protocol.ChannelWhisper, to.player
	case protocol.ChannelShout:
		args.Channel = protocol.ChannelShout
	}
	if _, err := b.d.Dispatch(context.Background(), session.Order{Task: protocol.TaskSay, Args: args}); err != nil {
		b.opts.Logger.Warn("could not speak", "text", text, "err", err)
		return
	}
	b.mu.Lock()
	b.spokeTo[to.player] = b.opts.Now()
	b.mu.Unlock()
	b.remember(Line{From: b.agentName(), Text: text, Bot: true})
}

func (b *Brain) remember(l Line) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.recent = append(b.recent, l)
	if len(b.recent) > historyLen {
		b.recent = slices.Delete(b.recent, 0, len(b.recent)-historyLen)
	}
}

// replyChannel answers in kind: a whisper with a whisper, and a shout with a
// shout, since whoever shouted may be too far off to hear normal chat.
func replyChannel(heard string) string {
	switch heard {
	case protocol.ChannelWhisper, protocol.ChannelShout, channelPanel:
		return heard
	}
	return protocol.ChannelNormal
}

func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return "nobody yet"
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// explain turns a failed task into something to say, or "" for nothing.
func explain(task, reason, player string) string {
	if task == protocol.TaskInteract && reason == protocol.ReasonTargetLost {
		return "It's gone."
	}
	if task == protocol.TaskPickup {
		switch reason {
		case protocol.ReasonTargetLost:
			return "It's gone. Someone got to it first."
		case protocol.ReasonInvalidTarget:
			return "I can't get that one."
		}
	}
	switch reason {
	case protocol.ReasonInventoryFull:
		return "I've no room for that."
	case protocol.ReasonTooHeavy:
		return "That's more than I can carry."
	case protocol.ReasonEncumbered:
		return "I'm carrying too much to move. I need to put something down."
	case protocol.ReasonTargetLost:
		return fmt.Sprintf("I lost track of you, %s.", player)
	case protocol.ReasonStuck:
		return "I'm stuck. Can you come back for me?"
	case protocol.ReasonUnreachable:
		return "I can't find a way there."
	case protocol.ReasonDeepWater:
		return "There's deep water in the way, and I'd drown in it. I'll stay on dry land."
	case protocol.ReasonTimeout:
		return "That took too long, so I gave up."
	case protocol.ReasonNotAllowed, protocol.ReasonUnsupported:
		return "I'm not allowed to do that."
	case protocol.ReasonInvalidTarget:
		return "I don't know where that is."
	case protocol.ReasonDied:
		return "I died. Give me a moment."
	case protocol.ReasonSuperseded, protocol.ReasonStopped:
		return ""
	}
	if task == protocol.TaskSay {
		return ""
	}
	return fmt.Sprintf("I couldn't do that (%s).", strings.ReplaceAll(reason, "_", " "))
}
