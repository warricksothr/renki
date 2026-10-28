package chat

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/warricksothr/renki/internal/protocol"
)

// status describes the bot from its newest state in one or two sentences,
// for the `status` command and as the Voice's summary.
func (b *Brain) status() string {
	st := b.d.State()
	if st == nil {
		return "I'm not in the world yet."
	}
	if st.Phase != protocol.PhaseInWorld {
		return fmt.Sprintf("I'm not in the world right now (%s).", strings.ReplaceAll(st.Phase, "_", " "))
	}
	var s strings.Builder
	fmt.Fprintf(&s, "Health %.0f/%.0f, stamina %.0f/%.0f.", st.HP, st.HPMax, st.Stamina, st.StaminaMax)
	where := []string{}
	if st.Biome != "" {
		where = append(where, words(st.Biome))
	}
	if st.Env != "" {
		where = append(where, strings.ToLower(words(st.Env)))
	}
	where = append(where, fmt.Sprintf("day %d, %s", st.Time.Day, partOfDay(st.Time.DayFraction, st.Time.Night)))
	fmt.Fprintf(&s, " %s.", strings.Join(where, ", "))
	if st.Task != nil && st.Task.Status == protocol.StatusRunning {
		fmt.Fprintf(&s, " %s.", doing(st.Task.Task))
	} else {
		s.WriteString(" Idle.")
	}
	if b.opts.Home != nil {
		if d, ok := b.opts.Home.HomeDistance(); ok {
			fmt.Fprintf(&s, " Home is %.0f m away.", d)
		} else {
			s.WriteString(" No home here yet.")
		}
	}
	return s.String()
}

var camel = regexp.MustCompile(`([a-z])([A-Z])`)

// words splits a game identifier like BlackForest into "Black Forest".
func words(id string) string { return camel.ReplaceAllString(id, "$1 $2") }

func partOfDay(f float64, night bool) string {
	switch {
	case night:
		return "night"
	case f < 0.35:
		return "morning"
	case f < 0.6:
		return "midday"
	default:
		return "evening"
	}
}

func doing(task string) string {
	switch task {
	case protocol.TaskFollow:
		return "Following"
	case protocol.TaskMoveTo:
		return "On the move"
	}
	return "Busy with " + strings.ReplaceAll(task, "_", " ")
}
