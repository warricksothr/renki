package chat

import (
	"context"
	"testing"
)

func TestFoodCommandRequiresCurrentCommanderBeforeQueueing(t *testing.T) {
	for _, by := range []string{"Sothr", "Visitor"} {
		t.Run(by, func(t *testing.T) {
			b, f, _ := newBrain(nil)
			called := false
			b.opts.ResupplyFood = func(_ context.Context, player string) (string, error) {
				called = true
				if player != "Sothr" {
					t.Fatal(player)
				}
				return "Checking food.", nil
			}
			b.Chat(line(by, "Bjorn resupply food"))
			if called != (by == "Sothr") || len(f.tasks()) != 0 {
				t.Fatal(called, f.tasks())
			}
		})
	}
}
