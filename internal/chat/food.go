package chat

import (
	"context"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func (b *Brain) foodCommand(text string, c *protocol.Chat, from origin, owner bool) bool {
	if text != "resupply food" && text != "stock up on food" {
		return false
	}
	if !owner {
		b.say(from, "I only take food resupply orders from configured owners and operators.")
		return true
	}
	if b.opts.ResupplyFood == nil {
		b.say(from, "Food resupply is unavailable.")
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	reply, err := b.opts.ResupplyFood(ctx, c.From.Name)
	if err != nil {
		b.say(from, "I can't resupply food: "+err.Error())
		return true
	}
	if b.opts.Home != nil {
		b.opts.Home.Touch()
		b.opts.Home.Ordered()
		b.opts.Home.Told()
	}
	b.say(from, reply)
	return true
}
