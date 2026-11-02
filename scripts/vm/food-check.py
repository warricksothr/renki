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


def safe(status, previous=None):
    hp, maximum = status.get('hp', 0), status.get('hpMax', 0)
    if not status.get('connected') or status.get('phase') != 'in_world': return 'not in world'
    if not math.isfinite(hp) or not math.isfinite(maximum) or maximum <= 0 or hp < max(20, maximum * .4): return 'unsafe health'
    if (status.get('location') or {}).get('inWater') or 'swimming' in status.get('flags', []): return 'unsafe water'
    if previous and ((status.get('world'), status.get('connection')) != (previous.get('world'), previous.get('connection')) or health_loss(previous, status)):
        return 'scope changed or actual injury'
    return None


def safe_to_remain(status, mapped):
    home, perception = status.get('home') or {}, mapped.get('perception')
    if (safe(status) or status.get('task') or 'in_shelter' not in status.get('flags', [])
            or home.get('world') != status.get('world') or not home.get('pos')
            or math.dist(status['pos'], home['pos']) > 8 or status['hp'] < status['hpMax'] * .8
            or (mapped.get('world'), mapped.get('connection')) != (status.get('world'), status.get('connection'))
            or not isinstance(perception, dict) or 'characters' not in perception): return False
    return not any(c.get('hpFrac', 0) > 0 and (c.get('hostile') or c.get('targetingMe'))
                   and math.dist(c['pos'], status['pos']) < 20 for c in perception['characters'])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--join', action='store_true')
    parser.add_argument('--storage', default='', help='comma-separated allowed current home chest IDs; default all eligible chests')
    parser.add_argument('--reserve', type=int, choices=range(1, 4), default=2)
    parser.add_argument('--require-cooking', action='store_true')
    parser.add_argument('--require-fuel', action='store_true')
    parser.add_argument('--keep-online', action='store_true', help='keep Tas online only after success and verified safe idle shelter near home')
    parser.add_argument('--stow-ready', help='stage a shortage by normally depositing carried ready meals into this excluded chest')
    args = parser.parse_args()
    folder = ROOT / '.checks' / ('food-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
    folder.mkdir(parents=True)
    log = (folder / 'commands.log').open('a')
    summary = dict(outcome='failed', reason='not started', evidence=str(folder))
    launch, owns_game, previous, goal = None, False, None, None
    deadline = time.monotonic() + 720

    def call(*arguments, decode=True):
        result = subprocess.run(['just', 'vm', *arguments], cwd=ROOT, capture_output=True, text=True, timeout=45)
        log.write(result.stdout + result.stderr); log.flush()
        if result.returncode: raise RuntimeError('VM request failed: ' + str(arguments[:3]))
        return json.loads(result.stdout) if decode else result.stdout

    def observe():
        nonlocal previous
        status = call('panel', 'GET', '/api/status')
        with (folder / 'observations.jsonl').open('a') as out:
            out.write(json.dumps(dict(at=datetime.datetime.now(datetime.timezone.utc).isoformat(), status=status)) + '\n')
        reason = safe(status, previous)
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
            foods = ('CookedDeerMeat', 'NeckTailGrilled', 'CookedMeat', 'Honey', 'Raspberry', 'Blueberries', 'Mushroom')
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
        if args.storage: spec['storage'] = args.storage.split(',')
        goal = call('panel', 'POST', '/api/goals', json.dumps(dict(action='start', spec=spec)))
        summary['goal'] = goal['id']
        while True:
            status = observe()
            goal = call('panel', 'GET', '/api/goals?' + urlencode(dict(id=goal['id'])))
            (folder / 'goal.json').write_text(json.dumps(goal, indent=2)+'\n')
            if goal['status'] in ('succeeded', 'needs_plan', 'cancelled', 'suspended'): break
            active = (status.get('task') or {}).get('id')
            if active and active != (goal.get('pending') or {}).get('taskId'):
                # Avoid misclassifying receipt-to-next-step races; confirm once.
                goal = call('panel', 'GET', '/api/goals?' + urlencode(dict(id=goal['id'])))
                if active != (goal.get('pending') or {}).get('taskId'): raise RuntimeError('body ownership changed')
            time.sleep(2)
        summary.update(status=goal['status'], tasks=goal['taskCount'], reason=goal.get('reason'), batches=(goal.get('food') or {}).get('batches', 0))
        if goal['status'] != 'succeeded': raise RuntimeError(summary['reason'])
        receipts = goal['food'].get('receipts', [])
        cooks = [r for r in receipts if r['action'] == 'cook']
        fuel = []
        for receipt in cooks:
            history = call('panel', 'GET', '/api/task?' + urlencode(dict(id=receipt['taskId'], at=goal['created'])))
            (folder / (receipt['taskId'] + '.json')).write_text(json.dumps(history, indent=2)+'\n')
            rows = history.get('value', []) if isinstance(history, dict) else history
            fuel += [r['frame'] for r in rows if r.get('frame', {}).get('msg') == 'verified wood fuel and station heat receipt']
        summary.update(cooked=cooks, fueled=bool(fuel), hp=status['hp'], hpMax=status['hpMax'], pos=status['pos'])
        if args.require_cooking and not cooks: raise RuntimeError('goal succeeded but requested cooking evidence is absent')
        if args.require_fuel and not fuel: raise RuntimeError('goal succeeded but requested cold-fire fuel evidence is absent')
        summary.update(outcome='succeeded', reason='production food goal verified reserve, active food and recovery')
    except Exception as error:
        summary['reason'] = str(error)
    finally:
        retained = False
        if owns_game and args.keep_online and summary['outcome'] == 'succeeded':
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
