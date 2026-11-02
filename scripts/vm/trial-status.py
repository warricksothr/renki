#!/usr/bin/env python3
"""Read compact progress from retained live-trial evidence, without game requests."""
import datetime
import json
import math
import re
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]


def report(folder, now=None):
    now = now or datetime.datetime.now(datetime.timezone.utc)
    result = dict(evidence=str(folder), outcome='preparing')
    previous = latest = first = None
    travelled = 0
    goal_ranges = {}
    observations = folder / 'observations.jsonl'
    if observations.exists():
        for line in observations.open():
            try:
                row = json.loads(line)
            except ValueError:
                # The writer may be appending its last observation right now.
                continue
            state = row.get('status', {})
            if state.get('phase') != 'in_world' or not state.get('pos'):
                continue
            if previous and (previous.get('world'), previous.get('connection')) == (state.get('world'), state.get('connection')):
                travelled += math.dist(previous['pos'], state['pos'])
            for observed_goal in state.get('goals', []):
                if (observed_goal.get('status') == 'running' and observed_goal.get('world') == state.get('world')
                        and observed_goal.get('center')):
                    goal_id = observed_goal['id']
                    goal_ranges[goal_id] = max(goal_ranges.get(goal_id, 0), math.dist(state['pos'], observed_goal['center']))
            first = first or row
            latest, previous = row, state
    if latest:
        state = latest['status']
        age = (now - datetime.datetime.fromisoformat(latest['at'])).total_seconds()
        result.update(outcome='in_progress', observationAgeS=round(max(0, age), 1),
                      pos=state['pos'], hp=[state.get('hp'), state.get('hpMax')],
                      observedTravelM=round(travelled, 1),
                      homeDistanceM=(state.get('home') or {}).get('distance'),
                      task=state.get('task'),
                      combatMode=(state.get('combat') or {}).get('mode'),
                      activeFood=[f['prefab'] for f in (state.get('food') or {}).get('active', [])])
        result['elapsedS'] = round((datetime.datetime.fromisoformat(latest['at']) -
                                    datetime.datetime.fromisoformat(first['at'])).total_seconds(), 1)
    goal = None
    goal_file = folder / 'goal.json'
    if goal_file.exists():
        try:
            goal = json.loads(goal_file.read_text())
        except ValueError:
            pass
    summary = folder / 'summary.json'
    if summary.exists():
        try:
            final = json.loads(summary.read_text())
        except ValueError:
            final = None
        if final:
            if isinstance(final.get('goal'), dict):
                goal = final['goal']
            elif isinstance(final.get('goal'), str):
                result['goalId'] = final['goal']
            for key in ('outcome', 'reason', 'supplied', 'minimumHP', 'damageEvents', 'finalNativeHP', 'finalNativePos', 'cleanupErrors', 'mode', 'treesHarvested', 'logsHarvested', 'defenses', 'woodGain', 'deposited', 'returned'):
                if key in final:
                    result[key] = final[key]
            tasks = final.get('tasks', [])
            if isinstance(tasks, int):
                result['taskCount'] = tasks
                result['batches'] = final.get('batches', 0)
                result['keptOnline'] = final.get('keptOnline', False)
            else:
                result['tasks'] = [dict(id=t['id'], task=t['task'], **{k: (t.get('result') or {}).get(k)
                                   for k in ('status', 'reason', 'detail')}) for t in tasks[-6:]]
            if final.get('result'):
                result['result'] = {k: final['result'][k] for k in ('status', 'reason', 'detail') if k in final['result']}
    if goal:
        result['goal'] = {k: goal[k] for k in ('status', 'reason', 'attempts', 'maxAttempts', 'taskCount', 'waitS') if k in goal}
        if goal.get('food'):
            food = goal['food']
            result['goal']['food'] = {k: food[k] for k in ('phase', 'reserve', 'batches') if k in food}
            result['goal']['food']['receipts'] = len(food.get('receipts', []))
        if goal.get('postCombat'):
            result['goal']['postCombat'] = goal['postCombat']
        if goal.get('proof'):
            result['goal']['killProof'] = {k: goal['proof'][k] for k in ('taskId', 'entity', 'detail') if k in goal['proof']}
        pending_search = (goal.get('pending') or {}).get('search')
        result['goal']['pendingSector'] = pending_search.get('sector') if pending_search else None
        result['goal']['pendingOffset'] = pending_search.get('offset', 0) if pending_search else None
        result['goal']['pendingSearchCenter'] = pending_search.get('center') if pending_search else None
        if goal.get('id') in goal_ranges:
            result['goal']['maxObservedRangeM'] = round(goal_ranges[goal['id']], 1)
        result['goal']['searches'] = [dict(sector=s['plan']['sector'], empty=s['empty'],
                                           center=s['plan'].get('center'), radius=s['plan'].get('radius'),
                                           offset=s['plan'].get('offset', 0),
                                           reason=s['outcome'].get('reason')) for s in goal.get('searches', [])]
    return result


def main():
    if len(sys.argv) > 1:
        folder = Path(sys.argv[1])
    else:
        choices = [p for pattern in ('combat-trial-*', 'harvest-trial-*', 'resupply-trial-*', 'gather-trial-*',
                                    'food-*', 'cooking-*', 'recovery-*', 'escort-*')
                   for p in (ROOT / '.checks').glob(pattern) if p.is_dir() and
                   re.fullmatch(r'\d{8}T\d{6,12}Z', p.name.rsplit('-', 1)[-1])]
        if not choices:
            raise SystemExit('No retained trial evidence')
        folder = max(choices, key=lambda p: p.name.rsplit('-', 1)[-1])
    if not folder.is_dir():
        raise SystemExit('Trial evidence directory does not exist')
    print(json.dumps(report(folder), indent=2))


if __name__ == '__main__':
    main()
