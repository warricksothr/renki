// Command audit prints the controller's audit log (internal/audit) as a
// timeline: what was said, what the controller decided and sent, what the
// plugin reported and what it decided on its own.
//
//	audit [flags] [file or directory ...]
//
// With no arguments it reads ./audit. `just audit` runs it on the dev box,
// and `just vm audit` fetches the VM's files first.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/warricksothr/renki/internal/audit"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: audit [flags] [file or directory ...]  (default: ./audit)")
		fs.PrintDefaults()
	}
	task := fs.String("task", "", "only what concerns this task id (t-6), and the chat that led to it")
	since := fs.String("since", "", "from this time: 16:42, 16:42:30, 2026-10-28T16:42 or RFC 3339")
	until := fs.String("until", "", "up to this time, in the same forms as -since")
	last := fs.Duration("last", 0, "only the last stretch of the log, such as 10m")
	state := fs.Duration("state", 0, "show the bot's state at most this often, such as 1s; 0 hides it")
	perception := fs.Duration("perception", 0, "show perception at most this often; 0 hides it")
	asJSON := fs.Bool("json", false, "print the chosen records as JSON Lines instead of a timeline")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"audit"}
	}

	recs, skipped, err := audit.ReadFiles(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, "audit:", err)
		return 1
	}
	if len(recs) == 0 {
		fmt.Fprintln(os.Stderr, "audit: no records in", strings.Join(paths, ", "))
		return 1
	}
	f := audit.Filter{Task: *task, State: *state, Perception: *perception}
	day := recs[len(recs)-1].At.Local()
	if f.Since, err = parseTime(*since, day); err != nil {
		fmt.Fprintln(os.Stderr, "audit: -since:", err)
		return 2
	}
	if f.Until, err = parseTime(*until, day); err != nil {
		fmt.Fprintln(os.Stderr, "audit: -until:", err)
		return 2
	}
	if *last > 0 {
		if from := recs[len(recs)-1].At.Add(-*last); from.After(f.Since) {
			f.Since = from
		}
	}

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	shown := f.Apply(recs)
	for i, r := range shown {
		if *asJSON {
			enc.Encode(r)
			continue
		}
		if i == 0 || !sameDay(shown[i-1].At, r.At) {
			fmt.Fprintf(out, "----- %s\n", r.At.Local().Format("Monday 2006-01-02"))
		}
		fmt.Fprintln(out, audit.Line(r))
	}
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "audit: skipped %d lines that were not records\n", skipped)
	}
	if len(shown) == 0 {
		fmt.Fprintln(os.Stderr, "audit: nothing matched")
		return 1
	}
	return 0
}

// parseTime reads a time of day on day, or a full date and time.
func parseTime(s string, day time.Time) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{"15:04", "15:04:05", "15:04:05.000"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			y, m, d := day.Date()
			return time.Date(y, m, d, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local), nil
		}
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Parse(time.RFC3339Nano, s)
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Local().Date()
	by, bm, bd := b.Local().Date()
	return ay == by && am == bm && ad == bd
}
