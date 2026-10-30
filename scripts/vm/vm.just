# Driving the bot VM over SSH, loaded by the root justfile as the `vm`
# module: `just vm deploy`. Not a component, so the root verbs do not route
# here. scripts/vm/setup-remote.ps1 prepares the VM; docs/VM.md walks it.
#
# The VM is RENKI_VM (user@host), or the first line of vm.local at the
# repository root (gitignored). RENKI_VM_DIR is where renki lives there.

set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

target := env("RENKI_VM", if path_exists(justfile_directory() / "vm.local") == "true" { trim(read(justfile_directory() / "vm.local")) } else { "" })
dir := env("RENKI_VM_DIR", "C:/renki")
ssh := "ssh -o BatchMode=yes -o ConnectTimeout=10 " + target
ps := "powershell -NoProfile -ExecutionPolicy Bypass"

# List the VM recipes.
default:
    @just --list vm

[private]
need-target:
    @[ -n "{{target}}" ] || { echo "no VM: set RENKI_VM=user@host or write it to vm.local" >&2; exit 1; }

# Check the connection, the account and the scheduled tasks.
check: need-target
    {{ssh}} "whoami; hostname; Get-ScheduledTask -TaskName Renki* | Format-Table TaskName,State -AutoSize | Out-String -Width 120"

# Show the game, the controller and the bot; LINES > 0 adds the log tails.
status LINES="0": need-target
    {{ssh}} "{{ps}} -File {{dir}}/status.ps1 -Lines {{LINES}}"

# The tail of the plugin's and the controller's logs.
logs LINES="60": (status LINES)

# Keep the web panel available while the game is offline.
panel-start: need-target
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/start-bot.ps1" "{{justfile_directory()}}/scripts/vm/start-panel.ps1" "{{target}}:/{{dir}}/"
    {{ssh}} "{{ps}} -File {{dir}}/start-panel.ps1"

# Stop a controller that keeps the panel available while Valheim is offline.
panel-stop: need-target
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/stop-panel.ps1" "{{target}}:/{{dir}}/stop-panel.ps1"
    {{ssh}} "{{ps}} -File {{dir}}/stop-panel.ps1"

# Start the bot in the VM's desktop session and wait for it to be in the world.
start MINUTES="5": need-target
    #!/usr/bin/env bash
    set -euo pipefail
    {{ssh}} "Start-ScheduledTask -TaskName RenkiStart"
    echo "started; waiting up to {{MINUTES}} min for the bot to be in the world"
    deadline=$(( $(date +%s) + {{MINUTES}} * 60 ))
    while [ "$(date +%s)" -lt "$deadline" ]; do
        out="$({{ssh}} "{{ps}} -File {{dir}}/status.ps1" 2>&1 || true)"
        if grep -q '"phase":"in_world"' <<<"$out"; then echo "$out"; exit 0; fi
        sleep 10
    done
    echo "$out"
    echo "not in the world after {{MINUTES}} min; try: just vm logs" >&2
    exit 1

# Log the bot out, close the game and stop the controller.
stop: need-target
    #!/usr/bin/env bash
    set -euo pipefail
    {{ssh}} "Start-ScheduledTask -TaskName RenkiStop"
    for _ in $(seq 1 30); do
        if ! {{ssh}} "if (Get-Process valheim -ErrorAction SilentlyContinue) { exit 1 }" >/dev/null 2>&1; then
            sleep 5
            continue
        fi
        echo "stopped"
        exit 0
    done
    echo "the game is still running after 150 s" >&2
    exit 1

# Stop, then start.
restart: stop start

# Restart only the controller, refreshing welcome/allowedTasks without reloading Valheim.
reconnect: need-target
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/reconnect.ps1" "{{target}}:/{{dir}}/reconnect.ps1"
    {{ssh}} "{{ps}} -File {{dir}}/reconnect.ps1"

# Build, stop and install with checksum verification; leave Tas offline.
install: need-target
    #!/usr/bin/env bash
    set -euo pipefail
    just --justfile "{{justfile()}}" package
    if {{ssh}} "if (Get-Process valheim -ErrorAction SilentlyContinue) { exit 10 }" >/dev/null; then
        echo "the game is not running"
    else
        probe_status=$?
        if [ "$probe_status" -ne 10 ]; then
            echo "cannot inspect the VM (SSH exit $probe_status); installation stopped before copying or stopping anything" >&2
            exit "$probe_status"
        fi
        just --justfile "{{justfile()}}" vm stop
    fi
    # Controller-only panel sessions also lock controller.exe on Windows.
    just --justfile "{{justfile()}}" vm panel-stop
    {{ssh}} "New-Item -ItemType Directory -Force -Path '{{dir}}' | Out-Null"
    scp -o BatchMode=yes -r "{{justfile_directory()}}/dist/renki-vm/." "{{target}}:/{{dir}}/"
    {{ssh}} "{{ps}} -File {{dir}}/install.ps1"

# Install the package, then start and wait for Tas to join.
deploy: install
    just --justfile "{{justfile()}}" vm start

# Deploy a controller-only change and refresh the bridge without restarting Valheim.
deploy-controller: need-target
    #!/usr/bin/env bash
    set -euo pipefail
    just --justfile "{{justfile()}}" controller build-windows
    scp -q -o BatchMode=yes "{{justfile_directory()}}/dist/controller.exe" "{{target}}:/{{dir}}/controller.next.exe"
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/deploy-controller.ps1" "{{target}}:/{{dir}}/deploy-controller.ps1"
    {{ssh}} "{{ps}} -File {{dir}}/deploy-controller.ps1"

# Takes renki-audit flags after the day count: `just vm audit 1 -task t-6`,
# `just vm audit 2 -last 30m -state 1s`. DAYS counts today as 1.
[doc("Fetch the VM's audit log and print it as a timeline.")]
audit DAYS="1" *ARGS: need-target
    #!/usr/bin/env bash
    set -euo pipefail
    local="{{justfile_directory()}}/scripts/vm/logs/audit"
    mkdir -p "$local"
    files=$({{ssh}} "Get-ChildItem '{{dir}}/audit' -Filter 'audit-*.jsonl' -ErrorAction SilentlyContinue | Where-Object { \$_.LastWriteTime -ge (Get-Date).Date.AddDays(1 - {{DAYS}}) } | ForEach-Object { \$_.Name }" | tr -d '\r')
    [ -n "$files" ] || { echo "no audit files on the VM in the last {{DAYS}} day(s) ({{dir}}/audit)" >&2; exit 1; }
    for f in $files; do
        scp -q -o BatchMode=yes "{{target}}:/{{dir}}/audit/$f" "$local/$f"
    done
    echo "fetched: $files" >&2
    cd "{{justfile_directory()}}"
    just controller audit {{ARGS}} $(for f in $files; do printf '%s ' "$local/$f"; done)

# Save an exact audit window locally, including state/perception, without a full-day download.
[positional-arguments]
audit-window SINCE UNTIL: need-target
    #!/usr/bin/env bash
    set -euo pipefail
    [[ "$1" =~ ^[0-9T:.Z+-]+$ && "$2" =~ ^[0-9T:.Z+-]+$ ]] || { echo 'Use ISO timestamps' >&2; exit 2; }
    mkdir -p "{{justfile_directory()}}/.checks"
    out=$(mktemp "{{justfile_directory()}}/.checks/audit-window-XXXXXX.jsonl")
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/audit-window.ps1" "{{target}}:/{{dir}}/audit-window.ps1"
    {{ssh}} "{{ps}} -File {{dir}}/audit-window.ps1 -Since '$1' -Until '$2'" | tr -d '\r' > "$out"
    echo "$out"

# Print bounded task, routing and inventory evidence; full frames stay in .checks/.
[positional-arguments]
audit-summary SINCE UNTIL:
    #!/usr/bin/env bash
    set -euo pipefail
    saved=$(just --justfile "{{justfile()}}" vm audit-window "$1" "$2")
    python3 "{{justfile_directory()}}/scripts/vm/audit-summary.py" "$saved"

# Run once, and again after the Tailscale address changes. Then `just vm restart`.
[doc("Turn the control panel on: Tailscale address, firewall, token.")]
panel-setup PORT="8780": need-target
    {{ssh}} "{{ps}} -File {{dir}}/setup-panel.ps1 -Port {{PORT}}"

# Read only the panel address, without printing its login token.
panel-address: need-target
    @{{ssh}} "\$c = Get-Content '{{dir}}/controller.local.json' -Raw | ConvertFrom-Json; if (-not \$c.panel.listen) { 'The panel is off: run just vm panel-setup'; exit 1 }; 'http://' + \$c.panel.listen + '/'"

# Prints to this terminal only; it is the panel's login.
[doc("Print the control panel's address and login token.")]
panel-token: need-target
    @{{ssh}} "\$c = Get-Content '{{dir}}/controller.local.json' -Raw | ConvertFrom-Json; if (-not \$c.panel.listen) { 'The panel is off: run just vm panel-setup'; exit 1 }; 'address: http://' + \$c.panel.listen + '/'; 'token:   ' + \$c.panel.token"

# Call an API endpoint; credentials stay on the VM. The script is synced so
# the recipe also works before the next full deployment.
[positional-arguments]
panel METHOD PATH BODY='': need-target
    #!/usr/bin/env bash
    set -euo pipefail
    method=$1
    endpoint=$2
    [[ "$method" = GET || "$method" = POST ]] || { echo 'METHOD must be GET or POST' >&2; exit 2; }
    [[ "$endpoint" =~ ^/api/[a-zA-Z0-9_/?=\&.%:-]+$ ]] || { echo 'Use an /api/ path without shell metacharacters' >&2; exit 2; }
    encoded=$(printf '%s' "${3:-}" | base64 -w0)
    bodyarg=()
    [ -z "$encoded" ] || bodyarg=(-BodyBase64 "$encoded")
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/panel-request.ps1" "{{target}}:/{{dir}}/panel-request.ps1"
    {{ssh}} "{{ps}} -File {{dir}}/panel-request.ps1 -Method '$method' -ApiPath '$endpoint' ${bodyarg[*]}"

# Send an exact task for a reproducible live check; normal safety still applies.
[positional-arguments]
task TASK ARGS='{}' TIMEOUT='60':
    #!/usr/bin/env bash
    set -euo pipefail
    body=$(python3 -c 'import json,sys; print(json.dumps(dict(task=sys.argv[1],args=json.loads(sys.argv[2]),timeoutS=float(sys.argv[3]))))' "$1" "$2" "$3")
    just --justfile "{{justfile()}}" vm panel POST /api/task "$body"

# Send once, wait for the terminal outcome, retain full task evidence locally.
[positional-arguments]
task-check TASK ARGS='{}' TIMEOUT='60':
    python3 "{{justfile_directory()}}/scripts/vm/task-check.py" "$1" "$2" "$3"

# Start one automatic normal-controls encounter if a safe Greyling is available;
# record results and always log out/stop, restoring combat off afterward.
[positional-arguments]
combat-trial *ARGS:
    python3 -B "{{justfile_directory()}}/scripts/vm/combat-trial.py" "$@"

# Harvest one isolated full beech and newly observed nearby logs; always stop.
[positional-arguments]
harvest-trial *ARGS:
    python3 -B "{{justfile_directory()}}/scripts/vm/combat-trial.py" --harvest-tree "$@"

# Retry already fallen beech logs without felling another tree; always stop.
[positional-arguments]
harvest-logs X Y Z *ARGS:
    python3 -B "{{justfile_directory()}}/scripts/vm/combat-trial.py" --harvest-logs "$1" "$2" "$3" "${@:4}"

# Read only matching bounded harvest evidence from a retained audit.
[positional-arguments]
harvest-evidence AUDIT +TASK_IDS:
    python3 -B "{{justfile_directory()}}/scripts/vm/harvest-evidence.py" "$@"

test-harvest-trial:
    python3 -B "{{justfile_directory()}}/scripts/vm/harvest-trial-test.py"

# Read a retained audit without starting the game; print counts and six examples.
[positional-arguments]
combat-evidence AUDIT TASK_ID:
    python3 -B "{{justfile_directory()}}/scripts/vm/combat-evidence.py" "$1" "$2"

test-combat-trial:
    python3 -B "{{justfile_directory()}}/scripts/vm/combat-trial-test.py"

# Read AllowCombat, or set true/false before a staged encounter and restart.
[positional-arguments]
combat VALUE='': need-target
    #!/usr/bin/env bash
    set -euo pipefail
    [[ -z "$1" || "$1" = true || "$1" = false ]] || { echo 'Use true or false' >&2; exit 2; }
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/combat.ps1" "{{target}}:/{{dir}}/combat.ps1"
    valuearg=()
    [ -z "$1" ] || valuearg=(-Value "$1")
    {{ssh}} "{{ps}} -File {{dir}}/combat.ps1 ${valuearg[*]}"

# Read AllowGather, or set true/false before restarting the game.
[positional-arguments]
gather VALUE='': need-target
    #!/usr/bin/env bash
    set -euo pipefail
    [[ -z "$1" || "$1" = true || "$1" = false ]] || { echo 'Use true or false' >&2; exit 2; }
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/gather.ps1" "{{target}}:/{{dir}}/gather.ps1"
    valuearg=()
    [ -z "$1" ] || valuearg=(-Value "$1")
    {{ssh}} "{{ps}} -File {{dir}}/gather.ps1 ${valuearg[*]}"

# Prompts here without echoing, and sends the value over SSH's standard
# input, so it is in no command line, log or chat. `just vm secret CPA_API_KEY`.
[doc("Store a secret for the controller on the VM, such as an LLM key.")]
secret NAME: need-target
    #!/usr/bin/env bash
    set -euo pipefail
    [[ "{{NAME}}" =~ ^[A-Z_][A-Z0-9_]*$ ]] || { echo "NAME must look like CPA_API_KEY" >&2; exit 2; }
    read -rsp "{{NAME}} (not shown): " value; echo
    [ -n "$value" ] || { echo "empty; nothing changed" >&2; exit 1; }
    printf '%s' "$value" | {{ssh}} "{{ps}} -File {{dir}}/set-secret.ps1 -Name {{NAME}}"

# An interactive PowerShell on the VM.
shell: need-target
    ssh {{target}}

# Monitor one durable combat objective across native encounter failures.
[positional-arguments]
goal-trial *ARGS:
    python3 -B "{{justfile_directory()}}/scripts/vm/combat-trial.py" --persistent "$@"

test-goal-trial:
    python3 -B "{{justfile_directory()}}/scripts/vm/goal-trial-test.py"

# With Tas online, verify native on/off marker telemetry and preserve the selected final state.
[positional-arguments]
marker-check FINAL='restore':
    python3 -B "{{justfile_directory()}}/scripts/vm/marker-check.py" "$1"

test-marker-check:
    python3 -B "{{justfile_directory()}}/scripts/vm/marker-check-test.py"

# Capture measured terrain, or compare a saved viewport after reconnect/logout.
[positional-arguments]
atlas-check *ARGS:
    python3 -B "{{justfile_directory()}}/scripts/vm/atlas-check.py" "$@"

test-atlas-check:
    python3 -B "{{justfile_directory()}}/scripts/vm/atlas-check-test.py"
