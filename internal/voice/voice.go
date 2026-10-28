// Package voice is the bot's voice in chat, through terva's Go SDK
// (decision in TKT-01M61T36F36BQFN16H52G329NK). The model gets a short
// summary and recent chat, never the raw streams, and may propose one task
// from the menu by calling a tool. The tools only record the proposal; the
// chat brain checks it like any command before anything reaches the plugin.
package voice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"terva.sh/terva/packages/agent/sdk"
	"terva.sh/terva/packages/core"
	"terva.sh/terva/packages/provider"

	"github.com/warricksothr/renki/internal/chat"
	"github.com/warricksothr/renki/internal/protocol"
)

// Config is the `llm.terva` block of the controller's configuration.
// Anything left empty falls back to terva's own configuration in
// $TERVA_HOME and the provider's environment variables.
type Config struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	BaseURL   string `json:"baseURL"`
	Reasoning string `json:"reasoning"`
	// APIKeyEnv names the environment variable holding the key, so the key
	// never sits in the file.
	APIKeyEnv string `json:"apiKeyEnv"`
	// TervaHome, when set, replaces $TERVA_HOME for this process. terva
	// appends $TERVA_HOME/AGENTS.md, skills and lore to every prompt, so a
	// bot is best given a home of its own.
	TervaHome string `json:"tervaHome"`
}

// Options configure a Voice.
type Options struct {
	Terva   Config
	Persona string
	// DailyTokens caps input plus output tokens per local day; 0 is no cap.
	DailyTokens int
	Logger      *slog.Logger
	Now         func() time.Time
	// newRuntime is replaced in tests.
	newRuntime func(sdk.Config) (runtime, error)
}

// runtime is the part of *sdk.Runtime the voice uses.
type runtime interface {
	Prompt(ctx context.Context, text string, images []sdk.Image) (<-chan sdk.Event, error)
	Cost() sdk.Usage
	Close() error
}

// ErrBudget is returned once the day's token cap is spent.
var ErrBudget = errors.New("daily token cap reached")

// Voice implements chat.Voice.
type Voice struct {
	opts   Options
	apiKey string
	cwd    string

	mu    sync.Mutex
	day   string
	spent int
}

// New checks the configuration and returns a voice. Nothing is called until
// the first Reply.
func New(opts Options) (*Voice, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.newRuntime == nil {
		opts.newRuntime = func(c sdk.Config) (runtime, error) { return sdk.New(c) }
	}
	v := &Voice{opts: opts}
	if env := opts.Terva.APIKeyEnv; env != "" {
		v.apiKey = os.Getenv(env)
		if v.apiKey == "" {
			return nil, fmt.Errorf("llm.terva.apiKeyEnv names %s, which is not set", env)
		}
	}
	if opts.Terva.TervaHome != "" {
		if err := os.Setenv("TERVA_HOME", opts.Terva.TervaHome); err != nil {
			return nil, err
		}
	}
	// An empty working directory, so terva finds no AGENTS.md, skills or
	// lore of a repository the controller happens to run in.
	dir, err := os.MkdirTemp("", "renki-voice-")
	if err != nil {
		return nil, err
	}
	v.cwd = dir
	return v, nil
}

// Reply implements chat.Voice.
func (v *Voice) Reply(ctx context.Context, req chat.Request) (chat.Reply, error) {
	if err := v.checkBudget(); err != nil {
		return chat.Reply{}, err
	}
	rec := &recorder{}
	rt, err := v.opts.newRuntime(sdk.Config{
		Provider:     v.opts.Terva.Provider,
		Model:        v.opts.Terva.Model,
		BaseURL:      v.opts.Terva.BaseURL,
		Reasoning:    v.opts.Terva.Reasoning,
		APIKey:       v.apiKey,
		CWD:          v.cwd,
		SystemPrompt: v.persona(req.AgentName),
		NoTools:      true,
		ExtraTools:   menu(req.Tasks, rec),
		// One step to speak and maybe propose, one to finish speaking after
		// the tool answers.
		MaxSteps: 2,
		// Our tools only record. Yolo keeps the machine's permission rules
		// and classifier out of a turn that can do nothing on this host.
		Yolo: true,
	})
	if err != nil {
		return chat.Reply{}, fmt.Errorf("terva: %w", err)
	}
	defer rt.Close()

	events, err := rt.Prompt(ctx, prompt(req), nil)
	if err != nil {
		return chat.Reply{}, fmt.Errorf("terva: %w", err)
	}
	var text strings.Builder
	var failure string
	for ev := range events {
		switch ev.Type {
		case "text_delta":
			text.WriteString(ev.Delta)
		case "error":
			failure = ev.Error
		case "turn_end":
			if ev.Error != "" {
				failure = ev.Error
			}
		}
	}
	v.charge(rt.Cost())
	if failure != "" {
		return chat.Reply{}, fmt.Errorf("terva: %s", failure)
	}
	reply := chat.Reply{Say: oneLine(text.String()), Task: rec.proposal()}
	v.opts.Logger.Info("voice", "player", req.Speaker, "say", reply.Say, "proposal", reply.Task)
	return reply, nil
}

func (v *Voice) persona(agent string) string {
	if v.opts.Persona != "" {
		return v.opts.Persona
	}
	return fmt.Sprintf("You are %s, a Viking farmhand in Valheim who follows the players around and helps out. "+
		"You are good-humoured, plain-spoken and brief.", agent)
}

func (v *Voice) checkBudget() error {
	if v.opts.DailyTokens <= 0 {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.rollDay()
	if v.spent >= v.opts.DailyTokens {
		return fmt.Errorf("%w (%d of %d)", ErrBudget, v.spent, v.opts.DailyTokens)
	}
	return nil
}

func (v *Voice) charge(u sdk.Usage) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.rollDay()
	v.spent += u.Input + u.Output
}

func (v *Voice) rollDay() {
	if day := v.opts.Now().Format(time.DateOnly); day != v.day {
		v.day, v.spent = day, 0
	}
}

// prompt is the user turn: who said what, the bot's situation, recent chat,
// and what is allowed.
func prompt(req chat.Request) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are in game chat in Valheim. %s just said to you: %q\n\n", req.Speaker, req.Text)
	fmt.Fprintf(&b, "Your situation: %s\n", req.Summary)
	if len(req.Recent) > 0 {
		b.WriteString("\nRecent chat, oldest first:\n")
		for _, l := range req.Recent {
			fmt.Fprintf(&b, "%s: %s\n", l.From, l.Text)
		}
	}
	b.WriteString("\nAnswer in character with one or two short sentences of plain text: no markdown, no stage directions, no quotation marks.")
	if len(req.Tasks) > 0 {
		b.WriteString(" If they ask for something one of your tools does, call exactly one tool as well as answering. Otherwise call none.")
	} else {
		fmt.Fprintf(&b, " You do not take orders from %s, so do not agree to do anything; just talk.", req.Speaker)
	}
	return b.String()
}

// oneLine makes the reply fit a chat line: no line breaks, no wrapping
// quotes.
func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	s = strings.Trim(s, `"“”`)
	const max = 200
	if r := []rune(s); len(r) > max {
		s = string(r[:max-1]) + "…"
	}
	return s
}

// recorder keeps the first proposal of a turn.
type recorder struct {
	mu sync.Mutex
	p  *chat.Proposal
}

func (r *recorder) record(task string, args json.RawMessage) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.p != nil {
		return false
	}
	r.p = &chat.Proposal{Task: task, Args: slices.Clone(args)}
	return true
}

func (r *recorder) proposal() *chat.Proposal {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.p
}

// tool offers one task to the model.
type tool struct {
	name, desc string
	schema     json.RawMessage
	rec        *recorder
}

func (t tool) Name() string            { return t.name }
func (t tool) Description() string     { return t.desc }
func (t tool) Schema() json.RawMessage { return t.schema }

func (t tool) Execute(_ context.Context, args json.RawMessage, _ func(string)) (core.ToolResult, error) {
	if !t.rec.record(t.name, args) {
		return text("Only one action per reply. This one was not taken.", true), nil
	}
	return text("Noted. It will be checked and carried out if allowed.", false), nil
}

func text(s string, isErr bool) core.ToolResult {
	return core.ToolResult{Content: []provider.Content{provider.TextBlock{Text: s}}, IsError: isErr}
}

// Emotes are the names the model may pick: Valheim's Emotes enum, lower
// case, as Emote.DoEmote uses them. The plugin checks them again.
var Emotes = []string{"wave", "sit", "challenge", "cheer", "nonono", "thumbsup", "point", "blowkiss", "bow",
	"cower", "cry", "despair", "flex", "comehere", "headbang", "kneel", "laugh", "roar", "shrug", "dance",
	"relax", "toast", "rest", "vibe", "loveyou"}

var catalogue = map[string]struct {
	desc   string
	schema string
}{
	protocol.TaskStop: {"Stop whatever you are doing and stand still.",
		`{"type":"object","properties":{}}`},
	protocol.TaskFollow: {"Follow a player, keeping a few metres behind them.",
		`{"type":"object","properties":{"player":{"type":"string","description":"The player's name."},"distance":{"type":"number","description":"Metres to keep; default 3.5."}},"required":["player"]}`},
	protocol.TaskMoveTo: {"Walk to an entity by its id.",
		`{"type":"object","properties":{"to":{"type":"string","description":"The entity id."},"run":{"type":"boolean"}},"required":["to"]}`},
	protocol.TaskLookAt: {"Turn to look at an entity by its id.",
		`{"type":"object","properties":{"to":{"type":"string","description":"The entity id."}},"required":["to"]}`},
	protocol.TaskEmote: {"Play an emote.",
		`{"type":"object","properties":{"name":{"type":"string","enum":` + mustJSON(Emotes) + `}},"required":["name"]}`},
}

// menu returns a tool for each allowed task the model may propose. `say`
// is never offered: the reply's own text is what the bot says.
func menu(allowed []string, rec *recorder) []core.Tool {
	var tools []core.Tool
	for _, name := range allowed {
		c, ok := catalogue[name]
		if !ok {
			continue
		}
		tools = append(tools, tool{name: name, desc: c.desc, schema: json.RawMessage(c.schema), rec: rec})
	}
	return tools
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
