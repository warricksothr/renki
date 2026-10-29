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
	mu      sync.Mutex
	orders  []session.Order
	allowed []string
	state   *protocol.State
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

func (f *fakeSession) State() *protocol.State { return f.state }
func (f *fakeSession) AllowedTasks() []string { return f.allowed }

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
	world     string
	home      *protocol.Vec3
	homeward  map[string]bool
	trips     []string
	touched   int
	respawned int
	died      int
	ordered   int
	resume    func(context.Context) bool
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
func (h *fakeHome) Touch()                                        { h.touched++ }
func (h *fakeHome) Died()                                         { h.died++ }
func (h *fakeHome) Respawned()                                    { h.respawned++ }
func (h *fakeHome) ResumeAfterDeath(f func(context.Context) bool) { h.resume = f }
func (h *fakeHome) Ordered()                                      { h.resume = nil; h.ordered++ }

func homeBrain() (*Brain, *fakeSession, *fakeHome) {
	b, f, _ := newBrain(nil)
	h := &fakeHome{world: "Midgard", homeward: map[string]bool{}}
	b.opts.Home = h
	return b, f, h
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
	b.Event(&protocol.Event{Event: protocol.EventDied})
	b.Chat(line("Sothr", "bjorn wait for me where you wake up and then tag along"))
	<-v.got
	waitSaid(t, f, 1)
	if h.died != 1 || h.ordered != 1 || h.touched != 1 {
		t.Errorf("died %d, ordered %d, touched %d; want 1 each", h.died, h.ordered, h.touched)
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
