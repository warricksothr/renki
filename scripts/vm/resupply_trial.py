"""Bounded operator resupply through native movement and verified transfers."""
import datetime
import json
import math
import time
from harvest_trial import health_loss

# Exact ordinary food prefabs, never inferred from arbitrary item names.
FOODS = ('CookedDeerMeat', 'NeckTailGrilled', 'CookedMeat', 'Honey', 'Raspberry', 'Blueberries', 'Mushroom')


def candidates(status, mapped, supplied, opened=None):
    home = status.get('home') or {}
    if home.get('world') != status.get('world') or not home.get('pos'):
        raise RuntimeError('no known home in the current world')
    choices = []
    opened = opened or {}
    visible = [s for s in (mapped.get('perception') or {}).get('structures', [])
               if s.get('kind') == 'container' and s.get('id') and s.get('pos')]
    visible_ids = {s['id'] for s in visible}
    for chest in list(mapped.get('containers', [])) + [c for c in opened.values() if c.get('pos')]:
        if math.dist(chest['pos'], home['pos']) > 32:
            continue
        entity = chest['entity']
        # Last-known contents are only a hint. Rebuilt/reidentified storage
        # may have a fresh entity at the same measured location. Bind the
        # request to that unique live chest, never infer a successful move.
        if entity not in visible_ids:
            replacements = [s for s in visible if math.dist(s['pos'], chest['pos']) <= .4]
            if len(replacements) == 1:
                entity = replacements[0]['id']
            elif len(replacements) > 1:
                continue
        for item in opened.get(entity, chest).get('items') or []:
            prefab = item.get('prefab')
            if prefab in FOODS and prefab not in supplied and item.get('stack', 0) > 0:
                choices.append((FOODS.index(prefab), math.dist(status['pos'], chest['pos']), entity, prefab, min(4, item['stack'])))
    # A failed food hint is a reason to check another box. A one-item ordinary
    # food request opens through the normal access path and reports actual
    # contents even if that food is missing. Exact receipts remain required.
    probe_food = next((p for p in FOODS if p not in supplied), None)
    for chest in visible:
        if probe_food and math.dist(chest['pos'], home['pos']) <= 32 and chest['id'] not in opened:
            choices.append((len(FOODS), math.dist(status['pos'], chest['pos']), chest['id'], probe_food, 1))
    return sorted(set(choices))


def count(pack, prefab):
    return sum(i.get('stack', 0) for i in pack.get('items', []) if i.get('prefab') == prefab)


def home_run(status):
    # Walk long trips, but do not linger outside the shelter on a short,
    # rested final approach. The native task still owns routing and stamina.
    return math.dist(status['pos'], status['home']['pos']) <= 40 and status.get('stamina', 0) >= 30


def startup_home(status, history):
    """A return already dispatched on rejoin can finish; never replace it."""
    home, task = status.get('home') or {}, status.get('task') or {}
    if home.get('world') != status.get('world') or not home.get('pos') or task.get('task') != 'move_to':
        return False
    rows = history.get('value', []) if isinstance(history, dict) else history
    return any(row.get('dir') == 'out' and row.get('frame', {}).get('type') == 'task'
               and row['frame'].get('id') == task.get('id') and row['frame'].get('task') == 'move_to'
               and row['frame'].get('args', {}).get('to') == home['pos'] for row in rows)


def run(summary, call, observe, terminal, saved):
    status, mapped = observe()
    home = status.get('home') or {}
    if home.get('world') != status.get('world') or not home.get('pos') or math.dist(status['pos'], home['pos']) > 500:
        raise RuntimeError('resupply requires known home within 500 m in this world')
    scope = (status['world'], status.get('connection'))
    previous_health = status
    deadline = time.monotonic() + 420
    summary.update(origin=status['pos'], tasks=[], supplied={}, checkedChests=0, requests=0)

    def checked_observe():
        nonlocal previous_health
        st, mp = observe()
        if (st.get('world'), st.get('connection')) != scope or not st.get('connected') or st.get('phase') != 'in_world':
            raise RuntimeError('resupply world/connection changed')
        if health_loss(previous_health, st):
            raise RuntimeError('observed damage during resupply')
        if st.get('location', {}).get('inWater') or 'swimming' in st.get('flags', []):
            raise RuntimeError('unsafe water during resupply')
        if time.monotonic() >= deadline or math.dist(st['pos'], summary['origin']) > 525:
            raise RuntimeError('resupply time/travel limit')
        previous_health = st
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
                # The task may finish while history is fetched, after the pose
                # above was sampled. Require its subsequent idle body state.
                settled_until = min(deadline, time.monotonic() + 5)
                while True:
                    st, mp = checked_observe()
                    active = (st.get('task') or {}).get('id')
                    if not active:
                        break
                    if active != task_id:
                        raise RuntimeError('resupply lost body ownership after task completion')
                    if time.monotonic() >= settled_until:
                        raise RuntimeError('resupply completion state remained stale')
                    time.sleep(.1)
                return result, st, mp, history
            if (st.get('task') or {}).get('id') != task_id:
                raise RuntimeError('resupply lost body ownership')
            time.sleep(1)
        raise RuntimeError('resupply native task time limit')

    if math.dist(status['pos'], home['pos']) > 3:
        result, status, mapped, _ = task('move_to', dict(to=home['pos'], stopDistance=2, run=home_run(status), dry=True), 270)
        if result['status'] != 'succeeded' or math.dist(status['pos'], home['pos']) > 3:
            raise RuntimeError('native return home did not complete: ' + str(result.get('detail') or result.get('reason')))
    attempted, missing_chests, opened = set(), set(), {}
    # Freeze the eligible area for this search. Each chest/food pair is tried
    # once; the finite chest set and ordinary-food allowlist bound the work.
    # An inspection and a subsequent verified withdrawal can share one chest.
    search_chests = {c[2] for c in candidates(status, mapped, {})}
    summary['searchChests'] = len(search_chests)
    while len(summary['supplied']) < 2:
        status, mapped = checked_observe()
        # Already-carried food contributes to readiness, never to transfer
        # proof. Avoid opening every remaining chest after replenishing one
        # type when two ordinary food reserves are already in the pack.
        if summary['supplied']:
            carried = pack_snapshot(call, scope)
            if sum(count(carried, prefab) >= 3 for prefab in FOODS) >= 2:
                break
        options = [c for c in candidates(status, mapped, summary['supplied'], opened)
                   if c[2] in search_chests and (c[2], c[3]) not in attempted and c[2] not in missing_chests]
        if not options:
            break
        _, _, chest, prefab, amount = options[0]
        attempted.add((chest, prefab))
        summary['requests'] = len(attempted)
        before = pack_snapshot(call, scope)
        if (before.get('size', [0, 0])[0] * before.get('size', [0, 0])[1] - len(before.get('items', []))) < 2:
            raise RuntimeError('resupply needs two free slots')
        result, status, mapped, history = task('withdraw', dict(container=chest, items=[dict(prefab=prefab, count=amount)]), 45)
        # The task-matched receipt is authoritative, including partial moves.
        history = history.get('value', []) if isinstance(history, dict) else history
        receipts = [r.get('frame', {}) for r in history if r.get('frame', {}).get('type') == 'container']
        receipt = next((c for c in receipts if c.get('entity') == chest and c.get('task') == summary['tasks'][-1]['id'] and c.get('direction') == 'withdraw'), None)
        if receipt is not None:
            opened[chest] = receipt
            summary['checkedChests'] = len(opened)
        elif result['status'] != 'succeeded':
            missing_chests.add(chest)
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
