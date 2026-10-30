"""One full beech and newly observed nearby logs, through normal harvest tasks."""
import datetime
from collections import Counter
import json
import math
import time


def diagnostics(audit, task_ids):
    """A bounded projection of matching harvest traces; raw evidence stays on disk."""
    fields = {
        'physical strike contact visible': ('entity', 'contact', 'origin', 'reach', 'probe'),
        'approaching physical contact': ('contact', 'goal', 'reach', 'attackHeight'),
        'resource broke after local hit': ('entity', 'hits'),
        'protected collider': ('collider', 'radius', 'center', 'boundsDistance', 'character', 'piece'),
        'safety scan': ('phase', 'radius', 'previousCapacity', 'capacity', 'count', 'complete'),
        'escape route rejected': ('attempt', 'goal', 'stage', 'collider'),
        'escape route selected': ('attempt', 'from', 'goal'),
        'unsupported physical contact': ('entity', 'resource', 'contact', 'feet', 'probeTop', 'probe', 'colliders'),
    }
    tasks = {key: dict(messages=Counter(), last={}) for key in list(dict.fromkeys(task_ids))[:5]}
    damage_events = 0
    with open(audit, encoding='utf-8-sig') as lines:
        for line in lines:
            if not line.strip():
                continue
            frame = json.loads(line).get('frame', {})
            if frame.get('type') == 'event' and frame.get('event') == 'damaged':
                damage_events += 1
            if frame.get('type') != 'trace' or frame.get('source') != 'harvest':
                continue
            task = tasks.get(frame.get('task'))
            message = frame.get('msg')
            if task is None or message not in fields:
                continue
            data = frame.get('data') or {}
            task['messages'][message] += 1
            task['last'][message] = {key: (data[key][:160] if isinstance(data[key], str) else data[key])
                                     for key in fields[message] if key in data}
    return dict(damageEvents=damage_events, tasks=tasks)


def distance(a, b):
    return math.dist(a, b)


def counts(pack):
    result = {}
    for item in (pack.get('inventory') or {}).get('items', []):
        result[item['prefab']] = result.get(item['prefab'], 0) + item['stack']
    return result


def worksite_clear(status, perception, at, falling=True):
    # Include neutral creatures and owners: trunks/rolling logs can damage them.
    radius = 32 if falling else 8
    for c in perception.get('characters', []):
        if c.get('hpFrac', 1) <= 0:
            continue
        if distance(c['pos'], status['pos']) < 15 or distance(c['pos'], at) < radius:
            return False
    return all(distance(piece['pos'], at) >= radius for piece in perception.get('structures', []))


def select_resource(status, perception, baseline=None, center=None, completed=()):
    candidates = []
    for r in perception.get('resources', []):
        if not r.get('id') or r['id'] in completed or r.get('minTool') != 0:
            continue
        if baseline is None:
            eligible = r.get('kind') == 'tree' and r.get('prefab') == 'Beech1'
        else:
            eligible = (r.get('kind') == 'log' and r.get('prefab', '').lower().startswith('beech')
                        and r['id'] not in baseline and distance(r['pos'], center) <= 20)
        if eligible and distance(r['pos'], status['pos']) <= 18 and worksite_clear(status, perception, r['pos'], r['kind'] == 'tree'):
            candidates.append(r)
    return min(candidates, key=lambda r: distance(r['pos'], status['pos']), default=None)


def remaining_logs(perception, baseline, center, completed):
    return [r for r in perception.get('resources', []) if r.get('kind') == 'log'
            and r.get('prefab', '').lower().startswith('beech') and r.get('id') not in baseline
            and r.get('id') not in completed and distance(r['pos'], center) <= 20]


def pack_ready(pack):
    inv = pack.get('inventory')
    if not inv or pack.get('weightMax', 0) - pack.get('weight', 0) < 60:
        return False
    if math.prod(inv.get('size', [0, 0])) - len(inv.get('items', [])) < 4:
        return False
    return any(i.get('prefab') in ('AxeFlint', 'AxeStone', 'AxeBronze', 'AxeIron', 'AxeBlackMetal')
               and i.get('durabilityMax', 0) > 0 and i.get('durability', 0) >= i['durabilityMax'] * .25
               for i in inv.get('items', []))


def run(summary, call, observe, terminal, saved, log_center=None):
    status, mapped = observe()
    world, connection, origin = status.get('world'), status.get('connection'), status['pos']
    if not world or not connection:
        raise RuntimeError('harvest needs a live world/connection scope')
    summary.update(origin=origin, world=world, connection=connection, tasks=[])
    deadline = time.monotonic() + 480
    previous_health = status.get('hp', 0)

    def snapshot():
        nonlocal previous_health
        st, mp = observe()
        if (not st.get('connected') or st.get('phase') != 'in_world' or st.get('world') != world
                or st.get('connection') != connection or mp.get('world') != world or mp.get('connection') != connection):
            raise RuntimeError('harvest world or connection changed')
        if (st.get('hpMax', 0) <= 0 or st.get('hp', 0) < st['hpMax'] * .8
                or st.get('location', {}).get('inWater') or distance(st['pos'], origin) > 40
                or 'encumbered' in st.get('flags', [])):
            raise RuntimeError('harvest health, footing, weight or travel guard')
        if st['hp'] < min(previous_health, st['hpMax']) - .15:
            raise RuntimeError('observed health loss during harvest trial')
        previous_health = st['hp']
        if time.monotonic() >= deadline:
            raise RuntimeError('harvest trial time limit')
        return st, mp

    def pack_snapshot():
        value = call('panel', 'GET', '/api/inventory', as_json=True)
        if value.get('world') != world or value.get('connection') != connection or not value.get('inventory'):
            raise RuntimeError('missing or stale scoped inventory')
        return value

    status, mapped = snapshot()
    if status.get('task'):
        raise RuntimeError('another task is active; harvest will not replace it')
    before = pack_snapshot()
    if not pack_ready(before):
        raise RuntimeError('harvest needs a usable axe, four free slots and 60 spare carry weight')
    perception = mapped.get('perception') or {}
    logs_only = log_center is not None
    target = select_resource(status, perception, set(), log_center) if logs_only else select_resource(status, perception)
    if target is None:
        raise RuntimeError('no nearby isolated tier-zero beech logs' if logs_only else 'no nearby isolated tier-zero full beech work site')
    summary['before'] = counts(before)
    center = log_center if logs_only else target['pos']
    summary.update(mode='logs_only' if logs_only else 'full_tree', site=center)
    if not logs_only:
        summary['tree'] = target
    baseline = set() if logs_only else {r['id'] for r in perception.get('resources', [])}
    completed, log_count = set(), 0
    for index in range(5):
        status, mapped = snapshot()
        if status.get('task'):
            raise RuntimeError('another task is active; harvest will not replace it')
        if not worksite_clear(status, mapped.get('perception') or {}, target['pos'], target['kind'] == 'tree'):
            raise RuntimeError('a character or structure is too close to the harvest site')
        anchor = datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z')
        remaining = min(195, deadline - time.monotonic())
        if remaining < 30:
            raise RuntimeError('insufficient remaining harvest budget')
        goal = dict(entity=target['id'], collect=True)
        dispatched = call('task', 'harvest', json.dumps(goal), str(remaining), as_json=True)
        task_id = dispatched['id']
        summary.update(task=task_id, outcome='running')
        receipt = dict(id=task_id, resource=target, goal=goal)
        summary['tasks'].append(receipt)
        task_deadline = time.monotonic() + remaining + 5
        while True:
            if time.monotonic() >= task_deadline:
                raise RuntimeError('harvest task wait limit')
            status, mapped = snapshot()
            history = call('panel', 'GET', '/api/task?id=' + task_id + '&at=' + anchor, as_json=True)
            (saved / ('task-' + str(index) + '.json')).write_text(json.dumps(history, indent=2) + '\n')
            ended = terminal(history, task_id)
            if ended:
                receipt['result'] = ended
                if ended['status'] != 'succeeded':
                    summary.update(outcome=ended['status'], result=ended)
                    return
                rows = history.get('value', []) if isinstance(history, dict) else history
                broke = any(r.get('frame', {}).get('task') == task_id
                            and r['frame'].get('source') == 'harvest'
                            and r['frame'].get('msg') == 'resource broke after local hit'
                            and (r['frame'].get('data') or {}).get('hits', 0) > 0 for r in rows)
                if not broke:
                    raise RuntimeError('harvest success lacks matching local-hit/break evidence')
                break
            if (status.get('task') or {}).get('id') != task_id:
                raise RuntimeError('harvest lost body ownership')
            if not worksite_clear(status, mapped.get('perception') or {}, target['pos'], target['kind'] == 'tree'):
                raise RuntimeError('a character approached the harvest route or falling area')
            time.sleep(.5)
        completed.add(target['id'])
        if target['kind'] == 'log':
            log_count += 1
        summary['after'] = counts(pack_snapshot())
        summary['logsHarvested'] = log_count
        summary['woodGain'] = summary['after'].get('Wood', 0) - summary['before'].get('Wood', 0)
        # Wait briefly for perception after break/spawn/settling; do not reuse
        # pre-existing logs or claim that proximity proves spawn lineage.
        settle_deadline = time.monotonic() + 8
        target = None
        while time.monotonic() < settle_deadline:
            status, mapped = snapshot()
            if status.get('task'):
                raise RuntimeError('another task took ownership between harvests')
            target = select_resource(status, mapped.get('perception') or {}, baseline, center, completed)
            if target:
                break
            time.sleep(.5)
        if target is None:
            if remaining_logs(mapped.get('perception') or {}, baseline, center, completed):
                raise RuntimeError('new nearby logs remain unavailable or unsafe; benchmark incomplete')
            if log_count < (1 if logs_only else 2) or summary['woodGain'] <= 0:
                raise RuntimeError('full tree finished but log splitting and collected Wood are not yet verified')
            detail = 'nearby beech logs harvested; collected Wood verified; full-tree benchmark not tested' if logs_only else 'full beech and newly observed logs harvested; collected Wood verified'
            summary.update(outcome='succeeded', result=dict(status='succeeded', detail=detail))
            return
    raise RuntimeError('harvest task budget exhausted; remaining logs left in place')
