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
from resupply_trial import candidates, count

ROOT = Path(__file__).resolve().parents[2]
RAW = ('RawMeat', 'DeerMeat', 'NeckTail')


def terminal(history, task_id):
    rows = history.get('value', []) if isinstance(history, dict) else history
    return next((r['frame'] for r in reversed(rows) if r.get('frame', {}).get('type') == 'task'
                 and r['frame'].get('id') == task_id and r['frame'].get('status') in
                 ('succeeded', 'failed', 'rejected', 'cancelled')), None)


def safe(status, previous=None):
    hp, maximum = status.get('hp', 0), status.get('hpMax', 0)
    if not status.get('connected') or status.get('phase') != 'in_world': return 'not in world'
    if not math.isfinite(hp) or not math.isfinite(maximum) or maximum <= 0 or hp < max(20, maximum * .4): return 'unsafe health'
    if not (status.get('food') or {}).get('active'): return 'no active food'
    if (status.get('location') or {}).get('inWater') or 'swimming' in status.get('flags', []): return 'unsafe water'
    if previous and ((status.get('world'), status.get('connection')) != (previous.get('world'), previous.get('connection'))
                     or health_loss(previous, status)): return 'scope changed or actual injury'
    return None


def choose_station(status, mapped):
    if (status.get('world'), status.get('connection')) != (mapped.get('world'), mapped.get('connection')): return None
    choices = []
    for s in (mapped.get('perception') or {}).get('structures', []):
        c = s.get('cooking') or {}
        slots = c.get('slots') or []
        if (s.get('id') and s.get('kind') == 'station' and s.get('pos') and c.get('supported') and c.get('heated')
                and 0 < len(slots) <= 16 and all(not x.get('item') and x.get('state') == 'empty' for x in slots)):
            distance = math.dist(status['pos'], s['pos'])
            if distance <= 12: choices.append((distance, s['id']))
    return min(choices)[1] if choices else None


def search_chests(status, mapped):
    live = {s['id'] for s in (mapped.get('perception') or {}).get('structures', [])
            if s.get('kind') == 'container' and s.get('id')}
    return {c[2] for c in candidates(status, mapped, {}, foods=RAW)} & live


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--join', action='store_true', help='start once with the opt-in native recovery guard already enabled')
    parser.add_argument('--search-home', action='store_true', help='inspect the finite set of unique visible home chests for one raw ingredient')
    args = parser.parse_args()
    folder = ROOT / '.checks' / ('cooking-' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ'))
    folder.mkdir(parents=True)
    log = (folder / 'commands.log').open('a')
    summary = dict(outcome='failed', reason='not started', tasks=[], checkedChests=0, evidence=str(folder))
    launch, owns_game, previous = None, False, None
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
        reason = safe(status, previous)
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

        inventory = pack()
        raw = next((p for p in RAW if count(inventory, p) > 0), None)
        opened, attempted, unavailable = {}, set(), set()
        if not raw and args.search_home:
            mapped = call('panel', 'GET', '/api/map')
            home = status.get('home') or {}
            if home.get('world') != status['world'] or not home.get('pos') or math.dist(status['pos'], home['pos']) > 20:
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
        station = choose_station(status, mapped)
        (folder / 'before-map.json').write_text(json.dumps(mapped, indent=2)+'\n')
        if not station: raise RuntimeError('no fresh empty heated station within 12 m')
        summary.update(input=raw, station=station)
        result, history = task('interact', dict(entity=station, cook=raw), 120)
        rows = history.get('value', []) if isinstance(history, dict) else history
        receipts = [r['frame'] for r in rows if r.get('frame', {}).get('type') == 'trace'
                    and r['frame'].get('source') == 'cooking' and (r['frame'].get('data') or {}).get('id') == summary['tasks'][-1]['id']]
        summary['cookingReceipts'] = receipts
        if result['status'] != 'succeeded' or not any(r.get('msg') == 'normal station output spawned' for r in receipts) or not any(
                r.get('msg') == 'verified cooking drop pickup' for r in receipts) or not any(
                r.get('msg') == 'verified cooked inventory receipt' for r in receipts):
            raise RuntimeError('cooking lacks succeeded native status and matched spawn/inventory receipts')
        summary.update(outcome='succeeded', reason='one input cooked and its full native output collected')
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
        print(json.dumps({k: v for k, v in summary.items() if k not in ('tasks', 'ingredientReceipt', 'cookingReceipts')} |
                         dict(taskCount=len(summary['tasks']))), flush=True); log.close()
    return 0 if summary['outcome'] == 'succeeded' else 1


if __name__ == '__main__':
    raise SystemExit(main())
