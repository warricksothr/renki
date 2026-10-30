#!/usr/bin/env python3
"""Verify live shared-position changes; retain evidence and restore on failure."""
import argparse
import datetime
import json
from pathlib import Path
import subprocess
import time

ROOT = Path(__file__).resolve().parents[2]


def verify(call, pause, final):
    initial = call('GET', '/api/status')
    if not initial.get('connected') or initial.get('phase') != 'in_world':
        raise RuntimeError('marker check needs a connected in-world bot')
    if initial.get('sharePosition') is None:
        raise RuntimeError('plugin does not report marker state')
    original = bool(initial['presence']['sharePosition'])
    target = original if final == 'restore' else final == 'on'
    scope = (initial.get('connection'), initial.get('world'), initial.get('character'))
    evidence = []
    try:
        for desired in (True, False, target):
            call('POST', '/api/setting', dict(key='presence.sharePosition', value=desired))
            deadline = time.monotonic() + 20
            while time.monotonic() < deadline:
                state = call('GET', '/api/status')
                if not state.get('connected') or state.get('phase') != 'in_world' or scope != (state.get('connection'), state.get('world'), state.get('character')):
                    raise RuntimeError('marker check connection/world changed')
                if state.get('hp', 0) < state.get('hpMax', 1) * .7:
                    raise RuntimeError('health below marker check floor')
                if state.get('sharePosition') is desired:
                    evidence.append(dict(desired=desired, observed=state['sharePosition'], hp=state.get('hp'), hpMax=state.get('hpMax'), task=state.get('task')))
                    break
                pause(1)
            else:
                raise RuntimeError('marker state did not apply within twenty seconds')
    except Exception:
        call('POST', '/api/setting', dict(key='presence.sharePosition', value=original))
        raise
    return dict(outcome='passed', finalState=target, transitions=evidence)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('final', choices=['on', 'off', 'restore'], nargs='?', default='restore')
    args = parser.parse_args()
    saved = ROOT / '.checks' / ('marker-check-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%S%fZ'))
    saved.mkdir(parents=True)
    def call(method, path, body=None):
        cmd = ['just', 'vm', 'panel', method, path]
        if body is not None:
            cmd.append(json.dumps(body))
        result = subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True, timeout=45)
        with (saved / 'commands.log').open('a') as log:
            log.write(result.stdout + result.stderr)
        if result.returncode:
            raise RuntimeError('VM panel request failed; see retained commands.log')
        return json.loads(result.stdout)
    try:
        summary = verify(call, time.sleep, args.final)
    except Exception as error:
        summary = dict(outcome='failed', reason=str(error))
    summary['evidence'] = str(saved)
    (saved / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    print(json.dumps(summary, indent=2))
    return 0 if summary['outcome'] == 'passed' else 1


if __name__ == '__main__':
    raise SystemExit(main())
