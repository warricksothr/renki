package voice

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"terva.sh/terva/packages/agent/sdk"

	"github.com/warricksothr/renki/internal/chat"
)

// call is a tool call the fake model makes.
type call struct {
	tool string
	args string
}

// fakeRuntime plays a model: it calls the tools it was given, then streams
// its text, as terva's loop would.
type fakeRuntime struct {
	cfg    sdk.Config
	calls  []call
	text   string
	fail   string
	usage  sdk.Usage
	prompt string
	errs   []bool
}

func (f *fakeRuntime) Prompt(_ context.Context, text string, _ []sdk.Image) (<-chan sdk.Event, error) {
	f.prompt = text
	out := make(chan sdk.Event, 8)
	go func() {
		defer close(out)
		for _, c := range f.calls {
			for _, t := range f.cfg.ExtraTools {
				if t.Name() == c.tool {
					res, _ := t.Execute(context.Background(), json.RawMessage(c.args), nil)
					f.errs = append(f.errs, res.IsError)
				}
			}
		}
		if f.fail != "" {
			out <- sdk.Event{Type: "error", Error: f.fail}
			return
		}
		for _, w := range strings.SplitAfter(f.text, " ") {
			out <- sdk.Event{Type: "text_delta", Delta: w}
		}
		out <- sdk.Event{Type: "turn_end", Stop: "end"}
	}()
	return out, nil
}

func (f *fakeRuntime) Cost() sdk.Usage { return f.usage }
func (f *fakeRuntime) Close() error    { return nil }

func newVoice(t *testing.T, f *fakeRuntime, daily int, now func() time.Time) *Voice {
	t.Helper()
	v, err := New(Options{
		DailyTokens: daily,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:         now,
		newRuntime: func(c sdk.Config) (runtime, error) {
			f.cfg = c
			return f, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

var owner = chat.Request{AgentName: "Bjorn", Speaker: "Sothr", Owner: true, Text: "bjorn, wave at me",
	Summary: "Health 47/60.", Recent: []chat.Line{{From: "Sothr", Text: "bjorn, wave at me"}},
	Tasks: []string{"stop", "follow", "say", "emote"}}

func TestReplyWithOneProposal(t *testing.T) {
	f := &fakeRuntime{text: "\"Aye,\nhello there!\"", calls: []call{
		{"emote", `{"name":"wave"}`},
		{"follow", `{"player":"Sothr"}`},
	}}
	v := newVoice(t, f, 0, nil)
	reply, err := v.Reply(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Say != "Aye, hello there!" {
		t.Errorf("say %q", reply.Say)
	}
	if reply.Task == nil || reply.Task.Task != "emote" || string(reply.Task.Args) != `{"name":"wave"}` {
		t.Errorf("proposal %+v; want the first call only", reply.Task)
	}
	if len(f.errs) != 2 || f.errs[0] || !f.errs[1] {
		t.Errorf("tool results %v; want the second call refused", f.errs)
	}
	if !f.cfg.NoTools || f.cfg.MaxSteps != 2 || !f.cfg.Yolo || !strings.Contains(f.cfg.SystemPrompt, "Bjorn") {
		t.Errorf("sdk config %+v", f.cfg)
	}
	if !strings.Contains(f.prompt, "Health 47/60.") || !strings.Contains(f.prompt, "Sothr: bjorn, wave at me") {
		t.Errorf("prompt lacks the summary or chat:\n%s", f.prompt)
	}
}

func TestMenuNeverOffersSayAndFollowsAllowed(t *testing.T) {
	f := &fakeRuntime{text: "Hi."}
	v := newVoice(t, f, 0, nil)
	if _, err := v.Reply(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range f.cfg.ExtraTools {
		names = append(names, tool.Name())
	}
	if strings.Join(names, ",") != "stop,follow,emote" {
		t.Errorf("tools %v", names)
	}
}

func TestNonOwnerGetsNoTools(t *testing.T) {
	f := &fakeRuntime{text: "Hello, stranger."}
	v := newVoice(t, f, 0, nil)
	req := owner
	req.Owner, req.Tasks = false, nil
	if _, err := v.Reply(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if len(f.cfg.ExtraTools) != 0 {
		t.Errorf("a non-owner's turn had tools %d", len(f.cfg.ExtraTools))
	}
	if !strings.Contains(f.prompt, "do not take orders from Sothr") {
		t.Errorf("prompt does not say so:\n%s", f.prompt)
	}
}

func TestProviderErrorIsAnError(t *testing.T) {
	f := &fakeRuntime{fail: "401 unauthorized"}
	v := newVoice(t, f, 0, nil)
	if _, err := v.Reply(context.Background(), owner); err == nil {
		t.Fatal("want an error")
	}
}

func TestDailyTokenCap(t *testing.T) {
	now := time.Date(2026, 10, 28, 23, 0, 0, 0, time.Local)
	f := &fakeRuntime{text: "Aye.", usage: sdk.Usage{Input: 800, Output: 300}}
	v := newVoice(t, f, 1000, func() time.Time { return now })
	if _, err := v.Reply(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Reply(context.Background(), owner); !errors.Is(err, ErrBudget) {
		t.Fatalf("second reply: %v; want ErrBudget", err)
	}
	now = now.Add(2 * time.Hour)
	if _, err := v.Reply(context.Background(), owner); err != nil {
		t.Fatalf("next day: %v", err)
	}
}

func TestMissingKeyEnvIsAnError(t *testing.T) {
	t.Setenv("RENKI_TEST_KEY", "")
	if _, err := New(Options{Terva: Config{APIKeyEnv: "RENKI_TEST_KEY"}}); err == nil {
		t.Fatal("want an error for an unset key variable")
	}
}
