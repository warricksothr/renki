package chat

import (
	"context"
	"testing"

	"github.com/warricksothr/renki/internal/protocol"
)

func TestBaseCommandsRequireOwnerAndForwardConfiguredScope(t *testing.T) {
	b, _, _ := newBrain(nil)
	calls := []string{}
	b.opts.BaseCommand = func(_ context.Context, by, text string, _ *protocol.Vec3) (string, bool, error) {
		calls = append(calls, by+":"+text)
		return "recorded", true, nil
	}
	b.Chat(line("Mallory", "Bjorn familiarize yourself with home"))
	if len(calls) != 0 {
		t.Fatal("visitor started survey")
	}
	for _, command := range []string{"familiarize yourself with home", "familiarize yourself with this area", "base radius 80", "base sleeping bedroom", "base storage warehouse", "base return home", "survey status"} {
		b.Chat(line("Sothr", "Bjorn "+command))
	}
	if len(calls) != 7 || calls[2] != "Sothr:base radius 80" {
		t.Fatal(calls)
	}
}
