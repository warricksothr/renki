package presence

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/access"
	"github.com/warricksothr/renki/internal/protocol"
)

func (m *Manager) queryTarget(text string, panel bool) (string, bool) {
	live := m.o.Live()
	text = strings.TrimSpace(text)
	if !panel {
		name := live.AgentName
		if name == "" {
			if st, _ := m.o.Session.StateObservation(); st != nil {
				name = st.Character
			}
		}
		addressed := false
		for _, alias := range append(append([]string(nil), live.Nicknames...), name) {
			if alias == "" || len(text) <= len(alias) || !strings.EqualFold(text[:len(alias)], alias) || !strings.ContainsRune(" ,:;.!", rune(text[len(alias)])) {
				continue
			}
			text = strings.TrimLeft(text[len(alias):], " ,:;.!")
			addressed = true
			break
		}
		if !addressed {
			return "", false
		}
	} else {
		// The panel already addresses Tas; also accept the usual named form.
		if target, ok := m.queryTarget(text, false); ok {
			return target, true
		}
	}
	for _, prefix := range []string{"where is ", "where's ", "where did you last see ", "when did you last see ", "where did you last hear from "} {
		if len(text) > len(prefix) && strings.EqualFold(text[:len(prefix)], prefix) {
			target := strings.Trim(strings.TrimSpace(strings.TrimRight(text[len(prefix):], "?!.")), "\"“”")
			if target != "" && len(target) <= 64 {
				return target, true
			}
		}
	}
	return "", false
}

// PanelQuery uses the same resolver but never fabricates the panel user's
// position. The Brain logs its reply to the panel instead of speaking in game.
func (m *Manager) PanelQuery(speaker, text string) (string, bool) {
	target, ok := m.queryTarget(text, true)
	if !ok || m.o.Players == nil {
		return "", false
	}
	live := m.o.Live()
	if !live.Players.Has(live.Owners, speaker, access.Locate) {
		return "You don't have permission to ask for player locations.", true
	}
	scope, valid := m.scope()
	if !valid {
		return "I need a fresh in-world position to give directions.", true
	}
	st, _ := m.o.Session.StateObservation()
	if st == nil || st.World != scope.world {
		return "I need a fresh in-world position to give directions.", true
	}
	return m.locationReply(speaker, target, scope.hello, scope.world, st.Pos, m.o.Now()), true
}

func (m *Manager) locationReply(speaker, target string, hello *protocol.Hello, world string, bot protocol.Vec3, now time.Time) string {
	if m.o.Players == nil {
		return "I don't have player location memory."
	}
	o, ok := m.o.Players.Last(world, target, now)
	if !ok {
		return fmt.Sprintf("I have no remembered location for %s in this world.", target)
	}
	from, reference := bot, "from me"
	if pos, fresh := m.o.Players.Reference(world, speaker, hello, now); fresh && pos.FlatDistance(bot) > 6 {
		from, reference = pos, "from your last known position (at most 30 seconds old)"
	} else if !fresh {
		reference = "from me; I don't have a recent position for you"
	}
	dx, dz := o.Pos[0]-from[0], o.Pos[2]-from[2]
	distance := math.Hypot(dx, dz)
	bearing := []string{"north", "north-east", "east", "south-east", "south", "south-west", "west", "north-west"}[int(math.Round(math.Atan2(dx, dz)/(math.Pi/4))+8)%8]
	verb := "saw"
	if o.Source == "heard" {
		verb = "heard from"
	}
	age := now.Sub(o.At)
	ago := "just now"
	switch {
	case age >= 48*time.Hour:
		ago = fmt.Sprintf("about %d days ago", int(age.Hours()/24))
	case age >= 2*time.Hour:
		ago = fmt.Sprintf("about %d hours ago", int(age.Hours()))
	case age >= 2*time.Minute:
		ago = fmt.Sprintf("about %d minutes ago", int(age.Minutes()))
	case age >= time.Minute:
		ago = "about a minute ago"
	case age >= 10*time.Second:
		ago = "less than a minute ago"
	}
	uncertainty := math.Ceil(o.ResolutionM / math.Sqrt2)
	step := math.Max(10, o.ResolutionM)
	where := fmt.Sprintf("about %.0f m %s", math.Round(distance/step)*step, bearing)
	if distance <= uncertainty+5 {
		where = "nearby"
	}
	return fmt.Sprintf("I last %s %s %s. That location was %s %s, straight-line (remembered area within about %.0f m).", verb, o.Name, ago, where, reference, uncertainty)
}
