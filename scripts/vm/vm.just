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
lease := "python3 -B " + quote(justfile_directory() / "scripts/vm/lease.py") + " --target " + quote(target) + " --dir " + quote(dir)

# List the VM recipes.
default:
    @just --list vm

[private]
need-target:
    @[ -n "{{target}}" ] || { echo "no VM: set RENKI_VM=user@host or write it to vm.local" >&2; exit 1; }

# Every recipe that changes the VM, the game or the bot depends on `guard` or
# runs through `lease.py run`: it refuses while another worktree holds the VM
# lease, and takes the lease when it is free. Reading needs no lease, and
# neither do the emergency stops, `stop` and `panel POST /api/act/stop`, when
# a person runs them; inside a supervised run they guard too. See
# docs/VM.md, "The VM lease", and scripts/vm/lease.py.

# Who holds the VM lease and until when, without changing it.
lease-status: need-target
    @{{lease}} status

# Take the VM lease for deploys and live trials, or renew the one you hold.
[positional-arguments]
lease ACTOR MINUTES="60": need-target
    @{{lease}} take "$1" "$2"

# Give the VM lease back. --force clears another holder's lease: owner only.
[positional-arguments]
release *FLAGS: need-target
    @{{lease}} release "$@"

[private]
guard: need-target
    @{{lease}} guard

test-lease:
    python3 -B "{{justfile_directory()}}/scripts/vm/lease-test.py"

# Guard shared tooling that starts existing streaming tasks.
lease-guard: guard

# Start panel, existing private video stack and Tas; stream survives game stops.
online MINUTES="5": need-target guard
    just --justfile "{{justfile()}}" vm panel-start
    just --justfile "{{justfile()}}" streaming windows-online
    just --justfile "{{justfile()}}" vm start "{{MINUTES}}"

# Check the connection, the account and the scheduled tasks.
check: need-target
    {{ssh}} "whoami; hostname; Get-ScheduledTask -TaskName Renki* | Format-Table TaskName,State -AutoSize | Out-String -Width 120"

# Show the game, the controller and the bot; LINES > 0 adds the log tails.
status LINES="0": need-target
    {{ssh}} "{{ps}} -File {{dir}}/status.ps1 -Lines {{LINES}}"

# The tail of the plugin's and the controller's logs.
logs LINES="60": (status LINES)

# Keep the web panel available while the game is offline.
panel-start: need-target guard
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/start-bot.ps1" "{{justfile_directory()}}/scripts/vm/start-panel.ps1" "{{target}}:/{{dir}}/"
    {{ssh}} "{{ps}} -File {{dir}}/start-panel.ps1"

# Stop a controller that keeps the panel available while Valheim is offline.
panel-stop: need-target guard
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/stop-panel.ps1" "{{target}}:/{{dir}}/stop-panel.ps1"
    {{ssh}} "{{ps}} -File {{dir}}/stop-panel.ps1"

# Start the bot in the VM's desktop session and wait for it to be in the world.
start MINUTES="5": need-target guard
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

# Log the bot out, close the game and stop the controller. This is the
# emergency stop, so a person needs no lease for it; inside a supervised run
# it still guards (`lease.py guard --emergency`).
stop: need-target
    #!/usr/bin/env bash
    set -euo pipefail
    {{lease}} guard --emergency
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

# Stop, then start. The guard comes first, so a restart under someone else's
# lease refuses before it stops anything.
restart: guard stop start

# Restart only the controller, refreshing welcome/allowedTasks without reloading Valheim.
reconnect: need-target guard
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/reconnect.ps1" "{{target}}:/{{dir}}/reconnect.ps1"
    {{ssh}} "{{ps}} -File {{dir}}/reconnect.ps1"

# Preview migration backup sources without reading settings or save contents to stdout.
migration-inspect: need-target
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/migration-backup.ps1" "{{target}}:/{{dir}}/migration-backup.ps1"
    {{ssh}} "{{ps}} -File {{dir}}/migration-backup.ps1"

# Back up the mod loader, Renki settings/memory and saves; require an offline client/controller.
migration-backup: need-target guard
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/migration-backup.ps1" "{{target}}:/{{dir}}/migration-backup.ps1"
    {{ssh}} "{{ps}} -File {{dir}}/migration-backup.ps1 -Create"

# Download the official pinned DLL into .checks and verify the release asset digest.
[positional-arguments]
vplus-stage VERSION='10.4.0.0':
    python3 -B "{{justfile_directory()}}/scripts/vm/vplus-stage.py" "$1"

# Install a staged V+ DLL while offline, requiring a completed verified backup.
[positional-arguments]
vplus-install VERSION='10.4.0.0': need-target guard
    #!/usr/bin/env bash
    set -euo pipefail
    [[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Use an exact four-part version' >&2; exit 2; }
    staged="{{justfile_directory()}}/.checks/valheim-plus/$1"
    sha=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["sha256"])' "$staged/release.json")
    [[ "$sha" =~ ^[0-9a-f]{64}$ ]] || exit 2
    {{ssh}} "New-Item -ItemType Directory -Force -Path '{{dir}}/staging/vplus-$1' | Out-Null"
    scp -q -o BatchMode=yes "$staged/ValheimPlus.dll" "$staged/release.json" "{{target}}:/{{dir}}/staging/vplus-$1/"
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/vplus.ps1" "{{target}}:/{{dir}}/vplus.ps1"
    {{ssh}} "{{ps}} -File {{dir}}/vplus.ps1 -Action Install -Version '$1' -Sha256 '$sha'"

# Read versions, selected loader diagnostics and non-secret local V+ flags.
vplus-inspect: need-target
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/vplus.ps1" "{{target}}:/{{dir}}/vplus.ps1"
    {{ssh}} "{{ps}} -File {{dir}}/vplus.ps1"

# Stage and check before stopping anything, then install; leave the game offline.
install: need-target guard
    #!/usr/bin/env bash
    set -euo pipefail
    just --justfile "{{justfile()}}" package
    stage="{{dir}}/staging/deploy-$(date -u +%Y%m%dT%H%M%SZ)-$RANDOM"
    {{ssh}} "New-Item -ItemType Directory -Force -Path '$stage' | Out-Null"
    scp -q -o BatchMode=yes -r "{{justfile_directory()}}/dist/renki-vm/." "{{target}}:/$stage/"
    {{ssh}} "{{ps}} -File '$stage/deployment.ps1' -Root '{{dir}}' -Stage '$stage'"
    {{lease}} guard
    if {{ssh}} "if (Get-Process valheim -ErrorAction SilentlyContinue) { exit 10 }" >/dev/null; then
        echo "the game is not running"
    else
        probe_status=$?
        [ "$probe_status" -eq 10 ] || exit "$probe_status"
        just --justfile "{{justfile()}}" vm stop
    fi
    just --justfile "{{justfile()}}" vm panel-stop
    {{lease}} guard
    {{ssh}} "{{ps}} -File '$stage/install.ps1' -Root '{{dir}}' -Stage '$stage'"

# Offline entry-point ordering checks; Go compatibility/data checks run in controller tests.
test-deployment:
    python3 -B "{{justfile_directory()}}/scripts/vm/deployment-test.py"

# Verify both real refusal paths with a staged manifest; retain the running panel/game-offline state.
deployment-refusal: need-target guard
    #!/usr/bin/env bash
    set -euo pipefail
    just --justfile "{{justfile()}}" package
    stage="{{dir}}/staging/refusal-$(date -u +%Y%m%dT%H%M%SZ)-$RANDOM"
    {{ssh}} "New-Item -ItemType Directory -Force -Path '$stage' | Out-Null"
    scp -q -o BatchMode=yes -r "{{justfile_directory()}}/dist/renki-vm/." "{{target}}:/$stage/"
    {{lease}} guard
    {{ssh}} "{{ps}} -File '$stage/deployment-refusal.ps1' -Root '{{dir}}' -Stage '$stage'"

# Explicitly opt into starting the game after a compatible installation.
deploy: install
    just --justfile "{{justfile()}}" vm start

# Stage and validate before releasing controller locks; preserve offline state.
deploy-controller: need-target guard
    #!/usr/bin/env bash
    set -euo pipefail
    just --justfile "{{justfile()}}" controller build-windows
    candidate="{{justfile_directory()}}/dist/controller-deployment"
    mkdir -p "$candidate"
    cp "{{justfile_directory()}}/dist/controller.exe" "$candidate/controller.exe"
    just --justfile "{{justfile()}}" controller deployment-manifest "$candidate"
    cp "{{justfile_directory()}}/scripts/vm/deployment.ps1" "{{justfile_directory()}}/scripts/vm/deploy-controller.ps1" "$candidate/"
    stage="{{dir}}/staging/controller-$(date -u +%Y%m%dT%H%M%SZ)-$RANDOM"
    {{ssh}} "New-Item -ItemType Directory -Force -Path '$stage' | Out-Null"
    scp -q -o BatchMode=yes -r "$candidate/." "{{target}}:/$stage/"
    {{lease}} guard
    {{ssh}} "{{ps}} -File '$stage/deploy-controller.ps1' -Root '{{dir}}' -Stage '$stage'"

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
panel-setup PORT="8780": need-target guard
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
    # POST /api/act/stop is the panel's emergency stop, guarded like `stop`.
    emergency=()
    [ "$endpoint" != /api/act/stop ] || emergency=(--emergency)
    [ "$method" = GET ] || {{lease}} guard "${emergency[@]}"
    encoded=$(printf '%s' "${3:-}" | base64 -w0)
    bodyarg=()
    [ -z "$encoded" ] || bodyarg=(-BodyBase64 "$encoded")
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/panel-request.ps1" "{{target}}:/{{dir}}/panel-request.ps1"
    {{ssh}} "{{ps}} -File {{dir}}/panel-request.ps1 -Method '$method' -ApiPath '$endpoint' ${bodyarg[*]}"

# Read status and body provenance through one SSH process; no body actions.
observe: need-target
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/panel-request.ps1" "{{target}}:/{{dir}}/panel-request.ps1"
    {{ssh}} "{{ps}} -File {{dir}}/panel-request.ps1 -Method GET -ApiPath '/api/status' -IncludeInspection"

# Compact read-only explanation shared with the panel; optional saved goal ID.
[positional-arguments]
inspect GOAL='':
    #!/usr/bin/env bash
    set -euo pipefail
    [[ -z "$1" || "$1" =~ ^[a-zA-Z0-9_-]{1,96}$ ]] || { echo 'Invalid goal ID' >&2; exit 2; }
    endpoint=/api/inspection
    [ -z "$1" ] || endpoint+="?goal=$1"
    just --justfile "{{justfile()}}" vm panel GET "$endpoint"

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
    {{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/task-check.py" "$1" "$2" "$3"

# Start one automatic normal-controls encounter if a safe Greyling is available;
# retain results and stay online. --restart-client and --allow-logout are opt-in.
[positional-arguments]
combat-trial *ARGS:
    {{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/combat-trial.py" "$@"

# Harvest one isolated full beech and newly observed nearby logs; retain the client.
[positional-arguments]
harvest-trial *ARGS:
    {{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/combat-trial.py" --harvest-tree "$@"

# Retry already fallen beech logs without felling another tree; retain the client.
[positional-arguments]
harvest-logs X Y Z *ARGS:
    {{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/combat-trial.py" --harvest-logs "$1" "$2" "$3" "${@:4}"

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
    [ -z "$1" ] || {{lease}} guard
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
    [ -z "$1" ] || {{lease}} guard
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/gather.ps1" "{{target}}:/{{dir}}/gather.ps1"
    valuearg=()
    [ -z "$1" ] || valuearg=(-Value "$1")
    {{ssh}} "{{ps}} -File {{dir}}/gather.ps1 ${valuearg[*]}"

# Prompts here without echoing, and sends the value over SSH's standard
# input, so it is in no command line, log or chat. `just vm secret CPA_API_KEY`.
[doc("Store a secret for the controller on the VM, such as an LLM key.")]
secret NAME: need-target guard
    #!/usr/bin/env bash
    set -euo pipefail
    [[ "{{NAME}}" =~ ^[A-Z_][A-Z0-9_]*$ ]] || { echo "NAME must look like CPA_API_KEY" >&2; exit 2; }
    read -rsp "{{NAME}} (not shown): " value; echo
    [ -n "$value" ] || { echo "empty; nothing changed" >&2; exit 1; }
    printf '%s' "$value" | {{ssh}} "{{ps}} -File {{dir}}/set-secret.ps1 -Name {{NAME}}"

# Read-only native DLL locations and checksums, including duplicate installs.
native-libraries: need-target
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/native-libraries.ps1" "{{target}}:/{{dir}}/native-libraries.ps1"
    {{ssh}} "{{ps}} -File {{dir}}/native-libraries.ps1"

# An interactive PowerShell on the VM.
shell: need-target
    ssh {{target}}

# Monitor one durable combat objective across native encounter failures.
[positional-arguments]
goal-trial *ARGS:
    {{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/combat-trial.py" --persistent "$@"

test-goal-trial:
    python3 -B "{{justfile_directory()}}/scripts/vm/goal-trial-test.py"

# Return home, verify food withdrawals and let the food keeper replenish meals.
[positional-arguments]
resupply-food *ARGS:
    {{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/combat-trial.py" --resupply-food "$@"

test-resupply-food:
    python3 -B "{{justfile_directory()}}/scripts/vm/resupply-trial-test.py"
    python3 -B "{{justfile_directory()}}/scripts/vm/trial-status-test.py"

# Read the latest local trial's compact progress without polling the VM.
[positional-arguments]
trial-status *ARGS:
    python3 -B "{{justfile_directory()}}/scripts/vm/trial-status.py" "$@"

# With Tas online, verify native on/off marker telemetry and preserve the selected final state.
[positional-arguments]
marker-check FINAL='restore':
    {{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/marker-check.py" "$1"

test-marker-check:
    python3 -B "{{justfile_directory()}}/scripts/vm/marker-check-test.py"

# Capture measured terrain, or compare a saved viewport after reconnect/logout.
[positional-arguments]
atlas-check *ARGS:
    python3 -B "{{justfile_directory()}}/scripts/vm/atlas-check.py" "$@"

test-atlas-check:
    python3 -B "{{justfile_directory()}}/scripts/vm/atlas-check-test.py"

# Save the latest rendered JPEG and its scoped metadata under .checks/.
camera: need-target
    @scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/camera.ps1" "{{target}}:/{{dir}}/camera.ps1"
    @{{ssh}} "{{ps}} -File {{dir}}/camera.ps1" | python3 -B "{{justfile_directory()}}/scripts/vm/camera-save.py"

# Inspect effective/base camera settings, or send a bounded authenticated command.
[positional-arguments]
camera-config JSON='':
    #!/usr/bin/env bash
    set -euo pipefail
    if [ -z "$1" ]; then
        just --justfile "{{justfile()}}" vm panel GET /api/camera
    else
        just --justfile "{{justfile()}}" vm panel POST /api/camera "$1"
    fi

# Temporarily enable diagnostic captures; controller restores base automatically.
# INTERVAL is seconds per capture (1 fastest); duration is bounded to 1–300s.
[positional-arguments]
camera-diagnostic INTERVAL='1' SECONDS='30':
    #!/usr/bin/env bash
    set -euo pipefail
    body=$(python3 -c 'import json,sys; print(json.dumps(dict(action="temporary",intervalS=float(sys.argv[1]),durationS=float(sys.argv[2]),enabled=True),allow_nan=False))' "$1" "$2")
    just --justfile "{{justfile()}}" vm camera-config "$body"

# Change the normal capture interval live using agent permissions and pins.
[positional-arguments]
camera-rate INTERVAL='5':
    #!/usr/bin/env bash
    set -euo pipefail
    body=$(python3 -c 'import json,sys; print(json.dumps(dict(action="set",key="intervalS",value=float(sys.argv[1])),allow_nan=False))' "$1")
    just --justfile "{{justfile()}}" vm camera-config "$body"

# Capture read-only map data and the actual panel renderer for image tools.
[positional-arguments]
map-snapshot *ARGS:
    @python3 -B "{{justfile_directory()}}/scripts/vm/map-snapshot.py" "$@"

# Verify live 5 -> temporary 1 -> automatic 5 cadence, JPEG and map retrieval.
# Requires an in-world bot and enabled normal 5s camera; retain the client.
[positional-arguments]
camera-check *ARGS: need-target
    @{{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/camera-check.py" "$@"

# Check actual-frame cadence analysis without game or network access.
test-camera-check:
    python3 -B "{{justfile_directory()}}/scripts/vm/camera-check-test.py"

# Failed live supervisors preserve the client unless explicitly authorized to stop it.
test-client-lifecycle:
    python3 -B "{{justfile_directory()}}/scripts/vm/client-lifecycle-test.py"

# Test the controller gather -> gain-only storage -> return chain; retain the client.
[positional-arguments]
gather-trial *ARGS:
    {{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/combat-trial.py" --gather-tree "$@"

test-gather-trial:
    python3 -B "{{justfile_directory()}}/scripts/vm/gather-trial-test.py"

# Start native escort in an already joined/recovered game; retain bounded evidence.
[positional-arguments]
escort-check +ARGS:
    {{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/escort-check.py" "$@"

test-escort-check:
    python3 -B "{{justfile_directory()}}/scripts/vm/escort-check-test.py"

# Native nighttime bed entry; optionally interrupt a finite owner/operator escort.
[positional-arguments]
sleep-check *ARGS:
    {{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/sleep-check.py" "$@"

test-sleep-check:
    python3 -B "{{justfile_directory()}}/scripts/vm/sleep-check-test.py"

# One real cooking batch, optionally finding one ingredient in known home chests.
# Retain the client after the proof; --allow-logout explicitly enables shutdown.
[positional-arguments]
cooking-check *ARGS:
    {{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/cooking-check.py" "$@"

test-cooking-check:
    python3 -B "{{justfile_directory()}}/scripts/vm/cooking-check-test.py"

# One production food-resupply goal, with optional guarded join and proof predicates.
[positional-arguments]
food-check *ARGS:
    {{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/food-check.py" "$@"

# One bounded marker food batch, optional movement and automatic idle dwell.
[positional-arguments]
marker-food-check *ARGS:
    {{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/marker-food-check.py" "$@"

test-marker-food-check:
    python3 -B "{{justfile_directory()}}/scripts/vm/marker-food-check-test.py"

test-food-check:
    python3 -B "{{justfile_directory()}}/scripts/vm/food-check-test.py"

# Inspect at most sixteen captured bank candidates without querying the game.
[positional-arguments]
bank-evidence AUDIT:
    python3 -B "{{justfile_directory()}}/scripts/vm/bank-evidence.py" "$@"

# Return an adequately supplied outdoor bot through guarded native movement.
[positional-arguments]
recover-home WORLD *ARGS:
    {{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/recover-home.py" "$@"

test-recover-home:
    python3 -B "{{justfile_directory()}}/scripts/vm/recover-home-test.py"

# Read native startup refusal details from a saved audit without joining the game.
[positional-arguments]
recovery-evidence AUDIT:
    python3 -B "{{justfile_directory()}}/scripts/vm/recovery-evidence.py" "$@"

# Finite existing-marker round trips, each proved by native receipts and arrival.
[positional-arguments]
route-benchmark *ARGS:
    {{lease}} run -- python3 -B "{{justfile_directory()}}/scripts/vm/route-benchmark.py" "$@"

# Read bounded support/search diagnostics from a retained audit; no VM requests.
[positional-arguments]
route-evidence *ARGS:
    python3 -B "{{justfile_directory()}}/scripts/vm/route-evidence.py" "$@"

test-route-benchmark:
    python3 -B "{{justfile_directory()}}/scripts/vm/route-benchmark-test.py"

# Arm/clear a one-shot order using the controller's pinned home.
[positional-arguments]
recovery-return ACTION='arm' WORLD='' PROFILE='local': need-target guard
    #!/usr/bin/env bash
    set -euo pipefail
    [[ "$1" = arm || "$1" = clear ]] || { echo 'Use arm or clear' >&2; exit 2; }
    [[ "$3" = local || "$3" = work_trip || "$3" = nearby ]] || { echo 'Use local, work_trip or nearby profile' >&2; exit 2; }
    encoded=$(printf '%s' "$2" | base64 -w0)
    worldarg=()
    [ -z "$encoded" ] || worldarg=(-WorldBase64 "$encoded")
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/recovery-return.ps1" "{{justfile_directory()}}/scripts/vm/panel-request.ps1" "{{target}}:/{{dir}}/"
    {{ssh}} "{{ps}} -File {{dir}}/recovery-return.ps1 -Action '$1' -ReturnProfile '$3' ${worldarg[*]}"

# One-shot native sheltered healing guard; observes threats every physics step.
[positional-arguments]
recovery VALUE='': need-target
    #!/usr/bin/env bash
    set -euo pipefail
    [[ -z "$1" || "$1" = true || "$1" = false ]] || { echo 'Use true or false' >&2; exit 2; }
    [ -z "$1" ] || {{lease}} guard
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/recovery.ps1" "{{target}}:/{{dir}}/recovery.ps1"
    valuearg=()
    [ -z "$1" ] || valuearg=(-Value "$1")
    {{ssh}} "{{ps}} -File {{dir}}/recovery.ps1 ${valuearg[*]}"

# Independent opt-in to native recovery-failure logout; applies on next launch.
[positional-arguments]
auto-logout VALUE='': need-target
    #!/usr/bin/env bash
    set -euo pipefail
    [[ -z "$1" || "$1" = true || "$1" = false ]] || { echo 'Use true or false' >&2; exit 2; }
    [ -z "$1" ] || {{lease}} guard
    scp -q -o BatchMode=yes "{{justfile_directory()}}/scripts/vm/recovery.ps1" "{{target}}:/{{dir}}/recovery.ps1"
    valuearg=()
    [ -z "$1" ] || valuearg=(-Value "$1")
    {{ssh}} "{{ps}} -File {{dir}}/recovery.ps1 -Key AutoLogout ${valuearg[*]}"
