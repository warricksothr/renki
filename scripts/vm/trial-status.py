#!/usr/bin/env python3
"""Read compact progress from retained live-trial evidence, without game requests."""
import datetime
import json
import math
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]


def report(folder, now=None):
    now = now or datetime.datetime.now(datetime.timezone.utc)
    result = dict(evidence=str(folder), outcome='preparing')
    previous = latest = first = None
    travelled = 0
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
            goal = final.get('goal') or goal
            for key in ('outcome', 'reason', 'supplied', 'minimumHP', 'damageEvents', 'finalNativeHP', 'finalNativePos', 'cleanupErrors', 'mode', 'logsHarvested', 'woodGain', 'deposited', 'returned'):
                if key in final:
                    result[key] = final[key]
            result['tasks'] = [dict(id=t['id'], task=t['task'], **{k: (t.get('result') or {}).get(k)
                               for k in ('status', 'reason', 'detail')}) for t in final.get('tasks', [])[-6:]]
            if final.get('result'):
                result['result'] = {k: final['result'][k] for k in ('status', 'reason', 'detail') if k in final['result']}
    if goal:
        result['goal'] = {k: goal[k] for k in ('status', 'reason', 'attempts', 'maxAttempts', 'taskCount', 'waitS') if k in goal}
        result['goal']['pendingSector'] = ((goal.get('pending') or {}).get('search') or {}).get('sector')
        result['goal']['pendingOffset'] = ((goal.get('pending') or {}).get('search') or {}).get('offset', 0)
        result['goal']['searches'] = [dict(sector=s['plan']['sector'], empty=s['empty'],
                                           offset=s['plan'].get('offset', 0),
                                           reason=s['outcome'].get('reason')) for s in goal.get('searches', [])]
    return result


def main():
    if len(sys.argv) > 1:
        folder = Path(sys.argv[1])
    else:
        choices = [p for pattern in ('combat-trial-*', 'harvest-trial-*', 'resupply-trial-*', 'gather-trial-*')
                   for p in (ROOT / '.checks').glob(pattern) if p.is_dir()]
        if not choices:
            raise SystemExit('No retained trial evidence')
        folder = max(choices, key=lambda p: p.name.split('-trial-', 1)[1])
    if not folder.is_dir():
        raise SystemExit('Trial evidence directory does not exist')
    print(json.dumps(report(folder), indent=2))


if __name__ == '__main__':
    main()
