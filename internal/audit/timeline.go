package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/protocol"
)

// ReadFiles reads the records of the named files and of the audit files in
// the named directories, oldest first. A line that is not a record is
// skipped and counted.
func ReadFiles(paths []string) (recs []Record, skipped int, err error) {
	var names []string
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			return nil, 0, err
		}
		if st.IsDir() {
			names = append(names, Files(p)...)
		} else {
			names = append(names, p)
		}
	}
	for _, name := range names {
		f, err := os.Open(name)
		if err != nil {
			return nil, 0, err
		}
		n, s, err := readRecords(f)
		f.Close()
		if err != nil {
			return nil, 0, fmt.Errorf("%s: %w", filepath.Base(name), err)
		}
		recs = append(recs, n...)
		skipped += s
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].At.Before(recs[j].At) })
	return recs, skipped, nil
}

func readRecords(r io.Reader) (recs []Record, skipped int, err error) {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 64<<10), 4<<20)
	for s.Scan() {
		var rec Record
		if json.Unmarshal(s.Bytes(), &rec) != nil || rec.Kind == "" {
			skipped++
			continue
		}
		recs = append(recs, rec)
	}
	return recs, skipped, s.Err()
}

// Filter chooses the records a timeline shows.
type Filter struct {
	// Since and Until bound the time; zero means unbounded.
	Since, Until time.Time
	// Task keeps what concerns one task: its request and statuses, its
	// traces, the controller's records naming it, and the chat and
	// commands in the seconds before it was sent.
	Task string
	// State shows state frames at most this often; 0 hides them.
	State time.Duration
	// Perception shows perception frames at most this often; 0 hides them.
	Perception time.Duration
}

// lead is how far before a task's request its cause is looked for.
const lead = 3 * time.Second

// Apply returns the records f keeps, in order.
func (f Filter) Apply(recs []Record) []Record {
	var from, to time.Time
	if f.Task != "" {
		for _, r := range recs {
			if names(r, f.Task) {
				if from.IsZero() {
					from = r.At
				}
				to = r.At
			}
		}
		if from.IsZero() {
			return nil
		}
		from = from.Add(-lead)
	}
	var out []Record
	var lastState, lastPerception time.Time
	for _, r := range recs {
		if !f.Since.IsZero() && r.At.Before(f.Since) || !f.Until.IsZero() && r.At.After(f.Until) {
			continue
		}
		typ := frameType(r)
		switch typ {
		case protocol.TypeState:
			if f.State <= 0 || r.At.Sub(lastState) < f.State {
				continue
			}
			lastState = r.At
		case protocol.TypePerception:
			if f.Perception <= 0 || r.At.Sub(lastPerception) < f.Perception {
				continue
			}
			lastPerception = r.At
		}
		if f.Task != "" {
			if r.At.Before(from) || r.At.After(to) {
				continue
			}
			cause := typ == protocol.TypeChat || r.Kind == KindLog && r.Msg == "command"
			body := typ == protocol.TypeState || typ == protocol.TypePerception
			if !names(r, f.Task) && !cause && !body {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

// names reports whether r is about the task id.
func names(r Record, id string) bool {
	switch r.Kind {
	case KindFrame:
		var env struct {
			ID   string `json:"id"`
			Re   string `json:"re"`
			Task any    `json:"task"`
		}
		if json.Unmarshal(r.Frame, &env) != nil {
			return false
		}
		if env.ID == id || env.Re == id {
			return true
		}
		switch t := env.Task.(type) {
		case string: // a trace's task
			return t == id && frameType(r) == protocol.TypeTrace
		case map[string]any: // a state's running task
			return t["id"] == id
		}
	case KindLog:
		return r.Attrs["id"] == id || r.Attrs["task"] == id
	}
	return false
}

func frameType(r Record) string {
	if r.Kind != KindFrame || r.Frame == nil {
		return ""
	}
	var env struct {
		Type string `json:"type"`
	}
	json.Unmarshal(r.Frame, &env)
	return env.Type
}

// Line renders one record as a line of the timeline.
func Line(r Record) string {
	at := r.At.Local().Format("15:04:05.000")
	switch r.Kind {
	case KindStart:
		return fmt.Sprintf("%s ===== %s", at, r.Msg)
	case KindLog:
		return fmt.Sprintf("%s %s %-5s %s%s", at, conn(r.Conn), r.Level, r.Msg, attrs(r.Attrs))
	case KindFrame:
		arrow := "<-"
		if r.Dir == Out {
			arrow = "->"
		}
		if r.Frame == nil {
			return fmt.Sprintf("%s %s %s not JSON: %q", at, conn(r.Conn), arrow, r.Bad)
		}
		return fmt.Sprintf("%s %s %s %s", at, conn(r.Conn), arrow, frame(r))
	}
	return fmt.Sprintf("%s %s %s", at, r.Kind, r.Msg)
}

func conn(c uint64) string {
	if c == 0 {
		return "[ ]"
	}
	return fmt.Sprintf("[%d]", c)
}

// frame describes a frame in a few words, falling back to its JSON.
func frame(r Record) string {
	from := protocol.FromPlugin
	if r.Dir == Out {
		from = protocol.FromController
	}
	m, err := protocol.Decode(r.Frame, from)
	if err != nil {
		return fmt.Sprintf("%s (%v)", r.Frame, err)
	}
	switch m := m.(type) {
	case *protocol.Hello:
		return fmt.Sprintf("hello     %s, phase %s, world %q, plugin %s, game %s", or(m.Character, "no character"),
			m.Phase, m.World, m.Plugin, m.Game)
	case *protocol.Welcome:
		return fmt.Sprintf("welcome   name %q, owners %v, allowed %v", m.AgentName, m.Owners, m.AllowedTasks)
	case *protocol.State:
		s := fmt.Sprintf("state     %s at %s, speed %.1f, hp %.0f/%.0f", m.Phase, vec(m.Pos), speed(m.Vel), m.HP, m.HPMax)
		if len(m.Flags) > 0 {
			s += ", " + strings.Join(m.Flags, " ")
		}
		if m.Task != nil {
			s += fmt.Sprintf(", task %s %s %s", m.Task.ID, m.Task.Task, m.Task.Status)
		}
		return s
	case *protocol.Perception:
		var players []string
		for _, c := range m.Characters {
			if c.Kind == "player" {
				players = append(players, fmt.Sprintf("%s at %s", c.Name, vec(c.Pos)))
			}
		}
		var after []string
		for _, c := range m.Characters {
			if c.TargetingMe {
				after = append(after, fmt.Sprintf("%s at %s", or(c.Prefab, c.Name), vec(c.Pos)))
			}
		}
		line := fmt.Sprintf("perception %d characters, %d structures, %d resources; players: %s",
			len(m.Characters), len(m.Structures), len(m.Resources), or(strings.Join(players, ", "), "none"))
		if len(after) > 0 {
			line += "; after the bot: " + strings.Join(after, ", ")
		}
		return line
	case *protocol.Inventory:
		var held []string
		for _, it := range m.Items {
			s := fmt.Sprintf("%d %s", it.Stack, it.Prefab)
			if it.Equipped {
				s += " (equipped)"
			}
			held = append(held, s)
		}
		return fmt.Sprintf("inventory %d free of %d; %s", m.Free(), m.Size[0]*m.Size[1], or(strings.Join(held, ", "), "empty"))
	case *protocol.Chat:
		return fmt.Sprintf("chat      %s (%s, %.0f m): %s", m.From.Name, m.Channel, m.Dist, m.Text)
	case *protocol.Event:
		return "event     " + m.Event + compactJSON(r.Frame, "v", "type", "seq", "t", "event")
	case *protocol.TaskRequest:
		return fmt.Sprintf("task      %s %s %s", m.ID, m.Task, or(string(m.Args), "{}"))
	case *protocol.TaskStatus:
		s := fmt.Sprintf("task      %s %s", m.Re, m.Status)
		if m.Reason != "" {
			s += " (" + m.Reason + ")"
		}
		if m.Detail != "" {
			s += ": " + m.Detail
		}
		return s
	case *protocol.Trace:
		task := ""
		if m.Task != "" {
			task = m.Task + " "
		}
		return fmt.Sprintf("trace     %s%s: %s%s", task, m.Source, m.Msg, attrs(m.Data))
	}
	return m.MessageType() + compactJSON(r.Frame, "v", "type", "seq", "t")
}

// compactJSON is frame without the named keys, or "" when nothing is left.
func compactJSON(frame json.RawMessage, drop ...string) string {
	var m map[string]any
	if json.Unmarshal(frame, &m) != nil {
		return " " + string(frame)
	}
	for _, k := range drop {
		delete(m, k)
	}
	if len(m) == 0 {
		return ""
	}
	return attrs(m)
}

// attrs renders a map as " key=value ...", keys sorted.
func attrs(m map[string]any) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(" " + k + "=" + value(m[k]))
	}
	return b.String()
}

func value(v any) string {
	switch x := v.(type) {
	case string:
		if strings.ContainsAny(x, " =\"") || x == "" {
			return fmt.Sprintf("%q", x)
		}
		return x
	case float64:
		return trimFloat(x)
	case []any:
		if p, ok := point(x); ok {
			return p
		}
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// point renders a three-number array as a position.
func point(x []any) (string, bool) {
	if len(x) != 3 {
		return "", false
	}
	var v protocol.Vec3
	for i, e := range x {
		f, ok := e.(float64)
		if !ok {
			return "", false
		}
		v[i] = f
	}
	return vec(v), true
}

func vec(v protocol.Vec3) string {
	return fmt.Sprintf("(%s, %s, %s)", trimFloat(v[0]), trimFloat(v[1]), trimFloat(v[2]))
}

func speed(v protocol.Vec3) float64 { return math.Hypot(v[0], v[2]) }

func trimFloat(f float64) string {
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.2f", f), "0"), ".")
}

func or(s, alt string) string {
	if s == "" {
		return alt
	}
	return s
}
