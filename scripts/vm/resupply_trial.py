"""Bounded operator resupply through native movement and verified transfers."""
import datetime
import json
import math
import time

# Exact ordinary food prefabs, never inferred from arbitrary item names.
FOODS = ('CookedDeerMeat', 'NeckTailGrilled', 'CookedMeat', 'Honey', 'Raspberry', 'Blueberries', 'Mushroom')


def candidates(status, mapped, supplied):
    home = status.get('home') or {}
    if home.get('world') != status.get('world') or not home.get('pos'):
        raise RuntimeError('no known home in the current world')
    choices = []
    for chest in mapped.get('containers', []):
        if math.dist(chest['pos'], home['pos']) > 32:
            continue
        for item in chest.get('items', []):
            prefab = item.get('prefab')
            if prefab in FOODS and prefab not in supplied and item.get('stack', 0) > 0:
                choices.append((FOODS.index(prefab), math.dist(status['pos'], chest['pos']), chest['entity'], prefab, min(4, item['stack'])))
    return sorted(choices)


def count(pack, prefab):
    return sum(i.get('stack', 0) for i in pack.get('items', []) if i.get('prefab') == prefab)


def home_run(status):
    # Walk long trips, but do not linger outside the shelter on a short,
    # rested final approach. The native task still owns routing and stamina.
    return math.dist(status['pos'], status['home']['pos']) <= 40 and status.get('stamina', 0) >= 30


def run(summary, call, observe, terminal, saved):
    status, mapped = observe()
    home = status.get('home') or {}
    if home.get('world') != status.get('world') or not home.get('pos') or math.dist(status['pos'], home['pos']) > 500:
        raise RuntimeError('resupply requires known home within 500 m in this world')
    scope = (status['world'], status.get('connection'))
    previous_hp = status['hp']
    deadline = time.monotonic() + 420
    summary.update(origin=status['pos'], tasks=[], supplied={})

    def checked_observe():
        nonlocal previous_hp
        st, mp = observe()
        if (st.get('world'), st.get('connection')) != scope or not st.get('connected') or st.get('phase') != 'in_world':
            raise RuntimeError('resupply world/connection changed')
        if st.get('hp', 0) < min(previous_hp, st.get('hpMax', 0)) - .15:
            raise RuntimeError('observed damage during resupply')
        if st.get('location', {}).get('inWater') or 'swimming' in st.get('flags', []):
            raise RuntimeError('unsafe water during resupply')
        if time.monotonic() >= deadline or math.dist(st['pos'], summary['origin']) > 525:
            raise RuntimeError('resupply time/travel limit')
        previous_hp = st['hp']
        return st, mp

    def task(name, args, timeout):
        st, _ = checked_observe()
        if st.get('task'):
            raise RuntimeError('another task is active; resupply will not replace it')
        request = call('task', name, json.dumps(args), str(timeout), as_json=True)
        task_id = request['id']
        summary['tasks'].append(dict(id=task_id, task=name))
        anchor = datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z')
        until = min(deadline, time.monotonic() + timeout + 5)
        while time.monotonic() < until:
            st, mp = checked_observe()
            history = call('panel', 'GET', '/api/task?id=' + task_id + '&at=' + anchor, as_json=True)
            (saved / (task_id + '.json')).write_text(json.dumps(history, indent=2) + '\n')
            result = terminal(history, task_id)
            if result:
                summary['tasks'][-1]['result'] = result
                return result, st, mp, history
            if (st.get('task') or {}).get('id') != task_id:
                raise RuntimeError('resupply lost body ownership')
            time.sleep(1)
        raise RuntimeError('resupply native task time limit')

    if math.dist(status['pos'], home['pos']) > 3:
        result, status, mapped, _ = task('move_to', dict(to=home['pos'], stopDistance=2, run=home_run(status)), 270)
        if result['status'] != 'succeeded' or math.dist(status['pos'], home['pos']) > 3:
            raise RuntimeError('native return home did not complete: ' + str(result.get('detail') or result.get('reason')))
    attempted = set()
    for _ in range(4):
        if len(summary['supplied']) >= 2:
            break
        status, mapped = checked_observe()
        options = [c for c in candidates(status, mapped, summary['supplied']) if (c[2], c[3]) not in attempted]
        if not options:
            break
        _, _, chest, prefab, amount = options[0]
        attempted.add((chest, prefab))
        before = pack_snapshot(call, scope)
        if (before.get('size', [0, 0])[0] * before.get('size', [0, 0])[1] - len(before.get('items', []))) < 2:
            raise RuntimeError('resupply needs two free slots')
        result, status, mapped, history = task('withdraw', dict(container=chest, items=[dict(prefab=prefab, count=amount)]), 45)
        # The task-matched receipt is authoritative, including partial moves.
        history = history.get('value', []) if isinstance(history, dict) else history
        receipts = [r.get('frame', {}) for r in history if r.get('frame', {}).get('type') == 'container']
        receipt = next((c for c in receipts if c.get('entity') == chest and c.get('task') == summary['tasks'][-1]['id'] and c.get('direction') == 'withdraw'), None)
        moved = (receipt or {}).get('moved', {}).get(prefab, 0)
        pack = pack_snapshot(call, scope)
        # Auto-eating may race the pack delta; the exact withdrawal receipt is
        # still required and retained, while active food explains one consumed item.
        verified = False
        for _ in range(3):
            status, mapped = checked_observe()
            active = {f['prefab'] for f in (status.get('food') or {}).get('active', [])}
            gain = count(pack, prefab) - count(before, prefab)
            verified = moved > 0 and pack.get('seq', 0) > before.get('seq', 0) and gain >= moved - (1 if prefab in active else 0)
            if verified or moved <= 0:
                break
            time.sleep(1)
            pack = pack_snapshot(call, scope)
        if verified:
            summary['supplied'][prefab] = moved
            summary['tasks'][-1]['receipt'] = receipt
        elif result['status'] == 'succeeded':
            raise RuntimeError('food withdrawal lacks matched receipt and measured inventory gain')
    if not summary['supplied']:
        raise RuntimeError('no verified food transfer from known home storage')
    until = min(deadline, time.monotonic() + 180)
    while time.monotonic() < until:
        status, _ = checked_observe()
        active = [f['prefab'] for f in (status.get('food') or {}).get('active', [])]
        if any(p in summary['supplied'] for p in active) and status['hp'] >= status['hpMax'] * .8:
            summary.update(outcome='succeeded', activeFood=active)
            return
        time.sleep(2)
    raise RuntimeError('food was supplied but healthy automatic recovery was not observed')


def pack_snapshot(call, scope):
    value = call('panel', 'GET', '/api/inventory', as_json=True)
    if (value.get('world'), value.get('connection')) != scope or not value.get('inventory'):
        raise RuntimeError('missing or stale scoped inventory')
    return value['inventory']
