package chat

import (
	"cmp"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/warricksothr/renki/internal/home"
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
	if inv := b.d.Inventory(); inv != nil {
		fmt.Fprintf(&s, " %s.", carrying(inv))
	}
	switch {
	case st.Stranded():
		fmt.Fprintf(&s, " Overloaded, %.0f/%.0f: I can't move until I carry less.", st.Weight, st.WeightMax)
	case st.Encumbered():
		fmt.Fprintf(&s, " Overloaded, %.0f/%.0f: I can only walk, and not for long.", st.Weight, st.WeightMax)
	case st.WeightMax > 0:
		fmt.Fprintf(&s, " Weight %.0f/%.0f.", st.Weight, st.WeightMax)
	}
	if b.opts.Home != nil {
		if d, ok := b.opts.Home.HomeDistance(); ok {
			fmt.Fprintf(&s, " Home is %.0f m away.", d)
		} else {
			s.WriteString(" No home here yet.")
		}
		if left := leftBehind(b.opts.Home.LeftBehind(), st.Pos); left != "" {
			fmt.Fprintf(&s, " %s.", left)
		}
	}
	return s.String()
}

// leftBehind names what the bot left on the ground nearest pos, and how
// many other places there are: "I left 6 Wood 40 m from here".
func leftBehind(caches []home.Cache, pos protocol.Vec3) string {
	if len(caches) == 0 {
		return ""
	}
	near := slices.MinFunc(caches, func(a, b home.Cache) int {
		return cmp.Compare(a.Pos.FlatDistance(pos), b.Pos.FlatDistance(pos))
	})
	prefabs := slices.Sorted(maps.Keys(near.Items))
	slices.SortStableFunc(prefabs, func(a, b string) int { return cmp.Compare(near.Items[b], near.Items[a]) })
	var parts []string
	for _, p := range prefabs {
		parts = append(parts, fmt.Sprintf("%d %s", near.Items[p], words(p)))
	}
	s := fmt.Sprintf("I left %s %.0f m from here", strings.Join(parts, ", "), near.Pos.FlatDistance(pos))
	switch n := len(caches) - 1; n {
	case 0:
	case 1:
		s += ", and things in one other place"
	default:
		s += fmt.Sprintf(", and things in %d other places", n)
	}
	return s
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

// carrying sums what the bot carries by prefab, most first, at most five
// kinds, and how many slots are left.
func carrying(inv *protocol.Inventory) string {
	if len(inv.Items) == 0 {
		return "Carrying nothing"
	}
	type kind struct {
		prefab string
		n      int
	}
	var kinds []kind
	for _, it := range inv.Items {
		if !slices.ContainsFunc(kinds, func(k kind) bool { return k.prefab == it.Prefab }) {
			kinds = append(kinds, kind{it.Prefab, inv.Count(it.Prefab)})
		}
	}
	slices.SortStableFunc(kinds, func(a, b kind) int { return cmp.Compare(b.n, a.n) })
	var parts []string
	for i, k := range kinds {
		if i == 5 {
			parts = append(parts, "and more")
			break
		}
		// An item with no prefab goes by its name token (6.3): $item_coins.
		name := strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(k.prefab, "$item_"), "$"), "_", " ")
		parts = append(parts, fmt.Sprintf("%d %s", k.n, words(name)))
	}
	free := inv.Free()
	slots := "slots"
	if free == 1 {
		slots = "slot"
	}
	return fmt.Sprintf("Carrying %s; %d %s free", strings.Join(parts, ", "), free, slots)
}
