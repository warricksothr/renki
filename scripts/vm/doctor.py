#!/usr/bin/env python3
"""`just vm doctor`: is the bot's setup whole, and what fixes each gap?

One read-only pass over everything a session otherwise checks by hand before
it can deploy or start the bot: this worktree's ignored local files, SSH, the
VM's desktop session and scheduled tasks, the game and plugin files,
Tailscale and the panel address, the controller, the installed build against
this worktree and origin/main, the VM lease, disk space and the video tasks.

doctor.ps1 gathers the VM's facts; every judgement is made in evaluate() so
it can be tested offline (doctor-test.py). Nothing on the VM is changed and
nothing is copied to it. No token, key, password or address is printed.

Exit 0 when nothing failed (warnings allowed), 1 when a check failed.
"""
import argparse
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
MARKER = 'RENKI-DOCTOR '
LOCAL_FILES = ('vm.local', 'plugin/game.local.props')
LOW_DISK_GB = 10
OK, INFO, WARN, FAIL = 'ok', 'info', 'warn', 'fail'


def check(level, name, detail, fix=''):
    return {'level': level, 'name': name, 'detail': detail, 'fix': fix}


def evaluate(local, vm, starting=False):
    """Turn gathered facts into checks. `vm` is None when SSH was not tried,
    or {'error': text} when it failed. `starting` is `vm up` asking: it copies
    this worktree's start scripts to the VM, so a build from another branch
    blocks it."""
    out = []
    missing = [name for name in LOCAL_FILES if not local['files'].get(name)]
    if not local.get('target'):
        out.append(check(FAIL, 'worktree', 'no VM target: vm.local is missing and RENKI_VM is unset',
                         'just setup-worktree'))
        return out
    if missing:
        props = 'plugin/game.local.props' in missing
        out.append(check(WARN if props else OK, 'worktree', 'missing ' + ', '.join(missing) +
                         (' (the plugin cannot build, so no deploy)' if props else ' (RENKI_VM is set)'),
                         'just setup-worktree'))
    else:
        out.append(check(OK, 'worktree', 'local files present'))

    if vm is None or vm.get('error'):
        out.append(check(FAIL, 'ssh', (vm or {}).get('error') or 'not reached',
                         'docs/VM.md, "When something is off": sshd, firewall rule, authorized key'))
        return out
    if vm.get('script_error'):
        out.append(check(OK, 'ssh', 'connected'))
        out.append(check(FAIL, 'doctor', vm['script_error'],
                         'just vm shell, then run the checks by hand; report the gap in doctor.ps1'))
        return out
    out.append(check(OK, 'ssh', 'connected; VM up since ' + (vm.get('boot') or 'unknown')))

    if vm.get('desktop'):
        out.append(check(OK, 'desktop', 'the bot account has a desktop session'))
    else:
        out.append(check(FAIL, 'desktop', 'no logged-on desktop session, so the game cannot start',
                         'log on over Remote Desktop, leave with C:\\renki\\release-rdp.ps1; check auto-logon'))

    tasks = vm.get('tasks') or {}
    absent = [name for name in ('RenkiStart', 'RenkiStop') if name not in tasks]
    if absent:
        out.append(check(FAIL, 'tasks', 'missing ' + ', '.join(absent),
                         'run C:\\renki\\setup-remote.ps1 as administrator on the VM (docs/VM.md)'))
    else:
        out.append(check(OK, 'tasks', 'RenkiStart and RenkiStop are registered'))

    files = vm.get('files') or {}
    needed = [name for name in ('bot.local.psd1', 'controller.local.json') if not files.get(name)]
    if needed:
        out.append(check(FAIL, 'settings', 'missing on the VM: ' + ', '.join(needed),
                         'docs/VM.md, "Fill in the settings"'))
    else:
        out.append(check(OK, 'settings', 'bot and controller settings present'))
        if not files.get('secrets.local.json'):
            out.append(check(WARN, 'secrets', 'no secrets file, so the LLM voice is off',
                             'just vm secret CPA_API_KEY'))

    game = vm.get('game') or {}
    if not game.get('exe'):
        out.append(check(FAIL, 'game', 'valheim.exe is not where bot.local.psd1 says',
                         'install Valheim or correct ValheimExe (docs/VM.md)'))
    elif not game.get('bepinex'):
        out.append(check(FAIL, 'game', 'BepInEx is not installed in the game folder',
                         'docs/VM.md, "Install BepInEx"'))
    elif not game.get('plugin'):
        out.append(check(FAIL, 'game', 'the renki plugin is not installed', 'just vm install'))
    else:
        out.append(check(OK, 'game', 'Valheim, BepInEx and the renki plugin are installed'))
    running = vm.get('running') or {}
    if not running.get('steam'):
        out.append(check(WARN, 'steam', 'Steam is not running; the game may not join',
                         'start Steam in the bot account and leave it signed in'))

    panel = vm.get('panel') or {}
    tailscale = vm.get('tailscale') or {}
    panel_ok = True
    if panel.get('kind') == 'tailscale':
        if tailscale.get('backend') != 'Running':
            panel_ok = False
            state = tailscale.get('backend') or tailscale.get('service') or 'not installed'
            out.append(check(FAIL, 'tailscale', f'the panel listens on Tailscale, which is {state}',
                             'sign in to Tailscale on the VM'))
        else:
            out.append(check(OK, 'tailscale', 'signed in and running'))

    controller = vm.get('controller') or {}
    restart_panel = 'just vm panel-stop && just vm panel-start (game offline), or just vm reconnect (game running)'
    if not controller.get('local'):
        out.append(check(INFO, 'controller', 'runs elsewhere, not on the VM'))
    elif not controller.get('exe'):
        out.append(check(FAIL, 'controller', 'controller executable is not installed', 'just vm install'))
    elif not controller.get('running'):
        out.append(check(WARN, 'controller', 'not running, so the panel is off', 'just vm panel-start'))
    elif not controller.get('answering'):
        out.append(check(FAIL, 'controller', 'running but not answering its control address', restart_panel))
    else:
        out.append(check(OK, 'controller', 'running and answering'))

    if not panel.get('configured'):
        out.append(check(WARN, 'panel', 'no panel in controller.local.json', 'just vm panel-setup'))
    elif not panel_ok:
        pass
    elif panel.get('error') == 'no port in panel.listen':
        out.append(check(FAIL, 'panel', 'panel.listen in controller.local.json has no port', 'just vm panel-setup'))
    elif not panel.get('addressPresent'):
        out.append(check(FAIL, 'panel', 'the address the panel listens on does not exist on the VM',
                         'just vm panel-setup'))
    elif controller.get('running') and not 200 <= panel.get('http', 0) < 500:
        why = panel.get('error') or f"HTTP {panel.get('http', 0)}"
        out.append(check(FAIL, 'panel', f'not answering ({why})', restart_panel))
    elif controller.get('running'):
        out.append(check(OK, 'panel', 'answering'))

    deployment = vm.get('deployment')
    build = local.get('build') or {}
    if not deployment:
        out.append(check(WARN, 'build', 'no deployment receipt on the VM', 'just check vm install'))
    else:
        name = f"{deployment.get('version')} from {deployment.get('branch') or 'an unknown branch'}"
        if build.get('in_head') is None:
            out.append(check(WARN, 'build', f'installed {name}; this checkout does not have that commit',
                             'git fetch origin'))
        elif not build['in_head']:
            if starting:
                out.append(check(FAIL, 'build', f'installed {name} is not in this branch, and up would run it '
                                 "with this branch's start scripts",
                                 'merge or rebase onto the installed branch, or run up from a worktree that has it'))
            else:
                out.append(check(WARN, 'build', f'installed {name} is not in this branch, so a deploy from here is '
                                 'refused', 'merge or rebase onto the installed branch before deploying or starting'))
        elif build.get('ahead'):
            out.append(check(INFO, 'build', f"installed {name}; this branch is {build['ahead']} commit(s) ahead",
                             'just check vm deploy, when you want them on the VM'))
        else:
            out.append(check(OK, 'build', f'installed {name} is this commit'))
        if build.get('in_main') is False:
            out.append(check(WARN, 'main', 'origin/main does not contain the installed build',
                             'merge its pull request, so other branches can deploy'))

    lease = local.get('lease') or ''
    if lease:
        out.append(check(INFO, 'lease', lease))

    low = {d: gb for d, gb in (vm.get('diskFreeGB') or {}).items() if gb < LOW_DISK_GB}
    if low:
        out.append(check(WARN, 'disk', ', '.join(f'{d}: {gb} GB free' for d, gb in sorted(low.items())),
                         'audit logs under C:\\renki\\audit grow by several hundred MB a day'))

    video = {name: tasks.get(name) for name in ('RenkiOBS', 'RenkiRelay', 'RenkiCapture', 'RenkiCaptureWatchdog')
             if name in tasks}
    if video:
        stopped = sorted(name for name, state in video.items() if state != 'Running')
        if not stopped:
            out.append(check(OK, 'video', 'OBS, relay and capture tasks are running'))
        elif len(stopped) == len(video):
            out.append(check(INFO, 'video', 'private video is off', 'just streaming windows-online'))
        else:
            out.append(check(WARN, 'video', 'partly running; stopped: ' + ', '.join(stopped),
                             'just streaming windows-online'))

    bot = controller.get('bot') or {}
    if running.get('valheim'):
        state = bot.get('phase') or ('connected' if bot.get('connected') else 'plugin not connected')
        out.append(check(INFO, 'bot', f'the game is running ({state})'))
    else:
        out.append(check(INFO, 'bot', 'offline: the game is not running', 'just vm up'))
    return out


def failed(checks):
    return any(c['level'] == FAIL for c in checks)


def render(checks):
    marks = {OK: 'ok  ', INFO: 'info', WARN: 'WARN', FAIL: 'FAIL'}
    lines = []
    for c in checks:
        lines.append(f"{marks[c['level']]}  {c['name'] + ':':<12}{c['detail']}")
        if c['fix'] and c['level'] != OK:
            lines.append(f"{'':18}-> {c['fix']}")
    fails = sum(c['level'] == FAIL for c in checks)
    warns = sum(c['level'] == WARN for c in checks)
    lines.append(f'doctor: {fails} failed, {warns} warnings' if fails or warns else 'doctor: all clear')
    return '\n'.join(lines)


def git(*args):
    return subprocess.run(['git', *args], cwd=ROOT, capture_output=True, text=True, stdin=subprocess.DEVNULL)


def build_facts(source):
    """Where the installed commit sits relative to HEAD and origin/main.
    None means this checkout does not have the commit at all."""
    if not source or git('cat-file', '-e', source + '^{commit}').returncode != 0:
        return {'in_head': None, 'in_main': None, 'ahead': 0}
    in_head = git('merge-base', '--is-ancestor', source, 'HEAD').returncode == 0
    main = git('merge-base', '--is-ancestor', source, 'origin/main')
    count = git('rev-list', '--count', source + '..HEAD').stdout.strip() if in_head else '0'
    return {'in_head': in_head, 'in_main': main.returncode == 0 if main.returncode in (0, 1) else None,
            'ahead': int(count or 0)}


SSH_REASONS = (('timed out', 'connection timed out'), ('refused', 'connection refused'),
               ('permission denied', 'key not accepted'), ('could not resolve', 'host name does not resolve'),
               ('no route', 'no route to the VM'), ('host key', 'host key not accepted'))


def ssh_reason(output):
    """Fixed text for a failed SSH run: ssh's own message names the host."""
    lowered = output.lower()
    for needle, reason in SSH_REASONS:
        if needle in lowered:
            return reason
    return 'no facts returned; run just vm check to see why'


def gather_vm(target, vm_dir):
    script = f"$RenkiDir = '{vm_dir}'\n" + (Path(__file__).with_name('doctor.ps1')).read_text()
    command = 'powershell -NoProfile -ExecutionPolicy Bypass -Command "[Console]::In.ReadToEnd() | Invoke-Expression"'
    try:
        done = subprocess.run(['ssh', '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=10', target, command],
                              input=script, capture_output=True, text=True, timeout=120)
    except subprocess.TimeoutExpired:
        return {'error': 'the VM did not answer within 120 s'}
    for line in done.stdout.splitlines():
        if line.startswith(MARKER):
            try:
                return json.loads(line[len(MARKER):])
            except ValueError:
                break
    # ssh itself fails with 255; any other exit is the script's own.
    if done.returncode != 255:
        return {'script_error': f'doctor.ps1 ran on the VM but returned no facts (exit {done.returncode})'}
    return {'error': f'ssh exit 255: {ssh_reason(done.stderr + done.stdout)}'}


def lease_line(target, vm_dir):
    done = subprocess.run([sys.executable, '-B', str(Path(__file__).with_name('lease.py')),
                           '--target', target, '--dir', vm_dir, 'status'],
                          capture_output=True, text=True, stdin=subprocess.DEVNULL)
    if done.returncode != 0:
        return 'lease state unknown: ' + ssh_reason(done.stderr + done.stdout)
    lines = done.stdout.strip().splitlines()
    return lines[-1] if lines else ''


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument('--target', default='')
    parser.add_argument('--dir', default='C:/renki')
    parser.add_argument('--starting', action='store_true', help='also fail on what would make `vm up` unsafe')
    parser.add_argument('--json', action='store_true', help='print facts and checks as JSON')
    args = parser.parse_args(argv)

    local = {'target': args.target, 'files': {name: (ROOT / name).exists() for name in LOCAL_FILES}}
    vm = None
    if args.target:
        vm = gather_vm(args.target, args.dir)
        if not vm.get('error') and not vm.get('script_error'):
            local['build'] = build_facts((vm.get('deployment') or {}).get('source'))
            local['lease'] = lease_line(args.target, args.dir)
    checks = evaluate(local, vm, starting=args.starting)
    if args.json:
        local.pop('target')
        print(json.dumps({'local': local, 'vm': vm, 'checks': checks}, indent=2))
    else:
        print(render(checks))
    return 1 if failed(checks) else 0


if __name__ == '__main__':
    sys.exit(main())
