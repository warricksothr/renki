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

STATUS_FIELDS = ('version', 'connected', 'phase', 'world', 'worldUid', 'connection', 'character',
                 'pos', 'hp', 'hpMax', 'stamina', 'food', 'flags', 'location', 'home', 'task',
                 'injurySeq', 'injury', 'journey')


def position(value):
    return (isinstance(value, list) and len(value) == 3
            and all(type(v) in (int, float) and math.isfinite(v) for v in value))

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
        self.inspection = None
        self.run_id = None
        self.floor_marker = None
        self.home_target = None
        self.result = dict(outcome='failed', completedTrips=0, receipts=[], observedTravelM=0, minimumHP=None)

    def observe(self, check_safety=True):
        requested = self.clock()
        # Share one SSH startup/upload without weakening the freshness window.
        bundle = self.call('observe')
        state, inspection = bundle['status'], bundle['inspection']
        state = {key: value for key, value in state.items() if key in STATUS_FIELDS}
        self.save('observations.jsonl', dict(at=datetime.datetime.now(datetime.timezone.utc).isoformat(), status=state), append=True)
        scope = state.get('world'), state.get('connection'), state.get('character'), state.get('worldUid')
        if self.scope and scope != self.scope:
            self.yielded = True
            raise RuntimeError('world, character or connection changed')
        observed = inspection.get('freshness') or {}
        if self.run_id and inspection.get('run') != self.run_id:
            self.yielded = True
            raise RuntimeError('controller run changed')
        if (inspection.get('connection'), inspection.get('world'), inspection.get('worldUid')) != (
                state.get('connection'), state.get('world'), state.get('worldUid')):
            self.yielded = bool(self.scope)
            raise RuntimeError('inspection scope changed')
        age = observed.get('ageS')
        elapsed = self.clock() - requested
        if (inspection.get('contract') != 'inspection-v1' or not inspection.get('run')
                or observed.get('status') != 'fresh' or type(age) not in (int, float)
                or not math.isfinite(age) or age < 0 or elapsed < 0 or age + elapsed > 3
                or type(observed.get('seq')) is not int or observed['seq'] <= 0):
            raise RuntimeError('fresh scoped inspection required')
        self.inspection = inspection
        self.save('inspection.json', {key: inspection.get(key) for key in
                                     ('run', 'world', 'worldUid', 'connection', 'freshness', 'body')})
        body = inspection.get('body') or {}
        task_ids = [(state.get('task') or {}).get('id'), body.get('taskId')]
        if (any(task and task != self.current_task for task in task_ids)
                or body.get('owner') == 'automation' and not self.current_task):
            self.yielded = True
            raise RuntimeError('new body order; benchmark yielded')
        if self.home and ((state.get('home') or {}).get('pos') != self.home or
                          (state.get('home') or {}).get('world') != self.scope[0]):
            self.yielded = True
            raise RuntimeError('pinned home changed')
        reason = policy.unsafe(state, self.previous)
        if check_safety and (reason or 'recovering' in state.get('flags', []) or body.get('owner') == 'native_recovery'):
            raise RuntimeError(reason or 'native recovery owns the body')
        if self.previous:
            self.result['observedTravelM'] += math.dist(self.previous['pos'], state['pos'])
        self.result['minimumHP'] = min(state['hp'], self.result['minimumHP'] or state['hp'])
        self.previous = state
        return state

    def map_scoped(self, mapped):
        return (mapped.get('world'), mapped.get('connection'), mapped.get('worldUid')) == (
            self.scope[0], self.scope[1], self.scope[3])

    def reference_unchanged(self, mapped):
        return not self.floor_marker or [w.get('pos') for w in mapped.get('waypoints', [])
                                        if w.get('name') == self.floor_marker['name']] == [self.floor_marker['pos']]

    def at_home(self, state, mapped):
        return (policy.safe_home(state, mapped) and state.get('home', {}).get('pos') == self.home
                and self.map_scoped(mapped) and self.reference_unchanged(mapped)
                and 'grounded' in state.get('flags', []) and position(state.get('pos'))
                and math.dist(state['pos'], self.home_target) <= 3
                and abs(state['pos'][1] - self.home_target[1]) <= .75)

    def pin_floor(self, state, mapped, name):
        if not name:
            return
        matches = [w for w in mapped.get('waypoints', []) if w.get('name') == name]
        if len(matches) != 1 or not position(matches[0].get('pos')):
            raise RuntimeError('missing, ambiguous or malformed home floor marker')
        pos = matches[0]['pos']
        if (not self.scope[3] or not position(state.get('pos')) or not policy.safe_home(state, mapped)
                or 'grounded' not in state.get('flags', []) or not self.map_scoped(mapped)
                or math.dist(pos, self.home) > 8 or abs(pos[1] - self.home[1]) > 1.75
                or math.hypot(state['pos'][0]-self.home[0], state['pos'][2]-self.home[2]) > 3
                or abs(state['pos'][1]-pos[1]) > .75):
            raise RuntimeError('home floor marker lacks scoped grounded sheltered observation within original home envelope')
        self.floor_marker = dict(name=name, pos=list(pos))
        self.home_target = [self.home[0], pos[1], self.home[2]]
        self.save('home-floor-reference.json', dict(
            run=self.run_id, world=self.scope[0], connection=self.scope[1], character=self.scope[2], worldUid=self.scope[3],
            home=self.home, marker=self.floor_marker, target=self.home_target, observedPosition=state['pos'],
            observation=self.inspection['freshness'], perceptionSeq=mapped['perception']['seq']))

    def leg(self, target, label, timeout):
        before = self.observe()
        if before.get('task') or not policy.travel_ready(before):
            raise RuntimeError('travel requires an idle body and a meal lasting at least 300s')
        if self.floor_marker:
            mapped = self.call('panel', 'GET', '/api/map')
            if not self.map_scoped(mapped) or not self.reference_unchanged(mapped):
                raise RuntimeError('home floor reference changed; no stale route dispatched')
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
                if not self.reference_unchanged(mapped):
                    raise RuntimeError('home floor reference changed')
                if (self.map_scoped(mapped)
                        and perception.get('seq', 0) > receipt['seq']
                        and self.inspection['freshness']['seq'] > receipt['seq'] and not state.get('task')):
                    expected = self.home_target if label.endswith('-home') else target
                    if math.dist(state['pos'], expected) > 2 or abs(state['pos'][1] - expected[1]) > .75:
                        raise RuntimeError('movement receipt lacks observed arrival on target floor')
                    if label.endswith('-home') and not self.at_home(state, mapped):
                        raise RuntimeError('return lacks fresh sheltered home-floor arrival')
                    self.save(label + '-map.json', mapped)
                    self.result['receipts'].append(dict(label=label, taskId=self.current_task, seq=receipt['seq'], pos=state['pos']))
                    self.current_task = None
                    return
            self.pause(1)
        raise RuntimeError('movement receipt/arrival deadline')

    def run(self, names, cycles, timeout, home_floor_marker=None):
        if not names or len(set(names)) != len(names) or cycles < 1 or cycles * len(names) > 10:
            raise RuntimeError('use unique markers and at most ten round trips')
        state = self.observe()
        self.scope = state['world'], state['connection'], state.get('character'), state.get('worldUid')
        self.run_id = self.inspection['run']
        home = state.get('home') or {}
        if home.get('world') != state['world'] or not position(home.get('pos')):
            raise RuntimeError('known home in this world required')
        self.home = list(home['pos'])
        self.home_target = list(self.home)
        fence = self.inspection['freshness']['seq']
        deadline = self.clock() + 3
        mapped = self.call('panel', 'GET', '/api/map')
        while (self.map_scoped(mapped) and type((mapped.get('perception') or {}).get('seq')) is int
               and mapped['perception']['seq'] <= fence and self.clock() < deadline):
            self.pause(.5)
            # Keep the original state/inspection pair as the admission evidence.
            # Each leg refreshes safety and body ownership before dispatch.
            mapped = self.call('panel', 'GET', '/api/map')
        if (not self.map_scoped(mapped) or type((mapped.get('perception') or {}).get('seq')) is not int
                or mapped['perception']['seq'] <= fence or self.clock() >= deadline):
            raise RuntimeError('new scoped perception required before route admission')
        self.pin_floor(state, mapped, home_floor_marker)
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
        self.save('plan.json', dict(world=self.scope[0], worldUid=self.scope[3], connection=self.scope[1],
                                    character=self.scope[2], run=self.run_id, version=state.get('version'),
                                    home=self.home, homeFloorReference=self.floor_marker, homeArrivalTarget=self.home_target,
                                    markers=routes, cycles=cycles, legTimeoutS=timeout))
        self.result['outcome'] = 'in_progress'
        self.save('summary.json', self.result)
        for cycle in range(cycles):
            for number, (name, pos) in enumerate(routes):
                current = self.call('panel', 'GET', '/api/map')
                if not self.map_scoped(current):
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

    def finish(self, keep_online=False, allow_logout=False):
        remain = False
        if self.owned and not self.yielded:
            try:
                state = self.observe(check_safety=False)
                recover_until = self.clock() + 90
                while (keep_online and self.result['outcome'] == 'passed' and
                       self.at_home(state, self.call('panel', 'GET', '/api/map')) and
                       state['hp'] < state['hpMax'] * .95 and self.clock() < recover_until):
                    self.pause(1)
                    state = self.observe()
                remain = bool(keep_online and self.result['outcome'] == 'passed' and
                              state['hp'] >= state['hpMax'] * .95 and
                              self.at_home(state, self.call('panel', 'GET', '/api/map')))
                if not remain and allow_logout:
                    self.call('panel', 'POST', '/api/act/logout', '{}')
            except Exception as error:
                # An owner command or changed session revokes cleanup ownership.
                if not self.yielded:
                    self.result['cleanupError'] = str(error)
        self.result.update(safeHomeVerified=remain, keptOnline=self.owned and not self.yielded and (remain or not allow_logout), yieldedToCommand=self.yielded)
        if 'cleanupError' in self.result:
            self.result['outcome'] = 'failed'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--allow-logout', action='store_true', help='explicitly allow test cleanup to log out/close the client; default stays online even on failure')
    parser.add_argument('--marker', action='append', required=True)
    parser.add_argument('--cycles', type=int, choices=range(1, 11), default=1)
    parser.add_argument('--leg-timeout', type=int, choices=range(30, 181), default=120)
    parser.add_argument('--keep-online', action='store_true')
    parser.add_argument('--home-floor-marker', help='explicit existing nearby marker whose height is verified at the sheltered home start; never changes home or native targets')
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
        trial.run(args.marker, args.cycles, args.leg_timeout, args.home_floor_marker)
    except Exception as error:
        trial.result.update(outcome='failed', reason=str(error))
        if trial.owned and not trial.yielded:
            try:
                trial.result['camera'] = call('camera')
            except Exception:
                pass
    finally:
        trial.finish(args.keep_online, args.allow_logout)
        if args.allow_logout and trial.owned and not trial.yielded and not trial.result['keptOnline']:
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
