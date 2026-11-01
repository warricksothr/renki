#!/usr/bin/env python3
"""Supervise one production food goal; never plan or puppet its native steps."""
import argparse
import datetime
import json
import math
from pathlib import Path
import subprocess
import time
from urllib.parse import urlencode
from harvest_trial import health_loss

ROOT = Path(__file__).resolve().parents[2]


def food_travel_damage(status, goal, previous=None, previous_goal=None):
    """Mirror the controller's finite allowance, with exact travel ownership."""
    if not goal or goal.get('kind') != 'food_resupply' or goal.get('status') not in ('running', 'succeeded') or goal.get('world') != status.get('world'):
        return False
    food = goal.get('food') or {}
    budget = food.get('travelBudget') or {}
    injury = status.get('injury') or {}
    seq, baseline = status.get('injurySeq'), budget.get('injury')
    damage, initial_damage = injury.get('damage', -1), budget.get('damage', -1)
    hp, maximum, initial = status.get('hp', 0), status.get('hpMax', 0), budget.get('hp', 0)
    if (type(seq) is not int or type(baseline) is not int or seq < baseline or
            injury.get('complete') is not True or injury.get('seq') != seq or
            not math.isfinite(damage) or not math.isfinite(initial_damage) or not 0 <= initial_damage <= damage <= initial_damage + 10 or
            not all(math.isfinite(v) for v in (hp, maximum, initial)) or maximum <= 0 or initial < 30 or
            hp < max(30, maximum * .65, initial - 10)):
        return False
    task, pending = status.get('task') or {}, goal.get('pending') or {}

    def travel(intent):
        args = intent.get('params') or {}
        return (bool(intent.get('run')) and bool(intent.get('taskId')) and
                intent.get('template') in ('search_food', 'food_return', 'cook_return', 'home') and
                args.get('recoverThreats') is True and args.get('dry') is True and args.get('avoidHostiles') is True)

    journey = status.get('journey') or {}
    if (task.get('id') == pending.get('taskId') and task.get('task') == 'move_to' and task.get('status') == 'running' and
            journey.get('id') == task.get('id') and journey.get('mode') in ('walk', 'defend', 'escape', 'observe', 'blocked') and
            type(journey.get('attempts')) is int and 0 <= journey['attempts'] <= 3 and travel(pending)):
        return True
    # A slow poll may see the next task. Require the previous exact travel
    # intent, its successful same-run outcome, and acknowledged injury count.
    old = (previous_goal or {}).get('pending') or {}
    prior_task = (previous or {}).get('task') or {}
    return (bool(previous_goal) and previous_goal.get('id') == goal.get('id') and travel(old) and
            prior_task.get('id') == old.get('taskId') and prior_task.get('task') == 'move_to' and
            food.get('injury') == seq and (not task or task.get('id') == pending.get('taskId')) and
            any(r.get('run') == old['run'] and r.get('taskId') == old['taskId'] and
                r.get('template') == old['template'] and r.get('status') == 'succeeded'
                for r in goal.get('history', [])))


def safe(status, previous=None, goal=None, previous_goal=None):
    hp, maximum = status.get('hp', 0), status.get('hpMax', 0)
    if not status.get('connected') or status.get('phase') != 'in_world': return 'not in world'
    if not math.isfinite(hp) or not math.isfinite(maximum) or maximum <= 0 or hp < max(20, maximum * .4): return 'unsafe health'
    if (status.get('location') or {}).get('inWater') or 'swimming' in status.get('flags', []): return 'unsafe water'
    if previous and (status.get('world'), status.get('connection')) != (previous.get('world'), previous.get('connection')):
        return 'scope changed'
    if previous and health_loss(previous, status) and not food_travel_damage(status, goal, previous, previous_goal):
        return 'injury outside owned food travel budget'
    return None


def safe_to_remain(status, mapped):
    home, perception = status.get('home') or {}, mapped.get('perception')
    if (safe(status) or status.get('task') or 'in_shelter' not in status.get('flags', [])
            or home.get('world') != status.get('world') or not home.get('pos')
            or math.dist(status['pos'], home['pos']) > 8 or status['hp'] < status['hpMax'] * .8
            or abs(status['pos'][1] - home['pos'][1]) > 1.75
            or (mapped.get('world'), mapped.get('connection')) != (status.get('world'), status.get('connection'))
            or not isinstance(perception, dict) or 'characters' not in perception): return False
    return not any(c.get('hpFrac', 0) > 0 and (c.get('hostile') or c.get('targetingMe'))
                   and math.dist(c['pos'], status['pos']) < 20 for c in perception['characters'])


def completed_or_returned_shortage(summary, goal):
    """A returned source shortage stays a failed proof, but need not restart a safe client."""
    if summary.get('outcome') == 'succeeded': return True
    reason = 'household source markers exhausted or unsafe; wait for regrowth or revise sources'
    return (bool(goal) and summary.get('status') == goal.get('status') == 'needs_plan'
            and summary.get('reason') == goal.get('reason') == reason and not goal.get('pending')
            and any(row.get('template') == 'food_return' and row.get('status') == 'succeeded'
                    for row in goal.get('history', [])))


def goal_owned_task(status, goal):
    """A sampled task can have completed between the status and goal reads."""
    active = (status.get('task') or {}).get('id')
    if not active or active == (goal.get('pending') or {}).get('taskId'): return True
    return any(row.get('taskId') == active and row.get('status') in ('succeeded', 'failed', 'rejected', 'cancelled')
               for row in goal.get('history', []))


def escort_active(status, intent, leader):
    body, escort = status.get('task') or {}, status.get('escort') or {}
    return (bool(intent.get('run')) and bool(intent.get('taskId')) and
            body.get('id') == intent['taskId'] and body.get('task') == 'escort' and
            escort.get('id') == intent['taskId'] and escort.get('leader') == leader and
            escort.get('mode') in ('follow', 'wait'))


def escort_receipt(history, intent, leader):
    rows = history.get('value', []) if isinstance(history, dict) else history
    frames = [r.get('frame', {}) for r in rows]
    requested = any(f.get('type') == 'task' and f.get('id') == intent.get('taskId') and
                    f.get('task') == 'escort' and f.get('args', {}).get('player') == leader and
                    f.get('args', {}).get('defend') is False and f.get('args', {}).get('collect') is False
                    for f in frames)
    outcomes = {f.get('status') for f in frames if f.get('type') == 'task' and f.get('re') == intent.get('taskId')}
    return bool(intent.get('run')) and bool(intent.get('taskId')) and requested and {'accepted', 'running'} <= outcomes


def household_definition(spec, marker):
    return dict(template='household_food', params=spec,
                trigger=dict(kind='idle', marker=dict(world=spec['world'], name=marker, kind='storage'), dwellS=30),
                repeat=dict(mode='trigger', cooldownS=300, maxRuns=1), priority=-10, enabled=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--stock-target', action='append', default=[], help='household food prefab=count; repeat for several foods')
    parser.add_argument('--sources', default='', help='comma-separated food source marker names')
    parser.add_argument('--idle-marker', help='start a household episode after idle dwell at this storage marker')
    parser.add_argument('--require-deposit', action='store_true')
    parser.add_argument('--join', action='store_true')
    parser.add_argument('--storage', default='', help='comma-separated allowed current home chest IDs; default all eligible chests')
    parser.add_argument('--reserve', type=int, choices=range(1, 4), default=2)
    parser.add_argument('--require-cooking', action='store_true')
    parser.add_argument('--require-fuel', action='store_true')
    parser.add_argument('--require-acquisition', action='store_true', help='require verified collection outside storage')
    parser.add_argument('--keep-online', action='store_true', help='keep Tas online after success or a returned source shortage, with verified safe idle shelter on the home floor')
    parser.add_argument('--stow-ready', help='stage a shortage by normally depositing carried ready meals into this excluded chest')
    parser.add_argument('--escort', help='verify a bounded food-to-escort handoff for this commander, then stop escort at home')
    args = parser.parse_args()
    targets = {}
    for target in args.stock_target:
        item, separator, count = target.partition('=')
        if not separator or not count.isdigit() or not 1 <= int(count) <= 20 or item in targets:
            parser.error('stock targets require unique prefab=count pairs with counts1–20')
        targets[item] = int(count)
    if (targets and (not args.storage or args.escort)) or (args.idle_marker and not targets):
        parser.error('household work needs targets and explicit storage; escort is separate')
    folder = ROOT / '.checks' / ('food-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
    folder.mkdir(parents=True)
    log = (folder / 'commands.log').open('a')
    summary = dict(outcome='failed', reason='not started', evidence=str(folder))
    launch, owns_game, previous, goal = None, False, None, None
    idle_name = None
    deadline = time.monotonic() + 720

    def call(*arguments, decode=True):
        result = subprocess.run(['just', 'vm', *arguments], cwd=ROOT, capture_output=True, text=True, timeout=45)
        log.write(result.stdout + result.stderr); log.flush()
        if result.returncode: raise RuntimeError('VM request failed: ' + str(arguments[:3]))
        return json.loads(result.stdout) if decode else result.stdout

    def observe():
        nonlocal previous, goal
        status = call('panel', 'GET', '/api/status')
        with (folder / 'observations.jsonl').open('a') as out:
            out.write(json.dumps(dict(at=datetime.datetime.now(datetime.timezone.utc).isoformat(), status=status)) + '\n')
        previous_goal = goal
        if previous and health_loss(previous, status) and goal:
            goal = call('panel', 'GET', '/api/goals?' + urlencode(dict(id=goal['id'])))
        reason = safe(status, previous, goal, previous_goal)
        if reason: raise RuntimeError(reason)
        if time.monotonic() > deadline: raise RuntimeError('food proof deadline')
        previous = status
        return status

    try:
        status = call('panel', 'GET', '/api/status')
        if args.join:
            if status.get('connected'): raise RuntimeError('--join requires offline bot')
            if 'RecoveryGuard = true' not in call('recovery', decode=False): raise RuntimeError('native recovery guard must be enabled')
            launch = subprocess.Popen(['just', 'vm', 'start', '2'], cwd=ROOT, stdout=log, stderr=log)
            owns_game = True
            until = time.monotonic() + 120
            while time.monotonic() < until:
                status = call('panel', 'GET', '/api/status')
                if status.get('phase') == 'in_world': break
                time.sleep(1)
            else: raise RuntimeError('join deadline')
        if status.get('task'): raise RuntimeError('initial body owned; no work replaced')
        owns_game = True
        status = observe()
        while 'recovering' in status.get('flags', []):
            time.sleep(1); status = observe()
        home = status.get('home') or {}
        if home.get('world') != status.get('world') or not home.get('pos'): raise RuntimeError('known home required')
        mapped = call('panel', 'GET', '/api/map')
        (folder / 'before-map.json').write_text(json.dumps(mapped, indent=2)+'\n')
        (folder / 'before-pack.json').write_text(json.dumps(call('panel', 'GET', '/api/inventory'), indent=2)+'\n')
        if args.stow_ready:
            if not args.storage or args.stow_ready in args.storage.split(','):
                raise RuntimeError('staging destination must be excluded from an explicit search scope')
            foods = ('CookedDeerMeat', 'NeckTailGrilled', 'CookedMeat', 'Honey', 'Raspberry', 'Blueberries', 'Mushroom', 'MushroomYellow')
            before = call('panel', 'GET', '/api/inventory')['inventory']
            items = [dict(prefab=p, count='all') for p in foods if any(i.get('prefab') == p for i in before['items'])]
            if items:
                anchor = datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z')
                request = call('task', 'deposit', json.dumps(dict(container=args.stow_ready, items=items)), '45')
                until = time.monotonic() + 50
                while time.monotonic() < until:
                    status = observe()
                    history = call('panel', 'GET', '/api/task?' + urlencode(dict(id=request['id'], at=anchor)))
                    (folder / 'staging.json').write_text(json.dumps(history, indent=2)+'\n')
                    rows = history.get('value', []) if isinstance(history, dict) else history
                    result = next((r['frame'] for r in rows if r.get('frame', {}).get('re') == request['id'] and r['frame'].get('status') in ('succeeded', 'failed', 'rejected', 'cancelled')), None)
                    if result and not status.get('task'): break
                    if status.get('task') and status['task']['id'] != request['id']: raise RuntimeError('staging lost body control')
                    time.sleep(1)
                else: raise RuntimeError('staging transfer deadline')
                receipt = next((r['frame'] for r in rows if r.get('frame', {}).get('type') == 'container' and r['frame'].get('task') == request['id']
                                and r['frame'].get('direction') == 'deposit' and r['frame'].get('entity') == args.stow_ready), None)
                after = call('panel', 'GET', '/api/inventory')['inventory']
                if result['status'] != 'succeeded' or not receipt or after.get('seq', 0) <= before.get('seq', 0): raise RuntimeError('staging lacks transfer/inventory receipts')
                for item in items:
                    prefab = item['prefab']
                    n = sum(i['stack'] for i in before['items'] if i['prefab'] == prefab)
                    if receipt.get('moved', {}).get(prefab) != n or any(i['prefab'] == prefab for i in after['items']): raise RuntimeError('staging ready food was not fully stowed')
                summary['staged'] = receipt['moved']
        spec = dict(kind='food_resupply', world=status['world'], center=home['pos'], radius=150, reserve=args.reserve)
        if args.escort:
            commanders = list(status.get('owners') or []) + [name for name, role in
                (status.get('players') or {}).get('profiles', {}).items() if role == 'operator']
            if args.escort not in commanders or args.reserve < 2:
                raise RuntimeError('handoff check needs a current commander and reserve of at least two')
            spec['escort'] = dict(player=args.escort, collect=False, defend=False, resupply=True)
        if args.storage: spec['storage'] = args.storage.split(',')
        if targets:
            spec['radius'] = 200
            spec['stock'] = dict(targets=targets, sources=[s.strip() for s in args.sources.split(',') if s.strip()])
        if args.idle_marker:
            candidate_name = 'household check ' + folder.name
            definition = household_definition(spec, args.idle_marker)
            call('panel', 'POST', '/api/behaviours', json.dumps(dict(action='save', name=candidate_name, definition=definition)))
            idle_name = candidate_name
            until = time.monotonic() + 90
            while time.monotonic() < until:
                status = observe()
                instances = call('panel', 'GET', '/api/behaviours')
                instances = instances.get('value', []) if isinstance(instances, dict) else instances
                instance = next(r for r in instances if r['name'] == idle_name)
                (folder / 'behaviour.json').write_text(json.dumps(instance, indent=2))
                if instance.get('episodeId'):
                    goal = call('panel', 'GET', '/api/goals?' + urlencode(dict(id=instance['episodeId'])))
                    break
                if instance.get('runs') or instance.get('last') == 'new order':
                    raise RuntimeError('household idle episode did not start: ' + str(instance.get('last')))
                time.sleep(2)
            else:
                raise RuntimeError('household idle admission deadline')
        else:
            goal = call('panel', 'POST', '/api/goals', json.dumps(dict(action='start', spec=spec)))
        summary['goal'] = goal['id']
        while True:
            status = observe()
            goal = call('panel', 'GET', '/api/goals?' + urlencode(dict(id=goal['id'])))
            (folder / 'goal.json').write_text(json.dumps(goal, indent=2)+'\n')
            if goal['status'] in ('succeeded', 'needs_plan', 'cancelled', 'suspended'): break
            if not goal_owned_task(status, goal):
                # Dispatch can also precede persistence of its assigned ID.
                # Refresh both samples once, retaining exact task identities.
                status = observe()
                goal = call('panel', 'GET', '/api/goals?' + urlencode(dict(id=goal['id'])))
                if not goal_owned_task(status, goal): raise RuntimeError('body ownership changed')
            time.sleep(2)
        summary.update(status=goal['status'], tasks=goal['taskCount'], reason=goal.get('reason'), batches=(goal.get('food') or {}).get('batches', 0))
        if goal['status'] != 'succeeded': raise RuntimeError(summary['reason'])
        receipts = goal['food'].get('receipts', [])
        deposits = [r for r in receipts if r['action'] == 'stock_deposit']
        summary['deposits'] = deposits
        summary['stock'] = goal['food'].get('stock')
        if args.require_deposit and not deposits: raise RuntimeError('household goal lacks a verified stock deposit')
        cooks = [r for r in receipts if r['action'] == 'cook']
        acquired = [r for r in receipts if r['action'] == 'collect']
        fuel = []
        for receipt in cooks:
            history = call('panel', 'GET', '/api/task?' + urlencode(dict(id=receipt['taskId'], at=goal['created'])))
            (folder / (receipt['taskId'] + '.json')).write_text(json.dumps(history, indent=2)+'\n')
            rows = history.get('value', []) if isinstance(history, dict) else history
            fuel += [r['frame'] for r in rows if r.get('frame', {}).get('msg') == 'verified wood fuel and station heat receipt']
        summary.update(cooked=cooks, acquired=acquired, fueled=bool(fuel), hp=status['hp'], hpMax=status['hpMax'], pos=status['pos'])
        if args.require_acquisition and not acquired: raise RuntimeError('goal succeeded but requested acquisition evidence is absent')
        if args.require_cooking and not cooks: raise RuntimeError('goal succeeded but requested cooking evidence is absent')
        if args.require_fuel and not fuel: raise RuntimeError('goal succeeded but requested cold-fire fuel evidence is absent')
        if args.escort:
            until = time.monotonic() + 8
            while time.monotonic() < until:
                goal = call('panel', 'GET', '/api/goals?' + urlencode(dict(id=goal['id'])))
                intent = (goal.get('food') or {}).get('continuation') or {}
                if intent.get('taskId'): break
                time.sleep(1)
            else: raise RuntimeError('food goal has no assigned escort continuation')
            summary['tasks'] = goal['taskCount']
            (folder / 'goal.json').write_text(json.dumps(goal, indent=2)+'\n')
            until = time.monotonic() + 8
            while time.monotonic() < until:
                history = call('panel', 'GET', '/api/task?' + urlencode(dict(id=intent['taskId'], at=goal['created'])))
                (folder / 'escort.json').write_text(json.dumps(history, indent=2)+'\n')
                if escort_receipt(history, intent, args.escort): break
                time.sleep(1)
            else: raise RuntimeError('escort handoff lacks matching native request/acceptance/running evidence')
            until, modes = time.monotonic() + 10, set()
            while time.monotonic() < until:
                status = observe()
                if not escort_active(status, intent, args.escort):
                    raise RuntimeError('escort identity, leader or quiet mode changed')
                mapped = call('panel', 'GET', '/api/map')
                if not safe_to_remain(status | dict(task=None), mapped):
                    raise RuntimeError('handoff trial left safe home conditions')
                modes.add(status['escort']['mode']); time.sleep(1)
            call('task', 'stop', '{}', '5')
            until = time.monotonic() + 5
            while time.monotonic() < until:
                status = observe()
                if not status.get('task'): break
                if (status.get('task') or {}).get('id') != intent['taskId']:
                    raise RuntimeError('new owner after escort stop')
                time.sleep(1)
            else: raise RuntimeError('escort stop not observed')
            summary['escortHandoff'] = dict(taskId=intent['taskId'], leader=args.escort, modes=sorted(modes), stopped=True)
        summary.update(outcome='succeeded', reason='household food targets verified' if targets else 'production food goal verified reserve, active food and recovery')
    except Exception as error:
        summary['reason'] = str(error)
    finally:
        if idle_name:
            try: call('panel', 'POST', '/api/behaviours', json.dumps(dict(action='disable', name=idle_name)))
            except Exception as error: summary.update(outcome='failed', reason='cannot disable household test definition: ' + str(error))
        retained = False
        if owns_game and args.keep_online and completed_or_returned_shortage(summary, goal):
            try:
                final = observe()
                mapped = call('panel', 'GET', '/api/map')
                retained = safe_to_remain(final, mapped)
            except Exception: retained = False
        summary['keptOnline'] = retained
        if owns_game and not retained:
            if goal and goal.get('status') not in ('succeeded', 'cancelled'):
                try: call('panel', 'POST', '/api/goals', json.dumps(dict(action='stop', id=goal['id'])))
                except Exception as error: summary['reason'] += '; goal cleanup: ' + str(error)
            try: call('panel', 'POST', '/api/act/logout', '{}')
            except Exception as error: summary['reason'] += '; logout: ' + str(error)
            if launch and launch.poll() is None:
                launch.terminate()
                try: launch.wait(timeout=5)
                except subprocess.TimeoutExpired: launch.kill(); launch.wait()
            for arguments in [('stop',), ('panel-start',)]:
                try: call(*arguments, decode=False)
                except Exception as error: summary['reason'] += '; cleanup: ' + str(error)
        (folder / 'summary.json').write_text(json.dumps(summary, indent=2)+'\n')
        print(json.dumps(summary), flush=True); log.close()
    return 0 if summary['outcome'] == 'succeeded' else 1


if __name__ == '__main__':
    raise SystemExit(main())
