package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/warricksothr/renki/internal/home"
	"github.com/warricksothr/renki/internal/protocol"
	"github.com/warricksothr/renki/internal/session"
)

func (b *Brain) waypointCommand(text string, c *protocol.Chat, from origin, owner bool) bool {
	action, name, renamed := "", "", ""
	switch {
	case text == "waypoints" || text == "list waypoints":
		action = "list"
	case strings.HasPrefix(text, "mark this as "):
		action, name = "mark", strings.TrimPrefix(text, "mark this as ")
	case strings.HasPrefix(text, "set waypoint "):
		action, name = "mark", strings.TrimPrefix(text, "set waypoint ")
	case strings.HasPrefix(text, "move ") && strings.HasSuffix(text, " here"):
		action, name = "mark", strings.TrimSuffix(strings.TrimPrefix(text, "move "), " here")
	case strings.HasPrefix(text, "rename "):
		name, renamed, _ = strings.Cut(strings.TrimPrefix(text, "rename "), " to ")
		action = "rename"
	case strings.HasPrefix(text, "forget "):
		action, name = "forget", strings.TrimPrefix(text, "forget ")
	case strings.HasPrefix(text, "go to "):
		action, name = "go", strings.TrimPrefix(text, "go to ")
	default:
		return false
	}
	if !owner {
		b.say(from, fmt.Sprintf("Sorry, %s, I only take orders from %s.", c.From.Name, joinNames(b.live().Owners)))
		return true
	}
	if b.opts.Places == nil {
		b.say(from, "I can't remember waypoints yet.")
		return true
	}
	st := b.d.State()
	if st == nil || st.World == "" || st.Phase != protocol.PhaseInWorld {
		b.say(from, "I need to be in the world to use its waypoints.")
		return true
	}
	if action == "list" {
		places := b.opts.Places.Waypoints(st.World)
		names := make([]string, 0, len(places))
		for _, w := range places {
			names = append(names, w.Name)
		}
		if len(names) == 0 {
			b.say(from, "I haven't marked any waypoints here yet.")
		} else {
			b.say(from, "Waypoints: "+strings.Join(names, ", ")+".")
		}
		return true
	}
	name = strings.TrimPrefix(name, "waypoint ")
	canonical, err := home.WaypointName(name)
	if err != nil {
		b.say(from, err.Error())
		return true
	}
	name = canonical
	old, exists := b.opts.Places.Waypoint(st.World, name)
	var expected = &old.At
	if !exists {
		expected = nil
	}
	switch action {
	case "mark":
		if !located(c) {
			b.say(from, "I can't see you, so I don't know where to mark it.")
			return true
		}
		next := old
		next.Name, next.Pos, next.SetBy, next.At = name, c.Pos, c.From.Name, b.opts.Now()
		original := ""
		if exists {
			original = name
		}
		err = b.opts.Places.EditWaypoint(st.World, original, &next, expected)
	case "rename":
		if !exists {
			err = home.ErrNoWaypoint
			break
		}
		next := old
		next.Name, next.SetBy, next.At = renamed, c.From.Name, b.opts.Now()
		err = b.opts.Places.EditWaypoint(st.World, name, &next, expected)
	case "forget":
		err = b.opts.Places.EditWaypoint(st.World, name, nil, expected)
	case "go":
		if !exists {
			err = home.ErrNoWaypoint
			break
		}
		if b.opts.Home != nil {
			b.opts.Home.Touch()
			b.opts.Home.Ordered()
		}
		_, err = b.d.Dispatch(context.Background(), session.Order{Task: protocol.TaskMoveTo, Args: &protocol.MoveToArgs{To: protocol.Target{Entity: name}}, Origin: from, TimeoutS: 90})
	}
	if err != nil {
		b.say(from, "I couldn't do that: "+err.Error())
		return true
	}
	if b.opts.Home != nil {
		b.opts.Home.Touch()
		b.opts.Home.Told()
	}
	switch action {
	case "mark":
		b.say(from, "Marked "+name+" here.")
	case "rename":
		b.say(from, "Renamed "+name+" to "+renamed+".")
	case "forget":
		b.say(from, "Forgot "+name+".")
	case "go":
		b.say(from, "Heading to "+name+".")
	}
	return true
}
