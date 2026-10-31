#!/usr/bin/env python3
"""Bounded native cooking proof with optional guarded login and home supply search."""
import argparse
import datetime
import json
import math
from pathlib import Path
import subprocess
import time
from urllib.parse import urlencode
from harvest_trial import health_loss
from resupply_trial import candidates, count, FOODS

ROOT = Path(__file__).resolve().parents[2]
RAW = ('RawMeat', 'DeerMeat', 'NeckTail')


def terminal(history, task_id):
    rows = history.get('value', []) if isinstance(history, dict) else history
    return next((r['frame'] for r in reversed(rows) if r.get('frame', {}).get('type') == 'task'
                 and r['frame'].get('re') == task_id and r['frame'].get('status') in
                 ('succeeded', 'failed', 'rejected', 'cancelled')), None)


def delivery_receipt(receipts):
    """Require matching identities/counts; keep remote correlation explicit."""
    finals = [r.get('data', {}) for r in receipts if r.get('msg') == 'verified cooked inventory receipt']
    requests = [r.get('data', {}) for r in receipts if r.get('msg') == 'normal cooking removal request observed']
    if len(finals) != 1 or len(requests) != 1: return None
    final, request = finals[0], requests[0]
    mode, count = final.get('evidence'), final.get('count', 0)
    message = {'direct_spawn': 'normal station output spawned',
               'remote_correlation': 'correlated remote station output observed'}.get(mode)
    if not message or not 1 <= count <= 10 or request.get('requestedCount') != count or request.get('evidence') != mode: return None
    outputs = [r.get('data', {}) for r in receipts if r.get('msg') == message]
    drops = {}
    for output in outputs:
        entity = output.get('entity')
        if not entity or entity in drops or output.get('output') != final.get('output') or output.get('count', 0) <= 0: return None
        drops[entity] = output['count']
    picked = {entity: 0 for entity in drops}
    for row in receipts:
        if row.get('msg') != 'verified cooking drop pickup': continue
        data = row.get('data', {}); entity = data.get('entity')
        if entity not in drops or data.get('output') != final.get('output') or data.get('count', 0) <= 0: return None
        picked[entity] += data['count']
    return dict(evidence=mode, output=final.get('output'), count=count) if sum(drops.values()) == count and picked == drops else None


def safe(status, previous=None, require_food=True):
    hp, maximum = status.get('hp', 0), status.get('hpMax', 0)
    if not status.get('connected') or status.get('phase') != 'in_world': return 'not in world'
    if not math.isfinite(hp) or not math.isfinite(maximum) or maximum <= 0 or hp < max(20, maximum * .4): return 'unsafe health'
    if require_food and not (status.get('food') or {}).get('active'): return 'no active food'
    if (status.get('location') or {}).get('inWater') or 'swimming' in status.get('flags', []): return 'unsafe water'
    if previous and ((status.get('world'), status.get('connection')) != (previous.get('world'), previous.get('connection'))
                     or health_loss(previous, status)): return 'scope changed or actual injury'
    return None


def choose_station(status, mapped, wood=False):
    if (status.get('world'), status.get('connection')) != (mapped.get('world'), mapped.get('connection')): return None
    choices = []
    for s in (mapped.get('perception') or {}).get('structures', []):
        c = s.get('cooking') or {}
        slots = c.get('slots') or []
        if (s.get('id') and s.get('kind') == 'station' and s.get('pos') and c.get('supported') and (c.get('heated') or wood)
                and 0 < len(slots) <= 16 and all(not x.get('item') and x.get('state') == 'empty' for x in slots)):
            distance = math.dist(status['pos'], s['pos'])
            if distance <= 12: choices.append((distance, s['id']))
    return min(choices)[1] if choices else None


def search_chests(status, mapped):
    live = {s['id'] for s in (mapped.get('perception') or {}).get('structures', [])
            if s.get('kind') == 'container' and s.get('id')}
    return {c[2] for c in candidates(status, mapped, {}, foods=RAW)} & live


def home_search_ready(status):
    home = status.get('home') or {}
    return bool(home.get('world') == status.get('world') and home.get('pos')
                and math.dist(status['pos'], home['pos']) <= 32)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--join', action='store_true', help='start once with the opt-in native recovery guard already enabled')
    parser.add_argument('--search-home', action='store_true', help='inspect the finite set of unique visible home chests for one raw ingredient')
    parser.add_argument('--refresh-food', action='store_true', help='withdraw one ready meal near home and observe eating/recovery before cooking')
    args = parser.parse_args()
    folder = ROOT / '.checks' / ('cooking-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
    folder.mkdir(parents=True)
    log = (folder / 'commands.log').open('a')
    summary = dict(outcome='failed', reason='not started', tasks=[], checkedChests=0, evidence=str(folder))
    launch, owns_game, previous = None, False, None
    preparing_food = args.refresh_food
    deadline = time.monotonic() + 600

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
        reason = safe(status, previous, require_food=not preparing_food)
        if reason: raise RuntimeError(reason)
        if time.monotonic() > deadline: raise RuntimeError('bounded cooking proof deadline')
        previous = status
        return status

    def task(name, parameters, seconds):
        status = observe()
        if status.get('task'): raise RuntimeError('another body owns control')
        anchor = datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z')
        task_id = call('task', name, json.dumps(parameters), str(seconds))['id']
        summary['tasks'].append(dict(id=task_id, task=name))
        until = min(deadline, time.monotonic() + seconds + 5)
        while time.monotonic() < until:
            history = call('panel', 'GET', '/api/task?' + urlencode(dict(id=task_id, at=anchor)))
            (folder / (task_id + '.json')).write_text(json.dumps(history, indent=2) + '\n')
            result = terminal(history, task_id)
            status = observe()
            active = (status.get('task') or {}).get('id')
            if active and active != task_id: raise RuntimeError('body ownership changed')
            if result and not active:
                summary['tasks'][-1]['result'] = result
                return result, history
            time.sleep(1)
        raise RuntimeError('native cooking/supply task deadline')

    try:
        initial = call('panel', 'GET', '/api/status')
        if args.join:
            if initial.get('connected'): raise RuntimeError('--join requires an offline bot')
            if 'RecoveryGuard = true' not in call('recovery', decode=False): raise RuntimeError('enable the native recovery guard before joining')
            launch = subprocess.Popen(['just', 'vm', 'start', '2'], cwd=ROOT, stdout=log, stderr=log)
            owns_game = True
            join_until = time.monotonic() + 120
            connecting = False
            while time.monotonic() < join_until:
                initial = call('panel', 'GET', '/api/status')
                connecting = connecting or initial.get('phase') == 'connecting'
                if initial.get('phase') == 'in_world': break
                if connecting and initial.get('phase') == 'menu': raise RuntimeError('native recovery declined the saved location')
                time.sleep(1)
            else: raise RuntimeError('join deadline reached')
        if initial.get('task'): raise RuntimeError('initial body already owned; no tasks replaced')
        owns_game = True
        status = observe()
        # Trust the native hold flag, not a floating comparison against rounded HP.
        while 'recovering' in status.get('flags', []):
            time.sleep(1); status = observe()
        if status['hp'] < status['hpMax'] * .8: raise RuntimeError('recover to 80% health before cooking/supply work')
        scope = (status['world'], status.get('connection'))

        def pack():
            payload = call('panel', 'GET', '/api/inventory')
            if (payload.get('world'), payload.get('connection')) != scope or not payload.get('inventory'):
                raise RuntimeError('scoped inventory unavailable')
            return payload['inventory']

        if args.refresh_food:
            if not home_search_ready(status): raise RuntimeError('ready-food preparation requires Tas near known home')
            mapped = call('panel', 'GET', '/api/map')
            live = {s['id'] for s in (mapped.get('perception') or {}).get('structures', [])
                    if s.get('kind') == 'container' and s.get('id')}
            eligible = {c[2] for c in candidates(status, mapped, {}, foods=FOODS)} & live
            opened, attempted, unavailable = {}, set(), set()
            meal = None
            while not meal:
                status = observe(); mapped = call('panel', 'GET', '/api/map')
                options = [c for c in candidates(status, mapped, {}, opened, foods=FOODS)
                           if c[2] in eligible and (c[2], c[3]) not in attempted and c[2] not in unavailable]
                if not options: raise RuntimeError('no verified ready meal in eligible home chests')
                _, _, chest, prefab, _ = options[0]; attempted.add((chest, prefab))
                before = pack()
                result, history = task('withdraw', dict(container=chest, items=[dict(prefab=prefab, count=1)]), 45)
                rows = history.get('value', []) if isinstance(history, dict) else history
                receipt = next((r['frame'] for r in rows if r.get('frame', {}).get('type') == 'container'
                                and r['frame'].get('task') == summary['tasks'][-1]['id'] and r['frame'].get('entity') == chest
                                and r['frame'].get('direction') == 'withdraw'), None)
                if receipt: opened[chest] = receipt
                else: unavailable.add(chest)
                if (receipt or {}).get('moved', {}).get(prefab, 0) == 1:
                    after = pack(); status = observe()
                    active = {f['prefab'] for f in (status.get('food') or {}).get('active', [])}
                    gain = count(after, prefab) - count(before, prefab)
                    if after.get('seq', 0) <= before.get('seq', 0) or gain not in (0, 1) or gain == 0 and prefab not in active:
                        raise RuntimeError('ready meal lacks matched transfer/inventory/food receipt')
                    meal = prefab; summary['foodReceipt'] = receipt
                elif result['status'] == 'succeeded': raise RuntimeError('ready meal transfer succeeded without its receipt')
            until = min(deadline, time.monotonic() + 180)
            while True:
                status = observe()
                active = (status.get('food') or {}).get('active', [])
                if any(f['prefab'] == meal and f.get('remainingS', 0) >= 120 for f in active) and status['hp'] >= status['hpMax'] * .8: break
                if time.monotonic() >= until: raise RuntimeError('ready meal eating/recovery was not observed')
                if status.get('task'): raise RuntimeError('body ownership changed during meal recovery')
                time.sleep(1)
            preparing_food = False
            summary['foodPreparation'] = dict(prefab=meal, hp=status['hp'], hpMax=status['hpMax'])

        inventory = pack()
        raw = next((p for p in RAW if count(inventory, p) > 0), None)
        opened, attempted, unavailable = {}, set(), set()
        if not raw and args.search_home:
            mapped = call('panel', 'GET', '/api/map')
            if not home_search_ready(status):
                raise RuntimeError('ingredient search requires Tas already near known home')
            eligible = search_chests(status, mapped)
            summary['searchChests'] = len(eligible)
            while not raw:
                status = observe(); mapped = call('panel', 'GET', '/api/map')
                options = [c for c in candidates(status, mapped, {}, opened, foods=RAW)
                           if c[2] in eligible and (c[2], c[3]) not in attempted and c[2] not in unavailable]
                if not options: break
                _, _, chest, prefab, _ = options[0]
                attempted.add((chest, prefab)); before = pack()
                result, history = task('withdraw', dict(container=chest, items=[dict(prefab=prefab, count=1)]), 45)
                rows = history.get('value', []) if isinstance(history, dict) else history
                receipt = next((r['frame'] for r in rows if r.get('frame', {}).get('type') == 'container'
                                and r['frame'].get('task') == summary['tasks'][-1]['id'] and r['frame'].get('entity') == chest
                                and r['frame'].get('direction') == 'withdraw'), None)
                if receipt:
                    opened[chest] = receipt; summary['checkedChests'] = len(opened)
                else: unavailable.add(chest)
                after = pack(); moved = (receipt or {}).get('moved', {}).get(prefab, 0)
                if moved:
                    if moved != 1 or after.get('seq', 0) <= before.get('seq', 0) or count(after, prefab) != count(before, prefab) + 1:
                        raise RuntimeError('ingredient transfer lacks a matched inventory receipt')
                    raw = prefab; summary['ingredientReceipt'] = receipt
                elif result['status'] == 'succeeded': raise RuntimeError('succeeded transfer without ingredient receipt')
        if not raw: raise RuntimeError('no verified raw ingredient in pack or eligible home chests')
        status = observe(); mapped = call('panel', 'GET', '/api/map')
        station = choose_station(status, mapped, wood=count(pack(), 'Wood') > 0)
        (folder / 'before-map.json').write_text(json.dumps(mapped, indent=2)+'\n')
        if not station: raise RuntimeError('no fresh empty station with heat or carried wood within 12 m')
        summary.update(input=raw, station=station)
        result, history = task('interact', dict(entity=station, cook=raw), 120)
        rows = history.get('value', []) if isinstance(history, dict) else history
        receipts = [r['frame'] for r in rows if r.get('frame', {}).get('type') == 'trace'
                    and r['frame'].get('source') == 'cooking' and (r['frame'].get('data') or {}).get('id') == summary['tasks'][-1]['id']]
        summary['cookingReceipts'] = receipts
        if result['status'] != 'succeeded':
            raise RuntimeError('native cooking ' + result['status'] + ' (' + result.get('reason', '') + '): ' + result.get('detail', ''))
        delivery = delivery_receipt(receipts)
        if not delivery:
            raise RuntimeError('cooking lacks succeeded native status and matched spawn/inventory receipts')
        summary.update(outcome='succeeded', reason='one input cooked and its full native output collected', delivery=delivery)
    except Exception as error:
        summary['reason'] = str(error)
    finally:
        if owns_game:
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
        print(json.dumps({k: v for k, v in summary.items() if k not in ('tasks', 'ingredientReceipt', 'foodReceipt', 'cookingReceipts')} |
                         dict(taskCount=len(summary['tasks']))), flush=True); log.close()
    return 0 if summary['outcome'] == 'succeeded' else 1


if __name__ == '__main__':
    raise SystemExit(main())
