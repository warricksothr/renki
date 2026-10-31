#!/usr/bin/env python3
"""One owner-independent native task trial, with retained evidence and automatic cleanup."""
import argparse
from collections import Counter
import datetime
import json
import math
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[2]


def distance(a, b):
    return math.dist(a, b)


def abort_reason(status, origin=None, characters=(), target=None, travel_limit=40):
    if not status.get('connected') or status.get('phase') != 'in_world':
        return 'not in the world'
    if status.get('hpMax', 0) <= 0 or status.get('hp', 0) < status['hpMax'] * .7:
        return 'health below trial floor'
    if status.get('location', {}).get('inWater') or 'swimming' in status.get('flags', []):
        return 'unsafe water'
    if origin and distance(origin, status['pos']) > travel_limit:
        return 'trial travel limit'
    if target and any(c['id'] != target and c.get('kind') == 'creature'
                      and c.get('hostile') and c.get('hpFrac', 0) > 0
                      and distance(c['pos'], status['pos']) < 8 for c in characters):
        return 'additional nearby creature'
    if (status.get('combat') or {}).get('mode') == 'retreat' and status.get('stamina', 0) < 10:
        return 'escape stamina exhausted'
    return None


def records(value):
    return value.get('value', []) if isinstance(value, dict) else value


def goal_attempts(goal):
    # Panel summaries trim history to three; the durable search ledger keeps
    # earlier receipts. Include them without calling a home return an encounter.
    attempts = {}
    for outcome in [s['outcome'] for s in goal.get('searches', [])] + goal.get('history', []):
        if outcome.get('template', 'combat') == 'combat' and outcome.get('taskId'):
            attempts[outcome['taskId']] = outcome
    return list(attempts.values())[-8:]


def water_recovery_args(status, characters):
    """One native return prerequisite after an interrupted wet trial, no steering."""
    home = status.get('home') or {}
    if characters is None:
        raise RuntimeError('fresh surroundings required for startup water recovery')
    if (home.get('world') != status.get('world') or not home.get('pos')
            or distance(status['pos'], home['pos']) > 150 or status.get('hp', 0) < status.get('hpMax', 1) * .8
            or status.get('stamina', 0) < 30 or not (status.get('food') or {}).get('active')):
        raise RuntimeError('water recovery requires known nearby home, food, health and stamina')
    if any(c.get('kind') == 'creature' and c.get('hpFrac', 0) > 0
           and (c.get('targetingMe') or distance(c['pos'], status['pos']) < (15 if c.get('hostile') else 8)) for c in characters):
        raise RuntimeError('another creature prevents startup water recovery')
    return dict(to=home['pos'], dry=True, stopDistance=2)


def terminal(value, task_id):
    for row in reversed(records(value)):
        frame = row.get('frame', {})
        if frame.get('re') == task_id and frame.get('status') in ('succeeded', 'failed', 'rejected', 'cancelled'):
            return frame
    return None


def vital_diagnostics(audit):
    """Include native frames missed between operator polls, even during cleanup."""
    result = dict(damageEvents=0)
    with Path(audit).open(encoding='utf-8-sig') as lines:
        for line in lines:
            frame = json.loads(line).get('frame', {})
            if frame.get('type') == 'event' and frame.get('event') == 'damaged':
                result['damageEvents'] += 1
            if frame.get('type') == 'state' and frame.get('phase') == 'in_world':
                hp = frame.get('hp')
                if isinstance(hp, (int, float)) and math.isfinite(hp):
                    result['minimumHP'] = min(result.get('minimumHP', hp), hp)
                    result['finalNativeHP'] = hp
                    result['finalNativePos'] = frame.get('pos')
    return result


def ground_diagnostics(audit, task_id):
    """Bounded projection of one native goal; full traces remain in the audit."""
    reasons, stages, regions = Counter(), Counter(), Counter()
    examples, seen = [], set()
    modes, selected, scout_examples = [], None, []
    scout_reasons, scout_routes, scout_recoveries = Counter(), Counter(), Counter()
    detour_states, detour_examples = Counter(), []
    detour_max_nodes = detour_worst_ms = 0
    return_hint = None
    area_travel = dict(routes=0, arrival=None)
    footing = dict(settles=0, lastRefusal=None)
    death = None
    with Path(audit).open() as lines:
        for line in lines:
            row = json.loads(line)
            frame = row.get('frame', {})
            if (frame.get('type') != 'trace' or frame.get('task') != task_id
                    or frame.get('source') != 'combat'):
                continue
            data = frame.get('data') or {}
            message = frame.get('msg')
            if message in ('prepare_search', 'travel_search_area', 'position', 'search', 'scout_plan', 'scout', 'scout_return', 'scout_settle', 'draw', 'kite', 'hold',
                           'engage', 'approach', 'strike', 'block', 'dodge', 'wait', 'retreat') and message not in modes:
                modes.append(message)
            if message == 'scout_settle':
                footing['settles'] += 1
            if message == 'footing_refused':
                footing['lastRefusal'] = {k: data[k] for k in ('pos', 'grounded', 'swimming', 'encumbered', 'supportGap') if k in data}
            if message == 'death_observed':
                death = {k: data[k] for k in ('entity', 'evidence') if k in data}
            if message == 'ground_selected':
                selected = {k: data[k] for k in ('area', 'escape', 'stanceRadius') if k in data}
            if message == 'scout_route':
                scout_routes['return' if data.get('returning') else 'outbound'] += 1
                return_hint = {k: data[k] for k in ('trailPoints', 'returnDistance') if k in data}
            if message == 'scout_recovery':
                scout_recoveries[data.get('reason', 'unknown')] += 1
            if message and message.startswith('scout_detour_'):
                state = message.removeprefix('scout_detour_')
                detour_states[state] += 1
                detour_max_nodes = max(detour_max_nodes, data.get('expanded', 0))
                detour_worst_ms = max(detour_worst_ms, data.get('worstFrameMs', 0))
                if state != 'started':
                    detour_examples.append(dict(state=state, **{k: data[k] for k in
                        ('attempt', 'returning', 'reason', 'origin', 'request', 'target', 'expanded', 'probes', 'ms', 'worstFrameMs') if k in data}))
                    detour_examples = detour_examples[-4:]
            if message == 'search_egress_route':
                scout_routes['preparation'] += 1
            if message == 'search_area_travel':
                area_travel['routes'] += 1
            if message == 'search_area_arrived':
                area_travel['arrival'] = {k: data[k] for k in ('requested', 'pos', 'radius', 'travel') if k in data}
            if message in ('scout_rejected', 'search_egress_rejected'):
                reason = data.get('reason', 'unknown')
                if reason not in scout_reasons and len(scout_examples) < 6:
                    scout_examples.append({k: data[k] for k in ('goal', 'reason', 'pos', 'from', 'desired', 'supported',
                                                               'angle', 'previousAngle', 'roughness') if k in data})
                scout_reasons[reason] += 1
            if message != 'ground_rejected':
                continue
            reasons[data.get('reason', 'unknown')] += 1
            if data.get('region'):
                regions[data['region']] += 1
            stage = data.get('probeStage')
            if stage:
                stages[stage] += 1
            kind = (data.get('reason'), stage)
            if kind not in seen and len(examples) < 6:
                seen.add(kind)
                examples.append({k: data[k] for k in ('candidate', 'reason', 'region', 'segment', 'from', 'desired',
                                'supported', 'angle', 'previousAngle', 'roughness', 'probeStage', 'probeAt', 'collider',
                                'colliderCenter') if k in data})
    return dict(candidates=sum(reasons.values()), reasons=dict(reasons), probeStages=dict(stages), regions=dict(regions), examples=examples,
                selected=selected, modes=modes, footing=footing, death=death, scouting=dict(routes=dict(scout_routes), reasons=dict(scout_reasons),
                recoveries=dict(scout_recoveries), returnHint=return_hint, areaTravel=area_travel, examples=scout_examples,
                detours=dict(states=dict(detour_states), maxExpanded=detour_max_nodes, worstFrameMs=detour_worst_ms, examples=detour_examples)))


def run_persistent_goal(summary, call, observe, saved):
    """One durable objective; a child-task failure is an outcome, not the end."""
    status, _ = observe()
    spec = dict(kind='combat_trial', world=status['world'], center=status['pos'], radius=200, returnHome=True)
    created = call('panel', 'POST', '/api/goals', json.dumps(dict(action='start', spec=spec)), as_json=True)
    goal_id = created['id']
    summary.update(goalId=goal_id, origin=status['pos'], outcome='running')
    print('Monitoring persistent goal ' + goal_id, flush=True)
    deadline, last_progress = time.monotonic() + 610, None
    while time.monotonic() < deadline:
        status, mapped = observe()
        goals = status.get('goals', [])
        goal = next((g for g in goals if g['id'] == goal_id), None)
        if goal is None:
            raise RuntimeError('persistent goal missing from status')
        (saved / 'goal.json').write_text(json.dumps(goal, indent=2) + '\n')
        summary['goal'] = goal
        progress = (goal['status'], goal.get('attempts', 0), goal.get('reason', ''))
        if progress != last_progress:
            print(json.dumps(dict(status=progress[0], attempts=progress[1], reason=progress[2])), flush=True)
            last_progress = progress
        if goal['status'] in ('succeeded', 'needs_plan', 'cancelled', 'suspended'):
            # The host must retain exact native evidence; never trust a script claim.
            if goal['status'] == 'succeeded' and not goal.get('proof'):
                raise RuntimeError('successful goal lacks host proof')
            if goal['status'] == 'succeeded' and ((goal.get('postCombat') or {}).get('phase') != 'complete'
                    or ((goal.get('postCombat') or {}).get('receipt') or {}).get('status') != 'succeeded'):
                raise RuntimeError('successful goal lacks confirmed native home return')
            summary.update(outcome=goal['status'], reason=goal.get('reason', ''))
            return
        target = (goal.get('pending') or {}).get('entity') or (goal.get('proof') or {}).get('entity')
        combat = status.get('combat') or {}
        if combat.get('id') == (goal.get('pending') or {}).get('taskId') and combat.get('entity'):
            target = combat['entity']
        reason = abort_reason(status, summary['origin'], (mapped.get('perception') or {}).get('characters', []), target, spec['radius'])
        if reason:
            raise RuntimeError(reason)
        time.sleep(2)
    raise RuntimeError('persistent goal supervisor time limit')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    harvest_mode = parser.add_mutually_exclusive_group()
    harvest_mode.add_argument('--gather-tree', action='store_true', help='test the controller gather -> storage -> return chain')
    harvest_mode.add_argument('--harvest-tree', action='store_true', help='run the bounded full-tree/log benchmark with combat off')
    harvest_mode.add_argument('--harvest-logs', type=float, nargs=3, help='retry existing nearby beech logs around this center; no full-tree claim')
    harvest_mode.add_argument('--resupply-food', action='store_true', help='return home and replenish ordinary carried food from known storage')
    parser.add_argument('--persistent', action='store_true', help='submit and monitor one durable goal through failed native tasks')
    parser.add_argument('--direct', action='store_true', help='test native closing/melee without lure; requires --entity')
    parser.add_argument('--entity', help='optional exact target; otherwise C# acquires a local ordinary Greyling')
    parser.add_argument('--wait-s', type=float, default=30, help='bounded food/stamina/target preparation, 0..120 seconds')
    parser.add_argument('--at', type=float, nargs=3, help='optional preferred nearby fighting-ground center')
    parser.add_argument('--gather-trees', type=int, default=1, choices=range(1, 4), help='full trees required by the controller gather benchmark')
    parser.add_argument('--gather-site', type=float, nargs=3, help='known work-site hint; controller revalidates actual resources on arrival')
    args = parser.parse_args()
    harvesting = args.harvest_tree or args.harvest_logs is not None or args.gather_tree
    if (args.gather_trees != 1 or args.gather_site is not None) and not args.gather_tree:
        parser.error('gather count/site require --gather-tree')
    if args.gather_site and not all(math.isfinite(v) for v in args.gather_site):
        parser.error('--gather-site must contain finite coordinates')
    if args.persistent and (harvesting or args.resupply_food or args.direct or args.entity or args.at):
        parser.error('--persistent chooses its own target and area; cannot combine with other trial modes')
    if args.direct and (not args.entity or args.at or harvesting or args.resupply_food):
        parser.error('--direct requires --entity and cannot combine with --at or harvesting')
    if (harvesting or args.resupply_food) and (args.entity or args.at):
        parser.error('harvest mode selects its own eligible nearby work site')
    if args.harvest_logs and not all(math.isfinite(v) for v in args.harvest_logs):
        parser.error('--harvest-logs must contain finite coordinates')
    if args.at and not all(math.isfinite(v) for v in args.at):
        parser.error('--at must contain finite coordinates')
    if not math.isfinite(args.wait_s) or not 0 <= args.wait_s <= 120:
        parser.error('--wait-s must be between 0 and 120')
    saved = ROOT / '.checks' / (('resupply-trial-' if args.resupply_food else 'gather-trial-' if args.gather_tree else 'harvest-trial-' if harvesting else 'combat-trial-') + datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%S%fZ'))
    saved.mkdir(parents=True)
    summary = {'outcome': 'not_started', 'evidence': str(saved)}
    since = datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z')

    def call(*arguments, timeout=45, as_json=False):
        p = subprocess.run(['just', 'vm', *arguments], cwd=ROOT, capture_output=True, text=True, timeout=timeout)
        with (saved / 'commands.log').open('a') as f:
            f.write('just vm ' + ' '.join(arguments) + '\n' + p.stdout + p.stderr)
        if p.returncode:
            raise RuntimeError('just vm ' + arguments[0] + ' failed; see commands.log')
        return json.loads(p.stdout) if as_json else p.stdout

    def observe():
        status = call('panel', 'GET', '/api/status', as_json=True)
        mapped = call('panel', 'GET', '/api/map', as_json=True)
        with (saved / 'observations.jsonl').open('a') as f:
            f.write(json.dumps({'at': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'status': status,
                                'perception': mapped.get('perception')}) + '\n')
        if status.get('hp') is not None:
            summary['minimumHP'] = min(summary.get('minimumHP', status['hp']), status['hp'])
        summary['lastStatus'] = status
        return status, mapped

    try:
        print('Preparing one bounded ' + ('resupply' if args.resupply_food else 'harvest' if harvesting else 'combat') + ' trial; evidence: ' + str(saved), flush=True)
        call('stop', timeout=200)
        call('combat', 'false' if harvesting or args.resupply_food else 'true')
        call('start', timeout=330)
        deadline = time.monotonic() + (max(args.wait_s, 120) if args.resupply_food else args.wait_s)
        startup_deadline = time.monotonic() + 120
        startup_id, startup_health, startup_scope = None, None, None
        startup_anchor, startup_home_pos = None, None
        while True:
            status, mapped = observe()
            reason = abort_reason(status)
            if reason == 'unsafe water':
                chars = (mapped.get('perception') or {}).get('characters')
                recovery = water_recovery_args(status, chars)
                if startup_id or (status.get('task') or {}).get('task') == 'move_to':
                    reason = None  # Existing home task is still checked below.
                elif not status.get('task') and not summary.get('startupRecoveryTask'):
                    request = call('task', 'move_to', json.dumps(recovery), '120', as_json=True)
                    summary['startupRecoveryTask'] = request['id']
                    print('Native dry-ground return is recovering the interrupted wet start', flush=True)
                    time.sleep(1)
                    continue
            if reason:
                raise RuntimeError(reason)
            if startup_id:
                from harvest_trial import health_loss
                if ((status.get('world'), status.get('worldUid'), status.get('connection')) != startup_scope
                        or (status.get('home') or {}).get('pos') != startup_home_pos or health_loss(startup_health, status)):
                    raise RuntimeError('startup home return changed scope or suffered damage')
                startup_health = status
            chars = (mapped.get('perception') or {}).get('characters', [])
            # Do not wait for regeneration while being chased, or replace an escape.
            if status.get('task'):
                if status['task'].get('task') == 'move_to':
                    from resupply_trial import startup_home
                    active_id = status['task']['id']
                    if startup_id and active_id != startup_id:
                        raise RuntimeError('startup home return was replaced')
                    if not startup_id:
                        startup_anchor = datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z')
                        history = call('panel', 'GET', '/api/task?id=' + active_id + '&at=' + startup_anchor, as_json=True)
                        if not startup_home(status, history):
                            raise RuntimeError('another task is active; trial will not replace it')
                        startup_id, startup_health = active_id, status
                        startup_scope = (status.get('world'), status.get('worldUid'), status.get('connection'))
                        startup_home_pos = list(status['home']['pos'])
                        summary['startupHomeTask'] = active_id
                        print('Waiting for the existing native return-home task', flush=True)
                    if time.monotonic() >= startup_deadline:
                        raise RuntimeError('startup home return preparation time limit')
                    time.sleep(1)
                    continue
                raise RuntimeError('another task is active; trial will not replace it')
            if startup_id:
                history = call('panel', 'GET', '/api/task?id=' + startup_id + '&at=' + startup_anchor, as_json=True)
                ended = terminal(history, startup_id)
                if not ended or ended.get('status') != 'succeeded' or distance(status['pos'], startup_home_pos) > 3:
                    raise RuntimeError('startup home return did not complete successfully at home')
                summary['startupHomeResult'] = {k: ended[k] for k in ('status', 'reason', 'detail') if k in ended}
                startup_id = None
                deadline = time.monotonic() + (max(args.wait_s, 120) if args.resupply_food else args.wait_s)
            ready = (status.get('hp', 0) >= status.get('hpMax', 1) * .8
                     and status.get('stamina', 0) >= 40 and status.get('food', {}).get('active'))
            if ready or args.resupply_food:
                break
            if any(c.get('hostile') and c.get('kind') == 'creature' and c.get('hpFrac', 0) > 0
                   and distance(c['pos'], status['pos']) < 15 for c in chars):
                raise RuntimeError('cannot prepare safely near a hostile creature')
            if time.monotonic() >= deadline:
                raise RuntimeError('no fed, rested, safe single-Greyling encounter available')
            time.sleep(1)
        if args.resupply_food:
            from resupply_trial import run
            print('Returning home for a bounded food resupply', flush=True)
            run(summary, call, observe, terminal, saved)
        elif args.persistent:
            run_persistent_goal(summary, call, observe, saved)
        elif args.gather_tree:
            from gather_trial import run
            print('Testing controller gather, gains-only delivery and return', flush=True)
            if args.gather_trees != 1 or args.gather_site is not None:
                run(summary, call, observe, saved, trees=args.gather_trees, site=args.gather_site)
            else:
                run(summary, call, observe, saved)
        elif harvesting:
            from harvest_trial import run
            print('Testing native ' + ('existing logs' if args.harvest_logs else 'full-tree/log harvesting'), flush=True)
            run(summary, call, observe, terminal, saved, log_center=args.harvest_logs)
        else:
            goal = {'kite': not args.direct, 'style': 'melee'}
            if args.entity:
                goal['entity'] = args.entity
            if args.at:
                goal['at'] = args.at
            summary['goal'] = goal
            summary['origin'] = status['pos']
            summary['outcome'] = 'running'
            task_timeout = 150 if args.direct else 270
            dispatched = call('task', 'attack', json.dumps(goal), str(task_timeout), as_json=True)
            task_id = dispatched['id']
            summary['task'] = task_id
            print('Testing one native ' + ('closing/melee' if args.direct else 'acquisition/kiting/melee') + ' goal', flush=True)
            anchor = datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z')
            deadline = time.monotonic() + task_timeout + 5
            while time.monotonic() < deadline:
                status, mapped = observe()
                result = call('panel', 'GET', '/api/task?id=' + task_id + '&at=' + anchor, as_json=True)
                (saved / 'task.json').write_text(json.dumps(result, indent=2) + '\n')
                ended = terminal(result, task_id)
                if ended:
                    summary.update(outcome=ended['status'], result=ended)
                    break
                if (status.get('task') or {}).get('id') != task_id:
                    raise RuntimeError('trial attack lost body ownership')
                combat = status.get('combat') or {}
                if combat.get('id') == task_id and combat.get('entity'):
                    summary['entity'] = combat['entity']
                reason = abort_reason(status, summary['origin'], (mapped.get('perception') or {}).get('characters', []), summary.get('entity'), 40 if args.direct else 112)
                if reason:
                    raise RuntimeError(reason)
                time.sleep(.5)
            else:
                raise RuntimeError('trial time limit')
    except Exception as e:
        summary.update(outcome='aborted', reason=str(e))
    finally:
        # Always save/logout, stop the game and restore combat off. Restart only
        # the controller so monitoring remains available between trials.
        cleanup = []
        if summary.get('behaviourName'):
            for action in ('stop', 'delete'):
                cleanup.append(('panel', 'POST', '/api/behaviours', json.dumps(dict(action=action, name=summary['behaviourName']))))
        if summary.get('goalId') and summary.get('goal', {}).get('status') not in ('succeeded', 'needs_plan', 'cancelled'):
            cleanup.append(('panel', 'POST', '/api/goals', json.dumps(dict(action='stop', id=summary['goalId']))))

        def cleanup_call(arguments):
            try:
                call(*arguments, timeout=200)
            except Exception as error:
                summary.setdefault('cleanupErrors', []).append(str(error))

        urgent = any(reason in summary.get('reason', '').lower() for reason in
                     ('hostile', 'health', 'damage', 'ownership', 'unsafe water'))
        if urgent:
            # Do not leave an aborted, stationary bot exposed while slow SSH
            # diagnostics or automation deletion consume its threat buffer.
            cleanup_call(('panel', 'POST', '/api/act/logout', '{}'))

        # Revoke this trial's automation before fetching diagnostic imagery;
        # a slow image transfer must not keep a rejected behavior moving.
        for arguments in cleanup:
            cleanup_call(arguments)
        # Retain the current view before logout for agent-assisted diagnosis. This
        # reads the latest frame; it never bypasses the native capture ceiling.
        if not urgent:
            try:
                summary['cameraEvidence'] = call('camera').strip()
            except Exception as error:
                summary['cameraUnavailable'] = str(error)
        for arguments in [('panel', 'POST', '/api/act/logout', '{}'), ('stop',), ('combat', 'false'), ('panel-start',)]:
            cleanup_call(arguments)
        until = datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z')
        try:
            audit = call('audit-window', since, until, timeout=60).strip()
            summary['audit'] = audit
            vitals = vital_diagnostics(audit)
            if 'minimumHP' in vitals:
                vitals['minimumHP'] = min(summary.get('minimumHP', vitals['minimumHP']), vitals['minimumHP'])
            summary.update(vitals)
            if args.gather_tree:
                from gather_trial import verify
                try:
                    summary.update(verify(audit, summary))
                    if summary.get('outcome') == 'succeeded' and summary.get('damageEvents', 0):
                        summary.update(outcome='failed', reason='native damage occurred during controller gather')
                except Exception as error:
                    summary.update(outcome='failed', reason=str(error))
            if harvesting:
                try:
                    from harvest_trial import diagnostics
                    summary['harvest'] = diagnostics(audit, [t['id'] for t in summary.get('tasks', [])])
                except Exception as e:
                    summary['diagnosticError'] = str(e)
            if summary.get('task') and not harvesting:
                try:
                    summary['ground'] = ground_diagnostics(audit, summary['task'])
                except Exception as e:
                    summary['diagnosticError'] = str(e)
            if summary.get('goalId'):
                try:
                    attempts = goal_attempts(summary.get('goal', {}))
                    summary['encounters'] = [dict(taskId=attempt['taskId'], status=attempt['status'],
                        reason=attempt.get('reason'), entity=attempt.get('entity'),
                        ground=ground_diagnostics(audit, attempt['taskId'])) for attempt in attempts if attempt.get('taskId')]
                except Exception as e:
                    summary['diagnosticError'] = str(e)
        except Exception as e:
            summary['auditError'] = str(e)
        if args.gather_tree and summary['outcome'] == 'awaiting_proof':
            summary.update(outcome='failed', reason='controller gather audit proof unavailable')
        (saved / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
    report = {k: summary[k] for k in ('outcome', 'reason', 'entity', 'minimumHP', 'damageEvents', 'finalNativeHP', 'evidence', 'cleanupErrors') if k in summary}
    if summary.get('result'):
        report['result'] = {k: summary['result'][k] for k in ('status', 'reason', 'detail') if k in summary['result']}
    if summary.get('goalId'):
        report['goal'] = {k: summary.get('goal', {}).get(k) for k in ('id', 'status', 'attempts', 'taskCount', 'reason', 'proof', 'home', 'postCombat')}
        report['encounters'] = [dict(taskId=e['taskId'], status=e['status'], reason=e['reason'],
            entity=e['entity'], modes=e['ground']['modes'], scouting={k:e['ground']['scouting'][k] for k in ('routes','reasons','recoveries','returnHint','areaTravel')},
            detours={k:e['ground']['scouting']['detours'][k] for k in ('states','maxExpanded','worstFrameMs')})
            for e in summary.get('encounters', [])]
    if harvesting:
        report.update({k: summary[k] for k in ('mode', 'treesHarvested', 'logsHarvested', 'woodGain', 'deposited', 'returned', 'selection') if k in summary})
        report['tasks'] = len(summary.get('tasks', []))
        if summary.get('harvest'):
            report['harvest'] = summary['harvest']
    if args.resupply_food:
        report.update({k: summary[k] for k in ('supplied', 'activeFood', 'checkedChests', 'searchChests', 'requests') if k in summary})
        report['tasks'] = len(summary.get('tasks', []))
    if summary.get('ground'):
        report['ground'] = {k: summary['ground'][k] for k in ('candidates', 'reasons', 'probeStages', 'modes', 'selected', 'footing', 'death')}
        report['scouting'] = {k: summary['ground']['scouting'][k] for k in ('routes', 'reasons', 'recoveries', 'returnHint', 'areaTravel', 'detours')}
    print(json.dumps(report, indent=2))
    return 0 if summary['outcome'] == 'succeeded' and not summary.get('cleanupErrors') else 1


if __name__ == '__main__':
    sys.exit(main())
