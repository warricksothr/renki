package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func (b *Brain) gatherCommand(text string, c *protocol.Chat, from origin, owner bool) bool {
	if text != "go chop wood" && text != "chop wood" && text != "gather wood" {
		return false
	}
	if !owner {
		b.say(from, fmt.Sprintf("Sorry, %s, I only take orders from %s.", c.From.Name, joinNames(b.live().Owners)))
		return true
	}
	if b.opts.GatherWood == nil {
		b.say(from, "Wood gathering isn't available yet.")
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	reply, err := b.opts.GatherWood(ctx, c.From.Name)
	if err != nil {
		b.say(from, "I can't gather wood: "+err.Error())
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
