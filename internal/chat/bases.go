package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

func (b *Brain) baseCommand(text string, c *protocol.Chat, from origin, owner bool) bool {
	known := text == "survey status" || text == "survey home" || text == "familiarize yourself with home" || text == "familiarize yourself with this area" || text == "familiarize yourself with the area" || strings.HasPrefix(text, "base radius ") || strings.HasPrefix(text, "base sleeping ") || strings.HasPrefix(text, "base storage ") || strings.HasPrefix(text, "base return ")
	if !known {
		return false
	}
	if !owner {
		b.say(from, fmt.Sprintf("Sorry, %s, I only take orders from %s.", c.From.Name, joinNames(b.live().Owners)))
		return true
	}
	if b.opts.BaseCommand == nil {
		b.say(from, "Base surveys are not available.")
		return true
	}
	var pos *protocol.Vec3
	if located(c) {
		p := c.Pos
		pos = &p
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	line, _, err := b.opts.BaseCommand(ctx, c.From.Name, text, pos)
	if err != nil {
		b.say(from, "I couldn't do that: "+err.Error())
		return true
	}
	if b.opts.Home != nil && text != "survey status" {
		b.opts.Home.Touch()
		b.opts.Home.Ordered()
		b.opts.Home.Told()
	}
	b.say(from, line)
	return true
}
