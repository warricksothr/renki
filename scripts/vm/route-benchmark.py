#!/usr/bin/env python3
"""Finite marker round trips with native receipts; stop at the first failure."""
import argparse
import datetime
import importlib.util
import json
import math
from pathlib import Path
import subprocess
import time
from urllib.parse import urlencode

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('route_marker_policy', Path(__file__).with_name('marker-food-check.py'))
policy = importlib.util.module_from_spec(spec)
spec.loader.exec_module(policy)


class Benchmark:
    def __init__(self, call, save, pause=time.sleep, clock=time.monotonic):
        self.call, self.save, self.pause, self.clock = call, save, pause, clock
        self.previous = self.scope = self.home = None
        self.owned = False
        self.yielded = False
        self.current_task = None
        self.result = dict(outcome='failed', completedTrips=0, receipts=[], observedTravelM=0, minimumHP=None)

    def observe(self, check_safety=True):
        state = self.call('panel', 'GET', '/api/status')
        self.save('observations.jsonl', dict(at=datetime.datetime.now(datetime.timezone.utc).isoformat(), status=state), append=True)
        scope = state.get('world'), state.get('connection'), state.get('character')
        if self.scope and scope != self.scope:
            self.yielded = True
            raise RuntimeError('world, character or connection changed')
        current = (state.get('task') or {}).get('id')
        if current and current != self.current_task:
            self.yielded = True
            raise RuntimeError('new body order; benchmark yielded')
        if self.home and ((state.get('home') or {}).get('pos') != self.home or
                          (state.get('home') or {}).get('world') != self.scope[0]):
            self.yielded = True
            raise RuntimeError('pinned home changed')
        reason = policy.unsafe(state, self.previous)
        if check_safety and (reason or 'recovering' in state.get('flags', [])):
            raise RuntimeError(reason or 'native recovery owns the body')
        if self.previous:
            self.result['observedTravelM'] += math.dist(self.previous['pos'], state['pos'])
        self.result['minimumHP'] = min(state['hp'], self.result['minimumHP'] or state['hp'])
        self.previous = state
        return state

    def at_home(self, state, mapped):
        return (policy.safe_home(state, mapped) and state.get('home', {}).get('pos') == self.home
                and 'grounded' in state.get('flags', []) and math.dist(state['pos'], self.home) <= 3
                and abs(state['pos'][1] - self.home[1]) <= .75)

    def leg(self, target, label, timeout):
        before = self.observe()
        if before.get('task') or not policy.travel_ready(before):
            raise RuntimeError('travel requires an idle body and a meal lasting at least 300s')
        anchor = datetime.datetime.now(datetime.timezone.utc).isoformat()
        self.owned = True
        # Ordinary native goal, never a sequence of controller-generated steps.
        request = self.call('task', 'move_to', json.dumps(dict(to=target, stopDistance=.75, dry=True, avoidHostiles=True)), str(timeout))
        self.current_task = request['id']
        until, receipt = self.clock() + timeout + 10, None
        while self.clock() < until:
            state = self.observe()
            history = self.call('panel', 'GET', '/api/task?' + urlencode(dict(id=self.current_task, at=anchor)))
            self.save(label + '-task.json', history)
            for row in policy.rows(history):
                frame = row.get('frame', {})
                if (row.get('conn') == self.scope[1] and frame.get('type') == 'task' and frame.get('re') == self.current_task
                        and frame.get('status') in ('succeeded', 'failed', 'rejected', 'cancelled')):
                    if frame['status'] != 'succeeded':
                        self.yielded = frame['status'] == 'cancelled'
                        raise RuntimeError('native movement ' + frame['status'] + ': ' + frame.get('detail', frame.get('reason', '')))
                    if type(frame.get('seq')) is not int or frame['seq'] <= 0:
                        raise RuntimeError('native receipt lacks sequence')
                    receipt = frame
            if receipt:
                mapped = self.call('panel', 'GET', '/api/map')
                perception = mapped.get('perception') or {}
                if ((mapped.get('world'), mapped.get('connection')) == self.scope[:2]
                        and perception.get('seq', 0) > receipt['seq'] and not state.get('task')):
                    if math.dist(state['pos'], target) > 2 or abs(state['pos'][1] - target[1]) > .75:
                        raise RuntimeError('movement receipt lacks observed arrival on target floor')
                    if label.endswith('-home') and not self.at_home(state, mapped):
                        raise RuntimeError('return lacks fresh sheltered home-floor arrival')
                    self.save(label + '-map.json', mapped)
                    self.result['receipts'].append(dict(label=label, taskId=self.current_task, seq=receipt['seq'], pos=state['pos']))
                    self.current_task = None
                    return
            self.pause(1)
        raise RuntimeError('movement receipt/arrival deadline')

    def run(self, names, cycles, timeout):
        if not names or len(set(names)) != len(names) or cycles < 1 or cycles * len(names) > 10:
            raise RuntimeError('use unique markers and at most ten round trips')
        state = self.observe()
        self.scope = state['world'], state['connection'], state.get('character')
        home = state.get('home') or {}
        if home.get('world') != state['world'] or not home.get('pos'):
            raise RuntimeError('known home in this world required')
        self.home = list(home['pos'])
        mapped = self.call('panel', 'GET', '/api/map')
        if not self.at_home(state, mapped):
            raise RuntimeError('benchmark requires a grounded, sheltered start on the pinned home floor; run supervised recovery separately')
        routes = []
        for name in names:
            matches = [w for w in mapped.get('waypoints', []) if w.get('name') == name]
            if len(matches) != 1:
                raise RuntimeError('missing or ambiguous marker: ' + name)
            pos = matches[0].get('pos')
            if (not isinstance(pos, list) or len(pos) != 3 or not all(type(v) in (int, float) and math.isfinite(v) for v in pos)
                    or not 4 <= math.dist(pos, self.home) <= 60):
                raise RuntimeError('marker must be finite and 4–60m from home: ' + name)
            routes.append((name, list(pos)))
        self.save('plan.json', dict(world=self.scope[0], home=self.home, markers=routes, cycles=cycles, legTimeoutS=timeout))
        self.result['outcome'] = 'in_progress'
        self.save('summary.json', self.result)
        for cycle in range(cycles):
            for number, (name, pos) in enumerate(routes):
                current = self.call('panel', 'GET', '/api/map')
                if (current.get('world'), current.get('connection')) != self.scope[:2]:
                    self.yielded = True
                    raise RuntimeError('map scope changed')
                if [w.get('pos') for w in current.get('waypoints', []) if w.get('name') == name] != [pos]:
                    raise RuntimeError('marker changed; no stale route dispatched')
                prefix = f'{cycle+1}-{number+1}'
                self.leg(pos, prefix + '-out', timeout)
                self.leg(self.home, prefix + '-home', timeout)
                self.result['completedTrips'] += 1
                self.save('summary.json', self.result)
        self.result.update(outcome='passed', reason='all round trips have native receipts and observed home-floor arrivals')

    def finish(self, keep_online):
        remain = False
        if self.owned and not self.yielded:
            try:
                state = self.observe(check_safety=False)
                remain = bool(keep_online and self.result['outcome'] == 'passed' and
                              self.at_home(state, self.call('panel', 'GET', '/api/map')))
                if not remain:
                    self.call('panel', 'POST', '/api/act/logout', '{}')
            except Exception as error:
                # An owner command or changed session revokes cleanup ownership.
                if not self.yielded:
                    self.result['cleanupError'] = str(error)
        self.result.update(keptOnline=remain, yieldedToCommand=self.yielded)
        if 'cleanupError' in self.result:
            self.result['outcome'] = 'failed'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--marker', action='append', required=True)
    parser.add_argument('--cycles', type=int, choices=range(1, 11), default=1)
    parser.add_argument('--leg-timeout', type=int, choices=range(30, 181), default=120)
    parser.add_argument('--keep-online', action='store_true')
    args = parser.parse_args()
    if len(set(args.marker)) != len(args.marker) or len(args.marker) * args.cycles > 10:
        parser.error('use unique markers and at most ten round trips')
    started = datetime.datetime.now(datetime.timezone.utc).isoformat()
    folder = ROOT / '.checks' / ('routes-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%S%fZ'))
    folder.mkdir(parents=True)

    def save(name, value, append=False):
        with (folder / name).open('a' if append else 'w') as out:
            out.write(json.dumps(value) + '\n')

    def call(*arguments):
        raw = arguments[0] in ('camera', 'audit-window', 'stop', 'panel-start')
        result = subprocess.run(['just', 'vm', *arguments], cwd=ROOT, capture_output=True, text=True, timeout=180 if arguments[0] == 'stop' else 45)
        with (folder / 'commands.log').open('a') as log:
            log.write(result.stdout + result.stderr)
        if result.returncode:
            raise RuntimeError('VM request failed: ' + str(arguments[:3]))
        return result.stdout.strip() if raw else json.loads(result.stdout)

    trial = Benchmark(call, save)
    try:
        trial.run(args.marker, args.cycles, args.leg_timeout)
    except Exception as error:
        trial.result.update(outcome='failed', reason=str(error))
        if trial.owned and not trial.yielded:
            try:
                trial.result['camera'] = call('camera')
            except Exception:
                pass
    finally:
        trial.finish(args.keep_online)
        if trial.owned and not trial.yielded and not trial.result['keptOnline']:
            for action in ('stop', 'panel-start'):
                try:
                    call(action)
                except Exception as error:
                    trial.result.update(outcome='failed', cleanupError=str(error))
        ended = datetime.datetime.now(datetime.timezone.utc).isoformat()
        if trial.owned:
            try:
                trial.result['audit'] = call('audit-window', started, ended)
            except Exception as error:
                trial.result['auditError'] = str(error)
        trial.result.update(evidence=str(folder), startedAt=started, endedAt=ended)
        save('summary.json', trial.result)
        print(json.dumps(trial.result, indent=2))
    return 0 if trial.result['outcome'] == 'passed' else 1


if __name__ == '__main__':
    raise SystemExit(main())
