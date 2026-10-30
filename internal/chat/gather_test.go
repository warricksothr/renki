package chat

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestGatherChatAuthorizesBeforeStartingAndExplainsRefusals(t *testing.T) {
	for _, mode := range []string{"owner", "stranger", "no axe", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			b, f, _ := newBrain(nil)
			calls := 0
			b.opts.GatherWood = func(_ context.Context, by string) (string, error) {
				calls++
				if by != "Sothr" {
					t.Fatal(by)
				}
				if mode == "no axe" {
					return "", errors.New("gather needs a usable carried axe")
				}
				return "One bounded tree, delivery and return.", nil
			}
			if mode == "unavailable" {
				b.opts.GatherWood = nil
			}
			by := "Sothr"
			if mode == "stranger" {
				by = "Someone"
			}
			b.Chat(line(by, "Bjorn go chop wood"))
			want := 1
			if mode == "stranger" || mode == "unavailable" {
				want = 0
			}
			if calls != want || len(f.tasks()) != 0 {
				t.Fatal(calls, f.tasks())
			}
			words := strings.Join(f.said(), " ")
			if words == "" || mode == "no axe" && !strings.Contains(words, "axe") || mode == "stranger" && !strings.Contains(words, "only take orders") {
				t.Fatal(words)
			}
		})
	}
}
