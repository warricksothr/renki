# The audit log

When the bot does something odd in the game, the audit log is how we find
out why. The controller records everything that passes through it and
everything it decides, in order, and the plugin sends the decisions it makes
on its own as `trace` messages (docs/BRIDGE-PROTOCOL.md 6.7). So one file
answers what was said, what the controller made of it, what it told the
plugin, what the plugin reported back, and what the plugin chose to do in
between.

## What is in it

The controller writes JSON Lines, one file per local day, to the `audit.dir`
in its configuration: `audit/` beside the controller by default, so
`C:\renki\audit\audit-2026-10-28.jsonl` on the VM. Files older than
`audit.keepDays` (14) are deleted. Set `"audit": {"dir": ""}` to turn it off.

Each line is one record, with `at` (the controller's clock) and `kind`:

| `kind` | What | Fields |
|---|---|---|
| `start` | The controller started | `msg` (version), `attrs.config` with the token and the terva block redacted |
| `frame` | A bridge frame, as it went over the wire | `conn`, `dir` (`in` from the plugin, `out` to it), `frame`; `bad` for one that was not JSON |
| `log` | A controller log record, at debug level and up, whatever `-debug` says | `level`, `msg`, `attrs`, `conn` when it belongs to a connection |

Every frame is there except `ping` and `pong`. That includes `state` at 5 Hz
and `perception` at 1 Hz, so the bot's position can be followed through any
moment. The token in `hello` is replaced with `[redacted]`. The `log` records
are the controller's decisions: `command` (a line it understood), refusals,
`task sent`, and connections opening and closing.

A day of play is tens of megabytes, almost all of it `state` and
`perception`.

## Reading it

`renki-audit` (`cmd/audit`) prints the records as a timeline:

    just vm audit                       # fetch today's file from the VM, print it all
    just vm audit 1 -last 10m           # the last ten minutes
    just vm audit 1 -task t-6           # one task: the chat and command behind it, the
                                        # request, its statuses and its traces
    just vm audit 1 -since 16:40 -until 16:45 -state 1s   # with the bot's state each second
    just vm audit 2 -json               # today's and yesterday's records, as JSON
    just controller audit -last 5m      # a controller run from the repository root

The first argument of `just vm audit` is how many days of files to fetch,
today counting as one. The files land in `scripts/vm/logs/audit/`, which git
ignores. `state` and `perception` are hidden unless `-state` or
`-perception` says how often to show them.

A stuck follow reads like this:

    16:42:10.000 [1] <- chat      Nimah (shout, 42 m): tas follow me
    16:42:10.001 [ ] INFO  command command=follow from=Nimah
    16:42:10.002 [1] -> task      t-6 follow {"player":"Nimah"}
    16:42:10.010 [1] <- task      t-6 running
    16:42:10.020 [1] <- trace     t-6 follow: walk distance=3.5 gap=12.4 leader=(…) pos=(…)
    16:42:13.000 [1] <- trace     t-6 walker: stuck; trying Jump attempt=1 center=(…) pos=(…)
    16:42:40.000 [1] <- task      t-6 failed (stuck): no progress after 9 recoveries

`<-` is from the plugin, `->` is to it, and `[1]` is the connection.

## What the plugin traces

| `source` | `msg` | When | `data` |
|---|---|---|---|
| `follow` | `stop`, `walk`, `run` | The gait changes | `gap`, `distance`, `pos`, `leader`, `leaderSpeed` |
| `follow` | `<player> out of sight`, `<player> is dead` | The leader is lost | — |
| `walker` | `path`, `no navmesh path; straight line` | A path is worked out | `from`, `to`, `waypoints` |
| `walker` | `stuck; trying Jump`, `Strafe`, `Repath`; `stuck; giving up` | Stuck recovery (7.3) | `attempt`, `pos`, `center`, `goal` |

The same lines go to the BepInEx log, so they can also be read there when
the controller is not running.

When something needs explaining that the log cannot answer, add a trace for
it rather than a one-off log line, so the next time it is already recorded.
