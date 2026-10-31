#!/usr/bin/env python3
"""Start a permitted native escort and retain bounded live evidence."""
import argparse
import datetime
import json
import math
from pathlib import Path
import subprocess
import time

ROOT = Path(__file__).resolve().parents[2]


def unsafe(status, world):
    if not status.get('connected') or status.get('phase') != 'in_world':
        return 'bot disconnected or left world'
    if status.get('world') != world:
        return 'world changed'
    hp, maximum = status.get('hp', 0), status.get('hpMax', 0)
    if not math.isfinite(hp) or not math.isfinite(maximum) or maximum <= 0 or hp < max(15, maximum * .4):
        return 'health below escort supervision floor'
    return None


def protected(status, leader):
    names = list(status.get('owners') or [])
    for name, role in (status.get('players') or {}).get('profiles', {}).items():
        if role == 'operator' and name not in names:
            names.append(name)
    if leader not in names:
        raise RuntimeError('escort leader must be a configured owner/operator')
    return names


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('leader')
    parser.add_argument('--seconds', type=int, default=90)
    parser.add_argument('--leave-running', action='store_true')
    args = parser.parse_args()
    if not 10 <= args.seconds <= 3600:
        parser.error('--seconds must be between 10 and 3600')
    folder = ROOT / '.checks' / ('escort-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
    folder.mkdir(parents=True)

    def call(*arguments):
        result = subprocess.run(['just', 'vm', *arguments], cwd=ROOT, capture_output=True, text=True, timeout=45)
        with (folder / 'commands.log').open('a') as out:
            out.write(result.stdout + result.stderr)
        if result.returncode:
            raise RuntimeError(f'VM request failed: {arguments[:3]} (see commands.log)')
        return json.loads(result.stdout)

    started = False
    outcome, reason, task_id = 'failed', 'not started', None
    minimum, observations, modes = None, 0, set()
    try:
        initial = call('panel', 'GET', '/api/status')
        world = initial.get('world')
        if unsafe(initial, world):
            raise RuntimeError(unsafe(initial, world))
        if initial.get('hp', 0) < 30 or initial['hp'] < initial.get('hpMax', 0) * .8 or not (initial.get('food') or {}).get('active'):
            raise RuntimeError('recover to 30 HP/80% health with active food before supervised escort')
        names = protected(initial, args.leader)
        mapped = call('panel', 'GET', '/api/map')
        if not any(c.get('kind') == 'player' and c.get('name') == args.leader
                   for c in (mapped.get('perception') or {}).get('characters', [])):
            raise RuntimeError('leader is not currently visible')
        task = call('task', 'escort', json.dumps(dict(player=args.leader, protect=names)), '0')
        task_id, started = task['id'], True
        started_at = time.monotonic()
        matched = False
        deadline = started_at + args.seconds
        while time.monotonic() < deadline:
            status = call('panel', 'GET', '/api/status')
            with (folder / 'observations.jsonl').open('a') as out:
                out.write(json.dumps(dict(at=datetime.datetime.now(datetime.timezone.utc).isoformat(), status=status)) + '\n')
            observations += 1
            minimum = min(minimum if minimum is not None else float('inf'), status.get('hp', 0))
            escort, body = status.get('escort') or {}, status.get('task') or {}
            reason = unsafe(status, world)
            if reason:
                # Survival floor is an actual condition, not a hypothetical risk.
                call('panel', 'POST', '/api/act/logout', '{}')
                outcome = 'safety_logout'
                break
            if body.get('id') != task_id or body.get('task') != 'escort' or escort.get('id') != task_id:
                if not matched and time.monotonic() - started_at < 5:
                    time.sleep(1)
                    continue
                history = call('panel', 'GET', '/api/task?id=' + task_id)
                (folder / 'task.json').write_text(json.dumps(history, indent=2) + '\n')
                # Native/controller retreat or a newer human order owns the body.
                outcome, reason = 'interrupted', 'escort ended or another task took ownership'
                break
            matched = True
            modes.add(escort.get('mode'))
            time.sleep(2)
        else:
            outcome, reason = 'observed', 'bounded observation completed; live defense success requires encounter evidence'
    except Exception as error:
        reason = str(error)
    finally:
        if started and (not args.leave_running or outcome == 'failed'):
            try:
                current = call('panel', 'GET', '/api/status')
                if (current.get('task') or {}).get('id') == task_id:
                    call('task', 'stop', '{}', '0')
            except Exception as error:
                reason += '; cleanup: ' + str(error)
        summary = dict(outcome=outcome, reason=reason, taskId=task_id, minimumHP=minimum,
                       observations=observations, modes=sorted(m for m in modes if m), evidence=str(folder))
        (folder / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
        print(json.dumps(summary))
    return 0 if outcome == 'observed' else 1


if __name__ == '__main__':
    raise SystemExit(main())
