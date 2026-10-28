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

# Build the package, stop the bot, copy it over, install the plugin, start again.
deploy: need-target
    #!/usr/bin/env bash
    set -euo pipefail
    just --justfile "{{justfile()}}" package
    if {{ssh}} "if (Get-Process valheim -ErrorAction SilentlyContinue) { exit 1 }" >/dev/null 2>&1; then
        echo "the game is not running"
    else
        just --justfile "{{justfile()}}" vm stop
    fi
    {{ssh}} "New-Item -ItemType Directory -Force -Path '{{dir}}' | Out-Null"
    scp -o BatchMode=yes -r "{{justfile_directory()}}/dist/renki-vm/." "{{target}}:/{{dir}}/"
    {{ssh}} "{{ps}} -File {{dir}}/install.ps1"
    just --justfile "{{justfile()}}" vm start

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

# Run once, and again after the Tailscale address changes. Then `just vm restart`.
[doc("Turn the control panel on: Tailscale address, firewall, token.")]
panel-setup PORT="8780": need-target
    {{ssh}} "{{ps}} -File {{dir}}/setup-panel.ps1 -Port {{PORT}}"

# Prints to this terminal only; it is the panel's login.
[doc("Print the control panel's address and login token.")]
panel-token: need-target
    @{{ssh}} "\$c = Get-Content '{{dir}}/controller.local.json' -Raw | ConvertFrom-Json; if (-not \$c.panel.listen) { 'The panel is off: run just vm panel-setup'; exit 1 }; 'address: http://' + \$c.panel.listen + '/'; 'token:   ' + \$c.panel.token"

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
