package chat

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

type ChestSearchRequest struct {
	Item   string
	Radius float64
	Chests []string
}

var chestRadius = regexp.MustCompile(` within ([0-9]+)(?: ?m| meters?| metres?)$`)
var chestList = regexp.MustCompile(`^(?:search|check) chests? ([0-9]+:[0-9]+(?:[ ,]+[0-9]+:[0-9]+)*) for ([a-z0-9 _-]+)$`)
var chestNearby = regexp.MustCompile(`^(?:search|check)(?: all)?(?: nearby)? chests? for ([a-z0-9 _-]+)$`)
var chestLook = regexp.MustCompile(`^(?:look|search) for ([a-z0-9 _-]+) in (?:a |the |all )?(?:nearby )?chests?$`)

func parseChestSearch(raw string) (ChestSearchRequest, bool, error) {
	text := strings.Trim(strings.ToLower(strings.TrimSpace(raw)), " ,:;.!?")
	text = strings.Join(strings.Fields(strings.ReplaceAll(text, "please ", "")), " ")
	p := ChestSearchRequest{Radius: 25}
	if m := chestRadius.FindStringSubmatch(text); m != nil {
		p.Radius, _ = strconv.ParseFloat(m[1], 64)
		text = strings.TrimSuffix(text, m[0])
	}
	if m := chestList.FindStringSubmatch(text); m != nil {
		p.Item = m[2]
		p.Chests = strings.Fields(strings.ReplaceAll(m[1], ",", " "))
	} else if m := chestNearby.FindStringSubmatch(text); m != nil {
		p.Item = m[1]
	} else if m := chestLook.FindStringSubmatch(text); m != nil {
		p.Item = m[1]
	} else {
		return p, false, nil
	}
	if p.Radius < 10 || p.Radius > 100 {
		return p, true, errors.New("use a search radius from 10 to 100 meters")
	}
	return p, true, nil
}

func (b *Brain) chestSearchCommand(c *protocol.Chat, from origin, owner bool) bool {
	// Keep colons in explicit chest IDs; the ordinary chat grammar removes them.
	raw := b.nameRE().ReplaceAllString(c.Text, " ")
	p, handled, err := parseChestSearch(raw)
	if !handled {
		return false
	}
	if !owner {
		b.say(from, fmt.Sprintf("Sorry, %s, I only take orders from %s.", c.From.Name, joinNames(b.live().Owners)))
		return true
	}
	if err != nil {
		b.say(from, err.Error())
		return true
	}
	if b.opts.SearchChests == nil {
		b.say(from, "Chest search isn't available.")
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	line, err := b.opts.SearchChests(ctx, c.From.Name, p)
	if err != nil {
		b.say(from, "I can't search chests: "+err.Error())
		return true
	}
	if b.opts.Home != nil {
		b.opts.Home.Touch()
		b.opts.Home.Ordered()
		b.opts.Home.Told()
	}
	b.say(from, line)
	return true
}
