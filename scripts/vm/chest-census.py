#!/usr/bin/env python3
"""`just vm chest-census`: open every known container near the bot and record it.

Container memory only changes when one is opened, so after time away it is
stale, and chests, barrels and other storage that others built are known only
as landmarks. Any piece the game reports as a container is included. This walks the
candidates nearest-first with the read-only native `inspect_container` task,
one chest at a time, and retains what each attempt returned. Nothing is
moved. Native reach, footing, threat and ownership rules stay authoritative:
a refused chest is recorded with its reason and the walk goes on.

    --plan           list the candidates and their order; send nothing
    --radius M       how far from the bot's starting position to look (40)
    --limit N        stop after N chests
    --only stale     skip chests whose remembered contents are under --fresh-s old
    --fresh-s S      what counts as fresh for --only stale (900)

Results go to .checks/census/<UTC>.json; summarise() is printed. Selection,
ordering and the summary are pure and tested offline (chest-census-test.py).
"""
import argparse
import datetime
import json
import math
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[2]
FINAL = ('succeeded', 'failed', 'rejected', 'cancelled')
# The plugin found no container under any id known for the place. With the
# body in range that is evidence of absence, not a failure to reach it.
GONE = 'invalid_target'
# Well inside the distance at which the client keeps objects loaded.
LOADED_M = 30
MIN_HP_FRACTION = 0.5
MAX_CONSECUTIVE_FAILURES = 6


def is_container(row):
    """Perception and memory mark every storage piece with kind `container`:
    wood, reinforced, black metal and private chests, barrels, carts and
    ships' holds. The prefab test only covers rows that carry no kind."""
    if row.get('kind'):
        return row['kind'] == 'container'
    return str(row.get('prefab') or '').lower().startswith('piece_chest')


SAME_PLACE_M = 0.6
# A landmark seen this recently counts as a sighting. Older ones are places a
# container once stood: on 2026-11-17 seven two-week-old landmarks within 5 m
# of the bot were all gone.
SIGHTED_S = 6 * 3600


def age_s(row, now):
    """Seconds since a row was last seen: its own ageS, else from `at`
    (landmarks on the map carry only the time). Unknown is infinitely old."""
    if row.get('ageS') is not None:
        return row['ageS']
    try:
        at = datetime.datetime.fromisoformat(str(row.get('at'))[:19] + '+00:00')
    except ValueError:
        return math.inf
    return (now - at).total_seconds()


def candidates(remembered, landmarks, structures, center, radius, now=None):
    """Every chest within radius, one per place. A chest's entity id can change
    between sessions, so the same box may be remembered under one id, seen
    under another and marked as a landmark under a third: rows within
    SAME_PLACE_M are one chest. The id currently seen is the one to ask for;
    remembered contents and exact positions are kept; rounded landmark
    positions only fill gaps."""
    now = now or datetime.datetime.now(datetime.timezone.utc)
    found = []
    for source, rows in (('remembered', remembered), ('seen', structures), ('landmark', landmarks)):
        for row in rows or []:
            entity, pos = row.get('entity') or row.get('id'), row.get('pos')
            if not entity or not pos or len(pos) != 3 or not is_container(row):
                continue
            if math.dist(pos, center) > radius:
                continue
            entry = next((c for c in found if entity in c['ids'] or math.dist(c['pos'], pos) <= SAME_PLACE_M), None)
            if entry is None:
                entry = {'entity': entity, 'ids': [], 'prefab': row.get('prefab'), 'pos': pos, 'source': source,
                         'opened': False, 'ageS': None, 'items': None, 'sighted': False}
                found.append(entry)
            for known in (entity, row.get('observedEntity')):
                if known and known not in entry['ids']:
                    entry['ids'].append(known)
            if source == 'remembered':
                # Its age is that of the contents, not of the last sighting,
                # and perception lists only the nearest containers, so an
                # opened container counts as sighted until asked for.
                entry.update(opened=True, ageS=row.get('ageS'), items=len(row.get('items') or []),
                             remembered=entity, sighted=True)
                if row.get('observedEntity'):
                    entry['entity'] = row['observedEntity']
            elif source == 'seen':
                entry.update(entity=entity, pos=pos if not entry['opened'] else entry['pos'], sighted=True)
            elif 0 <= age_s(row, now) <= SIGHTED_S:
                entry['sighted'] = True
    return found


def order(chests, start):
    """Nearest-neighbour walk from start, so the bot does not cross the room
    between neighbours. Ties break on entity for a stable plan. Containers
    with no recent sighting come after every sighted one: most are gone."""
    out, at = [], start
    for group in (True, False):
        walked = walk([c for c in chests if bool(c.get('sighted', True)) == group], at)
        out += walked
        at = walked[-1]['pos'] if walked else at
    return out


def walk(chests, start):
    left, at, out = list(chests), start, []
    while left:
        nxt = min(left, key=lambda c: (round(math.dist(c['pos'], at), 2), c['entity']))
        left.remove(nxt)
        out.append(nxt)
        at = nxt['pos']
    return out


def stale_only(chests, fresh_s):
    return [c for c in chests if not c['opened'] or c['ageS'] is None or c['ageS'] > fresh_s]


def summarise(results, after):
    """Counts, the refusal reasons, and where each item now sits."""
    by_entity = {c['entity']: c for c in after or []}
    opened = [r for r in results if r['status'] == 'succeeded']
    gone = sorted(r['entity'] for r in results if is_gone(r))
    reasons = {}
    for r in results:
        if r['status'] != 'succeeded' and not is_gone(r):
            key = f"{r['status']}: {why(r)}"
            reasons.setdefault(key, []).append(r['entity'])
    items, empty = {}, 0
    for r in opened:
        row = next((by_entity[i] for i in r.get('ids', [r['entity']]) if i in by_entity), None) or \
            next((c for c in after or [] if c.get('observedEntity') == r['entity']), {})
        contents = row.get('items') or []
        empty += not contents
        for stack in contents:
            entry = items.setdefault(stack.get('prefab'), {'count': 0, 'chests': set()})
            entry['count'] += int(stack.get('count') or stack.get('stack') or 1)
            entry['chests'].add(r['entity'])
    return {
        'attempted': len(results), 'opened': len(opened), 'empty': empty, 'gone': gone,
        'newlyOpened': sum(1 for r in opened if not r.get('openedBefore')),
        'refused': {key: sorted(value) for key, value in sorted(reasons.items())},
        'items': {name: {'count': v['count'], 'chests': sorted(v['chests'])}
                  for name, v in sorted(items.items(), key=lambda kv: (-len(kv[1]['chests']), kv[0] or ''))},
    }


def render(summary):
    lines = [f"census: {summary['opened']}/{summary['attempted']} chests opened "
             f"({summary['newlyOpened']} for the first time, {summary['empty']} empty)"]
    if summary['gone']:
        lines.append(f"  no longer there ({len(summary['gone'])}): {', '.join(summary['gone'])}"[:200])
    for reason, entities in summary['refused'].items():
        lines.append(f"  not opened ({len(entities)}): {reason}")
    scattered = [(name, v) for name, v in summary['items'].items() if len(v['chests']) > 1]
    lines.append(f"  {len(summary['items'])} distinct items; {len(scattered)} are split across chests")
    for name, v in scattered[:15]:
        lines.append(f"    {name}: {v['count']} in {len(v['chests'])} chests")
    return '\n'.join(lines)


def is_gone(outcome):
    """No container under any known id, asked from close enough that the game
    has the area loaded. From farther away the same answer only means the
    object is not loaded, so it stays an ordinary refusal."""
    near = outcome.get('bodyDistance') is not None and outcome['bodyDistance'] <= LOADED_M
    return outcome['status'] != 'succeeded' and near and (outcome.get('reason') or '').split(' ', 1)[0] == GONE


def vm(*arguments):
    done = subprocess.run(['just', 'vm', *arguments], cwd=ROOT, capture_output=True, text=True, timeout=60)
    if done.returncode:
        raise SystemExit(f"just vm {arguments[0]} failed: {done.stderr.strip()[-300:]}")
    data = json.loads(done.stdout)
    return data['value'] if isinstance(data, dict) and set(data) == {'value', 'Count'} else data


def run_task(task, arguments, timeout):
    body = json.dumps({'task': task, 'args': arguments, 'timeoutS': timeout})
    anchor = datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z')
    task_id = vm('panel', 'POST', '/api/task', body)['id']
    deadline = time.monotonic() + timeout + 20
    while time.monotonic() < deadline:
        records = vm('panel', 'GET', f'/api/task?id={task_id}&at={anchor}')
        for record in reversed(records):
            frame = record.get('frame', {})
            if frame.get('type') == 'task' and frame.get('re') == task_id and frame.get('status') in FINAL:
                reason = ' '.join(str(frame.get(k, '')) for k in ('reason', 'detail')).strip()
                return {'task': task_id, 'status': frame['status'], 'reason': reason[:1200],
                        'standing': standing_spots(records)}
        time.sleep(2)
    return {'task': task_id, 'status': 'unknown', 'reason': 'no terminal result within the wait limit'}


def why(outcome):
    """One short, stable phrase for a refusal: the native code and, when the
    receipt carries one, its detail. Receipts differ in travel and trail, so
    the raw reason cannot group like with like."""
    reason = outcome.get('reason') or 'no reason'
    code = reason.split(' ', 1)[0]
    marker = '"detail":"'
    if marker in reason:
        detail = reason.split(marker, 1)[1].split('"', 1)[0]
        return f'{code}: {detail}'[:110]
    return reason[:110]


def standing_spots(records):
    """Positions the native approach search verified for reach and body
    clearance and then tried to walk to, in order, from a task's trace."""
    spots = []
    for record in records or []:
        frame = record.get('frame') or {}
        to = (frame.get('data') or {}).get('to')
        if frame.get('type') == 'trace' and frame.get('msg') == 'approaching verified interaction position' \
                and isinstance(to, list) and len(to) == 3 and to not in spots:
            spots.append(to)
    return spots


def next_step(outcome, tried_ids, ids, walked, stepped_back):
    """What to try after one attempt.

    - `invalid_target`: another id known for the same place.
    - `unreachable` with a verified standing spot: walk there with an ordinary
      move and ask again. The inspection's own bounded walker gives up within
      its local search limit; seen live on 2026-11-17, a barrel behind a double
      row failed four approaches with no travel, then opened from the first
      traced spot once an ordinary move had gone round the row.
    - `unreachable` with no spot: one retry from the start point, in case the
      body was pressed against a row.
    """
    if outcome['status'] == 'succeeded':
        return None
    code = (outcome.get('reason') or '').split(' ', 1)[0]
    if code == GONE:
        return next((('id', i) for i in ids if i not in tried_ids), None)
    if code == 'unreachable':
        spot = next((s for s in outcome.get('standing') or [] if s not in walked), None)
        if spot is not None and len(walked) < 2:
            return ('walk', spot)
        if not walked and not stepped_back:
            return ('step_back', None)
    return None


def attempt(chest, scope, start, body):
    tried, walked, stepped_back, entity = [], [], False, chest['entity']
    distance = math.dist(body, chest['pos']) if body else None
    while True:
        tried.append(entity)
        outcome = run_task('inspect_container', {'container': entity, 'scope': scope}, 75)
        step = next_step(outcome, tried, chest['ids'], walked, stepped_back)
        if step is None:
            outcome.update(entity=entity, tried=tried, walked=walked, steppedBack=stepped_back,
                           bodyDistance=distance)
            return outcome
        if step[0] == 'id':
            entity = step[1]
        elif step[0] == 'walk':
            walked.append(step[1])
            run_task('move_to', {'to': step[1], 'stopDistance': 0.4}, 60)
        else:
            stepped_back = True
            run_task('move_to', {'to': start, 'stopDistance': 1.0}, 60)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument('--plan', action='store_true')
    parser.add_argument('--radius', type=float, default=40)
    parser.add_argument('--limit', type=int, default=0)
    parser.add_argument('--only', choices=['stale'])
    parser.add_argument('--fresh-s', type=float, default=900)
    args = parser.parse_args(argv)
    if not 10 <= args.radius <= 240:
        raise SystemExit('--radius must be 10-240 m')

    status = vm('panel', 'GET', '/api/status')
    if status.get('phase') != 'in_world' or not status.get('pos'):
        raise SystemExit('the bot is not in the world')
    start = status['pos']
    world_map = vm('panel', 'GET', '/api/map')
    chests = candidates(vm('panel', 'GET', '/api/memory/containers').get('containers'), world_map.get('landmarks'),
                        (world_map.get('perception') or {}).get('structures'), start, args.radius)
    if args.only == 'stale':
        chests = stale_only(chests, args.fresh_s)
    chests = order(chests, start)
    if args.limit:
        chests = chests[:args.limit]
    print(f'{len(chests)} chests within {args.radius:g} m; '
          f"{sum(1 for c in chests if not c['opened'])} never opened")
    if args.plan:
        for c in chests:
            age = 'never opened' if not c['opened'] else f"{int((c['ageS'] or 0) // 3600)} h old, {c['items']} stacks"
            print(f"  {c['entity']:<20} {c['prefab']:<20} {c['pos']}  {age}")
        return 0

    scope = {'center': start, 'radius': min(250, args.radius + 10), 'travelM': 80, 'durationS': 60}
    out = ROOT / '.checks' / 'census'
    out.mkdir(parents=True, exist_ok=True)
    path = out / (datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ') + '.json')
    results, failures, stopped, body = [], 0, '', start

    def save(after=None):
        # Written after every container: an SSH failure an hour in must not
        # lose the refusals and walks recorded so far.
        path.write_text(json.dumps({'start': start, 'scope': scope, 'stopped': stopped, 'planned': len(chests),
                                    'results': results, 'summary': summarise(results, after),
                                    'containers': after}, indent=1) + '\n')

    try:
        for index, chest in enumerate(chests, 1):
            outcome = attempt(chest, scope, start, body)
            outcome.update(ids=chest['ids'], prefab=chest['prefab'], pos=chest['pos'],
                           openedBefore=chest['opened'])
            results.append(outcome)
            save()
            print(f"  [{index}/{len(chests)}] {outcome['entity']} "
                  + ('no longer there' if is_gone(outcome) else outcome['status'])
                  + (' after walking round' if outcome['walked'] and outcome['status'] == 'succeeded' else '')
                  + (' after stepping back' if outcome['steppedBack'] and outcome['status'] == 'succeeded' else '')
                  + (f": {why(outcome)}" if outcome['status'] != 'succeeded' and not is_gone(outcome) else ''),
                  flush=True)
            # A container that is gone costs nothing and says nothing about the next one.
            if not is_gone(outcome):
                failures = 0 if outcome['status'] == 'succeeded' else failures + 1
            now = vm('panel', 'GET', '/api/status')
            body = now.get('pos') or body
            if now.get('phase') != 'in_world':
                stopped = 'the bot left the world'
            elif now.get('hpMax') and now.get('hp', 0) < MIN_HP_FRACTION * now['hpMax']:
                stopped = f"health {now.get('hp')} is under half of {now['hpMax']}"
            elif failures >= MAX_CONSECUTIVE_FAILURES:
                stopped = f'{failures} containers in a row could not be opened'
            if stopped:
                break
    except (SystemExit, subprocess.SubprocessError, ValueError, KeyboardInterrupt) as error:
        stopped = f'interrupted: {type(error).__name__}: {str(error)[:200]}'
    if stopped:
        print(f'stopping: {stopped}')
    try:
        after = vm('panel', 'GET', '/api/memory/containers').get('containers')
    except (SystemExit, subprocess.SubprocessError, ValueError):
        after = None
    save(after)
    print(render(summarise(results, after)))
    print(f'Evidence: {path}')
    return 0 if not stopped else 1


if __name__ == '__main__':
    sys.exit(main())
