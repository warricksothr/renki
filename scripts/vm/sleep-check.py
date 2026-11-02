#!/usr/bin/env python3
"""Prove native bed entry, optionally interrupting an owned finite escort."""
import argparse
import datetime
import json
import math
from pathlib import Path
import subprocess
import time
from urllib.parse import urlencode

ROOT = Path(__file__).resolve().parents[2]
FIELDS = ('connected', 'phase', 'world', 'worldUid', 'connection', 'character', 'pos',
          'hp', 'hpMax', 'flags', 'task', 'escort', 'time', 'owners', 'players')


def records(value):
    return value.get('value', []) if isinstance(value, dict) else value


def terminal(history, task_id, connection):
    return next((r['frame'] for r in reversed(records(history)) if r.get('conn') == connection
                 and r.get('frame', {}).get('type') == 'task'
                 and r['frame'].get('re') == task_id
                 and r['frame'].get('status') in ('succeeded', 'failed', 'rejected', 'cancelled')), None)


class Trial:
    def __init__(self, call, save, pause=time.sleep, clock=time.monotonic):
        self.call, self.save, self.pause, self.clock = call, save, pause, clock
        self.scope = self.run = None
        self.allowed = set()
        self.result = dict(outcome='failed', logoutRequested=False)
        self.anchor = datetime.datetime.now(datetime.timezone.utc).isoformat()

    def observe(self):
        started = self.clock()
        bundle = self.call('observe')
        state, inspection = bundle['status'], bundle['inspection']
        state = {k: state[k] for k in FIELDS if k in state}
        self.save('observations.jsonl', dict(status=state, inspection={k: inspection.get(k) for k in
                  ('run', 'world', 'worldUid', 'connection', 'freshness', 'body')}), append=True)
        scope = tuple(state.get(k) for k in ('world', 'worldUid', 'connection', 'character'))
        if not all(scope) or self.scope and scope != self.scope or self.run and inspection.get('run') != self.run:
            raise RuntimeError('world, character, connection or controller run changed; yielded')
        fresh = inspection.get('freshness') or {}
        age = fresh.get('ageS')
        elapsed = self.clock() - started
        if (not state.get('connected') or state.get('phase') != 'in_world'
                or inspection.get('contract') != 'inspection-v1' or not inspection.get('run')
                or tuple(inspection.get(k) for k in ('world', 'worldUid', 'connection')) != scope[:3]
                or fresh.get('status') != 'fresh' or type(age) not in (int, float)
                or not math.isfinite(age) or age < 0 or elapsed < 0 or age + elapsed > 3
                or type(fresh.get('seq')) is not int or fresh['seq'] <= 0):
            raise RuntimeError('fresh scoped native observation required')
        body = inspection.get('body') or {}
        ids = [(state.get('task') or {}).get('id'), body.get('taskId')]
        if any(key and key not in self.allowed for key in ids) or body.get('owner') in ('automation', 'behaviour', 'native_recovery'):
            raise RuntimeError('another body order owns Tas; yielded')
        if type(state.get('hp')) not in (int, float) or not math.isfinite(state['hp']) or state['hp'] <= 0:
            raise RuntimeError('bot is dead or health is invalid')
        self.scope, self.run = scope, inspection['run']
        return state, fresh['seq']

    def history(self, key, name):
        value = self.call('panel', 'GET', '/api/task?' + urlencode(dict(id=key, at=self.anchor)))
        self.save(name, value)
        return value

    def run_trial(self, leader=None, timeout=90):
        state, fence = self.observe()
        if (state.get('time') or {}).get('night') is not True:
            raise RuntimeError('nighttime required; no task dispatched')
        deadline = self.clock() + 6
        while True:
            mapped = self.call('panel', 'GET', '/api/map')
            self.save('initial-map.json', mapped)
            if tuple(mapped.get(k) for k in ('world', 'worldUid', 'connection')) != self.scope[:3]:
                raise RuntimeError('map scope differs from native state')
            seen = (mapped.get('perception') or {}).get('seq')
            if type(seen) is int and seen > fence:
                break
            if self.clock() >= deadline:
                raise RuntimeError('new scoped bed perception required; no task dispatched')
            self.pause(1)
        beds = [b for b in (mapped.get('perception') or {}).get('structures', [])
                if b.get('kind') == 'bed' and b.get('mine') is True and b.get('owned') is True and b.get('id')
                and math.dist(b['pos'], state['pos']) <= 20]
        if not beds:
            raise RuntimeError('no currently observed owned bed within 20 m')
        bed = min(beds, key=lambda b: math.dist(b['pos'], state['pos']))
        self.result['bed'] = bed
        escort_id = None
        if leader:
            # Profiles are intentionally read from status, not inferred from encountered players.
            permitted = list(state.get('owners') or [])
            permitted += [name for name, role in (state.get('players') or {}).get('profiles', {}).items()
                          if role == 'operator' and name not in permitted]
            if leader not in permitted:
                raise RuntimeError('escort leader must be a configured owner/operator')
            self.observe()
            request = dict(player=leader, protect=permitted, collect=False, resupply=False)
            escort_id = self.call('task', 'escort', json.dumps(request), '15')['id']
            self.allowed.add(escort_id)
            self.result['escortTask'] = escort_id
            deadline = self.clock() + 8
            while self.clock() < deadline:
                state, _ = self.observe()
                if ((state.get('task') or {}).get('id') == escort_id
                        and (state.get('escort') or {}).get('id') == escort_id):
                    break
                if terminal(self.history(escort_id, 'escort.json'), escort_id, self.scope[2]):
                    raise RuntimeError('escort ended before sleep could interrupt it')
                self.pause(1)
            else:
                raise RuntimeError('escort admission not observed')
        # Refresh ownership immediately before the only overriding request.
        self.observe()
        args = dict(entity=bed['id'], sleep=True)
        key = self.call('task', 'interact', json.dumps(args), str(timeout))['id']
        self.allowed.add(key)
        self.result['sleepTask'] = key
        deadline = self.clock() + timeout + 10
        receipt = None
        while self.clock() < deadline:
            history = self.history(key, 'sleep.json')
            receipt = terminal(history, key, self.scope[2])
            state, seq = self.observe()
            if receipt and receipt['status'] != 'succeeded':
                self.result.update(outcome='refused' if receipt.get('reason') == 'refused' else 'failed', receipt=receipt)
                return self.result
            if (receipt and seq > receipt.get('seq', float('inf')) and 'in_bed' in state.get('flags', [])
                    and math.dist(state['pos'], bed['pos']) <= 2):
                if escort_id:
                    ended = terminal(self.history(escort_id, 'escort.json'), escort_id, self.scope[2])
                    if not ended or ended.get('status') != 'cancelled':
                        raise RuntimeError('bed entry observed without exact escort cancellation receipt')
                self.result.update(outcome='succeeded', receipt=receipt, inBed=True,
                                   sleeping='sleeping' in state.get('flags', []), finalPos=state['pos'])
                return self.result
            self.pause(1)
        raise RuntimeError('bed receipt and newer in_bed observation not both verified')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--interrupt-escort', metavar='OWNER')
    parser.add_argument('--timeout', type=int, choices=range(15, 121), default=90, metavar='15..120')
    args = parser.parse_args()
    folder = ROOT / '.checks' / ('sleep-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%S%fZ'))
    folder.mkdir(parents=True)

    def save(name, value, append=False):
        with (folder / name).open('a' if append else 'w') as out:
            out.write(json.dumps(value, indent=None if append else 2) + '\n')

    def call(*arguments):
        result = subprocess.run(['just', 'vm', *arguments], cwd=ROOT, capture_output=True, text=True, timeout=45)
        with (folder / 'commands.log').open('a') as out:
            out.write(result.stdout + result.stderr)
        if result.returncode:
            raise RuntimeError('VM request failed; see retained commands.log')
        return json.loads(result.stdout)

    trial = Trial(call, save)
    try:
        trial.run_trial(args.interrupt_escort, args.timeout)
    except Exception as error:
        trial.result['reason'] = str(error)
    # All tasks have native timeouts. No cleanup order can displace a human,
    # wake Tas after successful entry, or log him out after a failed test.
    trial.result['evidence'] = str(folder)
    save('summary.json', trial.result)
    print(json.dumps(trial.result))
    return 0 if trial.result['outcome'] == 'succeeded' else 1


if __name__ == '__main__':
    raise SystemExit(main())
