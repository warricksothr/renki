package chat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

// fakeSession records orders, checking them the way Session.Dispatch does.
type fakeSession struct {
	mu         sync.Mutex
	orders     []session.Order
	allowed    []string
	state      *protocol.State
	inventory  *protocol.Inventory
	perception *protocol.Perception
}

func (f *fakeSession) Dispatch(_ context.Context, o session.Order) (*session.Task, error) {
	if !slices.Contains(f.allowed, o.Task) {
		return nil, session.ErrNotAllowed
	}
	req, err := protocol.NewTask("t", o.Task, o.Args, 0)
	if err != nil {
		return nil, err
	}
	if _, err := req.DecodeArgs(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.orders = append(f.orders, o)
	return &session.Task{ID: "t", Name: o.Task, Origin: o.Origin}, nil
}

func (f *fakeSession) State() *protocol.State           { return f.state }
func (f *fakeSession) Inventory() *protocol.Inventory   { return f.inventory }
func (f *fakeSession) Perception() *protocol.Perception { return f.perception }
func (f *fakeSession) AllowedTasks() []string           { return f.allowed }

func (f *fakeSession) sent() []session.Order {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.orders)
}

// said returns what the bot said, in order.
func (f *fakeSession) said() []string {
	var out []string
	for _, o := range f.sent() {
		if o.Task == protocol.TaskSay {
			out = append(out, o.Args.(*protocol.SayArgs).Text)
		}
	}
	return out
}

func (f *fakeSession) tasks() []string {
	var out []string
	for _, o := range f.sent() {
		if o.Task != protocol.TaskSay {
			out = append(out, o.Task)
		}
	}
	return out
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }

func (c *clock) advance(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

func newBrain(voice Voice) (*Brain, *fakeSession, *clock) {
	f := &fakeSession{allowed: []string{"stop", "move_to", "follow", "look_at", "say", "emote"}}
	c := &clock{t: time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)}
	b := New(Options{AgentName: "Bjorn", Owners: []string{"Sothr"}, Voice: voice,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: c.now}, f)
	return b, f, c
}

func line(from, text string) *protocol.Chat {
	return &protocol.Chat{From: protocol.ChatSender{Name: from}, Channel: protocol.ChannelNormal, Text: text,
		Pos: protocol.Vec3{10, 0, 10}, Dist: 4}
}

func TestGrammar(t *testing.T) {
	tests := []struct {
		text string
		task string
		say  string
	}{
		{"bjorn, follow me", protocol.TaskFollow, "Right behind you."},
		{"Bjorn follow!", protocol.TaskFollow, "Right behind you."},
		{"BJORN STOP", protocol.TaskStop, "Stopping."},
		{"bjorn: come here please", protocol.TaskMoveTo, "On my way."},
		{"wait here, Bjorn", protocol.TaskStop, "I'll wait here."},
	}
	for _, tc := range tests {
		t.Run(tc.text, func(t *testing.T) {
			b, f, _ := newBrain(nil)
			b.Chat(line("Sothr", tc.text))
			if got := f.tasks(); len(got) != 1 || got[0] != tc.task {
				t.Fatalf("tasks %v; want [%s]", got, tc.task)
			}
			if got := f.said(); len(got) != 1 || got[0] != tc.say {
				t.Errorf("said %q; want %q", got, tc.say)
			}
		})
	}
}

func TestFollowTargetsTheSpeakerAndComeUsesTheirPosition(t *testing.T) {
	b, f, _ := newBrain(nil)
	b.Chat(line("Sothr", "bjorn follow me"))
	b.Chat(line("Sothr", "bjorn come here"))
	orders := f.sent()
	if a := orders[0].Args.(*protocol.FollowArgs); a.Player != "Sothr" || a.Run != protocol.RunAuto {
		t.Errorf("follow args %+v", a)
	}
	if a := orders[2].Args.(*protocol.MoveToArgs); *a.To.Pos != (protocol.Vec3{10, 0, 10}) {
		t.Errorf("move_to args %+v", a)
	}
}

func TestNotAddressedIsIgnored(t *testing.T) {
	b, f, _ := newBrain(nil)
	b.Chat(line("Sothr", "follow me"))
	b.Chat(line("Sothr", "bjornsson is a fine name"))
	if len(f.sent()) != 0 {
		t.Fatalf("sent %v for chat not addressed to the bot", f.sent())
	}
}

func TestNameFollowsTheCharacterUnlessConfigured(t *testing.T) {
	b, f, c := newBrain(nil)
	b.opts.AgentName = ""
	b.Chat(line("Sothr", "tasleroch follow"))
	if len(f.sent()) != 0 {
		t.Fatalf("sent %v before the bot had a name", f.sent())
	}

	f.state = &protocol.State{Character: "Tasleroch"}
	b.Chat(line("Sothr", "tasleroch follow"))
	f.state = &protocol.State{Character: "Ylva"}
	c.advance(time.Minute) // past the reply window
	b.Chat(line("Sothr", "tasleroch stop"))
	b.Chat(line("Sothr", "Ylva, stop"))
	if got := f.tasks(); !slices.Equal(got, []string{protocol.TaskFollow, protocol.TaskStop}) {
		t.Errorf("tasks %v; want follow as Tasleroch, then stop as Ylva", got)
	}

	b2, f2, _ := newBrain(nil)
	f2.state = &protocol.State{Character: "Tasleroch"}
	b2.Chat(line("Sothr", "tasleroch follow"))
	b2.Chat(line("Sothr", "bjorn follow"))
	if got := f2.tasks(); !slices.Equal(got, []string{protocol.TaskFollow}) {
		t.Errorf("tasks %v; the configured name Bjorn should win over the character's", got)
	}
}

func TestNicknames(t *testing.T) {
	b, f, c := newBrain(nil)
	b.opts.AgentName = ""
	b.opts.Nicknames = []string{"Tas", " "}
	f.state = &protocol.State{Character: "Tasleroch"}
	b.Chat(line("Sothr", "tasty stew"))
	b.Chat(line("Sothr", "TAS follow me"))
	c.advance(time.Minute) // past the reply window
	b.Chat(line("Sothr", "Tasleroch, follow"))
	if got := f.tasks(); !slices.Equal(got, []string{protocol.TaskFollow, protocol.TaskFollow}) {
		t.Errorf("tasks %v; want follow twice, by nickname and by name", got)
	}
}

func TestWhisperIsAddressedAndAnsweredByWhisper(t *testing.T) {
	b, f, _ := newBrain(nil)
	c := line("Sothr", "stop")
	c.Channel = protocol.ChannelWhisper
	b.Chat(c)
	if got := f.tasks(); len(got) != 1 || got[0] != protocol.TaskStop {
		t.Fatalf("tasks %v", got)
	}
	say := f.sent()[1].Args.(*protocol.SayArgs)
	if say.Channel != protocol.ChannelWhisper || say.To != "Sothr" {
		t.Errorf("reply %+v; want a whisper to Sothr", say)
	}
}

func TestReplyWindow(t *testing.T) {
	b, f, c := newBrain(nil)
	b.Chat(line("Sothr", "bjorn follow"))
	c.advance(10 * time.Second)
	b.Chat(line("Sothr", "stop"))
	if got := f.tasks(); len(got) != 2 {
		t.Fatalf("within the window, tasks %v; want follow and stop", got)
	}
	c.advance(25 * time.Second)
	b.Chat(line("Sothr", "follow"))
	if got := f.tasks(); len(got) != 2 {
		t.Fatalf("after the window, tasks %v; want no new task", got)
	}
}

func TestOnlyOwnersCommand(t *testing.T) {
	b, f, _ := newBrain(nil)
	b.Chat(line("Mallory", "bjorn follow me"))
	if len(f.tasks()) != 0 {
		t.Fatalf("a non-owner's command was sent: %v", f.tasks())
	}
	if got := f.said(); len(got) != 1 || !strings.Contains(got[0], "only take orders from Sothr") {
		t.Errorf("said %q; want a polite refusal", got)
	}
}

func TestStatus(t *testing.T) {
	b, f, _ := newBrain(nil)
	b.Chat(line("Sothr", "bjorn status"))
	if got := f.said(); got[0] != "I'm not in the world yet." {
		t.Errorf("before any state: %q", got)
	}
	f.state = &protocol.State{Phase: protocol.PhaseInWorld, HP: 47, HPMax: 60, Stamina: 38, StaminaMax: 75,
		Biome: "BlackForest", Env: "Rain", Time: protocol.GameTime{Day: 14, DayFraction: 0.81},
		Task: &protocol.TaskRef{Task: "follow", Status: "running"}}
	b.Chat(line("Sothr", "bjorn status"))
	want := "Health 47/60, stamina 38/75. Black Forest, rain, day 14, evening. Following."
	if got := f.said()[1]; got != want {
		t.Errorf("status %q\nwant   %q", got, want)
	}
	if len(f.tasks()) != 0 {
		t.Error("status sent a body task")
	}

	f.inventory = &protocol.Inventory{Size: [2]int{8, 4}, Items: []protocol.InventoryItem{
		{Slot: [2]int{0, 0}, Prefab: "AxeBronze", Stack: 1},
		{Slot: [2]int{1, 0}, Prefab: "Wood", Stack: 50, StackMax: 50},
		{Slot: [2]int{2, 0}, Prefab: "Wood", Stack: 12, StackMax: 50},
	}}
	b.Chat(line("Sothr", "bjorn status"))
	want += " Carrying 62 Wood, 1 Axe Bronze; 29 slots free."
	if got := f.said()[2]; got != want {
		t.Errorf("status %q\nwant   %q", got, want)
	}

	// The weight, and what being over it means.
	for i, c := range []struct {
		weight, stamina float64
		flags           []string
		want            string
	}{
		{140, 38, nil, " Weight 140/300."},
		{305, 12, []string{protocol.FlagEncumbered}, " Overloaded, 305/300: I can only walk, and not for long."},
		{305, 0, []string{protocol.FlagEncumbered, protocol.FlagStranded}, " Overloaded, 305/300: I can't move until I carry less."},
	} {
		st := *f.state
		st.Weight, st.WeightMax, st.Stamina, st.Flags = c.weight, 300, c.stamina, c.flags
		f.state = &st
		b.Chat(line("Sothr", "bjorn status"))
		if got := f.said()[3+i]; !strings.HasSuffix(got, "free."+c.want) {
			t.Errorf("weight %v, stamina %v: %q\nwant it to end %q", c.weight, c.stamina, got, c.want)
		}
	}
}

func TestOverloadIsExplained(t *testing.T) {
	b, f, _ := newBrain(nil)
	b.Chat(line("Sothr", "bjorn follow me"))
	b.TaskDone(doneTask(f.sent()[0], protocol.StatusFailed, protocol.ReasonEncumbered))
	if got := f.said(); len(got) != 2 || got[1] != "I'm carrying too much to move. I need to put something down." {
		t.Errorf("said %q", got)
	}
	if explain(protocol.TaskPickup, protocol.ReasonTooHeavy, "Sothr") != "That's more than I can carry." {
		t.Error("too_heavy")
	}
}

// TestDeepWaterIsSaidAndDoesNotSendTheBotHome: the trip home would cross
// the same water.
func TestDeepWaterIsSaidAndDoesNotSendTheBotHome(t *testing.T) {
	b, f, h := homeBrain()
	h.home = &protocol.Vec3{}
	b.Chat(line("Sothr", "bjorn come here"))
	b.TaskDone(doneTask(f.sent()[0], protocol.StatusFailed, protocol.ReasonDeepWater))
	if got := f.said(); len(got) != 2 || !strings.Contains(got[1], "deep water") {
		t.Errorf("said %q", got)
	}
	if len(h.trips) != 0 {
		t.Errorf("went home: %v", h.trips)
	}
}

func TestFailedTaskIsExplainedToWhoAsked(t *testing.T) {
	b, f, _ := newBrain(nil)
	b.Chat(line("Sothr", "bjorn follow me"))
	follow := f.sent()[0]
	task := doneTask(follow, protocol.StatusFailed, protocol.ReasonTargetLost)
	b.TaskDone(task)
	if got := f.said(); len(got) != 2 || got[1] != "I lost track of you, Sothr." {
		t.Errorf("said %q", got)
	}
	// Superseded is the normal end of a task and says nothing.
	b.TaskDone(doneTask(follow, protocol.StatusCancelled, protocol.ReasonSuperseded))
	if len(f.said()) != 2 {
		t.Errorf("a superseded task was explained: %q", f.said())
	}
}

func TestShoutIsAnsweredByShoutEvenWhenStuck(t *testing.T) {
	b, f, _ := newBrain(nil)
	c := line("Sothr", "bjorn follow me")
	c.Channel = protocol.ChannelShout
	b.Chat(c)
	follow := f.sent()[0]
	b.TaskDone(doneTask(follow, protocol.StatusFailed, protocol.ReasonStuck))
	var says []*protocol.SayArgs
	for _, o := range f.sent() {
		if o.Task == protocol.TaskSay {
			says = append(says, o.Args.(*protocol.SayArgs))
		}
	}
	if len(says) != 2 || says[1].Text != "I'm stuck. Can you come back for me?" {
		t.Fatalf("said %+v", says)
	}
	for _, s := range says {
		if s.Channel != protocol.ChannelShout || s.To != "" {
			t.Errorf("%q went to %q; want a shout", s.Text, s.Channel)
		}
	}
}

// doneTask makes a finished task through a real session, since its status
// is only settable from inside that package.
func doneTask(o session.Order, status, reason string) *session.Task {
	return session.Finished(o.Task, o.Origin, status, reason)
}

type stubVoice struct {
	reply Reply
	err   error
	got   chan Request
}

func (v *stubVoice) Reply(_ context.Context, req Request) (Reply, error) {
	v.got <- req
	return v.reply, v.err
}

func waitSaid(t *testing.T, f *fakeSession, n int) []string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for len(f.said()) < n {
		if time.Now().After(deadline) {
			t.Fatalf("said %q; waited for %d lines", f.said(), n)
		}
		time.Sleep(5 * time.Millisecond)
	}
	return f.said()
}

func TestUnmatchedGoesToVoice(t *testing.T) {
	v := &stubVoice{got: make(chan Request, 1), reply: Reply{Say: "Wet, bruised, and still swinging.",
		Task: &Proposal{Task: "emote", Args: json.RawMessage(`{"name":"wave"}`)}}}
	b, f, _ := newBrain(v)
	b.Chat(line("Sothr", "hey"))
	b.Chat(line("Sothr", "bjorn how are you holding up?"))
	req := <-v.got
	if !req.Owner || req.Speaker != "Sothr" || len(req.Tasks) == 0 || req.Summary == "" {
		t.Errorf("request %+v", req)
	}
	if len(req.Recent) != 2 || req.Recent[0].Text != "hey" {
		t.Errorf("recent chat %+v; want both lines", req.Recent)
	}
	waitSaid(t, f, 1)
	if got := f.tasks(); len(got) != 1 || got[0] != "emote" {
		t.Errorf("tasks %v; want the proposed emote", got)
	}
}

func TestVoiceProposalsAreChecked(t *testing.T) {
	cases := []struct {
		name     string
		speaker  string
		proposal Proposal
	}{
		{"non-owner", "Mallory", Proposal{Task: "follow", Args: json.RawMessage(`{"player":"Mallory"}`)}},
		{"not allowed", "Sothr", Proposal{Task: "attack", Args: json.RawMessage(`{"entity":"0:1"}`)}},
		{"bad args", "Sothr", Proposal{Task: "follow", Args: json.RawMessage(`{"distance":2}`)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := &stubVoice{got: make(chan Request, 1), reply: Reply{Say: "Sure.", Task: &tc.proposal}}
			b, f, _ := newBrain(v)
			b.Chat(line(tc.speaker, "bjorn do the thing"))
			req := <-v.got
			if tc.speaker != "Sothr" && len(req.Tasks) != 0 {
				t.Errorf("a non-owner was offered tasks %v", req.Tasks)
			}
			waitSaid(t, f, 1)
			if got := f.tasks(); len(got) != 0 {
				t.Errorf("proposal reached the plugin: %v", got)
			}
		})
	}
}

func TestVoiceFailureSaysHm(t *testing.T) {
	v := &stubVoice{got: make(chan Request, 1), err: errors.New("provider down")}
	b, f, _ := newBrain(v)
	b.Chat(line("Sothr", "bjorn tell me a story"))
	<-v.got
	if got := waitSaid(t, f, 1); got[0] != "Hm?" {
		t.Errorf("said %q", got)
	}
}

func TestNoVoiceSaysHm(t *testing.T) {
	b, f, _ := newBrain(nil)
	b.Chat(line("Sothr", "bjorn tell me a story"))
	if got := f.said(); len(got) != 1 || got[0] != "Hm?" {
		t.Errorf("said %q", got)
	}
}

func TestVoiceCooldownPerPlayer(t *testing.T) {
	v := &stubVoice{got: make(chan Request, 4), reply: Reply{Say: "Aye."}}
	b, f, c := newBrain(v)
	b.Chat(line("Sothr", "bjorn hello"))
	<-v.got
	waitSaid(t, f, 1)
	c.advance(time.Second)
	b.Chat(line("Sothr", "bjorn hello again"))
	b.Chat(line("Ada", "bjorn hello"))
	waitSaid(t, f, 2)
	if len(v.got) != 1 { // Ada's; Sothr's second is still cooling down
		t.Errorf("voice called %d more times; want 1 (Ada only)", len(v.got))
	}
}

type fakeHome struct {
	world    string
	home     *protocol.Vec3
	homeward map[string]bool
	trips    []string
	touched  int
	ordered  int
	resume   func(context.Context) bool
	stranded bool
	left     []home.Cache
}

func (h *fakeHome) SetHome(pos protocol.Vec3, by string) (string, error) {
	if h.world == "" {
		return "", home.ErrNoWorld
	}
	h.home = &pos
	return h.world, nil
}

func (h *fakeHome) GoHome(_ context.Context, origin any, why string) (*session.Task, error) {
	if h.home == nil {
		return nil, home.ErrNoHome
	}
	if h.stranded {
		return nil, home.ErrStranded
	}
	h.trips = append(h.trips, why)
	t := session.Finished(protocol.TaskMoveTo, origin, protocol.StatusRunning, "")
	h.homeward[t.ID] = true
	return t, nil
}

func (h *fakeHome) Homeward(id string) bool { return h.homeward[id] }
func (h *fakeHome) ReturnOn(reason string) bool {
	return reason == protocol.ReasonTargetLost || reason == protocol.ReasonStuck
}
func (h *fakeHome) HomeDistance() (float64, bool)                 { return 42, h.home != nil }
func (h *fakeHome) LeftBehind() []home.Cache                      { return h.left }
func (h *fakeHome) Touch()                                        { h.touched++ }
func (h *fakeHome) ResumeAfterDeath(f func(context.Context) bool) { h.resume = f }
func (h *fakeHome) Ordered()                                      { h.resume = nil; h.ordered++ }

func homeBrain() (*Brain, *fakeSession, *fakeHome) {
	b, f, _ := newBrain(nil)
	h := &fakeHome{world: "Midgard", homeward: map[string]bool{}}
	b.opts.Home = h
	return b, f, h
}

func TestStatusSaysWhatWasLeftBehind(t *testing.T) {
	b, f, h := homeBrain()
	h.home = &protocol.Vec3{}
	f.state = &protocol.State{Phase: protocol.PhaseInWorld, Pos: protocol.Vec3{0, 0, 0}}
	h.left = []home.Cache{
		{Pos: protocol.Vec3{30, 0, 40}, Items: map[string]int{"Wood": 6, "Stone": 2}},
		{Pos: protocol.Vec3{300, 0, 0}, Items: map[string]int{"Flint": 1}},
	}
	b.Chat(line("Sothr", "bjorn status"))
	want := " Home is 42 m away. I left 6 Wood, 2 Stone 50 m from here, and things in one other place."
	if got := f.said(); len(got) != 1 || !strings.HasSuffix(got[0], want) {
		t.Errorf("said %q\nwant it to end %q", got, want)
	}
}

func TestStrandedBotSaysWhyItCannotGoHome(t *testing.T) {
	b, f, h := homeBrain()
	h.home, h.stranded = &protocol.Vec3{}, true
	b.Chat(line("Sothr", "bjorn go home"))
	if got := f.said(); len(got) != 1 || got[0] != "I'm carrying too much to move. I need to put something down." {
		t.Errorf("go home: %q", got)
	}
	// A failure that would send it home says only what went wrong.
	b.Chat(line("Sothr", "bjorn follow me"))
	b.TaskDone(doneTask(f.sent()[1], protocol.StatusFailed, protocol.ReasonTargetLost))
	if got := f.said(); len(got) != 3 || got[2] != "I lost track of you, Sothr." || len(h.trips) != 0 {
		t.Errorf("said %q, trips %v", got, h.trips)
	}
}

func TestSetAndGoHome(t *testing.T) {
	b, f, h := homeBrain()
	b.Chat(line("Sothr", "bjorn go home"))
	c := line("Sothr", "Bjorn, set home here")
	c.Pos = protocol.Vec3{7, 8, 9}
	b.Chat(c)
	b.Chat(line("Sothr", "bjorn home"))
	want := []string{`I don't have a home here yet. Say "set home" where you want it.`, "This is home now.", "Heading home."}
	if got := f.said(); !slices.Equal(got, want) {
		t.Errorf("said %q", got)
	}
	if *h.home != (protocol.Vec3{7, 8, 9}) || len(h.trips) != 1 || h.trips[0] != "told to" || h.touched != 3 {
		t.Errorf("home %+v", h)
	}
	b.Chat(line("Mallory", "bjorn set home"))
	if *h.home != (protocol.Vec3{7, 8, 9}) {
		t.Error("a non-owner moved home")
	}
}

func TestFailedFollowGoesHomeButAFailedTripHomeDoesNot(t *testing.T) {
	b, f, h := homeBrain()
	h.home = &protocol.Vec3{1, 0, 1}
	b.Chat(line("Sothr", "bjorn follow me"))
	follow := f.sent()[0]
	b.TaskDone(doneTask(follow, protocol.StatusFailed, protocol.ReasonTargetLost))
	if got := f.said(); got[len(got)-1] != "I lost track of you, Sothr. Heading home." || len(h.trips) != 1 {
		t.Fatalf("said %q, trips %v", got, h.trips)
	}

	// The trip home itself gets stuck: say so, and do not try again.
	trip := session.Finished(protocol.TaskMoveTo, origin{player: "Sothr", channel: protocol.ChannelNormal},
		protocol.StatusFailed, protocol.ReasonStuck)
	h.homeward[trip.ID] = true
	b.TaskDone(trip)
	if got := f.said(); got[len(got)-1] != "I'm stuck. Can you come back for me?" || len(h.trips) != 1 {
		t.Errorf("said %q, trips %v", got, h.trips)
	}
}

func TestAnOrderInTheVoiceCountsAsAnOrder(t *testing.T) {
	v := &stubVoice{got: make(chan Request, 1), reply: Reply{Say: "Right behind you.",
		Task: &Proposal{Task: "follow", Args: json.RawMessage(`{"player":"Sothr"}`)}}}
	b, f, _ := newBrain(v)
	h := &fakeHome{world: "Midgard", homeward: map[string]bool{}}
	b.opts.Home = h
	b.Chat(line("Sothr", "bjorn wait for me where you wake up and then tag along"))
	<-v.got
	waitSaid(t, f, 1)
	if h.ordered != 1 || h.touched != 1 {
		t.Errorf("ordered %d, touched %d; want 1 each", h.ordered, h.touched)
	}

	// That follow dies too: it is left to resume, for the same player.
	follow := f.sent()[0]
	b.TaskDone(doneTask(follow, protocol.StatusFailed, protocol.ReasonDied))
	if h.resume == nil {
		t.Error("no resume left after a follow the voice ordered died")
	}
}

func TestADeadFollowLeavesAResumeForTheSamePlayer(t *testing.T) {
	b, f, h := homeBrain()
	c := line("Sothr", "bjorn follow me")
	c.Channel = protocol.ChannelShout
	b.Chat(c)
	follow := f.sent()[0]
	b.TaskDone(doneTask(follow, protocol.StatusFailed, protocol.ReasonDied))
	if h.resume == nil {
		t.Fatal("no resume left after a follow died")
	}
	if !h.resume(context.Background()) {
		t.Fatal("the resume did not start")
	}
	orders := f.sent()
	last, said := orders[len(orders)-2], orders[len(orders)-1]
	if a, ok := last.Args.(*protocol.FollowArgs); !ok || a.Player != "Sothr" {
		t.Errorf("resumed %+v", last)
	}
	if s := said.Args.(*protocol.SayArgs); s.Text != "I'm back. Coming to find you, Sothr." || s.Channel != protocol.ChannelShout {
		t.Errorf("said %+v", s)
	}

	// A newer order, even while dead, clears the resume.
	b.TaskDone(doneTask(follow, protocol.StatusFailed, protocol.ReasonDied))
	b.Chat(line("Sothr", "bjorn go home"))
	if h.resume != nil {
		t.Error("an order after the death left the old follow to resume")
	}

	// A come-here that dies leaves nothing to resume.
	b.Chat(line("Sothr", "bjorn come here"))
	come := f.sent()[len(f.sent())-2]
	b.TaskDone(doneTask(come, protocol.StatusFailed, protocol.ReasonDied))
	if h.resume != nil {
		t.Error("a come-here left a resume")
	}
}

func TestPickThatUp(t *testing.T) {
	b, f, _ := newBrain(nil)
	f.allowed = append(f.allowed, protocol.TaskPickup)

	b.Chat(line("Sothr", "bjorn pick that up"))
	if got := f.said(); got[0] != "I don't see anything to pick up near you." {
		t.Errorf("with nothing seen: %q", got)
	}

	// Nearest the speaker (at 10, 10), not the bot, and within reach.
	f.perception = &protocol.Perception{Resources: []protocol.PerceivedResource{
		{ID: "0:1", Kind: protocol.ResourceDrop, Prefab: "Stone", Pos: protocol.Vec3{16, 0, 10}},
		{ID: "0:2", Kind: protocol.ResourceDrop, Prefab: "Wood", Pos: protocol.Vec3{11, 0, 11}},
		{ID: "0:3", Kind: protocol.ResourceTree, Prefab: "Beech1", Pos: protocol.Vec3{10, 0, 10}},
		{ID: "0:4", Kind: protocol.ResourceDrop, Prefab: "Resin", Pos: protocol.Vec3{40, 0, 40}},
	}}
	pickups := func() []session.Order {
		return slices.DeleteFunc(f.sent(), func(o session.Order) bool { return o.Task != protocol.TaskPickup })
	}
	b.Chat(line("Sothr", "bjorn pick that up"))
	sent := pickups()
	if len(sent) != 1 || sent[0].Args.(*protocol.PickupArgs).Entity != "0:2" {
		t.Fatalf("sent %+v; want pickup of 0:2", sent)
	}
	if got := f.said(); got[1] != "Getting the wood." {
		t.Errorf("ack %q", got[1])
	}
	b.TaskDone(doneTask(sent[0], protocol.StatusSucceeded, ""))
	b.TaskDone(doneTask(sent[0], protocol.StatusFailed, protocol.ReasonTargetLost))
	b.TaskDone(session.FinishedWith(sent[0].Task, sent[0].Origin, protocol.StatusSucceeded, "",
		"took 30 Wood; no room for the other 12"))
	if got := f.said(); got[2] != "Got it." || got[3] != "It's gone. Someone got to it first." ||
		got[4] != "I got some of it, but I'm out of room." {
		t.Errorf("results %q", got[2:])
	}

	// No slot free and no stack of it to add to.
	f.inventory = &protocol.Inventory{Size: [2]int{1, 1}, Items: []protocol.InventoryItem{{Prefab: "Stone", Stack: 50, StackMax: 50}}}
	b.Chat(line("Sothr", "bjorn grab that"))
	if got := f.said(); got[5] != "I've no room for that." || len(pickups()) != 1 {
		t.Errorf("when full: %q, sent %d", got[5], len(pickups()))
	}
}

// TestFromThePanel: words typed into the panel need no name, are answered
// on the panel, not in the game, use the owner's position when the bot
// sees them, and refuse what needs it when it does not.
func TestFromThePanel(t *testing.T) {
	b, f, _ := newBrain(nil)
	if err := b.FromPanel("Mallory", "follow me"); !errors.Is(err, ErrNotOwner) {
		t.Errorf("a non-owner: %v", err)
	}
	if err := b.FromPanel("Sothr", "stop"); !errors.Is(err, ErrNoBot) {
		t.Errorf("with no bot in the world: %v", err)
	}
	f.state = &protocol.State{Phase: protocol.PhaseDead}
	if err := b.FromPanel("Sothr", "stop"); err != nil {
		t.Fatal(err)
	}
	if got := f.tasks(); len(got) != 1 || got[0] != protocol.TaskStop {
		t.Fatalf("tasks %v", got)
	}
	if len(f.said()) != 0 {
		t.Errorf("answered the panel in the game: %v", f.said())
	}
	b.FromPanel("Sothr", "come here")
	if got := f.tasks(); len(got) != 1 {
		t.Errorf("came with no idea where: %v", got)
	}
	f.perception = &protocol.Perception{Characters: []protocol.PerceivedCharacter{
		{Kind: protocol.CharacterPlayer, Name: "Sothr", Pos: protocol.Vec3{5, 0, 7}}}}
	f.state.T = 100 // the sighting is stale
	b.FromPanel("Sothr", "come here")
	if got := f.tasks(); len(got) != 1 {
		t.Errorf("came on a stale sighting: %v", got)
	}
	f.perception.T = 99
	b.FromPanel("Sothr", "come here")
	orders := f.sent()
	if a, ok := orders[len(orders)-1].Args.(*protocol.MoveToArgs); !ok || *a.To.Pos != (protocol.Vec3{5, 0, 7}) {
		t.Errorf("come: %+v", orders[len(orders)-1])
	}
}

// TestNoFollowForAnUnseenPanelOwner: the voice is not offered follow for an
// owner on the panel the bot cannot see, and a follow it proposes anyway
// is not carried out.
func TestNoFollowForAnUnseenPanelOwner(t *testing.T) {
	v := &stubVoice{got: make(chan Request, 1), reply: Reply{Say: "Coming.",
		Task: &Proposal{Task: "follow", Args: json.RawMessage(`{"player":"Sothr"}`)}}}
	b, f, _ := newBrain(v)
	f.state = &protocol.State{Phase: protocol.PhaseInWorld}
	if err := b.FromPanel("Sothr", "could you keep me company"); err != nil {
		t.Fatal(err)
	}
	req := <-v.got
	if slices.Contains(req.Tasks, protocol.TaskFollow) {
		t.Errorf("offered follow for an unseen owner: %v", req.Tasks)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && len(f.tasks()) == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	if got := f.tasks(); len(got) != 0 {
		t.Errorf("carried out %v", got)
	}
}

// TestUseTheBedAndDoor: going to bed is to the bot's own bed only;
// claiming a bed or getting a door takes the nearest to the speaker, or to
// the bot when the speaker is out of sight; and what the game said comes
// back in the bot's words.
func TestUseTheBedAndDoor(t *testing.T) {
	b, f, _ := newBrain(nil)
	f.allowed = append(f.allowed, protocol.TaskInteract)
	uses := func() []string {
		var ids []string
		for _, o := range f.sent() {
			if o.Task == protocol.TaskInteract {
				ids = append(ids, o.Args.(*protocol.InteractArgs).Entity)
			}
		}
		return ids
	}

	b.Chat(line("Sothr", "bjorn claim the bed"))
	if got := f.said(); got[0] != "I don't see a bed nearby." {
		t.Errorf("with nothing seen: %q", got)
	}

	// The speaker stands at 10, 10.
	f.perception = &protocol.Perception{Structures: []protocol.PerceivedStructure{
		{ID: "1:1", Kind: protocol.StructureBed, Pos: protocol.Vec3{11, 0, 10}},
		{ID: "1:2", Kind: protocol.StructureDoor, Pos: protocol.Vec3{30, 0, 10}},
		{ID: "1:3", Kind: protocol.StructureDoor, Pos: protocol.Vec3{12, 0, 12}},
	}}
	// No bed of its own: going to bed claims nothing.
	b.Chat(line("Sothr", "bjorn go to bed"))
	if got := f.said(); got[len(got)-1] != "I don't see my bed. Tell me to claim one." || len(uses()) != 0 {
		t.Fatalf("went to a bed not its own: %q, %v", got, uses())
	}
	b.Chat(line("Sothr", "bjorn claim the bed"))
	b.Chat(line("Sothr", "bjorn open the door"))
	if got := uses(); !slices.Equal(got, []string{"1:1", "1:3"}) {
		t.Fatalf("used %v", got)
	}
	// Its own bed, however far, and not one to claim again.
	f.perception.Structures[0].Mine = false
	f.perception.Structures = append(f.perception.Structures,
		protocol.PerceivedStructure{ID: "1:4", Kind: protocol.StructureBed, Pos: protocol.Vec3{35, 0, 35}, Mine: true})
	b.Chat(line("Sothr", "bjorn go to bed"))
	if got := uses(); got[len(got)-1] != "1:4" {
		t.Errorf("used %v; want its own bed 1:4", got)
	}
	// Its own bed is nearest: another free one in reach is claimed instead,
	// and with none, nothing is.
	f.perception.Structures[0].Mine = true
	f.perception.Structures = append(f.perception.Structures,
		protocol.PerceivedStructure{ID: "1:5", Kind: protocol.StructureBed, Pos: protocol.Vec3{14, 0, 10}})
	b.Chat(line("Sothr", "bjorn claim this bed"))
	if got := uses(); got[len(got)-1] != "1:5" {
		t.Errorf("used %v; want the free bed 1:5", got)
	}
	f.perception.Structures = f.perception.Structures[:4]
	b.Chat(line("Sothr", "bjorn claim this bed"))
	if got := f.said(); got[len(got)-1] != "That one's mine already." {
		t.Errorf("claimed its own bed: %q", got[len(got)-1])
	}

	// From the panel, with the owner out of sight: nearest the bot, at 29, 10.
	f.perception.Structures = f.perception.Structures[:3]
	f.state = &protocol.State{Phase: protocol.PhaseInWorld, Pos: protocol.Vec3{29, 0, 10}}
	if err := b.FromPanel("Sothr", "get the door"); err != nil {
		t.Fatal(err)
	}
	if got := uses(); got[len(got)-1] != "1:2" {
		t.Errorf("used %v; want the door nearest the bot, 1:2", got)
	}

	sent := slices.DeleteFunc(f.sent(), func(o session.Order) bool { return o.Task != protocol.TaskInteract })
	n := len(f.said())
	b.TaskDone(session.FinishedWith(protocol.TaskInteract, sent[0].Origin, protocol.StatusSucceeded, "", "used the Bed: Spawn point set"))
	b.TaskDone(session.FinishedWith(protocol.TaskInteract, sent[0].Origin, protocol.StatusFailed, protocol.ReasonRefused, "You need a roof"))
	b.TaskDone(session.FinishedWith(protocol.TaskInteract, sent[0].Origin, protocol.StatusSucceeded, "", "used the Door"))
	b.TaskDone(session.FinishedWith(protocol.TaskInteract, sent[0].Origin, protocol.StatusFailed, protocol.ReasonTargetLost, "the Bed is gone"))
	if got := f.said()[n:]; !slices.Equal(got, []string{"Spawn point set", "It won't: You need a roof", "Done.", "It's gone."}) {
		t.Errorf("said %q", got)
	}
}
