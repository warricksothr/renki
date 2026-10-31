"""One controller-owned gather/deliver/return run; native tasks own all movement."""
from collections import Counter
import datetime
import json
import math
from pathlib import Path
import time

from harvest_trial import counts, pack_ready, select_resource, worksite_clear, health_loss

PRODUCTS = ('Wood', 'BeechSeeds', 'Resin', 'Feathers')


def select_site(status, perception):
    # Controller gather can approach its site before native acquisition.
    candidates = [r for r in perception.get('resources', []) if r.get('id') and
                  r.get('kind') == 'tree' and r.get('prefab') == 'Beech1' and r.get('minTool') == 0]
    nearest = sorted(candidates, key=lambda r: math.dist(r['pos'], status['pos']))[:6]
    diagnostics = dict(observedTrees=len(candidates), maxDistanceM=60, nearest=[dict(
        entity=r['id'], distanceM=round(math.dist(r['pos'], status['pos']), 2),
        worksiteClear=worksite_clear(status, perception, r['pos'])) for r in nearest])
    return select_resource(status, perception, max_distance=60), diagnostics


def records(value):
    return value.get('value', []) if isinstance(value, dict) else value


def destinations(status, mapped):
    home = status.get('home') or {}
    if home.get('world') != status.get('world') or not home.get('pos'):
        raise RuntimeError('controller gather needs known home/storage in this world')
    if math.dist(status['pos'], home['pos']) > 150:
        raise RuntimeError('controller gather benchmark needs home within 150 m')
    choices = [c for c in mapped.get('containers', []) if c.get('entity') and c.get('pos')
               and math.dist(c['pos'], home['pos']) <= 10]
    # A current visible chest supersedes an old identity at its measured position.
    visible = [c for c in (mapped.get('perception') or {}).get('structures', [])
               if c.get('kind') == 'container' and c.get('id') and c.get('pos')]
    result = []
    for c in sorted(choices, key=lambda c: (math.dist(c['pos'], home['pos']), c['entity'])):
        replacement = [v for v in visible if math.dist(v['pos'], c['pos']) <= .4]
        if len(replacement) > 1:
            continue
        entity = replacement[0]['id'] if replacement else c['entity']
        if entity not in result:
            result.append(entity)
    if not result:
        raise RuntimeError('no remembered home chests for gains-only delivery')
    return result[:16]


def run(summary, call, observe, saved, trees=1, site=None):
    status, mapped = observe()
    scope = (status.get('world'), status.get('worldUid'), status.get('connection'))
    if not all(scope) or status.get('task') or status.get('phase') != 'in_world':
        raise RuntimeError('controller gather requires an idle scoped in-world bot')
    pack = call('panel', 'GET', '/api/inventory', as_json=True)
    if (pack.get('world'), pack.get('connection')) != (scope[0], scope[2]) or not pack_ready(pack):
        raise RuntimeError('controller gather needs a scoped usable axe and spare capacity')
    target, summary['selection'] = select_site(status, mapped.get('perception') or {})
    if target is None and site is None:
        raise RuntimeError('no nearby isolated full beech for the controller gather benchmark')
    if site is not None and (not status.get('home') or math.dist(site, status['home']['pos']) > 150):
        raise RuntimeError('gather work-site hint must remain within 150 m of home')
    worksite = site if site is not None else target['pos']
    if site is not None:
        target = None  # The controller binds an observed tree after native travel.
    chests = destinations(status, mapped)
    initial_behaviours = records(call('panel', 'GET', '/api/behaviours', as_json=True))
    if any(b.get('pending') or b.get('running') for b in initial_behaviours):
        raise RuntimeError('another behavior is pending/running; gather will not replace it')
    name = 'M2 gather ' + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%S%fZ')
    summary.update(mode='controller_gather', origin=status['pos'], world=scope[0], worldUid=scope[1],
                   connection=scope[2], site=worksite, tree=target, before=counts(pack), maxTrees=trees,
                   home=status.get('home', {}).get('pos'), returnHome=trees > 1,
                   beforeInventorySeq=pack['inventory']['seq'], to=chests,
                   initialResourceIDs=[r['id'] for r in (mapped.get('perception') or {}).get('resources', [])])
    definition = dict(template='gather', params=dict(site=dict(world=scope[0], pos=worksite, radius=30 if trees > 1 else 20),
                      to=chests, maxTrees=trees, returnHome=trees > 1, **{'return': trees == 1}), trigger=dict(kind='manual'), repeat=dict(mode='once'),
                      priority=10, enabled=False)
    # Mark for cleanup before the write: a lost response may still have saved it.
    summary['behaviourName'] = name
    call('panel', 'POST', '/api/behaviours', json.dumps(dict(action='save', name=name, definition=definition)), as_json=True)
    call('panel', 'POST', '/api/behaviours', json.dumps(dict(action='start', name=name)), as_json=True)
    previous_health = status
    deadline = time.monotonic() + trees * 480 + 10
    while time.monotonic() < deadline:
        current, mp = observe()
        if (current.get('world'), current.get('worldUid'), current.get('connection')) != scope or current.get('phase') != 'in_world':
            raise RuntimeError('controller gather connection/world changed')
        if (mp.get('world'), mp.get('connection')) != (scope[0], scope[2]):
            raise RuntimeError('controller gather observations are stale')
        if health_loss(previous_health, current) or current['hp'] < current['hpMax'] * .8:
            raise RuntimeError('observed health loss during controller gather')
        previous_health = current
        if current.get('location', {}).get('inWater') or 'swimming' in current.get('flags', []) or math.dist(current['pos'], summary['origin']) > 180:
            raise RuntimeError('controller gather footing/travel guard')
        if any(c.get('hostile') and c.get('hpFrac', 0) > 0 and math.dist(c['pos'], current['pos']) < 10
               for c in (mp.get('perception') or {}).get('characters', [])):
            raise RuntimeError('hostile approached the controller gather trial')
        entries = records(call('panel', 'GET', '/api/behaviours', as_json=True))
        own = next((b for b in entries if b['name'] == name), None)
        if own is None:
            raise RuntimeError('controller gather behavior was removed')
        summary['behaviour'] = own
        if not own.get('running') and not own.get('pending'):
            if own.get('last') != 'completed' or own.get('runs') != 1:
                summary.update(outcome='failed', reason=own.get('last') or 'behavior did not complete')
                return
            after = call('panel', 'GET', '/api/inventory', as_json=True)
            if (after.get('world'), after.get('connection')) != (scope[0], scope[2]) or not after.get('inventory'):
                raise RuntimeError('missing final scoped inventory')
            summary.update(outcome='awaiting_proof', after=counts(after), afterInventorySeq=after['inventory']['seq'], finalPos=current['pos'])
            (saved / 'behaviour.json').write_text(json.dumps(own, indent=2) + '\n')
            return
        time.sleep(1)
    raise RuntimeError('controller gather trial time limit')


def verify(audit, summary):
    """Reconcile attributed native steps, break traces, pack gains and exact receipts."""
    rows = [json.loads(line) for line in Path(audit).read_text(encoding='utf-8-sig').splitlines() if line.strip()]
    steps = [r['attrs'] for r in rows if r.get('msg') == 'behaviour step'
             and r.get('attrs', {}).get('instance') == summary.get('behaviourName')]
    if not steps:
        return {} if summary.get('outcome') != 'awaiting_proof' else _fail('no attributed native steps')
    ids = {step['id'] for step in steps}
    scoped = [r for r in rows if r.get('conn') == summary['connection']]
    frames = [r.get('frame', {}) for r in scoped]
    requests = {f['id']: f for f in frames if f.get('type') == 'task' and f.get('id') in ids and f.get('task')}
    results = {f['re']: f for f in frames if f.get('type') == 'task' and f.get('re') in ids and f.get('status') in ('succeeded', 'failed', 'rejected', 'cancelled')}
    tasks = [dict(id=s['id'], task=s['task'], result=results.get(s['id'])) for s in steps]
    proof = dict(tasks=tasks)
    if summary.get('outcome') != 'awaiting_proof':
        return proof
    if any(f.get('worldUid') and f['worldUid'] != summary['worldUid'] for f in frames if f.get('type') in ('state', 'inventory', 'perception', 'container')):
        return _fail('gather audit world changed')
    harvests = [s for s in steps if s['task'] == 'harvest']
    required_trees = summary.get('maxTrees', 1)
    if not required_trees * 3 <= len(harvests) <= required_trees * 5:
        return _fail('full tree and two log harvests not verified')
    known = {}
    for f in frames:
        if f.get('type') == 'perception':
            for r in f.get('resources', []):
                known[r['id']] = r
    harvested = set()
    trees = logs = chain_logs = 0
    for index, step in enumerate(harvests):
        key = step['id']
        resource = known.get(requests.get(key, {}).get('args', {}).get('entity'), {})
        full_tree = resource.get('prefab') == 'Beech1' and resource.get('kind') == 'tree'
        new_log = (resource.get('kind') == 'log' and resource.get('prefab', '').lower().startswith('beech')
                   and resource.get('id') not in summary['initialResourceIDs'])
        if not (full_tree or new_log) or index == 0 and not full_tree:
            return _fail('harvest resource identity/kind missing from observations')
        if resource['id'] in harvested or (index == 0 and summary.get('tree') and resource['id'] != summary['tree']['id']):
            return _fail('harvest substituted or repeated a resource')
        if full_tree:
            if trees and chain_logs < 2:
                return _fail('another tree started before splitting its predecessor')
            trees += 1
            chain_logs = 0
        else:
            logs += 1
            chain_logs += 1
        harvested.add(resource['id'])
        if results.get(key, {}).get('status') != 'succeeded' or not any(f.get('type') == 'trace' and f.get('task') == key and f.get('source') == 'harvest'
               and f.get('msg') == 'resource broke after local hit' and f.get('data', {}).get('entity') == resource['id'] and f.get('data', {}).get('hits', 0) > 0 for f in frames):
            return _fail('missing task-matched harvest break proof')
    if trees != required_trees or chain_logs < 2:
        return _fail('required distinct complete trees not verified')
    deposits = [f for f in frames if f.get('type') == 'container' and f.get('task') in ids and f.get('direction') == 'deposit']
    moved = Counter()
    seen = set()
    for receipt in deposits:
        key = receipt['task']
        request = requests.get(key, {})
        if key in seen:
            return _fail('duplicate deposit receipt')
        seen.add(key)
        if request.get('task') != 'deposit' or receipt.get('entity') != request.get('args', {}).get('container') or receipt.get('entity') not in summary['to']:
            return _fail('deposit receipt scope/identity mismatch')
        for prefab, count in receipt.get('moved', {}).items():
            if prefab not in PRODUCTS or type(count) is not int or count < 0:
                return _fail('invalid moved product/count')
            moved[prefab] += count
    first_deposit = next((i for i, f in enumerate(frames) if f.get('type') == 'task' and f.get('task') == 'deposit' and f.get('id') in ids), None)
    packs = [f for f in frames[:first_deposit] if f.get('type') == 'inventory' and f.get('seq', 0) > summary['beforeInventorySeq']] if first_deposit is not None else []
    if not packs or moved['Wood'] <= 0:
        return _fail('missing observed gains or positive task-matched Wood deposit')
    gained = counts(dict(inventory=packs[-1]))
    for prefab in PRODUCTS:
        if gained.get(prefab, 0) - summary['before'].get(prefab, 0) != moved[prefab] or summary['after'].get(prefab, 0) != summary['before'].get(prefab, 0):
            return _fail('gain-only delivery or preservation of original products not verified')
    last = steps[-1]['id']
    final = requests.get(last, {})
    destination = summary['home'] if summary.get('returnHome') else summary['site']
    if final.get('task') != 'move_to' or final.get('args', {}).get('to') != destination or results.get(last, {}).get('status') != 'succeeded' or math.dist(summary['finalPos'], destination) > 3:
        return _fail('missing successful return to the requested destination')
    if summary['afterInventorySeq'] <= max(r.get('seq', 0) for r in deposits):
        return _fail('final inventory predates deposit receipts')
    return dict(proof, outcome='succeeded', treesHarvested=trees, logsHarvested=logs, woodGain=moved['Wood'], deposited=dict(moved), returned=True)


def _fail(reason):
    raise RuntimeError(reason)
