#!/usr/bin/env python3
"""Supervise the one-shot native startup return; never issue body steps."""
import datetime
import argparse
import importlib.util
import json
from pathlib import Path
import subprocess
import time

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('food_check', Path(__file__).with_name('food-check.py'))
food = importlib.util.module_from_spec(spec)
spec.loader.exec_module(food)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('world', help='exact known world name; native login must match')
    args = parser.parse_args()
    folder = ROOT / '.checks' / ('recovery-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
    folder.mkdir(parents=True)
    result = dict(outcome='failed', reason='not started', evidence=str(folder),
                  startedAt=datetime.datetime.now(datetime.timezone.utc).isoformat())
    launch, owns, previous = None, False, None
    log = (folder / 'commands.log').open('a')

    def call(*args, decode=True):
        p = subprocess.run(['just', 'vm', *args], cwd=ROOT, capture_output=True, text=True, timeout=45)
        log.write(p.stdout + p.stderr); log.flush()
        if p.returncode:
            raise RuntimeError('VM request failed: ' + str(args[:3]))
        return json.loads(p.stdout) if decode else p.stdout

    try:
        status = call('panel', 'GET', '/api/status')
        if status.get('connected'):
            raise RuntimeError('recovery requires offline bot; no body replaced')
        result['order'] = call('recovery-return', 'arm', args.world)
        owns = True
        launch = subprocess.Popen(['just', 'vm', 'start', '2'], cwd=ROOT, stdout=log, stderr=log)
        deadline, seen, scope = time.monotonic() + 240, False, None
        while time.monotonic() < deadline:
            status = call('panel', 'GET', '/api/status')
            with (folder / 'observations.jsonl').open('a') as out:
                out.write(json.dumps(dict(at=datetime.datetime.now(datetime.timezone.utc).isoformat(), status=status)) + '\n')
            if status.get('phase') != 'in_world':
                if seen:
                    raise RuntimeError('native recovery left the world')
                if launch.poll() is not None and launch.returncode:
                    raise RuntimeError('join failed')
                time.sleep(1)
                continue
            seen = True
            reason = food.safe(status, previous)
            if reason:
                raise RuntimeError(reason)
            current = status.get('world'), status.get('connection')
            if scope is None:
                scope = current
            if scope != current or status.get('task'):
                raise RuntimeError('recovery scope or body ownership changed')
            previous = status
            result['last'] = status
            if 'recovering' not in status.get('flags', []):
                mapped = call('panel', 'GET', '/api/map')
                (folder / 'map.json').write_text(json.dumps(mapped))
                if not food.safe_to_remain(status, mapped) or status['hp'] < status['hpMax'] * .95:
                    raise RuntimeError('native release lacks fresh safe-home recovery evidence')
                result.update(outcome='passed', reason='native return and sheltered recovery observed')
                break
            time.sleep(1)
        else:
            raise RuntimeError('recovery supervisor deadline')
    except Exception as exc:
        result['reason'] = str(exc)
    finally:
        errors = []
        if launch and launch.poll() is None:
            launch.terminate()
            try:
                launch.wait(timeout=5)
            except subprocess.TimeoutExpired:
                launch.kill(); launch.wait(timeout=5)
        if owns:
            if result['outcome'] != 'passed':
                for args in (('stop',), ('panel-start',)):
                    try:
                        call(*args, decode=False)
                    except Exception as exc:
                        errors.append(str(exc))
            try:
                call('recovery-return', 'clear', decode=False)
            except Exception as exc:
                errors.append(str(exc))
        if errors:
            result.update(outcome='failed', cleanupErrors=errors)
        result['endedAt'] = datetime.datetime.now(datetime.timezone.utc).isoformat()
        if owns:
            try:
                result['audit'] = call('audit-window', result['startedAt'], result['endedAt'], decode=False).strip()
            except Exception as exc:
                result['auditError'] = str(exc)
        log.close()
        (folder / 'summary.json').write_text(json.dumps(result, indent=2) + '\n')
        print(json.dumps({k: v for k, v in result.items() if k not in ('last', 'order')}, indent=2))
    return 0 if result['outcome'] == 'passed' else 1


if __name__ == '__main__':
    raise SystemExit(main())
